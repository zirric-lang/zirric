package ast

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

type DeclAttrInstance struct {
	Token     token.Token
	Reference StaticReference
	Arguments []Expr
}

// TokenLiteral implements Node
func (n DeclAttrInstance) TokenLiteral() token.Token {
	return n.Token
}

func MakeAttributeInstance(tok token.Token, ref StaticReference) *DeclAttrInstance {
	return &DeclAttrInstance{tok, ref, nil}
}

func (n *DeclAttrInstance) AddArgument(arg Expr) {
	n.Arguments = append(n.Arguments, arg)
}

func (n DeclAttrInstance) EnumerateChildNodes(action func(child Node)) {
	action(n.Reference)
	n.Reference.EnumerateChildNodes(action)
	for _, argument := range n.Arguments {
		action(argument)
		argument.EnumerateChildNodes(action)
	}
}
