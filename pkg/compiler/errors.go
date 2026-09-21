package compiler

import (
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// CompileError reports a problem with a declaration or expression, anchored to the node it came from.
//
// Summary says what is wrong in a few words and Details says what was found, so that the two read as one sentence without either repeating the other.
// Holding the node rather than a formatted string is what lets the message carry a position, and later an excerpt of the source it refers to.
type CompileError struct {
	Node    ast.Node
	Summary string
	Details string
}

// Error implements error.
func (e *CompileError) Error() string {
	var out strings.Builder
	if location := e.location(); location != "" {
		out.WriteString(location)
		out.WriteString(": ")
	}
	out.WriteString(e.Summary)
	if e.Details != "" {
		out.WriteString(": ")
		out.WriteString(e.Details)
	}
	return out.String()
}

// location renders where the error is, or nothing at all when the node carries no source.
// A node without a position is normal for generated declarations, and a bare message reads better than a fake one.
func (e *CompileError) location() string {
	if e.Node == nil {
		return ""
	}
	return e.Node.TokenLiteral().Source.String()
}

// Position implements diag.Positioned, so a renderer can show the line this refers to.
func (e *CompileError) Position() *token.Source {
	if e.Node == nil {
		return nil
	}
	return e.Node.TokenLiteral().Source
}

// CompileErrors is a collection of compile errors that implements the error interface.
// Callers can unwrap it via errors.As to reach the individual errors, mirroring ParseErrors.
type CompileErrors []*CompileError

// Unwrap implements the convention for an error holding several errors, which is what lets errors.As reach an individual CompileError and what a renderer walks to show each one.
func (e CompileErrors) Unwrap() []error {
	errs := make([]error, len(e))
	for i := range e {
		errs[i] = e[i]
	}
	return errs
}

func (e CompileErrors) Error() string {
	msgs := make([]string, len(e))
	for i, ce := range e {
		msgs[i] = ce.Error()
	}
	return strings.Join(msgs, "\n")
}

// errAt reports a problem at a node, with the details written in the call.
// This is the general form; prefer one of the named helpers below when the problem has a name, so that the same wording is used wherever it is reported.
func errAt(node ast.Node, summary string, format string, args ...any) *CompileError {
	return &CompileError{
		Node:    node,
		Summary: summary,
		Details: fmt.Sprintf(format, args...),
	}
}

// errInvariant reports a state the compiler expected to hold and found broken.
//
// Unlike the other helpers this describes a bug in the compiler rather than a mistake in the source, so it says "internal error" plainly: whoever reads it has done nothing wrong and can only report it.
// That is also why the details may run long — an invariant is worth explaining in full, since the person who hits one has no way to guess what it means.
func errInvariant(node ast.Node, format string, args ...any) *CompileError {
	return &CompileError{
		Node:    node,
		Summary: "internal error",
		Details: fmt.Sprintf(format, args...),
	}
}

// errUndeclaredSymbol reports a symbol the analyzer left without a declaration, anchored to the first place it was used.
// A symbol with no usages at all carries no node, which is why the position is allowed to be missing rather than reached for blindly.
func errUndeclaredSymbol(sym *ast.Symbol) *CompileError {
	var node ast.Node
	if len(sym.Usages) > 0 {
		node = sym.Usages[0].Node
	}
	return &CompileError{
		Node:    node,
		Summary: "undeclared symbol",
		Details: sym.Name,
	}
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
