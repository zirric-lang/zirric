package orchestra_test

import (
	"context"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
)

func TestDefaultStdlibProvider(t *testing.T) {
	provider, err := orchestra.DefaultStdlibProvider()
	if err != nil {
		t.Fatalf("default provider: %v", err)
	}
	pkgs, err := provider.Discover(context.Background())
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	if pkgs[0].Version().String() != "latest" {
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
	expectModule(t, moduleURIs, "code.knabel.dev.zirric_lang.zirric.future.cave")
	expectModule(t, moduleURIs, "code.knabel.dev.zirric_lang.zirric.future.reflect")
	expectModule(t, moduleURIs, "code.knabel.dev.zirric_lang.zirric.future.tasks")
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
