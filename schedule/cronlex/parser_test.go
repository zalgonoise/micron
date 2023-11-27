package cronlex

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zalgonoise/x/is"

	"github.com/zalgonoise/micron/schedule/resolve"
)

func TestParser(t *testing.T) {
	for _, testcase := range []struct {
		name  string
		input string
		wants Schedule
		err   error
	}{
		{
			name:  "Success/Simple/AllStar",
			input: "* * * * *",
			wants: Schedule{
				Sec:      resolve.FixedSchedule{Max: 59, At: 0},
				Min:      resolve.Everytime{},
				Hour:     resolve.Everytime{},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek:  resolve.Everytime{},
			},
		},
		{
			name:  "Success/Simple/AllStarWithSeconds",
			input: "* * * * * *",
			wants: Schedule{
				Sec:      resolve.Everytime{},
				Min:      resolve.Everytime{},
				Hour:     resolve.Everytime{},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek:  resolve.Everytime{},
			},
		},
		{
			name:  "Success/Simple/DefaultCronWithSeconds",
			input: "0 * * * * *",
			wants: Schedule{
				Sec:      resolve.FixedSchedule{Max: 59, At: 0},
				Min:      resolve.Everytime{},
				Hour:     resolve.Everytime{},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek:  resolve.Everytime{},
			},
		},
		{
			name:  "Success/Simple/EveryMinuteZero",
			input: "0 * * * *",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  0,
				},
				Hour:     resolve.Everytime{},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek:  resolve.Everytime{},
			},
		},
		{
			name:  "Success/Simple/LargeMinute",
			input: "50 * * * *",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  50,
				},
				Hour:     resolve.Everytime{},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek:  resolve.Everytime{},
			},
		},
		{
			name:  "Success/Simple/Every3rdMinute",
			input: "*/3 * * * *",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.StepSchedule{
					Max:   59,
					Steps: []int{0, 3, 6, 9, 12, 15, 18, 21, 24, 27, 30, 33, 36, 39, 42, 45, 48, 51, 54, 57},
				},
				Hour:     resolve.Everytime{},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek:  resolve.Everytime{},
			},
		},
		{
			name:  "Success/Range",
			input: "0/3 * * * *",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.StepSchedule{
					Max:   59,
					Steps: []int{0, 3, 6, 9, 12, 15, 18, 21, 24, 27, 30, 33, 36, 39, 42, 45, 48, 51, 54, 57},
				},
				Hour:     resolve.Everytime{},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek:  resolve.Everytime{},
			},
		},
		{
			name:  "Success/Simple/EveryMinuteFrom0Through3",
			input: "0-3 * * * *",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.RangeSchedule{
					Max:  59,
					From: 0,
					To:   3,
				},
				Hour:     resolve.Everytime{},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek:  resolve.Everytime{},
			},
		},
		{
			name:  "Success/Simple/EveryMinuteFrom0Through3And5And7",
			input: "0-3,5,7 * * * *",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.StepSchedule{
					Max:   59,
					Steps: []int{0, 1, 2, 3, 5, 7},
				},
				Hour:     resolve.Everytime{},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek:  resolve.Everytime{},
			},
		},
		{
			name: "Success/Simple/EveryMinuteLiteral",
			//nolint:lll // long string literals
			input: "0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59 * * * *",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.StepSchedule{
					Max: 59,
					Steps: []int{
						0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28,
						29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55,
						56, 57, 58, 59,
					},
				},
				Hour:     resolve.Everytime{},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek:  resolve.Everytime{},
			},
		},
		{
			name:  "Success/Simple/EveryHourRange",
			input: "0 0-23 * * *",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  0,
				},
				Hour: resolve.RangeSchedule{
					Max:  23,
					From: 0,
					To:   23,
				},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek:  resolve.Everytime{},
			},
		},
		{
			name:  "Success/Simple/EveryDayRange",
			input: "0 0 1-31 * *",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  0,
				},
				Hour: resolve.FixedSchedule{
					Max: 23,
					At:  0,
				},
				DayMonth: resolve.RangeSchedule{
					Max:  31,
					From: 1,
					To:   31,
				},
				Month:   resolve.Everytime{},
				DayWeek: resolve.Everytime{},
			},
		},
		{
			name:  "Success/Simple/EveryMonthNumericLiteral",
			input: "0 0 1 1,2,3,4,5,6,7,8,9,10,11,12 *",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  0,
				},
				Hour: resolve.FixedSchedule{
					Max: 23,
					At:  0,
				},
				DayMonth: resolve.FixedSchedule{
					Max: 31,
					At:  1,
				},
				Month: resolve.StepSchedule{
					Max:   12,
					Steps: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				},
				DayWeek: resolve.Everytime{},
			},
		},
		{
			name:  "Success/Simple/EveryMonthStringLiteral",
			input: "0 0 1 jan,Feb,MAR,aPR,maY,JuN,JUl,AUG,sep,oct,nov,dec *",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  0,
				},
				Hour: resolve.FixedSchedule{
					Max: 23,
					At:  0,
				},
				DayMonth: resolve.FixedSchedule{
					Max: 31,
					At:  1,
				},
				Month: resolve.StepSchedule{
					Max:   12,
					Steps: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				},
				DayWeek: resolve.Everytime{},
			},
		},
		{
			name:  "Success/Simple/SomeWeekdaysOnly",
			input: "* * * * 0,1,2",
			wants: Schedule{
				Sec:      resolve.FixedSchedule{Max: 59, At: 0},
				Min:      resolve.Everytime{},
				Hour:     resolve.Everytime{},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek: resolve.StepSchedule{
					Max:   7,
					Steps: []int{0, 1, 2},
				},
			},
		},
		{
			name:  "Success/Simple/EveryWeekdayNumericLiteralSundayFirst",
			input: "0 0 * * 0,1,2,3,4,5,6",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  0,
				},
				Hour: resolve.FixedSchedule{
					Max: 23,
					At:  0,
				},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek: resolve.StepSchedule{
					Max:   7,
					Steps: []int{0, 1, 2, 3, 4, 5, 6},
				},
			},
		},
		{
			name:  "Success/Simple/EveryWeekdayNumericLiteralSundayLast",
			input: "0 0 * * 1,2,3,4,5,6,7",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  0,
				},
				Hour: resolve.FixedSchedule{
					Max: 23,
					At:  0,
				},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek: resolve.StepSchedule{
					Max:   7,
					Steps: []int{0, 1, 2, 3, 4, 5, 6},
				},
			},
		},
		{
			name:  "Success/Simple/EveryWeekdayStringLiteral",
			input: "0 0 * * sun,Mon,TUE,wED,thU,FrI,sAt",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  0,
				},
				Hour: resolve.FixedSchedule{
					Max: 23,
					At:  0,
				},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek: resolve.StepSchedule{
					Max:   7,
					Steps: []int{0, 1, 2, 3, 4, 5, 6},
				},
			},
		},
		{
			name:  "Success/Overrides/reboot",
			input: "@reboot",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  0,
				},
				Hour:     resolve.Everytime{},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek:  resolve.Everytime{},
			},
		},
		{
			name:  "Success/Overrides/hourly",
			input: "@hourly",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  0,
				},
				Hour:     resolve.Everytime{},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek:  resolve.Everytime{},
			},
		},
		{
			name:  "Success/Overrides/daily",
			input: "@daily",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  0,
				},
				Hour: resolve.FixedSchedule{
					Max: 23,
					At:  0,
				},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek:  resolve.Everytime{},
			},
		},
		{
			name:  "Success/Overrides/weekly",
			input: "@weekly",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  0,
				},
				Hour: resolve.FixedSchedule{
					Max: 23,
					At:  0,
				},
				DayMonth: resolve.Everytime{},
				Month:    resolve.Everytime{},
				DayWeek: resolve.FixedSchedule{
					Max: 7,
					At:  0,
				},
			},
		},
		{
			name:  "Success/Overrides/monthly",
			input: "@monthly",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  0,
				},
				Hour: resolve.FixedSchedule{
					Max: 23,
					At:  0,
				},
				DayMonth: resolve.FixedSchedule{
					Max: 31,
					At:  1,
				},
				Month:   resolve.Everytime{},
				DayWeek: resolve.Everytime{},
			},
		},
		{
			name:  "Success/Overrides/annually",
			input: "@annually",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  0,
				},
				Hour: resolve.FixedSchedule{
					Max: 23,
					At:  0,
				},
				DayMonth: resolve.FixedSchedule{
					Max: 31,
					At:  1,
				},
				Month: resolve.FixedSchedule{
					Max: 12,
					At:  1,
				},
				DayWeek: resolve.Everytime{},
			},
		},
		{
			name:  "Success/Overrides/yearly",
			input: "@yearly",
			wants: Schedule{
				Sec: resolve.FixedSchedule{Max: 59, At: 0},
				Min: resolve.FixedSchedule{
					Max: 59,
					At:  0,
				},
				Hour: resolve.FixedSchedule{
					Max: 23,
					At:  0,
				},
				DayMonth: resolve.FixedSchedule{
					Max: 31,
					At:  1,
				},
				Month: resolve.FixedSchedule{
					Max: 12,
					At:  1,
				},
				DayWeek: resolve.Everytime{},
			},
		},
		{
			name:  "Fail/InvalidMonth",
			input: "* * * jan,jen,jin *",
			wants: Schedule{},
			err:   ErrInvalidAlphanum,
		},
		{
			name:  "Fail/TooManyTokens",
			input: "* * * * * * *",
			wants: Schedule{},
			err:   ErrInvalidNumNodes,
		},
		{
			name:  "Fail/InvalidOverride",
			input: "@take-a-guess",
			wants: Schedule{},
			err:   ErrInvalidFrequency,
		},
		{
			name:  "Fail/InvalidCharacter",
			input: "* * * * 0-!",
			wants: Schedule{},
			err:   ErrInvalidCharacter,
		},
		{
			name:  "Fail/InvalidAlphaNum",
			input: "* * * * 0//",
			wants: Schedule{},
			err:   ErrInvalidAlphanum,
		},
		{
			name:  "Fail/OverrideInvalidNumEdges",
			input: "@,",
			wants: Schedule{},
			err:   ErrInvalidNumEdges,
		},
		{
			name:  "Fail/InvalidNumNodes/TooFew",
			input: "-",
			wants: Schedule{},
			err:   ErrInvalidNumNodes,
		},
		{
			name:  "Fail/UnsupportedAlphanumeric",
			input: "*/A * * * *",
			wants: Schedule{},
			err:   ErrUnsupportedAlphanum,
		},
		{
			name:  "Fail/InvalidNodeType",
			input: "0/-3 * * * *",
			wants: Schedule{},
			err:   ErrInvalidNodeType,
		},
		{
			name:  "Fail/OutOfBounds",
			input: "0/64 * * * *",
			wants: Schedule{},
			err:   ErrOutOfBoundsAlphanum,
		},
		{
			name:  "Fail/TooManyWeekdays",
			input: "* * * * 0,1,2,3,4,5,6,7,8,9",
			wants: Schedule{},
			err:   ErrInvalidNumEdges,
		},
		{
			name:  "Fail/EmptyInput",
			input: "",
			wants: Schedule{},
			err:   ErrEmptyInput,
		},
		{
			name:  "Fail/SpecialCharacter",
			input: "İ",
			wants: Schedule{},
			err:   ErrInvalidCharacter,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			cron, err := Parse(testcase.input)

			is.True(t, errors.Is(err, testcase.err))
			require.Equal(t, testcase.wants, cron)
		})
	}
}

