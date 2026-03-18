package ast

import (
	"bytes"
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// TypeExprFunc represents a function type expression: fn(params) -> ReturnType.
type TypeExprFunc struct {
	Token      token.Token     // the `fn` token
	Parameters []DeclParameter // parameter names with optional type hints
	ReturnType TypeExpr        // nil if no return type specified
}

var _ TypeExpr = TypeExprFunc{}

func MakeTypeExprFunc(tok token.Token, params []DeclParameter, returnType TypeExpr) TypeExprFunc {
	return TypeExprFunc{Token: tok, Parameters: params, ReturnType: returnType}
}

func (e TypeExprFunc) TokenLiteral() token.Token { return e.Token }

func (e TypeExprFunc) EnumerateChildNodes(action func(Node)) {
	for i := range e.Parameters {
		action(&e.Parameters[i])
	}
	if e.ReturnType != nil {
		action(e.ReturnType)
	}
}

func (e TypeExprFunc) TypeExpression() string {
	var out bytes.Buffer
	out.WriteString("fn(")
	for i, p := range e.Parameters {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(p.Name.Value)
		if p.TypeHint != nil {
			fmt.Fprintf(&out, ": %s", p.TypeHint.TypeExpression())
		}
	}
	out.WriteString(")")
	if e.ReturnType != nil {
		fmt.Fprintf(&out, " -> %s", e.ReturnType.TypeExpression())
	}
	return out.String()
}
