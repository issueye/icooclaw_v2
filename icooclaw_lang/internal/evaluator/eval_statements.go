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
	if global, ok := builtin.Builtins[node.Value]; ok {
		return global
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

	return object.NullObject()
}

func evalForStmt(node *ast.ForStmt, env *object.Environment) object.Object {
	if rangeResult, handled := evalRangeForStmt(node, env); handled {
		return rangeResult
	}

	iterable := Eval(node.Iterable, env)
	if object.IsError(iterable) {
		return iterable
	}

	var result object.Object = object.NullObject()
	transientSafe := blockAllowsTransientReuse(node.Body)

	switch iterable := iterable.(type) {
	case *object.Array:
		for _, elem := range iterable.Elements {
			loopEnv := newScopedEnv(env, transientSafe)
			if assigned := loopEnv.DefineLocal(node.Ident.Value, elem); object.IsError(assigned) {
				releaseScopedEnv(loopEnv)
				return assigned
			}
			result = evalBlockStmt(node.Body, loopEnv)
			releaseScopedEnv(loopEnv)
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
			loopEnv := newScopedEnv(env, transientSafe)
			if assigned := loopEnv.DefineLocal(node.Ident.Value, pair.Key); object.IsError(assigned) {
				releaseScopedEnv(loopEnv)
				return assigned
			}
			result = evalBlockStmt(node.Body, loopEnv)
			releaseScopedEnv(loopEnv)
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

func evalRangeForStmt(node *ast.ForStmt, env *object.Environment) (object.Object, bool) {
	call, ok := node.Iterable.(*ast.CallExpr)
	if !ok {
		return nil, false
	}

	ident, ok := call.Function.(*ast.Identifier)
	if !ok || ident.Value != "range" {
		return nil, false
	}

	start, stop, errObj := evalRangeBounds(call.Arguments, env)
	if errObj != nil {
		return errObj, true
	}

	var result object.Object = object.NullObject()
	transientSafe := blockAllowsTransientReuse(node.Body)
	for i := start; i < stop; i++ {
		loopEnv := newScopedEnv(env, transientSafe)
		if assigned := loopEnv.DefineLocal(node.Ident.Value, &object.Integer{Value: i}); object.IsError(assigned) {
			releaseScopedEnv(loopEnv)
			return assigned, true
		}
		result = evalBlockStmt(node.Body, loopEnv)
		releaseScopedEnv(loopEnv)
		if object.IsError(result) {
			return result, true
		}
		if object.IsBreak(result) {
			break
		}
		if object.IsContinue(result) {
			continue
		}
		if object.IsReturn(result) {
			return result, true
		}
	}

	return result, true
}

func evalRangeBounds(args []ast.Expr, env *object.Environment) (int64, int64, *object.Error) {
	switch len(args) {
	case 1:
		stopObj := Eval(args[0], env)
		if errObj, ok := stopObj.(*object.Error); ok {
			return 0, 0, errObj
		}
		stop, ok := stopObj.(*object.Integer)
		if !ok {
			return 0, 0, object.NewError(0, "argument to `range` must be INTEGER, got %s", stopObj.Type())
		}
		return 0, stop.Value, nil
	case 2:
		startObj := Eval(args[0], env)
		if errObj, ok := startObj.(*object.Error); ok {
			return 0, 0, errObj
		}
		stopObj := Eval(args[1], env)
		if errObj, ok := stopObj.(*object.Error); ok {
			return 0, 0, errObj
		}

		start, ok := startObj.(*object.Integer)
		if !ok {
			return 0, 0, object.NewError(0, "argument to `range` must be INTEGER, got %s", startObj.Type())
		}
		stop, ok := stopObj.(*object.Integer)
		if !ok {
			return 0, 0, object.NewError(0, "argument to `range` must be INTEGER, got %s", stopObj.Type())
		}
		return start.Value, stop.Value, nil
	default:
		return 0, 0, object.NewError(0, "wrong number of arguments. got=%d, want=1 or 2", len(args))
	}
}

func evalWhileStmt(node *ast.WhileStmt, env *object.Environment) object.Object {
	var result object.Object = object.NullObject()

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
		Name:          node.Name.Value,
		Params:        node.Params,
		Body:          node.Body,
		Env:           env,
		TransientSafe: blockAllowsTransientReuse(node.Body),
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
			bindings := make(map[string]object.Object)
			matched, matchErr := matchPattern(subject, pattern, env, bindings)
			if matchErr != nil {
				return matchErr
			}
			if !matched {
				continue
			}

			caseEnv := object.AcquireTransientEnclosedEnvironment(env)
			if assigned := caseEnv.DefineLocals(bindings); object.IsError(assigned) {
				object.ReleaseTransientEnvironment(caseEnv)
				return assigned
			}

			if c.Guard != nil {
				guard := Eval(c.Guard, caseEnv)
				if object.IsError(guard) {
					object.ReleaseTransientEnvironment(caseEnv)
					return guard
				}
				if !object.IsTruthy(guard) {
					object.ReleaseTransientEnvironment(caseEnv)
					continue
				}
			}

			result := Eval(c.Result, caseEnv)
			object.ReleaseTransientEnvironment(caseEnv)
			return result
		}
	}

	return object.NullObject()
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

func matchPattern(subject object.Object, pattern ast.Expr, env *object.Environment, bindings map[string]object.Object) (bool, *object.Error) {
	switch pattern := pattern.(type) {
	case *ast.UnderscoreExpr:
		return true, nil
	case *ast.Identifier:
		if bound, ok := bindings[pattern.Value]; ok {
			return matchValues(subject, bound), nil
		}
		bindings[pattern.Value] = subject
		return true, nil
	case *ast.ArrayLiteral:
		subjectArray, ok := subject.(*object.Array)
		if !ok || len(subjectArray.Elements) != len(pattern.Elements) {
			return false, nil
		}
		for idx, elementPattern := range pattern.Elements {
			matched, matchErr := matchPattern(subjectArray.Elements[idx], elementPattern, env, bindings)
			if matchErr != nil || !matched {
				return matched, matchErr
			}
		}
		return true, nil
	case *ast.HashLiteral:
		subjectHash, ok := subject.(*object.Hash)
		if !ok {
			return false, nil
		}
		for keyNode, valuePattern := range pattern.Pairs {
			keyObj := Eval(keyNode, env)
			if errObj, ok := keyObj.(*object.Error); ok {
				return false, errObj
			}
			pair, ok := subjectHash.Pairs[object.HashKey(keyObj)]
			if !ok {
				return false, nil
			}
			matched, matchErr := matchPattern(pair.Value, valuePattern, env, bindings)
			if matchErr != nil || !matched {
				return matched, matchErr
			}
		}
		return true, nil
	default:
		patternVal := Eval(pattern, env)
		if errObj, ok := patternVal.(*object.Error); ok {
			return false, errObj
		}
		return matchValues(subject, patternVal), nil
	}
}

func evalTryStmt(node *ast.TryStmt, env *object.Environment) object.Object {
	result := evalBlockStmt(node.TryBlock, env)

	if object.IsError(result) {
		catchEnv := newScopedEnv(env, blockAllowsTransientReuse(node.CatchBlock))
		defer releaseScopedEnv(catchEnv)
		errObj, ok := result.(*object.Error)
		if ok {
			catchEnv.DefineLocal(node.CatchVar.Value, &object.String{Value: errObj.Message})
		} else {
			catchEnv.DefineLocal(node.CatchVar.Value, &object.String{Value: result.Inspect()})
		}
		return evalBlockStmt(node.CatchBlock, catchEnv)
	}

	return result
}

func newScopedEnv(outer *object.Environment, transientSafe bool) *object.Environment {
	if transientSafe {
		return object.AcquireTransientEnclosedEnvironment(outer)
	}
	return object.NewEnclosedEnvironment(outer)
}

func releaseScopedEnv(env *object.Environment) {
	if env == nil {
		return
	}
	if env.IsTransient() {
		object.ReleaseTransientEnvironment(env)
	}
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

	return object.NullObject()
}
