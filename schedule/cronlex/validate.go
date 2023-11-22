package cronlex

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/zalgonoise/parse"
	"github.com/zalgonoise/x/errs"
)

const (
	errDomain = errs.Domain("x/cron/schedule/cronlex")

	ErrEmpty       = errs.Kind("empty")
	ErrInvalid     = errs.Kind("invalid")
	ErrUnsupported = errs.Kind("unsupported")
	ErrOutOfBounds = errs.Kind("out-of-bounds")

	ErrInput     = errs.Entity("input")
	ErrNumNodes  = errs.Entity("number of nodes")
	ErrNodeType  = errs.Entity("node type")
	ErrNumEdges  = errs.Entity("number of edges")
	ErrFrequency = errs.Entity("frequency")
	ErrAlphanum  = errs.Entity("alphanumeric value")
	ErrCharacter = errs.Entity("character")

	ErrMinutes   = errs.Entity("minutes value")
	ErrHours     = errs.Entity("hours value")
	ErrMonthDays = errs.Entity("days of the month value")
	ErrMonths    = errs.Entity("month value")
	ErrWeekDays  = errs.Entity("days of the week value")
)

var (
	ErrEmptyInput          = errs.WithDomain(errDomain, ErrEmpty, ErrInput)
	ErrInvalidNumNodes     = errs.WithDomain(errDomain, ErrInvalid, ErrNumNodes)
	ErrInvalidNodeType     = errs.WithDomain(errDomain, ErrInvalid, ErrNodeType)
	ErrInvalidNumEdges     = errs.WithDomain(errDomain, ErrInvalid, ErrNumEdges)
	ErrInvalidFrequency    = errs.WithDomain(errDomain, ErrInvalid, ErrFrequency)
	ErrUnsupportedAlphanum = errs.WithDomain(errDomain, ErrUnsupported, ErrAlphanum)
	ErrOutOfBoundsAlphanum = errs.WithDomain(errDomain, ErrOutOfBounds, ErrAlphanum)
	ErrEmptyAlphanum       = errs.WithDomain(errDomain, ErrEmpty, ErrAlphanum)
	ErrInvalidAlphanum     = errs.WithDomain(errDomain, ErrInvalid, ErrAlphanum)
	ErrInvalidCharacter    = errs.WithDomain(errDomain, ErrInvalid, ErrCharacter)

	monthsList = []string{
		0:  "",
		1:  "JAN",
		2:  "FEB",
		3:  "MAR",
		4:  "APR",
		5:  "MAY",
		6:  "JUN",
		7:  "JUL",
		8:  "AUG",
		9:  "SEP",
		10: "OCT",
		11: "NOV",
		12: "DEC",
	}

	weekdaysList = []string{
		0: "SUN",
		1: "MON",
		2: "TUE",
		3: "WED",
		4: "THU",
		5: "FRI",
		6: "SAT",
		7: "SUN", // non-standard
	}

	exceptionsList = []string{
		0: "REBOOT",
		1: "HOURLY",
		2: "DAILY",
		3: "WEEKLY",
		4: "MONTHLY",
		5: "ANNUALLY",
		6: "YEARLY",
	}
)

func validateCharacters(s string) error {
	if s == "" {
		return ErrEmptyInput
	}

	for i := range s {
		if (s[i] >= 'a' && s[i] <= 'z') ||
			(s[i] >= 'A' && s[i] <= 'Z') ||
			(s[i] >= '0' && s[i] <= '9') ||
			s[i] == ' ' ||
			s[i] == '*' ||
			s[i] == ',' ||
			s[i] == '/' ||
			s[i] == '-' ||
			s[i] == '@' {
			continue
		}
		return fmt.Errorf("%w: %v -- %q", ErrInvalidCharacter, s[i], s)
	}

	return nil
}

// Validate scans the entire parse.Tree for inconsistencies or validation errors, returning them if raised.
func Validate(t *parse.Tree[Token, byte]) error {
	nodes := t.List()

	switch len(nodes) {
	case 1:
		return validateOverride(nodes[0])
	case 5:
		return errors.Join(
			validateMinutes(nodes[0]),
			validateHours(nodes[1]),
			validateMonthDays(nodes[2]),
			validateMonths(nodes[3]),
			validateWeekDays(nodes[4]),
		)
	case 6:
		return errors.Join(
			validateSeconds(nodes[0]),
			validateMinutes(nodes[1]),
			validateHours(nodes[2]),
			validateMonthDays(nodes[3]),
			validateMonths(nodes[4]),
			validateWeekDays(nodes[5]),
		)
	default:
		return fmt.Errorf("%w: %d", ErrInvalidNumNodes, len(nodes))
	}
}

func validateOverride(node *parse.Node[Token, byte]) error {
	if node.Type != TokenAt {
		return fmt.Errorf("%w: %T -- %v", ErrInvalidNodeType, node.Type, node.Value)
	}

	if len(node.Edges) != 1 {
		return fmt.Errorf("%w: %d", ErrInvalidNumEdges, len(node.Edges))
	}

	frequency := string(node.Edges[0].Value)

	switch frequency {
	case "yearly", "annually", "monthly", "weekly", "daily", "hourly", "reboot":
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidFrequency, frequency)
	}
}

