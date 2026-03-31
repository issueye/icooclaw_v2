package builtin

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/issueye/icooclaw_lang/internal/object"
)

func newTOMLLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"parse": builtinFunc(tomlParse),
		"parse_file": builtinFunc(tomlParseFile),
		"stringify": builtinFunc(tomlStringify),
	})
}

func tomlParse(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 1 && len(args) != 2 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=1 or 2", len(args))
	}
	input, errObj := stringArg(args[0], "argument to `parse` must be STRING, got %s")
	if errObj != nil {
		return errObj
	}

	value, err := parseTOML(input)
	if err != nil {
		return object.NewError(0, "could not parse toml: %s", err.Error())
	}
	var schema object.Object
	if len(args) == 2 {
		schema = args[1]
	}
	return objectFromNativeWithSchema(value, schema, "toml")
}

func tomlParseFile(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 1 && len(args) != 2 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=1 or 2", len(args))
	}
	path, errObj := stringArg(args[0], "argument to `parse_file` must be STRING, got %s")
	if errObj != nil {
		return errObj
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return object.NewError(0, "could not read toml file '%s': %s", path, err.Error())
	}

	value, err := parseTOML(string(data))
	if err != nil {
		return object.NewError(0, "could not parse toml file '%s': %s", path, err.Error())
	}
	var schema object.Object
	if len(args) == 2 {
		schema = args[1]
	}
	return objectFromNativeWithSchema(value, schema, "toml")
}

func tomlStringify(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 1 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
	}

	root, ok := nativeValueForFormat(args[0], "toml").(map[string]interface{})
	if !ok {
		return object.NewError(0, "argument to `stringify` must be HASH, got %s", args[0].Type())
	}

	output, err := stringifyTOML(root)
	if err != nil {
		return object.NewError(0, "could not stringify toml: %s", err.Error())
	}
	return &object.String{Value: output}
}

func parseTOML(input string) (map[string]interface{}, error) {
	root := map[string]interface{}{}
	current := root

	for lineNo, rawLine := range strings.Split(input, "\n") {
		line := strings.TrimSpace(stripTOMLComment(rawLine))
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") || strings.HasPrefix(line, "[[") {
				return nil, fmt.Errorf("line %d: invalid table header", lineNo+1)
			}

			path := strings.TrimSpace(line[1 : len(line)-1])
			if path == "" {
				return nil, fmt.Errorf("line %d: empty table header", lineNo+1)
			}

			next, err := ensureTOMLTable(root, strings.Split(path, "."))
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo+1, err)
			}
			current = next
			continue
		}

		key, valueText, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected key = value", lineNo+1)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", lineNo+1)
		}

		value, err := parseTOMLValue(strings.TrimSpace(valueText))
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo+1, err)
		}

		assignTarget := current
		keyParts := strings.Split(key, ".")
		if len(keyParts) > 1 {
			assignTarget, err = ensureTOMLTable(current, keyParts[:len(keyParts)-1])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo+1, err)
			}
		}
		assignTarget[keyParts[len(keyParts)-1]] = value
	}

	return root, nil
}

func ensureTOMLTable(root map[string]interface{}, path []string) (map[string]interface{}, error) {
	current := root
	for _, part := range path {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("invalid empty table name")
		}
		if existing, ok := current[part]; ok {
			table, ok := existing.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("key '%s' is not a table", part)
			}
			current = table
			continue
		}
		next := map[string]interface{}{}
		current[part] = next
		current = next
	}
	return current, nil
}

func parseTOMLValue(text string) (interface{}, error) {
	if text == "" {
		return "", nil
	}

	if strings.HasPrefix(text, "\"") {
		value, err := strconv.Unquote(text)
		if err != nil {
			return nil, fmt.Errorf("invalid string: %w", err)
		}
		return value, nil
	}

	if text == "true" {
		return true, nil
	}
	if text == "false" {
		return false, nil
	}

	if strings.HasPrefix(text, "[") {
		return parseTOMLArray(text)
	}

	if strings.ContainsAny(text, ".eE") {
		if value, err := strconv.ParseFloat(text, 64); err == nil {
			return value, nil
		}
	}
	if value, err := strconv.ParseInt(text, 10, 64); err == nil {
		return value, nil
	}

	return text, nil
}

