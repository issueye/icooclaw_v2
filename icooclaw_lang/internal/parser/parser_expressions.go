package parser

import (
	"fmt"
	"strconv"

	"github.com/issueye/icooclaw_lang/internal/ast"
	"github.com/issueye/icooclaw_lang/internal/lexer"
)

func (p *Parser) parseExpr(precedence int) ast.Expr {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		msg := fmt.Sprintf("line %d: no prefix parse function for %s found", p.curToken.Line, p.curToken.Type)
		p.errors = append(p.errors, msg)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(lexer.NEWLINE) &&
		!p.peekTokenIs(lexer.SEMICOLON) &&
		!p.peekTokenIs(lexer.EOF) &&
		!p.peekTokenIs(lexer.RBRACE) &&
		precedence < p.peekPrecedence() {
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

func (p *Parser) parseMatchExpr() ast.Expr {
	return p.parseMatchStmt()
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
	if expr.Right == nil {
		return nil
	}
	return expr
}

func (p *Parser) parseAssignExpr(left ast.Expr) ast.Expr {
	expr := &ast.AssignExpr{
		Token: p.curToken,
		Left:  left,
	}
	p.nextToken()
	expr.Right = p.parseExpr(ASSIGN - 1)
	if expr.Right == nil {
		return nil
	}
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
	if expr.Right == nil {
		return nil
	}
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
