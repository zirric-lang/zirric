package ast

import (
	"strings"

	"code.knabel.dev/zirric-lang/zirric/token"
)

// Represents a static reference or a fully qualified Identifier.
//
// Invariant: must always contain at least one identifier.
type StaticReference []Identifier

func (ref StaticReference) Name() Identifier {
	return ref[len(ref)-1]
}

func (ref StaticReference) TokenLiteral() token.Token {
	return ref[0].Token
}

func (ref StaticReference) String() string {
	if len(ref) == 1 {
		return ref[0].Value
	}

	length := len(ref) - 1
	for _, ident := range ref {
		length += len(ident.Value)
	}

	var result strings.Builder
	result.Grow(length)
	result.WriteString(ref[0].Value)
	for _, ident := range ref[1:] {
		result.WriteString(".")
		result.WriteString(ident.Value)
	}
	return result.String()
}

func (ref StaticReference) EnumerateChildNodes(action func(child Node)) {
	for _, id := range ref {
		action(id)
	}
}
