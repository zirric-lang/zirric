package runtime

import (
	"fmt"
)

var _ RuntimeValue = Float(0.0)

type Float float64

// Inspect implements runtime.RuntimeValue.
func (i Float) Inspect() string {
	return fmt.Sprintf("%f", float64(i))
}

// Lookup implements runtime.RuntimeValue.
func (i Float) Lookup(name string) RuntimeValue {
	return nil
}

// TypeConstantId implements runtime.RuntimeValue.
func (i Float) TypeConstantId() TypeId {
	return typeIdFloat
}
