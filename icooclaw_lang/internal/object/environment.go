package object

import (
	"sync"

	"github.com/issueye/icooclaw_lang/internal/ast"
)

const defaultLocalStoreCapacity = 4

type Environment struct {
	store      map[string]Object
	outer      *Environment
	consts     map[string]bool
	exports    map[string]bool
	cliArgs    []string
	scriptPath string
	runtime    *Runtime
	transient  bool
}

func NewEnvironment() *Environment {
	return &Environment{
		store:   make(map[string]Object),
		runtime: NewRuntime(),
	}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	return &Environment{
		store:      make(map[string]Object, defaultLocalStoreCapacity),
		outer:      outer,
		cliArgs:    outer.cliArgs,
		scriptPath: outer.scriptPath,
		runtime:    outer.runtime,
	}
}

func NewDetachedEnvironment(proto *Environment) *Environment {
	return &Environment{
		store:      make(map[string]Object, defaultLocalStoreCapacity),
		cliArgs:    proto.cliArgs,
		scriptPath: proto.scriptPath,
		runtime:    proto.runtime,
	}
}

func AcquireTransientEnclosedEnvironment(outer *Environment) *Environment {
	env := transientEnvPool.Get().(*Environment)
	env.outer = outer
	env.cliArgs = outer.cliArgs
	env.scriptPath = outer.scriptPath
	env.runtime = outer.runtime
	env.transient = true
	return env
}

func ReleaseTransientEnvironment(env *Environment) {
	clear(env.store)
	if env.consts != nil {
		clear(env.consts)
	}
	if env.exports != nil {
		clear(env.exports)
	}
	env.outer = nil
	env.cliArgs = nil
	env.scriptPath = ""
	env.runtime = nil
	env.transient = false
	transientEnvPool.Put(env)
}

func (e *Environment) IsTransient() bool {
	return e.transient
}

func (e *Environment) Runtime() *Runtime {
	return e.runtime
}

func (e *Environment) Get(name string) (Object, bool) {
	e.runtime.mu.RLock()
	defer e.runtime.mu.RUnlock()

	return e.getUnlocked(name)
}

func (e *Environment) Set(name string, val Object) Object {
	e.runtime.mu.Lock()
	defer e.runtime.mu.Unlock()

	if _, ok := e.store[name]; ok {
		if e.consts != nil && e.consts[name] {
			return NewError(0, "cannot reassign to constant '%s'", name)
		}
		e.store[name] = val
		return val
	}

	if target := e.findVarUnlocked(name); target != nil {
		if target.consts != nil && target.consts[name] {
			return NewError(0, "cannot reassign to constant '%s'", name)
		}
		target.store[name] = val
		return val
	}

	if e.consts != nil && e.consts[name] {
		return NewError(0, "cannot reassign to constant '%s'", name)
	}
	e.store[name] = val
	return val
}

func (e *Environment) DefineLocal(name string, val Object) Object {
	e.runtime.mu.Lock()
	defer e.runtime.mu.Unlock()

	if e.consts != nil && e.consts[name] {
		return NewError(0, "cannot reassign to constant '%s'", name)
	}
	e.store[name] = val
	return val
}

func (e *Environment) DefineLocals(bindings map[string]Object) Object {
	e.runtime.mu.Lock()
	defer e.runtime.mu.Unlock()

	for name, val := range bindings {
		if e.consts != nil && e.consts[name] {
			return NewError(0, "cannot reassign to constant '%s'", name)
		}
		e.store[name] = val
	}
	return NullObject()
}

func (e *Environment) DefineFunctionParams(params []*ast.Identifier, args []Object, line int) Object {
	if len(args) != len(params) {
		return NewError(line, "wrong number of arguments: want=%d, got=%d", len(params), len(args))
	}

	e.runtime.mu.Lock()
	defer e.runtime.mu.Unlock()

	for i, param := range params {
		name := param.Value
		if e.consts != nil && e.consts[name] {
			return NewError(0, "cannot reassign to constant '%s'", name)
		}
		e.store[name] = args[i]
	}
	return NullObject()
}

func (e *Environment) findVar(name string) *Environment {
	e.runtime.mu.RLock()
	defer e.runtime.mu.RUnlock()

	return e.findVarUnlocked(name)
}

