package langsrv

import (
	"testing"

	future "code.knabel.dev/zirric-lang/zirric/future"
	"github.com/go-git/go-billy/v5/memfs"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func BenchmarkParseModuleFiles(b *testing.B) {
	base := memfs.New()
	if err := base.MkdirAll("future/prelude", 0o755); err != nil {
		b.Fatal(err)
	}
	entries, err := future.FS.ReadDir("prelude")
	if err != nil {
		b.Fatal(err)
	}
	for _, e := range entries {
		data, err := future.FS.ReadFile("prelude/" + e.Name())
		if err != nil {
			b.Fatal(err)
		}
		f, err := base.Create("future/prelude/" + e.Name())
		if err != nil {
			b.Fatal(err)
		}
		if _, err := f.Write(data); err != nil {
			b.Fatal(err)
		}
		if err := f.Close(); err != nil {
			b.Fatal(err)
		}
	}

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ls.moduleCacheMu.Lock()
		ls.moduleCache = make(map[string]*moduleCacheEntry)
		ls.moduleCacheMu.Unlock()
		if _, _, _, err := ls.parseModuleFiles("future/prelude"); err != nil {
			b.Fatal(err)
		}
	}
}
