package toolchain_test

import (
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/toolchain"
	mv "github.com/metal-stack/v"
)

// A plain `go build` stamps no version, so the running test binary reports none and describes itself as a development build.
func TestUnstampedBuildHasNoVersion(t *testing.T) {
	if got := toolchain.Version(); got != nil {
		t.Errorf("Version() = %v, want nil for an unstamped build", got)
	}
	if got := toolchain.Description(); !strings.HasPrefix(got, "devel") {
		t.Errorf("Description() = %q, want it to start with %q", got, "devel")
	}
}

func TestStampedBuildReportsItsVersion(t *testing.T) {
	previous := mv.Version
	mv.Version = "1.2.3"
	t.Cleanup(func() { mv.Version = previous })

	got := toolchain.Version()
	if got == nil || got.String() != "1.2.3" {
		t.Errorf("Version() = %v, want 1.2.3", got)
	}
}
