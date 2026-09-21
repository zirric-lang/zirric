package runtime

import (
	"fmt"
	"io"
	"os"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
)

var _ ExternPlugin = &FSPlugin{}

// FSPlugin provides runtime bindings for the fs module's extern declarations.
// Filesystems are billy filesystems, so an in-memory one and the host's own differ only in which implementation is wrapped.
type FSPlugin struct{}

func (*FSPlugin) Module() string { return "fs" }

// Bind implements ExternPlugin.
func (*FSPlugin) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "memory":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return MakeFileSystem(caller, memfs.New())
		})
	}
	return nil
}

// MakeFileSystem wraps a billy filesystem as an fs.FileSystem.
func MakeFileSystem(caller VMCaller, bfs billy.Filesystem) (RuntimeValue, error) {
	fields := map[string]RuntimeValue{
		"readFile": MakeNativeFunc("readFile", 1, func(c VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			path, err := fsPath("readFile", args[0])
			if err != nil {
				return nil, err
			}
			content, readErr := readWholeFile(bfs, path)
			if readErr != nil {
				return ResultErr(c, String(readErr.Error()))
			}
			return ResultOk(c, Binary(content))
		}),
		"writeFile": MakeNativeFunc("writeFile", 2, func(c VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			path, err := fsPath("writeFile", args[0])
			if err != nil {
				return nil, err
			}
			content, ok := args[1].(Binary)
			if !ok {
				return nil, fmt.Errorf("writeFile expects Binary content, got %s", TypeName(args[1]))
			}
			written, writeErr := writeWholeFile(bfs, path, content)
			if writeErr != nil {
				return ResultErr(c, String(writeErr.Error()))
			}
			return ResultOk(c, Int(written))
		}),
		"open": MakeNativeFunc("open", 1, func(c VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return openAsFile(c, args[0], bfs.Open)
		}),
		"create": MakeNativeFunc("create", 1, func(c VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return openAsFile(c, args[0], bfs.Create)
		}),
		"exists": MakeNativeFunc("exists", 1, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			path, err := fsPath("exists", args[0])
			if err != nil {
				return nil, err
			}
			_, statErr := bfs.Stat(path)
			return Bool(statErr == nil), nil
		}),
		"remove": MakeNativeFunc("remove", 1, func(c VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			path, err := fsPath("remove", args[0])
			if err != nil {
				return nil, err
			}
			if removeErr := bfs.Remove(path); removeErr != nil {
				return ResultErr(c, String(removeErr.Error()))
			}
			return ResultOk(c, Void{})
		}),
		"move": MakeNativeFunc("move", 2, func(c VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			from, err := fsPath("move", args[0])
			if err != nil {
				return nil, err
			}
			to, err := fsPath("move", args[1])
			if err != nil {
				return nil, err
			}
			if renameErr := bfs.Rename(from, to); renameErr != nil {
				return ResultErr(c, String(renameErr.Error()))
			}
			return ResultOk(c, Void{})
		}),
		"list": MakeNativeFunc("list", 1, func(c VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			path, err := fsPath("list", args[0])
			if err != nil {
				return nil, err
			}
			infos, readErr := bfs.ReadDir(path)
			if readErr != nil {
				return ResultErr(c, String(readErr.Error()))
			}
			entries := make(Array, 0, len(infos))
			for _, info := range infos {
				entry, entryErr := makeEntry(c, info)
				if entryErr != nil {
					return nil, entryErr
				}
				entries = append(entries, entry)
			}
			return ResultOk(c, entries)
		}),
		"mkdirAll": MakeNativeFunc("mkdirAll", 1, func(c VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			path, err := fsPath("mkdirAll", args[0])
			if err != nil {
				return nil, err
			}
			if mkErr := bfs.MkdirAll(path, 0o755); mkErr != nil {
				return ResultErr(c, String(mkErr.Error()))
			}
			return ResultOk(c, Void{})
		}),
		"cd": MakeNativeFunc("cd", 1, func(c VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			path, err := fsPath("cd", args[0])
			if err != nil {
				return nil, err
			}
			rooted, chrootErr := bfs.Chroot(path)
			if chrootErr != nil {
				return ResultErr(c, String(chrootErr.Error()))
			}
			nested, nestedErr := MakeFileSystem(c, rooted)
			if nestedErr != nil {
				return nil, nestedErr
			}
			return ResultOk(c, nested)
		}),
		"root": MakeNativeFunc("root", 0, func(_ VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			return String(bfs.Root()), nil
		}),
	}
	return MakeDataValueNamed(caller, "fs", "FileSystem", fields)
}

func openAsFile(caller VMCaller, pathArg RuntimeValue, opener func(string) (billy.File, error)) (RuntimeValue, error) {
	path, err := fsPath("open", pathArg)
	if err != nil {
		return nil, err
	}
	handle, openErr := opener(path)
	if openErr != nil {
		return ResultErr(caller, String(openErr.Error()))
	}
	file, fileErr := makeFile(caller, handle)
	if fileErr != nil {
		return nil, fileErr
	}
	return ResultOk(caller, file)
}

func makeFile(caller VMCaller, handle billy.File) (RuntimeValue, error) {
	fields := map[string]RuntimeValue{
		"readFrom": MakeNativeFunc("readFrom", 1, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			length, ok := args[0].(Int)
			if !ok {
				return nil, fmt.Errorf("read expects an Int length, got %s", TypeName(args[0]))
			}
			buf := make([]byte, int(length))
			n, err := handle.Read(buf)
			if err != nil && err != io.EOF {
				return nil, fmt.Errorf("read: %w", err)
			}
			return Binary(buf[:n]), nil
		}),
		"writeTo": MakeNativeFunc("writeTo", 1, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			buf, ok := args[0].(Binary)
			if !ok {
				return nil, fmt.Errorf("write expects Binary, got %s", TypeName(args[0]))
			}
			n, err := handle.Write([]byte(buf))
			if err != nil {
				return nil, fmt.Errorf("write: %w", err)
			}
			return Int(n), nil
		}),
		"closeWith": MakeNativeFunc("closeWith", 0, func(c VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			if err := handle.Close(); err != nil {
				return ResultErr(c, String(err.Error()))
			}
			return ResultOk(c, Void{})
		}),
	}
	return MakeDataValueNamed(caller, "fs", "File", fields)
}

func makeEntry(caller VMCaller, info os.FileInfo) (RuntimeValue, error) {
	return MakeDataValueNamed(caller, "fs", "Entry", map[string]RuntimeValue{
		"name":  String(info.Name()),
		"isDir": Bool(info.IsDir()),
		"size":  Int(info.Size()),
	})
}

func readWholeFile(bfs billy.Filesystem, path string) ([]byte, error) {
	handle, err := bfs.Open(path)
	if err != nil {
		return nil, err
	}
	// A close failure after the bytes are already read says nothing about the contents, unlike the write path where it can mean the data never landed.
	defer func() { _ = handle.Close() }()
	return io.ReadAll(handle)
}

func writeWholeFile(bfs billy.Filesystem, path string, content []byte) (int, error) {
	handle, err := bfs.Create(path)
	if err != nil {
		return 0, err
	}
	written, writeErr := handle.Write(content)
	if closeErr := handle.Close(); closeErr != nil && writeErr == nil {
		return written, closeErr
	}
	return written, writeErr
}

func fsPath(fnName string, v RuntimeValue) (string, error) {
	s, ok := v.(String)
	if !ok {
		return "", fmt.Errorf("%s expects a String path, got %s", fnName, TypeName(v))
	}
	return string(s), nil
}
