package langsrv

import (
	"path/filepath"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// impliedModuleName is the module a file at path belongs to, fully qualified: the base its Cavefile declares, joined with the directory holding it.
//
// The Cavefile is the exception, and has to be. It is where the base is declared, so its own `mod` is the thing every other answer is derived from — there is nothing to derive it from in turn.
func (ls *zirricLangserver) impliedModuleName(path string) string {
	if ls.isCavefilePath(path) {
		return ""
	}
	base := ls.declaredPackageBase()
	if base == "" {
		return ""
	}
	dir := filepath.Dir(path)
	if dir == "." {
		return string(base)
	}
	return string(registry.JoinModuleURI(base, filepath.ToSlash(dir)))
}

// modNameContext detects the cursor completing the dotted module path of a `mod <path>` declaration, returning the position right after "mod" and its whitespace so the caller can build a replacing TextEdit.
func modNameContext(text string, pos protocol.Position) (startPos protocol.Position, ok bool) {
	line := lineAtPosition(text, pos)
	col := int(pos.Character)
	if col > len(line) {
		col = len(line)
	}

	trimmed := strings.TrimLeft(line, " \t")
	indent := len(line) - len(trimmed)
	if !strings.HasPrefix(trimmed, "mod") {
		return protocol.Position{}, false
	}
	// Reject identifiers that merely start with "mod" (e.g. "module", "modulo").
	if len(trimmed) > len("mod") && isSimpleIdentByte(trimmed[len("mod")]) {
		return protocol.Position{}, false
	}

	i := indent + len("mod")
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
	// Everything to the cursor must be identifier chars and dots; anything else means this is not the module-name position.
	for i < col {
		if c := line[i]; !isSimpleIdentByte(c) && c != '.' {
			return protocol.Position{}, false
		}
		i++
	}

	return protocol.Position{Line: pos.Line, Character: uint32(pathStart)}, true
}

// modModuleNameCompletions offers the one name `mod` can be followed by in this file, replacing whatever has been typed since "mod ".
func (ls *zirricLangserver) modModuleNameCompletions(path string, startPos, cursorPos protocol.Position) []protocol.CompletionItem {
	name := ls.impliedModuleName(path)
	if name == "" {
		return nil
	}
	kind := protocol.CompletionItemKindModule
	return []protocol.CompletionItem{{
		Label:      name,
		FilterText: &name,
		Kind:       &kind,
		TextEdit: protocol.TextEdit{
			Range:   protocol.Range{Start: startPos, End: cursorPos},
			NewText: name,
		},
	}}
}
