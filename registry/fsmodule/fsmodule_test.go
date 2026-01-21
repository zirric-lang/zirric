package fsmodule_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/registry"
	"code.knabel.dev/zirric-lang/zirric/registry/fsmodule"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/google/go-cmp/cmp"
)

type (
	testCase struct {
		name  string
		cwd   string
		base  registry.LogicalURI
		files map[string]string
		want  []testWant
	}
	testWant struct {
		uri     registry.LogicalURI
		sources map[registry.LogicalURI]string
	}
)

func TestDiscoderFSModules(t *testing.T) {
	fs := memfs.New()

	tests := []testCase{
		{
			name: "basic test",
			cwd:  "/code.knabel.dev/zirric-lang/zirric-example",
			base: "memory:///code.knabel.dev/zirric-lang/zirric-example",
			files: map[string]string{
				"/code.knabel.dev/zirric-lang/zirric-example/Cavefile":       "module cavefile",
				"/code.knabel.dev/zirric-lang/zirric-example/tools/fmt.zirr": "module tools",
				"/code.knabel.dev/zirric-lang/zirric-example/cmd/main.zirr":  "module cmd",
				"/code.knabel.dev/zirric-lang/zirric-example/app/root.zirr":  "module app",

				"/code.knabel.dev/zirric-lang/zirric-example/app/views/body.zirr": "module views",
			},
			want: []testWant{
				{
					uri: "memory:///code.knabel.dev/zirric-lang/zirric-example/app",
					sources: map[registry.LogicalURI]string{
						"memory:///code.knabel.dev/zirric-lang/zirric-example/app/root.zirr": "module app",
					},
				},
				{
					uri: "memory:///code.knabel.dev/zirric-lang/zirric-example/app/views",
					sources: map[registry.LogicalURI]string{
						"memory:///code.knabel.dev/zirric-lang/zirric-example/app/views/body.zirr": "module views",
					},
				},
				{
					uri: "memory:///code.knabel.dev/zirric-lang/zirric-example/cmd",
					sources: map[registry.LogicalURI]string{
						"memory:///code.knabel.dev/zirric-lang/zirric-example/cmd/main.zirr": "module cmd",
					},
				},
				{
					uri: "memory:///code.knabel.dev/zirric-lang/zirric-example/tools",
					sources: map[registry.LogicalURI]string{
						"memory:///code.knabel.dev/zirric-lang/zirric-example/tools/fmt.zirr": "module tools",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for path, cont := range tt.files {
				base := filepath.Base(path)
				err := fs.MkdirAll(base, 0644)
				if err != nil {
					t.Error(err)
				}

				file, err := fs.Create(path)
				if err != nil {
					t.Error(err)
				}
				_, err = file.Write([]byte(cont))
				if err != nil {
					t.Error(err)
				}
				err = file.Close()
				if err != nil {
					t.Error(err)
				}
			}

			pkgfs, err := fs.Chroot(tt.cwd)
			if err != nil {
				t.Error(err)
			}

			mods, err := fsmodule.DiscoverModules(tt.base, pkgfs)
			if err != nil {
				t.Error(err)
			}

			if len(mods) != len(tt.want) {
				t.Errorf("want %d modules, got %d (%s)", len(tt.want), len(mods), mods)
			}

			for i, mod := range mods {
				want := tt.want[i]
				t.Run(fmt.Sprintf("mod %d.", i), func(t *testing.T) {
					if mod.URI() != want.uri {
						t.Errorf("got uri %q, want %q", mod.URI(), want.uri)
					}
					srcs, err := mod.Sources()
					if err != nil {
						t.Error(err)
					}
					if len(srcs) != len(want.sources) {
						t.Errorf("app should have %d file, got %d", len(want.sources), len(srcs))
					}

					for j, src := range srcs {
						t.Run(fmt.Sprintf("src %d.", j), func(t *testing.T) {
							wantsrc, ok := want.sources[src.URI()]
							if !ok {
								t.Errorf("unknown source %q", src.URI())
							}

							contents, err := src.Read()
							if err != nil {
								t.Error(err)
							}
							diff := cmp.Diff(string(contents), wantsrc)
							if diff != "" {
								t.Error(diff)
							}
						})
					}
				})
			}
		})
	}
}
