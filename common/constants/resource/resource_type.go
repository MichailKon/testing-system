package resource

//go:generate stringer -type=Type

type Type int

const (
	SourceCode Type = iota
	CompiledBinary
	CompileOutput
	Test
	Checker
	Interactor
	// Will be increased
)
