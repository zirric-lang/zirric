package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
	Source  *Source

	// Stores leading decorative tokens.
	// Trailing decorative tokens belong to the following token.
	// Comments at the end of the file belong to EOF tokens.
	Leading []DecorativeToken
}

const (
	ILLEGAL TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"
	// A virtual token, that does not actually exist and represents directories.
	MODULE_DIRECTORY TokenType = "MODULE_DIRECTORY"

	// Identifiers + literals
	IDENT  TokenType = "IDENT"
	STRING TokenType = "STRING"
	CHAR   TokenType = "CHAR"
	INT    TokenType = "INT"
	FLOAT  TokenType = "FLOAT"

	// Operators
	BANG     TokenType = "!"
	PLUS     TokenType = "+"
	MINUS    TokenType = "-"
	ASTERISK TokenType = "*"
	SLASH    TokenType = "/"
	PERCENT  TokenType = "%"

	// Augmented assignment operators
	PLUS_ASSIGN    TokenType = "+="
	MINUS_ASSIGN   TokenType = "-="
	STAR_ASSIGN    TokenType = "*="
	SLASH_ASSIGN   TokenType = "/="
	PERCENT_ASSIGN TokenType = "%="

	LT  TokenType = "<"
	GT  TokenType = ">"
	EQ  TokenType = "=="
	NEQ TokenType = "!="
	GTE TokenType = ">="
	LTE TokenType = "<="
	AND TokenType = "&&"
	OR  TokenType = "||"

	// Delimiters
	ASSIGN      TokenType = "="
	RIGHT_ARROW TokenType = "->"
	LEFT_ARROW  TokenType = "<-"
	COLON       TokenType = ":"
	DOT         TokenType = "."
	COMMA       TokenType = ","
	LPAREN      TokenType = "("
	RPAREN      TokenType = ")"
	LBRACE      TokenType = "{"
	RBRACE      TokenType = "}"
	LBRACKET    TokenType = "["
	RBRACKET    TokenType = "]"
	AT          TokenType = "@"

	// KEYWORDS
	MODULE    TokenType = "MODULE"
	IMPORT    TokenType = "IMPORT"
	UNION     TokenType = "UNION"
	DATA      TokenType = "DATA"
	ATTRIBUTE TokenType = "ATTRIBUTE"
	EXTERN    TokenType = "EXTERN"
	FUNCTION  TokenType = "FUNCTION"
	CONST     TokenType = "CONST"
	VAR       TokenType = "VAR"
	TYPE      TokenType = "TYPE"
	SWITCH    TokenType = "SWITCH"
	CASE      TokenType = "CASE"
	IS        TokenType = "IS"
	BREAK     TokenType = "BREAK"
	CONTINUE  TokenType = "CONTINUE"
	RETURN    TokenType = "RETURN"
	IF        TokenType = "IF"
	ELSE      TokenType = "ELSE"
	FOR       TokenType = "FOR"
	BLANK     TokenType = "BLANK"
	TRUE      TokenType = "TRUE"
	FALSE     TokenType = "FALSE"
	VOID      TokenType = "VOID"
)

var keywords = map[string]TokenType{
	"mod":      MODULE,
	"import":   IMPORT,
	"union":    UNION,
	"data":     DATA,
	"attr":     ATTRIBUTE,
	"extern":   EXTERN,
	"fn":       FUNCTION,
	"const":    CONST,
	"var":      VAR,
	"type":     TYPE,
	"switch":   SWITCH,
	"case":     CASE,
	"is":       IS,
	"break":    BREAK,
	"continue": CONTINUE,
	"return":   RETURN,
	"if":       IF,
	"else":     ELSE,
	"for":      FOR,
	"true":     TRUE,
	"false":    FALSE,
	"void":     VOID,
	"_":        BLANK,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
