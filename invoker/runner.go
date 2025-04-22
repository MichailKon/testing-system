package invoker

import (
	"context"
	"sync"
	"testing_system/common"
	"testing_system/lib/logger"
)

type Runner struct {
	queue   chan []func()
	threads int
	funcs   []func()

	mutex sync.Mutex

	cancel context.CancelFunc
	ctx    context.Context

	activeSandboxes uint64
	sandboxesMutex  sync.Mutex
}

func newRunner(ts *common.TestingSystem) *Runner {
	r := &Runner{
		queue:           make(chan []func()),
		threads:         int(ts.Config.Invoker.Threads),
		activeSandboxes: ts.Config.Invoker.Sandboxes,
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())

	for i := range r.threads {
		ts.AddProcess(func() { r.run(i) })
	}
	return r
}

func (r *Runner) getLoadedFunc() func() {
	if len(r.funcs) == 0 {
		return nil
	}
	f := r.funcs[0]
	r.funcs = r.funcs[1:]
	return f
}

func (r *Runner) nextJob() func() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	f := r.getLoadedFunc()
	if f != nil {
		return f
	}
	select {
	case <-r.ctx.Done():
		return nil
	case r.funcs = <-r.queue:
		// If we have less threads than number of simultaneous processes that are required to run,
		// we run them in seperated goroutines
		for len(r.funcs) > r.threads {
			go r.funcs[len(r.funcs)-1]()
			r.funcs = r.funcs[:len(r.funcs)-1]
		}
	}
	return r.getLoadedFunc()
}

func (r *Runner) run(id int) {
	logger.Info("Started invoker process runner thread %d", id)
	for {
		select {
		case <-r.ctx.Done():
			logger.Info("Stopped invoker process runner thread %d", id)
			return
		default:
			// pass
		}

		f := r.getLoadedFunc()
		if f == nil {
			continue
		}
	}
}

func (r *Runner) stopSandbox() {
	r.sandboxesMutex.Lock()
	defer r.sandboxesMutex.Unlock()
	r.activeSandboxes--
	if r.activeSandboxes == 0 {
		r.cancel()
	}
}
