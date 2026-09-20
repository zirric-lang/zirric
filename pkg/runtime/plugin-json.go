package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ ExternPlugin = &JSONPlugin{}

// JSONPlugin provides runtime bindings for the json module's extern declarations.
// JSON maps onto Zirric's own values rather than onto a tree of its own: an object is a Dict, an array an Array, and null is void.
// Nothing here knows about data types or attributes; mapping those onto this tree is the coding module's job.
type JSONPlugin struct{}

func (*JSONPlugin) Module() string { return "json" }

// Bind implements ExternPlugin.
func (*JSONPlugin) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "parse":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			text, ok := args[0].(String)
			if !ok {
				return nil, fmt.Errorf("parse expects a String, got %T", args[0])
			}
			decoder := json.NewDecoder(bytes.NewReader([]byte(text)))
			// Numbers are kept as written so that the literal decides the type, rather than every number arriving as a float.
			decoder.UseNumber()

			var decoded any
			if err := decoder.Decode(&decoded); err != nil {
				return ResultErr(caller, String(err.Error()))
			}
			if decoder.More() {
				return ResultErr(caller, String("unexpected trailing content after the JSON value"))
			}
			value, err := jsonToValue(decoded)
			if err != nil {
				return ResultErr(caller, String(err.Error()))
			}
			return ResultOk(caller, value)
		})
	case "format":
		return makeJSONFormatter(decl, "")
	case "formatIndented":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			indent, ok := args[1].(String)
			if !ok {
				return nil, fmt.Errorf("formatIndented expects a String indent, got %T", args[1])
			}
			return formatJSON(caller, args[0], string(indent))
		})
	}
	return nil
}

func makeJSONFormatter(decl *ast.Symbol, indent string) RuntimeValue {
	return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
		return formatJSON(caller, args[0], indent)
	})
}

func formatJSON(caller VMCaller, value RuntimeValue, indent string) (RuntimeValue, error) {
	plain, err := valueToJSON(value)
	if err != nil {
		return ResultErr(caller, String(err.Error()))
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	// Zirric strings are text, not HTML, so escaping < and & would corrupt them for no benefit.
	encoder.SetEscapeHTML(false)
	if indent != "" {
		encoder.SetIndent("", indent)
	}
	if err := encoder.Encode(plain); err != nil {
		return ResultErr(caller, String(err.Error()))
	}
	// Encode always appends a newline, which belongs to streaming rather than to the value.
	return ResultOk(caller, String(bytes.TrimRight(buffer.Bytes(), "\n")))
}

// jsonToValue converts decoded JSON into Zirric values.
// A number becomes an Int when it was written as one and fits, and a Float otherwise, so that 1 and 1.0 stay distinguishable.
func jsonToValue(decoded any) (RuntimeValue, error) {
	switch decoded := decoded.(type) {
	case nil:
		return Void{}, nil
	case bool:
		return Bool(decoded), nil
	case string:
		return String(decoded), nil
	case json.Number:
		if whole, err := decoded.Int64(); err == nil {
			return Int(whole), nil
		}
		fractional, err := decoded.Float64()
		if err != nil {
			return nil, fmt.Errorf("number %s is out of range", decoded.String())
		}
		return Float(fractional), nil
	case []any:
		items := make(Array, len(decoded))
		for i, item := range decoded {
			converted, err := jsonToValue(item)
			if err != nil {
				return nil, err
			}
			items[i] = converted
		}
		return items, nil
	case map[string]any:
		entries := make(Dict, len(decoded))
		for key, item := range decoded {
			converted, err := jsonToValue(item)
			if err != nil {
				return nil, err
			}
			entries[String(key)] = converted
		}
		return entries, nil
	}
	return nil, fmt.Errorf("unsupported JSON value %T", decoded)
}

// valueToJSON converts Zirric values into something encoding/json accepts.
// Only the types JSON has a form for are accepted; anything else is reported by name rather than rendered as something it is not.
func valueToJSON(value RuntimeValue) (any, error) {
	switch value := value.(type) {
	case Void:
		return nil, nil
	case Bool:
		return bool(value), nil
	case Int:
		return int64(value), nil
	case Float:
		return float64(value), nil
	case String:
		return string(value), nil
	case Array:
		items := make([]any, len(value))
		for i, item := range value {
			converted, err := valueToJSON(item)
			if err != nil {
				return nil, err
			}
			items[i] = converted
		}
		return items, nil
	case Dict:
		return dictToJSON(value)
	}
	return nil, fmt.Errorf("%s cannot be written as JSON", typeNameForJSON(value))
}

func dictToJSON(entries Dict) (any, error) {
	// Keys are sorted so that the same dict always produces the same text, which map iteration alone would not give.
	keys := make([]string, 0, len(entries))
	byKey := make(map[string]RuntimeValue, len(entries))
	for key, item := range entries {
		name, ok := key.(String)
		if !ok {
			return nil, fmt.Errorf("a JSON object needs String keys, got %s", typeNameForJSON(key))
		}
		keys = append(keys, string(name))
		byKey[string(name)] = item
	}
	sort.Strings(keys)

	object := make(map[string]any, len(keys))
	for _, key := range keys {
		converted, err := valueToJSON(byKey[key])
		if err != nil {
			return nil, err
		}
		object[key] = converted
	}
	return object, nil
}

func typeNameForJSON(value RuntimeValue) string {
	switch value.(type) {
	case Binary:
		return "Binary"
	case Byte:
		return "Byte"
	case Char:
		return "Char"
	case Duration:
		return "Duration"
	case Instant:
		return "Instant"
	case Timestamp:
		return "Timestamp"
	case *DataValue:
		return "a data value"
	}
	return fmt.Sprintf("%T", value)
}
