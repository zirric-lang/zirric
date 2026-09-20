package compiler

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

type CompileError struct {
	Node    ast.Node
	Summary string
	Details string
}

func (e *CompileError) Error() string {
	if e.Node.TokenLiteral().Source == nil {
		return fmt.Sprintf("%s: %s", e.Summary, e.Details)
	}
	return fmt.Sprintf("%s:%d: %s: %s", e.Node.TokenLiteral().Source.File, e.Node.TokenLiteral().Source.Offset, e.Summary, e.Details)
}

func errUndefinedIdentifier(node *ast.ExprIdentifier) *CompileError {
	return &CompileError{
		node,
		"undefined",
		node.Name.String(),
	}
}

func errUnimplemented(node ast.Node, str string, args ...any) *CompileError {
	return &CompileError{
		node,
		"unimplemented",
		fmt.Sprintf(str, args...),
	}
}
