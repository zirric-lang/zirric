package runtime

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ ExternPlugin = &StringsPlugin{}

// StringsPlugin provides runtime bindings for the strings module's extern declarations.
// Everything here needs Go: character indexing must decode UTF-8, and repeated Zirric string concatenation (`result = result + part`) is O(n^2) since strings are immutable, unlike Binary's append (amortized O(1)) — so anything building a string from parts is implemented here rather than composed in bytes/bytes.zirr's style.
type StringsPlugin struct{}

func (*StringsPlugin) Module() string { return "strings" }

// Bind implements ExternPlugin.
func (*StringsPlugin) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "from":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, err := likeToString(args[0])
			if err != nil {
				return nil, err
			}
			return String(s), nil
		})
	case "count":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, err := likeToString(args[0])
			if err != nil {
				return nil, err
			}
			return Int(utf8.RuneCountInString(s)), nil
		})
	case "charAt":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, err := likeToString(args[0])
			if err != nil {
				return nil, err
			}
			idx, ok := args[1].(Int)
			if !ok {
				return nil, fmt.Errorf("charAt expects an Int index, got %s", TypeName(args[1]))
			}
			runes := []rune(s)
			if idx < 0 || int(idx) >= len(runes) {
				return nil, fmt.Errorf("charAt index %d out of bounds (length %d)", idx, len(runes))
			}
			return Char(runes[idx]), nil
		})
	case "slice":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, err := likeToString(args[0])
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
			runes := []rune(s)
			if start < 0 || end > Int(len(runes)) || start > end {
				return nil, fmt.Errorf("slice bounds out of range [%d:%d] with length %d", start, end, len(runes))
			}
			return String(runes[start:end]), nil
		})
	case "isDigit":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return everyRune(args[0], unicode.IsDigit)
		})
	case "isLetter":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return everyRune(args[0], unicode.IsLetter)
		})
	case "isSpace":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return everyRune(args[0], unicode.IsSpace)
		})
	case "contains":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, needle, err := likeToStringPair(args)
			if err != nil {
				return nil, err
			}
			return Bool(strings.Contains(s, needle)), nil
		})
	case "hasPrefix":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, prefix, err := likeToStringPair(args)
			if err != nil {
				return nil, err
			}
			return Bool(strings.HasPrefix(s, prefix)), nil
		})
	case "hasSuffix":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, suffix, err := likeToStringPair(args)
			if err != nil {
				return nil, err
			}
			return Bool(strings.HasSuffix(s, suffix)), nil
		})
	case "_firstIndexOf":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, needle, err := likeToStringPair(args)
			if err != nil {
				return nil, err
			}
			byteIdx := strings.Index(s, needle)
			if byteIdx == -1 {
				return Int(-1), nil
			}
			return Int(utf8.RuneCountInString(s[:byteIdx])), nil
		})
	case "_lastIndexOf":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, needle, err := likeToStringPair(args)
			if err != nil {
				return nil, err
			}
			byteIdx := strings.LastIndex(s, needle)
			if byteIdx == -1 {
				return Int(-1), nil
			}
			return Int(utf8.RuneCountInString(s[:byteIdx])), nil
		})
	case "replace":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, target, replacement, err := likeToStringTriple(args)
			if err != nil {
				return nil, err
			}
			return String(strings.ReplaceAll(s, target, replacement)), nil
		})
	case "replaceFirst":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, target, replacement, err := likeToStringTriple(args)
			if err != nil {
				return nil, err
			}
			return String(strings.Replace(s, target, replacement, 1)), nil
		})
	case "replaceLast":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, target, replacement, err := likeToStringTriple(args)
			if err != nil {
				return nil, err
			}
			idx := strings.LastIndex(s, target)
			if idx == -1 {
				return String(s), nil
			}
			return String(s[:idx] + replacement + s[idx+len(target):]), nil
		})
	case "split":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, sep, err := likeToStringPair(args)
			if err != nil {
				return nil, err
			}
			parts := strings.Split(s, sep)
			result := make(Array, len(parts))
			for i, p := range parts {
				result[i] = String(p)
			}
			return result, nil
		})
	case "join":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			arr, ok := args[0].(Array)
			if !ok {
				return nil, fmt.Errorf("join expects an Array argument, got %s", TypeName(args[0]))
			}
			sep, err := likeToString(args[1])
			if err != nil {
				return nil, err
			}
			parts := make([]string, len(arr))
			for i, v := range arr {
				s, err := likeToString(v)
				if err != nil {
					return nil, fmt.Errorf("join: element %d: %w", i, err)
				}
				parts[i] = s
			}
			return String(strings.Join(parts, sep)), nil
		})
	case "concat":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			arr, ok := args[0].(Array)
			if !ok {
				return nil, fmt.Errorf("concat expects an Array argument, got %s", TypeName(args[0]))
			}
			var b strings.Builder
			for i, v := range arr {
				s, err := likeToString(v)
				if err != nil {
					return nil, fmt.Errorf("concat: element %d: %w", i, err)
				}
				b.WriteString(s)
			}
			return String(b.String()), nil
		})
	case "repeat":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, err := likeToString(args[0])
			if err != nil {
				return nil, err
			}
			n, ok := args[1].(Int)
			if !ok {
				return nil, fmt.Errorf("repeat expects an Int count, got %s", TypeName(args[1]))
			}
			if n < 0 {
				return nil, fmt.Errorf("repeat count must be non-negative, got %d", n)
			}
			return String(strings.Repeat(s, int(n))), nil
		})
	case "toUpper":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			switch v := args[0].(type) {
			case Char:
				return Char(unicode.ToUpper(rune(v))), nil
			case String:
				return String(strings.ToUpper(string(v))), nil
			default:
				return nil, fmt.Errorf("toUpper expects a Char or String argument, got %s", TypeName(args[0]))
			}
		})
	case "toLower":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			switch v := args[0].(type) {
			case Char:
				return Char(unicode.ToLower(rune(v))), nil
			case String:
				return String(strings.ToLower(string(v))), nil
			default:
				return nil, fmt.Errorf("toLower expects a Char or String argument, got %s", TypeName(args[0]))
			}
		})
	case "trim":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, err := likeToString(args[0])
			if err != nil {
				return nil, err
			}
			return String(strings.TrimSpace(s)), nil
		})
	case "trimPrefix":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, prefix, err := likeToStringPair(args)
			if err != nil {
				return nil, err
			}
			return String(strings.TrimPrefix(s, prefix)), nil
		})
	case "trimSuffix":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, suffix, err := likeToStringPair(args)
			if err != nil {
				return nil, err
			}
			return String(strings.TrimSuffix(s, suffix)), nil
		})
	case "quote":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, err := likeToString(args[0])
			if err != nil {
				return nil, err
			}
			return String(strconv.Quote(s)), nil
		})
	case "unquote":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			s, err := likeToString(args[0])
			if err != nil {
				return nil, err
			}
			// Only the two literal forms Zirric itself has. Go's backquoted raw
			// strings are not one of them, so strconv.Unquote never sees them.
			if len(s) < 2 || (s[0] != '"' && s[0] != '\'') || s[len(s)-1] != s[0] {
				return ResultErr(caller, String(fmt.Sprintf("not a quoted literal: %s", strconv.Quote(s))))
			}
			unquoted, err := strconv.Unquote(s)
			if err != nil {
				return ResultErr(caller, String(fmt.Sprintf("invalid quoted literal %s: %v", strconv.Quote(s), err)))
			}
			return ResultOk(caller, String(unquoted))
		})
	}
	return nil
}

