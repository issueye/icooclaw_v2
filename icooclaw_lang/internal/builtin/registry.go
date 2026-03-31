package builtin

import "github.com/issueye/icooclaw_lang/internal/object"

var Builtins = buildBuiltins()

func buildBuiltins() map[string]object.Object {
	builtins := make(map[string]object.Object)

	for name, value := range coreBuiltins() {
		builtins[name] = value
	}

	for name, value := range libraryBuiltins() {
		builtins[name] = value
	}

	return builtins
}

func libraryBuiltins() map[string]object.Object {
	return map[string]object.Object{
		"db":        newDBLib(),
		"fs":        newFSLib(),
		"http":      newHTTPLib(),
		"json":      newJSONLib(),
		"toml":      newTOMLLib(),
		"yaml":      newYAMLLib(),
		"log":       newLogLib(),
		"time":      newTimeLib(),
		"os":        newOSLib(),
		"exec":      newExecLib(),
		"path":      newPathLib(),
		"crypto":    newCryptoLib(),
		"websocket": newWebSocketLib(),
		"sse":       newSSELib(),
	}
}
