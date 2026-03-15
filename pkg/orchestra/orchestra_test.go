package orchestra_test

import (
	"context"
	"path/filepath"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
)

func TestParseModulePreloadsPrelude(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "app/main.zirr", "mod app\nconst greeting = \"hi\"\n")

	orch := newTestOrchestra(t, projectFS, "project")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}

	module, err := orch.ParseModulePath(context.Background(), "app", resolver)
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
	writeFile(t, projectFS, "main.zirr", "mod main\n1\n")

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func TestParseFileUsesPreludeAttribute(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\n@Type(String)\ndata Example { name }\n")

	orch := newTestOrchestra(t, projectFS, "project")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	module, err := orch.ParseFile(context.Background(), "main.zirr", resolver)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}
	if module.Decls.Parent == nil {
		t.Fatal("expected prelude decls to be parented")
	}
	if module.Decls.Parent.Symbols["Type"] == nil {
		t.Fatal("expected prelude attribute Type to be present")
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
	if len(decl.Attributes) != 1 {
		t.Fatalf("expected 1 attribute, got %d", len(decl.Attributes))
	}
	anno := decl.Attributes[0]
	if len(anno.Reference) != 1 || anno.Reference[0].Value != "Type" {
		t.Fatalf("expected @Type attribute, got %v", anno.Reference)
	}
	if len(anno.Arguments) != 1 {
		t.Fatalf("expected 1 attribute argument, got %d", len(anno.Arguments))
	}
	arg, ok := anno.Arguments[0].(*ast.ExprIdentifier)
	if !ok {
		t.Fatalf("expected attribute argument to be identifier, got %T", anno.Arguments[0])
	}
	if arg.Name.Value != "String" {
		t.Fatalf("expected attribute argument String, got %q", arg.Name.Value)
	}
}

// TestRunFileNoDeclaredDependencies verifies that a project with no explicit
// dependencies (only stdlib injected automatically) can still resolve and run
// its own modules. This guards against a coupling bug where the project package
// itself was only installed when declared dependencies were missing.
func TestRunFileNoDeclaredDependencies(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst answer = \"42\"\n")

	// Deliberately no PackageName-derived dependencies beyond stdlib (injected automatically).
	orch := newTestOrchestra(t, projectFS, "main")

	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestRunFileWithCrossModuleImport verifies that a file importing another module
// within the same project is resolved correctly. This exercises the cavereg
// filesystem path: the resolver must discover project sub-modules from
// ProjectFS (not RegistryFS) via findResolvedModule.
func TestRunFileWithCrossModuleImport(t *testing.T) {
	projectFS := memfs.New()
	// utils/ subdirectory → URI "project.utils" (directory name is the URI segment)
	writeFile(t, projectFS, "utils/greet.zirr", "mod utils\nconst greeting = \"hello\"\n")
	writeFile(t, projectFS, "main.zirr", "mod main\nimport utils = project.utils\n")

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func newTestOrchestra(t *testing.T, projectFS billy.Filesystem, name string) *orchestra.Orchestra {
	t.Helper()
	registryFS := memfs.New()
	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:   projectFS,
		RegistryFS:  registryFS,
		PackageName: name,
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}
	return orch
}

// TestMainModuleIsRegisteredAfterParse verifies that after ParseFile the
// resolver's MainModule returns the same pointer, which is required for the
// compiler's pointer-equality checks to work correctly.
func TestMainModuleIsRegisteredAfterParse(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst x = 42\n")

	orch := newTestOrchestra(t, projectFS, "main")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}

	module, err := orch.ParseFile(context.Background(), "main.zirr", resolver)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}

	if resolver.MainModule() == nil {
		t.Fatal("expected MainModule to be non-nil after ParseFile")
	}
	if resolver.MainModule() != module {
		t.Fatal("expected MainModule to be the same pointer as the parsed module")
	}
}

// TestRunFileWithConstBinding verifies that a file with a const binding compiles
// and runs correctly — exercising full main-module symbol compilation.
func TestRunFileWithLetBinding(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst greeting = \"hello\"\n")

	orch := newTestOrchestra(t, projectFS, "main")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestREPLLoop verifies the REPL's parse→compile pattern: a single resolver
// is reused across multiple ParseFile+Compile calls, as the REPL does.
func TestREPLLoop(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "repl.zirr", "mod repl\n")

	orch := newTestOrchestra(t, projectFS, "repl")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}

	iterations := []string{
		"mod repl\n1\n",
		"mod repl\n2\n",
	}
	for i, src := range iterations {
		if err := projectFS.Remove("repl.zirr"); err != nil {
			t.Fatalf("iter %d: remove: %v", i, err)
		}
		writeFile(t, projectFS, "repl.zirr", src)

		module, err := orch.ParseFile(context.Background(), "repl.zirr", resolver)
		if err != nil {
			t.Fatalf("iter %d: parse file: %v", i, err)
		}
		if _, err := orch.Compile(module, resolver); err != nil {
			t.Fatalf("iter %d: compile: %v", i, err)
		}
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
