package parser

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

type (
	StatementPosition int
)

const (
	_ StatementPosition = iota

	IN_INITIAL
	IN_GLOBAL
	IN_UNION
	IN_DATA
	IN_EXTERN
	IN_FUNC
	IN_FOR
	IN_SWITCH
)

// parseStatementInContext parses one statement, attaching the doc comment written above it.
// The comment is read before the statement is parsed, because by then the tokens it was leading have been consumed.
func (p *Parser) parseStatementInContext(pos StatementPosition, annos ast.AttributeChain) (ast.Statement, []ast.StatementDeclaration) {
	docs := p.leadingDocs(pos)
	stmt, children := p.parseStatementNode(pos, annos)
	applyDocs(stmt, docs)
	return stmt, children
}

// leadingDocs reads the doc comment above the current token, but only where a declaration can be documented.
// A local binding inside a function body is reachable from nowhere else, so documenting one would only cost an attribute nothing can read.
func (p *Parser) leadingDocs(pos StatementPosition) *ast.Docs {
	if pos != IN_INITIAL && pos != IN_GLOBAL {
		return nil
	}
	return ast.MakeDocs(ast.LeadingDocComments(p.curToken))
}

// applyDocs hands a doc comment to the declaration it was written above, if that declaration can carry one.
// A declaration with attributes can be documented above them and again between them and the keyword, and the parser reaches the lower comment first. Both document the same declaration, so they are joined in the order they were written rather than one replacing the other.
func applyDocs(node ast.Node, docs *ast.Docs) {
	if docs == nil || docs.Content == "" {
		return
	}
	target, ok := node.(ast.Documentable)
	if !ok {
		return
	}
	if existing := target.ProvidedDocs(); existing != nil && existing.Content != "" {
		target.SetDocs(&ast.Docs{Content: docs.Content + "\n" + existing.Content})
		return
	}
	target.SetDocs(docs)
}

func (p *Parser) parseStatementNode(pos StatementPosition, annos ast.AttributeChain) (ast.Statement, []ast.StatementDeclaration) {
	switch p.curToken.Type {
	case token.MODULE:
		return p.parseModuleDecl(pos, annos), nil
	case token.EXTERN:
		return p.parseExternDecl(pos, annos), nil
	case token.UNION:
		return p.parseUnionDecl(pos, annos)
	case token.DATA:
		return p.parseDataDecl(pos, annos), nil
	case token.ATTRIBUTE:
		return p.parseAttrDecl(pos, annos), nil
	case token.FUNCTION:
		if p.peekIs(token.LPAREN) {
			// fn(...) { body } — anonymous closure expression
			if annos != nil {
				p.errCannotBeAnnotated()
			}
			return p.parseExprStmt(), nil
		}
		return p.parseFunctionDecl(pos, annos), nil
	case token.CONST, token.VAR:
		return p.parseVariableDecl(pos, annos), nil
	case token.IMPORT:
		return p.parseImportDecl(pos, annos), nil
	case token.AT:
		return p.parseAnnotatedStatementDeclaration(pos)
	case token.IF:
		return p.parseStatementIf(pos), nil
	case token.FOR:
		return p.parseStatementFor(pos), nil
	case token.BREAK:
		return p.parseStatementBreak(pos), nil
	case token.CONTINUE:
		return p.parseStatementContinue(pos), nil
	case token.RETURN:
		return p.parseStatementReturn(pos), nil
	case token.SWITCH:
		return p.parseStatementSwitch(pos), nil
	default:
		if _, ok := p.prefixParsers[p.curToken.Type]; ok {
			if annos != nil {
				p.errCannotBeAnnotated()
			}
			return p.parseExprStmt(), nil
		}

		p.errExpected("a statement or an expression")
		return nil, nil
	}
}
