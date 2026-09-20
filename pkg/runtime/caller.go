package runtime

import (
	"encoding/hex"
	"fmt"
	"strconv"
)

// VMCaller lets extern function implementations call back into VM-managed
// behavior that only exists once real values are available at runtime (not
// at plugin bind time, which runs during compilation, before any VM exists).
type VMCaller interface {
	// CallFunction invokes a compiled function or closure value with args
	// and returns its result.
	CallFunction(fn RuntimeValue, args ...RuntimeValue) (RuntimeValue, error)
	// AttributesOf returns the attribute map that applies to v — its own
	// instance-level attributes if it carries them directly (e.g. a
	// function's own @Attr), otherwise its declared type's attributes.
	AttributesOf(v RuntimeValue) map[TypeId]int
	// ResolveGlobal forces evaluation of the global at index id, e.g. to
	// read an attribute's payload value found via AttributesOf.
	ResolveGlobal(id int) (RuntimeValue, error)
	// ResolveModuleMember returns a public member of a compiled module, named as it is written in source (e.g. "prelude", "Ok").
	// It is how a plugin reaches a declared type in order to construct values of it: MakeDataValue copies the DataType's attributes, which a hand-built DataValue would otherwise lack.
	// The module must have been compiled into the program, which for prelude is always the case.
	ResolveModuleMember(moduleName string, memberName string) (RuntimeValue, error)
}

// TrivialString converts v to a display string for the handful of builtin
// types with an unambiguous textual form. Used by both string concatenation
// (`+`) and as fmt.sprint's fallback for values with no @Printable
// attribute. Byte renders as two hex digits.
func TrivialString(v RuntimeValue) (string, bool) {
	switch v := v.(type) {
	case String:
		return string(v), true
	case Int:
		return strconv.FormatInt(int64(v), 10), true
	case Float:
		return strconv.FormatFloat(float64(v), 'g', -1, 64), true
	case Char:
		return string(rune(v)), true
	case Byte:
		return fmt.Sprintf("%02x", byte(v)), true
	case Binary:
		return hex.EncodeToString(v), true
	case Bool:
		if v {
			return "true", true
		}
		return "false", true
	case Void:
		return "void", true
	default:
		return "", false
	}
}

// ResultOk builds a prelude Ok carrying value.
// Going through the declared type rather than assembling a DataValue directly is what preserves @AnyResult, without which results.from and tests.run would not recognize the value as a result at all.
func ResultOk(caller VMCaller, value RuntimeValue) (RuntimeValue, error) {
	return makePreludeResult(caller, "Ok", value)
}

// ResultErr builds a prelude Err carrying reason, which is conventionally a String.
func ResultErr(caller VMCaller, reason RuntimeValue) (RuntimeValue, error) {
	return makePreludeResult(caller, "Err", reason)
}

func makePreludeResult(caller VMCaller, name string, payload RuntimeValue) (RuntimeValue, error) {
	if caller == nil {
		return nil, fmt.Errorf("%s requires a VMCaller", name)
	}
	member, err := caller.ResolveModuleMember("prelude", name)
	if err != nil {
		return nil, err
	}
	dataType, ok := member.(*DataType)
	if !ok {
		return nil, fmt.Errorf("prelude.%s is %T, not a data type", name, member)
	}
	return MakeDataValue(dataType, []RuntimeValue{payload}), nil
}

// MakeDataValueNamed builds a value of a declared type, taking its fields by name.
// Field order is read from the resolved type rather than assumed, so reordering a data declaration cannot silently misalign the values a plugin supplies.
func MakeDataValueNamed(caller VMCaller, moduleName string, typeName string, fields map[string]RuntimeValue) (RuntimeValue, error) {
	if caller == nil {
		return nil, fmt.Errorf("%s.%s can only be built while a VM is running", moduleName, typeName)
	}
	member, err := caller.ResolveModuleMember(moduleName, typeName)
	if err != nil {
		return nil, err
	}
	dataType, ok := member.(*DataType)
	if !ok {
		return nil, fmt.Errorf("%s.%s is %T, not a data type", moduleName, typeName, member)
	}
	values := make([]RuntimeValue, len(dataType.FieldSymbols))
	for i, field := range dataType.FieldSymbols {
		value, ok := fields[field.Name]
		if !ok {
			return nil, fmt.Errorf("%s.%s is missing field %q", moduleName, typeName, field.Name)
		}
		values[i] = value
	}
	if len(fields) != len(values) {
		return nil, fmt.Errorf("%s.%s takes %d fields, got %d", moduleName, typeName, len(values), len(fields))
	}
	return MakeDataValue(dataType, values), nil
}
