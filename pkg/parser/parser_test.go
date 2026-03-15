package parser_test

import (
	"fmt"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/syncheck"
)

func TestParseSourceFile(t *testing.T) {
	contents := `
mod testingmodule
// <- ast.DeclModule

import json
// <- ast.DeclImport
import big
// <- ast.DeclImport

@json.Type(json.Null)
// <- ast.DeclAttrInstance
data None
// <- ast.DeclData

@json.Inline
data Some {
// <- ast.DeclData
   value
// ^ ast.DeclField
}

@json.Inline()
union Optional {
	None
	Some
}

extern const b // this is an extern constant
// <- ast.DeclExternValue
//           ^ ast.Identifier

extern fn doSomething()
// <- ast.DeclExternFunc
	
extern fn doSomethingWith(argument)
// <- ast.DeclExternFunc
//        ^ ast.Identifier
//                        ^ ast.DeclParameter

extern type SomeType {
// <- ast.DeclExternType
    name
//  ^ ast.DeclField
}

extern type SomeEmptyType {}
// <- ast.DeclExternType
//          ^ ast.Identifier

attr Type {
// <- ast.DeclAttr
	@AnyType
	value
//  ^ ast.DeclField
}

attr ValidationRule {
	@Type(Function)
    isValid(value)
//  ^ ast.DeclField
//          ^ ast.DeclParameter
}

fn doNothingWithNothing {}
// <- ast.DeclFunc
// ^ ast.Identifier
//                      ^ ast.ExprFunc
fn doNothingWithSomething(some, thing) {}
// <- ast.DeclFunc
//                        ^ ast.DeclParameter
//                              ^ ast.DeclParameter
//                                     ^ ast.ExprFunc

@Returns(None)
// <- ast.DeclAttrInstance
@big.O("constant")
fn greet(@String name) {}
// <- ast.DeclFunc
//       ^ ast.DeclAttrInstance
//               ^ ast.DeclParameter
//                     ^ ast.ExprFunc

fn example() {
    const x = 4
//  ^ ast.DeclConstant
//        ^ ast.Identifier
//            ^ ast.ExprInt
    if True {
//  ^ ast.StmtIf
//     ^ ast.ExprIdentifier
        return 3
//      ^ ast.StmtReturn
//             ^ ast.ExprInt
    }
    return 4
//  ^ ast.StmtReturn
}

`

	sourceFile := prepareSourceFileParsing(t, contents)
	h := syncheck.NewHarness(func(lineOffset int, line string, assert syncheck.Assertion) bool {
		var relevantChildren []ast.Node
		sourceFile.EnumerateChildNodes(func(child ast.Node) {
			tok := child.TokenLiteral()

			if tok.Source.Offset <= assert.SourceOffset-1 && assert.SourceOffset <= tok.Source.Offset+len(tok.Literal)+1 {
				relevantChildren = append(relevantChildren, child)
			}
		})
		for _, child := range relevantChildren {
			candidate := strings.TrimPrefix(fmt.Sprintf("%T", child), "*")
			if candidate == assert.Value {
				return !assert.Negated
			}
		}
		childTypes := make([]string, len(relevantChildren))
		for i, child := range relevantChildren {
			childTypes[i] = strings.TrimPrefix(fmt.Sprintf("%T", child), "*")
		}
		t.Errorf("no alternative found, want %q, got one of %q", assert.Value, childTypes)
		return false
	})
	err := h.Test(contents)
	if err != nil {
		t.Error(err)
	}
}

