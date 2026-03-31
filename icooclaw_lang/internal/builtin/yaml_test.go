package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/issueye/icooclaw_lang/internal/object"
)

func TestParseYAML(t *testing.T) {
	value, err := parseYAML(`
name: ic_agent
port: 8080
enabled: true
tags:
  - agent
  - http
service:
  addr: 127.0.0.1:8080
`)
	if err != nil {
		t.Fatalf("parseYAML error: %v", err)
	}

	root, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("parsed yaml root = %T", value)
	}
	if got := root["name"]; got != "ic_agent" {
		t.Fatalf("name = %#v", got)
	}

	service, ok := root["service"].(map[string]interface{})
	if !ok {
		t.Fatalf("service map missing: %#v", root["service"])
	}
	if got := service["addr"]; got != "127.0.0.1:8080" {
		t.Fatalf("service.addr = %#v", got)
	}
}

func TestYAMLBuiltinParseFileAndStringify(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "pkg.yaml")
	input := `
name: demo
service:
  addr: 127.0.0.1:9090
  debug: false
features:
  - chat
  - tools
`
	if err := os.WriteFile(configPath, []byte(input), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	parseFileBuiltin, ok := newYAMLLib().Pairs["parse_file"].Value.(*object.Builtin)
	if !ok {
		t.Fatal("parse_file builtin missing")
	}
	parsed := parseFileBuiltin.Fn(object.NewEnvironment(), &object.String{Value: configPath})
	hash, ok := parsed.(*object.Hash)
	if !ok {
		t.Fatalf("parse_file result = %T", parsed)
	}

	servicePair, ok := hash.Pairs["service"]
	if !ok {
		t.Fatalf("service pair missing: %#v", hash.Pairs)
	}
	serviceHash, ok := servicePair.Value.(*object.Hash)
	if !ok {
		t.Fatalf("service hash = %T", servicePair.Value)
	}
	addrPair, ok := serviceHash.Pairs["addr"]
	if !ok {
		t.Fatalf("service.addr missing")
	}
	addrValue, ok := addrPair.Value.(*object.String)
	if !ok || addrValue.Value != "127.0.0.1:9090" {
		t.Fatalf("service.addr = %#v", addrPair.Value)
	}
	debugPair, ok := serviceHash.Pairs["debug"]
	if !ok {
		t.Fatalf("service.debug missing")
	}
	debugValue, ok := debugPair.Value.(*object.Boolean)
	if !ok || debugValue.Value {
		t.Fatalf("service.debug = %#v", debugPair.Value)
	}

	featuresPair, ok := hash.Pairs["features"]
	if !ok {
		t.Fatalf("features pair missing")
	}
	features, ok := featuresPair.Value.(*object.Array)
	if !ok || len(features.Elements) != 2 {
		t.Fatalf("features = %#v", featuresPair.Value)
	}

	stringifyBuiltin, ok := newYAMLLib().Pairs["stringify"].Value.(*object.Builtin)
	if !ok {
		t.Fatal("stringify builtin missing")
	}
	stringified := stringifyBuiltin.Fn(object.NewEnvironment(), hash)
	stringValue, ok := stringified.(*object.String)
	if !ok {
		t.Fatalf("stringify result = %T", stringified)
	}

	if !strings.Contains(stringValue.Value, "name: demo") {
		t.Fatalf("stringified YAML missing name: %q", stringValue.Value)
	}
	if !strings.Contains(stringValue.Value, "addr: 127.0.0.1:9090") {
		t.Fatalf("stringified YAML missing service addr: %q", stringValue.Value)
	}
	if !strings.Contains(stringValue.Value, "- chat") {
		t.Fatalf("stringified YAML missing array item: %q", stringValue.Value)
	}
}
