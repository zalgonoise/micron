package cronlex

// Token represents a unique type to mark lexemes in groups.
type Token uint8

const (
	TokenEOF Token = iota
	TokenError
	TokenAlphaNum
	TokenStar
	TokenComma
	TokenDash
	TokenSlash
	TokenAt
	TokenSpace
)

var tokenStrings = [...]string{
	"TokenEOF",
	"TokenError",
	"TokenAlphaNum",
	"TokenStar",
	"TokenComma",
	"TokenDash",
	"TokenSlash",
	"TokenAt",
	"TokenSpace",
}

// String implements the fmt.Stringer interface.
func (t Token) String() string {
	return tokenStrings[t]
}
