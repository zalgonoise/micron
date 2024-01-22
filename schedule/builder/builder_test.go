package builder

import (
	"errors"
	"testing"

	"github.com/zalgonoise/micron/schedule/cronlex"
	"github.com/zalgonoise/micron/schedule/resolve"
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
			if steps, ok := testcase.wants.Sec.(resolve.StepSchedule); ok {
				got, ok := sched.Sec.(resolve.StepSchedule)

				isEqual(t, true, ok)

				isEqual(t, steps.Max, got.Max)
				for i := range steps.Steps {
					isEqual(t, steps.Steps[i], got.Steps[i])
				}
			} else {
				isEqual(t, testcase.wants.Sec, sched.Sec)
			}

			if steps, ok := testcase.wants.Min.(resolve.StepSchedule); ok {
				got, ok := sched.Min.(resolve.StepSchedule)

				isEqual(t, true, ok)

				isEqual(t, steps.Max, got.Max)
				for i := range steps.Steps {
					isEqual(t, steps.Steps[i], got.Steps[i])
				}
			} else {
				isEqual(t, testcase.wants.Min, sched.Min)
			}

			if steps, ok := testcase.wants.Hour.(resolve.StepSchedule); ok {
				got, ok := sched.Hour.(resolve.StepSchedule)

				isEqual(t, true, ok)

				isEqual(t, steps.Max, got.Max)
				for i := range steps.Steps {
					isEqual(t, steps.Steps[i], got.Steps[i])
				}
			} else {
				isEqual(t, testcase.wants.Hour, sched.Hour)
			}

			if steps, ok := testcase.wants.DayMonth.(resolve.StepSchedule); ok {
				got, ok := sched.DayMonth.(resolve.StepSchedule)

				isEqual(t, true, ok)

				isEqual(t, steps.Max, got.Max)
				for i := range steps.Steps {
					isEqual(t, steps.Steps[i], got.Steps[i])
				}
			} else {
				isEqual(t, testcase.wants.DayMonth, sched.DayMonth)
			}

			if steps, ok := testcase.wants.Month.(resolve.StepSchedule); ok {
				got, ok := sched.Month.(resolve.StepSchedule)

				isEqual(t, true, ok)

				isEqual(t, steps.Max, got.Max)
				for i := range steps.Steps {
					isEqual(t, steps.Steps[i], got.Steps[i])
				}
			} else {
				isEqual(t, testcase.wants.Month, sched.Month)
			}

			if steps, ok := testcase.wants.DayWeek.(resolve.StepSchedule); ok {
				got, ok := sched.DayWeek.(resolve.StepSchedule)

				isEqual(t, true, ok)

				isEqual(t, steps.Max, got.Max)
				for i := range steps.Steps {
					isEqual(t, steps.Steps[i], got.Steps[i])
				}
			} else {
				isEqual(t, testcase.wants.DayWeek, sched.DayWeek)
			}
		})
	}
}

func isEqual[T comparable](t *testing.T, wants, got T) {
	if got != wants {
		t.Errorf("output mismatch error: wanted %v ; got %v", wants, got)
		t.Fail()

		return
	}

	t.Logf("output matched expected value: %v", wants)
}
