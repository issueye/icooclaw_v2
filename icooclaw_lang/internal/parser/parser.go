package parser

import (
	"fmt"
	"strconv"

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
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.skipNewlines()
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

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("line %d: expected next token to be %s, got %s instead",
		p.curToken.Line, t, p.peekToken.Type)
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

func (p *Parser) parseStatement() ast.Stmt {
	switch p.curToken.Type {
	case lexer.FN:
		return p.parseFunctionStmt()
	case lexer.RETURN:
		return p.parseReturnStmt()
	case lexer.IF:
		return p.parseIfStmt()
	case lexer.FOR:
		return p.parseForStmt()
	case lexer.WHILE:
		return p.parseWhileStmt()
	case lexer.CONST:
		return p.parseConstStmt()
	case lexer.BREAK:
		stmt := &ast.BreakStmt{Token: p.curToken}
		p.nextToken()
		return stmt
	case lexer.CONTINUE:
		stmt := &ast.ContinueStmt{Token: p.curToken}
		p.nextToken()
		return stmt
	case lexer.MATCH:
		return p.parseMatchStmt()
	case lexer.TRY:
		return p.parseTryStmt()
	case lexer.IMPORT:
		return p.parseImportStmt()
	case lexer.EXPORT:
		return p.parseExportStmt()
	case lexer.GO:
		return p.parseGoStmt()
	default:
		return p.parseExpressionStmt()
	}
}

func (p *Parser) parseFunctionStmt() *ast.FunctionStmt {
	stmt := &ast.FunctionStmt{Token: p.curToken}
	p.nextToken()

	if !p.curTokenIs(lexer.IDENTIFIER) {
		msg := fmt.Sprintf("line %d: expected function name, got %s", p.curToken.Line, p.curToken.Type)
		p.errors = append(p.errors, msg)
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}
	stmt.Params = p.parseParams()

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStmt()

	return stmt
}

func (p *Parser) parseParams() []*ast.Identifier {
	var params []*ast.Identifier

	p.nextToken()
	if p.curTokenIs(lexer.RPAREN) {
		return params
	}

	params = append(params, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		params = append(params, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return params
}

func (p *Parser) parseReturnStmt() *ast.ReturnStmt {
	stmt := &ast.ReturnStmt{Token: p.curToken}
	p.nextToken()

	if p.curTokenIs(lexer.NEWLINE) || p.curTokenIs(lexer.SEMICOLON) || p.curTokenIs(lexer.RBRACE) || p.curTokenIs(lexer.EOF) {
		return stmt
	}

	stmt.ReturnValue = p.parseExpr(LOWEST)
	return stmt
}

func (p *Parser) parseIfStmt() *ast.IfStmt {
	stmt := &ast.IfStmt{Token: p.curToken}
	p.nextToken()

	stmt.Condition = p.parseExpr(LOWEST)

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}
	stmt.Consequence = p.parseBlockStmt()

	p.skipNewlines()

	if p.curTokenIs(lexer.ELSE) {
		p.nextToken()
		if !p.expectPeek(lexer.LBRACE) {
			return nil
		}
		stmt.Alternative = p.parseBlockStmt()
	}

	return stmt
}

func (p *Parser) parseForStmt() *ast.ForStmt {
	stmt := &ast.ForStmt{Token: p.curToken}
	p.nextToken()

	if !p.curTokenIs(lexer.IDENTIFIER) {
		msg := fmt.Sprintf("line %d: expected identifier after 'for', got %s", p.curToken.Line, p.curToken.Type)
		p.errors = append(p.errors, msg)
		return nil
	}
	stmt.Ident = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.IN) {
		return nil
	}
	p.nextToken()

	stmt.Iterable = p.parseExpr(LOWEST)

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStmt()

	return stmt
}

func (p *Parser) parseWhileStmt() *ast.WhileStmt {
	stmt := &ast.WhileStmt{Token: p.curToken}
	p.nextToken()

	stmt.Condition = p.parseExpr(LOWEST)

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStmt()

	return stmt
}

func (p *Parser) parseBlockStmt() *ast.BlockStmt {
	block := &ast.BlockStmt{Token: p.curToken}
	p.nextToken()
	p.skipNewlines()

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.NEWLINE) || p.curTokenIs(lexer.SEMICOLON) {
			p.nextToken()
			continue
		}
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.skipNewlines()
	}

	if p.curTokenIs(lexer.RBRACE) {
		p.nextToken()
	}

	return block
}

