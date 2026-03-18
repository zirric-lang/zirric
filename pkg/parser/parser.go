package parser

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

type Parser struct {
	srcFile *ast.SourceFile
	lex     *lexer.Lexer
	errors  []ParseError

	curToken  token.Token
	peekToken token.Token

	curSymbolTable *ast.DeclTable

	prefixParsers map[token.TokenType]prefixParser
	infixParsers  map[token.TokenType]infixParser
}

func NewSourceParser(lex *lexer.Lexer, parent *ast.DeclTable, path string) *Parser {
	p := &Parser{lex: lex}
	p.nextToken()
	p.nextToken()

	p.srcFile = ast.MakeSourceFile(parent, path, p.curToken)

	p.prefixParsers = make(map[token.TokenType]prefixParser)
	p.registerPrefix(token.IDENT, p.parsePrattExprIdentifier)
	p.registerPrefix(token.TRUE, p.parsePrattExprTrue)
	p.registerPrefix(token.FALSE, p.parsePrattExprFalse)
	p.registerPrefix(token.VOID, p.parsePrattExprVoid)
	p.registerPrefix(token.INT, p.parsePrattExprInt)
	p.registerPrefix(token.FLOAT, p.parsePrattExprFloat)
	p.registerPrefix(token.BANG, p.parsePrattExprPrefix)
	p.registerPrefix(token.MINUS, p.parsePrattExprPrefix)
	p.registerPrefix(token.PLUS, p.parsePrattExprPrefix)
	p.registerPrefix(token.LPAREN, p.parsePrattExprGroup)
	p.registerPrefix(token.IF, p.parsePrattExprIfElse)
	p.registerPrefix(token.FUNCTION, p.parsePrattExprFnClosure)
	p.registerPrefix(token.LBRACKET, p.parseExprListOrDict)
	p.registerPrefix(token.STRING, p.parsePrattExprString)
	p.registerPrefix(token.CHAR, p.parsePrattExprChar)
	p.registerPrefix(token.FOR, p.parsePrattExprFor)
	p.registerPrefix(token.SWITCH, p.parsePrattExprSwitch)

	p.infixParsers = make(map[token.TokenType]infixParser)
	p.registerInfix(token.OR, p.parsePrattExprInfix)
	p.registerInfix(token.AND, p.parsePrattExprInfix)
	p.registerInfix(token.EQ, p.parsePrattExprInfix)
	p.registerInfix(token.NEQ, p.parsePrattExprInfix)
	p.registerInfix(token.LTE, p.parsePrattExprInfix)
	p.registerInfix(token.GTE, p.parsePrattExprInfix)
	p.registerInfix(token.LT, p.parsePrattExprInfix)
	p.registerInfix(token.GT, p.parsePrattExprInfix)
	p.registerInfix(token.PLUS, p.parsePrattExprInfix)
	p.registerInfix(token.MINUS, p.parsePrattExprInfix)
	p.registerInfix(token.SLASH, p.parsePrattExprInfix)
	p.registerInfix(token.ASTERISK, p.parsePrattExprInfix)
	p.registerInfix(token.PERCENT, p.parsePrattExprInfix)
	p.registerInfix(token.LPAREN, p.parsePrattExprCall)
	p.registerInfix(token.DOT, p.parsePrattExprMember)
	p.registerInfix(token.LBRACKET, p.parsePrattExprIndex)
	p.registerInfix(token.IS, p.parsePrattExprIs)

	return p
}

func (p *Parser) Errors() []ParseError {
	return p.errors
}

func (p *Parser) SymbolErrors() []ParseError {
	return nil
}

func (p *Parser) ParseSourceFile() *ast.SourceFile {
	p.curSymbolTable = p.srcFile.Decls

	inPosition := IN_INITIAL
	for p.curToken.Type != token.EOF {
		stmt, childDecls := p.parseStatementInContext(inPosition, nil)
		inPosition = IN_GLOBAL
		if stmt != nil {
			p.srcFile.Add(stmt)
			for _, d := range childDecls {
				p.srcFile.Add(d)
			}
		} else {
			p.nextToken()
		}
	}

	return p.srcFile
}

