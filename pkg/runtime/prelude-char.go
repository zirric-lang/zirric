package runtime

import "strconv"

var _ RuntimeValue = Char('a')

type Char rune

// Inspect implements RuntimeValue.
func (b Char) Inspect() string {
	return strconv.QuoteRune(rune(b))
}

// Lookup implements RuntimeValue.
func (b Char) Lookup(name string) RuntimeValue {
	return nil
}

// TypeConstantId implements RuntimeValue.
func (b Char) TypeConstantId() TypeId {
	return typeIdChar
}
