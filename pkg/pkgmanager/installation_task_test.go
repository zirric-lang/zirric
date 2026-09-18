package pkgmanager

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

// stubProvider implements registry.Provider with injectable behaviours.
type stubProvider struct {
	discoverFn           func(context.Context) ([]registry.ResolvedPackage, error)
	discoverVersionsFunc func(context.Context, string, ...version.Predicate) ([]registry.Package, error)
}

func (s *stubProvider) Discover(ctx context.Context) ([]registry.ResolvedPackage, error) {
	if s.discoverFn != nil {
		return s.discoverFn(ctx)
	}
	return nil, nil
}

func (s *stubProvider) DiscoverPackageVersions(ctx context.Context, name string, preds ...version.Predicate) ([]registry.Package, error) {
	if s.discoverVersionsFunc != nil {
		return s.discoverVersionsFunc(ctx, name, preds...)
	}
	return nil, nil
}

type stubResolvedPackage struct {
	source              string
	version             version.Version
	resolveErr          error
	resolveModulesErr   error
	resolvedModulesResp []registry.ResolvedModule
}

func (p *stubResolvedPackage) Source() string {
	return p.source
}

func (p *stubResolvedPackage) Version() version.Version {
	return p.version
}

func (p *stubResolvedPackage) Resolve(ctx context.Context) (registry.ResolvedPackage, error) {
	if p.resolveErr != nil {
		return nil, p.resolveErr
	}
	return p, nil
}

func (p *stubResolvedPackage) ResolveModules() ([]registry.ResolvedModule, error) {
	if p.resolveModulesErr != nil {
		return nil, p.resolveModulesErr
	}
	return p.resolvedModulesResp, nil
}

type stubPackage struct {
	source     string
	version    version.Version
	resolved   registry.ResolvedPackage
	resolveErr error
}

func (p *stubPackage) Source() string {
	return p.source
}

func (p *stubPackage) Version() version.Version {
	return p.version
}

func (p *stubPackage) Resolve(ctx context.Context) (registry.ResolvedPackage, error) {
	if p.resolveErr != nil {
		return nil, p.resolveErr
	}
	return p.resolved, nil
}

func sourcesAndVersions(pkgs []registry.ResolvedPackage) []string {
	out := make([]string, len(pkgs))
	for i, p := range pkgs {
		out[i] = fmt.Sprintf("%s@%s", p.Source(), p.Version())
	}
	return out
}

