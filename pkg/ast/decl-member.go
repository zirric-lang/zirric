package ast

type DeclMember interface {
	Declaration
	IsConstantMember() bool
}
