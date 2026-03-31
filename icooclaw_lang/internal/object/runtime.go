package object

import "sync"

type Caller func(env *Environment, fn Object, args []Object, line int) Object

type Runtime struct {
	mu            sync.RWMutex
	wg            sync.WaitGroup
	caller        Caller
	moduleCache   map[string]*Hash
	loadingModule map[string]bool
}

var (
	defaultCallerMu sync.RWMutex
	defaultCaller   Caller
)

func NewRuntime() *Runtime {
	defaultCallerMu.RLock()
	caller := defaultCaller
	defaultCallerMu.RUnlock()

	return &Runtime{
		caller:        caller,
		moduleCache:   make(map[string]*Hash),
		loadingModule: make(map[string]bool),
	}
}

func RegisterDefaultCaller(caller Caller) {
	defaultCallerMu.Lock()
	defer defaultCallerMu.Unlock()
	defaultCaller = caller
}
