package object

type Environment struct {
	store  map[string]Object
	outer  *Environment
	consts map[string]bool
}

func NewEnvironment() *Environment {
	return &Environment{
		store:  make(map[string]Object),
		consts: make(map[string]bool),
	}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	if e.consts[name] {
		return NewError(0, "cannot reassign to constant '%s'", name)
	}
	if _, ok := e.store[name]; ok {
		e.store[name] = val
		return val
	}
	if e.outer != nil {
		if outerEnv := e.findVar(name); outerEnv != nil {
			if outerEnv.consts[name] {
				return NewError(0, "cannot reassign to constant '%s'", name)
			}
			outerEnv.store[name] = val
			return val
		}
	}
	e.store[name] = val
	return val
}

func (e *Environment) findVar(name string) *Environment {
	if _, ok := e.store[name]; ok {
		return e
	}
	if e.outer != nil {
		return e.outer.findVar(name)
	}
	return nil
}

func (e *Environment) SetConst(name string, val Object) Object {
	e.store[name] = val
	e.consts[name] = true
	return val
}