func TestParseForStatements(t *testing.T) {
	contents := `
fn sample() {
	for { break }
	for true { continue }
	for item <- items { break }
}
`

	sourceFile := prepareSourceFileParsing(t, contents)
	symbol, ok := sourceFile.Decls.Parent.Symbols["sample"]
	if !ok || symbol.Decl == nil {
		t.Fatal("expected function declaration")
	}
	decl, ok := symbol.Decl.(*ast.DeclFunc)
	if !ok {
		t.Fatalf("declaration is %T, want *ast.DeclFunc", symbol.Decl)
	}
	if decl.Impl == nil {
		t.Fatal("expected function implementation")
	}
	if len(decl.Impl.Impl) != 3 {
		t.Fatalf("expected three statements, got %d", len(decl.Impl.Impl))
	}

	first, ok := decl.Impl.Impl[0].(ast.StmtFor)
	if !ok {
		t.Fatalf("statement is %T, want ast.StmtFor", decl.Impl.Impl[0])
	}
	if first.Condition != nil {
		t.Fatalf("expected no condition, got %T", first.Condition)
	}
	if first.CollectionIdent != nil {
		t.Fatalf("expected no collection identifier, got %T", first.CollectionIdent)
	}
	if first.CollectionExpr != nil {
		t.Fatalf("expected no collection expr, got %T", first.CollectionExpr)
	}
	if len(first.Body) != 1 {
		t.Fatalf("expected one body statement, got %d", len(first.Body))
	}
	if _, ok := first.Body[0].(ast.StmtBreak); !ok {
		t.Fatalf("body statement is %T, want ast.StmtBreak", first.Body[0])
	}

	second, ok := decl.Impl.Impl[1].(ast.StmtFor)
	if !ok {
		t.Fatalf("statement is %T, want ast.StmtFor", decl.Impl.Impl[1])
	}
	if second.Condition == nil {
		t.Fatal("expected condition expression")
	}
	if second.CollectionIdent != nil {
		t.Fatalf("expected no collection identifier, got %T", second.CollectionIdent)
	}
	if second.CollectionExpr != nil {
		t.Fatalf("expected no collection expr, got %T", second.CollectionExpr)
	}
	if len(second.Body) != 1 {
		t.Fatalf("expected one body statement, got %d", len(second.Body))
	}
	if _, ok := second.Body[0].(ast.StmtContinue); !ok {
		t.Fatalf("body statement is %T, want ast.StmtContinue", second.Body[0])
	}

	third, ok := decl.Impl.Impl[2].(ast.StmtFor)
	if !ok {
		t.Fatalf("statement is %T, want ast.StmtFor", decl.Impl.Impl[2])
	}
	if third.Condition != nil {
		t.Fatalf("expected no condition, got %T", third.Condition)
	}
	if third.CollectionIdent == nil {
		t.Fatal("expected collection identifier")
	}
	if third.CollectionIdent.Value != "item" {
		t.Fatalf("expected collection identifier item, got %q", third.CollectionIdent.Value)
	}
	if third.CollectionExpr == nil {
		t.Fatal("expected collection expression")
	}
	if _, ok := third.CollectionExpr.(*ast.ExprIdentifier); !ok {
		t.Fatalf("collection expression is %T, want *ast.ExprIdentifier", third.CollectionExpr)
	}
	if len(third.Body) != 1 {
		t.Fatalf("expected one body statement, got %d", len(third.Body))
	}
	if _, ok := third.Body[0].(ast.StmtBreak); !ok {
		t.Fatalf("body statement is %T, want ast.StmtBreak", third.Body[0])
	}
}

func TestParseForStatementsEmptyBody(t *testing.T) {
	contents := `
fn sample() {
	for {
	}
}
`

	sourceFile := prepareSourceFileParsing(t, contents)
	symbol, ok := sourceFile.Decls.Parent.Symbols["sample"]
	if !ok || symbol.Decl == nil {
		t.Fatal("expected function declaration")
	}
	decl, ok := symbol.Decl.(*ast.DeclFunc)
	if !ok {
		t.Fatalf("declaration is %T, want *ast.DeclFunc", symbol.Decl)
	}
	if decl.Impl == nil {
		t.Fatal("expected function implementation")
	}
	if len(decl.Impl.Impl) != 1 {
		t.Fatalf("expected one statement, got %d", len(decl.Impl.Impl))
	}
	stmt, ok := decl.Impl.Impl[0].(ast.StmtFor)
	if !ok {
		t.Fatalf("statement is %T, want ast.StmtFor", decl.Impl.Impl[0])
	}
	if len(stmt.Body) != 0 {
		t.Fatalf("expected empty body, got %d statements", len(stmt.Body))
	}
}

