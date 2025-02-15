package invoker

import (
	"io"
	"os"
	"path/filepath"
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/invoker/sandbox"
	"testing_system/lib/logger"
)

type JobExecutor struct {
	ID      uint64
	Sandbox sandbox.ISandbox
}

func NewJobExecutor(ts *common.TestingSystem, id uint64) *JobExecutor {
	// TODO: Add sandbox
	return &JobExecutor{
		ID: id,
	}
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
			case invokerconn.Compile:
				i.Compile(tester, job)
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

func (t *JobExecutor) CreateSandboxFile(name string, perm os.FileMode, data []byte) error {
	return os.WriteFile(filepath.Join(t.Sandbox.Dir(), name), data, perm)
}
