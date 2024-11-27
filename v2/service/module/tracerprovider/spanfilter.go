package tracerprovider

import (
	"go.opentelemetry.io/otel/sdk/trace"
)

type SpanFilter struct {
	trace.SpanProcessor

	// Ignore is the map of dropped span names.
	Ignore map[string]struct{}
}

func (f SpanFilter) OnEnd(s trace.ReadOnlySpan) {
	if _, ok := f.Ignore[s.Name()]; ok {
		return
	}
	f.SpanProcessor.OnEnd(s)
}
