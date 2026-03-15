package runtime

import "fmt"

type AttributeValue struct {
	TypeId TypeId
	Values []RuntimeValue
	Fields map[string]int
}

func MakeAttributeValue(at *AttributeType, values []RuntimeValue) *AttributeValue {
	fields := make(map[string]int, len(at.FieldSymbols))
	for i, f := range at.FieldSymbols {
		fields[f.Name] = i
	}
	return &AttributeValue{
		TypeId: TypeId(*at.Symbol.ConstantId),
		Fields: fields,
		Values: values,
	}
}

// Inspect implements RuntimeValue.
func (av *AttributeValue) Inspect() string {
	return fmt.Sprintf("attr #%d { %+v }", av.TypeId, av.Fields)
}

// Lookup implements RuntimeValue.
func (av *AttributeValue) Lookup(name string) RuntimeValue {
	idx, ok := av.Fields[name]
	if !ok {
		return nil
	}
	return av.Values[idx]
}

// TypeConstantId implements RuntimeValue.
func (av *AttributeValue) TypeConstantId() TypeId {
	return av.TypeId
}
