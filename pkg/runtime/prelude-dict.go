package runtime

var _ RuntimeValue = Dict{}

type Dict map[RuntimeValue]RuntimeValue

// Inspect implements RuntimeValue.
func (a Dict) Inspect() string {
	panic("unimplemented")
}

// Lookup implements RuntimeValue.
func (a Dict) Lookup(name string) RuntimeValue {
	switch name {
	case "length":
		return Int(len(a))
	case "keys":
		keys := make(Array, 0, len(a))
		for k := range a {
			keys = append(keys, k)
		}
		return keys
	}
	return nil
}

// TypeConstantId implements RuntimeValue.
func (a Dict) TypeConstantId() TypeId {
	return typeIdDict
}
