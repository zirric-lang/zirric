package parser

import (
	"errors"
	"strconv"

	"code.knabel.dev/zirric-lang/zirric/ast"
	"code.knabel.dev/zirric-lang/zirric/token"
)

type (
	prefixParser func() ast.Expr
	infixParser  func(ast.Expr) ast.Expr

	Precedence    int
	Associativity int
)

const (
	_ Precedence = iota
	LOWEST
	LOGICAL_OR  // ||
	LOGICAL_AND // &&
	COMPARISON  // == or != or <= or >= or < or >
	COALESCING  // placeholder for ??
	RANGE       // placeholder for ..<
	SUM         // + or -
	PRODUCT     // * or / or %
	BITWISE     // placeholder for << and >>
	PREFIX      // -x or !x
	CALL        // fun(x)
	MEMBER      // . or ?.
)

var precedences = map[token.TokenType]Precedence{
	token.OR:       LOGICAL_OR,
	token.AND:      LOGICAL_AND,
	token.EQ:       COMPARISON,
	token.NEQ:      COMPARISON,
	token.LTE:      COMPARISON,
	token.GTE:      COMPARISON,
	token.LT:       COMPARISON,
	token.GT:       COMPARISON,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.SLASH:    PRODUCT,
	token.PERCENT:  PRODUCT,
	token.ASTERISK: PRODUCT,
	token.LPAREN:   CALL,
	token.LBRACKET: CALL,
	token.DOT:      MEMBER,
}

const (
	A_NONE Associativity = iota
	A_LEFT
	A_RIGHT
)

var associativities = map[Precedence]Associativity{
	LOWEST:      A_NONE,
	LOGICAL_OR:  A_LEFT,
	LOGICAL_AND: A_LEFT,
	COMPARISON:  A_NONE,
	COALESCING:  A_RIGHT,
	RANGE:       A_NONE,
	SUM:         A_LEFT,
	PRODUCT:     A_LEFT,
	BITWISE:     A_NONE,
}

func (p *Parser) peekPrecedence() Precedence {
	if prec, ok := precedences[p.peekToken.Type]; ok {
		return prec
	}
	return LOWEST
}

