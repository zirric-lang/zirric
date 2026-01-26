package parser_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
)

func TestModuleParserParsesMultipleSources(t *testing.T) {
	sourceOne := staticmodule.NewSourceString("test/module/one.zirr", `
module testingmodule

func first() {}
`)
	sourceTwo := staticmodule.NewSourceString("test/module/two.zirr", `
module testingmodule

func second() {}
`)

	module := staticmodule.NewModule("test/module", []registry.Source{sourceOne, sourceTwo})
	mp := parser.NewModuleParse(module)

	ctx, err := mp.Parse(module)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if got := len(mp.Errors()); got != 0 {
		t.Fatalf("expected no parse errors, got %d", got)
	}

	if got := len(ctx.Files); got != 2 {
		t.Fatalf("expected 2 source files, got %d", got)
	}

	paths := map[string]bool{
		ctx.Files[0].Path: true,
		ctx.Files[1].Path: true,
	}
	for _, want := range []string{string(sourceOne.URI()), string(sourceTwo.URI())} {
		if !paths[want] {
			t.Fatalf("expected parsed file path %q", want)
		}
	}

	symbols := mp.Decls().Symbols
	if _, ok := symbols["first"]; !ok {
		t.Fatalf("expected symbol %q", "first")
	}
	if _, ok := symbols["second"]; !ok {
		t.Fatalf("expected symbol %q", "second")
	}
}
