package runtime

import (
	"fmt"
	"io"
	"os"

	"github.com/go-git/go-billy/v5/osfs"

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
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return MakeWriteStream(caller, os.Stdout)
		})
	case "stdin":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return MakeReadStream(caller, os.Stdin)
		})
	case "stderr":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return MakeWriteStream(caller, os.Stderr)
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
	case "fs":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return MakeFileSystem(caller, osfs.New("/"))
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

// MakeWriteStream wraps a Go writer as an io.WriteStream.
// The type is resolved through the caller rather than assembled by hand, because a hand-built DataValue carries no attributes and would therefore not satisfy @Writer.
func MakeWriteStream(caller VMCaller, w io.Writer) (RuntimeValue, error) {
	writeFn := MakeNativeFunc("writeTo", 1, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
		buf, ok := args[0].(Binary)
		if !ok {
			return nil, fmt.Errorf("write expects Binary, got %T", args[0])
		}
		n, err := w.Write([]byte(buf))
		if err != nil {
			return nil, fmt.Errorf("write: %w", err)
		}
		return Int(n), nil
	})
	return makeStreamValue(caller, "WriteStream", writeFn)
}

// MakeReadStream wraps a Go reader as an io.ReadStream, yielding an empty Binary at the end of the stream.
func MakeReadStream(caller VMCaller, r io.Reader) (RuntimeValue, error) {
	readFn := MakeNativeFunc("readFrom", 1, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
		length, ok := args[0].(Int)
		if !ok {
			return nil, fmt.Errorf("read expects Int, got %T", args[0])
		}
		buf := make([]byte, int(length))
		n, err := r.Read(buf)
		if err == io.EOF {
			return Binary(buf[:n]), nil
		}
		if err != nil {
			return nil, fmt.Errorf("read: %w", err)
		}
		return Binary(buf[:n]), nil
	})
	return makeStreamValue(caller, "ReadStream", readFn)
}

func makeStreamValue(caller VMCaller, typeName string, fn RuntimeValue) (RuntimeValue, error) {
	if caller == nil {
		return nil, fmt.Errorf("io.%s can only be built while a VM is running", typeName)
	}
	member, err := caller.ResolveModuleMember("io", typeName)
	if err != nil {
		return nil, err
	}
	dataType, ok := member.(*DataType)
	if !ok {
		return nil, fmt.Errorf("io.%s is %T, not a data type", typeName, member)
	}
	return MakeDataValue(dataType, []RuntimeValue{fn}), nil
}
