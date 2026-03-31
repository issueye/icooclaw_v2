package builtin

import (
	"fmt"
	"os"
	"sync"

	"github.com/issueye/icooclaw_lang/internal/object"
)

type asyncPool struct {
	size int64
	sem  chan struct{}
	wg   sync.WaitGroup
}

type asyncWaitGroup struct {
	mu    sync.Mutex
	cond  *sync.Cond
	count int64
}

func newAsyncLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"pool": builtinFunc(asyncPoolNew),
		"wait_group": builtinFunc(asyncWaitGroupNew),
		"runtime_concurrency": builtinFunc(asyncRuntimeConcurrency),
		"set_runtime_concurrency": builtinFunc(asyncSetRuntimeConcurrency),
	})
}

func asyncRuntimeConcurrency(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 0 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
	}
	if env == nil || env.Runtime() == nil {
		return object.NewError(0, "runtime is not available")
	}
	return &object.Integer{Value: int64(env.Runtime().MaxConcurrency())}
}

func asyncSetRuntimeConcurrency(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 1 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
	}
	if env == nil || env.Runtime() == nil {
		return object.NewError(0, "runtime is not available")
	}

	size, errObj := integerArg(args[0], "argument to `set_runtime_concurrency` must be INTEGER, got %s")
	if errObj != nil {
		return errObj
	}
	if size <= 0 {
		return object.NewError(0, "argument to `set_runtime_concurrency` must be > 0")
	}

	env.Runtime().SetMaxConcurrency(int(size))
	return &object.Integer{Value: size}
}

func asyncPoolNew(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 1 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
	}
	size, errObj := integerArg(args[0], "argument to `pool` must be INTEGER, got %s")
	if errObj != nil {
		return errObj
	}
	if size <= 0 {
		return object.NewError(0, "argument to `pool` must be > 0")
	}

	pool := &asyncPool{
		size: size,
		sem:  make(chan struct{}, size),
	}

	return hashObject(map[string]object.Object{
		"submit": builtinFunc(func(callEnv *object.Environment, args ...object.Object) object.Object {
			return asyncPoolSubmit(callEnv, pool, args...)
		}),
		"wait": builtinFunc(func(callEnv *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			pool.wg.Wait()
			return object.NullObject()
		}),
		"size": builtinFunc(func(callEnv *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return &object.Integer{Value: pool.size}
		}),
	})
}

func asyncPoolSubmit(env *object.Environment, pool *asyncPool, args ...object.Object) object.Object {
	if len(args) != 1 && len(args) != 2 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=1 or 2", len(args))
	}

	fn, errObj := callableArg(args[0], "first argument to `submit` must be FUNCTION, got %s")
	if errObj != nil {
		return errObj
	}

	callArgs := []object.Object{}
	if len(args) == 2 {
		arrayValue, arrayErr := arrayArg(args[1], "second argument to `submit` must be ARRAY, got %s")
		if arrayErr != nil {
			return arrayErr
		}
		callArgs = append(callArgs, arrayValue.Elements...)
	}

	dispatchEnv := object.NewDetachedEnvironment(env)
	pool.wg.Add(1)
	env.Go(func() {
		pool.sem <- struct{}{}
		defer func() {
			<-pool.sem
			pool.wg.Done()
		}()

		result := dispatchEnv.Call(fn, callArgs, 0)
		if err, ok := result.(*object.Error); ok {
			fmt.Fprintln(os.Stderr, err.Inspect())
		}
	})

	return object.NullObject()
}

func asyncWaitGroupNew(env *object.Environment, args ...object.Object) object.Object {
	if len(args) != 0 {
		return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
	}

	group := &asyncWaitGroup{}
	group.cond = sync.NewCond(&group.mu)

	return hashObject(map[string]object.Object{
		"add": builtinFunc(func(callEnv *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			delta, errObj := integerArg(args[0], "argument to `add` must be INTEGER, got %s")
			if errObj != nil {
				return errObj
			}
			return group.add(delta)
		}),
		"done": builtinFunc(func(callEnv *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return group.add(-1)
		}),
		"wait": builtinFunc(func(callEnv *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			group.wait()
			return object.NullObject()
		}),
		"count": builtinFunc(func(callEnv *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			group.mu.Lock()
			defer group.mu.Unlock()
			return &object.Integer{Value: group.count}
		}),
	})
}

func (g *asyncWaitGroup) add(delta int64) object.Object {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.count+delta < 0 {
		return object.NewError(0, "wait group counter cannot be negative")
	}
	g.count += delta
	if g.count == 0 {
		g.cond.Broadcast()
	}
	return object.NullObject()
}

func (g *asyncWaitGroup) wait() {
	g.mu.Lock()
	defer g.mu.Unlock()

	for g.count > 0 {
		g.cond.Wait()
	}
}
