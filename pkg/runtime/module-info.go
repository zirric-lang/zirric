package runtime

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ RuntimeValue = &ModuleInfo{}

// ModuleInfo describes a module as it was declared rather than as it ended up: what it is called, the attributes its `mod` declaration carries, and one entry per public declaration.
// The exported values alone cannot answer what kind of declaration a member is or where it was written, and a member whose value is an Int can carry no attributes of its own, so reflection reads all of that from here instead.
// It lives in the constants table and never reaches the stack, which is why it is a RuntimeValue in name only.
type ModuleInfo struct {
	Name string
	// Docs is the comment written above each of the module's `mod` declarations, in file name order, separated by a blank line.
	Docs       string
	Attributes map[TypeId]int
	// Imports names every module this one imports, ordered by name and free of duplicates.
	Imports []string
	// Sources names every file the module was read from, ordered by name.
	Sources      []string
	Declarations []ModuleDeclaration
}

// ModuleDeclaration is one public declaration of a module, kept as the declaration itself so that nothing is rendered until something asks.
type ModuleDeclaration struct {
	Name       string
	Decl       ast.Decl
	Attributes map[TypeId]int
}

// Inspect implements RuntimeValue.
func (m *ModuleInfo) Inspect() string {
	return fmt.Sprintf("mod %s", m.Name)
}

// Lookup implements RuntimeValue.
func (*ModuleInfo) Lookup(string) RuntimeValue {
	return nil
}

// TypeConstantId implements RuntimeValue.
func (*ModuleInfo) TypeConstantId() TypeId {
	return typeIdModule
}

// TypeAttributes implements Attributable.
func (m *ModuleInfo) TypeAttributes() map[TypeId]int {
	return m.Attributes
}

// Docs is the comment written above the declaration, or the empty string when it carries none.
func (d ModuleDeclaration) Docs() string {
	if d.Decl == nil {
		return ""
	}
	return ast.DocsOf(d.Decl)
}

// Signature renders the declaration the way it was written, without its body.
func (d ModuleDeclaration) Signature() string {
	if overviewable, ok := d.Decl.(ast.Overviewable); ok {
		return overviewable.DeclOverview()
	}
	return d.Name
}

// KindTypeName is the reflect.DeclarationKind member describing the form the declaration was written in.
func (d ModuleDeclaration) KindTypeName() string {
	switch d.Decl.(type) {
	case *ast.DeclData:
		return "DataDecl"
	case *ast.DeclUnion:
		return "UnionDecl"
	case *ast.DeclAttr:
		return "AttrDecl"
	case *ast.DeclExternType:
		return "TypeDecl"
	case *ast.DeclFunc, *ast.DeclExternFunc:
		return "FuncDecl"
	case *ast.DeclConstant, *ast.DeclExternValue:
		return "ConstDecl"
	case *ast.DeclVariable:
		return "VarDecl"
	default:
		return "UnknownDecl"
	}
}

// Source is the file the declaration was read from and the line it starts on, or the empty string and 0 when it was not read from a file at all.
func (d ModuleDeclaration) Source() (string, int) {
	if d.Decl == nil {
		return "", 0
	}
	source := d.Decl.TokenLiteral().Source
	if source == nil {
		return "", 0
	}
	return source.File, source.Line
}
