package builder

import (
	"errors"
	"fmt"

	"github.com/zalgonoise/micron/schedule/cronlex"
	"github.com/zalgonoise/micron/schedule/resolve"
)

const (
	seconds = iota
	minutes
	hours
	monthDays
	months
	weekdays
)

const (
	Sunday = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

const (
	minSecond = 0
	maxSecond = 59

	minMinute = 0
	maxMinute = 59

	minHour = 0
	maxHour = 23

	minDay = 1
	maxDay = 31

	minMonth = 1
	maxMonth = 12

	minWeekday = 0
	maxWeekday = 7
)

var (
	ErrInvalidCategory = errors.New("invalid category")
	ErrInvalidResolver = errors.New("invalid resolver type")
	ErrOutOfBounds     = errors.New("value is out-of-bounds")
)

type Scheduler interface {
	Seconds() Resolver
	Minutes() Resolver
	Hours() Resolver
	MonthDays() Resolver
	Months() Resolver
	Weekdays() Resolver
}

type Resolver struct {
	category int
	resolver cronlex.Resolver
}

type schedule struct {
	value int
}

func (s schedule) Seconds() Resolver {
	if s.value < 0 {
		return Resolver{
			category: seconds,
			resolver: resolve.Everytime{},
		}
	}

	return Resolver{
		category: seconds,
		resolver: resolve.FixedSchedule{
			Max: maxSecond,
			At:  s.value,
		},
	}
}

func (s schedule) Minutes() Resolver {
	if s.value < 0 {
		return Resolver{
			category: minutes,
			resolver: resolve.Everytime{},
		}
	}

	return Resolver{
		category: minutes,
		resolver: resolve.FixedSchedule{
			Max: maxMinute,
			At:  s.value,
		},
	}
}

func (s schedule) Hours() Resolver {
	if s.value < 0 {
		return Resolver{
			category: hours,
			resolver: resolve.Everytime{},
		}
	}

	return Resolver{
		category: hours,
		resolver: resolve.FixedSchedule{
			Max: maxHour,
			At:  s.value,
		},
	}
}

func (s schedule) MonthDays() Resolver {
	if s.value < 0 {
		return Resolver{
			category: monthDays,
			resolver: resolve.Everytime{},
		}
	}

	return Resolver{
		category: monthDays,
		resolver: resolve.FixedSchedule{
			Max: maxDay,
			At:  s.value,
		},
	}
}

func (s schedule) Months() Resolver {
	if s.value < 0 {
		return Resolver{
			category: months,
			resolver: resolve.Everytime{},
		}
	}

	return Resolver{
		category: months,
		resolver: resolve.FixedSchedule{
			Max: maxMonth,
			At:  s.value,
		},
	}
}

func (s schedule) Weekdays() Resolver {
	if s.value < 0 {
		return Resolver{
			category: weekdays,
			resolver: resolve.Everytime{},
		}
	}

	return Resolver{
		category: weekdays,
		resolver: resolve.FixedSchedule{
			Max: maxWeekday,
			At:  s.value,
		},
	}
}

func All() Scheduler {
	return schedule{value: -1}
}

func Every(n int) Scheduler {
	return schedule{value: n}
}

type rangeSchedule struct {
	from, to int
}

func (s rangeSchedule) Seconds() Resolver {
	return Resolver{
		category: seconds,
		resolver: resolve.RangeSchedule{
			Max:  maxSecond,
			From: s.from,
			To:   s.to,
		},
	}
}

func (s rangeSchedule) Minutes() Resolver {
	return Resolver{
		category: minutes,
		resolver: resolve.RangeSchedule{
			Max:  maxMinute,
			From: s.from,
			To:   s.to,
		},
	}
}

func (s rangeSchedule) Hours() Resolver {
	return Resolver{
		category: hours,
		resolver: resolve.RangeSchedule{
			Max:  maxHour,
			From: s.from,
			To:   s.to,
		},
	}
}

func (s rangeSchedule) MonthDays() Resolver {
	return Resolver{
		category: monthDays,
		resolver: resolve.RangeSchedule{
			Max:  maxDay,
			From: s.from,
			To:   s.to,
		},
	}
}

func (s rangeSchedule) Months() Resolver {
	return Resolver{
		category: months,
		resolver: resolve.RangeSchedule{
			Max:  maxMonth,
			From: s.from,
			To:   s.to,
		},
	}
}

func (s rangeSchedule) Weekdays() Resolver {
	return Resolver{
		category: weekdays,
		resolver: resolve.RangeSchedule{
			Max:  maxWeekday,
			From: s.from,
			To:   s.to,
		},
	}
}

func Range(from, to int) Scheduler {
	return rangeSchedule{
		from: from,
		to:   to,
	}
}

type stepSchedule struct {
	values []int
}

func (s stepSchedule) Seconds() Resolver {
	return Resolver{
		category: seconds,
		resolver: resolve.StepSchedule{
			Max:   maxSecond,
			Steps: s.values,
		},
	}
}

func (s stepSchedule) Minutes() Resolver {
	return Resolver{
		category: minutes,
		resolver: resolve.StepSchedule{
			Max:   maxMinute,
			Steps: s.values,
		},
	}
}

func (s stepSchedule) Hours() Resolver {
	return Resolver{
		category: hours,
		resolver: resolve.StepSchedule{
			Max:   maxHour,
			Steps: s.values,
		},
	}
}

func (s stepSchedule) MonthDays() Resolver {
	return Resolver{
		category: monthDays,
		resolver: resolve.StepSchedule{
			Max:   maxDay,
			Steps: s.values,
		},
	}
}

func (s stepSchedule) Months() Resolver {
	return Resolver{
		category: months,
		resolver: resolve.StepSchedule{
			Max:   maxMonth,
			Steps: s.values,
		},
	}
}

func (s stepSchedule) Weekdays() Resolver {
	return Resolver{
		category: weekdays,
		resolver: resolve.StepSchedule{
			Max:   maxWeekday,
			Steps: s.values,
		},
	}
}

func On(values ...int) Scheduler {
	return stepSchedule{values: values}
}

func Build(resolvers ...Resolver) (*cronlex.Schedule, error) {
	sched := &cronlex.Schedule{}
	for i := range resolvers {
		if err := validateResolver(resolvers[i]); err != nil {
			return nil, err
		}

		switch resolvers[i].category {
		case seconds:
			sched.Sec = resolvers[i].resolver
		case minutes:
			sched.Min = resolvers[i].resolver
		case hours:
			sched.Hour = resolvers[i].resolver
		case monthDays:
			sched.DayMonth = resolvers[i].resolver
		case months:
			sched.Month = resolvers[i].resolver
		case weekdays:
			sched.DayWeek = resolvers[i].resolver
		}
	}

	return populateSchedule(sched), nil
}

func populateMinutes(start bool, sched *cronlex.Schedule) (bool, *cronlex.Schedule) {
	switch {
	case sched.Min == nil && !start:
		sched.Min = resolve.FixedSchedule{Max: maxMinute, At: minMinute}
	case sched.Min == nil:
		sched.Min = resolve.Everytime{}
	default:
		start = true
	}

	return start, sched
}

func populateHours(start bool, sched *cronlex.Schedule) (bool, *cronlex.Schedule) {
	switch {
	case sched.Hour == nil && !start:
		sched.Hour = resolve.FixedSchedule{Max: maxHour, At: minHour}
	case sched.Hour == nil:
		sched.Hour = resolve.Everytime{}
	default:
		start = true
	}

	return start, sched
}

func populateDays(start bool, sched *cronlex.Schedule) *cronlex.Schedule {
	switch {
	case sched.DayMonth == nil && !start:
		sched.DayMonth = resolve.FixedSchedule{Max: maxDay, At: minDay}
	case sched.DayMonth == nil:
		sched.DayMonth = resolve.Everytime{}
	default:
		start = true
	}

	switch {
	case sched.Month == nil && !start:
		sched.Month = resolve.FixedSchedule{Max: maxMonth, At: minMonth}
	case sched.Month == nil:
		sched.Month = resolve.Everytime{}
	}

	sched.DayWeek = resolve.Everytime{}

	return sched
}

func populateWeekdays(start bool, sched *cronlex.Schedule) *cronlex.Schedule {
	sched.DayMonth = resolve.Everytime{}
	sched.Month = resolve.Everytime{}

	switch {
	case sched.DayWeek == nil && !start:
		sched.DayWeek = resolve.FixedSchedule{Max: maxWeekday, At: minWeekday}
	case sched.DayWeek == nil:
		sched.DayWeek = resolve.Everytime{}
	}

	return sched
}

func populateSchedule(sched *cronlex.Schedule) *cronlex.Schedule {
	var start bool

	switch {
	case sched.Sec == nil:
		sched.Sec = resolve.FixedSchedule{Max: maxSecond, At: minSecond}
	default:
		start = true
	}

	start, sched = populateMinutes(start, sched)
	start, sched = populateHours(start, sched)

	if sched.DayWeek == nil {
		return populateDays(start, sched)
	}

	return populateWeekdays(start, sched)
}

func validateResolver(r Resolver) error {
	switch r.category {
	case seconds:
		return validate(r, minSecond)
	case minutes:
		return validate(r, minMinute)
	case hours:
		return validate(r, minHour)
	case monthDays:
		return validate(r, minDay)
	case months:
		return validate(r, minMonth)
	case weekdays:
		return validate(r, minWeekday)
	default:
		return fmt.Errorf("%w: %d", ErrInvalidCategory, r.category)
	}
}

func validate(r Resolver, minimum int) error {
	switch v := r.resolver.(type) {
	case resolve.Everytime:
		// valid
		return nil
	case resolve.FixedSchedule:
		if v.At < minimum || v.At > v.Max {
			return fmt.Errorf("%w: %d", ErrOutOfBounds, v.At)
		}

		return nil

	case resolve.RangeSchedule:
		var err error

		if v.From < minimum || v.From > v.Max {
			err = fmt.Errorf("%w: from: %d", ErrOutOfBounds, v.From)
		}

		if v.To < minimum || v.To > v.Max {
			return errors.Join(err, fmt.Errorf("%w: to: %d", ErrOutOfBounds, v.From))
		}

		return err
	case resolve.StepSchedule:
		errs := make([]error, 0, len(v.Steps))

		for i := range v.Steps {
			if v.Steps[i] < minimum || v.Steps[i] > v.Max {
				errs = append(errs, fmt.Errorf("%w: step #%d: %d", ErrOutOfBounds, i, v.Steps[i]))
			}
		}

		return errors.Join(errs...)
	default:
		return fmt.Errorf("%w: %#v", ErrInvalidResolver, r)
	}
}
