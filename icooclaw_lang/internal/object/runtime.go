package object

import (
	"os"
	"runtime"
	"strconv"
	"sync"
)

type Caller func(env *Environment, fn Object, args []Object, line int) Object

type Runtime struct {
	mu            sync.RWMutex
	wg            sync.WaitGroup
	caller        Caller
	moduleCache   map[string]*Hash
	loadingModule map[string]bool

	taskMu         sync.Mutex
	taskCond       *sync.Cond
	taskQueue      []func()
	taskRunning    bool
	taskStopping   bool
	workerWG       sync.WaitGroup
	maxConcurrency int
	workerCount    int
}

type RuntimeStats struct {
	MaxConcurrency int
	WorkerCount    int
	QueueLength    int
	IsRunning      bool
	IsStopping     bool
}

const runtimeMaxGoroutinesEnv = "ICLANG_MAX_GOROUTINES"

var (
	defaultCallerMu sync.RWMutex
	defaultCaller   Caller
)

func NewRuntime() *Runtime {
	defaultCallerMu.RLock()
	caller := defaultCaller
	defaultCallerMu.RUnlock()

	rt := &Runtime{
		caller:         caller,
		maxConcurrency: resolveDefaultConcurrency(),
	}
	rt.taskCond = sync.NewCond(&rt.taskMu)
	return rt
}

func RegisterDefaultCaller(caller Caller) {
	defaultCallerMu.Lock()
	defer defaultCallerMu.Unlock()
	defaultCaller = caller
}

func (r *Runtime) SetMaxConcurrency(size int) {
	if size <= 0 {
		size = 1
	}

	r.taskMu.Lock()
	defer r.taskMu.Unlock()
	r.maxConcurrency = size
}

func (r *Runtime) MaxConcurrency() int {
	r.taskMu.Lock()
	defer r.taskMu.Unlock()
	return r.maxConcurrency
}

func (r *Runtime) Go(fn func()) {
	if fn == nil {
		return
	}

	r.wg.Add(1)

	r.taskMu.Lock()
	r.startWorkersLocked()
	r.taskQueue = append(r.taskQueue, func() {
		defer r.wg.Done()
		fn()
	})
	r.taskCond.Signal()
	r.taskMu.Unlock()
}

func (r *Runtime) Wait() {
	r.wg.Wait()
	r.stopWorkers()
}

func (r *Runtime) startWorkersLocked() {
	if r.taskRunning {
		return
	}

	r.taskRunning = true
	r.taskStopping = false

	workerCount := r.maxConcurrency
	if workerCount <= 0 {
		workerCount = 1
	}
	r.workerCount = workerCount

	for i := 0; i < workerCount; i++ {
		r.workerWG.Add(1)
		go r.workerLoop()
	}
}

func (r *Runtime) workerLoop() {
	defer r.workerWG.Done()

	for {
		r.taskMu.Lock()
		for len(r.taskQueue) == 0 && !r.taskStopping {
			r.taskCond.Wait()
		}
		if len(r.taskQueue) == 0 && r.taskStopping {
			r.taskMu.Unlock()
			return
		}

		task := r.taskQueue[0]
		r.taskQueue[0] = nil
		r.taskQueue = r.taskQueue[1:]
		r.taskMu.Unlock()

		task()
	}
}

func (r *Runtime) stopWorkers() {
	r.taskMu.Lock()
	if !r.taskRunning {
		r.taskMu.Unlock()
		return
	}
	r.taskStopping = true
	r.taskCond.Broadcast()
	r.taskMu.Unlock()

	r.workerWG.Wait()

	r.taskMu.Lock()
	r.taskQueue = nil
	r.taskRunning = false
	r.taskStopping = false
	r.workerCount = 0
	r.taskMu.Unlock()
}

func (r *Runtime) Stats() RuntimeStats {
	r.taskMu.Lock()
	defer r.taskMu.Unlock()

	return RuntimeStats{
		MaxConcurrency: r.maxConcurrency,
		WorkerCount:    r.workerCount,
		QueueLength:    len(r.taskQueue),
		IsRunning:      r.taskRunning,
		IsStopping:     r.taskStopping,
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func resolveDefaultConcurrency() int {
	if value := os.Getenv(runtimeMaxGoroutinesEnv); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return parsed
		}
	}
	return max(1, runtime.NumCPU())
}
