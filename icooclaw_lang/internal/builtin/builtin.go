package builtin

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/issueye/icooclaw_lang/internal/object"
)

var Builtins = map[string]*object.Builtin{
	"print": {
		Fn: func(args ...object.Object) object.Object {
			var parts []string
			for _, arg := range args {
				parts = append(parts, arg.Inspect())
			}
			fmt.Println(strings.Join(parts, " "))
			return &object.Null{}
		},
	},
	"println": {
		Fn: func(args ...object.Object) object.Object {
			var parts []string
			for _, arg := range args {
				parts = append(parts, arg.Inspect())
			}
			fmt.Println(strings.Join(parts, " "))
			return &object.Null{}
		},
	},
	"len": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			switch arg := args[0].(type) {
			case *object.String:
				return &object.Integer{Value: int64(len(arg.Value))}
			case *object.Array:
				return &object.Integer{Value: int64(len(arg.Elements))}
			default:
				return object.NewError(0, "argument to `len` not supported, got %s", args[0].Type())
			}
		},
	},
	"range": {
		Fn: func(args ...object.Object) object.Object {
			var start, stop int64
			if len(args) == 1 {
				start = 0
				switch arg := args[0].(type) {
				case *object.Integer:
					stop = arg.Value
				default:
					return object.NewError(0, "argument to `range` must be INTEGER, got %s", args[0].Type())
				}
			} else if len(args) == 2 {
				switch arg := args[0].(type) {
				case *object.Integer:
					start = arg.Value
				default:
					return object.NewError(0, "argument to `range` must be INTEGER, got %s", args[0].Type())
				}
				switch arg := args[1].(type) {
				case *object.Integer:
					stop = arg.Value
				default:
					return object.NewError(0, "argument to `range` must be INTEGER, got %s", args[1].Type())
				}
			} else {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1 or 2", len(args))
			}

			elements := make([]object.Object, 0)
			for i := start; i < stop; i++ {
				elements = append(elements, &object.Integer{Value: i})
			}
			return &object.Array{Elements: elements}
		},
	},
	"type": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			return &object.String{Value: string(args[0].Type())}
		},
	},
	"str": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			return &object.String{Value: args[0].Inspect()}
		},
	},
	"int": {
		Fn: func(args ...object.Object) object.Object {
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
		},
	},
	"float": {
		Fn: func(args ...object.Object) object.Object {
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
		},
	},
	"input": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) > 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0 or 1", len(args))
			}
			if len(args) == 1 {
				fmt.Print(args[0].Inspect())
			}
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			return &object.String{Value: strings.TrimSpace(line)}
		},
	},
	"push": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2", len(args))
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return object.NewError(0, "first argument to `push` must be ARRAY, got %s", args[0].Type())
			}
			return &object.Array{Elements: append(arr.Elements, args[1])}
		},
	},
	"pop": {
		Fn: func(args ...object.Object) object.Object {
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
		},
	},
	"keys": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			hash, ok := args[0].(*object.Hash)
			if !ok {
				return object.NewError(0, "argument to `keys` must be HASH, got %s", args[0].Type())
			}
			elements := make([]object.Object, 0, len(hash.Pairs))
			for _, pair := range hash.Pairs {
				elements = append(elements, pair.Key)
			}
			return &object.Array{Elements: elements}
		},
	},
	"values": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			hash, ok := args[0].(*object.Hash)
			if !ok {
				return object.NewError(0, "argument to `values` must be HASH, got %s", args[0].Type())
			}
			elements := make([]object.Object, 0, len(hash.Pairs))
			for _, pair := range hash.Pairs {
				elements = append(elements, pair.Value)
			}
			return &object.Array{Elements: elements}
		},
	},
	"abs": {
		Fn: func(args ...object.Object) object.Object {
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
		},
	},
	"read_file": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(0, "argument to `read_file` must be STRING, got %s", args[0].Type())
			}
			data, err := os.ReadFile(path.Value)
			if err != nil {
				return object.NewError(0, "could not read file '%s': %s", path.Value, err.Error())
			}
			return &object.String{Value: string(data)}
		},
	},
	"write_file": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2", len(args))
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(0, "first argument to `write_file` must be STRING, got %s", args[0].Type())
			}
			content, ok := args[1].(*object.String)
			if !ok {
				return object.NewError(0, "second argument to `write_file` must be STRING, got %s", args[1].Type())
			}
			err := os.WriteFile(path.Value, []byte(content.Value), 0644)
			if err != nil {
				return object.NewError(0, "could not write file '%s': %s", path.Value, err.Error())
			}
			return &object.Null{}
		},
	},
}
