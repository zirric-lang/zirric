package runtime

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"github.com/goccy/go-yaml"
)

var _ ExternPlugin = &YAMLPlugin{}

// YAMLPlugin provides runtime bindings for the yaml module's extern declarations.
// YAML maps onto the same native tree as JSON — a mapping is a Dict, a sequence an Array, null is void — so that a tree means one thing whichever format produced it, which is what lets coding stay format-agnostic.
type YAMLPlugin struct{}

func (*YAMLPlugin) Module() string { return "yaml" }

// Bind implements ExternPlugin.
func (*YAMLPlugin) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "parse":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			text, ok := args[0].(String)
			if !ok {
				return nil, fmt.Errorf("parse expects a String, got %s", TypeName(args[0]))
			}
			documents, err := decodeYAMLDocuments(string(text))
			if err != nil {
				return ResultErr(caller, String(err.Error()))
			}
			if len(documents) == 0 {
				return ResultOk(caller, Void{})
			}
			// A stream holding more than one document is not one value, the same way trailing JSON is not.
			if len(documents) > 1 {
				return ResultErr(caller, String("the stream holds more than one document, use parseAll"))
			}
			return ResultOk(caller, documents[0])
		})
	case "parseAll":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			text, ok := args[0].(String)
			if !ok {
				return nil, fmt.Errorf("parseAll expects a String, got %s", TypeName(args[0]))
			}
			documents, err := decodeYAMLDocuments(string(text))
			if err != nil {
				return ResultErr(caller, String(err.Error()))
			}
			result := make(Array, 0, len(documents))
			result = append(result, documents...)
			return ResultOk(caller, result)
		})
	case "format":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return formatYAML(caller, args[0], 2)
		})
	case "formatIndented":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			indent, ok := args[1].(Int)
			if !ok {
				return nil, fmt.Errorf("formatIndented expects an Int indent, got %s", TypeName(args[1]))
			}
			if indent < 1 {
				return ResultErr(caller, String("the indent must be at least one space"))
			}
			return formatYAML(caller, args[0], int(indent))
		})
	}
	return nil
}

// decodeYAMLDocuments reads every document in a stream into the native tree.
func decodeYAMLDocuments(text string) ([]RuntimeValue, error) {
	decoder := yaml.NewDecoder(strings.NewReader(text))
	documents := make([]RuntimeValue, 0, 1)
	for {
		var decoded any
		if err := decoder.Decode(&decoded); err != nil {
			// The end of the stream is the only way out of this loop.
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		value, err := yamlToValue(decoded)
		if err != nil {
			return nil, err
		}
		documents = append(documents, value)
	}
	return documents, nil
}

func formatYAML(caller VMCaller, value RuntimeValue, indent int) (RuntimeValue, error) {
	plain, err := valueToYAML(value)
	if err != nil {
		return ResultErr(caller, String(err.Error()))
	}
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer, yaml.Indent(indent))
	if err := encoder.Encode(plain); err != nil {
		return ResultErr(caller, String(err.Error()))
	}
	if err := encoder.Close(); err != nil {
		return ResultErr(caller, String(err.Error()))
	}
	// The encoder ends a document with a newline, which belongs to the stream rather than to the value.
	return ResultOk(caller, String(bytes.TrimRight(buffer.Bytes(), "\n")))
}

// yamlToValue converts a decoded YAML document into Zirric values.
// Two things keep the tree format-neutral, so that a document and its JSON equivalent decode to the same thing: a timestamp stays the text it was written as, and a mapping key becomes a String whatever it was written as, so `1: one` and `"1": one` are one and the same.
func yamlToValue(decoded any) (RuntimeValue, error) {
	switch decoded := decoded.(type) {
	case nil:
		return Void{}, nil
	case bool:
		return Bool(decoded), nil
	case string:
		return String(decoded), nil
	case int:
		return Int(decoded), nil
	case int64:
		return Int(decoded), nil
	case uint64:
		return Int(decoded), nil
	case float64:
		return Float(decoded), nil
	case []any:
		items := make(Array, len(decoded))
		for i, item := range decoded {
			converted, err := yamlToValue(item)
			if err != nil {
				return nil, err
			}
			items[i] = converted
		}
		return items, nil
	case map[string]any:
		entries := make(Dict, len(decoded))
		for key, item := range decoded {
			converted, err := yamlToValue(item)
			if err != nil {
				return nil, err
			}
			entries[String(key)] = converted
		}
		return entries, nil
	}
	return nil, fmt.Errorf("unsupported YAML value %T", decoded)
}

// valueToYAML converts Zirric values into something the encoder accepts, accepting exactly what JSON does so that the two formats agree on what a tree may hold.
func valueToYAML(value RuntimeValue) (any, error) {
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
			converted, err := valueToYAML(item)
			if err != nil {
				return nil, err
			}
			items[i] = converted
		}
		return items, nil
	case Dict:
		return dictToYAML(value)
	}
	return nil, fmt.Errorf("%s cannot be written as YAML", TypeName(value))
}

func dictToYAML(entries Dict) (any, error) {
	// Keys are sorted so that the same dict always produces the same text, which map iteration alone would not give.
	keys := make([]string, 0, len(entries))
	byKey := make(map[string]RuntimeValue, len(entries))
	for key, item := range entries {
		name, ok := key.(String)
		if !ok {
			return nil, fmt.Errorf("a YAML mapping written from a Dict needs String keys, got %s", TypeName(key))
		}
		keys = append(keys, string(name))
		byKey[string(name)] = item
	}
	sort.Strings(keys)

	mapping := yaml.MapSlice{}
	for _, key := range keys {
		converted, err := valueToYAML(byKey[key])
		if err != nil {
			return nil, err
		}
		mapping = append(mapping, yaml.MapItem{Key: key, Value: converted})
	}
	return mapping, nil
}
