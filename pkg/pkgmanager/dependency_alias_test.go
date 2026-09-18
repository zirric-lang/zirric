package pkgmanager

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
)

type stubModule struct {
	uri registry.LogicalURI
}

func (m *stubModule) URI() registry.LogicalURI            { return m.uri }
func (m *stubModule) Sources() ([]registry.Source, error) { return nil, nil }

func modURIs(mods []registry.ResolvedModule) []string {
	out := make([]string, len(mods))
	for i, m := range mods {
		out[i] = string(m.URI())
	}
	return out
}

func TestAliasDependencyNoNameReturnsPackageUnchanged(t *testing.T) {
	pkg := &stubResolvedPackage{source: "local/pkg"}
	got := aliasDependency(pkg, cavefile.Dependency{})
	if got != pkg {
		t.Fatalf("expected the same package instance, got %#v", got)
	}
}

func TestAliasDependencyTreeRebase(t *testing.T) {
	pkg := &stubResolvedPackage{
		resolvedModulesResp: []registry.ResolvedModule{
			&stubModule{uri: "local_package"},
			&stubModule{uri: "local_package.sub"},
		},
	}
	dep := cavefile.Dependency{Package: cavefile.Package{Name: "helpers"}}

	aliased := aliasDependency(pkg, dep)
	mods, err := aliased.ResolveModules()
	if err != nil {
		t.Fatalf("resolve modules: %v", err)
	}
	got := modURIs(mods)
	want := []string{"helpers", "helpers.sub"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestAliasDependencySingleModuleRebase(t *testing.T) {
	pkg := &stubResolvedPackage{
		resolvedModulesResp: []registry.ResolvedModule{
			&stubModule{uri: "prelude"},
			&stubModule{uri: "cave"},
			&stubModule{uri: "io"},
		},
	}
	dep := cavefile.Dependency{Package: cavefile.Package{Name: "caveModule"}, Module: "cave"}

	aliased := aliasDependency(pkg, dep)
	mods, err := aliased.ResolveModules()
	if err != nil {
		t.Fatalf("resolve modules: %v", err)
	}
	got := modURIs(mods)
	if len(got) != 1 || got[0] != "caveModule" {
		t.Fatalf("got %v, want [caveModule]", got)
	}
}

func TestAliasDependencyTreeRebaseSkipsPackagesWithoutACommonRoot(t *testing.T) {
	pkg := &stubResolvedPackage{
		resolvedModulesResp: []registry.ResolvedModule{
			&stubModule{uri: "prelude"},
			&stubModule{uri: "future"},
			&stubModule{uri: "io"},
		},
	}
	dep := cavefile.Dependency{Package: cavefile.Package{Name: "stdlib"}}

	aliased := aliasDependency(pkg, dep)
	mods, err := aliased.ResolveModules()
	if err != nil {
		t.Fatalf("resolve modules: %v", err)
	}
	got := modURIs(mods)
	want := []string{"prelude", "future", "io"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
