package sandbox

type ISandbox interface {
	Init()
	Dir() string
	Run(config *ExecuteConfig) *RunResult
	Cleanup()
	Delete()
}
