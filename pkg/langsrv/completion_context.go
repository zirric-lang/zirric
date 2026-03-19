package langsrv

import (
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// completionScope describes the syntactic context at the cursor position,
// driving which completions are appropriate.
type completionScope struct {
	inFunc        bool            // cursor is inside a function body
	inFor         bool            // cursor is inside a for loop
	isTopLevel    bool            // cursor is NOT inside any function body
	isTypeExpr    bool            // cursor follows ':' or '->' (type annotation position)
	isImportBlock bool            // cursor is inside import { ... }
	isFirstStmt   bool            // no statements precede the cursor (for 'mod')
	usedAttrs     map[string]bool // attr names already present in the current @-chain
	dirName       string          // base name of the current file's directory (for mod completion)
}

// detectCompletionScope analyzes the text and AST to determine cursor context.
func detectCompletionScope(
	text string,
	pos protocol.Position,
	module *ast.ContextModule,
	sourceURI string,
	cursorOffset int,
) completionScope {
	scope := completionScope{}

	// Determine if cursor is inside a function or for loop by walking the AST.
	scope.inFunc, scope.inFor = detectEnclosingScope(text, module, sourceURI, cursorOffset)
	scope.isTopLevel = !scope.inFunc

	// Detect type expression context by scanning backward.
	scope.isTypeExpr = isTypeExprContext(text, pos)

	// Detect import block context.
	scope.isImportBlock = isImportBlockContext(text, cursorOffset)

	// Detect first statement position (no declarations before cursor).
	scope.isFirstStmt = isFirstStatementPosition(module, sourceURI, cursorOffset)

	// Collect attrs already used in the current @-chain before cursor.
	scope.usedAttrs = collectUsedAttrs(text, pos)

	return scope
}

// detectEnclosingScope walks the AST to determine if the cursor is inside
// a function body and/or a for loop. Uses brace-matching in the source text
// to accurately determine body boundaries.
func detectEnclosingScope(text string, module *ast.ContextModule, sourceURI string, cursorOffset int) (inFunc, inFor bool) {
	// Check function declarations.
	for _, sym := range module.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		nameToken := sym.Decl.DeclName().Token
		if nameToken.Source == nil || nameToken.Source.File != sourceURI {
			continue
		}
		if nodeOffset(sym.Decl) >= cursorOffset {
			continue
		}

		switch d := sym.Decl.(type) {
		case ast.DeclFunc:
			if d.Impl != nil && cursorInsideBraces(text, nodeOffset(sym.Decl), cursorOffset) {
				inFunc = true
				if blockContainsForWithBraces(text, d.Impl.Impl, cursorOffset) {
					inFor = true
				}
				return
			}
		case *ast.DeclFunc:
			if d.Impl != nil && cursorInsideBraces(text, nodeOffset(sym.Decl), cursorOffset) {
				inFunc = true
				if blockContainsForWithBraces(text, d.Impl.Impl, cursorOffset) {
					inFor = true
				}
				return
			}
		}
	}

	// Check top-level statements (global for-loops, if-blocks, etc.).
	for _, sf := range module.Files {
		if sf.Path != sourceURI {
			continue
		}
		for _, stmt := range sf.Statements {
			if nodeOffset(stmt) >= cursorOffset {
				continue
			}
			switch s := stmt.(type) {
			case ast.StmtFor:
				if cursorInsideBraces(text, nodeOffset(stmt), cursorOffset) {
					inFor = true
					return
				}
			case *ast.StmtFor:
				if cursorInsideBraces(text, nodeOffset(stmt), cursorOffset) {
					inFor = true
					return
				}
			case ast.StmtIf:
				if topLevelBlockContainsForWithBraces(text, s.IfBlock, cursorOffset) ||
					topLevelBlockContainsForWithBraces(text, s.ElseBlock, cursorOffset) {
					inFor = true
					return
				}
				for _, ei := range s.ElseIf {
					if topLevelBlockContainsForWithBraces(text, ei.Block, cursorOffset) {
						inFor = true
						return
					}
				}
			case *ast.StmtIf:
				if topLevelBlockContainsForWithBraces(text, s.IfBlock, cursorOffset) ||
					topLevelBlockContainsForWithBraces(text, s.ElseBlock, cursorOffset) {
					inFor = true
					return
				}
				for _, ei := range s.ElseIf {
					if topLevelBlockContainsForWithBraces(text, ei.Block, cursorOffset) {
						inFor = true
						return
					}
				}
			}
		}
	}

	return false, false
}

