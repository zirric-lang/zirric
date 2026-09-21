package analyzer

import (
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// AnalysisSeverity's zero value is AnalysisSeverityError, so existing call sites that don't set it keep their current behavior.
type AnalysisSeverity int

const (
	AnalysisSeverityError AnalysisSeverity = iota
	AnalysisSeverityWarning
)

type AnalysisError struct {
	Token    token.Token
	Summary  string
	Details  string
	Severity AnalysisSeverity
}

// Error implements error.
func (e AnalysisError) Error() string {
	if e.Token.Source == nil {
		return fmt.Sprintf("%s: %s", e.Summary, e.Details)
	}
	if e.Details == "" {
		return fmt.Sprintf("%s: %s", e.Token.Source.String(), e.Summary)
	}
	return fmt.Sprintf("%s: %s: %s", e.Token.Source.String(), e.Summary, e.Details)
}

// Position implements diag.Positioned, so a renderer can show the line this refers to.
func (e AnalysisError) Position() *token.Source {
	return e.Token.Source
}

// errUnknownReference reports a reference that names nothing that exists.
// The token is the part that failed rather than the whole reference, so `a.b.c` points at the segment that went wrong while the message still names the reference a person wrote.
func errUnknownReference(tok token.Token, ref ast.StaticReference) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "unknown reference",
		Details: fmt.Sprintf("%s could not be resolved", ref.String()),
	}
}

// errNotExported reports a reference to something that exists but is not public.
// Saying so beats reporting it as unknown, since the name is right and only the visibility is wrong.
func errNotExported(tok token.Token, ref ast.StaticReference, name string) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "not public",
		Details: fmt.Sprintf("%s is declared in %s but not exported from it", name, ref.String()),
	}
}

// errNotReferenceable reports a path that tries to reach inside a declaration whose members a reference cannot name, such as a data type's fields.
// A field reports the same export scope as an unexported declaration, so telling the two apart needs the declaration itself; saying "not public" here would send a reader off to export something that never can be.
func errNotReferenceable(tok token.Token, ref ast.StaticReference, name string) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "not referenceable",
		Details: fmt.Sprintf("%s is a field, which a reference cannot reach, so %s could not be resolved", name, ref.String()),
	}
}

// errHasNoMembers reports a segment of a reference that cannot be looked inside.
func errHasNoMembers(tok token.Token, ref ast.StaticReference, name string) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "has no members",
		Details: fmt.Sprintf("%s has nothing to look up inside it, so %s cannot be resolved", name, ref.String()),
	}
}

// errInvalidReference reports a reference that is malformed rather than merely unresolved.
func errInvalidReference(tok token.Token, details string) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "invalid reference",
		Details: details,
	}
}

// errUnknownModule reports an import naming a module that could not be resolved.
func errUnknownModule(tok token.Token, moduleName ast.ModuleName) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "unknown module",
		Details: fmt.Sprintf("%s could not be resolved", moduleName),
	}
}

// errUnknownImportMember reports an import of a name a module does not export.
func errUnknownImportMember(tok token.Token, moduleName ast.ModuleName, member string) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "unknown import member",
		Details: fmt.Sprintf("%s exports no %s", moduleName, member),
	}
}

// errDependencyNotInstalled reports a dependency the Cavefile declares but the project has not fetched.
// It is a warning rather than an error because the fix is a command to run, not a change to the source.
func errDependencyNotInstalled(tok token.Token, names []string) *AnalysisError {
	return &AnalysisError{
		Token:    tok,
		Summary:  "dependency not installed",
		Details:  fmt.Sprintf("run 'zirric install' to install %s", strings.Join(names, ", ")),
		Severity: AnalysisSeverityWarning,
	}
}

// AnalysisErrors is a collection of analysis errors that implements the error interface.
//
// Analysis reports everything it finds in one pass, so surfacing only the first would make fixing a file a matter of running the compiler once per mistake.
// Callers can reach the individual errors via errors.As, and a renderer walks them to show each one with the line it refers to.
type AnalysisErrors []AnalysisError

// Unwrap implements the convention for an error holding several errors.
func (e AnalysisErrors) Unwrap() []error {
	errs := make([]error, len(e))
	for i := range e {
		errs[i] = e[i]
	}
	return errs
}

