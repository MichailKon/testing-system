package invoker

import (
	"fmt"
	"io"
	"slices"
	"sync"
	"testing_system/invoker/compiler"
	"testing_system/invoker/sandbox"
)

type JobPipelineState struct {
	job     *Job
	invoker *Invoker
	sandbox sandbox.ISandbox

	executeWaitGroup sync.WaitGroup

	compile *pipelineCompileData
	test    *pipelineTestData

	loggerData string

	defers []func()
}

func (s *JobPipelineState) deferFunc() {
	slices.Reverse(s.defers)
	for _, f := range s.defers {
		f()
	}
	s.defers = nil
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

func (s *JobPipelineState) initSandbox() error {
	err := s.sandbox.Init()
	if err != nil {
		return fmt.Errorf("can not initialize sandbox, error: %v", err)
	}
	s.defers = append(s.defers, s.sandbox.Cleanup)
	return nil
}
