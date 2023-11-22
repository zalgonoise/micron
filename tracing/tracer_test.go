package tracing

import (
	"context"
	"testing"

	"github.com/zalgonoise/x/is"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestTracer(t *testing.T) {
	for _, testcase := range []struct {
		name  string
		setup func(*testing.T) ShutdownFunc
	}{
		{
			name: "Success/WithInit",
			setup: func(t *testing.T) ShutdownFunc {
				done, err := Init(NoopExporter())
				is.Empty(t, err)

				return done
			},
		},
		{
			name: "Success/NoInit",
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			if testcase.setup != nil {
				done := testcase.setup(t)

				//nolint:errcheck // testing: we are sure noopTracer returns a nil error
				defer done(context.Background())
			}

			tracer := Tracer()
			is.True(t, tracer != nil)
		})
	}
}

func TestInit(t *testing.T) {
	for _, testcase := range []struct {
		name     string
		exporter sdktrace.SpanExporter
	}{
		{
			name:     "Success",
			exporter: NoopExporter(),
		},
		{
			name: "Success/NilExporter",
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			ctx := context.Background()
			done, err := Init(testcase.exporter)
			//nolint:errcheck // testing: we are sure noopTracer returns a nil error
			defer done(ctx)
			is.Empty(t, err)
		})
	}
}
