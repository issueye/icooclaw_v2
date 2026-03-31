package lexer

import "testing"

func TestSlashSlashCommentsAreIgnored(t *testing.T) {
	l := New("x = 1 // comment\ny = 2\n")

	var types []TokenType
	for {
		tok := l.NextToken()
		types = append(types, tok.Type)
		if tok.Type == EOF {
			break
		}
	}

	expected := []TokenType{
		IDENTIFIER, ASSIGN, INTEGER, NEWLINE,
		IDENTIFIER, ASSIGN, INTEGER, NEWLINE,
		EOF,
	}

	if len(types) != len(expected) {
		t.Fatalf("token count mismatch: got %d want %d (%v)", len(types), len(expected), types)
	}

	for i := range expected {
		if types[i] != expected[i] {
			t.Fatalf("token %d = %s, want %s", i, types[i], expected[i])
		}
	}
}

func TestBlockCommentsPreserveStatementBoundary(t *testing.T) {
	l := New("x = 1\n/* block\ncomment */\ny = 2\n")

	var newlineCount int
	for {
		tok := l.NextToken()
		if tok.Type == NEWLINE {
			newlineCount++
		}
		if tok.Type == EOF {
			break
		}
	}

	if newlineCount < 3 {
		t.Fatalf("expected block comment to preserve newline separation, got %d newlines", newlineCount)
	}
}
