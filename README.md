## micron

### _This is my cron (micron); there are others like it but this one is mine. A cron-scheduler library in Go_

_______

### Concept

`cron` is a Go library that allows adding cron-like scheduler(s) to Go apps, compatible with 
[Unix's cron time/date strings](https://en.wikipedia.org/wiki/Cron), to execute actions within the context of the app.

By itself, `cron` is a fantastic tool released in the mid-70's, written in C, where the user defines a specification in 
a crontab, a file listing jobs with a reference of a time/date specification and a (Unix) command to execute.

Within Go, it should provide the same set of features as the original binary, but served in a library as a 
(blocking / background) pluggable service. This means full compatibility with cron strings for scheduling, support for 
multiple executable types or functions, and support for multiple job scheduling (with different time/date 
specifications). Additionally, the library extends this functionality to support definition of seconds' frequency in 
cron-strings, context support, error returns, and full observability coverage (with added support for metrics, logs and 
traces decorators).   

_______

### Motivation

In a work environment, we see cron many times, in different scenarios. From cron installations in bare-metal Linux 
servers to one-shot containers configured in Kubernetes deployments. Some use-cases are very simple, others complex, but
the point is that is a tool used for nearly 50 years at the time of writing.

But the original tool is a C binary that executes a Unix command. If we want to explore schedulers for Go applications 
(e.g. a script executed every # hours), this means that the app needs to be compiled as a binary, and then to configure
a cron-job to execute the app in a command.

While this is fine, it raises the question -- what if I want to include it _within_ the application? This should make a 
lot of sense to you if you're a fan of SQLite like me.

There were already two libraries with different implementations, in Go:
- [`robfig/cron`](https://github.com/robfig/cron) with 12k GitHub stars
- [`go-co-op/gocron`](https://github.com/go-co-op/gocron) with 4.3k GitHub stars

Don't get me wrong -- there is nothing inherently wrong about these implementations; I went through them carefully both 
for insight and to understand what could I explore differently. A very obvious change would be a more _"modern"_ 
approach including a newer Go version (as these required Go 1.12 and 1.16 respectively); which by itself includes
`log/slog` and all the other observability-related decorators that also leverage `context.Context`.

Another more obvious exploration path would be the parser logic, as I could use my 
[generic lexer](https://github.com/zalgonoise/lex) and [generic parser](https://github.com/zalgonoise/parse) in order to 
potentially improve it.

Lastly I could try to split the cron service's components to be more configurable even in future iterations, once I had
decided on the general API for the library. There was enough ground to explore and to give it a go. :)

A personal project that I have [for a Steam CLI app](https://github.com/zalgonoise/x/tree/master/steam) is currently 
using this cron library to regularly check for discounts in the Steam Store, for certain products on a certain 
frequency, as configured by the user.

_______

### Usage

Using `cron` is as layered and modular as you want it to be. This chapter describes how to use the library effectively.

#### Getting `cron`

You're able to fetch `cron` as a Go module by importing it in your project and running `go mod tidy`:

```go
package main

import (
	"fmt"
	"context"
	
	"github.com/zalgonoise/micron"
	"github.com/zalgonoise/micron/executor"
)

func main() {
	fn := func(context.Context) error {
		fmt.Println("done!")

		return nil
	}
	
	c, err := micron.New(micron.WithJob("my-job", "* * * * *", executor.Runnable(fn)))
	// ...
}
```
_______


#### Cron Runtime

The runtime is the component that will control (like the name implies) how the module runs -- that is, controlling the 
flow of job selection and execution. The runtime will allow cron to be executed as a goroutine, as its 
[`Runtime.Run`](./cron.go#L61) method has no returns, and errors are channeled via its [`Runtime.Err`](./cron.go#L78) 
method (which returns an error channel). The actual runtime of the cron is still managed with a `context.Context` that 
is provided when calling [`Runtime.Run`](./cron.go#L61) -- which can impose a cancellation or timeout strategy.

Just like the simple example above, creating a cron runtime starts with the 
[`cron.New` constructor function](./cron.go#L87).

This function only has [a variadic parameter for `cfg.Option[cron.Config]`](./cron.go#L87). This allows full modularity
on the way you build your cron runtime, to be as simple or as detailed as you want it to be -- provided that it complies 
with the minimum requirements to create one; to supply either:
- a [`selector.Selector`](./selector/selector.go#L37) 
- or, a (set of) [`executor.Runner`](./executor/executor.go#L41). This can be supplied as 
[`executor.Runnable`](./executor/executor.go#L54) as well.

```go
func New(options ...cfg.Option[*Config]) (Runtime, error)
```

Below is a table with all the options available for creating a cron runtime:

|                   Function                    |                                    Input Parameters                                    |                                                                                               Description                                                                                               |
|:---------------------------------------------:|:--------------------------------------------------------------------------------------:|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------:|
|    [`WithSelector`](./cron_config.go#L33)     |                 [`sel selector.Selector`](./selector/selector.go#L37)                  |                                                            Configures the  with the input [`selector.Selector`](./selector/selector.go#L37).                                                            |
|       [`WithJob`](./cron_config.go#L55)       | `id string`, `cron string`, [`runners ...executor.Runner`](./executor/executor.go#L41) | Adds a new [`executor.Executor`](./executor/executor.go#L85) to the [`Runtime`](./cron.go#L34) configuration from the input ID, cron string and set of [`executor.Runner`](./executor/executor.go#L41). |
| [`WithErrorBufferSize`](./cron_config.go#L85) |                                       `size int`                                       |                                   Defines the capacity of the error channel that the [`Runtime`](./cron.go#L34) exposes in its [`Runtime.Err`](./cron.go#L77) method.                                   |
|     [`WithMetrics`](./cron_config.go#L98)     |                     [`m cron.Metrics`](./cron_with_metrics.go#L10)                     |                                                               Configures the [`Runtime`](./cron.go#L34) with the input metrics registry.                                                                |
|     [`WithLogger`](./cron_config.go#L111)     |              [`logger *slog.Logger`](https://pkg.go.dev/log/slog#Logger)               |                                                                    Configures the [`Runtime`](./cron.go#L34) with the input logger.                                                                     |
|   [`WithLogHandler`](./cron_config.go#L124)   |             [`handler slog.Handler`](https://pkg.go.dev/log/slog#Handler)              |                                                           Configures the [`Runtime`](./cron.go#L34) with logging using the input log handler.                                                           |
|     [`WithTrace`](./cron_config.go#L137)      |   [`tracer trace.Tracer`](https://pkg.go.dev/go.opentelemetry.io/otel/trace#Tracer)    |                                                                 Configures the [`Runtime`](./cron.go#L34) with the input trace.Tracer.                                                                  |

The simplest possible cron runtime could be the result of a call to [`cron.New`](./cron.go#L87) with a single 
[`cron.WithJob`](./cron_config.go#L55) option. This creates all the components that a cron runtime needs with the most
minimal setup. It creates the underlying selector and executors.

Otherwise, the caller must use the [`WithSelector`](./cron_config.go#L33) option, and configure a 
[`selector.Selector`](./selector/selector.go#L37) manually when doing so. This results in more _boilerplate_ to get the
runtime set up, but provides deeper control on how the cron should be composed. The next chapter covers what is a
[`selector.Selector`](./selector/selector.go#L37) and how to create one.

_______

#### Cron Selector

This component is responsible for picking up the next job to execute, according to their schedule frequency. For this, 
the [`Selector`](./selector/selector.go#L37) is configured with a set of 
[`executor.Executor`](./executor/executor.go#L85), which in turn will expose a 
[`Next` method](./executor/executor.go#L93). With this information, the [`Selector`](./selector/selector.go#L37) cycles 
through its [`executor.Executor`](./executor/executor.go#L85) and picks up the next task(s) to run.

While the [`Selector`](./selector/selector.go#L37) calls the 
[`executor.Executor`'s `Exec` method](./executor/executor.go#L91), the actual waiting is within the
[`executor.Executor`'s](./executor/executor.go#L85) logic.

You're able to create a [`Selector`](./selector/selector.go#L37) through 
[its constructor function](./selector/selector.go#L143):

```go
func New(options ...cfg.Option[*Config]) (Selector, error)
```


Below is a table with all the options available for creating a cron job selector:


|                        Function                        |                                 Input Parameters                                  |                                                                                    Description                                                                                     |
|:------------------------------------------------------:|:---------------------------------------------------------------------------------:|:----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------:|
|  [`WithExecutors`](./selector/selector_config.go#L27)  |          [`executors ...executor.Executor`](./executor/executor.go#L85)           |                            Configures the [`Selector`](./selector/selector.go#L37) with the input [`executor.Executor`(s)](./executor/executor.go#L85).                            |
|    [`WithBlock`](./selector/selector_config.go#L62)    |                                                                                   |       Configures the [`Selector`](./selector/selector.go#L37) to block (wait) for the underlying [`executor.Executor`(s)](./executor/executor.go#L85) to complete the task.        |
|   [`WithTimeout`](./selector/selector_config.go#L75)   |                                `dur time.Duration`                                | Configures a (non-blocking) [`Selector`](./selector/selector.go#L37) to wait a certain duration before detaching of the executable task, before continuing to select the next one. |
|   [`WithMetrics`](./selector/selector_config.go#L88)   |          [`m selector.Metrics`](./selector/selector_with_metrics.go#L10)          |                                              Configures the [`Selector`](./selector/selector.go#L37) with the input metrics registry.                                              |
|   [`WithLogger`](./selector/selector_config.go#L101)   |            [`logger *slog.Logger`](https://pkg.go.dev/log/slog#Logger)            |                                                   Configures the [`Selector`](./selector/selector.go#L37) with the input logger.                                                   |
| [`WithLogHandler`](./selector/selector_config.go#L114) |           [`handler slog.Handler`](https://pkg.go.dev/log/slog#Handler)           |                                         Configures the [`Selector`](./selector/selector.go#L37) with logging using the input log handler.                                          |
|   [`WithTrace`](./selector/selector_config.go#L127)    | [`tracer trace.Tracer`](https://pkg.go.dev/go.opentelemetry.io/otel/trace#Tracer) |                                                Configures the [`Selector`](./selector/selector.go#L37) with the input trace.Tracer.                                                |

There is a catch to the [`Selector`](./selector/selector.go#L37), which is the actual job's execution time. While the 
[`Selector`](./selector/selector.go#L37) cycles through its [`executor.Executor`](./executor/executor.go#L85) list, it 
will execute the task while waiting for it to return with or without an error. This may cause issues when a given 
running task takes too long to complete when there are other, very frequent tasks. If there is a situation where the 
long-running task overlaps the execution time for another scheduled job, that job's execution is potentially skipped -- 
as the next task would only be picked up and waited for once the long-running one exits.

For this reason, there are two implementations of [`Selector`](./selector/selector.go#L37): 
- A blocking one, that waits for every job to run and return an error, accurately returning the correct outcome in its
`Next` call. This implementation is great for fast and snappy jobs, or less frequent / non-overlapping schedules and 
executions. There is less resource overhead to it, and the error returns are fully accurate with the actual outcome.
- A non-blocking one, that waits for a job to raise an error in a goroutine, with a set timeout (either set by the 
caller or a default one). This implementation is great if the jobs are too frequent and / or the tasks too long, when it
risks skipping executions due to stuck long-running tasks. It relies more heavily on having configured Observability at
least on the [`executor.Executor`](./executor/executor.go#L85) level to underline those events (which get detached from 
the [`Selector`](./selector/selector.go#L37) after timing out).

It is important to have a good idea of how your cron jobs will execute and how often, or simply ensure that there is at 
least logging enabled for the configured [`executor.Executor`(s)](./executor/executor.go#L85).
_______

#### Cron Executor

Like the name implies, the [`Executor`](./executor/executor.go#L85) is the component that actually executes the job, on 
its next scheduled time.

The [`Executor`](./executor/executor.go#L85) is composed of a [cron schedule](#cron-schedule) and a (set of) 
[`Runner`(s)](./executor/executor.go#L41). Also, the [`Executor`](./executor/executor.go#L85) stores an ID that is used 
to identify this particular job.

Having these 3 components in mind, it's natural that the [`Executor`](./executor/executor.go#L85) exposes three methods:
- [`Exec`](./executor/executor.go#L91) - runs the task when on its scheduled time.
- [`Next`](./executor/executor.go#L93) - calls the underlying 
[`schedule.Scheduler` Next method](./schedule/scheduler.go#L30).
- [`ID`](./executor/executor.go#L95) - returns the ID.

Considering that the [`Executor`](./executor/executor.go#L85) holds a specific 
[`schedule.Scheduler`](./schedule/scheduler.go#L28), it is also responsible for managing any waiting time before the 
job is executed. The strategy employed by the [`Executable`](./executor/executor.go#L99) type is one that calculates the
duration until the next job, and sleeps until that time is reached (instead of, for example, calling the
[`schedule.Scheduler` Next method](./schedule/scheduler.go#L30) every second).


To create an [`Executor`](./executor/executor.go#L85), you can use the [`New`](./executor/executor.go#L162) function 
that serves as a constructor. Note that the minimum requirements to creating an [`Executor`](./executor/executor.go#L85)
are to include both a [`schedule.Scheduler`](./schedule/scheduler.go#L28) with the 
[`WithScheduler`](./executor/executor_config.go#L62) option (or a cron string, using the 
[`WithSchedule`](./executor/executor_config.go#L79) option), 
and at least one [`Runner`](./executor/executor.go#L41) with the [`WithRunners`](./executor/executor_config.go#L29) 
option.

The [`Runner`](./executor/executor.go#L41) itself is an interface with a single method 
([`Run`](./executor/executor.go#L48)), that takes in a `context.Context` and returns an error. If your implementation is
so simple that you have it as a function and don't need to create a type for this 
[`Runner`](./executor/executor.go#L41), then you can use the [`Runnable` type](./executor/executor.go#L54) instead, 
which is a type alias to a function of the same signature, but implements [`Runner`](./executor/executor.go#L41) by 
calling itself as a function, in its [`Run`](./executor/executor.go#L61) method.

Creating an [`Executor`](./executor/executor.go#L85) is as easy as calling
[its constructor function](./executor/executor.go#L162):

```go
func New(id string, options ...cfg.Option[*Config]) (Executor, error)
```


Below is a table with all the options available for creating a cron job executor:



|                        Function                        |                                 Input Parameters                                  |                                                                     Description                                                                     |
|:------------------------------------------------------:|:---------------------------------------------------------------------------------:|:---------------------------------------------------------------------------------------------------------------------------------------------------:|
|   [`WithRunners`](./executor/executor_config.go#L29)   |                 [`runners ...Runner`](./executor/executor.go#L41)                 |                  Configures the [`Executor`](./executor/executor.go#L85) with the input [`Runner`(s)](./executor/executor.go#L41).                  |
|  [`WithScheduler`](./executor/executor_config.go#L62)  |             [`sched schedule.Scheduler`](./schedule/scheduler.go#L28)             |             Configures the [`Executor`](./executor/executor.go#L85) with the input [`schedule.Scheduler`](./schedule/scheduler.go#L28).             |
|  [`WithSchedule`](./executor/executor_config.go#L79)   |                                   `cron string`                                   |   Configures the [`Executor`](./executor/executor.go#L85) with a [`schedule.Scheduler`](./schedule/scheduler.go#L28) using the input cron string.   |
|  [`WithLocation`](./executor/executor_config.go#L97)   |                               `loc *time.Location`                                | Configures the [`Executor`](./executor/executor.go#L85) with a [`schedule.Scheduler`](./schedule/scheduler.go#L28) using the input `time.Location`. |
|  [`WithMetrics`](./executor/executor_config.go#L110)   |          [`m executor.Metrics`](./executor/executor_with_metrics.go#L11)          |                              Configures the [`Executor`](./executor/executor.go#L85) with the input metrics registry.                               |
|   [`WithLogger`](./executor/executor_config.go#L123)   |            [`logger *slog.Logger`](https://pkg.go.dev/log/slog#Logger)            |                                   Configures the [`Executor`](./executor/executor.go#L85) with the input logger.                                    |
| [`WithLogHandler`](./executor/executor_config.go#L136) |           [`handler slog.Handler`](https://pkg.go.dev/log/slog#Handler)           |                          Configures the [`Executor`](./executor/executor.go#L85) with logging using the input log handler.                          |
|   [`WithTrace`](./executor/executor_config.go#L149)    | [`tracer trace.Tracer`](https://pkg.go.dev/go.opentelemetry.io/otel/trace#Tracer) |                                Configures the [`Executor`](./executor/executor.go#L85) with the input trace.Tracer.                                 |


_______

#### Cron Scheduler

The [`Scheduler`](./schedule/scheduler.go#L28) is responsible for keeping schedule state (for example, derived from a 
cron string), and calculating the next job's execution time, with the context of the input timestamp. As such, the
[`Scheduler` interface only exposes one method, `Next`](./schedule/scheduler.go#L30) which is responsible of making such 
calculations.

The default implementation of [`Scheduler`](./schedule/scheduler.go#L28), [`CronSchedule`](./schedule/scheduler.go#L36), 
will be created from parsing a cron string, and is nothing but a data structure with a 
[`cronlex.Schedule`](./schedule/cronlex/process.go#L56) bounded to a `time.Location`.

While the [`CronSchedule`](./schedule/scheduler.go#L36) leverages different schedule elements with 
[`cronlex.Resolver` interfaces](./schedule/cronlex/process.go#L49), the [`Scheduler`](./schedule/scheduler.go#L28) uses 
these values as a difference from the input timestamp, to create a new date with a 
[`time.Date()`](https://pkg.go.dev/time#Date) call. This call merely adds the difference until the next job to the 
current time, on different elements of the timestamp, with added logic to calculate weekdays if set.

Fortunately, Go's `time` package is super solid and allows date overflows, calculating them accordingly. This makes the 
logic of the base implementation a total breeze, and simple enough to be pulled off as opposed to ticking every second, 
checking for new jobs.

You're able to create a [`Scheduler`](./schedule/scheduler.go#L28) by calling
[its constructor function](./schedule/scheduler.go#L98), with the mandatory minimum of supplying a cron string through 
its [`WithSchedule`](./schedule/scheduler_config.go#L23) option.

```go
func New(options ...cfg.Option[Config]) (Scheduler, error)
```

Below is a table with all the options available for creating a cron job scheduler:


|                        Function                        |                                 Input Parameters                                  |                                             Description                                             |
|:------------------------------------------------------:|:---------------------------------------------------------------------------------:|:---------------------------------------------------------------------------------------------------:|
|  [`WithSchedule`](./schedule/scheduler_config.go#L23)  |                                   `cron string`                                   |        Configures the [`Scheduler`](./schedule/scheduler.go#L28) with the input cron string.        |
|  [`WithLocation`](./schedule/scheduler_config.go#L38)  |                               `loc *time.Location`                                |      Configures the [`Scheduler`](./schedule/scheduler.go#L28) with the input `time.Location`.      |
|  [`WithMetrics`](./schedule/scheduler_config.go#L51)   |         [`m executor.Metrics`](./schedule/scheduler_with_metrics.go#L11)          |     Configures the [`Scheduler`](./schedule/scheduler.go#L28) with the input metrics registry.      |
|   [`WithLogger`](./schedule/scheduler_config.go#L64)   |            [`logger *slog.Logger`](https://pkg.go.dev/log/slog#Logger)            |          Configures the [`Scheduler`](./schedule/scheduler.go#L28) with the input logger.           |
| [`WithLogHandler`](./schedule/scheduler_config.go#L77) |           [`handler slog.Handler`](https://pkg.go.dev/log/slog#Handler)           | Configures the [`Scheduler`](./schedule/scheduler.go#L28) with logging using the input log handler. |
|   [`WithTrace`](./schedule/scheduler_config.go#L90)    | [`tracer trace.Tracer`](https://pkg.go.dev/go.opentelemetry.io/otel/trace#Tracer) |       Configures the [`Scheduler`](./schedule/scheduler.go#L28) with the input trace.Tracer.        |



_______

##### Cron Schedule

[`Schedule`](./schedule/cronlex/process.go#L56) is a data structure that holds a set of 
[`Resolver`s](./schedule/cronlex/process.go#L49), for each node, segment or unit of the schedule. This implementation 
focuses on the cron string specification with added support for a seconds definition (instead of the usual minutes, 
hours, days-of-the-month, months and weekdays). Each of these elements are 
[`Resolver`s](./schedule/cronlex/process.go#L49) interfaces that will calculate the difference until the target value(s)
is reached. More information on [`Resolver`s](./schedule/cronlex/process.go#L49) in 
[its own section](#schedule-resolver).

The [`Schedule`](./schedule/cronlex/process.go#L56) only holds the state of a parsed cron string, and its elements are 
made public so that implementations of [`Scheduler`](./schedule/scheduler.go#L28) can leverage it to calculate the 
job's next execution time.

To create a new [`Schedule`](./schedule/cronlex/process.go#L56) type, you're able to use the
[`Parse` function](./schedule/cronlex/process.go#L69), that consumes the input cron string, returning a 
[`Schedule`](./schedule/cronlex/process.go#L56) and an error if raised. More details on the actual parsing of the string
[in its own section](#schedule-parser).

Once created, the elements of [`Schedule`](./schedule/cronlex/process.go#L56) are accessed directly, where the caller 
can use the [`Resolver`](./schedule/cronlex/process.go#L49) interface:

```go
type Schedule struct {
	Sec      Resolver
	Min      Resolver
	Hour     Resolver
	DayMonth Resolver
	Month    Resolver
	DayWeek  Resolver
}
```
_______

##### Schedule Resolver

This component calculates the difference between the input value and the (set of) value(s) it is configured to be 
triggered on, also given a certain maximum value for that [`Resolver`'s](./schedule/cronlex/process.go#L49) range.

This difference is useful on a [`Scheduler`](./schedule/scheduler.go#L28), where 
[`time.Date()`](https://pkg.go.dev/time#Date) sums the input time to the difference until the next execution. As such, 
given each node in a [`Schedule`](./schedule/cronlex/process.go#L56), it is possible to derive the next time per node,
following this logic.

Take a minute as an example. It is an element that spans from 0 to 59; and consider all elements start at zero. A
minutes [`Resolver`](./schedule/cronlex/process.go#L49) is configured with a maximum value of 59, and this example is 
configured to trigger at minute 15. If the input time is 2:03 PM, the [`Resolver`](./schedule/cronlex/process.go#L49)
returns a difference of 12. When the [`Scheduler`](./schedule/scheduler.go#L28) gets this information, it adds up the 12
minutes to the input time, and returns the resulting datetime value.

This also makes the [`Resolver`](./schedule/cronlex/process.go#L49) very flexible, in tandem with the corresponding
[`Schedule`](./schedule/cronlex/process.go#L56) and [`Scheduler`](./schedule/scheduler.go#L28), providing customizable 
levels of precision. The seconds [`Resolver`](./schedule/cronlex/process.go#L49) in 
[`Schedule`](./schedule/cronlex/process.go#L56) is an example of this.

The implementations of [`Resolver`](./schedule/cronlex/process.go#L49) can be found in the 
[`resolve` package](./schedule/resolve/resolve.go). These are derived from parsing a cron string and assigned 
automatically when calling the [`Parse` function](./schedule/cronlex/process.go#L69).

To explore the different implementations, it's good to have in mind the modularity in cron schedules:
- it supports "every value" sequences when using a star (`*`)
- it supports range value sequences when using a dash (`-`, e.g. `0-15`)
- it supports step value sequences when using a slash (`/`, e.g. `*/3`)
- it supports separated range value elements when using commas (`,`, e.g. `0,20,25,30`) 
- it supports a combination of range values and steps (e.g. `0-15/3`)
- it supports overrides, for certain configurations (e.g. `@weekly`)

Having this in mind, this could technically be achieved with a single [`Resolver`](./schedule/cronlex/process.go#L49) 
type (that you will find for step values), but to maximize performance and reduce complexity where it is not needed, 
there are four types of [`Resolver`](./schedule/cronlex/process.go#L49):


###### `Everytime` Resolver

```go
type Everytime struct{}
```

The [`Everytime Resolver`](./schedule/resolve/resolve.go#L4) always returns zero, meaning that it resolves to the current time 
always (_trigger now_ type of action). This is the default [`Resolver`](./schedule/cronlex/process.go#L49) whenever a 
star (`*`) node is found, for example.


###### `FixedSchedule` Resolver

```go
type FixedSchedule struct {
	Max int
	At  int
}
```

The [`FixedSchedule Resolver`](./schedule/resolve/resolve.go#L13) resolves at a specific value within the entire range. 
It is especially useful when a node only contains a non-star (`*`) alphanumeric value, such as the cron string 
`0 0 * * *`, which has both minutes and hours as [`FixedSchedule Resolver`s](./schedule/resolve/resolve.go#L13).



###### `RangeSchedule` Resolver

```go
type RangeSchedule struct {
    Max  int
    From int
    To   int
}
```

The [`RangeSchedule Resolver`](./schedule/resolve/resolve.go#L25) resolves within a portion of the entire range. It is 
used when a range is provided without any gaps. This means that the schedule element does not contain any slashes (`/`)
or commas (`,`), and is only a range with a dash (`-`), such as the cron string `0-15 * * * *`, which contains a
[`RangeSchedule Resolver`](./schedule/resolve/resolve.go#L25) for its minutes' node.



###### `StepSchedule` Resolver

```go
type StepSchedule struct {
    Max   int
    Steps []int
}
```

The [`StepSchedule Resolver`](./schedule/resolve/resolve.go#L42) is the most complex of all -- that could potentially 
serve all the other implementations, however it can be the most resource-expensive of all 
[`Resolver`s](./schedule/cronlex/process.go#L49).

This implementation stores the defined values for the schedule and returns the difference of the closest value ahead of 
it. This involves scanning all the steps in the sequence as it requires looking into values that are less than the 
input, e.g., when the input is value 57, the maximum is 59, and the closest step is 0, with a difference of 3.

Doing this for complex cron setups can be more complex, and that is the major reason for having several other 
implementations. Regardless, if your cron string is one that involves many individual steps separated by commas (`,`), 
or contains a given frequency delimited by a slash (`/`), it surely will have an underlying 
[`StepSchedule Resolver`](./schedule/resolve/resolve.go#L42) to resolve that / those node(s). 

The [`StepSchedule Resolver`](./schedule/resolve/resolve.go#L42) is the only
[`Resolver`s](./schedule/cronlex/process.go#L49) which exposes a constructor, with the 
[`NewStepSchedule` function](./schedule/resolve/resolve.go#L77), that takes _from_ and _to_ values, 
a maximum, and a certain frequency (which should be 1 if no custom frequency is desired). Any further additions to 
the `Steps` in the [`StepSchedule Resolver`](./schedule/resolve/resolve.go#L42), should be added to the data structure,
separately:

```go
func NewStepSchedule(from, to, maximum, frequency int) StepSchedule
```

In the [Schedule Parser section](#schedule-parser), we explore how its processor will create the
[`Schedule`](./schedule/cronlex/process.go#L56) types following some rules, when working with the abstract syntax tree 
from parsing the cron string.
_______

##### Schedule Parser

To consume a cron string and make something meaningful of it, we must parse it. This step is most important to 
nail down accurately as it is the main source of user input within this library's logic. The executed jobs are of the 
caller's responsibility as they pass into it whatever they want. But having a correct understanding of the input 
schedule as well as calculating the times for the jobs' execution is fundamentally most important.

As mentioned before, this package exposes a [`Parse` function](./schedule/cronlex/process.go#L69) that consumes a cron 
string returning a [`Schedule`](./schedule/cronlex/process.go#L56) and an error:

```go
func Parse(cron string) (s Schedule, err error)
```

This is a process broken down in three phases that can be explored individually, having into consideration that the 
lexer and parser components work in tandem:
- A lexer, that consumes individual bytes from the cron string, emitting meaningful tokens about what they represent. 
The tokens also hold the value (bytes) that represent them. 
- A parser, which consumes the tokens as they are emitted from the lexer, and progressively builds an abstract syntax 
tree that is the cron schedule and its different nodes.
- A processor, which consumes the finalized abstract syntax tree created by the parser, validates its contents and 
creates the appropriate [`Schedule`](./schedule/cronlex/process.go#L56).

The implementations of the underlying lexer and parser logic are taken from Go's standard library, from its 
[`go/token`](https://pkg.go.dev/go/token) and [`text/template`](https://pkg.go.dev/text/template) packages. There is 
also [a fantastic talk from Rob Pike](https://www.youtube.com/watch?v=HxaD_trXwRE) not only describing how to 
write a lexer and parser as state machines in Go, but also carrying the viewer through the details of the 
`text/template` implementation. With this in mind, I had released the same general concept of a lexer and parser, but 
supporting generic types, allowing consumers of the library to write their parsers for anything, with any type 
(both for input, tokens and output). These implementations can be further explored below:
- [`zalgonoise/lex`](https://github.com/zalgonoise/lex)
- [`zalgonoise/parse`](https://github.com/zalgonoise/parse)

Taking this as a kick-off point, it's only required to implement 3 (major) functions for the lexer, parser and processor
phases of this pipeline. These functions are broken down with their own individual handler functions, as required.

Introducing the [`Token` type](./schedule/cronlex/token.go#L4), we can see how different symbols are set to trigger 
different tokens, while having a few general ones like [`TokenAlphaNum`](./schedule/cronlex/token.go#L9) for numbers and
characters, [`TokenEOF`](./schedule/cronlex/token.go#L7) when the end of the string is reached, etc.  

Starting with the lexer, it exposes a [`StateFunc` function](./schedule/cronlex/lexer.go#L19). This is the starting 
point for the state machine that consumes individual bytes. This function basically emits a token for a given character 
(if supported), except for alphanumeric bytes (that are collected and a single token emitted with them), and at the end 
of the string.

At the same time that the lexer is emitting these tokens, the parser's 
[`ParseFunc` function](./schedule/cronlex/parser.go#L18) is consuming them to build an abstract syntax tree. This tree 
has a root (top-level) node that is branched for how many nodes are there in a cron string -- say, `* * * * *` contains 
5 nodes, `* * * * * *` contains 6 nodes, and `@weekly` contains 1 node.

If these nodes are more complex than a single value, then the node will store the value while aggregating all following
(chained) values by their symbol as children. These symbol nodes must contain a value as child (a comma cannot be left 
by itself at the end of a node). Take the following node in a cron string: `0-15/3,20,25`. The node in the tree should
look like:

```
- (root)
  +- alphanum --> value: 0
    +- dash   --> value: 15
    +- slash  --> value: 3
    +- comma  --> value: 20
    +- comma  --> value: 25
```

Having created the cron's abstract syntax tree we arrive to the last phase, the 
[`ProcessFunc` function](./schedule/cronlex/process.go#L82). It starts off by validating the contents in the abstract 
syntax tree to ensure there are no unsupported values like greater than the maximum, etc.

Once ensured it is valid, the function checks how many nodes are children of the root node in the tree, with support for
3 types of lengths:
- 1 child node means this should be an exception (like `@weekly`).
- 5 child nodes represent a _classic_ cron string ranging from minutes to weekdays (e.g. `* * * * *`).
- 6 child nodes represent an _extended_ cron string supporting seconds to weekdays (e.g. `* * * * * *`).

Handling the exceptions is very simple as the function only switches on the supported values looking for a match. The 
switch statement is the fastest algorithm to perform this check. 

A _classic_ cron string with 5 nodes will still have a seconds [`Resolver`](./schedule/cronlex/process.go#L49) in its 
[`Schedule`](./schedule/cronlex/process.go#L56), by configuring it as a
[`FixedSchedule` type](./schedule/resolve/resolve.go#L13), triggering at value 0 with a maximum of 59 (for seconds). 

Generally, the Resolver types for each node from both _classic_ and _extended_ cron strings are built by checking
if it's a star (`*`) or alphanumeric node, creating the appropriate [`Resolver`](./schedule/cronlex/process.go#L49). 
Note that step values are sorted and compacted before being returned, for optimal efficiency when being used.

Lastly, considering the weekday support for 0 and 7 as Sundays, if the weekday 
[`Resolver`](./schedule/cronlex/process.go#L49) is a 
[`StepSchedule` type](./schedule/resolve/resolve.go#L42), it is normalized as a 0 value.


_______

### Example

A working example is the [Steam CLI app](https://github.com/zalgonoise/x/tree/master/steam) mentioned in the 
[Motivation](#motivation) section above. This application exposes some commands, one of them being 
[`monitor`](https://github.com/zalgonoise/x/blob/master/steam/cmd/steam/monitor/monitor.go). This file provides some 
insight on how the cron service is set up from a `main.go` / script-like approach.

You can also take a look 
[at its `runner.go` file](https://github.com/zalgonoise/x/blob/master/steam/cmd/steam/monitor/runner.go), that 
implements the [`executor.Runner`](./executor/executor.go#L41) interface.

_______

### Disclaimer

This is not a one-size-fits-all solution! Please take your time to evaluate it for your own needs with due diligence.
While having _a library for this and a library for that_ is pretty nice, it could potentially be only overhead hindering
the true potential of your app! Be sure to read the code that you are using to be a better judge if it is a good fit for
your project. With that in mind, I hope you enjoy this library. Feel free to contribute by filing either an issue or a
pull request.