package cronlex

import (
	"github.com/zalgonoise/parse"
)

// ParseFunc is the second and middle phase of the parser, which consumes a parse.Tree scoped to Token and byte,
// in tandem with StateFunc, as a lexer-parser state-machine strategy.
//
// This function continuously consumes tokens emitted by the StateFunc lexer portion of the logic, and organizes them
// in an abstract syntax tree. This AST is then processed into a Schedule through the ProcessFunc sequence, as a
// parse.Run function call.
//
// The AST keeps a top-level node that branches into several nodes, representative of the number of top-level child
// nodes in the cron string ("* * * * *" means there are 5 top-level child nodes; "@weekly" means there is 1 top-level
// child node). If a given top-level child node contains more information than a single value (e.g. ranges, sets), then
// the top-level child node will be the parent to more nodes containing any Token chained to that top-level child node.
func ParseFunc(t *parse.Tree[Token, byte]) parse.ParseFn[Token, byte] {
	switch t.Peek().Type {
	case TokenAt:
		return parseAt
	case TokenStar:
		return parseStar
	case TokenAlphaNum:
		return parseAlphanum
	case TokenEOF:
		return nil
	default:
		return nil
	}
}

func parseAt(t *parse.Tree[Token, byte]) parse.ParseFn[Token, byte] {
	t.Node(t.Next())

	switch t.Peek().Type {
	case TokenAlphaNum:
		return parseAlphanum
	default:
		item := t.Next()
		item.Type = TokenError
		_ = t.Set(t.Parent())

		return ParseFunc
	}
}

func parseStar(t *parse.Tree[Token, byte]) parse.ParseFn[Token, byte] {
	t.Node(t.Next())

	switch t.Peek().Type {
	case TokenSpace:
		_ = t.Set(t.Parent())
		t.Next()

		return ParseFunc
	case TokenSlash:
		return parseAlphanum
	default:
		_ = t.Set(t.Parent())

		return nil
	}
}

func parseAlphanumSymbols(t *parse.Tree[Token, byte]) parse.ParseFn[Token, byte] {
	t.Node(t.Next())

	switch t.Peek().Type {
	case TokenAlphaNum:
		t.Node(t.Next())
		_ = t.Set(t.Parent().Parent)

		return parseAlphanum
	default:
		item := t.Next()
		item.Type = TokenError
		t.Node(item)

		return ParseFunc
	}
}

func parseAlphanum(t *parse.Tree[Token, byte]) parse.ParseFn[Token, byte] {
	switch t.Peek().Type {
	case TokenAlphaNum:
		t.Node(t.Next())

		return parseAlphanum
	case TokenComma, TokenDash, TokenSlash:
		return parseAlphanumSymbols
	case TokenSpace:
		_ = t.Set(t.Parent())
		t.Next()

		return ParseFunc
	}

	return nil
}
