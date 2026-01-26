package ast

import "code.knabel.dev/zirric-lang/zirric/pkg/token"

type OperatorBinary token.Token

type OperatorUnary token.Token

func (o OperatorBinary) TokenLiteral() token.Token {
	return token.Token(o)
}
func (o OperatorUnary) TokenLiteral() token.Token {
	return token.Token(o)
}

func (o OperatorBinary) EnumerateChildNodes(action func(Node)) {
	// No child nodes.
}

func (o OperatorUnary) EnumerateChildNodes(action func(Node)) {
	// No child nodes.
}
