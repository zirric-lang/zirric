package runtime

var _ RuntimeValue = Void{}

type Void struct{}

// Inspect implements RuntimeValue.
func (n Void) Inspect() string {
	return "void"
}

// Lookup implements RuntimeValue.
func (n Void) Lookup(name string) RuntimeValue {
	return nil
}

// TypeConstantId implements RuntimeValue.
func (n Void) TypeConstantId() TypeId {
	return typeIdVoid
}
