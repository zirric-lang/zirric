package runtime

var _ RuntimeValue = Array{}

type Array []RuntimeValue

// Inspect implements RuntimeValue.
func (a Array) Inspect() string {
	panic("unimplemented")
}

// Lookup implements RuntimeValue.
func (a Array) Lookup(name string) RuntimeValue {
	panic("unimplemented")
}

// TypeConstantId implements RuntimeValue.
func (a Array) TypeConstantId() TypeId {
	return typeIdArray
}
