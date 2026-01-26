package registry

import "testing"

func TestLogicalURIJoin(t *testing.T) {
	tests := []struct {
		name     string
		base     LogicalURI
		segment  string
		expected string
	}{
		{
			name:     "without trailing slash",
			base:     LogicalURI("foo/bar"),
			segment:  "baz",
			expected: "foo/bar/baz",
		},
		{
			name:     "with trailing slash",
			base:     LogicalURI("foo/bar/"),
			segment:  "baz",
			expected: "foo/bar/baz",
		},
		{
			name:     "empty segment",
			base:     LogicalURI("foo/bar"),
			segment:  "",
			expected: "foo/bar/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			joined := tt.base.Join(tt.segment)
			if string(joined) != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, joined)
			}
		})
	}
}
