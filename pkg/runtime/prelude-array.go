package runtime

var _ RuntimeValue = Array{}

type Array []RuntimeValue

// Inspect implements RuntimeValue.
func (a Array) Inspect() string {
	panic("unimplemented")
}

// Lookup implements RuntimeValue.
func (a Array) Lookup(name string) RuntimeValue {
	switch name {
	case "length":
		return Int(len(a))
	}
	return nil
}

// TypeConstantId implements RuntimeValue.
func (a Array) TypeConstantId() TypeId {
	return typeIdArray
}
