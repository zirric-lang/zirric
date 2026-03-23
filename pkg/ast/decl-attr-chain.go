package ast

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

type AttributeChain []*DeclAttrInstance

func MakeAttributeChain(instances ...*DeclAttrInstance) AttributeChain {
	return instances
}

// TokenLiteral implements Node
func (n AttributeChain) TokenLiteral() token.Token {
	return n[0].TokenLiteral()
}

func (n AttributeChain) EnumerateChildNodes(action func(child Node)) {
	for _, c := range n {
		action(c)
		c.EnumerateChildNodes(action)
	}
}
