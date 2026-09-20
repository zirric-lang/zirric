package runtime

import (
	"encoding/hex"
)

var _ RuntimeValue = Binary([]byte{})

type Binary []byte

// Inspect implements runtime.RuntimeValue.
// Binary is raw bytes, not text, so it's rendered as hex (two digits per
// byte, e.g. "deadbeef") rather than reinterpreted as a (possibly invalid)
// UTF-8 string.
func (i Binary) Inspect() string {
	return hex.EncodeToString(i)
}

// Lookup implements runtime.RuntimeValue.
func (i Binary) Lookup(name string) RuntimeValue {
	return nil
}

// TypeConstantId implements runtime.RuntimeValue.
func (i Binary) TypeConstantId() TypeId {
	return typeIdBinary
}
