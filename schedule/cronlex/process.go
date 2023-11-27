package cronlex

import (
	"slices"
	"strconv"
	"strings"

	"github.com/zalgonoise/parse"

	"github.com/zalgonoise/micron/schedule/resolve"
)

const (
	maxSec     = 59
	maxMin     = 59
	maxHour    = 23
	maxDay     = 31
	maxMonth   = 12
	maxWeekday = 7

	extraSunday = 7

	double = 2
)

const (
	reboot = iota
	hourly
	daily
	weekly
	monthly
	yearly
	annually
)

// Resolver describes the capabilities of a cron schedule resolver.
//
// Implementations of Resolver should focus on calculating the difference until the
// next scheduled value, on a per-unit basis. This means that for each configurable schedule element
// (seconds, minutes, hours, etc.), an individual Resolver calculates the next occurrence for a given value.
//
// In the context of dates and timestamps, it enables to simply resolve the next occurrence's date as a difference
// of the current time's units against the Resolver's configuration, and with that information to build the
// timestamp for the next job execution, with a time.Date call, in the schedule.Scheduler component, that would sum the
// current time to the values taken from the Resolver.
//
// Implementations of Resolver must ensure that their logic functions for all date elements of Schedule, provided that
// the Resolver is used in that data structure.
type Resolver interface {
	// Resolve returns the distance to the next occurrence, as unit values.
	Resolve(value int) int
}

// Schedule describes the structure of an (extended) cron schedule, which includes all basic cron schedule elements
// (minutes, hours, day-of-the-month, month and weekdays), as well as support for seconds.
type Schedule struct {
	Sec      Resolver
	Min      Resolver
	Hour     Resolver
	DayMonth Resolver
	Month    Resolver
	DayWeek  Resolver
}

// Parse consumes the input cron string and creates a Schedule from it, also returning an error if raised.
//
// Before parsing the string, this function validates that the cron string does not contain any illegal characters,
// before actually scanning and processing it.
func Parse(cron string) (Schedule, error) {
	if err := validateCharacters(cron); err != nil {
		return Schedule{}, err
	}

	return parse.Run([]byte(cron), StateFunc, ParseFunc, ProcessFunc)
}

// ProcessFunc is the third and last phase of the parser, which consumes a parse.Tree scoped to Token and byte,
// returning the new Schedule and error if raised.
//
// This sequence will validate the nodes in the input parse.Tree, returning an error if raised. Then, depending on the
// configured top-level nodes, it will process the tree in the correct, supported way to derive a Schedule out of it.
func ProcessFunc(t *parse.Tree[Token, byte]) (Schedule, error) {
	if err := Validate(t); err != nil {
		return Schedule{}, err
	}

	var (
		s     Schedule
		nodes = t.List()
	)

	switch len(nodes) {
	case override:
		return buildException(nodes[0]), nil
	case noSeconds:
		s = Schedule{
			Sec: resolve.FixedSchedule{
				Max: maxSec,
				At:  0,
			},
			Min:      buildMinutes(nodes[0]),
			Hour:     buildHours(nodes[1]),
			DayMonth: buildMonthDays(nodes[2]),
			Month:    buildMonths(nodes[3]),
			DayWeek:  buildWeekdays(nodes[4]),
		}
	case withSeconds:
		s = Schedule{
			Sec:      buildSeconds(nodes[0]),
			Min:      buildMinutes(nodes[1]),
			Hour:     buildHours(nodes[2]),
			DayMonth: buildMonthDays(nodes[3]),
			Month:    buildMonths(nodes[4]),
			DayWeek:  buildWeekdays(nodes[5]),
		}
	}
	// convert sundays as 7 into a 0
	if r, ok := s.DayWeek.(resolve.StepSchedule); ok {
		for i := range r.Steps {
			if r.Steps[i] == extraSunday {
				r.Steps[i] = 0

				slices.Sort(r.Steps)
				s.DayWeek = r
			}
		}
	}

	return s, nil
}

