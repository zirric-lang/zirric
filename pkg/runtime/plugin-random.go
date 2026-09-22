package runtime

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"
	mathrand "math/rand"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ ExternPlugin = &RandomPlugin{}

// RandomPlugin provides runtime bindings for the random module's extern declarations.
// Only the seeded source is built here: the two drawing on the host belong to os, which is where everything reaching the machine enters.
// They share makeSource below, so anything taking a random.Source works with any of them.
type RandomPlugin struct{}

func (*RandomPlugin) Module() string { return "random" }

// Bind implements ExternPlugin.
func (*RandomPlugin) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "seeded":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			seed, ok := args[0].(Int)
			if !ok {
				return nil, fmt.Errorf("seeded expects an Int seed, got %s", TypeName(args[0]))
			}
			return makeSource(caller, &pseudoRandom{r: mathrand.New(mathrand.NewSource(int64(seed)))})
		})
	}
	return nil
}

// randomBits is what the sources have in common: everything the module exposes is derived from these.
type randomBits interface {
	intn(n int64) (int64, error)
	float() (float64, error)
	read(p []byte) error
}

// pseudoRandom is a seeded generator: fast, reproducible, and unsuitable for anything requiring secrecy.
type pseudoRandom struct{ r *mathrand.Rand }

func (p *pseudoRandom) intn(n int64) (int64, error) { return p.r.Int63n(n), nil }
func (p *pseudoRandom) float() (float64, error)     { return p.r.Float64(), nil }
func (p *pseudoRandom) read(b []byte) error {
	_, err := p.r.Read(b)
	return err
}

// cryptoRandom draws from the operating system's cryptographic generator.
type cryptoRandom struct{}

func (cryptoRandom) intn(n int64) (int64, error) {
	v, err := cryptorand.Int(cryptorand.Reader, big.NewInt(n))
	if err != nil {
		return 0, err
	}
	return v.Int64(), nil
}

func (c cryptoRandom) float() (float64, error) {
	var b [8]byte
	if err := c.read(b[:]); err != nil {
		return 0, err
	}
	// The top 53 bits are used because that is what a float64 can represent exactly, matching how math/rand builds its own.
	return float64(binary.BigEndian.Uint64(b[:])>>11) / (1 << 53), nil
}

func (cryptoRandom) read(b []byte) error {
	_, err := cryptorand.Read(b)
	return err
}

func makeSource(caller VMCaller, bits randomBits) (RuntimeValue, error) {
	fields := map[string]RuntimeValue{
		"int": MakeNativeFunc("int", 1, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			n, ok := args[0].(Int)
			if !ok {
				return nil, fmt.Errorf("int expects an Int bound, got %s", TypeName(args[0]))
			}
			if n <= 0 {
				return nil, fmt.Errorf("int expects a bound above zero, got %d", n)
			}
			v, err := bits.intn(int64(n))
			if err != nil {
				return nil, fmt.Errorf("random int: %w", err)
			}
			return Int(v), nil
		}),
		"float": MakeNativeFunc("float", 0, func(_ VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			v, err := bits.float()
			if err != nil {
				return nil, fmt.Errorf("random float: %w", err)
			}
			return Float(v), nil
		}),
		"bytes": MakeNativeFunc("bytes", 1, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			n, ok := args[0].(Int)
			if !ok {
				return nil, fmt.Errorf("bytes expects an Int count, got %s", TypeName(args[0]))
			}
			if n < 0 {
				return nil, fmt.Errorf("bytes expects a count of zero or more, got %d", n)
			}
			buf := make([]byte, int(n))
			if err := bits.read(buf); err != nil {
				return nil, fmt.Errorf("random bytes: %w", err)
			}
			return Binary(buf), nil
		}),
		"bool": MakeNativeFunc("bool", 0, func(_ VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			v, err := bits.intn(2)
			if err != nil {
				return nil, fmt.Errorf("random bool: %w", err)
			}
			return Bool(v == 1), nil
		}),
	}
	return MakeDataValueNamed(caller, "random", "Source", fields)
}
