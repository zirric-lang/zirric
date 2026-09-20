package runtime

import "strings"

var _ RuntimeValue = Array{}

type Array []RuntimeValue

// Inspect implements RuntimeValue.
func (a Array) Inspect() string {
	var b strings.Builder
	b.WriteByte('[')
	for i, v := range a {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(v.Inspect())
	}
	b.WriteByte(']')
	return b.String()
}

// Lookup implements RuntimeValue.
func (a Array) Lookup(name string) RuntimeValue {
	return nil
}

// TypeConstantId implements RuntimeValue.
func (a Array) TypeConstantId() TypeId {
	return typeIdArray
}
