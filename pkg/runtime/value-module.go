package runtime

import (
	"fmt"
	"sort"
)

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

func (m *ModuleValue) Name() string {
	return m.name
}

// MemberNames returns the module's public member names in sorted order, matching the order compileModuleValue emits them in.
func (m *ModuleValue) MemberNames() []string {
	names := make([]string, 0, len(m.exports))
	for name := range m.exports {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
