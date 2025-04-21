package invoker

import (
	"fmt"
	"slices"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/invoker/sandbox"
	"testing_system/lib/logger"
)

type Job struct {
	invokerconn.Job

	Submission *models.Submission `json:"-"`
	Problem    *models.Problem    `json:"-"`

	Defers []func() `json:"-"`
}

func (j *Job) deferFunc() {
	slices.Reverse(j.Defers)
	for _, f := range j.Defers {
		f()
	}
	j.Defers = nil
}

func (i *Invoker) failJob(j *Job, errf string, args ...interface{}) {
	i.Mutex.Lock()
	delete(i.ActiveJobs, j.ID)
	i.Mutex.Unlock()

	request := &masterconn.InvokerJobResult{
		JobID:         j.ID,
		Verdict:       verdict.CF,
		Error:         fmt.Sprintf(errf, args...),
		InvokerStatus: i.getStatus(),
	}
	err := i.TS.MasterConn.SendInvokerJobResult(request)
	if err != nil {
		logger.Error("Can not send job %s result, error: %s", j.ID, err.Error())
	}
}

func (i *Invoker) successJob(j *Job, runResult *sandbox.RunResult) {
	i.Mutex.Lock()
	delete(i.ActiveJobs, j.ID)
	i.Mutex.Unlock()

	request := &masterconn.InvokerJobResult{
		JobID:         j.ID,
		Verdict:       runResult.Verdict,
		Points:        runResult.Points,
		Statistics:    runResult.Statistics,
		InvokerStatus: i.getStatus(),
	}
	err := i.TS.MasterConn.SendInvokerJobResult(request)
	if err != nil {
		logger.Error("Can not send job %s result, error: %s", j.ID, err.Error())
	}
}
