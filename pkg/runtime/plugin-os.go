package runtime

import (
	"fmt"
	"io"
	mathrand "math/rand"
	"os"
	"sync"
	gotime "time"

	"github.com/go-git/go-billy/v5/osfs"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

// processStart anchors monotonic readings.
// Go exposes its monotonic clock only as the difference between two time.Time values, so readings are taken relative to a moment captured once at startup.
var processStart = gotime.Now()

var _ ExternPlugin = &OSPlugin{}

// OSPlugin provides runtime bindings for the os module extern declarations.
// The two host-backed random sources build a random.Source through makeSource, the same one random.seeded uses.
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
			return makeReadStreamOf(caller, sharedStdin())
		})
	case "stderr":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return MakeWriteStream(caller, os.Stderr)
		})
	case "exit":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			code, ok := args[0].(Int)
			if !ok {
				return nil, fmt.Errorf("exit expects Int code, got %s", TypeName(args[0]))
			}
			os.Exit(int(code))
			return Void{}, nil
		})
	case "env":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			key, ok := args[0].(String)
			if !ok {
				return nil, fmt.Errorf("env expects String key, got %s", TypeName(args[0]))
			}
			return String(os.Getenv(string(key))), nil
		})
	case "fs":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return MakeFileSystem(caller, osfs.New("/"))
		})
	case "cwd":
		return MakeExternFunc(decl, func(_ VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			dir, err := os.Getwd()
			if err != nil {
				return nil, err
			}
			return String(dir), nil
		})
	case "fastRandom":
		return MakeExternFunc(decl, func(caller VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			return makeSource(caller, &pseudoRandom{r: mathrand.New(mathrand.NewSource(gotime.Now().UnixNano()))})
		})
	case "strongRandom":
		return MakeExternFunc(decl, func(caller VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			return makeSource(caller, cryptoRandom{})
		})
	case "timer":
		return MakeExternFunc(decl, func(caller VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			return makeHostTimer(caller)
		})
	case "_nowTimestamp":
		return MakeExternFunc(decl, func(_ VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			return Timestamp(gotime.Now().UnixNano()), nil
		})
	case "_nowInstant":
		return MakeExternFunc(decl, func(_ VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			return Instant(gotime.Since(processStart)), nil
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
	writeFn := MakeHostFunc("writeTo", 1, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
		buf, ok := args[0].(Binary)
		if !ok {
			return nil, fmt.Errorf("write expects Binary, got %s", TypeName(args[0]))
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
// A hostReader owns the reads, so a routine cancelled while waiting strands no bytes.
func MakeReadStream(caller VMCaller, r io.Reader) (RuntimeValue, error) {
	return makeReadStreamOf(caller, newHostReader(r))
}

// Shared, so two os.stdin() calls read from one queue of bytes rather than two.
var (
	stdinOnce   sync.Once
	stdinShared *hostReader
)

func sharedStdin() *hostReader {
	stdinOnce.Do(func() { stdinShared = newHostReader(os.Stdin) })
	return stdinShared
}

func makeReadStreamOf(caller VMCaller, hr *hostReader) (RuntimeValue, error) {
	readFn := MakeSwitchingFunc("readFrom", 1, func(c VMCaller, args []RuntimeValue) (RuntimeValue, error) {
		length, ok := args[0].(Int)
		if !ok {
			return nil, fmt.Errorf("read expects Int, got %s", TypeName(args[0]))
		}
		return hr.read(c, int(length))
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
		return nil, fmt.Errorf("io.%s is %s, not a data type", typeName, TypeName(member))
	}
	return MakeDataValue(dataType, []RuntimeValue{fn}), nil
}

// makeHostTimer builds the co.Timer that really waits.
// Each armed timer is registered, so routines waiting for one are not mistaken for a deadlock.
func makeHostTimer(caller VMCaller) (RuntimeValue, error) {
	if caller == nil {
		return nil, fmt.Errorf("os.timer can only be built while a VM is running")
	}
	after := MakeNativeFunc("after", 1, func(c VMCaller, args []RuntimeValue) (RuntimeValue, error) {
		span, ok := args[0].(Duration)
		if !ok {
			return nil, fmt.Errorf("after expects a Duration, got %s", TypeName(args[0]))
		}
		sched := c.Routines()
		ch := sched.NewChannel(1)
		sched.ArmExternal()
		gotime.AfterFunc(gotime.Duration(span), func() {
			ch.Deliver(Void{})
			_ = ch.Close()
			sched.DisarmExternal()
		})
		return ch, nil
	})
	return MakeDataValueNamed(caller, "co", "Timer", map[string]RuntimeValue{"after": after})
}
