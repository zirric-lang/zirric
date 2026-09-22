package orchestra_test

import (
	"context"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
)

// TestLocalConstInLoopBodyReadsTheBinding is a regression test: a const declared inside a loop or an if at the top level of a file is promoted into the file's symbol table so that its name resolves, and the compiler used to compile it as one of the file's declarations as well. Its initializer then ran once before the loop, in a frame where the loop binding it reads has no value yet, which crashed as soon as the initializer did anything with that value.
func TestLocalConstInLoopBodyReadsTheBinding(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

data Row {
	tag: String
}

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

var seen = ""
for row <- [Row("a"), Row("b")] {
	const tag = row.tag
	seen = seen + tag
}
assertEqual(seen, "ab", "FAIL: a const in a loop body must read the binding of the iteration it is in")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestModuleDocsJoinEveryFileInNameOrder checks that a module's documentation is the comment above each of its `mod` declarations, joined in file name order, since a module is declared once per file and no single one of them holds all of it.
func TestModuleDocsJoinEveryFileInNameOrder(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "shapes/second.zirr", `// Second, because its file sorts second.
mod shapes

const two = 2
`)
	writeFile(t, projectFS, "shapes/first.zirr", `// First, because its file sorts first.
mod shapes

const one = 1
`)
	writeFile(t, projectFS, "main.zirr", `mod main

import reflect
import shapes

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

assertEqual(
	reflect.docs(shapes),
	"First, because its file sorts first.\n\nSecond, because its file sorts second.",
	"FAIL: a module's documentation is every file's mod comment, in file name order",
)
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestPackageCavefileReportsTheManifest checks that a program can read what its own project declared: the dependencies and tasks of its Cavefile, with the comments written above them.
func TestPackageCavefileReportsTheManifest(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "Cavefile", `mod proj

import tasks

// Writes the pages.
@tasks.Name("docs")
@tasks.Help("Generate documentation")
@tasks.Exec("tasks/docs.zirr")
data GenerateDocs {
	// Where the pages go.
	@tasks.Flag()
	@tasks.Name("out")
	out: String
}
`)
	writeFile(t, projectFS, "main.zirr", `mod proj

import pkgs = reflect.packages

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

const declared = pkgs.mainPackage().cavefile
assertEqual(declared is Some, true, "FAIL: a project with a Cavefile reports one")

const manifest = declared.value
assertEqual(len(manifest.tasks), 1, "FAIL: the Cavefile declares one task")

const task = manifest.tasks[0]
assertEqual(task.name, "docs", "FAIL: the task is named by @tasks.Name")
assertEqual(task.kind, "exec", "FAIL: the task runs a file")
assertEqual(task.exec, "tasks/docs.zirr", "FAIL: the task names the file it runs")
assertEqual(task.help, "Generate documentation", "FAIL: the task carries its @tasks.Help")
assertEqual(task.docs, "Writes the pages.", "FAIL: the task carries the comment written above it")
assertEqual(len(task.flags), 1, "FAIL: the task takes one flag")
assertEqual(task.flags[0].name, "out", "FAIL: the flag is named by @tasks.Name")
assertEqual(task.flags[0].type, "String", "FAIL: the flag takes a String")
assertEqual(task.flags[0].docs, "Where the pages go.", "FAIL: the flag carries the comment written above it")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestDeclarationsDescribeAConstant checks the one thing a module's exported values cannot answer for: a constant exports a plain value, so its form, position and documentation are only reachable through the module's declarations.
func TestDeclarationsDescribeAConstant(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

import reflect
import shapes

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

const found = reflect.declaration(reflect.moduleOf(shapes), "sides")
assertEqual(found is Some, true, "FAIL: sides is declared by shapes")
assertEqual(found.value.docs, "How many sides a square has.", "FAIL: a const answers with its own comment")
assertEqual(found.value.signature, "const sides: Int", "FAIL: a const answers with how it was written")
assertEqual(found.value.kind is reflect.ConstDecl, true, "FAIL: a const was declared with const")
assertEqual(found.value.source.line, 4, "FAIL: a const answers with the line it was written on")
`)
	writeFile(t, projectFS, "shapes/shapes.zirr", `mod shapes

// How many sides a square has.
const sides: Int = 4
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestInvalidateModulesSeesANewModule guards the module-discovery cache: discovering a package's modules walks its whole tree, so the resolver holds on to the result, and a language server that keeps one resolver for a session would otherwise never see a file that was added while it ran.
func TestInvalidateModulesSeesANewModule(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "shapes/shapes.zirr", "mod shapes\n\nconst sides = 4\n")

	orch := newTestOrchestra(t, projectFS, "project")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}

	if _, err := resolver.ResolveModule(context.Background(), "project.shapes"); err != nil {
		t.Fatalf("resolve shapes: %v", err)
	}

	writeFile(t, projectFS, "colours/colours.zirr", "mod colours\n\nconst count = 7\n")

	if _, err := resolver.ResolveModule(context.Background(), "project.colours"); err == nil {
		t.Fatal("expected the cached discovery to not know about a module added since")
	}

	resolver.InvalidateModules()

	if _, err := resolver.ResolveModule(context.Background(), "project.colours"); err != nil {
		t.Fatalf("resolve colours after invalidating: %v", err)
	}
}