// topLevelBlockContainsForWithBraces checks if the cursor is inside a for loop
// within the given block at the top level (no enclosing function).
func topLevelBlockContainsForWithBraces(text string, block ast.Block, cursorOffset int) bool {
	for _, stmt := range block {
		if nodeOffset(stmt) >= cursorOffset {
			break
		}
		switch s := stmt.(type) {
		case ast.StmtFor:
			if cursorInsideBraces(text, nodeOffset(stmt), cursorOffset) {
				return true
			}
		case *ast.StmtFor:
			if cursorInsideBraces(text, nodeOffset(stmt), cursorOffset) {
				return true
			}
		case ast.StmtIf:
			if topLevelBlockContainsForWithBraces(text, s.IfBlock, cursorOffset) ||
				topLevelBlockContainsForWithBraces(text, s.ElseBlock, cursorOffset) {
				return true
			}
		case *ast.StmtIf:
			if topLevelBlockContainsForWithBraces(text, s.IfBlock, cursorOffset) ||
				topLevelBlockContainsForWithBraces(text, s.ElseBlock, cursorOffset) {
				return true
			}
		}
	}
	return false
}

// cursorInsideBraces scans forward from startOffset in text to find the first '{',
// then matches braces. Returns true if cursorOffset falls within the matched braces.
func cursorInsideBraces(text string, startOffset, cursorOffset int) bool {
	// Find the first '{' after startOffset.
	openIdx := -1
	for i := startOffset; i < len(text); i++ {
		if text[i] == '{' {
			openIdx = i
			break
		}
	}
	if openIdx < 0 || cursorOffset <= openIdx {
		return false
	}

	// Match braces to find the closing '}'.
	depth := 1
	for i := openIdx + 1; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return cursorOffset > openIdx && cursorOffset <= i
			}
		}
	}
	// Unclosed brace — cursor is inside if after the opening brace.
	return cursorOffset > openIdx
}

// blockContainsForWithBraces checks if the cursor is inside a for loop body
// within the given block, using brace-matching for accurate boundary detection.
func blockContainsForWithBraces(text string, block ast.Block, cursorOffset int) bool {
	for _, stmt := range block {
		if nodeOffset(stmt) >= cursorOffset {
			break
		}
		switch s := stmt.(type) {
		case ast.StmtFor:
			if cursorInsideBraces(text, nodeOffset(stmt), cursorOffset) {
				return true
			}
		case *ast.StmtFor:
			if cursorInsideBraces(text, nodeOffset(stmt), cursorOffset) {
				return true
			}
		case ast.StmtIf:
			if blockContainsForWithBraces(text, s.IfBlock, cursorOffset) {
				return true
			}
			for _, ei := range s.ElseIf {
				if blockContainsForWithBraces(text, ei.Block, cursorOffset) {
					return true
				}
			}
			if blockContainsForWithBraces(text, s.ElseBlock, cursorOffset) {
				return true
			}
		case *ast.StmtIf:
			if blockContainsForWithBraces(text, s.IfBlock, cursorOffset) {
				return true
			}
			for _, ei := range s.ElseIf {
				if blockContainsForWithBraces(text, ei.Block, cursorOffset) {
					return true
				}
			}
			if blockContainsForWithBraces(text, s.ElseBlock, cursorOffset) {
				return true
			}
		}
	}
	return false
}

// isTypeExprContext scans backward from the cursor to detect if we're in a
// type annotation position. Recognizes:
//   - `: Type`, `-> Type` (parameter/return type hints)
//   - `[Type]`, `[Key: Value]` (array/dict type elements)
//   - `fn(Type, Type) -> Type` (function type expressions)
//   - `@Attr1 @Attr2` (attribute chains in type position)
func isTypeExprContext(text string, pos protocol.Position) bool {
	line := lineAtPosition(text, pos)
	col := int(pos.Character)
	if col > len(line) {
		col = len(line)
	}

	i := col - 1

	// Skip backward through tokens that can appear in type expressions.
	// This handles chains like `@Attr1 @Attr2` and nested brackets.
	for {
		// Skip current identifier.
		for i >= 0 && isSimpleIdentByte(line[i]) {
			i--
		}
		// Skip '@' prefix.
		if i >= 0 && line[i] == '@' {
			i--
		}
		// Skip whitespace.
		for i >= 0 && (line[i] == ' ' || line[i] == '\t') {
			i--
		}
		if i < 0 {
			return false
		}

		// Terminal checks — these delimiters indicate a type expression context.
		switch line[i] {
		case ':':
			return true
		case '[':
			return true
		case '(':
			return true
		case ',':
			return true
		}
		// Check for '->' (return type).
		if line[i] == '>' && i > 0 && line[i-1] == '-' {
			return true
		}

		// '.' in a qualified name (e.g. @alias.Attr) — skip backward past the dot
		// and the alias identifier before it, then continue scanning.
		if line[i] == '.' {
			i--
			// Skip the alias identifier before the dot.
			for i >= 0 && isSimpleIdentByte(line[i]) {
				i--
			}
			// Skip '@' prefix before the alias.
			if i >= 0 && line[i] == '@' {
				i--
			}
			continue
		}

		// If the preceding token ends with an identifier (another @Attr in a chain),
		// continue scanning backward.
		if i >= 0 && isSimpleIdentByte(line[i]) {
			continue
		}
		switch line[i] {
		case ']':
			// `[Type]` followed by more. Skip the balanced brackets.
			depth := 1
			i--
			for i >= 0 && depth > 0 {
				switch line[i] {
				case ']':
					depth++
				case '[':
					depth--
				}
				i--
			}
			continue
		case ')':
			// `fn(...)` type expression. Skip balanced parens.
			depth := 1
			i--
			for i >= 0 && depth > 0 {
				switch line[i] {
				case ')':
					depth++
				case '(':
					depth--
				}
				i--
			}
			continue
		default:
			return false
		}
	}
}