func (e AnalysisErrors) Error() string {
	msgs := make([]string, len(e))
	for i, ae := range e {
		msgs[i] = ae.Error()
	}
	return strings.Join(msgs, "\n")
}

// Failing returns the diagnostics that should stop compilation, which is every one that is not a warning.
// A warning describes something worth saying rather than something that prevents a program from being built, so it is reported without failing the build.
func (e AnalysisErrors) Failing() AnalysisErrors {
	var failing AnalysisErrors
	for _, one := range e {
		if one.Severity != AnalysisSeverityWarning {
			failing = append(failing, one)
		}
	}
	return failing
}

// Warnings returns the diagnostics that do not stop compilation.
func (e AnalysisErrors) Warnings() AnalysisErrors {
	var warnings AnalysisErrors
	for _, one := range e {
		if one.Severity == AnalysisSeverityWarning {
			warnings = append(warnings, one)
		}
	}
	return warnings
}

// errNotCallable reports a call on a value that cannot be called, which fails the moment it runs.
func errNotCallable(tok token.Token, found string) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "not callable",
		Details: fmt.Sprintf("%s cannot be called", found),
	}
}

// errWrongArgumentCount reports a call the callee cannot accept, which the VM refuses at the moment it runs.
func errWrongArgumentCount(tok token.Token, name string, want int, got int) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "wrong number of arguments",
		Details: fmt.Sprintf("%s takes %d, got %d", name, want, got),
	}
}

// errArgumentMismatch reports an argument that cannot be what the parameter says it is.
// The VM does not enforce type hints, so this is a promise the program breaks rather than a crash, which is why it names both what was declared and what was passed.
func errArgumentMismatch(tok token.Token, position int, name string, found string, want string) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "wrong argument type",
		Details: fmt.Sprintf("argument %d of %s is declared %s, got %s", position, name, want, found),
	}
}

// errUnsupportedOperator reports a combination the VM has no meaning for.
func errUnsupportedOperator(tok token.Token, operator string, lhs string, rhs string) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "unsupported operator",
		Details: fmt.Sprintf("%s is not defined for %s and %s", operator, lhs, rhs),
	}
}

// errUnknownField reports reading a field a data type does not declare, which yields nothing at runtime rather than the value expected.
func errUnknownField(tok token.Token, typeName string, field string) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "unknown field",
		Details: fmt.Sprintf("%s has no field %s", typeName, field),
	}
}

// errReturnMismatch reports a returned value that cannot be what the function promises.
func errReturnMismatch(tok token.Token, name string, found string, want string) *AnalysisError {
	where := "this function"
	if name != "" {
		where = name
	}
	return &AnalysisError{
		Token:   tok,
		Summary: "wrong return type",
		Details: fmt.Sprintf("%s returns %s, got %s", where, want, found),
	}
}

// errUndefinedName reports a name that stands for nothing, which is usually a typo.
func errUndefinedName(tok token.Token, name string) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "undefined",
		Details: fmt.Sprintf("%s is not declared anywhere in scope", name),
	}
}

// errUnknownModuleMember reports a name read off a module that the module does not offer.
func errUnknownModuleMember(tok token.Token, module string, member string, what string) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "unknown module member",
		Details: fmt.Sprintf("%s %s %s", module, what, member),
	}
}

// errDeclaredValueMismatch reports a value that cannot be what it was declared to be.
func errDeclaredValueMismatch(tok token.Token, name string, found string, want string) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "wrong type",
		Details: fmt.Sprintf("%s is declared %s, got %s", name, want, found),
	}
}

// errUnsupportedUnaryOperator reports an operand the operator has no meaning for.
func errUnsupportedUnaryOperator(tok token.Token, operator string, found string) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "unsupported operator",
		Details: fmt.Sprintf("%s is not defined for %s", operator, found),
	}
}

// errNotIndexable reports indexing a value that cannot be indexed.
func errNotIndexable(tok token.Token, found string) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "not indexable",
		Details: fmt.Sprintf("%s cannot be indexed", found),
	}
}

// errNotIterable reports iterating a value that cannot be iterated.
func errNotIterable(tok token.Token, found string) *AnalysisError {
	return &AnalysisError{
		Token:   tok,
		Summary: "not iterable",
		Details: fmt.Sprintf("%s carries no @Iterable, so it cannot be iterated", found),
	}
}
