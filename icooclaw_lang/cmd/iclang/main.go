package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/issueye/icooclaw_lang/internal/evaluator"
	"github.com/issueye/icooclaw_lang/internal/lexer"
	"github.com/issueye/icooclaw_lang/internal/object"
	"github.com/issueye/icooclaw_lang/internal/parser"
)

var VERSION = "0.1.0"

func main() {
	handled, err := tryRunBundledExecutable(os.Args[1:])
	if err != nil {
		fmt.Printf("Error: could not run bundled executable: %s\n", err)
		os.Exit(1)
	}
	if handled {
		return
	}

	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	replCmd := flag.NewFlagSet("repl", flag.ExitOnError)
	buildOutput := buildCmd.String("o", "", "output executable path")
	initName := initCmd.String("name", "", "project name (defaults to directory name)")
	runMaxGoroutines := runCmd.Int("max-goroutines", 0, "override runtime goroutine pool size")
	replMaxGoroutines := replCmd.Int("max-goroutines", 0, "override runtime goroutine pool size")
	versionFlag := flag.Bool("version", false, "print version")
	flag.Parse()

	if len(os.Args) < 2 {
		fmt.Println("icooclaw script language v" + VERSION)
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  iclang run [--max-goroutines n] <file.is> [args...]")
		fmt.Println("  iclang build <file.is> [-o app]   Bundle script and runtime into an executable")
		fmt.Println("  iclang init <dir> [-name demo]    Initialize a standard project")
		fmt.Println("  iclang version          Show version")
		fmt.Println("  iclang repl [--max-goroutines n]  Start interactive REPL")
		os.Exit(0)
	}

	if *versionFlag || isVersionArg(os.Args[1:]) {
		fmt.Println("iclang v" + VERSION)
		return
	}

	switch os.Args[1] {
	case "run":
		runCmd.Parse(os.Args[2:])
		args := runCmd.Args()
		if len(args) == 0 {
			fmt.Println("Error: no input file specified")
			fmt.Println("Usage: iclang run [--max-goroutines n] <file.is> [args...]")
			os.Exit(1)
		}
		runFile(args[0], args[1:], *runMaxGoroutines)
	case "build":
		buildCmd.Parse(os.Args[2:])
		args := buildCmd.Args()
		if len(args) == 0 {
			fmt.Println("Error: no input file specified")
			fmt.Println("Usage: iclang build <file.is|project_dir|pkg.toml> [-o app]")
			os.Exit(1)
		}

		output := *buildOutput
		if output == "" {
			output = defaultBundleOutputPath(args[0])
		}
		if err := buildBundle(args[0], output); err != nil {
			fmt.Printf("Error: could not build bundle: %s\n", err)
			os.Exit(1)
		}
		absOutput, err := filepath.Abs(output)
		if err != nil {
			absOutput = output
		}
		fmt.Println(absOutput)
	case "init":
		initCmd.Parse(os.Args[2:])
		args := initCmd.Args()
		if len(args) == 0 {
			fmt.Println("Error: no project directory specified")
			fmt.Println("Usage: iclang init <dir> [-name demo]")
			os.Exit(1)
		}

		projectDir, err := initProject(args[0], *initName)
		if err != nil {
			fmt.Printf("Error: could not initialize project: %s\n", err)
			os.Exit(1)
		}
		fmt.Println(projectDir)
	case "version":
		fmt.Println("iclang v" + VERSION)
	case "repl":
		replCmd.Parse(os.Args[2:])
		startRepl(*replMaxGoroutines)
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runFile(filename string, scriptArgs []string, maxGoroutines int) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error: could not read file '%s': %s\n", filename, err)
		os.Exit(1)
	}

	if err := executeScriptSource(filename, string(data), scriptArgs, maxGoroutines); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func executeScriptSource(scriptPath, source string, scriptArgs []string, maxGoroutines int) error {
	l := lexer.New(source)
	p := parser.New(l)

	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Println("Parse Error:", e)
		}
		return errors.New("parse failed")
	}

	env := object.NewEnvironment()
	env.SetCLIContext(scriptPath, scriptArgs)
	configureRuntimeConcurrency(env, maxGoroutines)
	result := evaluator.Eval(program, env)
	env.Wait()

	if result != nil {
		if err, ok := result.(*object.Error); ok {
			return errors.New(err.Inspect())
		}
	}
	return nil
}

