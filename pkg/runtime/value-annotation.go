package runtime

import "fmt"

type AnnotationValue struct {
	TypeId TypeId
	Values []RuntimeValue
	Fields map[string]int
}

func MakeAnnotationValue(at *AnnotationType, values []RuntimeValue) *AnnotationValue {
	fields := make(map[string]int, len(at.FieldSymbols))
	for i, f := range at.FieldSymbols {
		fields[f.Name] = i
	}
	return &AnnotationValue{
		TypeId: TypeId(*at.Symbol.ConstantId),
		Fields: fields,
		Values: values,
	}
}

// Inspect implements RuntimeValue.
func (av *AnnotationValue) Inspect() string {
	return fmt.Sprintf("annotation #%d { %+v }", av.TypeId, av.Fields)
}

// Lookup implements RuntimeValue.
func (av *AnnotationValue) Lookup(name string) RuntimeValue {
	idx, ok := av.Fields[name]
	if !ok {
		return nil
	}
	return av.Values[idx]
}

// TypeConstantId implements RuntimeValue.
func (av *AnnotationValue) TypeConstantId() TypeId {
	return av.TypeId
}
