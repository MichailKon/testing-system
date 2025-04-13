package invoker

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/invoker/sandbox"
	"testing_system/invoker/sandbox/isolate"
	"testing_system/invoker/sandbox/simple"
	"testing_system/lib/logger"
)

type JobExecutor struct {
	ID      uint64
	Sandbox sandbox.ISandbox
}

func NewJobExecutor(ts *common.TestingSystem, id uint64) *JobExecutor {
	e := &JobExecutor{
		ID: id,
	}
	switch ts.Config.Invoker.SandboxType {
	case "simple":
		var err error
		e.Sandbox, err = simple.NewSandbox(filepath.Join(ts.Config.Invoker.SandboxHomePath, strconv.FormatUint(id, 10)))
		if err != nil {
			logger.Panic("Can not create simple sandbox %d, error: %v", id, err)
		}
	case "isolate":
		var err error
		e.Sandbox, err = isolate.NewSandbox(e.ID, ts.Config.Invoker.SandboxHomePath)
		if err != nil {
			logger.Panic("Can not create isolate sandbox %d, error: %v", id, err)
		}
	default:
		logger.Panic("Unsupported sandbox type: %s", ts.Config.Invoker.SandboxType)
	}
	return e
}

func (t *JobExecutor) Delete() {
	t.Sandbox.Delete()
}

func (i *Invoker) RunJobExecutorThread(tester *JobExecutor) {
	logger.Info("Starting sandbox %d", tester.ID)
	for {
		select {
		case <-i.TS.StopCtx.Done():
			logger.Info("Stopped sandbox %d", tester.ID)
			break
		case job := <-i.JobQueue:
			switch job.Type {
			case invokerconn.CompileJob:
				i.Compile(tester, job)
			case invokerconn.TestJob:
				i.Test(tester, job)
			default:
				logger.Panic("Unknown job type %d", job.Type)
			}
		}
	}
}

func (t *JobExecutor) CopyFileToSandbox(src string, dst string, perm os.FileMode) error {
	srcReader, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcReader.Close()
	dstWriter, err := os.OpenFile(filepath.Join(t.Sandbox.Dir(), dst), os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	defer dstWriter.Close()
	_, err = io.Copy(dstWriter, srcReader)
	return nil
}

func (i *Invoker) limitedReader(r io.Reader) io.Reader {
	if i.TS.Config.Invoker.SaveOutputHead == nil {
		return r
	} else {
		return io.LimitReader(r, int64(*i.TS.Config.Invoker.SaveOutputHead))
	}
}
