package object

import "sync"

type Runtime struct {
	mu sync.RWMutex
	wg sync.WaitGroup
}

func NewRuntime() *Runtime {
	return &Runtime{}
}
