package builtin

import (
	"bufio"
	"bytes"
	"io"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/issueye/icooclaw_lang/internal/object"
)

type nativeExecLine struct {
	stream string
	text   string
}

type nativeExecProcess struct {
	mu         sync.Mutex
	cmd        *osexec.Cmd
	lines      chan nativeExecLine
	stdout     strings.Builder
	stderr     strings.Builder
	done       chan struct{}
	waitOnce   sync.Once
	waitErr    error
	exitCode   int64
	running    bool
	waited     bool
	closeLines sync.Once
}

func newExecLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"look_path": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			name, errObj := stringArg(args[0], "argument to `look_path` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}

			path, err := osexec.LookPath(name)
			if err != nil {
				return object.NullObject()
			}
			return &object.String{Value: path}
		}),
		"command": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			name, commandArgs, errObj := parseExecCommandArgs("command", args, 1)
			if errObj != nil {
				return errObj
			}
			return runExecCommand("", name, commandArgs)
		}),
		"command_in": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2 or 3", len(args))
			}

			dir, errObj := stringArg(args[0], "first argument to `command_in` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}

			name, commandArgs, errObj := parseExecCommandArgs("command_in", args[1:], 2)
			if errObj != nil {
				return errObj
			}
			return runExecCommand(dir, name, commandArgs)
		}),
		"start": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			name, commandArgs, errObj := parseExecCommandArgs("start", args, 1)
			if errObj != nil {
				return errObj
			}
			return startExecCommand("", name, commandArgs)
		}),
		"start_in": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2 or 3", len(args))
			}

			dir, errObj := stringArg(args[0], "first argument to `start_in` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}

			name, commandArgs, errObj := parseExecCommandArgs("start_in", args[1:], 2)
			if errObj != nil {
				return errObj
			}
			return startExecCommand(dir, name, commandArgs)
		}),
	})
}

func parseExecCommandArgs(name string, args []object.Object, commandIndex int) (string, []string, *object.Error) {
	commandName, errObj := stringArg(args[0], ordinalArgumentName(commandIndex)+" argument to `"+name+"` must be STRING, got %s")
	if errObj != nil {
		return "", nil, errObj
	}

	if len(args) == 1 {
		return commandName, nil, nil
	}

	commandArgs, errObj := stringArrayArg(args[1], ordinalArgumentName(commandIndex+1)+" argument to `"+name+"` must be ARRAY<STRING>, got %s")
	if errObj != nil {
		return "", nil, errObj
	}
	return commandName, commandArgs, nil
}

func stringArrayArg(arg object.Object, message string) ([]string, *object.Error) {
	array, ok := arg.(*object.Array)
	if !ok {
		return nil, object.NewError(0, message, arg.Type())
	}

	values := make([]string, 0, len(array.Elements))
	for _, item := range array.Elements {
		text, errObj := stringArg(item, message)
		if errObj != nil {
			return nil, errObj
		}
		values = append(values, text)
	}
	return values, nil
}

func ordinalArgumentName(index int) string {
	switch index {
	case 1:
		return "first"
	case 2:
		return "second"
	case 3:
		return "third"
	default:
		return "argument"
	}
}

func runExecCommand(dir, name string, args []string) object.Object {
	cmd := buildExecCommand(dir, name, args)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode, ok := exitCodeFromError(err)
	if err != nil && exitCode == -1 && stderr.Len() == 0 {
		stderr.WriteString(err.Error())
	}

	return execResultObject(ok, exitCode, stdout.String(), stderr.String(), joinExecCommand(name, args), cmd.Dir)
}

func startExecCommand(dir, name string, args []string) object.Object {
	cmd := buildExecCommand(dir, name, args)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return object.NewError(0, "could not create stdout pipe: %s", err.Error())
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return object.NewError(0, "could not create stderr pipe: %s", err.Error())
	}
	if err := cmd.Start(); err != nil {
		return object.NewError(0, "could not start command '%s': %s", name, err.Error())
	}

	state := &nativeExecProcess{
		cmd:      cmd,
		lines:    make(chan nativeExecLine, 64),
		done:     make(chan struct{}),
		exitCode: -1,
		running:  true,
	}

	var readers sync.WaitGroup
	readers.Add(2)
	go state.readPipe("stdout", stdoutPipe, &readers)
	go state.readPipe("stderr", stderrPipe, &readers)
	go state.waitForExit(&readers)

	return newExecProcessObject(state, name, args)
}

