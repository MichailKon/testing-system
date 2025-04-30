package invoker

import (
	"fmt"
	"io"
	"slices"
	"sync"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/invoker/compiler"
	"testing_system/invoker/sandbox"
	"testing_system/lib/logger"
	"time"
)

type JobPipelineState struct {
	job     *Job
	invoker *Invoker
	sandbox sandbox.ISandbox

	executeWaitGroup sync.WaitGroup

	compile *pipelineCompileData
	test    *pipelineTestData

	metrics *masterconn.InvokerJobMetrics

	loggerData string

	defers   []func()
	finished bool
}

type pipelineCompileData struct {
	language *compiler.Language
	config   *sandbox.ExecuteConfig
	result   *sandbox.RunResult

	sourceName    string
	messageReader io.Reader
}

type pipelineTestData struct {
	runConfig *sandbox.ExecuteConfig
	runResult *sandbox.RunResult

	checkConfig *sandbox.ExecuteConfig
	checkResult *sandbox.RunResult

	checkerOutputReader io.Reader
	hasResources        bool
}

func (i *Invoker) newPipeline(sandbox sandbox.ISandbox, job *Job) *JobPipelineState {
	s := &JobPipelineState{
		sandbox: sandbox,
		invoker: i,
		job:     job,
		metrics: new(masterconn.InvokerJobMetrics),
	}

	createTime := time.Now()
	s.metrics.TestingWaitDuration = createTime.Sub(job.createTime)
	s.defers = append(s.defers, func() {
		s.metrics.TotalSandboxOccupation = time.Since(createTime)
	})

	s.defers = append(s.defers, job.deferFunc)
	return s
}

func (s *JobPipelineState) initSandbox() error {
	err := s.sandbox.Init()
	if err != nil {
		return fmt.Errorf("can not initialize sandbox, error: %v", err)
	}
	s.defers = append(s.defers, s.sandbox.Cleanup)
	return nil
}

func (s *JobPipelineState) runProcess(f func()) {
	queueEnter := time.Now()
	s.invoker.RunQueue <- func() {
		processStart := time.Now()
		s.metrics.ExecutionWaitDuration += processStart.Sub(queueEnter)
		f()
		s.metrics.ExecutionDuration += time.Since(processStart)
	}
}

func (s *JobPipelineState) checkFinish() {
	if !s.finished {
		logger.Panic("Job pipeline for %s was not finished", s.loggerData)
	}
}

func (s *JobPipelineState) finish() {
	if s.finished {
		logger.Panic("Job pipeline for %s is finished twice", s.loggerData)
	}
	slices.Reverse(s.defers)
	for _, f := range s.defers {
		f()
	}
	s.finished = true

	s.invoker.Mutex.Lock()
	defer s.invoker.Mutex.Unlock()
	delete(s.invoker.ActiveJobs, s.job.ID)
}

func (s *JobPipelineState) failJob(errf string, args ...interface{}) {
	s.finish()
	request := &masterconn.InvokerJobResult{
		Job:           &s.job.Job,
		Verdict:       verdict.CF,
		Error:         fmt.Sprintf(errf, args...),
		InvokerStatus: s.invoker.getStatus(),
		Metrics:       s.metrics,
	}
	go s.uploadJobResult(request)
}

func (s *JobPipelineState) successJob(runResult *sandbox.RunResult) {
	s.finish()
	request := &masterconn.InvokerJobResult{
		Job:           &s.job.Job,
		Verdict:       runResult.Verdict,
		Points:        runResult.Points,
		Statistics:    runResult.Statistics,
		InvokerStatus: s.invoker.getStatus(),
		Metrics:       s.metrics,
	}
	s.invoker.TS.Go(func() { s.uploadJobResult(request) })
}

func (s *JobPipelineState) uploadJobResult(request *masterconn.InvokerJobResult) {
	err := s.invoker.TS.MasterConn.SendInvokerJobResult(request)
	if err != nil {
		logger.Error("Can not send job %s result, error: %s", s.job.ID, err.Error())
	}
}
