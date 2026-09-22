package runtime

import "code.knabel.dev/zirric-lang/zirric/pkg/ast"

var (
	_ FieldedType = &DataType{}
	_ FieldedType = &AttributeType{}
)

// FieldedType is a declared type whose fields reflection can enumerate, which is a data type or an attribute type.
// Everything a field was written with is reached by position, so that a type carrying none of one kind can keep a nil slice rather than an empty one per field.
type FieldedType interface {
	Fields() []*ast.Symbol
	FieldAttributesAt(i int) map[TypeId]int
	FieldTypeAt(i int) TypeRef
	FieldDocsAt(i int) string
}
