package runtime

import (
	"strconv"
)

var _ RuntimeValue = String("")

type String string

// Inspect implements runtime.RuntimeValue.
func (i String) Inspect() string {
	return strconv.Quote(string(i))
}

// Lookup implements runtime.RuntimeValue.
func (i String) Lookup(name string) RuntimeValue {
	switch name {
	case "chars":
		// Decodes the whole string into an Array of Char in one O(n) pass.
		return MakeNativeFunc("chars", 0, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			chars := make(Array, 0, len(i))
			for _, r := range string(i) {
				chars = append(chars, Char(r))
			}
			return chars, nil
		})
	}
	return nil
}

// TypeConstantId implements runtime.RuntimeValue.
func (i String) TypeConstantId() TypeId {
	return typeIdString
}
