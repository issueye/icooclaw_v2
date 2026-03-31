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
	case *ast.GoStmt:
		return true
	case *ast.IfStmt:
		if blockAllowsTransientReuse(stmt.Consequence) {
			return !blockAllowsTransientReuse(stmt.Alternative)
		}
		return true
	case *ast.ForStmt:
		return !blockAllowsTransientReuse(stmt.Body)
	case *ast.WhileStmt:
		return !blockAllowsTransientReuse(stmt.Body)
	case *ast.TryStmt:
		return !blockAllowsTransientReuse(stmt.TryBlock) || !blockAllowsTransientReuse(stmt.CatchBlock)
	default:
		return false
	}
}
