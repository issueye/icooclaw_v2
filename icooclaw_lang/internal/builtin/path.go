package builtin

import (
	"path/filepath"

	"github.com/issueye/icooclaw_lang/internal/object"
)

func newPathLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"join": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(0, "wrong number of arguments. got=0, want>=1")
			}
			parts := make([]string, 0, len(args))
			for _, arg := range args {
				part, errObj := stringArg(arg, "path join argument must be STRING, got %s")
				if errObj != nil {
					return errObj
				}
				parts = append(parts, part)
			}
			return &object.String{Value: filepath.Join(parts...)}
		}),
		"base": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			path, errObj := stringArg(args[0], "argument to `base` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			return &object.String{Value: filepath.Base(path)}
		}),
		"ext": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			path, errObj := stringArg(args[0], "argument to `ext` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			return &object.String{Value: filepath.Ext(path)}
		}),
		"dir": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			path, errObj := stringArg(args[0], "argument to `dir` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			return &object.String{Value: filepath.Dir(path)}
		}),
		"clean": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			path, errObj := stringArg(args[0], "argument to `clean` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			return &object.String{Value: filepath.Clean(path)}
		}),
	})
}
