package builder

import (
	"errors"
	"testing"

	"github.com/zalgonoise/micron/v3/schedule/cronlex"
	"github.com/zalgonoise/micron/v3/schedule/resolve"
)

func TestBuild(t *testing.T) {
	for _, testcase := range []struct {
		name      string
		resolvers []Resolver
		wants     cronlex.Schedule
		err       error
	}{
		{
			name: "EveryMinute",
			resolvers: []Resolver{
				All().Minutes(),
			},
			wants: cronlex.Schedule{
				Sec: resolve.FixedSchedule{
					Max: maxSecond,
					At:  minSecond,
				},
				Min:      resolve.Everytime{},
				Hour:     resolve.Everytime{},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek:  resolve.Everytime{},
			},
		},
		{
			name: "At5thMonth",
			resolvers: []Resolver{
				Every(5).Months(),
			},
			wants: cronlex.Schedule{
				Sec: resolve.FixedSchedule{
					Max: maxSecond,
					At:  minSecond,
				},
				Min: resolve.FixedSchedule{
					Max: maxMinute,
					At:  minMinute,
				},
				Hour: resolve.FixedSchedule{
					Max: maxHour,
					At:  minHour,
				},
				DayMonth: resolve.FixedSchedule{
					Max: maxDay,
					At:  minDay,
				},
				Month: resolve.FixedSchedule{
					Max: maxMonth,
					At:  5,
				},
				DayWeek: resolve.Everytime{},
			},
		},
		{
			name: "RangeMondayToFriday",
			resolvers: []Resolver{
				Range(Monday, Friday).Weekdays(),
			},
			wants: cronlex.Schedule{
				Sec: resolve.FixedSchedule{
					Max: maxSecond,
					At:  minSecond,
				},
				Min: resolve.FixedSchedule{
					Max: maxMinute,
					At:  minMinute,
				},
				Hour: resolve.FixedSchedule{
					Max: maxHour,
					At:  minHour,
				},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek: resolve.RangeSchedule{
					Max:  maxWeekday,
					From: Monday,
					To:   Friday,
				},
			},
		},
		{
			name: "OnAFewDaysOfEveryMonth",
			resolvers: []Resolver{
				On(1, 3, 4, 7, 10).MonthDays(),
			},
			wants: cronlex.Schedule{
				Sec: resolve.FixedSchedule{
					Max: maxSecond,
					At:  minSecond,
				},
				Min: resolve.FixedSchedule{
					Max: maxMinute,
					At:  minMinute,
				},
				Hour: resolve.FixedSchedule{
					Max: maxHour,
					At:  minHour,
				},
				DayMonth: resolve.StepSchedule{
					Max:   maxDay,
					Steps: []int{1, 3, 4, 7, 10},
				},
				Month:   resolve.Everytime{},
				DayWeek: resolve.Everytime{},
			},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			sched, err := Build(testcase.resolvers...)

			isEqual(t, true, errors.Is(err, testcase.err))
			isEqualResolver(t, testcase.wants.Sec, sched.Sec)
			isEqualResolver(t, testcase.wants.Min, sched.Min)
			isEqualResolver(t, testcase.wants.Hour, sched.Hour)
			isEqualResolver(t, testcase.wants.DayMonth, sched.DayMonth)
			isEqualResolver(t, testcase.wants.Month, sched.Month)
			isEqualResolver(t, testcase.wants.DayWeek, sched.DayWeek)
		})
	}
}

func isEqualResolver(t *testing.T, wants, got cronlex.Resolver) {
	if steps, ok := wants.(resolve.StepSchedule); ok {
		got, ok := got.(resolve.StepSchedule)

		isEqual(t, true, ok)
		isEqual(t, steps.Max, got.Max)

		for i := range steps.Steps {
			isEqual(t, steps.Steps[i], got.Steps[i])
		}

		return
	}

	isEqual(t, wants, got)
}

func isEqual[T comparable](t *testing.T, wants, got T) {
	if got != wants {
		t.Errorf("output mismatch error: wanted %v ; got %v", wants, got)
		t.Fail()

		return
	}

	t.Logf("output matched expected value: %v", wants)
}
