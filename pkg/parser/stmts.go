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

func (p *Parser) parseStatementInContext(pos StatementPosition, annos ast.AttributeChain) (ast.Statement, []ast.StatementDeclaration) {
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
