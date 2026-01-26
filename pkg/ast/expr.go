package ast

type Expr interface {
	Node

	Expression() string
}
