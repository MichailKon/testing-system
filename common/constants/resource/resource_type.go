package resource

//go:generate stringer -type=Type
//go:generate stringer -type=DataType

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

type DataType int

const (
	UnknownDataType DataType = iota
	Problem
	Submission
	// Will be increased
)