func buildExecCommand(dir, name string, args []string) *osexec.Cmd {
	cmd := osexec.Command(name, args...)
	if dir != "" {
		if absDir, err := filepath.Abs(dir); err == nil {
			cmd.Dir = absDir
		} else {
			cmd.Dir = dir
		}
	}
	return cmd
}

func joinExecCommand(name string, args []string) string {
	return strings.TrimSpace(strings.Join(append([]string{name}, args...), " "))
}

func execResultObject(ok bool, exitCode int64, stdout, stderr, command, dir string) *object.Hash {
	return hashObject(map[string]object.Object{
		"ok":      boolObject(ok),
		"code":    &object.Integer{Value: exitCode},
		"stdout":  &object.String{Value: stdout},
		"stderr":  &object.String{Value: stderr},
		"output":  &object.String{Value: stdout + stderr},
		"command": &object.String{Value: command},
		"dir":     &object.String{Value: dir},
	})
}

func newExecProcessObject(state *nativeExecProcess, name string, args []string) *object.Hash {
	command := joinExecCommand(name, args)

	return hashObject(map[string]object.Object{
		"read": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return state.read()
		}),
		"wait": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return state.wait(command)
		}),
		"kill": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return state.kill()
		}),
		"is_running": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return boolObject(state.isRunning())
		}),
		"pid": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			state.mu.Lock()
			defer state.mu.Unlock()
			if state.cmd == nil || state.cmd.Process == nil {
				return object.NullObject()
			}
			return &object.Integer{Value: int64(state.cmd.Process.Pid)}
		}),
		"stdout": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			state.mu.Lock()
			defer state.mu.Unlock()
			return &object.String{Value: state.stdout.String()}
		}),
		"stderr": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			state.mu.Lock()
			defer state.mu.Unlock()
			return &object.String{Value: state.stderr.String()}
		}),
		"command": &object.String{Value: command},
	})
}

func (p *nativeExecProcess) readPipe(stream string, reader io.ReadCloser, wg *sync.WaitGroup) {
	defer wg.Done()
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		p.mu.Lock()
		if stream == "stdout" {
			p.stdout.WriteString(line)
			p.stdout.WriteString("\n")
		} else {
			p.stderr.WriteString(line)
			p.stderr.WriteString("\n")
		}
		p.mu.Unlock()
		p.lines <- nativeExecLine{stream: stream, text: line}
	}

	if err := scanner.Err(); err != nil {
		p.mu.Lock()
		p.stderr.WriteString(err.Error())
		p.stderr.WriteString("\n")
		p.mu.Unlock()
		p.lines <- nativeExecLine{stream: "stderr", text: err.Error()}
	}
}

func (p *nativeExecProcess) waitForExit(readers *sync.WaitGroup) {
	err := p.cmd.Wait()
	readers.Wait()

	p.mu.Lock()
	p.waitErr = err
	p.exitCode, p.running = exitCodeFromError(err)
	p.waited = true
	p.mu.Unlock()

	p.closeLines.Do(func() {
		close(p.lines)
	})
	close(p.done)
}

func (p *nativeExecProcess) read() object.Object {
	line, ok := <-p.lines
	if !ok {
		return object.NullObject()
	}
	return hashObject(map[string]object.Object{
		"stream": &object.String{Value: line.stream},
		"text":   &object.String{Value: line.text},
	})
}

func (p *nativeExecProcess) wait(command string) object.Object {
	<-p.done

	p.mu.Lock()
	defer p.mu.Unlock()
	return execResultObject(p.exitCode == 0, p.exitCode, p.stdout.String(), p.stderr.String(), command, p.cmd.Dir)
}

func (p *nativeExecProcess) kill() object.Object {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running || p.cmd == nil || p.cmd.Process == nil {
		return object.NullObject()
	}
	if err := p.cmd.Process.Kill(); err != nil {
		return object.NewError(0, "could not kill process: %s", err.Error())
	}
	return object.NullObject()
}

func (p *nativeExecProcess) isRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

func exitCodeFromError(err error) (int64, bool) {
	if err == nil {
		return 0, true
	}
	if exitErr, isExitError := err.(*osexec.ExitError); isExitError {
		return int64(exitErr.ExitCode()), false
	}
	return -1, false
}
