package vm_test

import "testing"

// printableShape is what an interpolation is compiled against. The VM tests run without the real prelude, so @Printable is declared here instead, mirroring prelude/attributes.zirr.
const printableShape = `
attr Printable {
	toString(self: @Printable) -> String
}
`

func TestStringInterpolation(t *testing.T) {
	tests := []vmTestCase{
		{label: "a literal alone", input: printableShape + `"n = \(3)"`, expected: "n = 3"},
		{label: "several parts", input: printableShape + `"\(1) + \(2) = \(1 + 2)"`, expected: "1 + 2 = 3"},
		{label: "a string renders as itself", input: printableShape + `"say \("hi")"`, expected: "say hi"},
		{label: "nested literals", input: printableShape + `"a \("b \(3)") c"`, expected: "a b 3 c"},
		{label: "an escaped backslash is no interpolation", input: printableShape + `"\\(literal)"`, expected: `\(literal)`},
		{label: "every builtin renders", input: printableShape + `"\(1) \(true) \('x')"`, expected: "1 true x"},
		{label: "an expression form", input: printableShape + `"\(if true { "yes" } else { "no" })"`, expected: "yes"},
		{label: "an array renders as one", input: printableShape + `"\([1, 2])"`, expected: "[1, 2]"},
		{
			label:    "a @Printable type decides how it looks",
			input:    printableShape + "@Printable(fn(v) { return \"v\" + v.major })\ndata Version { major: Int }\n\"at \\(Version(2))\"",
			expected: "at v2",
		},
		{
			label:    "parts are evaluated left to right, each once",
			input:    printableShape + "var calls = 0\nfn next() {\n\tcalls = calls + 1\n\treturn calls\n}\n\"\\(next())\\(next())\\(next())\"",
			expected: "123",
		},
	}

	runVmTests(t, tests)
}

// TestInterpolationNeedsPrintable holds the rule the compiler relies on: rendering is @Printable's, so a program without it in scope is refused rather than rendered some other way.
func TestInterpolationNeedsPrintable(t *testing.T) {
	runVmTests(t, []vmTestCase{{
		input: `"\(1)"`,
		err:   "testing:///test/test.zirr:1:1: undefined built-in type: Printable is not in scope, which usually means prelude was not imported",
	}})
}