func TestInstallationTaskRun(t *testing.T) {
	versionOne := version.Parse("1.0.0")
	predicateExactOne := []version.Predicate{
		version.Predicate{Comparison: version.ComparisonExact, Version: versionOne},
	}

	resolvedFromDiscover := &stubResolvedPackage{source: "local/pkg", version: versionOne}
	resolvedFromRemote := &stubResolvedPackage{source: "remote/pkg", version: versionOne}

	errDiscover := errors.New("discover error")
	errDiscoverVersions := errors.New("discover versions error")
	errResolve := errors.New("resolve error")

	tests := []struct {
		name          string
		cave          cavefile.Cavefile
		initialQueue  []cavefile.Dependency
		provider      registry.Provider
		wantCompleted []registry.ResolvedPackage
		wantQueue     []cavefile.Dependency
		wantErr       error
	}{
		{
			name: "initializes queue from cavefile dependencies and uses local discovery",
			cave: cavefile.Cavefile{Dependencies: []cavefile.Dependency{{
				Package:    cavefile.Package{Name: "pkg", Source: "local/pkg"},
				Predicates: predicateExactOne,
			}}},
			provider: &stubProvider{
				discoverFn: func(context.Context) ([]registry.ResolvedPackage, error) {
					return []registry.ResolvedPackage{resolvedFromDiscover}, nil
				},
			},
			wantCompleted: []registry.ResolvedPackage{resolvedFromDiscover},
			wantQueue: []cavefile.Dependency{{
				Package:    cavefile.Package{Name: "pkg", Source: "local/pkg"},
				Predicates: predicateExactOne,
			}},
		},
		{
			name: "falls back to DiscoverPackageVersions when local match unavailable",
			cave: cavefile.Cavefile{Dependencies: []cavefile.Dependency{{
				Package:    cavefile.Package{Name: "pkg", Source: "remote/pkg"},
				Predicates: predicateExactOne,
			}}},
			initialQueue: []cavefile.Dependency{{
				Package:    cavefile.Package{Name: "pkg", Source: "remote/pkg"},
				Predicates: predicateExactOne,
			}},
			provider: &stubProvider{
				discoverFn: func(context.Context) ([]registry.ResolvedPackage, error) {
					// provide a mismatching version so DiscoverPackageVersions must be used
					otherVersion := version.Parse("0.9.0")
					return []registry.ResolvedPackage{&stubResolvedPackage{source: "remote/pkg", version: otherVersion}}, nil
				},
				discoverVersionsFunc: func(_ context.Context, name string, preds ...version.Predicate) ([]registry.Package, error) {
					if name != "remote/pkg" {
						t.Fatalf("unexpected name: %s", name)
					}
					if !reflect.DeepEqual(preds, predicateExactOne) {
						t.Fatalf("unexpected predicates: %#v", preds)
					}
					pkg := &stubPackage{
						source:   "remote/pkg",
						version:  versionOne,
						resolved: resolvedFromRemote,
					}
					return []registry.Package{pkg}, nil
				},
			},
			wantCompleted: []registry.ResolvedPackage{resolvedFromRemote},
			wantQueue: []cavefile.Dependency{{
				Package:    cavefile.Package{Name: "pkg", Source: "remote/pkg"},
				Predicates: predicateExactOne,
			}},
		},
		{
			name: "propagates discover error",
			cave: cavefile.Cavefile{Dependencies: []cavefile.Dependency{{
				Package:    cavefile.Package{Name: "pkg", Source: "local/pkg"},
				Predicates: predicateExactOne,
			}}},
			provider: &stubProvider{
				discoverFn: func(context.Context) ([]registry.ResolvedPackage, error) {
					return nil, errDiscover
				},
			},
			wantQueue: []cavefile.Dependency{{
				Package:    cavefile.Package{Name: "pkg", Source: "local/pkg"},
				Predicates: predicateExactOne,
			}},
			wantErr: errDiscover,
		},
		{
			name: "propagates discover package versions error",
			cave: cavefile.Cavefile{Dependencies: []cavefile.Dependency{{
				Package:    cavefile.Package{Name: "pkg", Source: "remote/pkg"},
				Predicates: predicateExactOne,
			}}},
			provider: &stubProvider{
				discoverFn: func(context.Context) ([]registry.ResolvedPackage, error) {
					return nil, nil
				},
				discoverVersionsFunc: func(context.Context, string, ...version.Predicate) ([]registry.Package, error) {
					return nil, errDiscoverVersions
				},
			},
			wantQueue: []cavefile.Dependency{{
				Package:    cavefile.Package{Name: "pkg", Source: "remote/pkg"},
				Predicates: predicateExactOne,
			}},
			wantErr: errDiscoverVersions,
		},
		{
			name: "propagates package resolve error",
			cave: cavefile.Cavefile{Dependencies: []cavefile.Dependency{{
				Package:    cavefile.Package{Name: "pkg", Source: "remote/pkg"},
				Predicates: predicateExactOne,
			}}},
			provider: &stubProvider{
				discoverFn: func(context.Context) ([]registry.ResolvedPackage, error) {
					return nil, nil
				},
				discoverVersionsFunc: func(context.Context, string, ...version.Predicate) ([]registry.Package, error) {
					pkg := &stubPackage{
						source:     "remote/pkg",
						version:    versionOne,
						resolveErr: errResolve,
					}
					return []registry.Package{pkg}, nil
				},
			},
			wantQueue: []cavefile.Dependency{{
				Package:    cavefile.Package{Name: "pkg", Source: "remote/pkg"},
				Predicates: predicateExactOne,
			}},
			wantErr: errResolve,
		},
		{
			name: "returns error when no registry can satisfy dependency",
			cave: cavefile.Cavefile{Dependencies: []cavefile.Dependency{{
				Package:    cavefile.Package{Name: "pkg", Source: "missing/pkg"},
				Predicates: predicateExactOne,
			}}},
			provider: &stubProvider{
				discoverFn: func(context.Context) ([]registry.ResolvedPackage, error) {
					return nil, nil
				},
				discoverVersionsFunc: func(context.Context, string, ...version.Predicate) ([]registry.Package, error) {
					return []registry.Package{}, nil
				},
			},
			wantQueue: []cavefile.Dependency{{
				Package:    cavefile.Package{Name: "pkg", Source: "missing/pkg"},
				Predicates: predicateExactOne,
			}},
			wantErr: errors.New("no registry can provide package missing/pkg"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			task := &InstallationTask{
				cave:       tt.cave,
				pkgmanager: &PackageManager{registries: []registry.Provider{tt.provider}},
				queue:      tt.initialQueue,
			}

			completed, err := task.Run(context.Background())

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// completed packages are wrapped by aliasDependency, so compare by Source/Version instead of deep-equating the wrapper.
				if !reflect.DeepEqual(sourcesAndVersions(completed), sourcesAndVersions(tt.wantCompleted)) {
					t.Fatalf("completed packages mismatch: got %#v, want %#v", completed, tt.wantCompleted)
				}
				if !reflect.DeepEqual(task.queue, tt.wantQueue) {
					t.Fatalf("queue modified: got %#v, want %#v", task.queue, tt.wantQueue)
				}
			}

			if tt.wantErr != nil {
				if len(completed) != 0 {
					t.Fatalf("expected no completed packages, got %#v", completed)
				}
				if !reflect.DeepEqual(task.queue, tt.wantQueue) {
					t.Fatalf("queue mismatch after error: got %#v, want %#v", task.queue, tt.wantQueue)
				}
			}
		})
	}
}