// collectUsedAttrs scans backward from the cursor through the full text to collect
// all @AttrName tokens that are part of the current attribute chain. This handles
// both type-expression chains on a single line (`: @Tag @Validated @`) and
// declaration-level attrs spread across multiple lines.
func collectUsedAttrs(text string, pos protocol.Position) map[string]bool {
	// Compute byte offset for the cursor position.
	offset := 0
	line := int(pos.Line)
	for i := 0; i < line && offset < len(text); offset++ {
		if text[offset] == '\n' {
			i++
		}
	}
	offset += int(pos.Character)
	if offset > len(text) {
		offset = len(text)
	}

	used := make(map[string]bool)
	i := offset - 1

	// Skip the partial identifier being typed (the current completion target).
	for i >= 0 && isSimpleIdentByte(text[i]) {
		i--
	}
	// Skip the current '@' (the one triggering completion).
	if i >= 0 && text[i] == '@' {
		i--
	}

	// Now scan backward collecting @Name and @Name(...) pairs.
	for {
		// Skip whitespace including newlines.
		for i >= 0 && (text[i] == ' ' || text[i] == '\t' || text[i] == '\n' || text[i] == '\r') {
			i--
		}
		if i < 0 {
			break
		}

		// If we see ')' this could be @Attr(...) — skip the balanced parens.
		if text[i] == ')' {
			depth := 1
			i--
			for i >= 0 && depth > 0 {
				switch text[i] {
				case ')':
					depth++
				case '(':
					depth--
				}
				i--
			}
			// i now points to the char before '('
			// Skip whitespace between name and '(' (shouldn't normally be any, but be safe).
			for i >= 0 && (text[i] == ' ' || text[i] == '\t') {
				i--
			}
		}

		// Try to read an identifier ending at position i.
		nameEnd := i + 1
		for i >= 0 && isSimpleIdentByte(text[i]) {
			i--
		}
		nameStart := i + 1

		if nameStart >= nameEnd {
			break
		}
		// The identifier must be preceded by '@'.
		if i < 0 || text[i] != '@' {
			break
		}
		name := text[nameStart:nameEnd]
		used[name] = true
		i-- // skip the '@'
	}

	return used
}

// isImportBlockContext detects if the cursor is inside an import { ... } block.
// Scans backward from cursor to find an unmatched '{', then checks if it's
// preceded by an import statement.
func isImportBlockContext(text string, cursorOffset int) bool {
	depth := 0
	for i := cursorOffset - 1; i >= 0; i-- {
		switch text[i] {
		case '}':
			depth++
		case '{':
			if depth == 0 {
				// Found unmatched '{'. Check if preceded by 'import'.
				prefix := strings.TrimRight(text[:i], " \t\n\r")
				return strings.HasSuffix(prefix, "import") ||
					importPrecedesBrace(prefix)
			}
			depth--
		}
	}
	return false
}

// importPrecedesBrace checks if text (ending just before '{') contains
// an import statement pattern like "import foo.bar".
func importPrecedesBrace(text string) bool {
	// Find the last newline and check if the line starts with 'import'.
	lastNL := strings.LastIndexByte(text, '\n')
	var lastLine string
	if lastNL < 0 {
		lastLine = text
	} else {
		lastLine = text[lastNL+1:]
	}
	trimmed := strings.TrimSpace(lastLine)
	return strings.HasPrefix(trimmed, "import ")
}

// isFirstStatementPosition returns true when no module-level declarations
// or statements exist before the cursor in the current file.
func isFirstStatementPosition(module *ast.ContextModule, sourceURI string, cursorOffset int) bool {
	// Check module-level declarations that belong to this file.
	for _, sym := range module.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		tok := sym.Decl.DeclName().Token
		if tok.Source != nil && tok.Source.File == sourceURI && nodeOffset(sym.Decl) >= 0 && nodeOffset(sym.Decl) < cursorOffset {
			return false
		}
	}
	// Check statements in this source file.
	for _, sf := range module.Files {
		if sf.Path != sourceURI {
			continue
		}
		for _, stmt := range sf.Statements {
			if nodeOffset(stmt) >= 0 && nodeOffset(stmt) < cursorOffset {
				return false
			}
		}
		// Also check source file's own declarations (internal symbols).
		for _, sym := range sf.Decls.Symbols {
			if sym != nil && sym.Decl != nil && nodeOffset(sym.Decl) >= 0 && nodeOffset(sym.Decl) < cursorOffset {
				return false
			}
		}
	}
	return true
}
