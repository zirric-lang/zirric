package pkgmanager

import (
	"context"
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

// InstallEvent reports a dependency having finished installing. More granular phases
// (e.g. discovered/installing) may be added here in the future.
type InstallEvent struct {
	Dependency cavefile.Dependency
	Package    registry.ResolvedPackage
}

type InstallProgress func(InstallEvent)

type InstallationTask struct {
	cave        cavefile.Cavefile
	pkgmanager  *PackageManager
	ReadOnly    bool
	OnInstalled InstallProgress

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
	var missing []string

	for _, dependency := range t.queue {
		pkg, ok := t.tryResolveAvailable(dependency, availables)
		if ok {
			aliased := aliasDependency(pkg, dependency)
			completed = append(completed, aliased)
			t.notify(dependency, aliased)
			continue
		}

		if t.ReadOnly {
			name := dependency.Source
			if name == "" {
				name = dependency.Name
			}
			missing = append(missing, name)
			continue
		}

		localPkg, err := t.tryResolveRemote(ctx, dependency)
		if err != nil {
			return nil, err
		}
		if localPkg == nil {
			return nil, fmt.Errorf("no registry can provide package %s", dependency.Source)
		}
		aliased := aliasDependency(localPkg, dependency)
		completed = append(completed, aliased)
		t.notify(dependency, aliased)
	}
	if len(missing) > 0 {
		return completed, fmt.Errorf("dependencies not installed locally: %s; run 'zirric install'", strings.Join(missing, ", "))
	}
	return completed, nil
}

func (t *InstallationTask) notify(dep cavefile.Dependency, pkg registry.ResolvedPackage) {
	if t.OnInstalled == nil {
		return
	}
	t.OnInstalled(InstallEvent{Dependency: dep, Package: pkg})
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
