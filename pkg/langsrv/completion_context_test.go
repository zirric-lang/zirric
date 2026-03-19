package langsrv

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"github.com/go-git/go-billy/v5/memfs"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func parseModuleForTest(t *testing.T, src string) (*zirricLangserver, string) {
	t.Helper()
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)
	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")
	return ls, "main.zirr"
}

func getScope(t *testing.T, src string, pos protocol.Position) completionScope {
	t.Helper()
	ls, path := parseModuleForTest(t, src)

	text, err := readFileText(ls.fs, path)
	if err != nil {
		t.Fatalf("readFileText: %v", err)
	}
	module, _, _, err := ls.parseModuleFiles("/")
	if err != nil {
		t.Fatalf("parseModuleFiles: %v", err)
	}
	sourceURI := string(registry.JoinModuleURI("", path))
	cursorOffset := offsetForPosition(text, pos)
	return detectCompletionScope(text, pos, module, sourceURI, cursorOffset)
}

func TestCompletionScopeTopLevel(t *testing.T) {
	scope := getScope(t, "fn greet() {}\n", protocol.Position{Line: 1, Character: 0})
	if !scope.isTopLevel {
		t.Error("expected isTopLevel=true at module level")
	}
	if scope.inFunc {
		t.Error("expected inFunc=false at module level")
	}
}

func TestCompletionScopeFirstStatement(t *testing.T) {
	// Empty file — cursor at start.
	scope := getScope(t, "", protocol.Position{Line: 0, Character: 0})
	if !scope.isFirstStmt {
		t.Error("expected isFirstStmt=true in empty file")
	}

	// After a declaration, no longer first.
	scope = getScope(t, "fn greet() {}\n", protocol.Position{Line: 1, Character: 0})
	if scope.isFirstStmt {
		t.Error("expected isFirstStmt=false after a declaration")
	}
}

func TestCompletionScopeInFunc(t *testing.T) {
	src := "fn test() {\n  \n}"
	scope := getScope(t, src, protocol.Position{Line: 1, Character: 2})
	if !scope.inFunc {
		t.Error("expected inFunc=true inside function body")
	}
	if scope.isTopLevel {
		t.Error("expected isTopLevel=false inside function")
	}
}

func TestCompletionScopeInFor(t *testing.T) {
	src := "fn test() {\n  for x <- [1] {\n    \n  }\n}"
	scope := getScope(t, src, protocol.Position{Line: 2, Character: 4})
	if !scope.inFor {
		t.Error("expected inFor=true inside for loop")
	}
	if !scope.inFunc {
		t.Error("expected inFunc=true inside for loop (which is inside a function)")
	}
}

