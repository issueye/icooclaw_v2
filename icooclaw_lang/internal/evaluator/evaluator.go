package evaluator

import (
	"github.com/issueye/icooclaw_lang/internal/ast"
	"github.com/issueye/icooclaw_lang/internal/object"
)

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {
	case *ast.Program:
		return evalProgram(node, env)
	case *ast.ExpressionStmt:
		return Eval(node.Expr, env)
	case *ast.LetStmt:
		val := Eval(node.Value, env)
		if object.IsError(val) {
			return val
		}
		return env.Set(node.Name.Value, val)
	case *ast.ConstStmt:
		val := Eval(node.Value, env)
		if object.IsError(val) {
			return val
		}
		return env.SetConst(node.Name.Value, val)
	case *ast.ReturnStmt:
		if node.ReturnValue != nil {
			val := Eval(node.ReturnValue, env)
			if object.IsError(val) {
				return val
			}
			return &object.Return{Value: val}
		}
		return &object.Return{Value: object.NullObject()}
	case *ast.BreakStmt:
		return &object.Break{}
	case *ast.ContinueStmt:
		return &object.Continue{}
	case *ast.IfStmt:
		return evalIfStmt(node, env)
	case *ast.ForStmt:
		return evalForStmt(node, env)
	case *ast.WhileStmt:
		return evalWhileStmt(node, env)
	case *ast.FunctionStmt:
		return evalFunctionStmt(node, env)
	case *ast.MatchStmt:
		return evalMatchStmt(node, env)
	case *ast.TryStmt:
		return evalTryStmt(node, env)
	case *ast.ImportStmt:
		return evalImportStmt(node, env)
	case *ast.ExportStmt:
		return evalExportStmt(node, env)
	case *ast.GoStmt:
		return evalGoStmt(node, env)
	case *ast.BlockStmt:
		return evalBlockStmt(node, env)
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}
	case *ast.FloatLiteral:
		return &object.Float{Value: node.Value}
	case *ast.StringLiteral:
		return &object.String{Value: node.Value}
	case *ast.BooleanLiteral:
		return object.BoolObject(node.Value)
	case *ast.NullLiteral:
		return object.NullObject()
	case *ast.Identifier:
		return evalIdentifier(node, env)
	case *ast.PrefixExpr:
		return evalPrefixExpr(node, env)
	case *ast.InfixExpr:
		return evalInfixExpr(node, env)
	case *ast.AssignExpr:
		return evalAssignExpr(node, env)
	case *ast.CompoundAssignExpr:
		return evalCompoundAssignExpr(node, env)
	case *ast.CallExpr:
		return evalCallExpr(node, env)
	case *ast.ArrayLiteral:
		return evalArrayLiteral(node, env)
	case *ast.HashLiteral:
		return evalHashLiteral(node, env)
	case *ast.IndexExpr:
		return evalIndexExpr(node, env)
	case *ast.DotExpr:
		return evalDotExpr(node, env)
	case *ast.MethodCallExpr:
		return evalMethodCallExpr(node, env)
	case *ast.UnderscoreExpr:
		return object.NullObject()
	default:
		return object.NewError(0, "unknown node type: %T", node)
	}
}