func startRepl(maxGoroutines int) {
	fmt.Println("iclang REPL v" + VERSION)
	fmt.Println("Type 'exit' to quit, 'help' for help")
	fmt.Println()

	env := object.NewEnvironment()
	env.SetCLIContext("", nil)
	configureRuntimeConcurrency(env, maxGoroutines)
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("iclang> ")

		input, err := reader.ReadString('\n')
		if err != nil && len(input) == 0 {
			fmt.Println()
			break
		}
		input = strings.TrimSpace(input)

		if input == "exit" || input == "quit" {
			fmt.Println("Bye!")
			break
		}
		if input == "help" {
			printHelp()
			continue
		}
		if input == "" {
			continue
		}

		l := lexer.New(input)
		p := parser.New(l)

		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			for _, e := range p.Errors() {
				fmt.Println("Error:", e)
			}
			continue
		}

		result := evaluator.Eval(program, env)
		env.Wait()
		if result != nil {
			if err, ok := result.(*object.Error); ok {
				fmt.Println(err.Inspect())
			} else if _, ok := result.(*object.Null); !ok {
				fmt.Println(result.Inspect())
			}
		}
	}
}

func printHelp() {
	fmt.Println("icooclaw script language (iclang)")
	fmt.Println()
	fmt.Println("Keywords: fn, if, else, for, while, match, break, continue,")
	fmt.Println("          return, const, import, export, try, catch, go,")
	fmt.Println("          null, true, false, in")
	fmt.Println()
	fmt.Println("Built-in functions:")
	fmt.Println("  print(...), len(obj), range(n), type(obj), type_of(obj),")
	fmt.Println("  str(obj), int(obj), float(obj), input(msg),")
	fmt.Println("  push(arr, val), pop(arr), abs(n),")
	fmt.Println("  read_file(path), write_file(path, content)")
	fmt.Println()
	fmt.Println("Built-in libraries:")
	fmt.Println("  async, db, fs, http, json, toml, yaml, log, time, os, exec, path, crypto, websocket, sse")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  exit/quit - exit REPL")
	fmt.Println("  help      - show this help")
}

func configureRuntimeConcurrency(env *object.Environment, maxGoroutines int) {
	if env == nil || env.Runtime() == nil || maxGoroutines <= 0 {
		return
	}
	env.Runtime().SetMaxConcurrency(maxGoroutines)
}

func parseRuntimeOptions(args []string) (int, []string, error) {
	if len(args) == 0 {
		return 0, nil, nil
	}

	maxGoroutines := 0
	remaining := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			remaining = append(remaining, args[i+1:]...)
			break
		}

		if strings.HasPrefix(arg, "--max-goroutines=") {
			valueText := strings.TrimPrefix(arg, "--max-goroutines=")
			value, err := strconv.Atoi(valueText)
			if err != nil || value <= 0 {
				return 0, nil, fmt.Errorf("invalid value for --max-goroutines: %q", valueText)
			}
			maxGoroutines = value
			continue
		}

		if arg == "--max-goroutines" {
			if i+1 >= len(args) {
				return 0, nil, errors.New("missing value for --max-goroutines")
			}
			valueText := args[i+1]
			value, err := strconv.Atoi(valueText)
			if err != nil || value <= 0 {
				return 0, nil, fmt.Errorf("invalid value for --max-goroutines: %q", valueText)
			}
			maxGoroutines = value
			i++
			continue
		}

		remaining = append(remaining, args[i:]...)
		break
	}

	return maxGoroutines, remaining, nil
}

func isVersionArg(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "-v", "--version", "version":
			return true
		}
	}
	return false
}
