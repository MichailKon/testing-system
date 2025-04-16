package jobgenerators

// used in IOI and ICPC generators
type generatorState int

const (
	compilationNotStarted generatorState = iota
	compilationStarted
	compilationFinished
)
