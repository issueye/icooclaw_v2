package builtin

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/issueye/icooclaw_lang/internal/object"
)

func coreBuiltins() map[string]object.Object {
	return map[string]object.Object{
		"print": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			var parts []string
			for _, arg := range args {
				parts = append(parts, arg.Inspect())
			}
			fmt.Println(strings.Join(parts, " "))
			return &object.Null{}
		}),
		"println": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			var parts []string
			for _, arg := range args {
				parts = append(parts, arg.Inspect())
			}
			fmt.Println(strings.Join(parts, " "))
			return &object.Null{}
		}),
		"len": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			switch arg := args[0].(type) {
			case *object.String:
				return &object.Integer{Value: int64(len(arg.Value))}
			case *object.Array:
				return &object.Integer{Value: int64(len(arg.Elements))}
			case *object.Hash:
				return &object.Integer{Value: int64(len(arg.Pairs))}
			default:
				return object.NewError(0, "argument to `len` not supported, got %s", args[0].Type())
			}
		}),
		"range": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			var start, stop int64
			if len(args) == 1 {
				start = 0
				value, errObj := integerArg(args[0], "argument to `range` must be INTEGER, got %s")
				if errObj != nil {
					return errObj
				}
				stop = value
			} else if len(args) == 2 {
				value, errObj := integerArg(args[0], "argument to `range` must be INTEGER, got %s")
				if errObj != nil {
					return errObj
				}
				start = value
				value, errObj = integerArg(args[1], "argument to `range` must be INTEGER, got %s")
				if errObj != nil {
					return errObj
				}
				stop = value
			} else {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1 or 2", len(args))
			}

			elements := make([]object.Object, 0)
			for i := start; i < stop; i++ {
				elements = append(elements, &object.Integer{Value: i})
			}
			return &object.Array{Elements: elements}
		}),
		"type": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			return &object.String{Value: string(args[0].Type())}
		}),
		"str": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			return &object.String{Value: args[0].Inspect()}
		}),
		"int": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			switch arg := args[0].(type) {
			case *object.Integer:
				return arg
			case *object.Float:
				return &object.Integer{Value: int64(arg.Value)}
			case *object.String:
				val, err := strconv.ParseInt(arg.Value, 0, 64)
				if err != nil {
					return object.NewError(0, "could not parse string '%s' as integer", arg.Value)
				}
				return &object.Integer{Value: val}
			case *object.Boolean:
				if arg.Value {
					return &object.Integer{Value: 1}
				}
				return &object.Integer{Value: 0}
			default:
				return object.NewError(0, "cannot convert %s to INTEGER", args[0].Type())
			}
		}),
		"float": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			switch arg := args[0].(type) {
			case *object.Float:
				return arg
			case *object.Integer:
				return &object.Float{Value: float64(arg.Value)}
			case *object.String:
				val, err := strconv.ParseFloat(arg.Value, 64)
				if err != nil {
					return object.NewError(0, "could not parse string '%s' as float", arg.Value)
				}
				return &object.Float{Value: val}
			default:
				return object.NewError(0, "cannot convert %s to FLOAT", args[0].Type())
			}
		}),
		"input": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) > 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0 or 1", len(args))
			}
			if len(args) == 1 {
				fmt.Print(args[0].Inspect())
			}
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			return &object.String{Value: strings.TrimSpace(line)}
		}),
		"push": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 2 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2", len(args))
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return object.NewError(0, "first argument to `push` must be ARRAY, got %s", args[0].Type())
			}
			return &object.Array{Elements: append(arr.Elements, args[1])}
		}),
		"pop": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return object.NewError(0, "argument to `pop` must be ARRAY, got %s", args[0].Type())
			}
			if len(arr.Elements) == 0 {
				return &object.Null{}
			}
			return arr.Elements[len(arr.Elements)-1]
		}),
		"keys": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			hashObj, ok := args[0].(*object.Hash)
			if !ok {
				return object.NewError(0, "argument to `keys` must be HASH, got %s", args[0].Type())
			}
			elements := make([]object.Object, 0, len(hashObj.Pairs))
			for _, pair := range hashObj.Pairs {
				elements = append(elements, pair.Key)
			}
			return &object.Array{Elements: elements}
		}),
		"values": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			hashObj, ok := args[0].(*object.Hash)
			if !ok {
				return object.NewError(0, "argument to `values` must be HASH, got %s", args[0].Type())
			}
			elements := make([]object.Object, 0, len(hashObj.Pairs))
			for _, pair := range hashObj.Pairs {
				elements = append(elements, pair.Value)
			}
			return &object.Array{Elements: elements}
		}),
		"abs": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			switch arg := args[0].(type) {
			case *object.Integer:
				if arg.Value < 0 {
					return &object.Integer{Value: -arg.Value}
				}
				return arg
			case *object.Float:
				if arg.Value < 0 {
					return &object.Float{Value: -arg.Value}
				}
				return arg
			default:
				return object.NewError(0, "argument to `abs` must be numeric, got %s", args[0].Type())
			}
		}),
		"read_file":  builtinFunc(fsReadFile),
		"write_file": builtinFunc(fsWriteFile),
	}
}
