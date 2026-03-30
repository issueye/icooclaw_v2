package parser

import (
	"fmt"

	"github.com/issueye/icooclaw_lang/internal/ast"
	"github.com/issueye/icooclaw_lang/internal/lexer"
)

type Parser struct {
	lexer          *lexer.Lexer
	errors         []string
	curToken       lexer.Token
	peekToken      lexer.Token
	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
}

type (
	prefixParseFn func() ast.Expr
	infixParseFn  func(ast.Expr) ast.Expr
)

const (
	_ int = iota
	LOWEST
	ASSIGN
	OR
	AND
	EQUALS
	LESSGREATER
	SUM
	PRODUCT
	PREFIX
	CALL
	INDEX
)

var precedences = map[lexer.TokenType]int{
	lexer.ASSIGN:   ASSIGN,
	lexer.PLUS_EQ:  ASSIGN,
	lexer.MINUS_EQ: ASSIGN,
	lexer.STAR_EQ:  ASSIGN,
	lexer.SLASH_EQ: ASSIGN,
	lexer.OR:       OR,
	lexer.AND:      AND,
	lexer.EQ:       EQUALS,
	lexer.NE:       EQUALS,
	lexer.LT:       LESSGREATER,
	lexer.GT:       LESSGREATER,
	lexer.LE:       LESSGREATER,
	lexer.GE:       LESSGREATER,
	lexer.PLUS:     SUM,
	lexer.MINUS:    SUM,
	lexer.SLASH:    PRODUCT,
	lexer.STAR:     PRODUCT,
	lexer.PERCENT:  PRODUCT,
	lexer.LPAREN:   CALL,
	lexer.LBRACKET: INDEX,
	lexer.DOT:      INDEX,
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		lexer:          l,
		errors:         []string{},
		prefixParseFns: make(map[lexer.TokenType]prefixParseFn),
		infixParseFns:  make(map[lexer.TokenType]infixParseFn),
	}

	p.registerPrefix(lexer.IDENTIFIER, p.parseIdentifier)
	p.registerPrefix(lexer.INTEGER, p.parseIntegerLiteral)
	p.registerPrefix(lexer.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(lexer.STRING, p.parseStringLiteral)
	p.registerPrefix(lexer.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.NULL, p.parseNullLiteral)
	p.registerPrefix(lexer.BANG, p.parsePrefixExpr)
	p.registerPrefix(lexer.MINUS, p.parsePrefixExpr)
	p.registerPrefix(lexer.LPAREN, p.parseGroupedExpr)
	p.registerPrefix(lexer.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(lexer.LBRACE, p.parseHashLiteral)
	p.registerPrefix(lexer.UNDERSCORE, p.parseUnderscore)
	p.registerPrefix(lexer.MATCH, p.parseMatchExpr)

	p.registerInfix(lexer.PLUS, p.parseInfixExpr)
	p.registerInfix(lexer.MINUS, p.parseInfixExpr)
	p.registerInfix(lexer.SLASH, p.parseInfixExpr)
	p.registerInfix(lexer.STAR, p.parseInfixExpr)
	p.registerInfix(lexer.PERCENT, p.parseInfixExpr)
	p.registerInfix(lexer.EQ, p.parseInfixExpr)
	p.registerInfix(lexer.NE, p.parseInfixExpr)
	p.registerInfix(lexer.LT, p.parseInfixExpr)
	p.registerInfix(lexer.GT, p.parseInfixExpr)
	p.registerInfix(lexer.LE, p.parseInfixExpr)
	p.registerInfix(lexer.GE, p.parseInfixExpr)
	p.registerInfix(lexer.AND, p.parseInfixExpr)
	p.registerInfix(lexer.OR, p.parseInfixExpr)
	p.registerInfix(lexer.ASSIGN, p.parseAssignExpr)
	p.registerInfix(lexer.PLUS_EQ, p.parseCompoundAssignExpr)
	p.registerInfix(lexer.MINUS_EQ, p.parseCompoundAssignExpr)
	p.registerInfix(lexer.STAR_EQ, p.parseCompoundAssignExpr)
	p.registerInfix(lexer.SLASH_EQ, p.parseCompoundAssignExpr)
	p.registerInfix(lexer.LPAREN, p.parseCallExpr)
	p.registerInfix(lexer.LBRACKET, p.parseIndexExpr)
	p.registerInfix(lexer.DOT, p.parseDotExpr)

	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}

	for p.curToken.Type != lexer.EOF {
		if p.curToken.Type == lexer.NEWLINE || p.curToken.Type == lexer.SEMICOLON {
			p.nextToken()
			continue
		}
		prevErrorCount := len(p.errors)
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		if p.curToken.Type == lexer.RBRACE || p.curToken.Type == lexer.RBRACKET || p.curToken.Type == lexer.RPAREN {
			p.nextToken()
		}
		p.skipNewlines()
		if len(p.errors) > prevErrorCount && stmt == nil {
			p.synchronize()
		}
	}

	return program
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

func (p *Parser) skipNewlines() {
	for p.curToken.Type == lexer.NEWLINE || p.curToken.Type == lexer.SEMICOLON {
		p.nextToken()
	}
}

func (p *Parser) finishStatement() {
	if p.peekTokenIs(lexer.NEWLINE) || p.peekTokenIs(lexer.SEMICOLON) ||
		p.peekTokenIs(lexer.RBRACE) || p.peekTokenIs(lexer.EOF) {
		p.nextToken()
	}
}

func (p *Parser) synchronize() {
	for p.curToken.Type != lexer.EOF {
		if p.curToken.Type == lexer.NEWLINE || p.curToken.Type == lexer.SEMICOLON {
			p.nextToken()
			return
		}
		if p.curToken.Type == lexer.RBRACE {
			return
		}
		switch p.curToken.Type {
		case lexer.FN, lexer.IF, lexer.FOR, lexer.WHILE, lexer.MATCH,
			lexer.RETURN, lexer.CONST, lexer.TRY, lexer.IMPORT, lexer.EXPORT, lexer.GO:
			return
		}
		p.nextToken()
	}
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("line %d: expected next token to be %s, got %s instead (peek=%s)",
		p.curToken.Line, t, p.peekToken.Type, p.peekToken.Literal)
	p.errors = append(p.errors, msg)
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) registerPrefix(tokenType lexer.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType lexer.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) peekPrecedence() int {
	if pr, ok := precedences[p.peekToken.Type]; ok {
		return pr
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if pr, ok := precedences[p.curToken.Type]; ok {
		return pr
	}
	return LOWEST
}