func (p *Parser) nextToken() token.Token {
	cur := p.curToken
	p.curToken = p.peekToken
	p.peekToken = p.lex.NextToken()
	return cur
}

func (p *Parser) peekIs(tokTypes ...token.TokenType) bool {
	for _, tok := range tokTypes {
		if p.peekToken.Type == tok {
			return true
		}
	}
	return false
}

func (p *Parser) curIs(tokTypes ...token.TokenType) bool {
	for _, tok := range tokTypes {
		if p.curToken.Type == tok {
			return true
		}
	}
	return false
}

func (p *Parser) expect(tokTypes ...token.TokenType) (token.Token, bool) {
	if !p.curIs(tokTypes...) {
		p.errUnexpectedToken(tokTypes...)
		return p.errorToken(), false
	}
	cur := p.curToken
	p.nextToken()
	return cur, true
}

func (p *Parser) errorToken() token.Token {
	return token.Token{
		Type:    token.ILLEGAL,
		Literal: "ERROR",
		Source:  p.curToken.Source,
		Leading: p.curToken.Leading,
	}
}

func (p *Parser) detectError(err ParseError) {
	p.errors = append(p.errors, err)
}

func (p *Parser) popSymbolTable() *ast.DeclTable {
	old := p.curSymbolTable
	p.curSymbolTable = old.Parent
	return old
}

func (p *Parser) parseAnnotatedStatementDeclaration(pos StatementPosition) (ast.Statement, []ast.StatementDeclaration) {
	annos := p.parseAttributeChain()
	return p.parseStatementInContext(pos, annos)
}

// parseUnionDecl parsed union declarations in various forms:
//
//		union <identifier> // empty union
//		union <identifier> { } // empty union
//		union <identifier> {
//		  <identifier> // referencing: no attributes allowed!
//		  <fully-qualified-identifier> // global reference
//		  <optional:attributes> <data_decl>
//		  <optional:attributes> <union_decl>
//	 	}
func (p *Parser) parseUnionDecl(pos StatementPosition, annos ast.AttributeChain) (*ast.DeclUnion, []ast.StatementDeclaration) {
	unionToken, _ := p.expect(token.UNION)
	identToken, _ := p.expect(token.IDENT)
	ident := ast.MakeIdentifier(identToken)
	union := ast.MakeDeclUnion(unionToken, ident)
	union.Attributes = annos

	if !p.curIs(token.LBRACE) {
		return union, nil
	}

	p.expect(token.LBRACE)

	var childDecls []ast.StatementDeclaration
	for !p.curIs(token.RBRACE) {
		unionMember, children := p.parseUnionDeclMember(pos)
		childDecls = append(childDecls, children...)
		union.AddMember(unionMember)
	}
	p.expect(token.RBRACE)

	return union, childDecls
}

// parseUnionDeclMember parses union members in these forms:
//
//	<identifier> // referencing: no attributes allowed!
//	<fully-qualified-identifier> // global reference
//	<optional:attributes> <data_decl>
//	<optional:attributes> <union_decl>
func (p *Parser) parseUnionDeclMember(pos StatementPosition) (*ast.DeclUnionMember, []ast.StatementDeclaration) {
	if p.curToken.Type == token.IDENT {
		ref := p.parseStaticIdentifierReference()
		return ast.MakeDeclUnionMember(ref.TokenLiteral(), ref), nil
	}
	attributes := p.parseAttributeChain()
	switch p.curToken.Type {
	case token.DATA:
		dataDecl := p.parseDataDecl(pos, attributes)
		return ast.MakeDeclUnionMember(dataDecl.Token, ast.StaticReference{dataDecl.DeclName()}), []ast.StatementDeclaration{dataDecl}
	case token.UNION:
		unionDecl, childDecls := p.parseUnionDecl(pos, attributes)
		return ast.MakeDeclUnionMember(unionDecl.Token, ast.StaticReference{unionDecl.DeclName()}), append(childDecls, unionDecl)
	default:
		p.errUnexpectedToken(token.DATA, token.UNION)
		return nil, nil
	}
}

