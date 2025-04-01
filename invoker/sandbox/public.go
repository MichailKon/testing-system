package sandbox

type ISandbox interface {
	Init()
	Dir() string
	Run(process string, args []string, config *RunConfig) *RunResult
	Cleanup()
	Delete()
}
