package invoker

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing_system/common/connectors/invokerconn"
	"testing_system/invoker/sandbox"
	"testing_system/invoker/sandbox/isolate"
	"testing_system/invoker/sandbox/simple"
	"testing_system/lib/logger"
	"time"
)

type threadsExecutor[Value any] struct {
	values []Value
	closed bool

	threadsCount int
	name         string

	waitDuration   []time.Duration
	lastActiveTime []time.Time
	isActive       []bool
	mutex          sync.Mutex
	cond           *sync.Cond
}

func (e *threadsExecutor[Value]) add(v Value) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.closed {
		return fmt.Errorf("%s threads are stopped", e.name)
	}
	e.values = append(e.values, v)
	e.cond.Signal()
	return nil
}

func (e *threadsExecutor[Value]) stop() {
	logger.Info("Stopping invoker %s, finishing last added jobs", e.name)

	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.closed {
		logger.Warn("%s threads are already stopped, calling stop twice", e.name)
	}
	e.closed = true
	e.cond.Broadcast()
}

func (e *threadsExecutor[Value]) metrics() *invokerconn.StatusThreadsMetrics {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	status := &invokerconn.StatusThreadsMetrics{
		Count: e.threadsCount,
	}
	for i := range e.threadsCount {
		duration := e.waitDuration[i]
		if !e.isActive[i] {
			duration += time.Since(e.lastActiveTime[i])
		}
		status.TotalWaitDuration = append(status.TotalWaitDuration, duration)
	}
	return status
}

func (e *threadsExecutor[Value]) runThread(id int, valueProcessor func(Value)) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.lastActiveTime[id-1] = time.Now()
	logger.Info("Starting invoker %s thread id: %d", e.name, id)

receiverLoop:
	for {
		for len(e.values) == 0 {
			if e.closed {
				break receiverLoop
			}
			e.cond.Wait()
		}
		value := e.values[0]
		e.values = e.values[1:]

		e.waitDuration[id-1] += time.Since(e.lastActiveTime[id-1])
		e.isActive[id-1] = true
		e.mutex.Unlock()

		valueProcessor(value)

		e.mutex.Lock()
		e.isActive[id-1] = false
		e.lastActiveTime[id-1] = time.Now()
	}

	logger.Info("Finished invoker %s thread id: %d", e.name, id)
}

func (i *Invoker) newSandbox(id int) sandbox.ISandbox {
	homePath := filepath.Join(i.TS.Config.Invoker.SandboxHomePath, strconv.Itoa(id))
	err := os.RemoveAll(homePath)
	if err != nil {
		logger.Panic("Can not clean up sandbox home path for sandbox %d", id)
	}
	err = os.MkdirAll(homePath, 0777)
	if err != nil {
		logger.Panic("Can not create sandbox home for sandbox %d", id)
	}

	var s sandbox.ISandbox
	switch i.TS.Config.Invoker.SandboxType {
	case "simple":
		s, err = simple.NewSandbox(homePath)
	case "isolate":
		s, err = isolate.NewSandbox(id, homePath)
	default:
		logger.Panic("Unsupported sandbox type: %s", i.TS.Config.Invoker.SandboxType)
	}
	if err != nil {
		logger.Panic("Can not create %s sandbox %d, error: %v", i.TS.Config.Invoker.SandboxType, id, err)
	}
	return s
}

func (i *Invoker) initializeSandboxThreads() {
	i.SandboxThreads = &threadsExecutor[*Job]{
		values:         make([]*Job, 0),
		closed:         false,
		threadsCount:   i.TS.Config.Invoker.Sandboxes,
		name:           "sandbox",
		waitDuration:   make([]time.Duration, i.TS.Config.Invoker.Sandboxes),
		lastActiveTime: make([]time.Time, i.TS.Config.Invoker.Sandboxes),
		isActive:       make([]bool, i.TS.Config.Invoker.Sandboxes),
	}
	i.SandboxThreads.cond = sync.NewCond(&i.SandboxThreads.mutex)

	var threadsWaiter sync.WaitGroup
	threadsWaiter.Add(i.SandboxThreads.threadsCount)

	for id := 1; id <= i.SandboxThreads.threadsCount; id++ {
		s := i.newSandbox(id)

		i.TS.AddDefer(s.Delete)

		i.TS.AddProcess(func() {
			i.SandboxThreads.runThread(id, func(job *Job) {
				switch job.Type {
				case invokerconn.CompileJob:
					i.fullCompilationPipeline(s, job)
				case invokerconn.TestJob:
					i.fullTestingPipeline(s, job)
				default:
					logger.Panic("Unknown job type %d", job.Type)
				}
			})
			threadsWaiter.Done()
		})
	}
	i.TS.AddProcess(func() {
		<-i.TS.StopCtx.Done()
		i.SandboxThreads.stop()
	})

	i.TS.AddProcess(func() {
		threadsWaiter.Wait()
		i.RunnerThreads.stop()
	})
}

func (i *Invoker) initializeRunnerThreads() {
	i.RunnerThreads = &threadsExecutor[func()]{
		values:         make([]func(), 0),
		closed:         false,
		threadsCount:   i.TS.Config.Invoker.Threads,
		name:           "process execution",
		waitDuration:   make([]time.Duration, i.TS.Config.Invoker.Threads),
		lastActiveTime: make([]time.Time, i.TS.Config.Invoker.Threads),
		isActive:       make([]bool, i.TS.Config.Invoker.Threads),
	}
	i.RunnerThreads.cond = sync.NewCond(&i.RunnerThreads.mutex)
	for id := 1; id <= i.TS.Config.Invoker.Threads; id++ {
		i.TS.AddProcess(func() {
			i.RunnerThreads.runThread(id, func(f func()) {
				f()
			})
		})
	}
}
