package runtime

import "strings"

type DataValue struct {
	TypeId TypeId
	Attrs  map[TypeId]int
	Values []RuntimeValue
	Fields map[string]int

	// TypeName is captured at construction time, from the DataType that created this value, purely for Inspect() — a DataValue otherwise has no way to reach its type's declared name later (TypeId is just a number).
	TypeName string
}

func MakeDataValue(dt *DataType, values []RuntimeValue) *DataValue {
	fields := make(map[string]int, len(dt.FieldSymbols))
	for i, f := range dt.FieldSymbols {
		fields[f.Name] = i
	}
	return &DataValue{
		TypeId:   TypeId(*dt.Symbol.ConstantId),
		Attrs:    dt.Attributes,
		Fields:   fields,
		Values:   values,
		TypeName: dt.Symbol.Decl.DeclName().Value,
	}
}

// Inspect implements RuntimeValue.
// Mirrors the constructor-call syntax used to build the value, e.g.
// _notPrintable("nested") for `data _notPrintable { text }`.
func (dv *DataValue) Inspect() string {
	var b strings.Builder
	b.WriteString(dv.TypeName)
	b.WriteByte('(')
	for i, v := range dv.Values {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(v.Inspect())
	}
	b.WriteByte(')')
	return b.String()
}

// Lookup implements RuntimeValue.
func (dv *DataValue) Lookup(name string) RuntimeValue {
	idx, ok := dv.Fields[name]
	if !ok {
		return nil
	}
	return dv.Values[idx]
}

// TypeConstantId implements RuntimeValue.
func (dv *DataValue) TypeConstantId() TypeId {
	return dv.TypeId
}

// TypeAttributes implements Attributable.
// Returns the attribute map from the DataType that created this value, allowing attribute lookups (e.g. @Printable) to work without a TypeId table lookup.
func (dv *DataValue) TypeAttributes() map[TypeId]int {
	return dv.Attrs
}
