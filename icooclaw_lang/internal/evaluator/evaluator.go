package evaluator

import (
	"fmt"

	"github.com/issueye/icooclaw_lang/internal/ast"
	"github.com/issueye/icooclaw_lang/internal/builtin"
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
		env.Set(node.Name.Value, val)
		return val
	case *ast.ConstStmt:
		val := Eval(node.Value, env)
		if object.IsError(val) {
			return val
		}
		env.SetConst(node.Name.Value, val)
		return val
	case *ast.ReturnStmt:
		if node.ReturnValue != nil {
			val := Eval(node.ReturnValue, env)
			if object.IsError(val) {
				return val
			}
			return &object.Return{Value: val}
		}
		return &object.Return{Value: &object.Null{}}
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
		return &object.Null{}
	case *ast.ExportStmt:
		return &object.Null{}
	case *ast.GoStmt:
		return &object.Null{}
	case *ast.BlockStmt:
		return evalBlockStmt(node, env)
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}
	case *ast.FloatLiteral:
		return &object.Float{Value: node.Value}
	case *ast.StringLiteral:
		return &object.String{Value: node.Value}
	case *ast.BooleanLiteral:
		return &object.Boolean{Value: node.Value}
	case *ast.NullLiteral:
		return &object.Null{}
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
		return &object.Null{}
	default:
		return object.NewError(0, "unknown node type: %T", node)
	}
}

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
			loopEnv.Set(node.Ident.Value, elem)
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
			loopEnv.Set(node.Ident.Value, pair.Key)
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
	env.Set(node.Name.Value, fn)
	return fn
}

