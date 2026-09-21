package codefmt

import "testing"

// formatExpr formats a one-line snippet, which is enough to pin spacing rules.
func formatExpr(t *testing.T, src string) string {
	t.Helper()
	out, err := String("spacing.zirr", src, DefaultOptions())
	if err != nil {
		t.Fatalf("format %q: %v", src, err)
	}
	return out
}

func TestSpacing(t *testing.T) {
	tests := []struct{ in, want string }{
		{"const a = 1+2", "const a = 1 + 2"},
		{"const a = 1 - 2", "const a = 1 - 2"},
		{"const a = -1", "const a = -1"},
		{"const a = f( -1 )", "const a = f(-1)"},
		{"const a = [1,-2]", "const a = [1, -2]"},
		{"const a = b * -c", "const a = b * -c"},
		{"const a = -b - -c", "const a = -b - -c"},
		{"const a = !ok", "const a = !ok"},
		{"const a = b . c", "const a = b.c"},
		{"const a = b [ 0 ]", "const a = b[0]"},
		{"const a = [ 1 , 2 ]", "const a = [1, 2]"},
		{"const a : Int = 1", "const a: Int = 1"},
		{"const a = [ \"k\" : 1 ]", "const a = [\"k\": 1]"},
		{"@Attr ( )\nattr X {}", "@Attr()\nattr X {}"},
		{"@a.B ( 1 )\nattr X {}", "@a.B(1)\nattr X {}"},
		{"fn f ( a , b ) { a }", "fn f(a, b) { a }"},
		{"attr X { }", "attr X {}"},
		{"const a = fn ( x ) -> Int { x }", "const a = fn(x) -> Int { x }"},
		{"const a = b is @Iterable", "const a = b is @Iterable"},
		{"fn f() { a += 1 }", "fn f() { a += 1 }"},
		{"const a = b&&c", "const a = b && c"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := formatExpr(t, tt.in); got != tt.want+"\n" {
				t.Errorf("want %q, got %q", tt.want+"\n", got)
			}
		})
	}
}

// Removing a space must never let two tokens lex as one.
func TestAdjacencyGuard(t *testing.T) {
	tests := []struct {
		left, right string
		want        bool
	}{
		{"a <", "- b", true},
		{"a <", "= b", true},
		{"a", "b", false},
		{"a +", "+ b", false}, // Zirric has no "++"
		{"a /", "/ b", true},
		{"", "a", false},
		{"a", "", false},
	}
	for _, tt := range tests {
		if got := wouldGlue(tt.left, tt.right); got != tt.want {
			t.Errorf("wouldGlue(%q, %q) = %v, want %v", tt.left, tt.right, got, tt.want)
		}
	}
}
