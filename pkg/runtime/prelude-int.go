package runtime

import (
	"strconv"
)

var _ RuntimeValue = Int(0)

type Int int64

// Inspect implements runtime.RuntimeValue.
func (i Int) Inspect() string {
	return strconv.FormatInt(int64(i), 10)
}

// Lookup implements runtime.RuntimeValue.
func (i Int) Lookup(name string) RuntimeValue {
	return nil
}

// TypeConstantId implements runtime.RuntimeValue.
func (i Int) TypeConstantId() TypeId {
	return typeIdInt
}
