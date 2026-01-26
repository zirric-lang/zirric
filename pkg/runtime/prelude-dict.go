package runtime

var _ RuntimeValue = Dict{}

type Dict map[RuntimeValue]RuntimeValue

// Inspect implements RuntimeValue.
func (a Dict) Inspect() string {
	panic("unimplemented")
}

// Lookup implements RuntimeValue.
func (a Dict) Lookup(name string) RuntimeValue {
	panic("unimplemented")
}

// TypeConstantId implements RuntimeValue.
func (a Dict) TypeConstantId() TypeId {
	return typeIdDict
}