func (p *Parser) parseConstStmt() *ast.ConstStmt {
	stmt := &ast.ConstStmt{Token: p.curToken}
	p.nextToken()

	if !p.curTokenIs(lexer.IDENTIFIER) {
		msg := fmt.Sprintf("line %d: expected identifier after 'const', got %s", p.curToken.Line, p.curToken.Type)
		p.errors = append(p.errors, msg)
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}
	p.nextToken()

	stmt.Value = p.parseExpr(LOWEST)
	return stmt
}

func (p *Parser) parseMatchStmt() *ast.MatchStmt {
	stmt := &ast.MatchStmt{Token: p.curToken}
	p.nextToken()

	stmt.Subject = p.parseExpr(LOWEST)

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}
	p.skipNewlines()

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		mc := p.parseMatchCase()
		if mc != nil {
			stmt.Cases = append(stmt.Cases, *mc)
		}
		p.skipNewlines()
	}

	if p.curTokenIs(lexer.RBRACE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseMatchCase() *ast.MatchCase {
	mc := &ast.MatchCase{Token: p.curToken}

	mc.Patterns = append(mc.Patterns, p.parseExpr(LOWEST))

	for p.curTokenIs(lexer.PIPE) {
		p.nextToken()
		mc.Patterns = append(mc.Patterns, p.parseExpr(LOWEST))
	}

	if !p.curTokenIs(lexer.ARROW) {
		msg := fmt.Sprintf("line %d: expected '->' in match case, got %s", p.curToken.Line, p.curToken.Type)
		p.errors = append(p.errors, msg)
		return nil
	}
	p.nextToken()

	mc.Result = p.parseExpr(LOWEST)
	return mc
}

func (p *Parser) parseTryStmt() *ast.TryStmt {
	stmt := &ast.TryStmt{Token: p.curToken}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}
	stmt.TryBlock = p.parseBlockStmt()
	p.skipNewlines()

	if !p.curTokenIs(lexer.CATCH) {
		msg := fmt.Sprintf("line %d: expected 'catch' after try block, got %s", p.curToken.Line, p.curToken.Type)
		p.errors = append(p.errors, msg)
		return nil
	}
	p.nextToken()

	if !p.curTokenIs(lexer.IDENTIFIER) {
		msg := fmt.Sprintf("line %d: expected identifier after 'catch', got %s", p.curToken.Line, p.curToken.Type)
		p.errors = append(p.errors, msg)
		return nil
	}
	stmt.CatchVar = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}
	stmt.CatchBlock = p.parseBlockStmt()

	return stmt
}

func (p *Parser) parseImportStmt() *ast.ImportStmt {
	stmt := &ast.ImportStmt{Token: p.curToken}
	p.nextToken()

	stmt.Module = p.parseExpr(LOWEST)

	if p.curTokenIs(lexer.IDENTIFIER) && p.curToken.Literal == "as" {
		p.nextToken()
		if p.curTokenIs(lexer.IDENTIFIER) {
			stmt.Alias = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			p.nextToken()
		}
	}

	return stmt
}

