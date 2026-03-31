package object

type Environment struct {
	store      map[string]Object
	outer      *Environment
	consts     map[string]bool
	exports    map[string]bool
	cliArgs    []string
	scriptPath string
	runtime    *Runtime
}

func NewEnvironment() *Environment {
	return &Environment{
		store:   make(map[string]Object),
		consts:  make(map[string]bool),
		exports: make(map[string]bool),
		runtime: NewRuntime(),
	}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	env.runtime = outer.runtime
	env.cliArgs = append([]string(nil), outer.cliArgs...)
	env.scriptPath = outer.scriptPath
	return env
}

func NewDetachedEnvironment(proto *Environment) *Environment {
	env := NewEnvironment()
	env.runtime = proto.runtime
	env.cliArgs = append([]string(nil), proto.cliArgs...)
	env.scriptPath = proto.scriptPath
	return env
}

func (e *Environment) Get(name string) (Object, bool) {
	e.runtime.mu.RLock()
	defer e.runtime.mu.RUnlock()

	return e.getUnlocked(name)
}

func (e *Environment) Set(name string, val Object) Object {
	e.runtime.mu.Lock()
	defer e.runtime.mu.Unlock()

	if target := e.findVarUnlocked(name); target != nil {
		if target.consts[name] {
			return NewError(0, "cannot reassign to constant '%s'", name)
		}
		target.store[name] = val
		return val
	}

	if e.consts[name] {
		return NewError(0, "cannot reassign to constant '%s'", name)
	}
	e.store[name] = val
	return val
}

func (e *Environment) findVar(name string) *Environment {
	e.runtime.mu.RLock()
	defer e.runtime.mu.RUnlock()

	return e.findVarUnlocked(name)
}

func (e *Environment) SetConst(name string, val Object) Object {
	e.runtime.mu.Lock()
	defer e.runtime.mu.Unlock()

	e.store[name] = val
	e.consts[name] = true
	return val
}

func (e *Environment) Wait() {
	e.runtime.wg.Wait()
}

func (e *Environment) SetCLIContext(scriptPath string, args []string) {
	e.cliArgs = append([]string(nil), args...)
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
	e.exports[name] = true
	return &Null{}
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

	module, ok := e.runtime.moduleCache[path]
	return module, ok
}

func (e *Environment) MarkModuleLoading(path string) bool {
	e.runtime.mu.Lock()
	defer e.runtime.mu.Unlock()

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
	e.runtime.wg.Add(1)
	go func() {
		defer e.runtime.wg.Done()
		fn()
	}()
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