func (p *Parser) parseStaticIdentifierReference() ast.StaticReference {
	var ref ast.StaticReference
	for {
		identTok, ok := p.expect(token.IDENT)
		if !ok {
			break
		}
		id := ast.MakeIdentifier(identTok)
		ref = append(ref, id)

		if !p.curIs(token.DOT) {
			break
		}
		p.expect(token.DOT)
	}
	if len(ref) == 0 {
		return ast.StaticReference{ast.MakeIdentifier(p.errorToken())}
	}
	return ref
}

// parseDataDecl parses data declarations in various forms:
//
//	data <identifier> // name
//	data <identifier> { }
//	data <identifier> {
//	  <optional:attributes> <identifer> // property name
//	  <optional:attributes> <identifier>(<param_list>) // function member
//	  <optional:attributes> <identifier> = <expr> // defaulted member
//
//	  // optional
//	  <optional:attributes> <func_decl>
//	  <optional:attributes> <var_decl>
//	}
func (p *Parser) parseDataDecl(_ StatementPosition, annos ast.AttributeChain) *ast.DeclData {
	declToken, _ := p.expect(token.DATA)
	identToken, _ := p.expect(token.IDENT)
	ident := ast.MakeIdentifier(identToken)
	data := ast.MakeDeclData(declToken, ident)
	data.Attributes = annos

	sym := p.curSymbolTable.Insert(data)

	sym.ChildTable = p.curSymbolTable.MakeChild(data)
	p.curSymbolTable = sym.ChildTable
	defer func() { p.popSymbolTable() }()

	if !p.curIs(token.LBRACE) {
		return data
	}
	p.expect(token.LBRACE)
	fields := p.parsePropertyDeclarationList()
	for _, f := range fields {
		data.AddField(f)
	}
	p.expect(token.RBRACE)
	return data
}

// parseDataDeclField parses a single data declaration field.
//
//	simple
//	simple: Type
//	method()
//	method() -> ReturnType
//	@Attribute() field
//	@Attribute() field: Type
//	@Attribute() method()
func (p *Parser) parseDataDeclField() *ast.DeclField {
	attributes := p.parseAttributeChain()
	identTok, _ := p.expect(token.IDENT, token.TYPE)
	name := ast.MakeIdentifier(identTok)

	if p.curIs(token.LPAREN) {
		p.expect(token.LPAREN)
		params := p.parseDeclParameterListWithInsert(false)
		p.expect(token.RPAREN)
		var returnType ast.TypeExpr
		if p.curIs(token.RIGHT_ARROW) {
			p.expect(token.RIGHT_ARROW)
			returnType = p.parseTypeHintExpr()
		}
		return ast.MakeDeclField(name, params, attributes, returnType)
	}

	var typeHint ast.TypeExpr
	if p.curIs(token.COLON) {
		p.expect(token.COLON)
		typeHint = p.parseTypeHintExpr()
	}
	return ast.MakeDeclField(name, nil, attributes, typeHint)
}

// parseAttrDecl parses the declaration of an attribute type.
//
//	attribute <identifier>
//	attribute <identifier> {
//	  // properties
//	}
func (p *Parser) parseAttrDecl(_ StatementPosition, annos ast.AttributeChain) *ast.DeclAttr {
	declToken, _ := p.expect(token.ATTRIBUTE)
	identToken, _ := p.expect(token.IDENT)
	ident := ast.MakeIdentifier(identToken)
	declAnno := ast.MakeDeclAttr(declToken, ident)
	declAnno.Attributes = annos

	sym := p.curSymbolTable.Insert(declAnno)
	sym.ChildTable = p.curSymbolTable.MakeChild(declAnno)
	p.curSymbolTable = sym.ChildTable
	defer func() { p.popSymbolTable() }()

	if !p.curIs(token.LBRACE) {
		return declAnno
	}
	p.expect(token.LBRACE)
	fields := p.parsePropertyDeclarationList()
	for _, f := range fields {
		declAnno.AddField(f)
	}
	p.expect(token.RBRACE)
	return declAnno
}

