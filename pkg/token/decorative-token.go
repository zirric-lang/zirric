package token

type DecorativeTokenType string

type DecorativeToken struct {
	Type    DecorativeTokenType
	Literal string
}

const (
	DECO_COMMENT DecorativeTokenType = "COMMENT"
	DECO_INLINE  DecorativeTokenType = "INLINE_WS"
	DECO_MULTI   DecorativeTokenType = "MULTILINE_WS"
)
