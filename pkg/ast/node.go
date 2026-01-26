package ast

import "code.knabel.dev/zirric-lang/zirric/pkg/token"

type Node interface {
	TokenLiteral() token.Token
	EnumerateChildNodes(func(child Node))
	// String() string
}

// Expressions produce values.
type Expression interface {
	Node
	expressionNode()
}

// Statements can be evaluated.
type Statement interface {
	Node
	statementNode()
}

// Delarations are statements that provide new bindings.
type Declaration interface {
	Node
	declarationNode()
}

type StatementDeclaration interface {
	Statement
	Declaration
}
