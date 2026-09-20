package runtime

import (
	"fmt"
	"path"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ ExternPlugin = &PathsPlugin{}

// PathsPlugin provides runtime bindings for the paths module's extern declarations.
// These operate on slash-separated paths rather than the host's separator, matching the filesystem abstraction rather than the machine the program happens to run on.
type PathsPlugin struct{}

func (*PathsPlugin) Module() string { return "paths" }

// Bind implements ExternPlugin.
func (*PathsPlugin) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "join":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			parts, err := pathStrings(args[0])
			if err != nil {
				return nil, err
			}
			return String(path.Join(parts...)), nil
		})
	case "clean":
		return makePathUnary(decl, path.Clean)
	case "base":
		return makePathUnary(decl, path.Base)
	case "dir":
		return makePathUnary(decl, path.Dir)
	case "ext":
		return makePathUnary(decl, path.Ext)
	case "stem":
		return makePathUnary(decl, func(p string) string {
			base := path.Base(p)
			return strings.TrimSuffix(base, path.Ext(base))
		})
	case "isAbs":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			p, err := pathString("isAbs", args[0])
			if err != nil {
				return nil, err
			}
			return Bool(path.IsAbs(p)), nil
		})
	case "segments":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			p, err := pathString("segments", args[0])
			if err != nil {
				return nil, err
			}
			result := make(Array, 0)
			for _, segment := range strings.Split(path.Clean(p), "/") {
				if segment == "" || segment == "." {
					continue
				}
				result = append(result, String(segment))
			}
			return result, nil
		})
	case "match":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			pattern, err := pathString("match", args[0])
			if err != nil {
				return nil, err
			}
			p, err := pathString("match", args[1])
			if err != nil {
				return nil, err
			}
			matched, matchErr := path.Match(pattern, p)
			if matchErr != nil {
				return ResultErr(caller, String(fmt.Sprintf("malformed pattern %q", pattern)))
			}
			return ResultOk(caller, Bool(matched))
		})
	}
	return nil
}

func makePathUnary(decl *ast.Symbol, apply func(string) string) RuntimeValue {
	return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
		p, err := pathString(decl.Name, args[0])
		if err != nil {
			return nil, err
		}
		return String(apply(p)), nil
	})
}

func pathString(fnName string, v RuntimeValue) (string, error) {
	s, ok := v.(String)
	if !ok {
		return "", fmt.Errorf("%s expects a String path, got %T", fnName, v)
	}
	return string(s), nil
}

func pathStrings(v RuntimeValue) ([]string, error) {
	arr, ok := v.(Array)
	if !ok {
		return nil, fmt.Errorf("join expects an Array of String, got %T", v)
	}
	parts := make([]string, len(arr))
	for i, item := range arr {
		s, err := pathString("join", item)
		if err != nil {
			return nil, err
		}
		parts[i] = s
	}
	return parts, nil
}
