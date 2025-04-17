//go:generate go run golang.org/x/tools/cmd/stringer@latest -type=Type

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
)