func (e *Environment) SetConst(name string, val Object) Object {
	e.runtime.mu.Lock()
	defer e.runtime.mu.Unlock()

	if e.consts == nil {
		e.consts = make(map[string]bool)
	}
	e.store[name] = val
	e.consts[name] = true
	return val
}

func (e *Environment) DefineConstLocal(name string, val Object) Object {
	e.runtime.mu.Lock()
	defer e.runtime.mu.Unlock()

	if e.consts == nil {
		e.consts = make(map[string]bool)
	}
	e.store[name] = val
	e.consts[name] = true
	return val
}

func (e *Environment) Wait() {
	if e.runtime == nil {
		return
	}
	e.runtime.Wait()
}

func (e *Environment) SetCLIContext(scriptPath string, args []string) {
	e.cliArgs = args
	e.scriptPath = scriptPath
}

func (e *Environment) CLIArgs() []string {
	if e.outer != nil && len(e.cliArgs) == 0 && e.scriptPath == "" {
		return e.outer.CLIArgs()
	}
	return append([]string(nil), e.cliArgs...)
}

func (e *Environment) ScriptPath() string {
	if e.scriptPath == "" && e.outer != nil {
		return e.outer.ScriptPath()
	}
	return e.scriptPath
}

func (e *Environment) Export(name string) Object {
	e.runtime.mu.Lock()
	defer e.runtime.mu.Unlock()

	if _, ok := e.getUnlocked(name); !ok {
		return NewError(0, "cannot export undefined name '%s'", name)
	}
	if e.exports == nil {
		e.exports = make(map[string]bool)
	}
	e.exports[name] = true
	return NullObject()
}

func (e *Environment) ExportedHash() *Hash {
	e.runtime.mu.RLock()
	defer e.runtime.mu.RUnlock()

	values := make(map[string]Object, len(e.exports))
	for name := range e.exports {
		if value, ok := e.getUnlocked(name); ok {
			values[name] = value
		}
	}
	return HashFromObjects(values)
}

func (e *Environment) CachedModule(path string) (*Hash, bool) {
	e.runtime.mu.RLock()
	defer e.runtime.mu.RUnlock()

	if e.runtime.moduleCache == nil {
		return nil, false
	}
	module, ok := e.runtime.moduleCache[path]
	return module, ok
}

func (e *Environment) MarkModuleLoading(path string) bool {
	e.runtime.mu.Lock()
	defer e.runtime.mu.Unlock()

	if e.runtime.loadingModule == nil {
		e.runtime.loadingModule = make(map[string]bool)
	}
	if e.runtime.loadingModule[path] {
		return false
	}
	e.runtime.loadingModule[path] = true
	return true
}

func (e *Environment) FinishModuleLoading(path string, exports *Hash) {
	e.runtime.mu.Lock()
	defer e.runtime.mu.Unlock()

	delete(e.runtime.loadingModule, path)
	if e.runtime.moduleCache == nil {
		e.runtime.moduleCache = make(map[string]*Hash)
	}
	e.runtime.moduleCache[path] = exports
}

func (e *Environment) FailModuleLoading(path string) {
	e.runtime.mu.Lock()
	defer e.runtime.mu.Unlock()

	delete(e.runtime.loadingModule, path)
}

func (e *Environment) Call(fn Object, args []Object, line int) Object {
	e.runtime.mu.RLock()
	caller := e.runtime.caller
	e.runtime.mu.RUnlock()

	if caller == nil {
		return NewError(line, "no runtime caller registered")
	}

	return caller(e, fn, args, line)
}

func (e *Environment) Go(fn func()) {
	runtime := e.runtime
	if runtime == nil {
		go fn()
		return
	}
	runtime.Go(fn)
}

func (e *Environment) getUnlocked(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.getUnlocked(name)
	}
	return obj, ok
}

func (e *Environment) findVarUnlocked(name string) *Environment {
	if _, ok := e.store[name]; ok {
		return e
	}
	if e.outer != nil {
		return e.outer.findVarUnlocked(name)
	}
	return nil
}

var transientEnvPool = sync.Pool{
	New: func() any {
		return &Environment{
			store: make(map[string]Object, defaultLocalStoreCapacity),
		}
	},
}