// parseModuleDecl parses a declared module
//
//	module <identifier>
func (p *Parser) parseModuleDecl(pos StatementPosition, annos ast.AttributeChain) *ast.DeclModule {
	if pos != IN_INITIAL {
		p.errStatementMisplaced(pos)
	}
	modToken, _ := p.expect(token.MODULE)
	nameTok, _ := p.expect(token.IDENT)
	name := ast.MakeIdentifier(nameTok)

	mod := ast.MakeDeclModule(modToken, name)
	mod.Attributes = annos
	return mod
}

// parseExternDecl parses three possible types:
// 1. an external type: extern type <identifier> [{ fields }]
// 2. an external function: extern fn <identifier>([params])
// 3. an external value: extern const <identifier>
func (p *Parser) parseExternDecl(pos StatementPosition, annos ast.AttributeChain) ast.StatementDeclaration {
	if pos != IN_INITIAL && pos != IN_GLOBAL {
		p.errStatementMisplaced(pos)
	}
	externTok, _ := p.expect(token.EXTERN)

	// Expect one of: type, fn, const
	if p.curIs(token.TYPE) {
		return p.parseExternTypeDecl(externTok, annos)
	} else if p.curIs(token.FUNCTION) {
		return p.parseExternFuncDecl(externTok, annos)
	} else if p.curIs(token.CONST) {
		return p.parseExternValueDecl(externTok, annos)
	} else {
		p.detectError(ParseError{
			Token:   p.curToken,
			Summary: "expected 'type', 'fn', or 'const' after 'extern'",
			Details: fmt.Sprintf("got %q", p.curToken.Literal),
		})
		return nil
	}
}

// parseExternTypeDecl parses extern type declarations
func (p *Parser) parseExternTypeDecl(externTok token.Token, annos ast.AttributeChain) *ast.DeclExternType {
	p.expect(token.TYPE)
	nameTok, _ := p.expect(token.IDENT)
	nameIdent := ast.MakeIdentifier(nameTok)

	extern := ast.MakeDeclExternType(externTok, nameIdent)
	extern.Attributes = annos
	sym := p.curSymbolTable.Insert(extern)
	sym.ChildTable = p.curSymbolTable.MakeChild(extern)

	if p.curIs(token.LBRACE) {
		p.expect(token.LBRACE)
		p.curSymbolTable = sym.ChildTable

		fields := p.parsePropertyDeclarationList()
		for _, f := range fields {
			extern.AddField(f)
		}
		p.expect(token.RBRACE)
		p.popSymbolTable()
	}

	return extern
}

// parseExternFuncDecl parses extern fn declarations
func (p *Parser) parseExternFuncDecl(externTok token.Token, annos ast.AttributeChain) *ast.DeclExternFunc {
	p.expect(token.FUNCTION)
	nameTok, _ := p.expect(token.IDENT)
	nameIdent := ast.MakeIdentifier(nameTok)

	extern := ast.MakeDeclExternFunc(externTok, nameIdent)
	extern.Attributes = annos
	sym := p.curSymbolTable.Insert(extern)
	sym.ChildTable = p.curSymbolTable.MakeChild(extern)
	p.curSymbolTable = sym.ChildTable

	p.expect(token.LPAREN)
	if !p.curIs(token.RPAREN) {
		params := p.parseDeclParameterListWithInsert(false)
		extern.SetParams(params)
	}
	p.expect(token.RPAREN)

	p.popSymbolTable()
	return extern
}

// parseExternValueDecl parses extern const declarations
func (p *Parser) parseExternValueDecl(externTok token.Token, annos ast.AttributeChain) *ast.DeclExternValue {
	p.expect(token.CONST)
	nameTok, _ := p.expect(token.IDENT, token.TRUE, token.FALSE, token.VOID)
	nameIdent := ast.MakeIdentifier(nameTok)

	extern := ast.MakeDeclExternValue(externTok, nameIdent)
	extern.Attributes = annos

	if p.curIs(token.COLON) {
		p.expect(token.COLON)
		extern.TypeHint = p.parseTypeHintExpr()
	}

	p.curSymbolTable.Insert(extern)
	return extern
}

