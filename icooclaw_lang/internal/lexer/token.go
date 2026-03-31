package lexer

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	IDENTIFIER = "IDENTIFIER"
	INTEGER    = "INTEGER"
	FLOAT      = "FLOAT"
	STRING     = "STRING"

	FN         = "fn"
	RETURN     = "return"
	IF         = "if"
	ELSE       = "else"
	FOR        = "for"
	WHILE      = "while"
	MATCH      = "match"
	BREAK      = "break"
	CONTINUE   = "continue"
	CONST      = "const"
	IMPORT     = "import"
	EXPORT     = "export"
	TRY        = "try"
	CATCH      = "catch"
	GO         = "go"
	SELECT     = "select"
	INTERFACE  = "interface"
	TYPE       = "type"
	NULL       = "null"
	TRUE       = "true"
	FALSE      = "false"
	IN         = "in"
	UNDERSCORE = "UNDERSCORE"

	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	STAR     = "*"
	SLASH    = "/"
	PERCENT  = "%"
	BANG     = "!"
	LT       = "<"
	GT       = ">"
	EQ       = "=="
	NE       = "!="
	LE       = "<="
	GE       = ">="
	AND      = "&&"
	OR       = "||"
	PLUS_EQ  = "+="
	MINUS_EQ = "-="
	STAR_EQ  = "*="
	SLASH_EQ = "/="
	PLUS_PLUS = "++"
	MINUS_MINUS = "--"
	ARROW    = "->"

	LPAREN    = "("
	RPAREN    = ")"
	LBRACE    = "{"
	RBRACE    = "}"
	LBRACKET  = "["
	RBRACKET  = "]"
	COMMA     = ","
	COLON     = ":"
	DOT       = "."
	SAFE_DOT  = "?."
	SAFE_LBRACKET = "?["
	SEMICOLON = ";"
	PIPE      = "|"
	NEWLINE   = "\n"
)

var keywords = map[string]TokenType{
	"fn":        FN,
	"return":    RETURN,
	"if":        IF,
	"else":      ELSE,
	"for":       FOR,
	"while":     WHILE,
	"match":     MATCH,
	"break":     BREAK,
	"continue":  CONTINUE,
	"const":     CONST,
	"import":    IMPORT,
	"export":    EXPORT,
	"try":       TRY,
	"catch":     CATCH,
	"go":        GO,
	"null":      NULL,
	"true":      TRUE,
	"false":     FALSE,
	"in":        IN,
	"_":         UNDERSCORE,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENTIFIER
}
