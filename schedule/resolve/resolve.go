package resolve

import (
	"slices"
)

func EqualResolver(a, b interface {
	Resolve(int) int
}) bool {
	switch resolver := a.(type) {
	case Everytime:
		if r, ok := b.(Everytime); ok {
			return EqualEverytime(resolver, r)
		}

		return false
	case FixedSchedule:
		if r, ok := b.(FixedSchedule); ok {
			return EqualFixedSchedule(resolver, r)
		}

		return false
	case RangeSchedule:
		if r, ok := b.(RangeSchedule); ok {
			return EqualRangeSchedule(resolver, r)
		}

		return false
	case StepSchedule:
		if r, ok := b.(StepSchedule); ok {
			return EqualStepSchedule(resolver, r)
		}

		return false
	default:
		return false
	}
}

// Everytime always resolves to zero, as a constantly occurring resolver.
type Everytime struct{}

// Resolve returns the distance to the next occurrence, as unit values.
func (s Everytime) Resolve(_ int) int {
	return 0
}

func EqualEverytime(_, _ Everytime) bool { return true }

// FixedSchedule resolves on a specific value, described as At. It also stores Max to delimit the maximum range for
// this resolver.
type FixedSchedule struct {
	Max int
	At  int
}

// Resolve returns the distance to the next occurrence, as unit values.
func (s FixedSchedule) Resolve(value int) int {
	return diff(value, s.At, s.At, s.Max)
}

func EqualFixedSchedule(a, b FixedSchedule) bool {
	return a.Max == b.Max &&
		a.At == b.At
}

// RangeSchedule resolves on every value between From and To. It also stores Max to delimit the maximum range for
// this resolver.
type RangeSchedule struct {
	Max  int
	From int
	To   int
}

// Resolve returns the distance to the next occurrence, as unit values.
func (s RangeSchedule) Resolve(value int) int {
	if value > s.From && value < s.To {
		return 0
	}

	return diff(value, s.From, s.To, s.Max)
}

func EqualRangeSchedule(a, b RangeSchedule) bool {
	return a.Max == b.Max &&
		a.From == b.From &&
		a.To == b.Max
}

// StepSchedule resolves on specific values listed in Steps. It also stores Max to delimit the maximum range for
// this resolver.
type StepSchedule struct {
	Max   int
	Steps []int
}

// Resolve returns the distance to the next occurrence, as unit values.
func (s StepSchedule) Resolve(value int) int {
	offset := -1

	for i := range s.Steps {
		if offset == -1 {
			offset = diff(value, s.Steps[i], s.Steps[i], s.Max)

			continue
		}

		if n := diff(value, s.Steps[i], s.Steps[i], s.Max); n < offset {
			offset = n
		}
	}

	return offset
}

func EqualStepSchedule(a, b StepSchedule) bool {
	return a.Max == b.Max &&
		slices.Equal(a.Steps, b.Steps)
}

func diff(value, from, to, maximum int) int {
	if value > to {
		return from + maximum - value
	}

	return from - value
}

// NewStepSchedule is a constructor to quickly build StepSchedule types, using key values to
// create the steps -- using from and to delimiters as well as the resolver's maximum value, and a
// frequency.
func NewStepSchedule(from, to, maximum, frequency int) StepSchedule {
	return StepSchedule{
		Max:   maximum,
		Steps: newValueRange(from, to, frequency),
	}
}

func newValueRange(from, to, frequency int) []int {
	if frequency == 0 || from > to {
		return []int{}
	}

	r := make([]int, 0, to-from/frequency)

	for i := from; i <= to; i += frequency {
		r = append(r, i)
	}

	return r
}
