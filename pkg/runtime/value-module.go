package runtime

import (
	"fmt"
	"sort"
)

var _ RuntimeValue = &ModuleValue{}
var _ Attributable = &ModuleValue{}

type ModuleValue struct {
	info    *ModuleInfo
	exports map[string]RuntimeValue
}

func MakeModuleValue(info *ModuleInfo, exports map[string]RuntimeValue) *ModuleValue {
	return &ModuleValue{
		info:    info,
		exports: exports,
	}
}

// Inspect implements RuntimeValue.
func (m *ModuleValue) Inspect() string {
	if m.Name() == "" {
		return "mod"
	}
	return fmt.Sprintf("mod %s", m.Name())
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
	if m.info == nil {
		return ""
	}
	return m.info.Name
}

// Info is what the module declared, which is where reflection reads a member's kind, source and attributes from.
func (m *ModuleValue) Info() *ModuleInfo {
	return m.info
}

// TypeAttributes implements Attributable, carrying the attributes written on the module's own `mod` declaration.
func (m *ModuleValue) TypeAttributes() map[TypeId]int {
	if m.info == nil {
		return nil
	}
	return m.info.Attributes
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