func FuzzParse(f *testing.F) {
	// load test strings as seeds
	f.Add("* * * * *")
	f.Add("0 * * * *")
	f.Add("50 * * * *")
	f.Add("*/3 * * * *")
	f.Add("0/3 * * * *")
	f.Add("0-3 * * * *")
	f.Add("0-3,5,7 * * * *")
	//nolint:lll // long string literals
	f.Add("0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59 * * * *")
	f.Add("0 0-23 * * *")
	f.Add("0 0 1-31 * *")
	f.Add("0 0 1 1,2,3,4,5,6,7,8,9,10,11,12 *")
	f.Add("0 0 1 jan,Feb,MAR,aPR,maY,JuN,JUl,AUG,sep,oct,nov,dec *")
	f.Add("* * * * 0,1,2")
	f.Add("0 0 * * 0,1,2,3,4,5,6")
	f.Add("0 0 * * 1,2,3,4,5,6,7")
	f.Add("0 0 * * sun,Mon,TUE,wED,thU,FrI,sAt")
	f.Add("@reboot")
	f.Add("@hourly")
	f.Add("@daily")
	f.Add("@weekly")
	f.Add("@monthly")
	f.Add("@annually")
	f.Add("@yearly")
	f.Add("* * * * * *")
	f.Add("@take-a-guess")
	f.Add("* * * * 0-!")
	f.Add("* * * * 0//")
	f.Add("@,")
	f.Add("-")
	f.Add("*/A * * * *")
	f.Add("0/-3 * * * *")
	f.Add("0/64 * * * *")
	f.Add("* * * * 0,1,2,3,4,5,6,7,8,9")

	f.Fuzz(func(t *testing.T, s string) {
		_, err := Parse(s)

		switch {
		case err == nil, errors.Is(err, ErrInvalidNumNodes), errors.Is(err, ErrInvalidNodeType),
			errors.Is(err, ErrInvalidNumEdges), errors.Is(err, ErrInvalidFrequency),
			errors.Is(err, ErrUnsupportedAlphanum), errors.Is(err, ErrOutOfBoundsAlphanum),
			errors.Is(err, ErrEmptyAlphanum), errors.Is(err, ErrInvalidAlphanum),
			errors.Is(err, ErrInvalidCharacter), errors.Is(err, ErrEmptyInput):
		default:
			t.Errorf("unexpected error: %v -- input: %q", err, s)
		}
	})
}
