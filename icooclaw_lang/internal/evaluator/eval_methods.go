package evaluator

import (
	"fmt"

	"github.com/issueye/icooclaw_lang/internal/ast"
	"github.com/issueye/icooclaw_lang/internal/object"
)

func evalMethodCallExpr(node *ast.MethodCallExpr, env *object.Environment) object.Object {
	obj := Eval(node.Object, env)
	if object.IsError(obj) {
		return obj
	}
	if node.Safe {
		if _, ok := obj.(*object.Null); ok {
			return object.NullObject()
		}
	}

	return withCallArgs(node.Arguments, env, func(args []object.Object) object.Object {
		if node.Method.Value == "to_string" {
			if len(args) != 0 {
				return object.NewError(node.Token.Line, "to_string() expects 0 arguments")
			}
			return &object.String{Value: obj.Inspect()}
		}

		switch o := obj.(type) {
		case *object.String:
			return evalStringMethod(o, node.Method.Value, args, node.Token.Line)
		case *object.Array:
			return evalArrayMethod(o, node.Method.Value, args, node.Token.Line)
		case *object.Hash:
			return evalHashMethod(o, node.Method.Value, args, node.Token.Line, env)
		default:
			return object.NewError(node.Token.Line, "no method '%s' on type %s", node.Method.Value, o.Type())
		}
	})
}

func evalHashMethod(hash *object.Hash, method string, args []object.Object, line int, env *object.Environment) object.Object {
	key := &object.String{Value: method}
	pair, ok := hash.Pairs[object.HashKey(key)]
	if !ok {
		return object.NewError(line, "unknown method '%s' on HASH", method)
	}

	switch callee := pair.Value.(type) {
	case *object.Function:
		locals := map[string]object.Object{
			"this": hash,
			"self": hash,
		}
		if callee.ReceiverName != "" {
			locals[callee.ReceiverName] = hash
		}
		return callFunctionWithLocals(callee, args, locals, line)
	case *object.Builtin:
		return callRuntimeObject(env, callee, args, line)
	default:
		return object.NewError(line, "property '%s' is not callable", method)
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
		return object.BoolObject(stringsContains(s.Value, sub.Value))
	case "starts_with":
		if len(args) != 1 {
			return object.NewError(line, "starts_with() expects 1 argument")
		}
		prefix, ok := args[0].(*object.String)
		if !ok {
			return object.NewError(line, "starts_with() expects STRING argument")
		}
		return object.BoolObject(len(s.Value) >= len(prefix.Value) && s.Value[:len(prefix.Value)] == prefix.Value)
	case "ends_with":
		if len(args) != 1 {
			return object.NewError(line, "ends_with() expects 1 argument")
		}
		suffix, ok := args[0].(*object.String)
		if !ok {
			return object.NewError(line, "ends_with() expects STRING argument")
		}
		return object.BoolObject(len(s.Value) >= len(suffix.Value) && s.Value[len(s.Value)-len(suffix.Value):] == suffix.Value)
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
			return object.NullObject()
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
				return object.BoolObject(true)
			}
		}
		return object.BoolObject(false)
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
