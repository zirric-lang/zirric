package langsrv

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// importNameContext detects the cursor completing the dotted module path of an `import <path>` statement, returning the position right after "import" and its whitespace so the caller can build a replacing TextEdit.
func importNameContext(text string, pos protocol.Position) (startPos protocol.Position, ok bool) {
	line := lineAtPosition(text, pos)
	col := int(pos.Character)
	if col > len(line) {
		col = len(line)
	}

	trimmed := strings.TrimLeft(line, " \t")
	indent := len(line) - len(trimmed)
	if !strings.HasPrefix(trimmed, "import") {
		return protocol.Position{}, false
	}
	// Reject identifiers that merely start with "import" (e.g. "importer").
	if len(trimmed) > len("import") && isSimpleIdentByte(trimmed[len("import")]) {
		return protocol.Position{}, false
	}

	i := indent + len("import")
	if col < i {
		return protocol.Position{}, false
	}

	sawSpace := false
	for i < col && (line[i] == ' ' || line[i] == '\t') {
		i++
		sawSpace = true
	}
	if !sawSpace {
		return protocol.Position{}, false
	}

	pathStart := i
	// Everything to the cursor must be identifier chars and dots; a '{' or '=' means this isn't the module-name position.
	for i < col {
		c := line[i]
		if !(isSimpleIdentByte(c) || c == '.') {
			return protocol.Position{}, false
		}
		i++
	}

	return protocol.Position{Line: pos.Line, Character: uint32(pathStart)}, true
}

// importModuleNameCompletions offers every dotted module name `import <name>` could resolve to; the TextEdit always replaces everything typed since "import " with the full candidate name.
func (ls *zirricLangserver) importModuleNameCompletions(startPos, cursorPos protocol.Position) []protocol.CompletionItem {
	editRange := protocol.Range{Start: startPos, End: cursorPos}
	kind := protocol.CompletionItemKindModule

	var items []protocol.CompletionItem
	for _, name := range ls.importModuleNameCandidates() {
		label := name
		items = append(items, protocol.CompletionItem{
			Label:      label,
			FilterText: &label,
			Kind:       &kind,
			TextEdit:   protocol.TextEdit{Range: editRange, NewText: name},
		})
	}
	return items
}

// importModuleNameCandidates collects every dotted module name available for `import <name>`: embedded stdlib packages and their submodules, the Cavefile's declared dependencies, and the project's own submodules at any depth.
func (ls *zirricLangserver) importModuleNameCandidates() []string {
	seen := make(map[string]bool)
	var names []string
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}

	// Embedded stdlib packages are baked into the binary, so always available regardless of Cavefile/install state.
	if stdlib, err := orchestra.DefaultStdlibProvider(); err == nil {
		if pkgs, err := stdlib.Discover(context.Background()); err == nil {
			for _, pkg := range pkgs {
				mods, err := pkg.ResolveModules()
				if err != nil {
					continue
				}
				for _, mod := range mods {
					add(string(mod.URI()))
				}
			}
		}
	}

	// Excludes the auto-injected standard library dependency, already covered above by its actual submodule names.
	if ls.orch != nil {
		for _, dep := range ls.orch.Cavefile().Dependencies {
			if dep.Source == cavefile.StandardLibrarySource {
				continue
			}
			add(dep.Name)
		}
	}

	for _, dir := range ls.collectModuleDirs(".") {
		if dir == "" || dir == "." {
			continue
		}
		add(strings.ReplaceAll(filepath.ToSlash(dir), "/", "."))
	}

	sort.Strings(names)
	return names
}
