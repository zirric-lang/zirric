package runtime

// Attributable is implemented by runtime types that carry an attribute map.
type Attributable interface {
	RuntimeValue
	TypeAttributes() map[TypeId]int
}
