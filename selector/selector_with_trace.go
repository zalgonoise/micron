package selector

import (
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// AddTraces replaces the input Selector's tracer with a different one, using the input trace.Tracer.
//
// If the input tracer is nil, the Selector's tracer will be set to be a no-op.
//
// If the input Selector is nil or a no-op Selector, a no-op Selector is returned.
//
// If the input Selector is a valid Selector, then its tracer is replaced with the input one.
//
// Otherwise, the Selector is returned as-is.
func AddTraces(s Selector, tracer trace.Tracer) Selector {
	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("selector's no-op tracer")
	}

	if s == nil || s == NoOp() {
		return NoOp()
	}

	switch sel := s.(type) {
	case selector:
		sel.tracer = tracer

		return sel
	case blockingSelector:
		sel.tracer = tracer

		return sel
	default:
		return s
	}
}
