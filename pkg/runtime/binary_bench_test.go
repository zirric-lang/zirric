package runtime

import (
	"testing"
)

// Compares Binary ([]byte, unboxed) against an Array of Byte (each element boxed as a RuntimeValue interface) for the operations a byte sequence is actually used for: building one up (via raw Go append, and via the `append` extern function Zirric code actually calls), indexed access, iteration, and producing its hex Inspect() representation. The point isn't absolute throughput — it's whether boxing every byte as an interface value (what [Byte] necessarily does, since Array is []RuntimeValue) costs enough to justify Binary existing as its own type alongside [Byte].

var byteSizes = []int{16, 256, 4096}

func makeBinary(n int) Binary {
	b := make(Binary, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}

func makeByteArray(n int) Array {
	a := make(Array, n)
	for i := range a {
		a[i] = Byte(byte(i))
	}
	return a
}

func BenchmarkConstruct(b *testing.B) {
	for _, n := range byteSizes {
		b.Run("Binary", benchConstructBinary(n))
		b.Run("ByteArray", benchConstructByteArray(n))
	}
}

func benchConstructBinary(n int) func(b *testing.B) {
	return func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			bin := make(Binary, 0, n)
			for j := 0; j < n; j++ {
				bin = append(bin, byte(j))
			}
			sink = bin
		}
	}
}

func benchConstructByteArray(n int) func(b *testing.B) {
	return func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			arr := make(Array, 0, n)
			for j := 0; j < n; j++ {
				arr = append(arr, Byte(byte(j)))
			}
			sink = arr
		}
	}
}

// BenchmarkAppendExtern benchmarks the actual `append` extern function (pkg/runtime/prelude.go's Prelude.Bind("append")) that Zirric code calls — not just raw Go append (already exercised, incidentally, by BenchmarkConstruct) — since both Binary and Array accept it, per its "append expects an Array argument" special-case for Binary.
func BenchmarkAppendExtern(b *testing.B) {
	var p Prelude
	appendFn := p.Bind(nil, nil, makeExternFuncSymbol("append", 2)).(*ExternFunc)

	for _, n := range byteSizes {
		b.Run("Binary", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var v RuntimeValue = Binary{}
				for j := 0; j < n; j++ {
					var err error
					v, err = appendFn.Impl(nil, []RuntimeValue{v, Byte(byte(j))})
					if err != nil {
						b.Fatal(err)
					}
				}
				sink = v
			}
		})

		b.Run("ByteArray", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var v RuntimeValue = Array{}
				for j := 0; j < n; j++ {
					var err error
					v, err = appendFn.Impl(nil, []RuntimeValue{v, Byte(byte(j))})
					if err != nil {
						b.Fatal(err)
					}
				}
				sink = v
			}
		})
	}
}

func BenchmarkIndex(b *testing.B) {
	for _, n := range byteSizes {
		bin := makeBinary(n)
		arr := makeByteArray(n)

		b.Run("Binary", func(b *testing.B) {
			b.ReportAllocs()
			var sum int
			for i := 0; i < b.N; i++ {
				sum = 0
				for j := 0; j < n; j++ {
					sum += int(bin[j])
				}
			}
			sinkInt = sum
		})

		b.Run("ByteArray", func(b *testing.B) {
			b.ReportAllocs()
			var sum int
			for i := 0; i < b.N; i++ {
				sum = 0
				for j := 0; j < n; j++ {
					sum += int(arr[j].(Byte))
				}
			}
			sinkInt = sum
		})
	}
}

func BenchmarkIterate(b *testing.B) {
	for _, n := range byteSizes {
		bin := makeBinary(n)
		arr := makeByteArray(n)

		b.Run("Binary", func(b *testing.B) {
			b.ReportAllocs()
			var sum int
			for i := 0; i < b.N; i++ {
				sum = 0
				for _, v := range bin {
					sum += int(v)
				}
			}
			sinkInt = sum
		})

		b.Run("ByteArray", func(b *testing.B) {
			b.ReportAllocs()
			var sum int
			for i := 0; i < b.N; i++ {
				sum = 0
				for _, v := range arr {
					sum += int(v.(Byte))
				}
			}
			sinkInt = sum
		})
	}
}

func BenchmarkInspect(b *testing.B) {
	for _, n := range byteSizes {
		bin := makeBinary(n)
		arr := makeByteArray(n)

		b.Run("Binary", func(b *testing.B) {
			b.ReportAllocs()
			var s string
			for i := 0; i < b.N; i++ {
				s = bin.Inspect()
			}
			sinkStr = s
		})

		b.Run("ByteArray", func(b *testing.B) {
			b.ReportAllocs()
			var s string
			for i := 0; i < b.N; i++ {
				s = arr.Inspect()
			}
			sinkStr = s
		})
	}
}

// Package-level sinks so the compiler can't optimize the benchmarked work away.
var (
	sink    RuntimeValue
	sinkInt int
	sinkStr string
)
