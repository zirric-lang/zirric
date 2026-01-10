package pkgmanager

import (
	"context"
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/cavefile"
	"code.knabel.dev/zirric-lang/zirric/registry"
)

type InstallationTask struct {
	cave       cavefile.Cavefile
	pkgmanager *PackageManager

	completed []registry.ResolvedPackage
	queue     []cavefile.Dependency
}

// TODO: Recursively install dependencies!
func (t *InstallationTask) Run(ctx context.Context) error {
	if t.queue == nil {
		t.queue = t.cave.Dependencies
	}
	availables := make(map[string][]registry.ResolvedPackage, 0)

	for _, reg := range t.pkgmanager.registries {
		available, err := reg.Discover(ctx)
		if err != nil {
			return err
		}
		for _, pkg := range available {
			availables[pkg.Source()] = append(availables[pkg.Source()], pkg)
		}
	}

	for _, dependency := range t.queue {
		if available, ok := availables[dependency.Source]; ok {
			hasFound := false
			for _, pkg := range available {
				if pkg.Version().Matches(dependency.Predicate) {
					t.completed = append(t.completed, pkg)
					hasFound = true
					break
				}
			}
			if hasFound {
				continue
			}
		}

		hasFound := false
		for _, reg := range t.pkgmanager.registries {
			pkgs, err := reg.DiscoverPackageVersions(ctx, dependency.Source, dependency.Predicate)
			if err != nil {
				return err
			}
			if len(pkgs) == 0 {
				continue
			}
			pkg := pkgs[0]
			localPkg, err := pkg.Resolve(ctx)
			if err != nil {
				return err
			}
			t.completed = append(t.completed, localPkg)
			hasFound = true
			break
		}
		if !hasFound {
			return fmt.Errorf("no registry can provide package %s", dependency.Source)
		}
	}
	return nil
}
