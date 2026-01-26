package registry

import "testing"

func TestCanonicalizeModuleSource(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   LogicalURI
	}{
		{
			name:   "https with git suffix",
			source: "https://code.knabel.dev/zirric-lang/zirric.git",
			want:   "code.knabel.dev.zirric_lang.zirric",
		},
		{
			name:   "https with trailing slash",
			source: "https://code.knabel.dev/zirric-lang/zirric/",
			want:   "code.knabel.dev.zirric_lang.zirric",
		},
		{
			name:   "already normalized",
			source: "code.knabel.dev/zirric-lang/zirric",
			want:   "code.knabel.dev.zirric_lang.zirric",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanonicalizeModuleSource(tt.source)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestJoinModuleURI(t *testing.T) {
	tests := []struct {
		name       string
		base       LogicalURI
		modulePath string
		want       LogicalURI
	}{
		{
			name:       "nested module path",
			base:       "code.knabel.dev.zirric_lang.zirric",
			modulePath: "future/reflect",
			want:       "code.knabel.dev.zirric_lang.zirric.future.reflect",
		},
		{
			name:       "single segment",
			base:       "code.knabel.dev.zirric_lang.zirric",
			modulePath: "future",
			want:       "code.knabel.dev.zirric_lang.zirric.future",
		},
		{
			name:       "empty module path",
			base:       "code.knabel.dev.zirric_lang.zirric",
			modulePath: "",
			want:       "code.knabel.dev.zirric_lang.zirric",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JoinModuleURI(tt.base, tt.modulePath)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
