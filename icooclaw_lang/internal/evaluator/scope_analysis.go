package evaluator

import "github.com/issueye/icooclaw_lang/internal/ast"

func blockAllowsTransientReuse(block *ast.BlockStmt) bool {
	if block == nil {
		return true
	}
	for _, stmt := range block.Statements {
		if stmtCreatesEscapingScope(stmt) {
			return false
		}
	}
	return true
}

func stmtCreatesEscapingScope(stmt ast.Stmt) bool {
	switch stmt := stmt.(type) {
	case *ast.FunctionStmt:
		return true
	case *ast.ExpressionStmt:
		return exprCreatesEscapingScope(stmt.Expr)
	case *ast.LetStmt:
		return exprCreatesEscapingScope(stmt.Value)
	case *ast.ConstStmt:
		return exprCreatesEscapingScope(stmt.Value)
	case *ast.ReturnStmt:
		return exprCreatesEscapingScope(stmt.ReturnValue)
	case *ast.GoStmt:
		return true
	case *ast.IfStmt:
		return exprCreatesEscapingScope(stmt.Condition) ||
			!blockAllowsTransientReuse(stmt.Consequence) ||
			!blockAllowsTransientReuse(stmt.Alternative)
	case *ast.ForStmt:
		return exprCreatesEscapingScope(stmt.Iterable) || !blockAllowsTransientReuse(stmt.Body)
	case *ast.WhileStmt:
		return exprCreatesEscapingScope(stmt.Condition) || !blockAllowsTransientReuse(stmt.Body)
	case *ast.TryStmt:
		return !blockAllowsTransientReuse(stmt.TryBlock) || !blockAllowsTransientReuse(stmt.CatchBlock)
	default:
		return false
	}
}

func exprCreatesEscapingScope(expr ast.Expr) bool {
	switch expr := expr.(type) {
	case nil:
		return false
	case *ast.FunctionLiteral:
		return true
	case *ast.PrefixExpr:
		return exprCreatesEscapingScope(expr.Right)
	case *ast.InfixExpr:
		return exprCreatesEscapingScope(expr.Left) || exprCreatesEscapingScope(expr.Right)
	case *ast.AssignExpr:
		return exprCreatesEscapingScope(expr.Left) || exprCreatesEscapingScope(expr.Right)
	case *ast.CompoundAssignExpr:
		return exprCreatesEscapingScope(expr.Left) || exprCreatesEscapingScope(expr.Right)
	case *ast.PostfixExpr:
		return exprCreatesEscapingScope(expr.Left)
	case *ast.CallExpr:
		if exprCreatesEscapingScope(expr.Function) {
			return true
		}
		for _, arg := range expr.Arguments {
			if exprCreatesEscapingScope(arg) {
				return true
			}
		}
		return false
	case *ast.ArrayLiteral:
		for _, item := range expr.Elements {
			if exprCreatesEscapingScope(item) {
				return true
			}
		}
		return false
	case *ast.HashLiteral:
		for key, value := range expr.Pairs {
			if exprCreatesEscapingScope(key) || exprCreatesEscapingScope(value) {
				return true
			}
		}
		return false
	case *ast.IndexExpr:
		return exprCreatesEscapingScope(expr.Left) || exprCreatesEscapingScope(expr.Index)
	case *ast.DotExpr:
		return exprCreatesEscapingScope(expr.Left)
	case *ast.MethodCallExpr:
		if exprCreatesEscapingScope(expr.Object) {
			return true
		}
		for _, arg := range expr.Arguments {
			if exprCreatesEscapingScope(arg) {
				return true
			}
		}
		return false
	case *ast.MatchStmt:
		if exprCreatesEscapingScope(expr.Subject) {
			return true
		}
		for _, c := range expr.Cases {
			for _, pattern := range c.Patterns {
				if exprCreatesEscapingScope(pattern) {
					return true
				}
			}
			if exprCreatesEscapingScope(c.Guard) || exprCreatesEscapingScope(c.Result) {
				return true
			}
		}
		return false
	default:
		return false
	}
}