func (p *Parser) parseFunctionDecl(_ StatementPosition, annos ast.AttributeChain) *ast.DeclFunc {
	funcTok, _ := p.expect(token.FUNCTION)
	nameTok, _ := p.expect(token.IDENT)

	var impl *ast.ExprFunc
	var returnType ast.TypeExpr

	impl, p.curSymbolTable = ast.MakeExprFunc(funcTok, nameTok.Literal, p.curSymbolTable)

	p.expect(token.LPAREN)
	params := p.parseDeclParameterListWithInsert(false)
	impl.SetParams(params)
	p.expect(token.RPAREN)

	if p.curIs(token.RIGHT_ARROW) {
		p.expect(token.RIGHT_ARROW)
		returnType = p.parseTypeHintExpr()
		impl.ReturnType = returnType
	}

	fexprTok, _ := p.expect(token.LBRACE)
	block := p.parseStmtBlock(IN_FUNC)
	p.expect(token.RBRACE)

	impl.SetImplBlock(block)
	impl.Token = fexprTok

	p.popSymbolTable()

	decl := ast.MakeDeclFunc(funcTok, ast.MakeIdentifier(nameTok), impl)
	decl.Attributes = annos
	decl.ReturnType = returnType
	sym := p.curSymbolTable.Insert(decl)
	sym.ChildTable = impl.Decls
	return decl
}

func (p *Parser) parseImportDecl(pos StatementPosition, annos ast.AttributeChain) *ast.DeclImport {
	if pos != IN_INITIAL && pos != IN_GLOBAL {
		p.errStatementMisplaced(pos)
	}
	if annos != nil {
		p.errCannotBeAnnotated()
	}
	importTok, _ := p.expect(token.IMPORT)

	var importDecl *ast.DeclImport
	if p.peekIs(token.ASSIGN) {
		aliasTok, _ := p.expect(token.IDENT)
		p.expect(token.ASSIGN)
		moduleName := p.parseStaticIdentifierReference()
		importDecl = ast.MakeDeclAliasImport(importTok, ast.MakeIdentifier(aliasTok), moduleName)
	} else {
		moduleName := p.parseStaticIdentifierReference()
		importDecl = ast.MakeDeclImport(importTok, moduleName)
	}

	if !p.curIs(token.LBRACE) {
		return importDecl
	}
	p.expect(token.LBRACE)
	for !p.curIs(token.RBRACE) {
		memberTok, _ := p.expect(token.IDENT)
		member := ast.MakeDeclImportMember(memberTok, importDecl.ModuleName, ast.MakeIdentifier(memberTok))
		importDecl.AddMember(member)

		if p.curIs(token.COMMA) {
			p.expect(token.COMMA)
		}
	}
	p.expect(token.RBRACE)
	return importDecl
}

func (p *Parser) parseVariableDecl(pos StatementPosition, annos ast.AttributeChain) ast.Statement {
	declTok, _ := p.expect(token.CONST, token.VAR)
	nameTok, _ := p.expect(token.IDENT, token.TRUE, token.FALSE, token.VOID)
	name := ast.MakeIdentifier(nameTok)

	var typeHint ast.TypeExpr
	if p.curIs(token.COLON) {
		p.expect(token.COLON)
		typeHint = p.parseTypeHintExpr()
	}

	p.expect(token.ASSIGN)
	expr := p.parseExpr()

	var decl ast.Decl
	if declTok.Type == token.CONST {
		c := ast.MakeDeclConstant(declTok, name, expr)
		c.IsGlobal = pos < IN_FUNC
		c.Attributes = annos
		c.TypeHint = typeHint
		decl = c
	} else {
		v := ast.MakeDeclVariable(declTok, name, expr)
		v.IsGlobal = pos < IN_FUNC
		v.Attributes = annos
		v.TypeHint = typeHint
		decl = v
	}

	p.curSymbolTable.Insert(decl)
	return decl.(ast.Statement)
}

func (p *Parser) parsePropertyDeclarationList() []ast.DeclField {
	var fields []ast.DeclField
	for {
		if p.curToken.Type == token.RBRACE {
			return fields
		}
		field := p.parseDataDeclField()
		if field != nil {
			fields = append(fields, *field)
		} else {
			p.errors = append(p.errors, ParseError{Token: p.curToken,
				Summary: "invalid data declaration field",
				Details: fmt.Sprintf("unexpected token %q in data declaration fields", p.curToken.Literal),
			})
			return fields
		}
	}
}

