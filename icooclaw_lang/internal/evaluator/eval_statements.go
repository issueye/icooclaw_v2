package evaluator

import (
	"fmt"
	"os"

	"github.com/issueye/icooclaw_lang/internal/ast"
	"github.com/issueye/icooclaw_lang/internal/builtin"
	"github.com/issueye/icooclaw_lang/internal/object"
)

func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object
	for _, stmt := range program.Statements {
		result = Eval(stmt, env)
		if object.IsError(result) {
			return result
		}
		if ret, ok := result.(*object.Return); ok {
			return ret.Value
		}
	}
	return result
}

func evalBlockStmt(block *ast.BlockStmt, env *object.Environment) object.Object {
	var result object.Object
	for _, stmt := range block.Statements {
		result = Eval(stmt, env)
		if result == nil {
			continue
		}
		if result.Type() == object.RETURN_OBJ || result.Type() == object.ERROR_OBJ ||
			result.Type() == object.BREAK_OBJ || result.Type() == object.CONTINUE_OBJ {
			return result
		}
	}
	return result
}

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}
	if builtin, ok := builtin.Builtins[node.Value]; ok {
		return builtin
	}
	return object.NewError(node.Token.Line, "identifier not found: %s", node.Value)
}

func evalIfStmt(node *ast.IfStmt, env *object.Environment) object.Object {
	condition := Eval(node.Condition, env)
	if object.IsError(condition) {
		return condition
	}

	if object.IsTruthy(condition) {
		return evalBlockStmt(node.Consequence, env)
	} else if node.Alternative != nil {
		return evalBlockStmt(node.Alternative, env)
	}

	return &object.Null{}
}

func evalForStmt(node *ast.ForStmt, env *object.Environment) object.Object {
	iterable := Eval(node.Iterable, env)
	if object.IsError(iterable) {
		return iterable
	}

	var result object.Object = &object.Null{}

	switch iterable := iterable.(type) {
	case *object.Array:
		for _, elem := range iterable.Elements {
			loopEnv := object.NewEnclosedEnvironment(env)
			if assigned := loopEnv.Set(node.Ident.Value, elem); object.IsError(assigned) {
				return assigned
			}
			result = evalBlockStmt(node.Body, loopEnv)
			if object.IsError(result) {
				return result
			}
			if object.IsBreak(result) {
				break
			}
			if object.IsContinue(result) {
				continue
			}
			if object.IsReturn(result) {
				return result
			}
		}
	case *object.Hash:
		for _, pair := range iterable.Pairs {
			loopEnv := object.NewEnclosedEnvironment(env)
			if assigned := loopEnv.Set(node.Ident.Value, pair.Key); object.IsError(assigned) {
				return assigned
			}
			result = evalBlockStmt(node.Body, loopEnv)
			if object.IsError(result) {
				return result
			}
			if object.IsBreak(result) {
				break
			}
			if object.IsContinue(result) {
				continue
			}
			if object.IsReturn(result) {
				return result
			}
		}
	default:
		return object.NewError(node.Token.Line, "object is not iterable: %s", iterable.Type())
	}

	return result
}

func evalWhileStmt(node *ast.WhileStmt, env *object.Environment) object.Object {
	var result object.Object = &object.Null{}

	for {
		condition := Eval(node.Condition, env)
		if object.IsError(condition) {
			return condition
		}
		if !object.IsTruthy(condition) {
			break
		}

		result = evalBlockStmt(node.Body, env)
		if object.IsError(result) {
			return result
		}
		if object.IsBreak(result) {
			break
		}
		if object.IsContinue(result) {
			continue
		}
		if object.IsReturn(result) {
			return result
		}
	}

	return result
}

func evalFunctionStmt(node *ast.FunctionStmt, env *object.Environment) object.Object {
	fn := &object.Function{
		Name:   node.Name.Value,
		Params: node.Params,
		Body:   node.Body,
		Env:    env,
	}
	return env.Set(node.Name.Value, fn)
}

func evalMatchStmt(node *ast.MatchStmt, env *object.Environment) object.Object {
	subject := Eval(node.Subject, env)
	if object.IsError(subject) {
		return subject
	}

	for _, c := range node.Cases {
		for _, pattern := range c.Patterns {
			if _, ok := pattern.(*ast.UnderscoreExpr); ok {
				return Eval(c.Result, env)
			}

			patternVal := Eval(pattern, env)
			if object.IsError(patternVal) {
				return patternVal
			}
			if matchValues(subject, patternVal) {
				return Eval(c.Result, env)
			}
		}
	}

	return &object.Null{}
}

func matchValues(subject, pattern object.Object) bool {
	if _, ok := pattern.(*object.Null); ok {
		if _, ok := subject.(*object.Null); ok {
			return true
		}
		return false
	}

	switch s := subject.(type) {
	case *object.Integer:
		if p, ok := pattern.(*object.Integer); ok {
			return s.Value == p.Value
		}
	case *object.Float:
		if p, ok := pattern.(*object.Float); ok {
			return s.Value == p.Value
		}
	case *object.String:
		if p, ok := pattern.(*object.String); ok {
			return s.Value == p.Value
		}
	case *object.Boolean:
		if p, ok := pattern.(*object.Boolean); ok {
			return s.Value == p.Value
		}
	}
	return false
}

func evalTryStmt(node *ast.TryStmt, env *object.Environment) object.Object {
	result := evalBlockStmt(node.TryBlock, env)

	if object.IsError(result) {
		catchEnv := object.NewEnclosedEnvironment(env)
		errObj, ok := result.(*object.Error)
		if ok {
			catchEnv.Set(node.CatchVar.Value, &object.String{Value: errObj.Message})
		} else {
			catchEnv.Set(node.CatchVar.Value, &object.String{Value: result.Inspect()})
		}
		return evalBlockStmt(node.CatchBlock, catchEnv)
	}

	return result
}

func evalGoStmt(node *ast.GoStmt, env *object.Environment) object.Object {
	switch node.Call.(type) {
	case *ast.CallExpr, *ast.MethodCallExpr:
	default:
		return object.NewError(node.Token.Line, "`go` expects a function or method call")
	}

	env.Go(func() {
		result := Eval(node.Call, env)
		if err, ok := result.(*object.Error); ok {
			fmt.Fprintln(os.Stderr, err.Inspect())
		}
	})

	return &object.Null{}
}
