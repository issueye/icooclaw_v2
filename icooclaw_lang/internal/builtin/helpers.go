package builtin

import (
	"fmt"
	"strings"

	"github.com/issueye/icooclaw_lang/internal/object"
)

const serdeMetaKey = "__serde__"

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

func arrayArg(arg object.Object, message string) (*object.Array, *object.Error) {
	arrayObj, ok := arg.(*object.Array)
	if !ok {
		return nil, object.NewError(0, message, arg.Type())
	}
	return arrayObj, nil
}

func callableArg(arg object.Object, message string) (object.Object, *object.Error) {
	switch arg.(type) {
	case *object.Function, *object.Builtin:
		return arg, nil
	default:
		return nil, object.NewError(0, message, arg.Type())
	}
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
	return nativeValueForFormat(obj, "")
}

func nativeValueForFormat(obj object.Object, format string) interface{} {
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
			result = append(result, nativeValueForFormat(item, format))
		}
		return result
	case *object.Hash:
		result := make(map[string]interface{}, len(value.Pairs))
		meta := serdeMetaForFormat(value, format)
		for _, pair := range value.Pairs {
			key := pair.Key.Inspect()
			if key == serdeMetaKey {
				continue
			}
			name, skip, omitEmpty := serdeFieldRule(meta, key, format)
			if skip {
				continue
			}
			if name == "" {
				name = key
			}
			if omitEmpty && isSerdeEmpty(pair.Value) {
				continue
			}
			result[name] = nativeValueForFormat(pair.Value, format)
		}
		return result
	default:
		return value.Inspect()
	}
}

func objectFromNative(value interface{}) object.Object {
	return objectFromNativeWithSchema(value, nil, "")
}

func objectFromNativeWithSchema(value interface{}, schema object.Object, format string) object.Object {
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
		var itemSchema object.Object
		if schemaArr, ok := schema.(*object.Array); ok && len(schemaArr.Elements) > 0 {
			itemSchema = schemaArr.Elements[0]
		}
		for _, item := range v {
			items = append(items, objectFromNativeWithSchema(item, itemSchema, format))
		}
		return &object.Array{Elements: items}
	case map[string]interface{}:
		values := make(map[string]object.Object, len(v))
		schemaHash, _ := schema.(*object.Hash)
		meta := serdeMetaForFormat(schemaHash, format)
		for key, item := range v {
			targetKey, nestedSchema, skip := serdeTargetForInputKey(schemaHash, meta, key)
			if skip {
				continue
			}
			values[targetKey] = objectFromNativeWithSchema(item, nestedSchema, format)
		}
		if schemaHash != nil {
			if metaPair, ok := schemaHash.Pairs[serdeMetaKey]; ok {
				values[serdeMetaKey] = metaPair.Value
			}
		}
		return hashObject(values)
	default:
		return &object.String{Value: fmt.Sprintf("%v", v)}
	}
}

func serdeMetaForFormat(hash *object.Hash, format string) *object.Hash {
	if hash == nil {
		return nil
	}
	pair, ok := hash.Pairs[serdeMetaKey]
	if !ok {
		return nil
	}
	meta, ok := pair.Value.(*object.Hash)
	if !ok {
		return nil
	}
	if format == "" {
		return meta
	}
	return meta
}

func serdeFieldRule(meta *object.Hash, field, fallbackFormat string) (string, bool, bool) {
	if meta == nil {
		return field, false, false
	}
	fieldPair, ok := meta.Pairs[field]
	if !ok {
		return field, false, false
	}
	switch spec := fieldPair.Value.(type) {
	case *object.String:
		return parseSerdeTag(spec.Value, field)
	case *object.Hash:
		if tag := serdeTagFromMetaHash(spec, fallbackFormat); tag != "" {
			return parseSerdeTag(tag, field)
		}
		if tag := serdeTagFromMetaHash(spec, "json"); tag != "" && fallbackFormat != "json" {
			return parseSerdeTag(tag, field)
		}
	}
	return field, false, false
}

func serdeTagFromMetaHash(meta *object.Hash, format string) string {
	if meta == nil {
		return ""
	}
	if format != "" {
		if pair, ok := meta.Pairs[format]; ok {
			if str, ok := pair.Value.(*object.String); ok {
				return str.Value
			}
		}
	}
	if pair, ok := meta.Pairs["name"]; ok {
		if str, ok := pair.Value.(*object.String); ok {
			return str.Value
		}
	}
	return ""
}

func parseSerdeTag(tag, field string) (string, bool, bool) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return field, false, false
	}
	if tag == "-" {
		return "", true, false
	}
	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name == "" {
		name = field
	}
	omitEmpty := false
	for _, opt := range parts[1:] {
		if strings.TrimSpace(opt) == "omitempty" {
			omitEmpty = true
		}
	}
	return name, false, omitEmpty
}

func isSerdeEmpty(obj object.Object) bool {
	switch value := obj.(type) {
	case *object.Null:
		return true
	case *object.Boolean:
		return !value.Value
	case *object.Integer:
		return value.Value == 0
	case *object.Float:
		return value.Value == 0
	case *object.String:
		return value.Value == ""
	case *object.Array:
		return len(value.Elements) == 0
	case *object.Hash:
		count := 0
		for key := range value.Pairs {
			if key == serdeMetaKey {
				continue
			}
			count++
		}
		return count == 0
	default:
		return false
	}
}

func serdeTargetForInputKey(schema *object.Hash, meta *object.Hash, inputKey string) (string, object.Object, bool) {
	if schema == nil {
		return inputKey, nil, false
	}
	if pair, ok := schema.Pairs[inputKey]; ok {
		return inputKey, pair.Value, false
	}
	for field := range schema.Pairs {
		if field == serdeMetaKey {
			continue
		}
		name, skip, _ := serdeFieldRule(meta, field, "")
		if skip {
			continue
		}
		if name == inputKey {
			return field, schema.Pairs[field].Value, false
		}
	}
	return inputKey, nil, false
}
