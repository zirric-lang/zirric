package embedreg_test

import (
	"context"
	"embed"
	"io/fs"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/embedreg"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

//go:embed testdata/prelude/*.zirr
var preludeFS embed.FS

//go:embed testdata/future/*.zirr testdata/future/**/*.zirr
var futureFS embed.FS

func TestEmbedRegistryDiscoverModules(t *testing.T) {
	preludeSub, err := fs.Sub(preludeFS, "testdata/prelude")
	if err != nil {
		t.Fatalf("sub prelude fs: %v", err)
	}
	futureSub, err := fs.Sub(futureFS, "testdata/future")
	if err != nil {
		t.Fatalf("sub future fs: %v", err)
	}

	provider, err := embedreg.New(
		cavefile.Package{
			Name:   "code.knabel.dev.zirric_lang.zirric",
			Source: "https://code.knabel.dev/zirric-lang/zirric",
		},
		version.Parse("1.2.3"),
		embedreg.FSConfig{Name: "prelude", FS: preludeSub},
		embedreg.FSConfig{Name: "future", FS: futureSub},
	)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	pkgs, err := provider.Discover(context.Background())
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	if pkgs[0].Source() != "https://code.knabel.dev/zirric-lang/zirric" {
		t.Fatalf("unexpected package source: %s", pkgs[0].Source())
	}
	if pkgs[0].Version().String() != "1.2.3" {
		t.Fatalf("unexpected version: %s", pkgs[0].Version())
	}

	resolved, err := pkgs[0].Resolve(context.Background())
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	mods, err := resolved.ResolveModules()
	if err != nil {
		t.Fatalf("resolve modules: %v", err)
	}

	moduleURIs := collectModuleURIs(mods)
	expectModule(t, moduleURIs, "code.knabel.dev.zirric_lang.zirric.prelude")
	expectModule(t, moduleURIs, "code.knabel.dev.zirric_lang.zirric.future")
	expectModule(t, moduleURIs, "code.knabel.dev.zirric_lang.zirric.future.reflect")
	expectModule(t, moduleURIs, "code.knabel.dev.zirric_lang.zirric.future.prelude")
}

func collectModuleURIs(mods []registry.ResolvedModule) map[string]struct{} {
	out := make(map[string]struct{}, len(mods))
	for _, mod := range mods {
		out[string(mod.URI())] = struct{}{}
	}
	return out
}

func expectModule(t *testing.T, mods map[string]struct{}, uri string) {
	t.Helper()
	if _, ok := mods[uri]; !ok {
		t.Fatalf("expected module %s to be discovered", uri)
	}
}
