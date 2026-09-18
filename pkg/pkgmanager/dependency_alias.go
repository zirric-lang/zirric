package pkgmanager

import (
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
)

// aliasDependency makes pkg's modules addressable under dep.Name, the name declared for it in the Cavefile.
func aliasDependency(pkg registry.ResolvedPackage, dep cavefile.Dependency) registry.ResolvedPackage {
	if dep.Name == "" {
		return pkg
	}
	alias := registry.LogicalURI(dep.Name)
	if dep.Module != "" {
		return &aliasedPackage{ResolvedPackage: pkg, rebase: singleModuleRebase(dep.Module, alias)}
	}
	return &aliasedPackage{ResolvedPackage: pkg, rebase: treeRebase(alias)}
}

type rebaseFunc func([]registry.ResolvedModule) []registry.ResolvedModule

type aliasedPackage struct {
	registry.ResolvedPackage
	rebase rebaseFunc
}

func (p *aliasedPackage) ResolveModules() ([]registry.ResolvedModule, error) {
	mods, err := p.ResolvedPackage.ResolveModules()
	if err != nil {
		return nil, err
	}
	return p.rebase(mods), nil
}

func singleModuleRebase(source, alias registry.LogicalURI) rebaseFunc {
	return func(mods []registry.ResolvedModule) []registry.ResolvedModule {
		for _, m := range mods {
			if m.URI() == source {
				return []registry.ResolvedModule{&renamedModule{ResolvedModule: m, uri: alias}}
			}
		}
		return nil
	}
}

// treeRebase renames every module onto alias, if they share a single common root; otherwise it leaves mods unchanged.
func treeRebase(alias registry.LogicalURI) rebaseFunc {
	return func(mods []registry.ResolvedModule) []registry.ResolvedModule {
		base, ok := commonModuleBase(mods)
		if !ok {
			return mods
		}
		renamed := make([]registry.ResolvedModule, len(mods))
		for i, m := range mods {
			renamed[i] = &renamedModule{ResolvedModule: m, uri: rebaseURI(m.URI(), base, alias)}
		}
		return renamed
	}
}

func commonModuleBase(mods []registry.ResolvedModule) (registry.LogicalURI, bool) {
	if len(mods) == 0 {
		return "", false
	}
	base := mods[0].URI()
	for _, m := range mods[1:] {
		if len(m.URI()) < len(base) {
			base = m.URI()
		}
	}
	for _, m := range mods {
		if m.URI() != base && !strings.HasPrefix(string(m.URI()), string(base)+".") {
			return "", false
		}
	}
	return base, true
}

func rebaseURI(uri, oldBase, newBase registry.LogicalURI) registry.LogicalURI {
	if uri == oldBase {
		return newBase
	}
	rest := strings.TrimPrefix(string(uri), string(oldBase)+".")
	return newBase + "." + registry.LogicalURI(rest)
}

type renamedModule struct {
	registry.ResolvedModule
	uri registry.LogicalURI
}

func (m *renamedModule) URI() registry.LogicalURI {
	return m.uri
}
