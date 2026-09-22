package parser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
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
	COALESCING  // ?? or !!
	RANGE       // placeholder for ..<
	SUM         // + or -
	PRODUCT     // * or / or %
	BITWISE     // placeholder for << and >>
	PREFIX      // -x or !x
	CALL        // fun(x)
	MEMBER      // . or ?. or !.
)

var precedences = map[token.TokenType]Precedence{
	token.OR:                LOGICAL_OR,
	token.AND:               LOGICAL_AND,
	token.EQ:                COMPARISON,
	token.NEQ:               COMPARISON,
	token.LTE:               COMPARISON,
	token.GTE:               COMPARISON,
	token.LT:                COMPARISON,
	token.GT:                COMPARISON,
	token.IS:                COMPARISON,
	token.PLUS:              SUM,
	token.MINUS:             SUM,
	token.SLASH:             PRODUCT,
	token.PERCENT:           PRODUCT,
	token.ASTERISK:          PRODUCT,
	token.QUESTION_QUESTION: COALESCING,
	token.BANG_BANG:         COALESCING,
	token.LPAREN:            CALL,
	token.LBRACKET:          CALL,
	token.DOT:               MEMBER,
	token.QUESTION_DOT:      MEMBER,
	token.BANG_DOT:          MEMBER,
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

func (p *Parser) parseExprStmt() ast.Statement {
	stmtTok := p.curToken
	expr := p.parsePrattExpr(LOWEST)
	if expr == nil {
		return nil
	}

	if augOp, assignTok, isAssign := p.tryConsumeAssignOp(); isAssign {
		return p.parseAssignStmt(assignTok, expr, augOp)
	}

	return ast.MakeStmtExpr(stmtTok, expr)
}

// tryConsumeAssignOp checks whether the current token is an assignment operator.
// If so, it advances past it and returns the arithmetic op (empty for plain =), the token, and true.
func (p *Parser) tryConsumeAssignOp() (token.TokenType, token.Token, bool) {
	switch p.curToken.Type {
	case token.ASSIGN:
		tok := p.nextToken()
		return "", tok, true
	case token.PLUS_ASSIGN:
		tok := p.nextToken()
		return token.PLUS, tok, true
	case token.MINUS_ASSIGN:
		tok := p.nextToken()
		return token.MINUS, tok, true
	case token.STAR_ASSIGN:
		tok := p.nextToken()
		return token.ASTERISK, tok, true
	case token.SLASH_ASSIGN:
		tok := p.nextToken()
		return token.SLASH, tok, true
	case token.PERCENT_ASSIGN:
		tok := p.nextToken()
		return token.PERCENT, tok, true
	}
	return "", token.Token{}, false
}

func (p *Parser) parseAssignStmt(assignTok token.Token, target ast.Expr, augOp token.TokenType) *ast.StmtAssign {
	if !isValidLValue(target) {
		p.detectError(ParseError{
			Token:   assignTok,
			Summary: "invalid assignment target",
			Details: invalidLValueDetails(target),
		})
		return nil
	}
	value := p.parsePrattExpr(LOWEST)
	return ast.MakeStmtAssign(assignTok, target, augOp, value)
}

// invalidLValueDetails says why this particular target cannot be assigned to, since a guarded access looks like an ordinary member access but is not one.
func invalidLValueDetails(target ast.Expr) string {
	if guard, ok := guardedAccessIn(target); ok {
		return fmt.Sprintf("%s reads a field but may short-circuit instead, so there is not always a place to assign to", guard.Operator())
	}
	return "left-hand side of assignment must be an identifier, member access, or index expression"
}

func isValidLValue(expr ast.Expr) bool {
	if _, guarded := guardedAccessIn(expr); guarded {
		return false
	}
	switch expr.(type) {
	case *ast.ExprIdentifier:
		return true
	case *ast.ExprMemberAccess:
		return true
	case *ast.ExprIndexAccess:
		return true
	}
	return false
}

// guardedAccessIn finds a `?.` or `!.` anywhere along a postfix chain. Anywhere is what matters for an assignment target: `user?.value.name` ends in a plain dot, yet the place it would write to only exists when the guard lets the chain through.
func guardedAccessIn(expr ast.Expr) (ast.MemberAccess, bool) {
	for {
		switch node := expr.(type) {
		case *ast.ExprMemberAccess:
			if access := node.Access(); access != ast.MemberAccessPlain {
				return access, true
			}
			expr = node.Target
		case *ast.ExprIndexAccess:
			expr = node.Target
		default:
			return ast.MemberAccessPlain, false
		}
	}
}

func (p *Parser) parsePrattExpr(precedence Precedence) ast.Expr {
	prefix := p.prefixParsers[p.curToken.Type]
	if prefix == nil {
		p.errExpected("an expression")
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

func (p *Parser) parsePrattExprVoid() ast.Expr {
	tok, _ := p.expect(token.VOID)
	return ast.MakeExprVoid(tok)
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
	open, _ := p.expect(token.LPAREN)
	expr := p.parsePrattExpr(LOWEST)

	_, ok := p.expectClosing(open, token.RPAREN)
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
	bodySymbols := p.curSymbolTable.MakeChild(ast.MakeIdentifier(forTok))

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

// parsePrattExprFnClosure parses the new closure syntax: fn(params) { body }
// or fn(params) -> ReturnType { body }.
func (p *Parser) parsePrattExprFnClosure() ast.Expr {
	fnTok, _ := p.expect(token.FUNCTION)
	p.expect(token.LPAREN)

	var fun *ast.ExprFunc
	fun, p.curSymbolTable = ast.MakeExprFunc(fnTok, p.curSymbolTable.NextAnonymousFunctionName(), p.curSymbolTable)

	params := p.parseDeclParameterListWithInsert(false)
	fun.SetParams(params)
	p.expect(token.RPAREN)

	if p.curIs(token.RIGHT_ARROW) {
		p.expect(token.RIGHT_ARROW)
		fun.ReturnType = p.parseTypeHintExpr()
	}

	p.expect(token.LBRACE)

	stmts := p.parseStmtBlock(IN_FUNC)
	if len(stmts) == 1 {
		// If the body is a single expression statement, convert it to a return statement implicitly.
		exprStmt, ok := stmts[0].(*ast.StmtExpr)
		if ok {
			returnStmt := ast.MakeStmtReturn(exprStmt.Token, exprStmt.Expr)
			stmts[0] = returnStmt
		}
	}

	fun.SetImplBlock(stmts)

	p.expect(token.RBRACE)
	p.popSymbolTable()
	return fun
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
		if p.curIs(token.RPAREN) {
			break
		}
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
	// `type` is accepted for the same reason a field may be declared under it: after a dot there is nothing else the word could introduce, so a field named `type` can be read as well as written.
	identTok, ok := p.expect(token.IDENT, token.TYPE)
	if !ok {
		return nil
	}
	return ast.MakeExprMemberAccess(dotTok, owner, ast.MakeIdentifier(identTok))
}

func (p *Parser) parsePrattExprIndex(owner ast.Expr) ast.Expr {
	indexTok := p.nextToken()
	indexExpr := p.parsePrattExpr(LOWEST)
	_, ok := p.expectClosing(indexTok, token.RBRACKET)
	if !ok {
		return nil
	}
	return ast.MakeExprIndexAccess(indexTok, owner, indexExpr)
}

func (p *Parser) parsePrattExprString() ast.Expr {
	tok := p.nextToken()
	str, err := parseStringLiteral(tok.Literal)
	if err != nil {
		p.errUnderlyingErrorf(err, "invalid string literal %q", tok.Literal)
	}
	return ast.MakeExprString(tok, str)
}

// parseStringLiteral decodes the escapes in a string literal's raw source. It
// shares strconv.UnquoteChar with parseCharLiteral, so a string accepts the
// escapes a char does — \n, \t, \\ and also \a, \b, \f, \r, \v, \xNN, \uNNNN,
// \UNNNNNNNN and a three-digit octal — differing only in the delimiter each
// escapes, \" here and \' there. Everything that is not an escape is copied
// verbatim, leaving multi-byte characters and raw newlines untouched.
func parseStringLiteral(literal string) (string, error) {
	if !strings.Contains(literal, "\\") {
		return literal, nil
	}
	var out strings.Builder
	for literal != "" {
		if literal[0] != '\\' {
			next := strings.IndexByte(literal, '\\')
			if next == -1 {
				out.WriteString(literal)
				break
			}
			out.WriteString(literal[:next])
			literal = literal[next:]
			continue
		}
		ch, multibyte, tail, err := strconv.UnquoteChar(literal, '"')
		if err != nil {
			return "", err
		}
		// A \xNN escape names a single byte, not the code point of that value.
		if ch < utf8.RuneSelf || !multibyte {
			out.WriteByte(byte(ch))
		} else {
			out.WriteRune(ch)
		}
		literal = tail
	}
	return out.String(), nil
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
	// Non-nil so a solitary trailing comma (e.g. `[a,]`, no further
	// elements) isn't mistaken by the caller for a parse error — nil is
	// reserved for actual failures below.
	elements := []ast.Expr{}
	for p.curIs(token.COMMA) {
		p.nextToken()
		if p.curIs(token.RBRACKET) {
			break
		}

		expr := p.parsePrattExpr(LOWEST)
		if expr == nil {
			return nil
		}
		elements = append(elements, expr)
	}
	return elements
}

func (p *Parser) parsePrattExprDictEntries() []ast.ExprDictEntry {
	// Non-nil for the same reason as parsePrattExprArrayElements: a
	// solitary trailing comma must not look like a parse error to the caller.
	elements := []ast.ExprDictEntry{}
	for p.curIs(token.COMMA) {
		p.nextToken()
		if p.curIs(token.RBRACKET) {
			break
		}

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
