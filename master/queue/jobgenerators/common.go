package jobgenerators

import (
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/db/models"
	"testing_system/lib/customfields"
)

// used in IOI and ICPC generators
type generatorState int

const (
	compilationNotStarted generatorState = iota
	compilationStarted
	compilationFinished
)

func buildTestResult(job *invokerconn.Job, result *masterconn.InvokerJobResult) models.TestResult {
	testResult := models.TestResult{
		TestNumber: job.Test,
		Verdict:    result.Verdict,
		Points:     result.Points,
		Error:      result.Error,
	}

	if result.Statistics != nil {
		testResult.Time = roundTime(result.Statistics.Time)
		testResult.Memory = roundMemory(result.Statistics.Memory)
		testResult.WallTime = roundTime(result.Statistics.WallTime)
		testResult.ExitCode = &result.Statistics.ExitCode
	}

	return testResult
}

const (
	timeRoundFactor   = 1000
	memoryRoundFactor = 1024
)

func roundTime(time customfields.Time) *customfields.Time {
	return roundValue(time, timeRoundFactor)
}

func roundMemory(memory customfields.Memory) *customfields.Memory {
	return roundValue(memory, memoryRoundFactor)
}

func roundValue[T ~uint64](value T, roundFactor T) *T {
	for _ = range 2 {
		if value < roundFactor {
			return &value
		}
		value = (value + roundFactor - 1) / roundFactor * roundFactor
		roundFactor *= roundFactor
	}
	return &value
}
