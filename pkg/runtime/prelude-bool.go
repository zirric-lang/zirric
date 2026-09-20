package runtime

var _ RuntimeValue = Bool(true)

type Bool bool

// Inspect implements RuntimeValue.
func (b Bool) Inspect() string {
	if b {
		return "true"
	} else {
		return "false"
	}
}

// Lookup implements RuntimeValue.
func (b Bool) Lookup(name string) RuntimeValue {
	return nil
}

// TypeConstantId implements RuntimeValue.
func (b Bool) TypeConstantId() TypeId {
	return typeIdBool
}
