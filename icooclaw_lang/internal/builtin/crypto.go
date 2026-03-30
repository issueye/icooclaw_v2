package builtin

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"github.com/issueye/icooclaw_lang/internal/object"
)

func newCryptoLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"md5": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			input, errObj := stringArg(args[0], "argument to `md5` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			sum := md5.Sum([]byte(input))
			return &object.String{Value: hex.EncodeToString(sum[:])}
		}),
		"sha_1": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			input, errObj := stringArg(args[0], "argument to `sha_1` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			sum := sha1.Sum([]byte(input))
			return &object.String{Value: hex.EncodeToString(sum[:])}
		}),
		"sha_256": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			input, errObj := stringArg(args[0], "argument to `sha_256` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			sum := sha256.Sum256([]byte(input))
			return &object.String{Value: hex.EncodeToString(sum[:])}
		}),
		"base_64_encode": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			input, errObj := stringArg(args[0], "argument to `base_64_encode` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			return &object.String{Value: base64.StdEncoding.EncodeToString([]byte(input))}
		}),
		"base_64_decode": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			input, errObj := stringArg(args[0], "argument to `base_64_decode` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			data, err := base64.StdEncoding.DecodeString(input)
			if err != nil {
				return object.NewError(0, "could not decode base64: %s", err.Error())
			}
			return &object.String{Value: string(data)}
		}),
	})
}
