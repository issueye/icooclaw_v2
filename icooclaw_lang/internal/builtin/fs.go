package builtin

import (
	"os"
	"path/filepath"

	"github.com/issueye/icooclaw_lang/internal/object"
)

func newFSLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"read_file":  builtinFunc(fsReadFile),
		"write_file": builtinFunc(fsWriteFile),
		"append_file": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 2 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2", len(args))
			}
			path, errObj := stringArg(args[0], "first argument to `append_file` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			content, errObj := stringArg(args[1], "second argument to `append_file` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				return object.NewError(0, "could not open file '%s': %s", path, err.Error())
			}
			defer file.Close()
			if _, err := file.WriteString(content); err != nil {
				return object.NewError(0, "could not append file '%s': %s", path, err.Error())
			}
			return &object.Null{}
		}),
		"exists": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			path, errObj := stringArg(args[0], "argument to `exists` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			_, err := os.Stat(path)
			return boolObject(err == nil)
		}),
		"mkdir": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			path, errObj := stringArg(args[0], "argument to `mkdir` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			if err := os.MkdirAll(path, 0o755); err != nil {
				return object.NewError(0, "could not create directory '%s': %s", path, err.Error())
			}
			return &object.Null{}
		}),
		"remove": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			path, errObj := stringArg(args[0], "argument to `remove` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			if err := os.RemoveAll(path); err != nil {
				return object.NewError(0, "could not remove path '%s': %s", path, err.Error())
			}
			return &object.Null{}
		}),
		"read_dir": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			path, errObj := stringArg(args[0], "argument to `read_dir` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			entries, err := os.ReadDir(path)
			if err != nil {
				return object.NewError(0, "could not read directory '%s': %s", path, err.Error())
			}
			result := make([]object.Object, 0, len(entries))
			for _, entry := range entries {
				info, err := entry.Info()
				if err != nil {
					return object.NewError(0, "could not stat entry '%s': %s", entry.Name(), err.Error())
				}
				result = append(result, fileInfoObject(filepath.Join(path, entry.Name()), info))
			}
			return &object.Array{Elements: result}
		}),
		"stat": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			path, errObj := stringArg(args[0], "argument to `stat` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			info, err := os.Stat(path)
			if err != nil {
				return object.NewError(0, "could not stat path '%s': %s", path, err.Error())
			}
			return fileInfoObject(path, info)
		}),
		"abs": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			path, errObj := stringArg(args[0], "argument to `abs` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			absPath, err := filepath.Abs(path)
			if err != nil {
				return object.NewError(0, "could not resolve absolute path '%s': %s", path, err.Error())
			}
			return &object.String{Value: absPath}
		}),
	})
}

func fsReadFile(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 1 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
	}
	path, errObj := stringArg(args[0], "argument to `read_file` must be STRING, got %s")
	if errObj != nil {
		return errObj
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return object.NewError(0, "could not read file '%s': %s", path, err.Error())
	}
	return &object.String{Value: string(data)}
}

func fsWriteFile(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 2 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=2", len(args))
	}
	path, errObj := stringArg(args[0], "first argument to `write_file` must be STRING, got %s")
	if errObj != nil {
		return errObj
	}
	content, errObj := stringArg(args[1], "second argument to `write_file` must be STRING, got %s")
	if errObj != nil {
		return errObj
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return object.NewError(0, "could not write file '%s': %s", path, err.Error())
	}
	return &object.Null{}
}

func fileInfoObject(path string, info os.FileInfo) *object.Hash {
	return hashObject(map[string]object.Object{
		"name":   &object.String{Value: info.Name()},
		"path":   &object.String{Value: path},
		"size":   &object.Integer{Value: info.Size()},
		"is_dir": boolObject(info.IsDir()),
		"mode":   &object.String{Value: info.Mode().String()},
	})
}