func (p *Parser) parseAttributeChain() ast.AttributeChain {
	var attributeChain ast.AttributeChain
	for p.curIs(token.AT) {
		anno := p.parseAttributeInstance()
		attributeChain = append(attributeChain, anno)
	}
	return attributeChain
}

func (p *Parser) parseAttributeInstance() *ast.DeclAttrInstance {
	atTok, _ := p.expect(token.AT)
	ref := p.parseStaticIdentifierReference()

	attr := ast.MakeAttributeInstance(atTok, ref)
	// Attribute resolution is handled by the analyzer.

	p.expect(token.LPAREN)
	args := p.parseExprArgumentList()
	for _, arg := range args {
		attr.AddArgument(arg)
	}
	p.expect(token.RPAREN)
	return attr
}

func (p *Parser) parseDeclParameterListWithInsert(insert bool) []ast.DeclParameter {
	params := make([]ast.DeclParameter, 0)

	for {
		attrs := p.parseAttributeChain()
		if !p.curIs(token.IDENT) {
			// eventual errors will be triggered by parent
			return params
		}
		identTok, _ := p.expect(token.IDENT)
		ident := ast.MakeIdentifier(identTok)

		var typeHint ast.TypeExpr
		if p.curIs(token.COLON) {
			p.expect(token.COLON)
			typeHint = p.parseTypeHintExpr()
		}

		decl := ast.MakeDeclParameter(ident, attrs, typeHint)
		if insert {
			p.curSymbolTable.Insert(decl)
		}

		params = append(params, *decl)

		if !p.curIs(token.COMMA) {
			return params
		}
		p.expect(token.COMMA)
	}
}

// parseTypeHintExpr parses a type expression.
// Supports:
//   - Named types: String, prelude.String
//   - Array types: [ElementType]
//   - Dict types: {KeyType: ValueType}
//   - Function types: fn(params) -> ReturnType
//   - Attribute constraints: @Attr, @A @B @C
func (p *Parser) parseTypeHintExpr() ast.TypeExpr {
	// Attribute constraints: @Attr or @A @B @C
	if p.curIs(token.AT) {
		return p.parseTypeHintAttrs()
	}

	// Array type [T] or Dict type [K: V] — disambiguated after first type expr
	if p.curIs(token.LBRACKET) {
		return p.parseTypeHintArrayOrDict()
	}

	// Function type: fn(params) -> ReturnType
	if p.curIs(token.FUNCTION) {
		return p.parseTypeHintFunc()
	}

	// Named type: Ident or Ident.Ident.Ident...
	ref := p.parseStaticIdentifierReference()
	return ast.MakeTypeExprRef(ref)
}

// parseTypeHintAttrs parses one or more @Attr references as a type expression.
func (p *Parser) parseTypeHintAttrs() ast.TypeExpr {
	atTok := p.curToken
	var attrs []ast.TypeExprRef
	for p.curIs(token.AT) {
		p.expect(token.AT)
		ref := p.parseStaticIdentifierReference()
		attrs = append(attrs, ast.MakeTypeExprRef(ref))
	}
	return ast.MakeTypeExprAttrs(atTok, attrs)
}

// parseTypeHintArrayOrDict parses [ElementType] or [KeyType: ValueType].
// After consuming `[` and the first type expression, `:` indicates a dict type.
func (p *Parser) parseTypeHintArrayOrDict() ast.TypeExpr {
	lbracketTok, _ := p.expect(token.LBRACKET)
	first := p.parseTypeHintExpr()
	if p.curIs(token.COLON) {
		// Dict type: [K: V]
		p.expect(token.COLON)
		value := p.parseTypeHintExpr()
		p.expect(token.RBRACKET)
		return ast.MakeTypeExprDict(lbracketTok, first, value)
	}
	// Array type: [T]
	p.expect(token.RBRACKET)
	return ast.MakeTypeExprArray(lbracketTok, first)
}

