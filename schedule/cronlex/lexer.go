package cronlex

import (
	"github.com/zalgonoise/lex"
)

// StateFunc is the first phase of the parser, which consumes the cron string's lexemes while emitting
// meaningful tokens on what type of data they portray.
//
// This function works in tandem with ParseFunc, as a parser-lexer state-machine during the parse.Run call, in Parse.
//
// As the lexer scans through each character in the cron string, it emits tokens that are representative on the kind of
// data at hand, as well as the actual (zero-to-many) bytes that compose that token. E.g. a set of alphanumeric
// characters emit a TokenAlphaNum Token containing all of those characters, while a "*" emits a TokenStar Token,
// containing the "*" as value. Of course, a TokenEOF Token would hold no value.
//
// The ParseFunc will then consume these emitted Token from a channel, and organize its AST appropriately within its
// own logic.
func StateFunc(l lex.Lexer[Token, byte]) lex.StateFn[Token, byte] {
	switch l.Next() {
	case '@':
		l.Emit(TokenAt)

		return stateException
	case '-':
		l.Emit(TokenDash)

		return StateFunc
	case ',':
		l.Emit(TokenComma)

		return StateFunc
	case '/':
		l.Emit(TokenSlash)

		return StateFunc
	case '*':
		l.Emit(TokenStar)

		return StateFunc
	case ' ':
		l.Emit(TokenSpace)

		return StateFunc
	case 0:
		l.Emit(TokenEOF)

		return nil
	default:
		return stateAlphanumeric
	}
}

func stateAlphanumeric(l lex.Lexer[Token, byte]) lex.StateFn[Token, byte] {
	l.Backup() // undo l.Next() for the l.AcceptRun call

	for {
		if item := l.Cur(); (item >= '0' && item <= '9') || (item >= 'A' && item <= 'Z') || (item >= 'a' && item <= 'z') {
			l.Next()

			continue
		}
		break
	}

	if l.Width() > 0 {
		l.Emit(TokenAlphaNum)
	}

	return StateFunc
}

func stateException(l lex.Lexer[Token, byte]) lex.StateFn[Token, byte] {
	l.Backup() // undo l.Next() for the l.AcceptRun call

	for {
		if item := l.Cur(); (item >= 'A' && item <= 'Z') || (item >= 'a' && item <= 'z') {
			l.Next()

			continue
		}
		break
	}

	if l.Width() > 0 {
		l.Emit(TokenAlphaNum)
	}

	return StateFunc
}
