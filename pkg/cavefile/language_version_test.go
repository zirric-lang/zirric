package cavefile_test

import (
	"errors"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

func TestCheckLanguageVersion(t *testing.T) {
	tests := []struct {
		name      string
		predicate string
		toolchain version.Version
		wantErr   bool
	}{
		{"no predicate", "", version.Parse("0.1.0"), false},
		{"unknown toolchain", ">=9.0.0", nil, false},
		{"satisfied", "^0.1.0", version.Parse("0.1.2"), false},
		{"unsatisfied", ">=9.0.0", version.Parse("0.1.0"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cavefile.CheckLanguageVersion(cavefile.Package{Name: "app", LanguageVersion: tt.predicate}, tt.toolchain)
			if tt.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				return
			}
			var mismatch *cavefile.LanguageVersionError
			if !errors.As(err, &mismatch) {
				t.Fatalf("err = %T, want *cavefile.LanguageVersionError", err)
			}
			if mismatch.Package != "app" || mismatch.Predicate != tt.predicate {
				t.Errorf("err = %+v, want it to name the package and its predicate", mismatch)
			}
		})
	}
}