func (p *Parser) curPrecendence() Precedence {
	if prec, ok := precedences[p.curToken.Type]; ok {
		return prec
	}
	return LOWEST
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParser) {
	p.prefixParsers[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParser) {
	p.infixParsers[tokenType] = fn
}

func (p *Parser) parseExprStmt() *ast.StmtExpr {
	stmtTok := p.curToken
	expr := p.parsePrattExpr(LOWEST)
	return ast.MakeStmtExpr(stmtTok, expr)
}

func (p *Parser) parsePrattExpr(precedence Precedence) ast.Expr {
	prefix := p.prefixParsers[p.curToken.Type]
	if prefix == nil {
		expectedTypes := make([]token.TokenType, 0, len(p.prefixParsers))
		for t := range p.prefixParsers {
			expectedTypes = append(expectedTypes, t)
		}
		p.expect(expectedTypes...)
		return nil
	}
	lhs := prefix()

	for precedence < p.curPrecendence() {
		infix := p.infixParsers[p.curToken.Type]
		if infix == nil {
			return lhs
		}
		lhs = infix(lhs)
	}
	return lhs
}

func (p *Parser) parsePrattExprIdentifier() ast.Expr {
	tok, _ := p.expect(token.IDENT)
	return ast.MakeExprIdentifier(ast.MakeIdentifier(tok))
}

func (p *Parser) parsePrattExprTrue() ast.Expr {
	tok, _ := p.expect(token.TRUE)
	return ast.MakeExprBool(true, tok)
}

func (p *Parser) parsePrattExprFalse() ast.Expr {
	tok, _ := p.expect(token.FALSE)
	return ast.MakeExprBool(false, tok)
}

func (p *Parser) parsePrattExprNull() ast.Expr {
	tok, _ := p.expect(token.NULL)
	return ast.MakeExprNull(tok)
}

func (p *Parser) parsePrattExprInt() ast.Expr {
	tok, _ := p.expect(token.INT)
	int, err := strconv.ParseInt(tok.Literal, 0, 64)
	if err != nil {
		p.errUnderlyingErrorf(err, "invalid int literal %q", tok.Literal)
	}
	return ast.MakeExprInt(int, tok)
}

func (p *Parser) parsePrattExprFloat() ast.Expr {
	tok, _ := p.expect(token.FLOAT)
	float, err := strconv.ParseFloat(tok.Literal, 64)
	if err != nil {
		p.errUnderlyingErrorf(err, "invalid float literal %q", tok.Literal)
	}
	return ast.MakeExprFloat(float, tok)
}

func (p *Parser) parsePrattExprChar() ast.Expr {
	tok, _ := p.expect(token.CHAR)
	ch, err := parseCharLiteral(tok.Literal)
	if err != nil {
		p.errUnderlyingErrorf(err, "invalid char literal %q", tok.Literal)
	}
	return ast.MakeExprChar(ch, tok)
}

func (p *Parser) parsePrattExprPrefix() ast.Expr {
	tok := p.nextToken()
	op := ast.OperatorUnary(tok)
	expr := p.parsePrattExpr(PREFIX)
	return ast.MakeExprOperatorUnary(op, expr)
}

func (p *Parser) parsePrattExprInfix(lhs ast.Expr) ast.Expr {
	prec := p.curPrecendence()
	asso := associativities[p.curPrecendence()]
	if asso == A_RIGHT {
		prec -= 1
	}
	tok := p.nextToken()
	op := ast.OperatorBinary(tok)
	rhs := p.parsePrattExpr(prec)
	return ast.MakeExprOperatorBinary(op, lhs, rhs)
}

func (p *Parser) parsePrattExprGroup() ast.Expr {
	p.expect(token.LPAREN)
	expr := p.parsePrattExpr(LOWEST)

	_, ok := p.expect(token.RPAREN)
	if !ok {
		return nil
	}
	return expr
}

func (p *Parser) parsePrattExprIfElse() ast.Expr {
	ifTok := p.nextToken()

	condition := p.parsePrattExpr(LOWEST)
	if condition == nil {
		return nil
	}

	_, ok := p.expect(token.LBRACE)
	if !ok {
		return nil
	}
	then := p.parsePrattExpr(LOWEST)
	if then == nil {
		return nil
	}

	_, ok = p.expect(token.RBRACE)
	if !ok {
		return nil
	}

	elseTok, ok := p.expect(token.ELSE)
	if !ok {
		return nil
	}

	ifExpr := ast.MakeExprIf(ifTok, condition, then)

	for p.curIs(token.IF) {
		p.nextToken()
		elseCond := p.parsePrattExpr(LOWEST)
		p.expect(token.LBRACE)
		elseExpr := p.parsePrattExpr(LOWEST)
		p.expect(token.RBRACE)
		elif := ast.MakeExprElseIf(elseTok, elseCond, elseExpr)
		ifExpr.AddElseIf(elif)
		p.expect(token.ELSE)
	}

	_, ok = p.expect(token.LBRACE)
	if !ok {
		return nil
	}
	els := p.parsePrattExpr(LOWEST)
	if els == nil {
		return nil
	}

	_, ok = p.expect(token.RBRACE)
	if !ok {
		return nil
	}

	ifExpr.SetElse(els)
	return ifExpr
}

func (p *Parser) parsePrattExprFor() ast.Expr {
	forTok, _ := p.expect(token.FOR)
	bodySymbols := ast.MakeSymbolTable(p.curSymbolTable, ast.MakeIdentifier(forTok))

	if p.curIs(token.LBRACE) {
		p.expect(token.LBRACE)
		block := p.parseExprForBlock(bodySymbols)
		p.expect(token.RBRACE)
		return ast.MakeExprFor(forTok, nil, nil, nil, block)
	}

	if p.curIs(token.IDENT) && p.peekIs(token.LEFT_ARROW) {
		identTok, _ := p.expect(token.IDENT)
		ident := ast.MakeIdentifier(identTok)
		bodySymbols.Insert(ast.MakeDeclForBinding(identTok, ident))
		p.expect(token.LEFT_ARROW)
		collectionExpr := p.parseExpr()
		p.expect(token.LBRACE)
		block := p.parseExprForBlock(bodySymbols)
		p.expect(token.RBRACE)
		return ast.MakeExprFor(forTok, nil, &ident, collectionExpr, block)
	}

	cond := p.parseExpr()
	p.expect(token.LBRACE)
	block := p.parseExprForBlock(bodySymbols)
	p.expect(token.RBRACE)
	return ast.MakeExprFor(forTok, cond, nil, nil, block)
}

func (p *Parser) parsePrattExprFunc() ast.Expr {
	return p.parseExprFunction()
}

func (p *Parser) parsePrattExprCall(fn ast.Expr) ast.Expr {
	fnExpr := ast.MakeExprInvocation(fn)
	p.nextToken()

	if p.curIs(token.RPAREN) {
		p.nextToken()
		return fnExpr
	}
	fnExpr.AddArgument(p.parsePrattExpr(LOWEST))

	for p.curIs(token.COMMA) {
		p.nextToken()
		fnExpr.AddArgument(p.parsePrattExpr(LOWEST))
	}

	_, ok := p.expect(token.RPAREN)
	if !ok {
		return nil
	}

	return fnExpr
}

func (p *Parser) parsePrattExprMember(owner ast.Expr) ast.Expr {
	dotTok := p.nextToken()
	identTok, ok := p.expect(token.IDENT)
	if !ok {
		return nil
	}
	return ast.MakeExprMemberAccess(dotTok, owner, ast.MakeIdentifier(identTok))
}

func (p *Parser) parsePrattExprIndex(owner ast.Expr) ast.Expr {
	indexTok := p.nextToken()
	indexExpr := p.parsePrattExpr(LOWEST)
	_, ok := p.expect(token.RBRACKET)
	if !ok {
		return nil
	}
	return ast.MakeExprIndexAccess(indexTok, owner, indexExpr)
}

func (p *Parser) parsePrattExprString() ast.Expr {
	tok := p.nextToken()
	return ast.MakeExprString(tok, tok.Literal)
}

func parseCharLiteral(literal string) (rune, error) {
	if literal == "" {
		return 0, errors.New("char literal must contain exactly one character")
	}
	ch, _, tail, err := strconv.UnquoteChar(literal, '\'')
	if err != nil {
		return 0, err
	}
	if tail != "" {
		return 0, errors.New("char literal must contain exactly one character")
	}
	return ch, nil
}

func (p *Parser) parseExprListOrDict() ast.Expr {
	tok := p.nextToken()

	if p.curIs(token.RBRACKET) {
		p.nextToken()
		return ast.MakeExprArray(nil, tok)
	}
	if p.curIs(token.COLON) {
		p.nextToken()
		p.expect(token.RBRACKET)
		return ast.MakeExprDict(nil, tok)
	}

	initialExpr := p.parsePrattExpr(LOWEST)

	if p.curIs(token.RBRACKET) {
		p.nextToken()
		return ast.MakeExprArray([]ast.Expr{initialExpr}, tok)
	}

	if p.curIs(token.COMMA) {
		rest := p.parsePrattExprArrayElements()
		if rest == nil {
			return nil
		}
		elements := append([]ast.Expr{initialExpr}, rest...)
		_, ok := p.expect(token.RBRACKET)
		if !ok {
			return nil
		}
		return ast.MakeExprArray(elements, tok)
	}

	_, ok := p.expect(token.COLON)
	if !ok {
		return nil
	}
	valueExpr := p.parsePrattExpr(LOWEST)
	initialEntry := ast.MakeExprDictEntry(initialExpr, valueExpr)
	entries := []ast.ExprDictEntry{initialEntry}

	if p.curIs(token.RBRACKET) {
		p.nextToken()
		return ast.MakeExprDict(entries, tok)
	}
	rest := p.parsePrattExprDictEntries()
	if rest == nil {
		return nil
	}
	p.expect(token.RBRACKET)
	entries = append(entries, rest...)
	return ast.MakeExprDict(entries, tok)
}

func (p *Parser) parsePrattExprArrayElements() []ast.Expr {
	var elements []ast.Expr
	for p.curIs(token.COMMA) {
		p.nextToken()

		expr := p.parsePrattExpr(LOWEST)
		if expr == nil {
			return nil
		}
		elements = append(elements, expr)
	}
	return elements
}

func (p *Parser) parsePrattExprDictEntries() []ast.ExprDictEntry {
	var elements []ast.ExprDictEntry
	for p.curIs(token.COMMA) {
		p.nextToken()

		key := p.parsePrattExpr(LOWEST)
		if key == nil {
			return nil
		}
		_, ok := p.expect(token.COLON)
		if !ok {
			return nil
		}
		value := p.parsePrattExpr(LOWEST)
		if value == nil {
			return nil
		}
		elements = append(elements, ast.MakeExprDictEntry(key, value))
	}
	return elements
}
