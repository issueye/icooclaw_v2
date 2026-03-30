package builtin

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/issueye/icooclaw_lang/internal/object"
)

type logLevel int

const (
	logLevelDebug logLevel = 10
	logLevelInfo  logLevel = 20
	logLevelWarn  logLevel = 30
	logLevelError logLevel = 40
)

type nativeLogger struct {
	mu         sync.Mutex
	level      logLevel
	jsonMode   bool
	writer     io.Writer
	closer     io.Closer
	outputName string
}

var defaultLogger = newNativeLogger()

func newNativeLogger() *nativeLogger {
	return &nativeLogger{
		level:      logLevelInfo,
		writer:     os.Stdout,
		outputName: "stdout",
	}
}

func newLogLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"debug": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			return defaultLogger.log(logLevelDebug, args...)
		}),
		"info": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			return defaultLogger.log(logLevelInfo, args...)
		}),
		"warn": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			return defaultLogger.log(logLevelWarn, args...)
		}),
		"error": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			return defaultLogger.log(logLevelError, args...)
		}),
		"set_level": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			levelName, errObj := stringArg(args[0], "argument to `set_level` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			level, ok := parseLogLevel(levelName)
			if !ok {
				return object.NewError(0, "unknown log level '%s'", levelName)
			}
			defaultLogger.mu.Lock()
			defaultLogger.level = level
			defaultLogger.mu.Unlock()
			return &object.Null{}
		}),
		"level": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			defaultLogger.mu.Lock()
			defer defaultLogger.mu.Unlock()
			return &object.String{Value: defaultLogger.level.String()}
		}),
		"set_json": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			value, ok := args[0].(*object.Boolean)
			if !ok {
				return object.NewError(0, "argument to `set_json` must be BOOLEAN, got %s", args[0].Type())
			}
			defaultLogger.mu.Lock()
			defaultLogger.jsonMode = value.Value
			defaultLogger.mu.Unlock()
			return &object.Null{}
		}),
		"is_json": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			defaultLogger.mu.Lock()
			defer defaultLogger.mu.Unlock()
			return boolObject(defaultLogger.jsonMode)
		}),
		"set_output": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			output, errObj := stringArg(args[0], "argument to `set_output` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			return defaultLogger.setOutput(output)
		}),
		"output": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			defaultLogger.mu.Lock()
			defer defaultLogger.mu.Unlock()
			return &object.String{Value: defaultLogger.outputName}
		}),
		"reset": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			defaultLogger.reset()
			return &object.Null{}
		}),
	})
}

func (l *nativeLogger) log(level logLevel, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(0, "wrong number of arguments. got=0, want>=1")
	}

	fields := map[string]any{}
	messageArgs := args
	if hash, ok := args[0].(*object.Hash); ok {
		fields = hashToNativeMap(hash)
		messageArgs = args[1:]
	}
	if len(messageArgs) == 0 {
		return object.NewError(0, "log message is required")
	}

	parts := make([]string, 0, len(messageArgs))
	for _, arg := range messageArgs {
		parts = append(parts, arg.Inspect())
	}
	message := strings.Join(parts, " ")

	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return &object.Null{}
	}

	line, err := l.formatLine(level, message, fields)
	if err != nil {
		return object.NewError(0, "could not format log line: %s", err.Error())
	}
	if _, err := io.WriteString(l.writer, line); err != nil {
		return object.NewError(0, "could not write log line: %s", err.Error())
	}
	return &object.Null{}
}

func (l *nativeLogger) formatLine(level logLevel, message string, fields map[string]any) (string, error) {
	timestamp := time.Now().Format(time.RFC3339)

	if l.jsonMode {
		payload := map[string]any{
			"timestamp": timestamp,
			"level":     level.String(),
			"message":   message,
		}
		if len(fields) > 0 {
			payload["fields"] = fields
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	}

	var builder strings.Builder
	builder.WriteString(timestamp)
	builder.WriteString(" ")
	builder.WriteString(level.String())
	builder.WriteString(" ")
	builder.WriteString(message)

	if len(fields) > 0 {
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			builder.WriteString(" ")
			builder.WriteString(key)
			builder.WriteString("=")
			builder.WriteString(fmt.Sprintf("%v", fields[key]))
		}
	}

	builder.WriteString("\n")
	return builder.String(), nil
}

func (l *nativeLogger) setOutput(output string) object.Object {
	l.mu.Lock()
	defer l.mu.Unlock()

	switch output {
	case "stdout":
		l.closeCurrentWriterLocked()
		l.writer = os.Stdout
		l.outputName = "stdout"
		return &object.Null{}
	case "stderr":
		l.closeCurrentWriterLocked()
		l.writer = os.Stderr
		l.outputName = "stderr"
		return &object.Null{}
	}

	dir := filepath.Dir(output)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return object.NewError(0, "could not create log directory '%s': %s", dir, err.Error())
		}
	}

	file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return object.NewError(0, "could not open log file '%s': %s", output, err.Error())
	}

	l.closeCurrentWriterLocked()
	l.writer = file
	l.closer = file
	l.outputName = output
	return &object.Null{}
}

func (l *nativeLogger) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closeCurrentWriterLocked()
	l.level = logLevelInfo
	l.jsonMode = false
	l.writer = os.Stdout
	l.outputName = "stdout"
}

func (l *nativeLogger) closeCurrentWriterLocked() {
	if l.closer != nil {
		_ = l.closer.Close()
		l.closer = nil
	}
}

func parseLogLevel(value string) (logLevel, bool) {
	switch strings.ToLower(value) {
	case "debug":
		return logLevelDebug, true
	case "info":
		return logLevelInfo, true
	case "warn", "warning":
		return logLevelWarn, true
	case "error":
		return logLevelError, true
	default:
		return 0, false
	}
}

func (l logLevel) String() string {
	switch l {
	case logLevelDebug:
		return "DEBUG"
	case logLevelInfo:
		return "INFO"
	case logLevelWarn:
		return "WARN"
	case logLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func hashToNativeMap(hash *object.Hash) map[string]any {
	values := make(map[string]any, len(hash.Pairs))
	for _, pair := range hash.Pairs {
		values[pair.Key.Inspect()] = nativeValue(pair.Value)
	}
	return values
}
