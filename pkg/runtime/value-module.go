package runtime

import "fmt"

var _ RuntimeValue = &ModuleValue{}

type ModuleValue struct {
	name    string
	exports map[string]RuntimeValue
}

func MakeModuleValue(name string, exports map[string]RuntimeValue) *ModuleValue {
	return &ModuleValue{
		name:    name,
		exports: exports,
	}
}

// Inspect implements RuntimeValue.
func (m *ModuleValue) Inspect() string {
	if m.name == "" {
		return "mod"
	}
	return fmt.Sprintf("mod %s", m.name)
}

// Lookup implements RuntimeValue.
func (m *ModuleValue) Lookup(name string) RuntimeValue {
	if m.exports == nil {
		return nil
	}
	return m.exports[name]
}

// TypeConstantId implements RuntimeValue.
func (m *ModuleValue) TypeConstantId() TypeId {
	return typeIdModule
}
