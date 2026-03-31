package builtin

import (
	"fmt"

	"github.com/issueye/icooclaw_lang/internal/object"
)

func builtinFunc(fn object.BuiltinFunction) *object.Builtin {
	return &object.Builtin{Fn: fn}
}

func hashObject(values map[string]object.Object) *object.Hash {
	pairs := make(map[string]object.HashPair, len(values))
	for key, value := range values {
		keyObj := &object.String{Value: key}
		pairs[key] = object.HashPair{Key: keyObj, Value: value}
	}
	return &object.Hash{Pairs: pairs}
}

func integerArg(arg object.Object, message string) (int64, *object.Error) {
	intObj, ok := arg.(*object.Integer)
	if !ok {
		return 0, object.NewError(0, message, arg.Type())
	}
	return intObj.Value, nil
}

func stringArg(arg object.Object, message string) (string, *object.Error) {
	strObj, ok := arg.(*object.String)
	if !ok {
		return "", object.NewError(0, message, arg.Type())
	}
	return strObj.Value, nil
}

func hashArg(arg object.Object, message string) (*object.Hash, *object.Error) {
	hashObj, ok := arg.(*object.Hash)
	if !ok {
		return nil, object.NewError(0, message, arg.Type())
	}
	return hashObj, nil
}

func boolObject(v bool) *object.Boolean {
	return object.BoolObject(v)
}

func arrayOfStrings(items []string) *object.Array {
	values := make([]object.Object, 0, len(items))
	for _, item := range items {
		values = append(values, &object.String{Value: item})
	}
	return &object.Array{Elements: values}
}

func typeError(name string, got object.Object) *object.Error {
	return object.NewError(0, "%s, got %s", name, got.Type())
}

func nativeValue(obj object.Object) interface{} {
	switch value := obj.(type) {
	case *object.String:
		return value.Value
	case *object.Integer:
		return value.Value
	case *object.Float:
		return value.Value
	case *object.Boolean:
		return value.Value
	case *object.Null:
		return nil
	case *object.Array:
		result := make([]interface{}, 0, len(value.Elements))
		for _, item := range value.Elements {
			result = append(result, nativeValue(item))
		}
		return result
	case *object.Hash:
		result := make(map[string]interface{}, len(value.Pairs))
		for _, pair := range value.Pairs {
			result[pair.Key.Inspect()] = nativeValue(pair.Value)
		}
		return result
	default:
		return value.Inspect()
	}
}

func objectFromNative(value interface{}) object.Object {
	switch v := value.(type) {
	case nil:
		return object.NullObject()
	case string:
		return &object.String{Value: v}
	case bool:
		return boolObject(v)
	case int:
		return &object.Integer{Value: int64(v)}
	case int64:
		return &object.Integer{Value: v}
	case float64:
		if float64(int64(v)) == v {
			return &object.Integer{Value: int64(v)}
		}
		return &object.Float{Value: v}
	case []interface{}:
		items := make([]object.Object, 0, len(v))
		for _, item := range v {
			items = append(items, objectFromNative(item))
		}
		return &object.Array{Elements: items}
	case map[string]interface{}:
		values := make(map[string]object.Object, len(v))
		for key, item := range v {
			values[key] = objectFromNative(item)
		}
		return hashObject(values)
	default:
		return &object.String{Value: fmt.Sprintf("%v", v)}
	}
}
