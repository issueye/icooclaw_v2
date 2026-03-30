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
	data, err := json.Marshal(nativeValue(args[0]))
	if err != nil {
		return object.NewError(0, "could not stringify json: %s", err.Error())
	}
	return &object.String{Value: string(data)}
}

func jsonStringifyPretty(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 1 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
	}
	data, err := json.MarshalIndent(nativeValue(args[0]), "", "  ")
	if err != nil {
		return object.NewError(0, "could not stringify pretty json: %s", err.Error())
	}
	return &object.String{Value: string(data)}
}

func jsonParse(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 1 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
	}
	input, errObj := stringArg(args[0], "argument to `parse` must be STRING, got %s")
	if errObj != nil {
		return errObj
	}
	var decoded interface{}
	if err := json.Unmarshal([]byte(input), &decoded); err != nil {
		return object.NewError(0, "could not parse json: %s", err.Error())
	}
	return objectFromNative(decoded)
}
