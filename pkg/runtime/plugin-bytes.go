package runtime

import (
	stdbytes "bytes"
	"encoding/hex"
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ ExternPlugin = &BytesPlugin{}

// BytesPlugin provides runtime bindings for the bytes module's extern declarations: raw byte reinterpretation, hex codec, slicing, and substring search, none of which are expressible in Zirric itself.
// Higher-level helpers built from these live in bytes/bytes.zirr.
type BytesPlugin struct{}

func (*BytesPlugin) Module() string { return "bytes" }

// Bind implements ExternPlugin.
func (*BytesPlugin) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "fromString":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			str, ok := args[0].(String)
			if !ok {
				return nil, fmt.Errorf("fromString expects a String argument, got %s", TypeName(args[0]))
			}
			return Binary(str), nil
		})
	case "fromChar":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			c, ok := args[0].(Char)
			if !ok {
				return nil, fmt.Errorf("fromChar expects a Char argument, got %s", TypeName(args[0]))
			}
			return Binary(string(rune(c))), nil
		})
	case "from":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return likeToBinary(args[0])
		})
	case "toString":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			b, err := likeToBinary(args[0])
			if err != nil {
				return nil, err
			}
			return String(b), nil
		})
	case "toHex":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			b, err := likeToBinary(args[0])
			if err != nil {
				return nil, err
			}
			return String(hex.EncodeToString(b)), nil
		})
	case "fromHex":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			str, ok := args[0].(String)
			if !ok {
				return nil, fmt.Errorf("fromHex expects a String argument, got %s", TypeName(args[0]))
			}
			decoded, err := hex.DecodeString(string(str))
			if err != nil {
				return nil, fmt.Errorf("fromHex: %w", err)
			}
			return Binary(decoded), nil
		})
	case "slice":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			b, err := likeToBinary(args[0])
			if err != nil {
				return nil, err
			}
			start, ok := args[1].(Int)
			if !ok {
				return nil, fmt.Errorf("slice expects an Int start argument, got %s", TypeName(args[1]))
			}
			end, ok := args[2].(Int)
			if !ok {
				return nil, fmt.Errorf("slice expects an Int end argument, got %s", TypeName(args[2]))
			}
			if start < 0 || end > Int(len(b)) || start > end {
				return nil, fmt.Errorf("slice bounds out of range [%d:%d] with length %d", start, end, len(b))
			}
			out := make(Binary, int(end-start))
			copy(out, b[start:end])
			return out, nil
		})
	case "_firstIndexOf":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			b, needle, err := likeToBinaryPair(args)
			if err != nil {
				return nil, err
			}
			return Int(stdbytes.Index(b, needle)), nil
		})
	case "_lastIndexOf":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			b, needle, err := likeToBinaryPair(args)
			if err != nil {
				return nil, err
			}
			return Int(stdbytes.LastIndex(b, needle)), nil
		})
	}
	return nil
}

// likeToBinary normalizes a Zirric Like value (Byte or Binary) to Binary.
func likeToBinary(v RuntimeValue) (Binary, error) {
	switch v := v.(type) {
	case Byte:
		return Binary{byte(v)}, nil
	case Binary:
		return v, nil
	default:
		return nil, fmt.Errorf("expected a Byte or Binary argument, got %s", TypeName(v))
	}
}

func likeToBinaryPair(args []RuntimeValue) (Binary, Binary, error) {
	b, err := likeToBinary(args[0])
	if err != nil {
		return nil, nil, err
	}
	needle, err := likeToBinary(args[1])
	if err != nil {
		return nil, nil, err
	}
	return b, needle, nil
}
