package orchestra_test

import (
	"context"
	"path/filepath"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
)

func TestParseModulePreloadsPrelude(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "app/main.zirr", "module app\nlet greeting = \"hi\"\n")

	registryFS := memfs.New()
	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:      projectFS,
		ProjectBaseURI: registry.LogicalURI("project"),
		RegistryFS:     registryFS,
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}

	module, err := orch.ParseModulePath(context.Background(), "app", orch.NewResolver(cavefile.Cavefile{}))
	if err != nil {
		t.Fatalf("parse module: %v", err)
	}

	if module.Decls.Parent == nil {
		t.Fatal("expected module decls to have a prelude parent")
	}
	sym := module.Decls.Parent.Symbols["String"]
	if sym == nil || sym.Decl == nil {
		t.Fatal("expected prelude symbol String to be present")
	}
	if _, ok := sym.Decl.(*ast.DeclExternType); !ok {
		t.Fatalf("expected String to be an extern type, got %T", sym.Decl)
	}
}

func TestRunFile(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "module main\n1\n")

	registryFS := memfs.New()
	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:      projectFS,
		ProjectBaseURI: registry.LogicalURI("project"),
		RegistryFS:     registryFS,
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}

	if err := orch.RunFile(context.Background(), "main.zirr", cavefile.Cavefile{}); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func TestParseFileUsesPreludeAnnotation(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "module main\n@Type(String)\ndata Example { name }\n")

	registryFS := memfs.New()
	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:      projectFS,
		ProjectBaseURI: registry.LogicalURI("project"),
		RegistryFS:     registryFS,
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}

	module, err := orch.ParseFile(context.Background(), "main.zirr", orch.NewResolver(cavefile.Cavefile{}))
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}
	if module.Decls.Parent == nil {
		t.Fatal("expected prelude decls to be parented")
	}
	if module.Decls.Parent.Symbols["Type"] == nil {
		t.Fatal("expected prelude annotation Type to be present")
	}
	if module.Decls.Parent.Symbols["String"] == nil {
		t.Fatal("expected prelude type String to be present")
	}

	sym := module.Decls.Symbols["Example"]
	if sym == nil || sym.Decl == nil {
		t.Fatal("expected data Example to be declared")
	}
	decl, ok := sym.Decl.(*ast.DeclData)
	if !ok {
		t.Fatalf("expected Example to be data, got %T", sym.Decl)
	}
	if len(decl.Annotations) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(decl.Annotations))
	}
	anno := decl.Annotations[0]
	if len(anno.Reference) != 1 || anno.Reference[0].Value != "Type" {
		t.Fatalf("expected @Type annotation, got %v", anno.Reference)
	}
	if len(anno.Arguments) != 1 {
		t.Fatalf("expected 1 annotation argument, got %d", len(anno.Arguments))
	}
	arg, ok := anno.Arguments[0].(*ast.ExprIdentifier)
	if !ok {
		t.Fatalf("expected annotation argument to be identifier, got %T", anno.Arguments[0])
	}
	if arg.Name.Value != "String" {
		t.Fatalf("expected annotation argument String, got %q", arg.Name.Value)
	}
}

func writeFile(t *testing.T, fs billy.Filesystem, path string, contents string) {
	t.Helper()

	dir := filepath.Dir(path)
	if dir != "." {
		if err := fs.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	f, err := fs.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}

	if _, err := f.Write([]byte(contents)); err != nil {
		_ = f.Close()
		t.Fatalf("write %s: %v", path, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close %s: %v", path, err)
	}
}