func TestInstallationTaskRun_NotifiesOnInstalled(t *testing.T) {
	versionOne := version.Parse("1.0.0")
	provider := &stubProvider{
		discoverFn: func(context.Context) ([]registry.ResolvedPackage, error) {
			return []registry.ResolvedPackage{&stubResolvedPackage{source: "local/pkg", version: versionOne}}, nil
		},
	}

	var events []InstallEvent
	task := &InstallationTask{
		cave: cavefile.Cavefile{Dependencies: []cavefile.Dependency{
			{Package: cavefile.Package{Name: "pkg", Source: "local/pkg"}},
		}},
		pkgmanager:  &PackageManager{registries: []registry.Provider{provider}},
		OnInstalled: func(evt InstallEvent) { events = append(events, evt) },
	}

	if _, err := task.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 install event, got %d: %+v", len(events), events)
	}
	if events[0].Dependency.Name != "pkg" || events[0].Package.Source() != "local/pkg" {
		t.Errorf("unexpected event: %+v", events[0])
	}
}

func TestInstallationTaskRun_ReadOnlyMissingDoesNotNotify(t *testing.T) {
	provider := &stubProvider{
		discoverFn: func(context.Context) ([]registry.ResolvedPackage, error) { return nil, nil },
	}

	var events []InstallEvent
	task := &InstallationTask{
		cave: cavefile.Cavefile{Dependencies: []cavefile.Dependency{
			{Package: cavefile.Package{Name: "pkg", Source: "missing/pkg"}},
		}},
		pkgmanager:  &PackageManager{registries: []registry.Provider{provider}},
		ReadOnly:    true,
		OnInstalled: func(evt InstallEvent) { events = append(events, evt) },
	}

	if _, err := task.Run(context.Background()); err == nil {
		t.Fatal("expected an error for a missing dependency in read-only mode")
	}
	if len(events) != 0 {
		t.Fatalf("expected no install events for a missing dependency, got %+v", events)
	}
}
