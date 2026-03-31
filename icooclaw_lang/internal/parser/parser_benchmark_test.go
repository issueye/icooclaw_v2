package parser

import (
	"testing"

	"github.com/issueye/icooclaw_lang/internal/lexer"
)

const benchmarkProgram = `
fn fib(n) {
    if n <= 1 {
        return n
    }
    return fib(n - 1) + fib(n - 2)
}

payload = {
    "name": "icooclaw",
    "items": [1, 2, 3, 4, 5],
    "meta": {"ok": true, "version": "1.0.0"},
}

result = match payload {
    {"name": name, "items": [a, b, c, d, e]} -> name + ":" + str(a + e)
    _ -> "unknown"
}

for i in range(100) {
    value = i * 2 + 1
}
`

func BenchmarkParseProgram(b *testing.B) {
	b.SetBytes(int64(len(benchmarkProgram)))
	b.ReportAllocs()
	for b.Loop() {
		l := lexer.New(benchmarkProgram)
		p := New(l)
		program := p.ParseProgram()
		if program == nil {
			b.Fatal("expected parsed program")
		}
		if errs := p.Errors(); len(errs) != 0 {
			b.Fatalf("unexpected parser errors: %v", errs)
		}
	}
}

func BenchmarkLexProgram(b *testing.B) {
	b.SetBytes(int64(len(benchmarkProgram)))
	b.ReportAllocs()
	for b.Loop() {
		l := lexer.New(benchmarkProgram)
		tokens := l.Tokenize()
		if len(tokens) == 0 {
			b.Fatal("expected tokens")
		}
	}
}

func BenchmarkParseLargeProgram(b *testing.B) {
	source := benchmarkProgram + "\n" + benchmarkProgram + "\n" + benchmarkProgram

	b.SetBytes(int64(len(source)))
	b.ReportAllocs()
	for b.Loop() {
		l := lexer.New(source)
		p := New(l)
		program := p.ParseProgram()
		if program == nil {
			b.Fatal("expected parsed program")
		}
		if errs := p.Errors(); len(errs) != 0 {
			b.Fatalf("unexpected parser errors: %v", errs)
		}
	}
}
