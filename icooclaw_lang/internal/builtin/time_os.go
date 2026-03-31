package builtin

import (
	"os"
	"strings"
	"time"

	"github.com/issueye/icooclaw_lang/internal/object"
)

func newTimeLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"now": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return timeObject(time.Now())
		}),
		"now_unix": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return &object.Integer{Value: time.Now().Unix()}
		}),
		"now_unix_ms": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return &object.Integer{Value: time.Now().UnixMilli()}
		}),
		"sleep_ms": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			ms, errObj := integerArg(args[0], "argument to `sleep_ms` must be INTEGER, got %s")
			if errObj != nil {
				return errObj
			}
			time.Sleep(time.Duration(ms) * time.Millisecond)
			return &object.Null{}
		}),
		"sleep": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			switch value := args[0].(type) {
			case *object.Integer:
				time.Sleep(time.Duration(value.Value) * time.Second)
			case *object.Float:
				time.Sleep(time.Duration(value.Value * float64(time.Second)))
			default:
				return object.NewError(0, "argument to `sleep` must be INTEGER or FLOAT, got %s", args[0].Type())
			}
			return &object.Null{}
		}),
	})
}

func newOSLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"cwd": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			dir, err := os.Getwd()
			if err != nil {
				return object.NewError(0, "could not get cwd: %s", err.Error())
			}
			return &object.String{Value: dir}
		}),
		"getenv": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			name, errObj := stringArg(args[0], "argument to `getenv` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			return &object.String{Value: os.Getenv(name)}
		}),
		"setenv": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 2 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2", len(args))
			}
			name, errObj := stringArg(args[0], "first argument to `setenv` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			value, errObj := stringArg(args[1], "second argument to `setenv` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			if err := os.Setenv(name, value); err != nil {
				return object.NewError(0, "could not set env '%s': %s", name, err.Error())
			}
			return &object.Null{}
		}),
		"pid": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return &object.Integer{Value: int64(os.Getpid())}
		}),
		"hostname": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			host, err := os.Hostname()
			if err != nil {
				return object.NewError(0, "could not get hostname: %s", err.Error())
			}
			return &object.String{Value: host}
		}),
		"temp_dir": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return &object.String{Value: os.TempDir()}
		}),
		"args": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return arrayOfStrings(env.CLIArgs())
		}),
		"arg": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			index, errObj := integerArg(args[0], "argument to `arg` must be INTEGER, got %s")
			if errObj != nil {
				return errObj
			}

			values := env.CLIArgs()
			if index < 0 || int(index) >= len(values) {
				return &object.Null{}
			}
			return &object.String{Value: values[index]}
		}),
		"has_flag": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			name, errObj := stringArg(args[0], "argument to `has_flag` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}

			_, ok := lookupCLIFlag(env.CLIArgs(), name)
			return boolObject(ok)
		}),
		"flag": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			name, errObj := stringArg(args[0], "argument to `flag` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}

			value, ok := lookupCLIFlag(env.CLIArgs(), name)
			if !ok {
				return &object.Null{}
			}
			return &object.String{Value: value}
		}),
		"flag_or": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 2 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2", len(args))
			}
			name, errObj := stringArg(args[0], "first argument to `flag_or` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			fallback, errObj := stringArg(args[1], "second argument to `flag_or` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}

			value, ok := lookupCLIFlag(env.CLIArgs(), name)
			if !ok {
				return &object.String{Value: fallback}
			}
			return &object.String{Value: value}
		}),
		"script_path": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			scriptPath := env.ScriptPath()
			if scriptPath == "" {
				return &object.Null{}
			}
			return &object.String{Value: scriptPath}
		}),
	})
}

func timeObject(now time.Time) *object.Hash {
	return hashObject(map[string]object.Object{
		"unix":      &object.Integer{Value: now.Unix()},
		"unix_ms":   &object.Integer{Value: now.UnixMilli()},
		"rfc_3339":  &object.String{Value: now.Format(time.RFC3339)},
		"date":      &object.String{Value: now.Format("2006-01-02")},
		"time":      &object.String{Value: now.Format("15:04:05")},
		"year":      &object.Integer{Value: int64(now.Year())},
		"month":     &object.Integer{Value: int64(now.Month())},
		"day":       &object.Integer{Value: int64(now.Day())},
		"hour":      &object.Integer{Value: int64(now.Hour())},
		"minute":    &object.Integer{Value: int64(now.Minute())},
		"second":    &object.Integer{Value: int64(now.Second())},
		"weekday":   &object.String{Value: now.Weekday().String()},
		"timestamp": &object.String{Value: now.Format("2006-01-02 15:04:05")},
	})
}

func lookupCLIFlag(args []string, name string) (string, bool) {
	key := normalizeCLIFlagName(name)
	if key == "" {
		return "", false
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}

		if !strings.HasPrefix(arg, "-") || arg == "-" {
			continue
		}

		raw := strings.TrimLeft(arg, "-")
		if raw == "" {
			continue
		}

		if parts := strings.SplitN(raw, "=", 2); len(parts) == 2 {
			if parts[0] == key {
				return parts[1], true
			}
			continue
		}

		if raw != key {
			continue
		}

		if i+1 < len(args) && isCLIFlagValue(args[i+1]) {
			return args[i+1], true
		}
		return "true", true
	}

	return "", false
}

func normalizeCLIFlagName(name string) string {
	return strings.TrimLeft(strings.TrimSpace(name), "-")
}

func isCLIFlagValue(value string) bool {
	if value == "" {
		return true
	}
	if value == "--" {
		return false
	}
	if strings.HasPrefix(value, "-") {
		if len(value) == 1 {
			return false
		}
		switch value[1] {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.':
			return true
		default:
			return false
		}
	}
	return true
}
