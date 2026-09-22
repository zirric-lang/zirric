package ast

import (
	"bytes"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Expr = ExprMemberAccess{}

// MemberAccess distinguishes the three ways a field can be read off a value.
type MemberAccess int

const (
	// MemberAccessPlain reads the field unconditionally, as in `target.field`.
	MemberAccessPlain MemberAccess = iota
	// MemberAccessOption reads the field unless the target is None, in which case the whole chain is None, as in `target?.field`.
	MemberAccessOption
	// MemberAccessResult reads the field unless the target is an error, in which case the enclosing function returns it, as in `target!.field`.
	MemberAccessResult
)

// Operator is how this form is written in source.
func (a MemberAccess) Operator() string {
	switch a {
	case MemberAccessOption:
		return "?."
	case MemberAccessResult:
		return "!."
	default:
		return "."
	}
}

type ExprMemberAccess struct {
	Token    token.Token
	Target   Expr
	Property Identifier
}

func MakeExprMemberAccess(tok token.Token, target Expr, prop Identifier) *ExprMemberAccess {
	return &ExprMemberAccess{
		Token:    tok,
		Target:   target,
		Property: prop,
	}
}

// Access is which of the three member access forms this is, read back off the operator token so the two can never disagree.
func (n ExprMemberAccess) Access() MemberAccess {
	switch n.Token.Type {
	case token.QUESTION_DOT:
		return MemberAccessOption
	case token.BANG_DOT:
		return MemberAccessResult
	default:
		return MemberAccessPlain
	}
}

// EnumerateChildNodes implements Expr.
func (n ExprMemberAccess) EnumerateChildNodes(action func(child Node)) {
	action(n.Target)
	n.Target.EnumerateChildNodes(action)

	action(n.Property)
}

// TokenLiteral implements Expr.
func (n ExprMemberAccess) TokenLiteral() token.Token {
	return n.Token
}

// Expression implements Expr.
func (e ExprMemberAccess) Expression() string {
	var out bytes.Buffer

	out.WriteString(e.Target.Expression())
	out.WriteString(e.Access().Operator())
	out.WriteString(e.Property.Value)

	return out.String()
}
