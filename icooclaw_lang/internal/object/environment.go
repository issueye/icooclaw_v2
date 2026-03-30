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
	e.store[name] = val
	return val
}

func (e *Environment) SetConst(name string, val Object) Object {
	e.store[name] = val
	e.consts[name] = true
	return val
}