func evalMatchStmt(node *ast.MatchStmt, env *object.Environment) object.Object {
	subject := Eval(node.Subject, env)
	if object.IsError(subject) {
		return subject
	}

	for _, c := range node.Cases {
		for _, pattern := range c.Patterns {
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

func evalPrefixExpr(node *ast.PrefixExpr, env *object.Environment) object.Object {
	right := Eval(node.Right, env)
	if object.IsError(right) {
		return right
	}

	switch node.Operator {
	case "!":
		return evalBangOperator(right)
	case "-":
		return evalMinusOperator(right, node.Token.Line)
	default:
		return object.NewError(node.Token.Line, "unknown operator: %s%s", node.Operator, right.Type())
	}
}

func evalBangOperator(right object.Object) object.Object {
	return &object.Boolean{Value: !object.IsTruthy(right)}
}

func evalMinusOperator(right object.Object, line int) object.Object {
	switch right := right.(type) {
	case *object.Integer:
		return &object.Integer{Value: -right.Value}
	case *object.Float:
		return &object.Float{Value: -right.Value}
	default:
		return object.NewError(line, "unknown operator: -%s", right.Type())
	}
}

func evalInfixExpr(node *ast.InfixExpr, env *object.Environment) object.Object {
	left := Eval(node.Left, env)
	if object.IsError(left) {
		return left
	}

	right := Eval(node.Right, env)
	if object.IsError(right) {
		return right
	}

	switch node.Operator {
	case "+":
		return evalPlusOperator(left, right, node.Token.Line)
	case "-":
		return evalArithmeticOperator(left, right, "-", node.Token.Line)
	case "*":
		return evalArithmeticOperator(left, right, "*", node.Token.Line)
	case "/":
		return evalArithmeticOperator(left, right, "/", node.Token.Line)
	case "%":
		return evalModuloOperator(left, right, node.Token.Line)
	case "==":
		return &object.Boolean{Value: evalEquality(left, right)}
	case "!=":
		return &object.Boolean{Value: !evalEquality(left, right)}
	case "<":
		return evalComparison(left, right, "<", node.Token.Line)
	case ">":
		return evalComparison(left, right, ">", node.Token.Line)
	case "<=":
		return evalComparison(left, right, "<=", node.Token.Line)
	case ">=":
		return evalComparison(left, right, ">=", node.Token.Line)
	case "&&":
		return &object.Boolean{Value: object.IsTruthy(left) && object.IsTruthy(right)}
	case "||":
		return &object.Boolean{Value: object.IsTruthy(left) || object.IsTruthy(right)}
	default:
		return object.NewError(node.Token.Line, "unknown operator: %s %s %s", left.Type(), node.Operator, right.Type())
	}
}

func evalPlusOperator(left, right object.Object, line int) object.Object {
	switch left := left.(type) {
	case *object.Integer:
		if right, ok := right.(*object.Integer); ok {
			return &object.Integer{Value: left.Value + right.Value}
		}
		if right, ok := right.(*object.Float); ok {
			return &object.Float{Value: float64(left.Value) + right.Value}
		}
	case *object.Float:
		if right, ok := right.(*object.Integer); ok {
			return &object.Float{Value: left.Value + float64(right.Value)}
		}
		if right, ok := right.(*object.Float); ok {
			return &object.Float{Value: left.Value + right.Value}
		}
	case *object.String:
		if right, ok := right.(*object.String); ok {
			return &object.String{Value: left.Value + right.Value}
		}
	case *object.Array:
		if right, ok := right.(*object.Array); ok {
			return &object.Array{Elements: append(left.Elements, right.Elements...)}
		}
	}
	return object.NewError(line, "unknown operator: %s + %s", left.Type(), right.Type())
}

func evalArithmeticOperator(left, right object.Object, op string, line int) object.Object {
	switch left := left.(type) {
	case *object.Integer:
		if right, ok := right.(*object.Integer); ok {
			switch op {
			case "-":
				return &object.Integer{Value: left.Value - right.Value}
			case "*":
				return &object.Integer{Value: left.Value * right.Value}
			case "/":
				if right.Value == 0 {
					return object.NewError(line, "division by zero")
				}
				return &object.Integer{Value: left.Value / right.Value}
			}
		}
		if right, ok := right.(*object.Float); ok {
			switch op {
			case "-":
				return &object.Float{Value: float64(left.Value) - right.Value}
			case "*":
				return &object.Float{Value: float64(left.Value) * right.Value}
			case "/":
				if right.Value == 0 {
					return object.NewError(line, "division by zero")
				}
				return &object.Float{Value: float64(left.Value) / right.Value}
			}
		}
	case *object.Float:
		if right, ok := right.(*object.Integer); ok {
			switch op {
			case "-":
				return &object.Float{Value: left.Value - float64(right.Value)}
			case "*":
				return &object.Float{Value: left.Value * float64(right.Value)}
			case "/":
				if right.Value == 0 {
					return object.NewError(line, "division by zero")
				}
				return &object.Float{Value: left.Value / float64(right.Value)}
			}
		}
		if right, ok := right.(*object.Float); ok {
			switch op {
			case "-":
				return &object.Float{Value: left.Value - right.Value}
			case "*":
				return &object.Float{Value: left.Value * right.Value}
			case "/":
				if right.Value == 0 {
					return object.NewError(line, "division by zero")
				}
				return &object.Float{Value: left.Value / right.Value}
			}
		}
	}
	return object.NewError(line, "unknown operator: %s %s %s", left.Type(), op, right.Type())
}

func evalModuloOperator(left, right object.Object, line int) object.Object {
	if left, ok := left.(*object.Integer); ok {
		if right, ok := right.(*object.Integer); ok {
			if right.Value == 0 {
				return object.NewError(line, "modulo by zero")
			}
			return &object.Integer{Value: left.Value % right.Value}
		}
	}
	return object.NewError(line, "unknown operator: %s %% %s", left.Type(), right.Type())
}

func evalEquality(left, right object.Object) bool {
	switch left := left.(type) {
	case *object.Integer:
		if right, ok := right.(*object.Integer); ok {
			return left.Value == right.Value
		}
		if right, ok := right.(*object.Float); ok {
			return float64(left.Value) == right.Value
		}
	case *object.Float:
		if right, ok := right.(*object.Integer); ok {
			return left.Value == float64(right.Value)
		}
		if right, ok := right.(*object.Float); ok {
			return left.Value == right.Value
		}
	case *object.String:
		if right, ok := right.(*object.String); ok {
			return left.Value == right.Value
		}
	case *object.Boolean:
		if right, ok := right.(*object.Boolean); ok {
			return left.Value == right.Value
		}
	case *object.Null:
		_, ok := right.(*object.Null)
		return ok
	}
	return false
}

func evalComparison(left, right object.Object, op string, line int) object.Object {
	switch left := left.(type) {
	case *object.Integer:
		if right, ok := right.(*object.Integer); ok {
			switch op {
			case "<":
				return &object.Boolean{Value: left.Value < right.Value}
			case ">":
				return &object.Boolean{Value: left.Value > right.Value}
			case "<=":
				return &object.Boolean{Value: left.Value <= right.Value}
			case ">=":
				return &object.Boolean{Value: left.Value >= right.Value}
			}
		}
		if right, ok := right.(*object.Float); ok {
			switch op {
			case "<":
				return &object.Boolean{Value: float64(left.Value) < right.Value}
			case ">":
				return &object.Boolean{Value: float64(left.Value) > right.Value}
			case "<=":
				return &object.Boolean{Value: float64(left.Value) <= right.Value}
			case ">=":
				return &object.Boolean{Value: float64(left.Value) >= right.Value}
			}
		}
	case *object.Float:
		if right, ok := right.(*object.Integer); ok {
			switch op {
			case "<":
				return &object.Boolean{Value: left.Value < float64(right.Value)}
			case ">":
				return &object.Boolean{Value: left.Value > float64(right.Value)}
			case "<=":
				return &object.Boolean{Value: left.Value <= float64(right.Value)}
			case ">=":
				return &object.Boolean{Value: left.Value >= float64(right.Value)}
			}
		}
		if right, ok := right.(*object.Float); ok {
			switch op {
			case "<":
				return &object.Boolean{Value: left.Value < right.Value}
			case ">":
				return &object.Boolean{Value: left.Value > right.Value}
			case "<=":
				return &object.Boolean{Value: left.Value <= right.Value}
			case ">=":
				return &object.Boolean{Value: left.Value >= right.Value}
			}
		}
	case *object.String:
		if right, ok := right.(*object.String); ok {
			switch op {
			case "<":
				return &object.Boolean{Value: left.Value < right.Value}
			case ">":
				return &object.Boolean{Value: left.Value > right.Value}
			case "<=":
				return &object.Boolean{Value: left.Value <= right.Value}
			case ">=":
				return &object.Boolean{Value: left.Value >= right.Value}
			}
		}
	}
	return object.NewError(line, "unknown operator: %s %s %s", left.Type(), op, right.Type())
}

func evalAssignExpr(node *ast.AssignExpr, env *object.Environment) object.Object {
	right := Eval(node.Right, env)
	if object.IsError(right) {
		return right
	}

	switch left := node.Left.(type) {
	case *ast.Identifier:
		env.Set(left.Value, right)
		return right
	case *ast.IndexExpr:
		obj := Eval(left.Left, env)
		if object.IsError(obj) {
			return obj
		}
		index := Eval(left.Index, env)
		if object.IsError(index) {
			return index
		}
		switch obj := obj.(type) {
		case *object.Array:
			idx, ok := index.(*object.Integer)
			if !ok {
				return object.NewError(node.Token.Line, "array index must be INTEGER")
			}
			if idx.Value < 0 || int(idx.Value) >= len(obj.Elements) {
				return object.NewError(node.Token.Line, "index out of bounds: %d", idx.Value)
			}
			obj.Elements[idx.Value] = right
			return right
		case *object.Hash:
			key := object.HashKey(index)
			obj.Pairs[key] = object.HashPair{Key: index, Value: right}
			return right
		default:
			return object.NewError(node.Token.Line, "index assignment not supported on %s", obj.Type())
		}
	default:
		return object.NewError(node.Token.Line, "invalid assignment target")
	}
}

func evalCompoundAssignExpr(node *ast.CompoundAssignExpr, env *object.Environment) object.Object {
	ident, ok := node.Left.(*ast.Identifier)
	if !ok {
		return object.NewError(node.Token.Line, "invalid compound assignment target")
	}

	left, ok := env.Get(ident.Value)
	if !ok {
		return object.NewError(node.Token.Line, "identifier not found: %s", ident.Value)
	}

	right := Eval(node.Right, env)
	if object.IsError(right) {
		return right
	}

	var op string
	switch node.Operator {
	case "+=":
		op = "+"
	case "-=":
		op = "-"
	case "*=":
		op = "*"
	case "/=":
		op = "/"
	default:
		return object.NewError(node.Token.Line, "unknown compound operator: %s", node.Operator)
	}

	result := evalArithmeticForCompound(left, right, op, node.Token.Line)
	if object.IsError(result) {
		return result
	}

	env.Set(ident.Value, result)
	return result
}

func evalArithmeticForCompound(left, right object.Object, op string, line int) object.Object {
	switch left := left.(type) {
	case *object.Integer:
		if right, ok := right.(*object.Integer); ok {
			switch op {
			case "+":
				return &object.Integer{Value: left.Value + right.Value}
			case "-":
				return &object.Integer{Value: left.Value - right.Value}
			case "*":
				return &object.Integer{Value: left.Value * right.Value}
			case "/":
				if right.Value == 0 {
					return object.NewError(line, "division by zero")
				}
				return &object.Integer{Value: left.Value / right.Value}
			}
		}
	case *object.Float:
		if right, ok := right.(*object.Float); ok {
			switch op {
			case "+":
				return &object.Float{Value: left.Value + right.Value}
			case "-":
				return &object.Float{Value: left.Value - right.Value}
			case "*":
				return &object.Float{Value: left.Value * right.Value}
			case "/":
				if right.Value == 0 {
					return object.NewError(line, "division by zero")
				}
				return &object.Float{Value: left.Value / right.Value}
			}
		}
	}
	return object.NewError(line, "unknown operator: %s %s %s", left.Type(), op, right.Type())
}

func evalCallExpr(node *ast.CallExpr, env *object.Environment) object.Object {
	fn := Eval(node.Function, env)
	if object.IsError(fn) {
		return fn
	}

	args := evalArgs(node.Arguments, env)
	if len(args) == 1 && object.IsError(args[0]) {
		return args[0]
	}

	switch fn := fn.(type) {
	case *object.Function:
		callEnv := object.NewEnclosedEnvironment(fn.Env)
		if len(args) != len(fn.Params) {
			return object.NewError(node.Token.Line, "wrong number of arguments: want=%d, got=%d",
				len(fn.Params), len(args))
		}
		for i, param := range fn.Params {
			callEnv.Set(param.Value, args[i])
		}
		result := evalBlockStmt(fn.Body, callEnv)
		if ret, ok := result.(*object.Return); ok {
			return ret.Value
		}
		return result
	case *object.Builtin:
		return fn.Fn(args...)
	default:
		return object.NewError(node.Token.Line, "not a function: %s", fn.Type())
	}
}

func evalArgs(args []ast.Expr, env *object.Environment) []object.Object {
	var result []object.Object
	for _, arg := range args {
		evaluated := Eval(arg, env)
		if object.IsError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}
	return result
}

func evalArrayLiteral(node *ast.ArrayLiteral, env *object.Environment) object.Object {
	elements := evalArgs(node.Elements, env)
	if len(elements) == 1 && object.IsError(elements[0]) {
		return elements[0]
	}
	return &object.Array{Elements: elements}
}

func evalHashLiteral(node *ast.HashLiteral, env *object.Environment) object.Object {
	pairs := make(map[string]object.HashPair)
	for keyNode, valueNode := range node.Pairs {
		key := Eval(keyNode, env)
		if object.IsError(key) {
			return key
		}
		value := Eval(valueNode, env)
		if object.IsError(value) {
			return value
		}
		pairs[object.HashKey(key)] = object.HashPair{Key: key, Value: value}
	}
	return &object.Hash{Pairs: pairs}
}

func evalIndexExpr(node *ast.IndexExpr, env *object.Environment) object.Object {
	left := Eval(node.Left, env)
	if object.IsError(left) {
		return left
	}
	index := Eval(node.Index, env)
	if object.IsError(index) {
		return index
	}

	switch left := left.(type) {
	case *object.Array:
		idx, ok := index.(*object.Integer)
		if !ok {
			return object.NewError(node.Token.Line, "array index must be INTEGER")
		}
		if idx.Value < 0 || int(idx.Value) >= len(left.Elements) {
			return object.NewError(node.Token.Line, "index out of bounds: %d", idx.Value)
		}
		return left.Elements[idx.Value]
	case *object.Hash:
		key := object.HashKey(index)
		pair, ok := left.Pairs[key]
		if !ok {
			return &object.Null{}
		}
		return pair.Value
	case *object.String:
		idx, ok := index.(*object.Integer)
		if !ok {
			return object.NewError(node.Token.Line, "string index must be INTEGER")
		}
		runes := []rune(left.Value)
		if idx.Value < 0 || int(idx.Value) >= len(runes) {
			return object.NewError(node.Token.Line, "index out of bounds: %d", idx.Value)
		}
		return &object.String{Value: string(runes[idx.Value])}
	default:
		return object.NewError(node.Token.Line, "index operator not supported: %s", left.Type())
	}
}

func evalDotExpr(node *ast.DotExpr, env *object.Environment) object.Object {
	obj := Eval(node.Left, env)
	if object.IsError(obj) {
		return obj
	}

	switch o := obj.(type) {
	case *object.Hash:
		key := &object.String{Value: node.Right.Value}
		pair, ok := o.Pairs[object.HashKey(key)]
		if !ok {
			return &object.Null{}
		}
		return pair.Value
	default:
		return object.NewError(node.Token.Line, "dot access not supported on %s", obj.Type())
	}
}

func evalMethodCallExpr(node *ast.MethodCallExpr, env *object.Environment) object.Object {
	obj := Eval(node.Object, env)
	if object.IsError(obj) {
		return obj
	}

	args := evalArgs(node.Arguments, env)
	if len(args) == 1 && object.IsError(args[0]) {
		return args[0]
	}

	switch o := obj.(type) {
	case *object.String:
		return evalStringMethod(o, node.Method.Value, args, node.Token.Line)
	case *object.Array:
		return evalArrayMethod(o, node.Method.Value, args, node.Token.Line)
	default:
		return object.NewError(node.Token.Line, "no method '%s' on type %s", node.Method.Value, o.Type())
	}
}

func evalStringMethod(s *object.String, method string, args []object.Object, line int) object.Object {
	switch method {
	case "len":
		return &object.Integer{Value: int64(len(s.Value))}
	case "upper":
		return &object.String{Value: fmt.Sprintf("%s", stringsToUpper(s.Value))}
	case "lower":
		return &object.String{Value: stringsToLower(s.Value)}
	case "trim":
		return &object.String{Value: stringsTrimSpace(s.Value)}
	case "split":
		if len(args) != 1 {
			return object.NewError(line, "split() expects 1 argument")
		}
		sep, ok := args[0].(*object.String)
		if !ok {
			return object.NewError(line, "split() expects STRING argument")
		}
		parts := stringsSplit(s.Value, sep.Value)
		elements := make([]object.Object, len(parts))
		for i, p := range parts {
			elements[i] = &object.String{Value: p}
		}
		return &object.Array{Elements: elements}
	case "contains":
		if len(args) != 1 {
			return object.NewError(line, "contains() expects 1 argument")
		}
		sub, ok := args[0].(*object.String)
		if !ok {
			return object.NewError(line, "contains() expects STRING argument")
		}
		return &object.Boolean{Value: stringsContains(s.Value, sub.Value)}
	case "starts_with":
		if len(args) != 1 {
			return object.NewError(line, "starts_with() expects 1 argument")
		}
		prefix, ok := args[0].(*object.String)
		if !ok {
			return object.NewError(line, "starts_with() expects STRING argument")
		}
		return &object.Boolean{Value: len(s.Value) >= len(prefix.Value) && s.Value[:len(prefix.Value)] == prefix.Value}
	case "ends_with":
		if len(args) != 1 {
			return object.NewError(line, "ends_with() expects 1 argument")
		}
		suffix, ok := args[0].(*object.String)
		if !ok {
			return object.NewError(line, "ends_with() expects STRING argument")
		}
		return &object.Boolean{Value: len(s.Value) >= len(suffix.Value) && s.Value[len(s.Value)-len(suffix.Value):] == suffix.Value}
	default:
		return object.NewError(line, "unknown method '%s' on STRING", method)
	}
}

func evalArrayMethod(arr *object.Array, method string, args []object.Object, line int) object.Object {
	switch method {
	case "len":
		return &object.Integer{Value: int64(len(arr.Elements))}
	case "push":
		if len(args) != 1 {
			return object.NewError(line, "push() expects 1 argument")
		}
		return &object.Array{Elements: append(arr.Elements, args[0])}
	case "pop":
		if len(arr.Elements) == 0 {
			return &object.Null{}
		}
		return arr.Elements[len(arr.Elements)-1]
	case "join":
		if len(args) != 1 {
			return object.NewError(line, "join() expects 1 argument")
		}
		sep, ok := args[0].(*object.String)
		if !ok {
			return object.NewError(line, "join() expects STRING argument")
		}
		parts := make([]string, len(arr.Elements))
		for i, e := range arr.Elements {
			parts[i] = e.Inspect()
		}
		return &object.String{Value: stringsJoin(parts, sep.Value)}
	case "contains":
		if len(args) != 1 {
			return object.NewError(line, "contains() expects 1 argument")
		}
		for _, e := range arr.Elements {
			if evalEquality(e, args[0]) {
				return &object.Boolean{Value: true}
			}
		}
		return &object.Boolean{Value: false}
	default:
		return object.NewError(line, "unknown method '%s' on ARRAY", method)
	}
}

func stringsToUpper(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'a' && r <= 'z' {
			result[i] = r - 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

func stringsToLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

func stringsTrimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func stringsSplit(s, sep string) []string {
	if sep == "" {
		result := make([]string, len(s))
		for i, r := range s {
			result[i] = string(r)
		}
		return result
	}
	var result []string
	for {
		idx := indexOf(s, sep)
		if idx == -1 {
			result = append(result, s)
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	return result
}

func stringsContains(s, sub string) bool {
	return indexOf(s, sub) >= 0
}

func stringsJoin(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

func indexOf(s, sub string) int {
	if len(sub) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
