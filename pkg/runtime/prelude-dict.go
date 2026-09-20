package runtime

import "strings"

var _ RuntimeValue = Dict{}

type Dict map[RuntimeValue]RuntimeValue

// Inspect implements RuntimeValue.
func (a Dict) Inspect() string {
	if len(a) == 0 {
		return "[:]"
	}
	var b strings.Builder
	b.WriteByte('[')
	first := true
	for k, v := range a {
		if !first {
			b.WriteString(", ")
		}
		first = false
		b.WriteString(k.Inspect())
		b.WriteString(": ")
		b.WriteString(v.Inspect())
	}
	b.WriteByte(']')
	return b.String()
}

// Lookup implements RuntimeValue.
func (a Dict) Lookup(name string) RuntimeValue {
	switch name {
	case "keys":
		return MakeNativeFunc("keys", 0, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			keys := make(Array, 0, len(a))
			for k := range a {
				keys = append(keys, k)
			}
			return keys, nil
		})
	}
	return nil
}

// TypeConstantId implements RuntimeValue.
func (a Dict) TypeConstantId() TypeId {
	return typeIdDict
}
