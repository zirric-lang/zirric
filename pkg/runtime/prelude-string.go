package runtime

import (
	"strconv"
	"unicode/utf8"
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
	case "length":
		// Character count, not byte length, to match index-by-character-position.
		return Int(utf8.RuneCountInString(string(i)))
	case "chars":
		// Decodes the whole string into an Array of Char in one O(n) pass.
		chars := make(Array, 0, len(i))
		for _, r := range string(i) {
			chars = append(chars, Char(r))
		}
		return chars
	}
	return nil
}

// TypeConstantId implements runtime.RuntimeValue.
func (i String) TypeConstantId() TypeId {
	return typeIdString
}
