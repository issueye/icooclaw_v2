package object

import "sync"

type Caller func(env *Environment, fn Object, args []Object, line int) Object

type Runtime struct {
	mu         sync.RWMutex
	wg         sync.WaitGroup
	caller     Caller
	cliArgs    []string
	scriptPath string
}

var (
	defaultCallerMu sync.RWMutex
	defaultCaller   Caller
)

func NewRuntime() *Runtime {
	defaultCallerMu.RLock()
	caller := defaultCaller
	defaultCallerMu.RUnlock()

	return &Runtime{caller: caller}
}

func RegisterDefaultCaller(caller Caller) {
	defaultCallerMu.Lock()
	defer defaultCallerMu.Unlock()
	defaultCaller = caller
}