// parseTypeHintFunc parses fn(params) -> ReturnType.
func (p *Parser) parseTypeHintFunc() ast.TypeExpr {
	fnTok, _ := p.expect(token.FUNCTION)
	p.expect(token.LPAREN)

	var params []ast.DeclParameter
	if !p.curIs(token.RPAREN) {
		params = p.parseTypeHintFuncParams()
	}
	p.expect(token.RPAREN)

	var returnType ast.TypeExpr
	if p.curIs(token.RIGHT_ARROW) {
		p.expect(token.RIGHT_ARROW)
		returnType = p.parseTypeHintExpr()
	}
	return ast.MakeTypeExprFunc(fnTok, params, returnType)
}

// parseTypeHintFuncParams parses parameter list for fn type expressions.
func (p *Parser) parseTypeHintFuncParams() []ast.DeclParameter {
	var params []ast.DeclParameter
	for {
		identTok, _ := p.expect(token.IDENT)
		ident := ast.MakeIdentifier(identTok)

		var typeHint ast.TypeExpr
		if p.curIs(token.COLON) {
			p.expect(token.COLON)
			typeHint = p.parseTypeHintExpr()
		}

		decl := ast.MakeDeclParameter(ident, nil, typeHint)
		params = append(params, *decl)

		if !p.curIs(token.COMMA) {
			return params
		}
		p.expect(token.COMMA)
	}
}

func (p *Parser) parseStatementReturn(pos StatementPosition) *ast.StmtReturn {
	if pos != IN_FUNC && pos != IN_FOR {
		p.errStatementMisplaced(pos)
	}
	retTok, _ := p.expect(token.RETURN)
	for _, dec := range p.curToken.Leading {
		if dec.Type != token.DECO_INLINE {
			return ast.MakeStmtReturn(retTok, nil)
		}
	}

	if p.curIs(token.RBRACE, token.CASE) {
		return ast.MakeStmtReturn(retTok, nil)
	}

	expr := p.parseExpr()
	return ast.MakeStmtReturn(retTok, expr)
}

func (p *Parser) parseStatementBreak(pos StatementPosition) ast.StmtBreak {
	if pos != IN_FOR {
		p.errStatementMisplaced(pos)
	}
	breakTok, _ := p.expect(token.BREAK)
	return ast.MakeStmtBreak(breakTok)
}

func (p *Parser) parseStatementContinue(pos StatementPosition) ast.StmtContinue {
	if pos != IN_FOR {
		p.errStatementMisplaced(pos)
	}
	continueTok, _ := p.expect(token.CONTINUE)
	return ast.MakeStmtContinue(continueTok)
}

func (p *Parser) parseStatementFor(_ StatementPosition) ast.StmtFor {
	forTok, _ := p.expect(token.FOR)
	if p.curIs(token.LBRACE) {
		p.expect(token.LBRACE)
		block := p.parseStmtBlock(IN_FOR)
		p.expect(token.RBRACE)
		return ast.MakeStmtFor(forTok, nil, nil, nil, block)
	}

	if p.curIs(token.IDENT) && p.peekIs(token.LEFT_ARROW) {
		identTok, _ := p.expect(token.IDENT)
		ident := ast.MakeIdentifier(identTok)
		p.curSymbolTable.Insert(ast.MakeDeclForBinding(identTok, ident))
		p.expect(token.LEFT_ARROW)
		collectionExpr := p.parseExpr()
		p.expect(token.LBRACE)
		block := p.parseStmtBlock(IN_FOR)
		p.expect(token.RBRACE)
		return ast.MakeStmtFor(forTok, nil, &ident, collectionExpr, block)
	}

	cond := p.parseExpr()
	p.expect(token.LBRACE)
	block := p.parseStmtBlock(IN_FOR)
	p.expect(token.RBRACE)
	return ast.MakeStmtFor(forTok, cond, nil, nil, block)
}

