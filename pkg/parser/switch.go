package parser

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// parsePrattExprIs parses the `is` infix operator:
//
//	expr is TypeExpr
//
// TypeExpr can be a named type, composite type, or attribute constraints.
func (p *Parser) parsePrattExprIs(lhs ast.Expr) ast.Expr {
	isTok, _ := p.expect(token.IS)
	typeExpr := p.parseTypeHintExpr()
	if typeExpr == nil {
		return nil
	}
	return ast.MakeExprIs(isTok, lhs, typeExpr)
}

// parsePrattExprSwitch parses a switch expression.
//
//	switch value {
//	 case is Type: expr
//	 case literal: expr
//	 case _: expr
//	}
func (p *Parser) parsePrattExprSwitch() ast.Expr {
	switchTok, _ := p.expect(token.SWITCH)
	value := p.parsePrattExpr(LOWEST)
	if value == nil {
		return nil
	}
	_, ok := p.expect(token.LBRACE)
	if !ok {
		return nil
	}

	switchExpr := ast.MakeExprSwitch(switchTok, value)

	for p.curIs(token.CASE) {
		c := p.parseSwitchExprCase()
		switchExpr.AddCase(c)
	}

	_, ok = p.expect(token.RBRACE)
	if !ok {
		return nil
	}
	return switchExpr
}

func (p *Parser) parseSwitchExprCase() ast.ExprSwitchCase {
	caseTok, _ := p.expect(token.CASE)
	c := ast.ExprSwitchCase{Token: caseTok}

	p.parseSwitchCasePattern(&c.Kind, &c.TypeRef, &c.Pattern)

	p.expect(token.COLON)
	c.Body = p.parsePrattExpr(LOWEST)
	return c
}

// parseStatementSwitch parses a switch statement.
//
//	switch value {
//	 case is Type:
//	   statements...
//	 case literal:
//	   statements...
//	 case _:
//	   statements...
//	}
func (p *Parser) parseStatementSwitch(pos StatementPosition) ast.StmtSwitch {
	switchTok, _ := p.expect(token.SWITCH)
	value := p.parseExpr()
	p.expect(token.LBRACE)

	switchStmt := ast.MakeStmtSwitch(switchTok, value)

	for p.curIs(token.CASE) {
		c := p.parseSwitchStmtCase(pos)
		switchStmt.AddCase(c)
	}

	p.expect(token.RBRACE)
	return *switchStmt
}

func (p *Parser) parseSwitchStmtCase(pos StatementPosition) ast.StmtSwitchCase {
	caseTok, _ := p.expect(token.CASE)
	c := ast.StmtSwitchCase{Token: caseTok}

	p.parseSwitchCasePattern(&c.Kind, &c.TypeRef, &c.Pattern)

	p.expect(token.COLON)
	c.Body = p.parseSwitchCaseBlock(pos)
	return c
}

// parseSwitchCasePattern parses the pattern after `case` and before `:`.
// Supports: `is TypeExpr`, `_`, or an expression.
// All type matching (named, composite, attribute) requires `is` keyword.
func (p *Parser) parseSwitchCasePattern(kind *ast.SwitchCaseKind, typeRef *ast.TypeExpr, pattern *ast.Expr) {
	if p.curIs(token.IS) {
		p.nextToken()
		*kind = ast.SwitchCaseIsType
		*typeRef = p.parseTypeHintExpr()
	} else if p.curIs(token.BLANK) {
		p.nextToken()
		*kind = ast.SwitchCaseDefault
	} else {
		*kind = ast.SwitchCaseValue
		*pattern = p.parsePrattExpr(LOWEST)
	}
}

// parseSwitchCaseBlock parses a block of statements that terminates at the next `case` keyword or closing `}`.
func (p *Parser) parseSwitchCaseBlock(pos StatementPosition) ast.Block {
	block := make([]ast.Statement, 0)

	blockPos := IN_FUNC
	if pos == IN_FOR {
		blockPos = IN_FOR
	}

	for !p.curIs(token.RBRACE, token.CASE, token.EOF) {
		stmt, decls := p.parseAnnotatedStatementDeclaration(blockPos)
		if len(decls) > 0 {
			p.errStatementMisplaced(blockPos)
		}
		if stmt == nil {
			if !p.curIs(token.RBRACE, token.CASE, token.EOF) {
				p.nextToken()
			}
			continue
		}
		block = append(block, stmt)
	}
	return block
}