func parseTOMLArray(text string) ([]interface{}, error) {
	if !strings.HasSuffix(text, "]") {
		return nil, fmt.Errorf("invalid array")
	}
	inner := strings.TrimSpace(text[1 : len(text)-1])
	if inner == "" {
		return []interface{}{}, nil
	}

	parts, err := splitTOMLArray(inner)
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, 0, len(parts))
	for _, part := range parts {
		value, err := parseTOMLValue(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func splitTOMLArray(input string) ([]string, error) {
	var parts []string
	var current strings.Builder
	inString := false
	escaped := false
	depth := 0

	for _, r := range input {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\' && inString:
			current.WriteRune(r)
			escaped = true
		case r == '"':
			current.WriteRune(r)
			inString = !inString
		case r == '[' && !inString:
			depth++
			current.WriteRune(r)
		case r == ']' && !inString:
			depth--
			current.WriteRune(r)
		case r == ',' && !inString && depth == 0:
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}

	if inString || depth != 0 {
		return nil, fmt.Errorf("invalid array")
	}
	parts = append(parts, current.String())
	return parts, nil
}

func stripTOMLComment(line string) string {
	var out strings.Builder
	inString := false
	escaped := false

	for _, r := range line {
		switch {
		case escaped:
			out.WriteRune(r)
			escaped = false
		case r == '\\' && inString:
			out.WriteRune(r)
			escaped = true
		case r == '"':
			out.WriteRune(r)
			inString = !inString
		case r == '#' && !inString:
			return out.String()
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

func stringifyTOML(root map[string]interface{}) (string, error) {
	var lines []string
	if err := writeTOMLTable(&lines, nil, root, false); err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.Join(lines, "\n")) + "\n", nil
}

func writeTOMLTable(lines *[]string, path []string, table map[string]interface{}, withHeader bool) error {
	scalarKeys := make([]string, 0)
	tableKeys := make([]string, 0)

	for key, value := range table {
		switch value.(type) {
		case map[string]interface{}:
			tableKeys = append(tableKeys, key)
		default:
			scalarKeys = append(scalarKeys, key)
		}
	}

	sort.Strings(scalarKeys)
	sort.Strings(tableKeys)

	if withHeader {
		if len(*lines) > 0 {
			*lines = append(*lines, "")
		}
		*lines = append(*lines, "["+strings.Join(path, ".")+"]")
	}

	for _, key := range scalarKeys {
		value, err := encodeTOMLValue(table[key])
		if err != nil {
			return fmt.Errorf("key %s: %w", strings.Join(append(path, key), "."), err)
		}
		*lines = append(*lines, key+" = "+value)
	}

	for _, key := range tableKeys {
		child, _ := table[key].(map[string]interface{})
		if err := writeTOMLTable(lines, append(path, key), child, true); err != nil {
			return err
		}
	}
	return nil
}

func encodeTOMLValue(value interface{}) (string, error) {
	switch typed := value.(type) {
	case nil:
		return "\"\"", nil
	case string:
		return strconv.Quote(typed), nil
	case bool:
		if typed {
			return "true", nil
		}
		return "false", nil
	case int:
		return strconv.FormatInt(int64(typed), 10), nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), nil
	case []interface{}:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			part, err := encodeTOMLValue(item)
			if err != nil {
				return "", err
			}
			parts = append(parts, part)
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	case map[string]interface{}:
		return "", fmt.Errorf("inline tables are not supported")
	default:
		return "", fmt.Errorf("unsupported value type %T", value)
	}
}