func (p *Parser) parseExportStmt() *ast.ExportStmt {
	stmt := &ast.ExportStmt{Token: p.curToken}
	p.nextToken()

	if !p.curTokenIs(lexer.IDENTIFIER) {
		msg := fmt.Sprintf("line %d: expected identifier after 'export', got %s", p.curToken.Line, p.curToken.Type)
		p.errors = append(p.errors, msg)
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	p.nextToken()
	return stmt
}

func (p *Parser) parseGoStmt() *ast.GoStmt {
	stmt := &ast.GoStmt{Token: p.curToken}
	p.nextToken()
	stmt.Call = p.parseExpr(LOWEST)
	return stmt
}

func (p *Parser) parseExpressionStmt() *ast.ExpressionStmt {
	stmt := &ast.ExpressionStmt{Token: p.curToken}
	stmt.Expr = p.parseExpr(LOWEST)

	if p.peekTokenIs(lexer.NEWLINE) || p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

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

func (p *Parser) parseExpr(precedence int) ast.Expr {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		msg := fmt.Sprintf("line %d: no prefix parse function for %s found", p.curToken.Line, p.curToken.Type)
		p.errors = append(p.errors, msg)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(lexer.NEWLINE) && !p.peekTokenIs(lexer.SEMICOLON) && !p.peekTokenIs(lexer.EOF) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}
		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() ast.Expr {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expr {
	lit := &ast.IntegerLiteral{Token: p.curToken}
	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("line %d: could not parse %q as integer", p.curToken.Line, p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expr {
	lit := &ast.FloatLiteral{Token: p.curToken}
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		msg := fmt.Sprintf("line %d: could not parse %q as float", p.curToken.Line, p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expr {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() ast.Expr {
	return &ast.BooleanLiteral{Token: p.curToken, Value: p.curToken.Type == lexer.TRUE}
}

func (p *Parser) parseNullLiteral() ast.Expr {
	return &ast.NullLiteral{Token: p.curToken}
}

func (p *Parser) parseUnderscore() ast.Expr {
	return &ast.UnderscoreExpr{Token: p.curToken}
}

func (p *Parser) parsePrefixExpr() ast.Expr {
	expr := &ast.PrefixExpr{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}
	p.nextToken()
	expr.Right = p.parseExpr(PREFIX)
	return expr
}

func (p *Parser) parseInfixExpr(left ast.Expr) ast.Expr {
	expr := &ast.InfixExpr{
		Token:    p.curToken,
		Left:     left,
		Operator: p.curToken.Literal,
	}
	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpr(precedence)
	return expr
}

func (p *Parser) parseAssignExpr(left ast.Expr) ast.Expr {
	expr := &ast.AssignExpr{
		Token: p.curToken,
		Left:  left,
	}
	p.nextToken()
	expr.Right = p.parseExpr(ASSIGN - 1)
	return expr
}

func (p *Parser) parseCompoundAssignExpr(left ast.Expr) ast.Expr {
	expr := &ast.CompoundAssignExpr{
		Token:    p.curToken,
		Left:     left,
		Operator: p.curToken.Literal,
	}
	p.nextToken()
	expr.Right = p.parseExpr(ASSIGN - 1)
	return expr
}

func (p *Parser) parseCallExpr(function ast.Expr) ast.Expr {
	expr := &ast.CallExpr{Token: p.curToken, Function: function}
	expr.Arguments = p.parseArgs()
	return expr
}

func (p *Parser) parseArgs() []ast.Expr {
	var args []ast.Expr
	p.nextToken()

	if p.curTokenIs(lexer.RPAREN) {
		return args
	}

	args = append(args, p.parseExpr(LOWEST))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpr(LOWEST))
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return args
}

func (p *Parser) parseGroupedExpr() ast.Expr {
	p.nextToken()
	exp := p.parseExpr(LOWEST)

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}
	return exp
}

func (p *Parser) parseArrayLiteral() ast.Expr {
	array := &ast.ArrayLiteral{Token: p.curToken}
	array.Elements = p.parseExprList(lexer.RBRACKET)
	return array
}

func (p *Parser) parseExprList(end lexer.TokenType) []ast.Expr {
	var list []ast.Expr
	p.nextToken()

	if p.curTokenIs(end) {
		return list
	}

	list = append(list, p.parseExpr(LOWEST))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpr(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseHashLiteral() ast.Expr {
	hash := &ast.HashLiteral{Token: p.curToken}
	hash.Pairs = make(map[ast.Expr]ast.Expr)
	p.nextToken()

	for !p.curTokenIs(lexer.RBRACE) {
		key := p.parseExpr(LOWEST)
		if !p.expectPeek(lexer.COLON) {
			return hash
		}
		p.nextToken()
		value := p.parseExpr(LOWEST)
		hash.Pairs[key] = value

		if !p.peekTokenIs(lexer.RBRACE) && !p.expectPeek(lexer.COMMA) {
			return hash
		}
		p.nextToken()
	}

	return hash
}

func (p *Parser) parseIndexExpr(left ast.Expr) ast.Expr {
	expr := &ast.IndexExpr{Token: p.curToken, Left: left}
	p.nextToken()
	expr.Index = p.parseExpr(LOWEST)

	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}
	return expr
}

func (p *Parser) parseDotExpr(left ast.Expr) ast.Expr {
	p.nextToken()

	if !p.curTokenIs(lexer.IDENTIFIER) {
		msg := fmt.Sprintf("line %d: expected identifier after '.', got %s", p.curToken.Line, p.curToken.Type)
		p.errors = append(p.errors, msg)
		return nil
	}

	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken()
		return &ast.MethodCallExpr{
			Token:     p.curToken,
			Object:    left,
			Method:    ident,
			Arguments: p.parseArgs(),
		}
	}

	return &ast.DotExpr{Token: p.curToken, Left: left, Right: ident}
}