func buildSeconds(node *parse.Node[Token, byte]) Resolver {
	switch node.Type {
	case TokenStar:
		return processStar(node, 0, maxSec)
	default:
		return processAlphaNum(node, maxSec, nil)
	}
}

func buildMinutes(node *parse.Node[Token, byte]) Resolver {
	switch node.Type {
	case TokenStar:
		return processStar(node, 0, maxMin)
	default:
		return processAlphaNum(node, maxMin, nil)
	}
}

func buildHours(node *parse.Node[Token, byte]) Resolver {
	switch node.Type {
	case TokenStar:
		return processStar(node, 0, maxHour)
	default:
		return processAlphaNum(node, maxHour, nil)
	}
}

func buildMonthDays(node *parse.Node[Token, byte]) Resolver {
	switch node.Type {
	case TokenStar:
		return processStar(node, 1, maxDay)
	default:
		return processAlphaNum(node, maxDay, nil)
	}
}

func buildMonths(node *parse.Node[Token, byte]) Resolver {
	switch node.Type {
	case TokenStar:
		return processStar(node, 1, maxMonth)
	default:
		return processAlphaNum(node, maxMonth, monthsList)
	}
}

func buildWeekdays(node *parse.Node[Token, byte]) Resolver {
	switch node.Type {
	case TokenStar:
		return processStar(node, 0, maxWeekday)
	default:
		return processAlphaNum(node, maxWeekday, weekdaysList)
	}
}

func defaultSchedule() Schedule {
	return Schedule{
		Sec:      resolve.FixedSchedule{Max: maxSec, At: 0},
		Min:      resolve.FixedSchedule{Max: maxMin, At: 0},
		Hour:     resolve.Everytime{},
		DayMonth: resolve.Everytime{},
		Month:    resolve.Everytime{},
		DayWeek:  resolve.Everytime{},
	}
}

func buildException(node *parse.Node[Token, byte]) Schedule {
	if node.Type != TokenAt {
		return defaultSchedule()
	}

	value := getValue(node.Edges[0], exceptionsList)
	switch value {
	// TODO: implement reboot (case 0:)
	case reboot:
		return defaultSchedule()
	case daily:
		return Schedule{
			Sec:      resolve.FixedSchedule{Max: maxSec, At: 0},
			Min:      resolve.FixedSchedule{Max: maxMin, At: 0},
			Hour:     resolve.FixedSchedule{Max: maxHour, At: 0},
			DayMonth: resolve.Everytime{},
			Month:    resolve.Everytime{},
			DayWeek:  resolve.Everytime{},
		}
	case weekly:
		return Schedule{
			Sec:      resolve.FixedSchedule{Max: maxSec, At: 0},
			Min:      resolve.FixedSchedule{Max: maxMin, At: 0},
			Hour:     resolve.FixedSchedule{Max: maxHour, At: 0},
			DayMonth: resolve.Everytime{},
			Month:    resolve.Everytime{},
			DayWeek: resolve.FixedSchedule{
				Max: maxWeekday,
				At:  0,
			},
		}
	case monthly:
		return Schedule{
			Sec:      resolve.FixedSchedule{Max: maxSec, At: 0},
			Min:      resolve.FixedSchedule{Max: maxMin, At: 0},
			Hour:     resolve.FixedSchedule{Max: maxHour, At: 0},
			DayMonth: resolve.FixedSchedule{Max: maxDay, At: 1},
			Month:    resolve.Everytime{},
			DayWeek:  resolve.Everytime{},
		}
	case yearly, annually:
		return Schedule{
			Sec:      resolve.FixedSchedule{Max: maxSec, At: 0},
			Min:      resolve.FixedSchedule{Max: maxMin, At: 0},
			Hour:     resolve.FixedSchedule{Max: maxHour, At: 0},
			DayMonth: resolve.FixedSchedule{Max: maxDay, At: 1},
			Month:    resolve.FixedSchedule{Max: maxMonth, At: 1},
			DayWeek:  resolve.Everytime{},
		}
	default:
		// case 1 -- set as default behavior
		return defaultSchedule()
	}
}

