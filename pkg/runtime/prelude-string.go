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
	return nil
}

// TypeConstantId implements runtime.RuntimeValue.
func (i String) TypeConstantId() TypeId {
	return typeIdString
}
