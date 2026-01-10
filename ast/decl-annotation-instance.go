package ast

import (
	"code.knabel.dev/zirric-lang/zirric/token"
)

type DeclAnnotationInstance struct {
	Token     token.Token
	Reference StaticReference
	Arguments []Expr
}

// TokenLiteral implements Node
func (n DeclAnnotationInstance) TokenLiteral() token.Token {
	return n.Token
}

func MakeAnnotationInstance(tok token.Token, ref StaticReference) *DeclAnnotationInstance {
	return &DeclAnnotationInstance{tok, ref, nil}
}

func (n DeclAnnotationInstance) AddArgument(arg Expr) {
	n.Arguments = append(n.Arguments, arg)
}

func (n DeclAnnotationInstance) EnumerateChildNodes(action func(child Node)) {
	action(n.Reference)
	for _, argument := range n.Arguments {
		action(argument)
	}
}
