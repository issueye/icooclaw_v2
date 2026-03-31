package builtin

import (
	"os"

	"github.com/issueye/icooclaw_lang/internal/object"
	"gopkg.in/yaml.v3"
)

func newYAMLLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"parse":      builtinFunc(yamlParse),
		"parse_file": builtinFunc(yamlParseFile),
		"stringify":  builtinFunc(yamlStringify),
	})
}

func yamlParse(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 1 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
	}
	input, errObj := stringArg(args[0], "argument to `parse` must be STRING, got %s")
	if errObj != nil {
		return errObj
	}

	value, err := parseYAML(input)
	if err != nil {
		return object.NewError(0, "could not parse yaml: %s", err.Error())
	}
	return objectFromNative(value)
}

func yamlParseFile(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 1 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
	}
	path, errObj := stringArg(args[0], "argument to `parse_file` must be STRING, got %s")
	if errObj != nil {
		return errObj
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return object.NewError(0, "could not read yaml file '%s': %s", path, err.Error())
	}

	value, err := parseYAML(string(data))
	if err != nil {
		return object.NewError(0, "could not parse yaml file '%s': %s", path, err.Error())
	}
	return objectFromNative(value)
}

func yamlStringify(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 1 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
	}

	data, err := yaml.Marshal(nativeValue(args[0]))
	if err != nil {
		return object.NewError(0, "could not stringify yaml: %s", err.Error())
	}
	return &object.String{Value: string(data)}
}

func parseYAML(input string) (interface{}, error) {
	var decoded interface{}
	if err := yaml.Unmarshal([]byte(input), &decoded); err != nil {
		return nil, err
	}
	return normalizeYAMLValue(decoded), nil
}

func normalizeYAMLValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			result[key] = normalizeYAMLValue(item)
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			result[toStringKey(key)] = normalizeYAMLValue(item)
		}
		return result
	case []interface{}:
		items := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, normalizeYAMLValue(item))
		}
		return items
	default:
		return typed
	}
}

func toStringKey(value interface{}) string {
	if key, ok := value.(string); ok {
		return key
	}
	return objectFromNative(value).Inspect()
}
