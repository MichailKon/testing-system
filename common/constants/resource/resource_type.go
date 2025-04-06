package resource

//go:generate stringer -type=Type

type Type int

const (
	SourceCode Type = iota
	CompiledBinary
	CompileOutput
	TestInput
	TestAnswer
	TestOutput
	TestStderr
	Checker
	CheckerOutput
	Interactor
	// Will be increased
)
