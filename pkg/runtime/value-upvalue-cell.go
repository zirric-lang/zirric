package runtime

// UpvalueCell is a heap-allocated cell holding a mutable RuntimeValue.
// Used for var bindings captured by closures. The VM always dereferences
// cells transparently; user code should never observe an UpvalueCell
// directly. All RuntimeValue methods panic to surface compiler/VM bugs.
type UpvalueCell struct {
	Value RuntimeValue
}

func (c *UpvalueCell) Inspect() string {
	panic("internal error: UpvalueCell leaked into user-visible code")
}

func (c *UpvalueCell) Lookup(_ string) RuntimeValue {
	panic("internal error: UpvalueCell leaked into user-visible code")
}

func (c *UpvalueCell) TypeConstantId() TypeId {
	panic("internal error: UpvalueCell leaked into user-visible code")
}
