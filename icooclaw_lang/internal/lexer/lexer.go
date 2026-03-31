package lexer

import "strings"

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
	line         int
	column       int
	pendingNL    bool
	pendingLine  int
	pendingCol   int
}

func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() Token {
	var tok Token

	if l.pendingNL {
		l.pendingNL = false
		return Token{
			Type:    NEWLINE,
			Literal: "\n",
			Line:    l.pendingLine,
			Column:  l.pendingCol,
		}
	}

	l.skipIgnored()
	if l.pendingNL {
		l.pendingNL = false
		return Token{
			Type:    NEWLINE,
			Literal: "\n",
			Line:    l.pendingLine,
			Column:  l.pendingCol,
		}
	}

	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: EQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(ASSIGN, l.ch, l.line, l.column)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: NE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(BANG, l.ch, l.line, l.column)
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: LE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(LT, l.ch, l.line, l.column)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: GE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(GT, l.ch, l.line, l.column)
		}
	case '+':
		if l.peekChar() == '+' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: PLUS_PLUS, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: PLUS_EQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(PLUS, l.ch, l.line, l.column)
		}
	case '-':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: ARROW, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else if l.peekChar() == '-' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: MINUS_MINUS, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: MINUS_EQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(MINUS, l.ch, l.line, l.column)
		}
	case '*':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: STAR_EQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(STAR, l.ch, l.line, l.column)
		}
	case '/':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: SLASH_EQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(SLASH, l.ch, l.line, l.column)
		}
	case '%':
		tok = newToken(PERCENT, l.ch, l.line, l.column)
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: AND, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: OR, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(PIPE, l.ch, l.line, l.column)
		}
	case '(':
		tok = newToken(LPAREN, l.ch, l.line, l.column)
	case ')':
		tok = newToken(RPAREN, l.ch, l.line, l.column)
	case '{':
		tok = newToken(LBRACE, l.ch, l.line, l.column)
	case '}':
		tok = newToken(RBRACE, l.ch, l.line, l.column)
	case '[':
		tok = newToken(LBRACKET, l.ch, l.line, l.column)
	case ']':
		tok = newToken(RBRACKET, l.ch, l.line, l.column)
	case ',':
		tok = newToken(COMMA, l.ch, l.line, l.column)
	case ':':
		tok = newToken(COLON, l.ch, l.line, l.column)
	case '.':
		tok = newToken(DOT, l.ch, l.line, l.column)
	case ';':
		tok = newToken(SEMICOLON, l.ch, l.line, l.column)
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString()
		tok.Line = l.line
		tok.Column = l.column
	case '\n':
		tok = newToken(NEWLINE, l.ch, l.line, l.column)
		l.line++
		l.column = 0
	case 0:
		tok.Literal = ""
		tok.Type = EOF
	default:
		if isLetter(l.ch) {
			ident := l.readIdentifier()
			tok.Literal = ident
			tok.Type = LookupIdent(ident)
			tok.Line = l.line
			tok.Column = l.column - len(ident)
			return tok
		} else if isDigit(l.ch) {
			num := l.readNumber()
			tok.Line = l.line
			tok.Column = l.column - len(num)
			if strings.Contains(num, ".") {
				tok.Type = FLOAT
			} else {
				tok.Type = INTEGER
			}
			tok.Literal = num
			return tok
		} else {
			tok = newToken(ILLEGAL, l.ch, l.line, l.column)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) Tokenize() []Token {
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}
	return tokens
}

func newToken(tokenType TokenType, ch byte, line, column int) Token {
	return Token{Type: tokenType, Literal: string(ch), Line: line, Column: column}
}

func (l *Lexer) readIdentifier() string {
	pos := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[pos:l.position]
}

func (l *Lexer) readNumber() string {
	pos := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	return l.input[pos:l.position]
}

func (l *Lexer) readString() string {
	l.readChar()
	pos := l.position
	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
		}
		l.readChar()
	}
	result := l.input[pos:l.position]
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(result, "\\n", "\n"), "\\t", "\t"), "\\\"", "\"")
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) skipIgnored() {
	for {
		l.skipWhitespace()
		if !l.skipComment() {
			return
		}
		if l.pendingNL {
			return
		}
	}
}

func (l *Lexer) skipComment() bool {
	if l.ch == '#' || (l.ch == '/' && l.peekChar() == '/') {
		for l.ch != '\n' && l.ch != 0 {
			l.readChar()
		}
		return true
	}

	if l.ch == '/' && l.peekChar() == '*' {
		l.skipBlockComment()
		return true
	}

	return false
}

func (l *Lexer) skipBlockComment() {
	l.readCommentChar()
	l.readCommentChar()

	sawNewline := false
	firstNewlineLine := 0
	firstNewlineColumn := 0

	for l.ch != 0 {
		if l.ch == '\n' {
			if !sawNewline {
				sawNewline = true
				firstNewlineLine = l.line
				firstNewlineColumn = l.column
			}
			l.readCommentChar()
			continue
		}

		if l.ch == '*' && l.peekChar() == '/' {
			l.readCommentChar()
			l.readCommentChar()
			break
		}

		l.readCommentChar()
	}

	if sawNewline {
		l.pendingNL = true
		l.pendingLine = firstNewlineLine
		l.pendingCol = firstNewlineColumn
	}
}

func (l *Lexer) readCommentChar() {
	if l.ch == '\n' {
		l.line++
		l.column = 0
	}
	l.readChar()
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
