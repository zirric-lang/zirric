package runtime

import (
	"fmt"
	"os"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ ExternPlugin = &OSPlugin{}

// OSPlugin provides runtime bindings for the os module extern declarations.
type OSPlugin struct{}

func (*OSPlugin) Module() string { return "os" }

// Bind implements ExternPlugin.
func (*OSPlugin) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "stdout":
		writerSym := ctx.ResolveModuleSymbol("io", "Writer")
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return makeWriterValue(writerSym, os.Stdout)
		})
	case "stdin":
		readerSym := ctx.ResolveModuleSymbol("io", "Reader")
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return makeReaderValue(readerSym, os.Stdin)
		})
	case "stderr":
		writerSym := ctx.ResolveModuleSymbol("io", "Writer")
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return makeWriterValue(writerSym, os.Stderr)
		})
	case "exit":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			code, ok := args[0].(Int)
			if !ok {
				return nil, fmt.Errorf("exit expects Int code, got %T", args[0])
			}
			os.Exit(int(code))
			return Void{}, nil
		})
	case "env":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			key, ok := args[0].(String)
			if !ok {
				return nil, fmt.Errorf("env expects String key, got %T", args[0])
			}
			return String(os.Getenv(string(key))), nil
		})
	case "args":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			result := make([]RuntimeValue, len(os.Args))
			for i, arg := range os.Args {
				result[i] = String(arg)
			}
			return Array(result), nil
		})
	}
	return nil
}

func makeWriterValue(writerSym *ast.Symbol, w *os.File) (RuntimeValue, error) {
	if writerSym == nil || writerSym.ConstantId == nil {
		return nil, fmt.Errorf("writer type not resolved")
	}
	writeFn := MakeNativeFunc("write", 1, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
		buf, ok := args[0].(Binary)
		if !ok {
			return nil, fmt.Errorf("write expects Bytes, got %T", args[0])
		}
		n, err := w.Write([]byte(buf))
		if err != nil {
			return nil, fmt.Errorf("write: %w", err)
		}
		return Int(n), nil
	})
	return &DataValue{
		TypeId: TypeId(*writerSym.ConstantId),
		Fields: map[string]int{"write": 0},
		Values: []RuntimeValue{writeFn},
	}, nil
}

func makeReaderValue(readerSym *ast.Symbol, r *os.File) (RuntimeValue, error) {
	if readerSym == nil || readerSym.ConstantId == nil {
		return nil, fmt.Errorf("reader type not resolved")
	}
	readFn := MakeNativeFunc("read", 1, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
		length, ok := args[0].(Int)
		if !ok {
			return nil, fmt.Errorf("read expects Int, got %T", args[0])
		}
		buf := make([]byte, int(length))
		n, err := r.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("read: %w", err)
		}
		return Binary(buf[:n]), nil
	})
	return &DataValue{
		TypeId: TypeId(*readerSym.ConstantId),
		Fields: map[string]int{"read": 0},
		Values: []RuntimeValue{readFn},
	}, nil
}
