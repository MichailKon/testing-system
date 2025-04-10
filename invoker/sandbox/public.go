package sandbox

type ISandbox interface {
	Init() error
	Dir() string
	Run(config *ExecuteConfig) *RunResult
	Cleanup()
	Delete()
}
