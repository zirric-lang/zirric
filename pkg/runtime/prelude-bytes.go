package runtime

import (
	"strconv"
)

var _ RuntimeValue = Bytes([]byte{})

type Bytes []byte

// Inspect implements runtime.RuntimeValue.
func (i Bytes) Inspect() string {
	return strconv.Quote(string(i))
}

// Lookup implements runtime.RuntimeValue.
func (i Bytes) Lookup(name string) RuntimeValue {
	return nil
}

// TypeConstantId implements runtime.RuntimeValue.
func (i Bytes) TypeConstantId() TypeId {
	return typeIdBytes
}
