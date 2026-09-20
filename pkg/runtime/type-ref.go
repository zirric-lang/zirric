package runtime

// TypeRefKind names the shape of a type hint as it was written.
type TypeRefKind int

const (
	// TypeRefUnknown is the absence of a type hint, which is also the zero value.
	TypeRefUnknown TypeRefKind = iota
	// TypeRefNamed is a type referred to by name, e.g. String or prelude.Option.
	TypeRefNamed
	// TypeRefArray is [Element].
	TypeRefArray
	// TypeRefDict is [Key: Value].
	TypeRefDict
	// TypeRefFunc is fn(Parameters) -> Returns.
	TypeRefFunc
	// TypeRefAttrs is an attribute constraint, e.g. @Iterable @Countable.
	TypeRefAttrs
)

// TypeRef describes a type hint structurally rather than as text, so that reflection can walk it.
// Type hints are not enforced at runtime, so a TypeRef says what was written, not what a value actually is.
type TypeRef struct {
	Kind TypeRefKind
	// Name is the written name of a TypeRefNamed, e.g. "String" or "prelude.Option".
	Name string
	// Type is the constant id the name resolved to, or nil when the name resolves to nothing that has a runtime type value.
	Type *TypeId
	// Element is an array's element type.
	Element *TypeRef
	// Key and Value are a dict's key and value types.
	Key   *TypeRef
	Value *TypeRef
	// Parameters and Returns describe a function type; Returns is an unknown TypeRef when none was written.
	Parameters []TypeRef
	Returns    *TypeRef
	// Attributes holds each attribute of an attribute constraint, every one of them a TypeRefNamed.
	Attributes []TypeRef
}
