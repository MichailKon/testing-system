//go:generate go run golang.org/x/tools/cmd/stringer@latest -type=Type
//go:generate go run golang.org/x/tools/cmd/stringer@latest -type=DataType

package resource

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
	// Don't forget to add a new type to storage/filesystem/resource_info.go
)

type DataType int

const (
	UnknownDataType DataType = iota
	Problem
	Submission
	// Will be increased
)