// likeToString normalizes a Zirric Like value (Char or String) to a Go string.
func likeToString(v RuntimeValue) (string, error) {
	switch v := v.(type) {
	case Char:
		return string(rune(v)), nil
	case String:
		return string(v), nil
	default:
		return "", fmt.Errorf("expected a Char or String argument, got %s", TypeName(v))
	}
}

// everyRune reports whether v is non-empty text in which every character satisfies pred.
func everyRune(v RuntimeValue, pred func(rune) bool) (RuntimeValue, error) {
	s, err := likeToString(v)
	if err != nil {
		return nil, err
	}
	if s == "" {
		return Bool(false), nil
	}
	for _, r := range s {
		if !pred(r) {
			return Bool(false), nil
		}
	}
	return Bool(true), nil
}

func likeToStringPair(args []RuntimeValue) (string, string, error) {
	a, err := likeToString(args[0])
	if err != nil {
		return "", "", err
	}
	b, err := likeToString(args[1])
	if err != nil {
		return "", "", err
	}
	return a, b, nil
}

func likeToStringTriple(args []RuntimeValue) (string, string, string, error) {
	a, b, err := likeToStringPair(args)
	if err != nil {
		return "", "", "", err
	}
	c, err := likeToString(args[2])
	if err != nil {
		return "", "", "", err
	}
	return a, b, c, nil
}
