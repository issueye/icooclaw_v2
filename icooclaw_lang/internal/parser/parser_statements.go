package parser

import (
	"fmt"

	"github.com/issueye/icooclaw_lang/internal/ast"
	"github.com/issueye/icooclaw_lang/internal/lexer"
)

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
	case lexer.NEWLINE, lexer.SEMICOLON:
		p.nextToken()
		return nil
	case lexer.RBRACE, lexer.RBRACKET, lexer.RPAREN:
		return nil
	case lexer.EOF:
		return nil
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
	p.skipNewlines()
	if p.curTokenIs(lexer.RPAREN) {
		return params
	}

	params = append(params, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})

	for {
		p.skipPeekNewlines()
		if p.peekTokenIs(lexer.RPAREN) {
			p.nextToken()
			return params
		}
		if !p.peekTokenIs(lexer.COMMA) {
			break
		}
		p.nextToken()
		p.nextToken()
		p.skipNewlines()
		if p.curTokenIs(lexer.RPAREN) {
			return params
		}
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
	p.finishStatement()
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

	if p.curToken.Type == lexer.ELSE {
		if p.peekToken.Type == lexer.IF {
			p.nextToken()
			nestedIf := p.parseIfStmt()
			stmt.Alternative = &ast.BlockStmt{
				Statements: []ast.Stmt{nestedIf},
			}
		} else {
			if !p.expectPeek(lexer.LBRACE) {
				return nil
			}
			stmt.Alternative = p.parseBlockStmt()
		}
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
	if p.curToken.Type == lexer.LBRACE {
		p.nextToken()
	}
	p.skipNewlines()

	for p.curToken.Type != lexer.RBRACE && p.curToken.Type != lexer.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		if p.curToken.Type == lexer.RBRACE {
			break
		}
		p.skipNewlines()
		if p.curToken.Type != lexer.RBRACE && p.curToken.Type != lexer.EOF && stmt == nil {
			p.nextToken()
		}
	}

	if p.curToken.Type == lexer.RBRACE {
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
	p.finishStatement()
	return stmt
}

func (p *Parser) parseMatchStmt() *ast.MatchStmt {
	stmt := &ast.MatchStmt{Token: p.curToken}
	p.nextToken()

	stmt.Subject = p.parseExpr(LOWEST)
	if stmt.Subject == nil {
		return nil
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}
	p.nextToken()
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

	pattern := p.parseExpr(LOWEST)
	if pattern == nil {
		return nil
	}
	mc.Patterns = append(mc.Patterns, pattern)

	for p.peekTokenIs(lexer.PIPE) {
		p.nextToken()
		p.nextToken()
		pattern = p.parseExpr(LOWEST)
		if pattern == nil {
			return nil
		}
		mc.Patterns = append(mc.Patterns, pattern)
	}

	if p.peekTokenIs(lexer.IF) {
		p.nextToken()
		p.nextToken()
		mc.Guard = p.parseExpr(LOWEST)
		if mc.Guard == nil {
			return nil
		}
	}

	if !p.expectPeek(lexer.ARROW) {
		return nil
	}
	p.nextToken()

	mc.Result = p.parseExpr(LOWEST)
	if mc.Result == nil {
		return nil
	}

	p.finishStatement()
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

	if p.curTokenIs(lexer.LBRACE) {
		stmt.Names = p.parseImportNames()
		if stmt.Names == nil {
			return nil
		}
		if !p.curTokenIs(lexer.IDENTIFIER) || p.curToken.Literal != "from" {
			msg := fmt.Sprintf("line %d: expected 'from' after import list, got %s", p.curToken.Line, p.curToken.Literal)
			p.errors = append(p.errors, msg)
			return nil
		}
		p.nextToken()
	}

	stmt.Module = p.parseExpr(LOWEST)
	if stmt.Module == nil {
		return nil
	}

	if p.peekTokenIs(lexer.IDENTIFIER) && p.peekToken.Literal == "as" {
		p.nextToken()
		p.nextToken()
		if p.curTokenIs(lexer.IDENTIFIER) {
			stmt.Alias = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		}
	}

	p.finishStatement()
	return stmt
}

func (p *Parser) parseImportNames() []*ast.Identifier {
	var names []*ast.Identifier

	p.nextToken()
	p.skipNewlines()

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		if !p.curTokenIs(lexer.IDENTIFIER) {
			msg := fmt.Sprintf("line %d: expected identifier in import list, got %s", p.curToken.Line, p.curToken.Type)
			p.errors = append(p.errors, msg)
			return nil
		}
		names = append(names, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})

		p.skipPeekNewlines()
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			p.nextToken()
			p.skipNewlines()
			continue
		}
		if p.peekTokenIs(lexer.RBRACE) {
			p.nextToken()
			p.nextToken()
			return names
		}

		msg := fmt.Sprintf("line %d: expected ',' or '}' in import list, got %s", p.curToken.Line, p.peekToken.Type)
		p.errors = append(p.errors, msg)
		return nil
	}

	msg := fmt.Sprintf("line %d: unterminated import list", p.curToken.Line)
	p.errors = append(p.errors, msg)
	return nil
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
	p.finishStatement()
	return stmt
}

func (p *Parser) parseGoStmt() *ast.GoStmt {
	stmt := &ast.GoStmt{Token: p.curToken}
	p.nextToken()
	stmt.Call = p.parseExpr(LOWEST)
	p.finishStatement()
	return stmt
}

func (p *Parser) parseExpressionStmt() *ast.ExpressionStmt {
	stmt := &ast.ExpressionStmt{Token: p.curToken}
	stmt.Expr = p.parseExpr(LOWEST)
	if stmt.Expr == nil {
		return nil
	}
	p.finishStatement()
	return stmt
}
