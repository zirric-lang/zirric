package pkgmanager

import (
	"context"
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

type InstallationTask struct {
	cave       cavefile.Cavefile
	pkgmanager *PackageManager

	queue []cavefile.Dependency
}

// TODO: Recursively install dependencies!
func (t *InstallationTask) Run(ctx context.Context) ([]registry.ResolvedPackage, error) {
	if t.queue == nil {
		t.queue = t.cave.Dependencies
		if t.cave.Name != "" || t.cave.Source != "" {
			t.queue = append(t.queue, cavefile.Dependency{
				Package: cavefile.Package{
					Name:   t.cave.Name,
					Source: t.cave.Source,
				},
				Predicates: nil,
			})
		}
	}
	availables := make(map[string][]registry.ResolvedPackage, 0)

	for _, reg := range t.pkgmanager.registries {
		available, err := reg.Discover(ctx)
		if err != nil {
			return nil, err
		}
		for _, pkg := range available {
			availables[pkg.Source()] = append(availables[pkg.Source()], pkg)
		}
	}

	var completed []registry.ResolvedPackage

	for _, dependency := range t.queue {
		pkg, ok := t.tryResolveAvailable(dependency, availables)
		if ok {
			completed = append(completed, pkg)
			continue
		}

		localPkg, err := t.tryResolveRemote(ctx, dependency)
		if err != nil {
			return nil, err
		}
		if localPkg == nil {
			return nil, fmt.Errorf("no registry can provide package %s", dependency.Source)
		}
		completed = append(completed, localPkg)
	}
	return completed, nil
}

func (t *InstallationTask) tryResolveAvailable(dep cavefile.Dependency, availables map[string][]registry.ResolvedPackage) (registry.ResolvedPackage, bool) {
	for _, key := range dependencyKeys(dep) {
		if available, ok := availables[key]; ok {
			for _, pkg := range available {
				if version.MatchesAll(pkg.Version(), dep.Predicates...) {
					return pkg, true
				}
			}
		}
	}
	return nil, false
}

func (t *InstallationTask) tryResolveRemote(ctx context.Context, dep cavefile.Dependency) (registry.ResolvedPackage, error) {
	for _, key := range dependencyKeys(dep) {
		for _, reg := range t.pkgmanager.registries {
			pkgs, err := reg.DiscoverPackageVersions(ctx, key, dep.Predicates...)
			if err != nil {
				return nil, err
			}
			if len(pkgs) == 0 {
				continue
			}
			localPkg, err := pkgs[0].Resolve(ctx)
			if err != nil {
				return nil, err
			}
			return localPkg, nil
		}
	}
	return nil, nil
}

func dependencyKeys(dep cavefile.Dependency) []string {
	keys := make([]string, 0, 2)
	if dep.Source != "" {
		keys = append(keys, dep.Source)
	}
	if dep.Name != "" && dep.Name != dep.Source {
		keys = append(keys, dep.Name)
	}
	return keys
}
