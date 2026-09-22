package cmds

import (
	"encoding/json"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
	"github.com/goccy/go-yaml"
)

func testCavefile() cavefile.Cavefile {
	return cavefile.Cavefile{
		Package: cavefile.Package{
			Name:            "proj",
			Source:          "file:///proj",
			Version:         "1.2.3",
			LanguageVersion: "^0.1.0",
			Description:     "A project",
			Documentation:   "https://example.com/docs",
		},
		Dependencies: []cavefile.Dependency{
			{Package: cavefile.Package{Name: "io", Source: cavefile.StandardLibrarySource}, Module: "io"},
			{Package: cavefile.Package{Name: "helpers", Source: "file:///helpers"}},
			{
				Package:    cavefile.Package{Name: "zirric", Source: "https://code.knabel.dev/zirric-lang/zirric"},
				Predicates: []version.Predicate{version.ParsePredicate("latest")},
			},
		},
		Tasks: []cavefile.Task{
			{
				Name: "build", Aliases: []string{"b"}, Help: "Builds the project", Kind: cavefile.TaskKindCall,
				Flags: []cavefile.TaskParam{
					{Name: "dry", Short: "d", Type: cavefile.TaskParamTypeBool},
					{Name: "target", Type: cavefile.TaskParamTypeString},
				},
			},
			{Name: "generate", Help: "Generates something", Kind: cavefile.TaskKindExec, Exec: "tasks/generate.zirr"},
		},
	}
}

func TestPrintCavefile_YAML(t *testing.T) {
	var buf strings.Builder
	if err := printCavefile(&buf, testCavefile(), "yaml"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc cavefileDoc
	if err := yaml.Unmarshal([]byte(buf.String()), &doc); err != nil {
		t.Fatalf("output isn't valid YAML: %v\n%s", err, buf.String())
	}
	assertCavefileDoc(t, doc)
}

func TestPrintCavefile_JSON(t *testing.T) {
	var buf strings.Builder
	if err := printCavefile(&buf, testCavefile(), "json"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc cavefileDoc
	if err := json.Unmarshal([]byte(buf.String()), &doc); err != nil {
		t.Fatalf("output isn't valid JSON: %v\n%s", err, buf.String())
	}
	assertCavefileDoc(t, doc)
}

func assertCavefileDoc(t *testing.T, doc cavefileDoc) {
	t.Helper()
	if doc.Package.Name != "proj" || doc.Package.Source != "file:///proj" {
		t.Errorf("got Name=%q Source=%q, want proj/file:///proj", doc.Package.Name, doc.Package.Source)
	}
	if doc.Package.Version != "1.2.3" || doc.Package.LanguageVersion != "^0.1.0" {
		t.Errorf("got Version=%q LanguageVersion=%q, want 1.2.3/^0.1.0", doc.Package.Version, doc.Package.LanguageVersion)
	}
	if doc.Package.Description != "A project" || doc.Package.Documentation != "https://example.com/docs" {
		t.Errorf("got Description=%q Documentation=%q", doc.Package.Description, doc.Package.Documentation)
	}
	if len(doc.Dependencies) != 3 {
		t.Fatalf("expected 3 dependencies, got %+v", doc.Dependencies)
	}

	stdlib := doc.Dependencies[0]
	if stdlib.Kind != "stdlib" || stdlib.Module != "io" {
		t.Errorf("stdlib dep = %+v, want Kind=stdlib Module=io", stdlib)
	}

	local := doc.Dependencies[1]
	if local.Kind != "local" || local.Source != "file:///helpers" {
		t.Errorf("local dep = %+v, want Kind=local Source=file:///helpers", local)
	}

	git := doc.Dependencies[2]
	if git.Kind != "git" || git.Version != "latest" {
		t.Errorf("git dep = %+v, want Kind=git Version=latest", git)
	}

	if len(doc.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %+v", doc.Tasks)
	}

	call := doc.Tasks[0]
	if call.Name != "build" || call.Kind != "call" || len(call.Aliases) != 1 || call.Aliases[0] != "b" {
		t.Errorf("call task = %+v, want Name=build Kind=call Aliases=[b]", call)
	}
	if len(call.Flags) != 2 || call.Flags[0].Type != "Bool" || call.Flags[0].Short != "d" || call.Flags[1].Type != "String" {
		t.Errorf("call task flags = %+v, want [{dry Bool d} {target String}]", call.Flags)
	}

	exec := doc.Tasks[1]
	if exec.Name != "generate" || exec.Kind != "exec" || exec.Exec != "tasks/generate.zirr" {
		t.Errorf("exec task = %+v, want Name=generate Kind=exec Exec=tasks/generate.zirr", exec)
	}
}

func TestPrintCavefile_UnsupportedFormat(t *testing.T) {
	var buf strings.Builder
	err := printCavefile(&buf, cavefile.Cavefile{}, "toml")
	if err == nil {
		t.Fatal("expected an error for an unsupported format, got nil")
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output on error, got %q", buf.String())
	}
}
