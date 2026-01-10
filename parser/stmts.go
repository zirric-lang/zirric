package parser

import (
	"code.knabel.dev/zirric-lang/zirric/ast"
	"code.knabel.dev/zirric-lang/zirric/token"
)

type (
	StatementPosition int
)

const (
	_ StatementPosition = iota

	IN_INITIAL
	IN_GLOBAL
	IN_ENUM
	IN_DATA
	IN_EXTERN
	IN_FUNC
	IN_FOR
	IN_SWITCH
)

func (p *Parser) parseStatementInContext(pos StatementPosition, annos ast.AnnotationChain) (ast.Statement, []ast.StatementDeclaration) {
	switch p.curToken.Type {
	case token.MODULE:
		return p.parseModuleDecl(pos, annos), nil
	case token.EXTERN:
		return p.parseExternDecl(pos, annos), nil
	case token.ENUM:
		return p.parseEnumDecl(pos, annos)
	case token.DATA:
		return p.parseDataDecl(pos, annos), nil
	case token.ANNOTATION:
		return p.parseAnnotationDecl(pos, annos), nil
	case token.FUNCTION:
		return p.parseFunctionDecl(pos, annos), nil
	case token.LET:
		return p.parseVariableDecl(pos, annos), nil
	case token.IMPORT:
		return p.parseImportDecl(pos, annos), nil
	case token.AT:
		return p.parseAnnotatedStatementDeclaration(pos)
	case token.IF:
		return p.parseStatementIf(pos), nil
	case token.RETURN:
		return p.parseStatementReturn(pos), nil
	default:
		if _, ok := p.prefixParsers[p.curToken.Type]; ok {
			if annos != nil {
				p.errCannotBeAnnotated()
			}
			return p.parseExprStmt(), nil
		}

		prefixes := []token.TokenType{
			token.ENUM, token.DATA, token.MODULE, token.EXTERN, token.FUNCTION, token.IMPORT, token.AT, token.LET, token.IF, token.FOR,
		}
		for t := range p.prefixParsers {
			prefixes = append(prefixes, t)
		}
		p.errUnexpectedToken(prefixes...)
		return nil, nil
	}
}
