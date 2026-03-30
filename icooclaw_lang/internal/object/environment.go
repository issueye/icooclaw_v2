package object

type Environment struct {
	store   map[string]Object
	outer   *Environment
	consts  map[string]bool
	runtime *Runtime
}

func NewEnvironment() *Environment {
	return &Environment{
		store:   make(map[string]Object),
		consts:  make(map[string]bool),
		runtime: NewRuntime(),
	}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	env.runtime = outer.runtime
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