func validateNumber(value string, minimum, maximum int) error {
	num, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("%w [%s]: %w", ErrUnsupportedAlphanum, value, err)
	}

	if num < minimum || num > maximum {
		return fmt.Errorf("%w [%d]: min: %d; max: %d", ErrOutOfBoundsAlphanum, num, minimum, maximum)
	}

	return nil
}

func validateAlpha(value string, minimum, maximum int, valueList []string) error {
	if value == "" {
		return ErrEmptyAlphanum
	}

	if value[0] >= '0' && value[0] <= '9' {
		return validateNumber(value, minimum, maximum)
	}

	value = strings.ToUpper(value)

	for i := range valueList {
		if value == valueList[i] {
			return nil
		}
	}

	return fmt.Errorf("%w: %s", ErrInvalidAlphanum, value)
}

func validateSymbols(
	edges []*parse.Node[Token, byte],
	maxEdges int,
	validSymbols []Token,
	valueFunc func(string) error,
) error {
	switch {
	case len(edges) == 0:
		return nil
	case len(edges) > maxEdges:
		return fmt.Errorf("%w: %d", ErrInvalidNumEdges, len(edges))
	default:
	edgeLoop:
		for i := range edges {
			for idx := range validSymbols {
				if edges[i].Type == validSymbols[idx] {
					if len(edges[i].Edges) != 1 {
						return fmt.Errorf("%w: %d", ErrInvalidNumEdges, len(edges[i].Edges))
					}

					switch edges[i].Edges[0].Type {
					case TokenAlphaNum:
					// ok state
					case TokenError:
						return fmt.Errorf("%w: %v -- %q", ErrInvalidAlphanum, edges[i].Edges[0].Type, string(edges[i].Edges[0].Value))
					}

					if err := valueFunc(string(edges[i].Edges[0].Value)); err != nil {
						return err
					}

					continue edgeLoop
				}
			}
		}

		return nil
	}
}

func validateField(node *parse.Node[Token, byte], maxEdges, minimum, maximum int, valueFunc func(string) error) error {
	switch node.Type {
	case TokenStar:
		// star is OK by itself -- check if there is a slash token
		if err := validateSymbols(node.Edges, 1, []Token{TokenSlash}, valueFunc); err != nil {
			return err
		}

		return nil
	case TokenAlphaNum:
		err := validateNumber(string(node.Value), minimum, maximum)

		if symbolErr := validateSymbols(node.Edges, maxEdges, []Token{TokenAlphaNum, TokenSlash, TokenComma, TokenDash}, valueFunc); symbolErr != nil {
			return errors.Join(err, symbolErr)
		}

		// check the values of the symbols, if any
		if len(node.Edges) > 0 {
			for i := range node.Edges {
				for idx := range node.Edges[i].Edges {
					if err := validateField(node.Edges[i].Edges[idx], maxEdges, minimum, maximum, valueFunc); err != nil {
						return err
					}
				}
			}
		}

		return nil
	default:
		return fmt.Errorf("%w: %T -- %v", ErrInvalidNodeType, node.Type, node.Value)
	}
}

func validateSeconds(node *parse.Node[Token, byte]) error {
	if err := validateField(node, 60, 0, 59, func(s string) error {
		return validateNumber(s, 0, 59)
	}); err != nil {
		return fmt.Errorf("%w (%w)", err, ErrMinutes)
	}

	return nil
}

func validateMinutes(node *parse.Node[Token, byte]) error {
	if err := validateField(node, 60, 0, 59, func(s string) error {
		return validateNumber(s, 0, 59)
	}); err != nil {
		return fmt.Errorf("%w (%w)", err, ErrMinutes)
	}

	return nil
}

func validateHours(node *parse.Node[Token, byte]) error {
	if err := validateField(node, 24, 0, 23, func(s string) error {
		return validateNumber(s, 0, 23)
	}); err != nil {
		return fmt.Errorf("%w (%w)", err, ErrHours)
	}

	return nil
}

func validateMonthDays(node *parse.Node[Token, byte]) error {
	if err := validateField(node, 31, 1, 31, func(s string) error {
		return validateNumber(s, 1, 31)
	}); err != nil {
		return fmt.Errorf("%w (%w)", err, ErrMonthDays)
	}

	return nil
}

func validateMonths(node *parse.Node[Token, byte]) error {
	if err := validateField(node, 12, 1, 12, func(s string) error {
		return validateAlpha(s, 1, 12, monthsList)
	}); err != nil {
		return fmt.Errorf("%w (%w)", err, ErrMonths)
	}

	return nil
}

func validateWeekDays(node *parse.Node[Token, byte]) error {
	if err := validateField(node, 7, 0, 7, func(s string) error {
		return validateAlpha(s, 0, 7, weekdaysList)
	}); err != nil {
		return fmt.Errorf("%w (%w)", err, ErrWeekDays)
	}

	return nil
}