func TestParseForStatementsMultipleBody(t *testing.T) {
	contents := `
fn sample() {
	for {
		const x = 1
		const y = 2
		return x
	}
}
`

	sourceFile := prepareSourceFileParsing(t, contents)
	symbol, ok := sourceFile.Decls.Parent.Symbols["sample"]
	if !ok || symbol.Decl == nil {
		t.Fatal("expected function declaration")
	}
	decl, ok := symbol.Decl.(*ast.DeclFunc)
	if !ok {
		t.Fatalf("declaration is %T, want *ast.DeclFunc", symbol.Decl)
	}
	if decl.Impl == nil {
		t.Fatal("expected function implementation")
	}
	if len(decl.Impl.Impl) != 1 {
		t.Fatalf("expected one statement, got %d", len(decl.Impl.Impl))
	}
	stmt, ok := decl.Impl.Impl[0].(ast.StmtFor)
	if !ok {
		t.Fatalf("statement is %T, want ast.StmtFor", decl.Impl.Impl[0])
	}
	if len(stmt.Body) != 3 {
		t.Fatalf("expected three body statements, got %d", len(stmt.Body))
	}
	if _, ok := stmt.Body[0].(*ast.DeclConstant); !ok {
		t.Fatalf("statement is %T, want *ast.DeclConstant", stmt.Body[0])
	}
	if _, ok := stmt.Body[1].(*ast.DeclConstant); !ok {
		t.Fatalf("statement is %T, want *ast.DeclConstant", stmt.Body[1])
	}
	if _, ok := stmt.Body[2].(*ast.StmtReturn); !ok {
		t.Fatalf("statement is %T, want *ast.StmtReturn", stmt.Body[2])
	}
}

func TestParseForStatementsNestedIfBreakContinue(t *testing.T) {
	contents := `
fn sample() {
	for {
		if True {
			continue
		} else {
			break
		}
	}
}
`

	sourceFile := prepareSourceFileParsing(t, contents)
	symbol, ok := sourceFile.Decls.Parent.Symbols["sample"]
	if !ok || symbol.Decl == nil {
		t.Fatal("expected function declaration")
	}
	decl, ok := symbol.Decl.(*ast.DeclFunc)
	if !ok {
		t.Fatalf("declaration is %T, want *ast.DeclFunc", symbol.Decl)
	}
	if decl.Impl == nil {
		t.Fatal("expected function implementation")
	}
	if len(decl.Impl.Impl) != 1 {
		t.Fatalf("expected one statement, got %d", len(decl.Impl.Impl))
	}
	stmt, ok := decl.Impl.Impl[0].(ast.StmtFor)
	if !ok {
		t.Fatalf("statement is %T, want ast.StmtFor", decl.Impl.Impl[0])
	}
	if len(stmt.Body) != 1 {
		t.Fatalf("expected one body statement, got %d", len(stmt.Body))
	}
	ifStmt, ok := stmt.Body[0].(ast.StmtIf)
	if !ok {
		t.Fatalf("statement is %T, want ast.StmtIf", stmt.Body[0])
	}
	if len(ifStmt.IfBlock) != 1 {
		t.Fatalf("expected one if body statement, got %d", len(ifStmt.IfBlock))
	}
	if _, ok := ifStmt.IfBlock[0].(ast.StmtContinue); !ok {
		t.Fatalf("if statement is %T, want ast.StmtContinue", ifStmt.IfBlock[0])
	}
	if len(ifStmt.ElseBlock) != 1 {
		t.Fatalf("expected one else body statement, got %d", len(ifStmt.ElseBlock))
	}
	if _, ok := ifStmt.ElseBlock[0].(ast.StmtBreak); !ok {
		t.Fatalf("else statement is %T, want ast.StmtBreak", ifStmt.ElseBlock[0])
	}
}

// TestParseAttrNoPanic ensures the parser does not panic on malformed
// attributes (regression test for invariant panic in parseStaticIdentifierReference).
func TestParseAttrNoPanic(t *testing.T) {
	inputs := []string{
		"@@Numeric",
		"@(Numeric",
		"@",
		"@ ",
	}
	for _, input := range inputs {
		t.Run(fmt.Sprintf("input=%q", input), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("parser panicked on %q: %v", input, r)
				}
			}()
			module := ast.MakeContextModule(registry.LogicalURI("test"))
			l, err := lexer.New(staticmodule.NewSourceString("testing:///test.zirr", input))
			if err != nil {
				t.Fatal(err)
			}
			p := parser.NewSourceParser(l, module.Decls, "test.zirr")
			p.ParseSourceFile()
			// We expect parse errors but no panic.
		})
	}
}

