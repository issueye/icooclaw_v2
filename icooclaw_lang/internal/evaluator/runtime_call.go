package evaluator

import "github.com/issueye/icooclaw_lang/internal/object"

func init() {
	object.RegisterDefaultCaller(callRuntimeObject)
}

func callRuntimeObject(env *object.Environment, fn object.Object, args []object.Object, line int) object.Object {
	switch fn := fn.(type) {
	case *object.Function:
		return callFunction(fn, args, line)
	case *object.Builtin:
		return fn.Fn(env, args...)
	default:
		return object.NewError(line, "not a function: %s", fn.Type())
	}
}
