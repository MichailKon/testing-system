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
func roundTime(time customfields.Time) *customfields.Time {
	roundFactor := customfields.Time(1000)
	if time < roundFactor {
		return &time
	}
	time = (time + roundFactor - 1) / roundFactor * roundFactor
	roundFactor *= roundFactor
	if time < roundFactor {
		return &time
	}
	time = (time + roundFactor - 1) / roundFactor * roundFactor
	return &time
}

func roundMemory(memory customfields.Memory) *customfields.Memory {
	memoryFactor := customfields.Memory(1024)
	if memory < memoryFactor {
		return &memory
	}
	memory = (memory + memoryFactor - 1) / memoryFactor * memoryFactor
	memoryFactor *= memoryFactor
	if memory < memoryFactor {
		return &memory
	}
	memory = (memory + memoryFactor - 1) / memoryFactor * memoryFactor
	return &memory
}
