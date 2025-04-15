package invoker

import "testing_system/lib/logger"

func (i *Invoker) startRunners() {
	// TODO: specify cpu core for extra isolation
	for id := range i.TS.Config.Invoker.Threads {
		i.TS.AddProcess(func() { i.runRunnerThread(id) })
	}
}

func (i *Invoker) runRunnerThread(id uint64) {
	logger.Info("Started invoker thread %d", id)
	for {
		select {
		case <-i.TS.StopCtx.Done():
			logger.Info("Stopped invoker thread %d", id)
			break
		case f := <-i.RunQueue:
			f()
		}
	}
}
