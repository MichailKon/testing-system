package invoker

import "testing_system/lib/logger"

type Runner struct {
	ID uint64
	// TODO: specify cpu core for extra isolation
}

func (i *Invoker) StartRunners() {
	// TODO: specify cpu core for extra isolation
	for id := range i.TS.Config.Invoker.Threads {
		r := Runner{ID: id}
		i.TS.AddProcess(func() { i.RunRunnerThread(r) })
	}
}

func (i *Invoker) RunRunnerThread(runner Runner) {
	logger.Info("Started invoker thread %d", runner.ID)
	for {
		select {
		case <-i.TS.StopCtx.Done():
			logger.Info("Stopped invoker thread %d", runner.ID)
			break
		case f := <-i.RunQueue:
			f()
		}
	}
}
