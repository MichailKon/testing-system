package invoker

import (
	"path/filepath"
	"strconv"
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/invoker/sandbox"
	"testing_system/invoker/sandbox/isolate"
	"testing_system/invoker/sandbox/simple"
	"testing_system/lib/logger"
)

func newSandbox(ts *common.TestingSystem, id uint64) sandbox.ISandbox {
	var s sandbox.ISandbox
	var err error
	switch ts.Config.Invoker.SandboxType {
	case "simple":
		s, err = simple.NewSandbox(filepath.Join(ts.Config.Invoker.SandboxHomePath, strconv.FormatUint(id, 10)))
	case "isolate":
		s, err = isolate.NewSandbox(id, ts.Config.Invoker.SandboxHomePath)

	default:
		logger.Panic("Unsupported sandbox type: %s", ts.Config.Invoker.SandboxType)
	}
	if err != nil {
		logger.Panic("Can not create %s sandbox %d, error: %v", ts.Config.Invoker.SandboxType, id, err)
	}
	return s
}

func (i *Invoker) runSandboxThread(sandbox sandbox.ISandbox, id uint64) {
	logger.Info("Starting sandbox %d", id)
	for {
		select {
		case <-i.TS.StopCtx.Done():
			logger.Info("Stopped sandbox %d", id)
			break
		case job := <-i.JobQueue:
			switch job.Type {
			case invokerconn.CompileJob:
				i.fullCompilationPipeline(sandbox, job)
			case invokerconn.TestJob:
				i.fullTestingPipeline(sandbox, job)
			case invokerconn.InteractiveSolutionJob:
				i.fullInteractiveSolutionPipeline(sandbox, job)
			case invokerconn.InteractiveInteractorJob:
				i.fullInteractiveInteractorPipeline(sandbox, job)
			default:
				logger.Panic("Unknown job type %d", job.Type)
			}
		}
	}
}
