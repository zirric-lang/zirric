package runtime

import (
	"strconv"
)

var _ RuntimeValue = Byte(0)

type Byte byte

// Inspect implements runtime.RuntimeValue.
func (i Byte) Inspect() string {
	return strconv.FormatInt(int64(i), 16)
}

// Lookup implements runtime.RuntimeValue.
func (i Byte) Lookup(name string) RuntimeValue {
	return nil
}

// TypeConstantId implements runtime.RuntimeValue.
func (i Byte) TypeConstantId() TypeId {
	return typeIdByte
}
