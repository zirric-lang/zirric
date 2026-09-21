package token

import "fmt"

// Source locates a token in the file it was read from.
//
// Offset is the byte index, which is what slicing the source text needs.
// Line and Column are 1-based and are what a person or an editor needs: printing an offset where a line number belongs sends a reader to the wrong place, so the two are kept apart rather than one standing in for the other.
// Column counts bytes rather than characters, matching the convention most compilers follow.
type Source struct {
	File   string
	Offset int
	Line   int
	Column int
}

func MakeSource(
	fileName string,
	offset int,
	line int,
	column int,
) *Source {
	return &Source{
		File:   fileName,
		Offset: offset,
		Line:   line,
		Column: column,
	}
}

// String renders the location the way editors and tooling expect to read it, as file:line:column.
// A Source with no line recorded falls back to the file alone rather than printing a zero that looks like a real position.
func (s *Source) String() string {
	if s == nil {
		return ""
	}
	if s.Line <= 0 {
		return s.File
	}
	return fmt.Sprintf("%s:%d:%d", s.File, s.Line, s.Column)
}
