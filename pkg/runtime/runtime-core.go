package runtime

type TypeId uint32

type RuntimeValue interface {
	TypeConstantId() TypeId
	// for printing
	Inspect() string
	// Member access. If nil returned, this produces a runtime error.
	Lookup(name string) RuntimeValue
}

type CallableRuntimeValue interface {
	RuntimeValue
	Arity() int
}