func TestCompletionScopeTypeExpr(t *testing.T) {
	tests := []struct {
		name string
		src  string
		pos  protocol.Position
		want bool
	}{
		{
			name: "after colon in param",
			src:  "fn test(x: ) {}",
			pos:  protocol.Position{Line: 0, Character: 11},
			want: true,
		},
		{
			name: "after arrow in return type",
			src:  "fn test() ->  {}",
			pos:  protocol.Position{Line: 0, Character: 13},
			want: true,
		},
		{
			name: "after colon in const type hint",
			src:  "const x:  = 1",
			pos:  protocol.Position{Line: 0, Character: 9},
			want: true,
		},
		{
			name: "normal expression position",
			src:  "const x = ",
			pos:  protocol.Position{Line: 0, Character: 10},
			want: false,
		},
		{
			name: "after colon typing a type name",
			src:  "fn test(x: Str) {}",
			pos:  protocol.Position{Line: 0, Character: 14},
			want: true,
		},
		{
			name: "after colon with @ attr",
			src:  "fn test(x: @",
			pos:  protocol.Position{Line: 0, Character: 12},
			want: true,
		},
		{
			name: "after colon with @ and partial name",
			src:  "fn test(x: @Any",
			pos:  protocol.Position{Line: 0, Character: 15},
			want: true,
		},
		{
			name: "after arrow with @",
			src:  "fn test() -> @",
			pos:  protocol.Position{Line: 0, Character: 14},
			want: true,
		},
		{
			name: "attr chain: second attr after first",
			src:  "fn test(x: @Attr1 @",
			pos:  protocol.Position{Line: 0, Character: 19},
			want: true,
		},
		{
			name: "attr chain: partial second attr",
			src:  "fn test(x: @Attr1 @Attr",
			pos:  protocol.Position{Line: 0, Character: 23},
			want: true,
		},
		{
			name: "inside array type",
			src:  "fn test(x: [@",
			pos:  protocol.Position{Line: 0, Character: 13},
			want: true,
		},
		{
			name: "inside dict type value",
			src:  "fn test(x: [Key: @",
			pos:  protocol.Position{Line: 0, Character: 18},
			want: true,
		},
		{
			name: "fn type param",
			src:  "fn test(x: fn(@",
			pos:  protocol.Position{Line: 0, Character: 15},
			want: true,
		},
		{
			name: "fn type second param",
			src:  "fn test(x: fn(@Attr, @",
			pos:  protocol.Position{Line: 0, Character: 22},
			want: true,
		},
		{
			name: "fn type return",
			src:  "fn test(x: fn() -> @",
			pos:  protocol.Position{Line: 0, Character: 20},
			want: true,
		},
		{
			name: "after array type in chain",
			src:  "fn test(x: [Item] @",
			pos:  protocol.Position{Line: 0, Character: 19},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTypeExprContext(tt.src, tt.pos)
			if got != tt.want {
				t.Errorf("isTypeExprContext = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompletionScopeImportBlock(t *testing.T) {
	src := "import foo {\n  \n}"
	offset := offsetForPosition(src, protocol.Position{Line: 1, Character: 2})
	if !isImportBlockContext(src, offset) {
		t.Error("expected isImportBlock=true inside import { }")
	}

	src2 := "import foo\nfn test() {}"
	offset2 := offsetForPosition(src2, protocol.Position{Line: 1, Character: 0})
	if isImportBlockContext(src2, offset2) {
		t.Error("expected isImportBlock=false outside import braces")
	}
}

func TestCollectUsedAttrs(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pos      protocol.Position
		expected map[string]bool
	}{
		{
			name:     "no attrs",
			text:     "fn greet(person: ",
			pos:      protocol.Position{Line: 0, Character: 17},
			expected: map[string]bool{},
		},
		{
			name:     "one attr before cursor @",
			text:     "fn greet(person: @Tag @",
			pos:      protocol.Position{Line: 0, Character: 23},
			expected: map[string]bool{"Tag": true},
		},
		{
			name:     "two attrs before cursor @",
			text:     "fn greet(person: @Tag @Validated @",
			pos:      protocol.Position{Line: 0, Character: 34},
			expected: map[string]bool{"Tag": true, "Validated": true},
		},
		{
			name:     "partial ident after @",
			text:     "fn greet(person: @Tag @Va",
			pos:      protocol.Position{Line: 0, Character: 25},
			expected: map[string]bool{"Tag": true},
		},
		{
			name:     "just @ at cursor",
			text:     "fn greet(person: @",
			pos:      protocol.Position{Line: 0, Character: 18},
			expected: map[string]bool{},
		},
		{
			name:     "colon before first @",
			text:     "name: @Tag @",
			pos:      protocol.Position{Line: 0, Character: 12},
			expected: map[string]bool{"Tag": true},
		},
		{
			name:     "multi-line decl attrs",
			text:     "@Tag()\n@Validated()\n@",
			pos:      protocol.Position{Line: 2, Character: 1},
			expected: map[string]bool{"Tag": true, "Validated": true},
		},
		{
			name:     "multi-line decl attrs with args",
			text:     "@Type(Person)\n@Has(name, age)\n@",
			pos:      protocol.Position{Line: 2, Character: 1},
			expected: map[string]bool{"Type": true, "Has": true},
		},
		{
			name:     "multi-line with partial ident",
			text:     "@Tag()\n@Va",
			pos:      protocol.Position{Line: 1, Character: 3},
			expected: map[string]bool{"Tag": true},
		},
		{
			name:     "single attr no parens multi-line",
			text:     "@Tag\n@",
			pos:      protocol.Position{Line: 1, Character: 1},
			expected: map[string]bool{"Tag": true},
		},
		{
			name:     "stops at non-attr content",
			text:     "data Foo {}\n@Tag()\n@",
			pos:      protocol.Position{Line: 2, Character: 1},
			expected: map[string]bool{"Tag": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectUsedAttrs(tt.text, tt.pos)
			for k := range tt.expected {
				if !result[k] {
					t.Errorf("expected %q in used attrs", k)
				}
			}
			for k := range result {
				if !tt.expected[k] {
					t.Errorf("unexpected %q in used attrs", k)
				}
			}
		})
	}
}
