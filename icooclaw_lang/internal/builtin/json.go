package builtin

import (
	"encoding/json"

	"github.com/issueye/icooclaw_lang/internal/object"
)

func newJSONLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"stringify":        builtinFunc(jsonStringify),
		"stringify_pretty": builtinFunc(jsonStringifyPretty),
		"parse":            builtinFunc(jsonParse),
	})
}

func jsonStringify(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 1 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
	}
	data, err := json.Marshal(nativeValueForFormat(args[0], "json"))
	if err != nil {
		return object.NewError(0, "could not stringify json: %s", err.Error())
	}
	return &object.String{Value: string(data)}
}

func jsonStringifyPretty(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 1 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
	}
	data, err := json.MarshalIndent(nativeValueForFormat(args[0], "json"), "", "  ")
	if err != nil {
		return object.NewError(0, "could not stringify pretty json: %s", err.Error())
	}
	return &object.String{Value: string(data)}
}

func jsonParse(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 1 && len(args) != 2 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=1 or 2", len(args))
	}
	input, errObj := stringArg(args[0], "argument to `parse` must be STRING, got %s")
	if errObj != nil {
		return errObj
	}
	var decoded interface{}
	if err := json.Unmarshal([]byte(input), &decoded); err != nil {
		return object.NewError(0, "could not parse json: %s", err.Error())
	}
	var schema object.Object
	if len(args) == 2 {
		schema = args[1]
	}
	return objectFromNativeWithSchema(decoded, schema, "json")
}
