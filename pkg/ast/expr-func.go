package ast

import (
	"bytes"
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Expr = ExprFunc{}

type ExprFunc struct {
	Token      token.Token
	Name       string
	Parameters []DeclParameter
	ReturnType TypeExpr
	Impl       Block
	Decls      *DeclTable
	Symbols    *SymbolTable
}

func MakeExprFunc(token token.Token, name string, parent *DeclTable) (*ExprFunc, *DeclTable) {
	f := &ExprFunc{
		Token: token,
		Name:  name,
	}
	f.Decls = parent.MakeChild(f)
	return f, f.Decls
}

func (f *ExprFunc) SetParams(ps []DeclParameter) {
	f.Parameters = ps
}
func (f *ExprFunc) SetImplBlock(impl Block) {
	f.Impl = impl
}

// EnumerateChildNodes implements Expr.
func (n ExprFunc) EnumerateChildNodes(action func(child Node)) {
	for _, node := range n.Parameters {
		action(node)
		node.EnumerateChildNodes(action)
	}
	if n.ReturnType != nil {
		action(n.ReturnType)
		n.ReturnType.EnumerateChildNodes(action)
	}
	for _, node := range n.Impl {
		action(node)
		node.EnumerateChildNodes(action)
	}
}

// TokenLiteral implements Expr.
func (e ExprFunc) TokenLiteral() token.Token {
	return e.Token
}

// Expression implements Expr.
func (e ExprFunc) Expression() string {
	var out bytes.Buffer

	out.WriteString("fn(")
	for i, p := range e.Parameters {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(p.Name.String())
	}
	out.WriteString(") {")
	fmt.Fprintf(&out, "/* %d stmts */", len(e.Impl))
	out.WriteString("}")

	return out.String()
}