func getValue(node *parse.Node[Token, byte], valueList []string) int {
	value := node.Value

	// try to use the value as a number
	if len(value) > 0 && value[0] >= '0' && value[0] <= '9' {
		if num, err := strconv.Atoi(string(value)); err == nil {
			return num
		}
	}

	// fallback to using it as a string
	v := strings.ToUpper(string(value))
	// input has already been validated, there will be a match.
	// returning the n variable set here ensures more test coverage
	n := -1

	for idx := range valueList {
		if v == valueList[idx] {
			n = idx

			break
		}
	}

	return n
}

func getValueFromSymbol(symbol *parse.Node[Token, byte], valueList []string) int {
	if len(symbol.Edges) == 1 {
		return getValue(symbol.Edges[0], valueList)
	}

	return -1
}

func processAlphaNum(n *parse.Node[Token, byte], maximum int, valueList []string) Resolver {
	value := getValue(n, valueList)

	switch len(n.Edges) {
	case 0:
		return resolve.FixedSchedule{
			Max: maximum,
			At:  value,
		}
	default:
		// there is only one range in the set, do a range-schedule approach
		if len(n.Edges) == 1 && n.Edges[0].Type == TokenDash {
			return resolve.RangeSchedule{
				Max:  maximum,
				From: value,
				To:   getValueFromSymbol(n.Edges[0], valueList),
			}
		}

		stepValues := make([]int, 0, len(n.Edges)*double)

		// on a mixed scenario we walk through the edges and build a step-schedule out of the combinations provided
		// for reference, TokenDash means a range, TokenSlash means a frequency and TokenComma carries the next value
		//
		// the value variable is reused for this purpose
		for i := range n.Edges {
			//nolint:exhaustive // no need to check on all token types
			switch n.Edges[i].Type {
			case TokenComma:
				// don't leave the initial value dangling when changing Tokens
				if i == 0 {
					stepValues = append(stepValues, value)
				}

				// it's OK to append the (child) value in a comma node
				// even if the next node is a range or a frequency, the same value will be included and repeated values deleted
				//
				// this Token also sets the `cur` variable in case the following Token is a range or frequency
				if v := getValueFromSymbol(n.Edges[i], valueList); v >= 0 {
					stepValues = append(stepValues, v)

					value = v
				}

			case TokenDash:
				if to := getValueFromSymbol(n.Edges[i], valueList); to >= 0 {
					stepValues = append(stepValues, buildRange(value, to)...)
				}

			case TokenSlash:
				if freq := getValueFromSymbol(n.Edges[i], valueList); freq >= 0 {
					stepValues = append(stepValues, buildFreq(value, maximum, freq)...)
				}
			}
		}

		slices.Sort(stepValues)
		stepValues = slices.Compact(stepValues)

		return resolve.StepSchedule{
			Max:   maximum,
			Steps: stepValues,
		}
	}
}

func processStar(n *parse.Node[Token, byte], minimum, maximum int) Resolver {
	switch len(n.Edges) {
	case 1:
		if n.Edges[0].Type == TokenSlash && len(n.Edges[0].Edges) == 1 {
			stepValue, err := strconv.Atoi(string(n.Edges[0].Edges[0].Value))
			if err != nil {
				return resolve.Everytime{}
			}

			return resolve.NewStepSchedule(minimum, maximum, maximum, stepValue)
		}
	default:
	}

	return resolve.Everytime{}
}

func buildRange(from, to int) []int {
	if to < from {
		return []int{}
	}

	out := make([]int, 0, to-from)
	for i := from; i <= to; i++ {
		out = append(out, i)
	}

	return out
}

func buildFreq(base, maximum, freq int) []int {
	if freq == 0 || base > maximum {
		return []int{}
	}

	out := make([]int, 0, maximum-base/freq)
	for i := base; i <= maximum; i += freq {
		out = append(out, i)
	}

	return out
}
