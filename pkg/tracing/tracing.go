package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	// Tracer is the global tracer instance
	Tracer trace.Tracer
	// TracerProvider is the global tracer provider
	TracerProvider *sdktrace.TracerProvider
)

// Config holds the tracing configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	JaegerEndpoint string
	Enabled        bool
	SamplingRate   float64
}

// InitTracer initializes the global tracer with Jaeger exporter
func InitTracer(config Config) error {
	if !config.Enabled {
		// Use noop tracer if tracing is disabled
		Tracer = otel.Tracer(config.ServiceName)
		return nil
	}

	// Create Jaeger exporter
	exporter, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(config.JaegerEndpoint),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			attribute.String("environment", config.Environment),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Determine sampler based on sampling rate
	var sampler sdktrace.Sampler
	if config.SamplingRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if config.SamplingRate <= 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(config.SamplingRate)
	}

	// Create tracer provider
	TracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(TracerProvider)

	// Set global propagator for context propagation
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	// Create tracer
	Tracer = otel.Tracer(config.ServiceName)

	return nil
}

// Shutdown gracefully shuts down the tracer provider
func Shutdown(ctx context.Context) error {
	if TracerProvider == nil {
		return nil
	}

	// Create a context with timeout for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return TracerProvider.Shutdown(shutdownCtx)
}

// StartSpan starts a new span with the given name
func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if Tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return Tracer.Start(ctx, spanName, opts...)
}

// SpanFromContext returns the current span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SetAttributes sets attributes on the current span
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attrs...)
	}
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error, opts ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err, opts...)
	}
}

// SetStatus sets the status of the current span
func SetStatus(ctx context.Context, code trace.StatusCode, description string) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetStatus(code, description)
	}
}

// GetTraceID returns the trace ID from the current span
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span != nil && span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID returns the span ID from the current span
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span != nil && span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// Helper functions for common tracing patterns

// TraceFunction wraps a function with tracing
func TraceFunction(ctx context.Context, functionName string, fn func(context.Context) error) error {
	ctx, span := StartSpan(ctx, functionName)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		RecordError(ctx, err)
		SetStatus(ctx, trace.StatusCodeError, err.Error())
	}

	return err
}

// TraceFunctionWithResult wraps a function with tracing and returns a result
func TraceFunctionWithResult[T any](ctx context.Context, functionName string, fn func(context.Context) (T, error)) (T, error) {
	ctx, span := StartSpan(ctx, functionName)
	defer span.End()

	result, err := fn(ctx)
	if err != nil {
		RecordError(ctx, err)
		SetStatus(ctx, trace.StatusCodeError, err.Error())
	}

	return result, err
}

// AddStringAttribute adds a string attribute to the current span
func AddStringAttribute(ctx context.Context, key, value string) {
	SetAttributes(ctx, attribute.String(key, value))
}

// StringAttr creates a string attribute
func StringAttr(key, value string) attribute.KeyValue {
	return attribute.String(key, value)
}

// IntAttr creates an int attribute
func IntAttr(key string, value int) attribute.KeyValue {
	return attribute.Int(key, value)
}

// AddIntAttribute adds an int attribute to the current span
func AddIntAttribute(ctx context.Context, key string, value int) {
	SetAttributes(ctx, attribute.Int(key, value))
}

// AddInt64Attribute adds an int64 attribute to the current span
func AddInt64Attribute(ctx context.Context, key string, value int64) {
	SetAttributes(ctx, attribute.Int64(key, value))
}

// AddBoolAttribute adds a bool attribute to the current span
func AddBoolAttribute(ctx context.Context, key string, value bool) {
	SetAttributes(ctx, attribute.Bool(key, value))
}

// AddFloatAttribute adds a float64 attribute to the current span
func AddFloatAttribute(ctx context.Context, key string, value float64) {
	SetAttributes(ctx, attribute.Float64(key, value))
}

// InjectContext injects the tracing context into a carrier (for propagation)
func InjectContext(ctx context.Context, carrier propagation.TextMapCarrier) {
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// ExtractContext extracts the tracing context from a carrier
func ExtractContext(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}