// TestParseConstVarTypes verifies that `const` produces DeclConstant and `var` produces DeclVariable.
func TestParseConstVarTypes(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		varName   string
		wantConst bool
	}{
		{
			name:      "const global binding",
			input:     "const x = 42",
			varName:   "x",
			wantConst: true,
		},
		{
			name:      "var global binding",
			input:     "var x = 42",
			varName:   "x",
			wantConst: false,
		},
		{
			name:      "const local binding",
			input:     "fn f() { const x = 42 x }",
			varName:   "x",
			wantConst: true,
		},
		{
			name:      "var local binding",
			input:     "fn f() { var x = 42 x }",
			varName:   "x",
			wantConst: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			sym, ok := srcFile.Decls.Parent.Symbols[tt.varName]
			if !ok {
				// Also look inside the function symbol table.
				for _, s := range srcFile.Decls.Parent.Symbols {
					if s.ChildTable != nil {
						if inner, ok2 := s.ChildTable.Symbols[tt.varName]; ok2 {
							sym = inner
							ok = true
							break
						}
					}
				}
			}
			if !ok {
				t.Fatalf("symbol %q not found", tt.varName)
			}
			if tt.wantConst {
				if _, ok := sym.Decl.(*ast.DeclConstant); !ok {
					t.Errorf("expected *ast.DeclConstant for const, got %T", sym.Decl)
				}
			} else {
				if _, ok := sym.Decl.(*ast.DeclVariable); !ok {
					t.Errorf("expected *ast.DeclVariable for var, got %T", sym.Decl)
				}
			}
		})
	}
}

// TestParseDeclOverview verifies DeclOverview returns the correct keyword.
func TestParseDeclOverview(t *testing.T) {
	tests := []struct {
		input    string
		varName  string
		wantWord string
	}{
		{"const answer = 42", "answer", "const answer"},
		{"var counter = 0", "counter", "var counter"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			sym, ok := srcFile.Decls.Parent.Symbols[tt.varName]
			if !ok {
				t.Fatalf("symbol %q not found", tt.varName)
			}
			type overviewable interface{ DeclOverview() string }
			ov, ok := sym.Decl.(overviewable)
			if !ok {
				t.Fatalf("declaration does not implement DeclOverview")
			}
			if got := ov.DeclOverview(); got != tt.wantWord {
				t.Errorf("DeclOverview() = %q, want %q", got, tt.wantWord)
			}
		})
	}
}

func prepareSourceFileParsing(t *testing.T, input string) *ast.SourceFile {
	t.Helper()

	module := ast.MakeContextModule(registry.LogicalURI("test"))
	parentTable := module.Decls
	l, err := lexer.New(staticmodule.NewSourceString("testing:///test.zirr", input))
	if err != nil {
		t.Fatal(err)
	}
	p := parser.NewSourceParser(l, parentTable, "test.zirr")

	srcFile := p.ParseSourceFile()
	checkParserErrors(t, p, input)
	return srcFile
}

func checkParserErrors(t *testing.T, p *parser.Parser, contents string) {
	t.Helper()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			src := err.Token.Source
			contentsBeforeOffset := contents[:src.Offset]
			loc := strings.Count(contentsBeforeOffset, "\n")
			lastLineIndex := strings.LastIndex(contentsBeforeOffset, "\n")
			col := src.Offset - lastLineIndex
			relevantLine, _, _ := strings.Cut(contents[lastLineIndex+1:], "\n")

			t.Errorf("%s:%d:%d: %s\n\n  %s\n  %s^\n  %s\n\n", err.Token.Source.File, loc, col, err.Summary, relevantLine, strings.Repeat(" ", col-1), err.Details)
		}
		t.FailNow()
	}
}

// Symbol errors are reported by the analyzer.
