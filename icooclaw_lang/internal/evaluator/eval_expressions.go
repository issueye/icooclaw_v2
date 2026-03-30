package evaluator

import (
	"github.com/issueye/icooclaw_lang/internal/ast"
	"github.com/issueye/icooclaw_lang/internal/object"
)

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
		return env.Set(left.Value, right)
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

	return env.Set(ident.Value, result)
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
			if assigned := callEnv.Set(param.Value, args[i]); object.IsError(assigned) {
				return assigned
			}
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
