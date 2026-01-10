package parser_test

import (
	"fmt"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/ast"
	"code.knabel.dev/zirric-lang/zirric/lexer"
	"code.knabel.dev/zirric-lang/zirric/parser"
	"code.knabel.dev/zirric-lang/zirric/registry/staticmodule"
	"code.knabel.dev/zirric-lang/zirric/syncheck"
)

func TestParseSourceFile(t *testing.T) {
	contents := `
module testingmodule
// <- ast.DeclModule

import json
// <- ast.DeclImport
import big
// <- ast.DeclImport

@json.Type(json.Null)
// <- ast.DeclAnnotationInstance
data None
// <- ast.DeclData

@json.Inline
data Some {
// <- ast.DeclData
   value
// ^ ast.DeclField
}

@json.Inline()
enum Optional {
	None
	Some
}

extern let b // this is an extern constant
// <- ast.DeclExternValue
//         ^ ast.Identifier

extern func doSomething()
// <- ast.DeclExternFunc
	
extern func doSomethingWith(argument)
// <- ast.DeclExternFunc
//          ^ ast.Identifier
//                          ^ ast.DeclParameter

extern type SomeType {
// <- ast.DeclExternType
    name
//  ^ ast.DeclField
}

extern type SomeEmptyType {}
// <- ast.DeclExternType
//          ^ ast.Identifier

annotation Type {
// <- ast.DeclAnnotation
	@AnyType
	value
//  ^ ast.DeclField
}

annotation ValidationRule {
	@Type(Function)
    isValid(value)
//  ^ ast.DeclField
//          ^ ast.DeclParameter
}

func doNothingWithNothing {}
// <- ast.DeclFunc
//   ^ ast.Identifier
//                        ^ ast.ExprFunc
func doNothingWithSomething(some, thing) {}
// <- ast.DeclFunc
//                          ^ ast.DeclParameter
//                                ^ ast.DeclParameter
//                                       ^ ast.ExprFunc

@Returns(None)
// <- ast.DeclAnnotationInstance
@big.O("constant")
func greet(@String name) {}
// <- ast.DeclFunc
//         ^ ast.DeclAnnotationInstance
//                 ^ ast.DeclParameter
//                       ^ ast.ExprFunc

func example() {
    let x = 4
//  ^ ast.DeclVariable
//      ^ ast.Identifier
//          ^ ast.ExprInt
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

func prepareSourceFileParsing(t *testing.T, input string) *ast.SourceFile {
	t.Helper()

	parentTable := ast.MakeSymbolTable(nil, ast.Identifier{
		Value: "test",
	})
	l, err := lexer.New(staticmodule.NewSourceString("testing:///test.zirr", input))
	if err != nil {
		t.Fatal(err)
	}
	p := parser.NewSourceParser(l, parentTable, "test.zirr")

	srcFile := p.ParseSourceFile()
	checkParserErrors(t, p, input)
	checkSymbolErrors(t, p, input)
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

func checkSymbolErrors(t *testing.T, p *parser.Parser, contents string) {
	t.Helper()

	symerrs := p.SymbolErrors()
	if len(symerrs) > 0 {
		for _, err := range symerrs {
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
