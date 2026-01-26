package syncheck

import (
	"bufio"
	"strings"
)

// to test tokens and highlighting, we follow the format:
// https://tree-sitter.github.io/tree-sitter/syntax-highlighting#unit-testing
// Test lines always connect to the previous non-test-line
// There are Carot ^ tests testing the token in the same column above
// And arrow <- tests that test at the column of the comment `//`
// Negation with `!`
// Test lines will stripped from the output.

// TODO: migrate to offset based assertions
type Assertion struct {
	Line         int
	Column       int
	Value        string
	Negated      bool
	SourceOffset int
	SourceLine   int
}

func ParseAssertions(str string) []Assertion {
	var assertions []Assertion
	scanner := bufio.NewScanner(strings.NewReader(str))
	var (
		beginOfNextLine = 0
		offsetUnderTest = 0
		lineUnderTest   int
	)
	for i := 1; scanner.Scan(); i++ {
		line := scanner.Text()
		beginOfNextLine += len(line) + 1

		assert := extractBeginAssertionFromLine(lineUnderTest, line)
		if assert != nil {
			assert.SourceLine = i
			assert.SourceOffset = offsetUnderTest
			assertions = append(assertions, *assert)
			continue
		}

		assert = extractCarotAssertionFromLine(lineUnderTest, line)
		if assert != nil {
			assert.SourceLine = i
			assert.SourceOffset = offsetUnderTest + assert.Column
			assertions = append(assertions, *assert)
			continue
		}

		lineUnderTest = i
		offsetUnderTest = beginOfNextLine - len(line)
	}
	return assertions
}

func extractBeginAssertionFromLine(lnum int, line string) *Assertion {
	_, value, found := strings.Cut(line, "<-")
	if !found {
		return nil
	}

	assert := &Assertion{
		Line:   lnum,
		Column: 1,
	}
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "!") {
		assert.Negated = true
		value = strings.TrimPrefix(value, "!")
	}
	assert.Value = value
	return assert
}

func extractCarotAssertionFromLine(lnum int, line string) *Assertion {
	prefix, value, found := strings.Cut(line, "^")
	if !found {
		return nil
	}

	col := len(prefix) + 1
	assert := &Assertion{
		Line:   lnum,
		Column: col,
	}
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "!") {
		assert.Negated = true
		value = strings.TrimPrefix(value, "!")
	}
	assert.Value = value
	return assert
}

// to test parsing, we try to follow this format:
// https://tree-sitter.github.io/tree-sitter/creating-parsers#command-test
