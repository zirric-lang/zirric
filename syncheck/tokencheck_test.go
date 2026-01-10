package syncheck_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/syncheck"
)

func TestParsingAssertions(t *testing.T) {
	content := `
	data xyz
	`
	asserts := syncheck.ParseAssertions(content)
	if len(asserts) != 0 {
		t.Errorf("no asserts expected, got: %v", asserts)
	}
	content = "data xyz {\n" + // len 11
		"//   ^ label0\n" + // len 14
		"// <- label1\n" + // len 13
		"}\n" + // len 2
		"^ label2" // len 8

	asserts = syncheck.ParseAssertions(content)
	if len(asserts) != 3 {
		t.Errorf("three assertions expected, got: %v", asserts)
	}
	if asserts[0].Line != 1 || asserts[0].Column != 6 || asserts[0].SourceOffset != 7 {
		t.Errorf("expected matcher [0] on line 1 column 6 offset 7, got: %+v", asserts[0])
	}
	if asserts[1].Line != 1 || asserts[1].Column != 1 || asserts[1].SourceOffset != 1 {
		t.Errorf("expected matcher [1] on line 1 column 1 offset 1, got: %+v", asserts[1])
	}
	if asserts[2].Line != 4 || asserts[1].Column != 1 || asserts[2].SourceOffset != 40 {
		t.Errorf("expected matcher [2] on line 4 column 1 offset 40, got: %+v", asserts[2])
	}
}
