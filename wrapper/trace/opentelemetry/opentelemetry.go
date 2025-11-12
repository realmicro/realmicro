package opentelemetry

import (
	"context"
	"fmt"
	"strings"

	"github.com/realmicro/realmicro/metadata"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName = "github.com/realmicro/realmicro/wrapper/trace/opentelemetry"
)

// StartSpanFromContext returns a new span with the given operation name and options. If a span
// is found in the context, it will be used as the parent of the resulting span.
func StartSpanFromContext(ctx context.Context, tp trace.TracerProvider, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = make(metadata.Metadata)
	}
	propagator, carrier := otel.GetTextMapPropagator(), make(propagation.MapCarrier)
	for k, v := range md {
		for _, f := range propagator.Fields() {
			if strings.EqualFold(k, f) {
				fmt.Println(f, "->", v)
				carrier[f] = v
			}
		}
	}
	fmt.Println("1", carrier)
	ctx = propagator.Extract(ctx, carrier)
	spanCtx := trace.SpanContextFromContext(ctx)
	ctx = baggage.ContextWithBaggage(ctx, baggage.FromContext(ctx))

	var tracer trace.Tracer
	var span trace.Span
	if tp != nil {
		tracer = tp.Tracer(instrumentationName)
	} else {
		tracer = otel.Tracer(instrumentationName)
	}
	ctx, span = tracer.Start(trace.ContextWithRemoteSpanContext(ctx, spanCtx), name, opts...)

	fmt.Println("TraceID:", TraceIDFromContext(ctx))

	carrier = make(propagation.MapCarrier)
	propagator.Inject(ctx, carrier)
	fmt.Println("1", carrier)
	for k, v := range carrier {
		md.Set(strings.Title(k), v)
	}
	ctx = metadata.NewContext(ctx, md)

	return ctx, span
}

func SpanIDFromContext(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasSpanID() {
		return spanCtx.SpanID().String()
	}

	return ""
}

func TraceIDFromContext(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}

	return ""
}
