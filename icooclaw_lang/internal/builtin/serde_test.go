package builtin

import (
	"strings"
	"testing"

	"github.com/issueye/icooclaw_lang/internal/object"
)

func TestSerdeStringifyHonorsFieldAnnotations(t *testing.T) {
	payload := hashObject(map[string]object.Object{
		serdeMetaKey: hashObject(map[string]object.Object{
			"name": hashObject(map[string]object.Object{
				"json": &object.String{Value: "user_name"},
				"yaml": &object.String{Value: "user_name"},
				"toml": &object.String{Value: "user_name"},
			}),
			"password": hashObject(map[string]object.Object{
				"json": &object.String{Value: "-"},
				"yaml": &object.String{Value: "-"},
				"toml": &object.String{Value: "-"},
			}),
			"age": hashObject(map[string]object.Object{
				"json": &object.String{Value: "age,omitempty"},
				"yaml": &object.String{Value: "age,omitempty"},
				"toml": &object.String{Value: "age,omitempty"},
			}),
		}),
		"name":     &object.String{Value: "icooclaw"},
		"password": &object.String{Value: "secret"},
		"age":      &object.Integer{Value: 0},
	})

	jsonText := jsonStringify(object.NewEnvironment(), payload).(*object.String).Value
	if !strings.Contains(jsonText, `"user_name":"icooclaw"`) {
		t.Fatalf("json stringify missing aliased field: %q", jsonText)
	}
	if strings.Contains(jsonText, "password") {
		t.Fatalf("json stringify should skip password: %q", jsonText)
	}
	if strings.Contains(jsonText, `"age":`) {
		t.Fatalf("json stringify should omit empty age: %q", jsonText)
	}

	yamlText := yamlStringify(object.NewEnvironment(), payload).(*object.String).Value
	if !strings.Contains(yamlText, "user_name: icooclaw") {
		t.Fatalf("yaml stringify missing aliased field: %q", yamlText)
	}
	if strings.Contains(yamlText, "password:") {
		t.Fatalf("yaml stringify should skip password: %q", yamlText)
	}
	if strings.Contains(yamlText, "\nage:") {
		t.Fatalf("yaml stringify should omit empty age: %q", yamlText)
	}

	tomlText := tomlStringify(object.NewEnvironment(), payload).(*object.String).Value
	if !strings.Contains(tomlText, "user_name = \"icooclaw\"") {
		t.Fatalf("toml stringify missing aliased field: %q", tomlText)
	}
	if strings.Contains(tomlText, "password =") {
		t.Fatalf("toml stringify should skip password: %q", tomlText)
	}
	if strings.Contains(tomlText, "\nage =") {
		t.Fatalf("toml stringify should omit empty age: %q", tomlText)
	}
}

func TestSerdeParseWithSchemaMapsAliasesBackToInternalNames(t *testing.T) {
	schema := hashObject(map[string]object.Object{
		serdeMetaKey: hashObject(map[string]object.Object{
			"name": hashObject(map[string]object.Object{
				"json": &object.String{Value: "user_name"},
				"yaml": &object.String{Value: "user_name"},
				"toml": &object.String{Value: "user_name"},
			}),
			"profile": hashObject(map[string]object.Object{
				"json": &object.String{Value: "user_profile"},
				"yaml": &object.String{Value: "user_profile"},
				"toml": &object.String{Value: "user_profile"},
			}),
		}),
		"name": &object.String{},
		"profile": hashObject(map[string]object.Object{
			serdeMetaKey: hashObject(map[string]object.Object{
				"display_name": hashObject(map[string]object.Object{
					"json": &object.String{Value: "displayName"},
					"yaml": &object.String{Value: "displayName"},
					"toml": &object.String{Value: "displayName"},
				}),
			}),
			"display_name": &object.String{},
		}),
	})

	jsonParsed := jsonParse(
		object.NewEnvironment(),
		&object.String{Value: `{"user_name":"icooclaw","user_profile":{"displayName":"agent"}}`},
		schema,
	)
	assertSerdeParsed(t, jsonParsed, "json")

	yamlParsed := yamlParse(
		object.NewEnvironment(),
		&object.String{Value: "user_name: icooclaw\nuser_profile:\n  displayName: agent\n"},
		schema,
	)
	assertSerdeParsed(t, yamlParsed, "yaml")

	tomlParsed := tomlParse(
		object.NewEnvironment(),
		&object.String{Value: "user_name = \"icooclaw\"\n[user_profile]\ndisplayName = \"agent\"\n"},
		schema,
	)
	assertSerdeParsed(t, tomlParsed, "toml")
}

func assertSerdeParsed(t *testing.T, parsed object.Object, format string) {
	t.Helper()

	root, ok := parsed.(*object.Hash)
	if !ok {
		t.Fatalf("%s parsed result = %T", format, parsed)
	}

	namePair, ok := root.Pairs["name"]
	if !ok || namePair.Value.Inspect() != "icooclaw" {
		t.Fatalf("%s parsed name = %#v", format, namePair.Value)
	}

	profilePair, ok := root.Pairs["profile"]
	if !ok {
		t.Fatalf("%s parsed profile missing", format)
	}
	profile, ok := profilePair.Value.(*object.Hash)
	if !ok {
		t.Fatalf("%s parsed profile = %T", format, profilePair.Value)
	}
	displayPair, ok := profile.Pairs["display_name"]
	if !ok || displayPair.Value.Inspect() != "agent" {
		t.Fatalf("%s parsed display_name = %#v", format, displayPair.Value)
	}

	if _, ok := root.Pairs[serdeMetaKey]; !ok {
		t.Fatalf("%s parsed result should preserve __serde__ metadata from schema", format)
	}
}