func (p *Parser) parseStatementIf(pos StatementPosition) ast.StmtIf {
	ifTok, _ := p.expect(token.IF)
	cond := p.parseExpr()
	p.expect(token.LBRACE)
	ifBlock := p.parseStmtBlock(pos)
	p.expect(token.RBRACE)

	ifStmt := ast.MakeStmtIf(ifTok, cond, ifBlock)

	for p.curIs(token.ELSE) {
		if p.peekIs(token.IF) {
			elseIf := p.parseStatementElseIf(pos)
			ifStmt.AddElseIf(elseIf)
			continue
		}
		p.expect(token.ELSE)
		p.expect(token.LBRACE)
		elseBlock := p.parseStmtBlock(pos)
		p.expect(token.RBRACE)
		ifStmt.SetElse(elseBlock)
		break
	}
	return ifStmt
}

func (p *Parser) parseStatementElseIf(pos StatementPosition) ast.StmtElseIf {
	elseTok, _ := p.expect(token.ELSE)
	p.expect(token.IF)
	cond := p.parseExpr()
	p.expect(token.LBRACE)
	block := p.parseStmtBlock(pos)
	p.expect(token.RBRACE)

	return ast.MakeStmtIfElse(elseTok, cond, block)
}

func (p *Parser) parseExprArgumentList() []ast.Expr {
	var args []ast.Expr
	for !p.curIs(token.RPAREN) {
		args = append(args, p.parseExpr())
		if !p.curIs(token.COMMA) {
			return args
		}
		p.expect(token.COMMA)
	}
	return args
}

func (p *Parser) parseExpr() ast.Expr {
	expr := p.parsePrattExpr(LOWEST)
	return expr
}

func (p *Parser) parseStmtBlock(pos StatementPosition) ast.Block {
	block := make([]ast.Statement, 0)

	blockPos := IN_FUNC
	if pos == IN_FOR {
		blockPos = IN_FOR
	}

	for !p.curIs(token.RBRACE, token.RBRACKET, token.RPAREN) {
		if p.curIs(token.EOF) {
			break
		}
		stmt, decls := p.parseAnnotatedStatementDeclaration(blockPos)
		if len(decls) > 0 {
			p.errStatementMisplaced(blockPos)
		}
		if stmt == nil {
			// No progress was made; advance past the unrecognized token to prevent an infinite loop.
			if !p.curIs(token.RBRACE, token.RBRACKET, token.RPAREN, token.EOF) {
				p.nextToken()
			}
			continue
		}
		block = append(block, stmt)
	}
	return block
}

func (p *Parser) parseExprForBlock(symbols *ast.DeclTable) ast.ExprForBody {
	decls := make([]ast.Decl, 0)
	stmts := make([]ast.Statement, 0)
	seenStmt := false

	prevSymbols := p.curSymbolTable
	p.curSymbolTable = symbols

	for !p.curIs(token.RBRACE, token.RBRACKET, token.RPAREN) {
		if p.curIs(token.EOF) {
			break
		}
		annos := p.parseAttributeChain()
		if p.curIs(token.CONST, token.VAR) {
			if seenStmt {
				p.detectError(ParseError{
					Token:   p.curToken,
					Summary: "expected declarations before statements in expr-for block",
					Details: "expr-for requires variable declarations before statements",
				})
			}
			decl := p.parseVariableDecl(IN_FOR, annos)
			if !seenStmt {
				decls = append(decls, decl.(ast.Decl))
			}
			continue
		}
		if annos != nil {
			p.errCannotBeAnnotated()
		}
		switch p.curToken.Type {
		case token.IF:
			stmts = append(stmts, p.parseStatementIf(IN_FOR))
			seenStmt = true
			continue
		case token.BREAK:
			stmts = append(stmts, p.parseStatementBreak(IN_FOR))
			seenStmt = true
			continue
		case token.CONTINUE:
			stmts = append(stmts, p.parseStatementContinue(IN_FOR))
			seenStmt = true
			continue
		}
		stmtTok := p.curToken
		expr := p.parseExpr()
		if expr == nil {
			// No progress was made; advance past the unrecognized token to prevent an infinite loop.
			if !p.curIs(token.RBRACE, token.RBRACKET, token.RPAREN, token.EOF) {
				p.nextToken()
			}
			continue
		}
		stmts = append(stmts, ast.MakeStmtExpr(stmtTok, expr))
		seenStmt = true
	}
	p.curSymbolTable = prevSymbols
	return ast.ExprForBody{Decls: decls, Stmts: stmts, DeclsTable: symbols}
}
