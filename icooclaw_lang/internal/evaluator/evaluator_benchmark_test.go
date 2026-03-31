package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/issueye/icooclaw_lang/internal/ast"
	"github.com/issueye/icooclaw_lang/internal/lexer"
	"github.com/issueye/icooclaw_lang/internal/object"
	"github.com/issueye/icooclaw_lang/internal/parser"
)

const benchmarkEvalProgram = `
fn sum_to(n) {
    total = 0
    for i in range(n) {
        total += i
    }
    return total
}

value = sum_to(1000)
payload = {"kind": "ok", "value": value}
result = match payload {
    {"kind": "ok", "value": v} if v > 10 -> v
    _ -> 0
}
`

const benchmarkCallProgram = `
fn inc(n) {
    return n + 1
}

value = 0
for i in range(500) {
    value = inc(value)
}
`

const benchmarkJSONProgram = `
payload = {
    "name": "icooclaw",
    "items": [1, 2, 3, 4, 5],
    "meta": {"ok": true, "version": "1.0.0"},
}
encoded = json.stringify(payload)
decoded = json.parse(encoded)
value = decoded.items[4]
`

func BenchmarkEvalProgram(b *testing.B) {
	program := mustParseBenchmarkProgram(b, benchmarkEvalProgram)

	b.SetBytes(int64(len(benchmarkEvalProgram)))
	b.ReportAllocs()
	for b.Loop() {
		env := object.NewEnvironment()
		result := Eval(program, env)
		env.Wait()
		if object.IsError(result) {
			b.Fatalf("unexpected eval error: %s", result.Inspect())
		}
	}
}

func BenchmarkEvalFunctionCalls(b *testing.B) {
	program := mustParseBenchmarkProgram(b, benchmarkCallProgram)

	b.SetBytes(int64(len(benchmarkCallProgram)))
	b.ReportAllocs()
	for b.Loop() {
		env := object.NewEnvironment()
		result := Eval(program, env)
		env.Wait()
		if object.IsError(result) {
			b.Fatalf("unexpected eval error: %s", result.Inspect())
		}
	}
}

func BenchmarkEvalJSONRoundTrip(b *testing.B) {
	program := mustParseBenchmarkProgram(b, benchmarkJSONProgram)

	b.SetBytes(int64(len(benchmarkJSONProgram)))
	b.ReportAllocs()
	for b.Loop() {
		env := object.NewEnvironment()
		result := Eval(program, env)
		env.Wait()
		if object.IsError(result) {
			b.Fatalf("unexpected eval error: %s", result.Inspect())
		}
	}
}

func BenchmarkEvalModuleImportCold(b *testing.B) {
	rootDir := b.TempDir()
	modulePath := filepath.Join(rootDir, "math.is")
	if err := os.WriteFile(modulePath, []byte(`
fn add(a, b) {
    return a + b
}

export add
`), 0o644); err != nil {
		b.Fatalf("write module: %v", err)
	}

	program := mustParseBenchmarkProgram(b, `
import "./math.is"
value = math.add(2, 3)
`)

	b.SetBytes(int64(len(`import "./math.is"`)))
	b.ReportAllocs()
	for b.Loop() {
		env := object.NewEnvironment()
		env.SetCLIContext(filepath.Join(rootDir, "main.is"), nil)
		result := Eval(program, env)
		env.Wait()
		if object.IsError(result) {
			b.Fatalf("unexpected eval error: %s", result.Inspect())
		}
	}
}

func BenchmarkEvalModuleImportWarm(b *testing.B) {
	rootDir := b.TempDir()
	modulePath := filepath.Join(rootDir, "math.is")
	if err := os.WriteFile(modulePath, []byte(`
fn add(a, b) {
    return a + b
}

export add
`), 0o644); err != nil {
		b.Fatalf("write module: %v", err)
	}

	program := mustParseBenchmarkProgram(b, `
import "./math.is"
value = math.add(2, 3)
`)

	rootEnv := object.NewEnvironment()
	rootEnv.SetCLIContext(filepath.Join(rootDir, "main.is"), nil)

	warmUp := Eval(program, rootEnv)
	rootEnv.Wait()
	if object.IsError(warmUp) {
		b.Fatalf("unexpected warmup error: %s", warmUp.Inspect())
	}

	b.SetBytes(int64(len(`import "./math.is"`)))
	b.ReportAllocs()
	for b.Loop() {
		env := object.NewDetachedEnvironment(rootEnv)
		result := Eval(program, env)
		env.Wait()
		if object.IsError(result) {
			b.Fatalf("unexpected eval error: %s", result.Inspect())
		}
	}
}

func mustParseBenchmarkProgram(b *testing.B, input string) *ast.Program {
	b.Helper()

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		b.Fatalf("unexpected parser errors: %v", errs)
	}
	return program
}
