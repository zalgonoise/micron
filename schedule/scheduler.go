package schedule

import (
	"context"
	"log/slog"
	"time"

	"github.com/zalgonoise/cfg"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/zalgonoise/micron/schedule/cronlex"
	"github.com/zalgonoise/micron/schedule/resolve"
)

const maxSec = 59

//nolint:gochecknoglobals // immutable instance of resolve.FixedSchedule for a fixed seconds schedule
var fixedSeconds = resolve.FixedSchedule{Max: maxSec, At: 0}

// Scheduler describes the capabilities of a cron job scheduler. Its sole responsibility is to provide
// the timestamp for the next job's execution, after calculating its frequency from its configuration.
//
// Scheduler exposes one method, Next, that takes in a context.Context and a time.Time. It is implied that the
// input time is the time.Now value, however it is open to any input that the caller desires to pass to it. The returned
// time.Time value must always be the following occurrence according to the schedule, in the context of the input time.
//
// Implementations of Next should take into consideration the cron specification; however the interface allows a custom
// approach to the scheduler, especially if added functionality is necessary (e.g. frequency overriding schedulers,
// dynamic frequencies, and pipeline-approaches where the frequency is evaluated after a certain check).
type Scheduler interface {
	// Next calculates and returns the following scheduled time, from the input time.Time.
	Next(ctx context.Context, now time.Time) time.Time
}

// Metrics describes the actions that register Scheduler-related metrics.
type Metrics interface {
	// IncSchedulerNextCalls increases the count of Next calls, by the Scheduler.
	IncSchedulerNextCalls()
}

// CronSchedule represents a basic implementation of a Scheduler, following the cron schedule specification.
//
// It is composed of a time.Location specifier, as well as a cronlex.Schedule definition.
type CronSchedule struct {
	// Loc will localize the times to a certain region or geolocation.
	Loc *time.Location
	// Schedule describes the schedule frequency definition, with different cron schedule elements.
	Schedule cronlex.Schedule

	logger  *slog.Logger
	metrics Metrics
	tracer  trace.Tracer
}

// Next calculates and returns the following scheduled time, from the input time.Time.
func (s *CronSchedule) Next(ctx context.Context, t time.Time) time.Time {
	ctx, span := s.tracer.Start(ctx, "Scheduler.Next")
	defer span.End()

	s.metrics.IncSchedulerNextCalls()

	year, month, day := t.Date()
	hour := t.Hour()
	minute := t.Minute()
	second := t.Second() + 1

	nextSecond := s.Schedule.Sec.Resolve(second)
	if s.Schedule.Sec == fixedSeconds {
		nextSecond++
	}

	nextMinute := s.Schedule.Min.Resolve(minute)
	nextHour := s.Schedule.Hour.Resolve(hour)
	nextDay := s.Schedule.DayMonth.Resolve(day)
	nextMonth := s.Schedule.Month.Resolve(int(month))

	// time.Date automatically normalizes overflowing values in the context of dates
	// (e.g. a result containing 27 hours is 3 AM on the next day)
	dayOfMonthTime := time.Date(
		year,
		month+time.Month(nextMonth),
		day+nextDay,
		hour+nextHour,
		minute+nextMinute,
		second+nextSecond,
		0, s.Loc,
	)

	// short circuit if unset or star '*'
	if _, ok := (s.Schedule.DayWeek).(resolve.Everytime); s.Schedule.DayWeek == nil || ok {
		span.SetAttributes(attribute.String("at", dayOfMonthTime.Format(time.RFC3339)))
		s.logger.InfoContext(ctx, "next job", slog.Time("at", dayOfMonthTime))

		return dayOfMonthTime
	}

	curWeekday := dayOfMonthTime.Weekday()
	nextWeekday := s.Schedule.DayWeek.Resolve(int(curWeekday))

	weekdayTime := time.Date(
		dayOfMonthTime.Year(),
		dayOfMonthTime.Month(),
		dayOfMonthTime.Day()+nextWeekday,
		dayOfMonthTime.Hour(),
		dayOfMonthTime.Minute(),
		dayOfMonthTime.Second(),
		0, s.Loc,
	)

	span.SetAttributes(attribute.String("at", weekdayTime.Format(time.RFC3339)))
	s.logger.InfoContext(ctx, "next job", slog.Time("at", weekdayTime))

	return weekdayTime
}

// New creates a Scheduler with the input cfg.Option(s), also returning an error if raised.
//
// Creating a Scheduler requires the caller to provide at least a cron string, using the WithSchedule option.
//
// If a time.Location is not specified with the WithLocation option, then time.Local is used.
func New(options ...cfg.Option[Config]) (Scheduler, error) {
	config := cfg.Set(defaultConfig(), options...)

	cron, err := newScheduler(config)
	if err != nil {
		return noOpScheduler{}, err
	}

	return cron, nil
}

func newScheduler(config Config) (Scheduler, error) {
	// parse cron string
	sched, err := cronlex.Parse(config.cron)
	if err != nil {
		return noOpScheduler{}, err
	}

	if config.loc == nil {
		config.loc = time.Local
	}

	return &CronSchedule{
		Loc:      config.loc,
		Schedule: sched,

		logger:  slog.New(config.handler),
		metrics: config.metrics,
		tracer:  config.tracer,
	}, nil
}

func NoOp() Scheduler {
	return noOpScheduler{}
}

type noOpScheduler struct{}

func (s noOpScheduler) Next(_ context.Context, _ time.Time) time.Time {
	return time.Time{}
}
