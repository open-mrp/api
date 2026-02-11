// Package tracing provides OpenTelemetry tracing initialization and helpers for all
// microservices. It configures OTLP exporters (HTTP or gRPC), sampling strategies,
// and context propagation. Each service calls InitProvider at startup to install a
// global TracerProvider, then uses GetTracer to obtain per-package tracers.
//
// Configuration is resolved from a combination of struct fields and OTEL_* environment
// variables, with struct fields taking precedence. See Config for details.
package tracing

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/env"
	apierror "github.com/augno/api/shared/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	// errConfigNil is returned when a nil Config reaches a method that cannot
	// apply defaults (e.g. newExporter, newTraceProvider).
	errConfigNil = fmt.Errorf("tracing config is nil")
)

// noopTracer is a process-wide no-op tracer used by [StartSpan] when tracing is
// suppressed. It produces valid but non-recording spans that satisfy the trace.Span
// interface without exporting anything.
var noopTracer = noop.NewTracerProvider().Tracer("")

// StartSpan creates a new span if tracing is enabled on the context, otherwise
// delegates to the no-op tracer so the caller receives a valid (but non-recording)
// span. This lets callers write straight-line code without checking [ShouldTrace]
// themselves — the returned span is always safe to use.
func StartSpan(ctx context.Context, tracer trace.Tracer, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !appctx.ShouldTrace(ctx) {
		return noopTracer.Start(ctx, name, opts...)
	}
	return tracer.Start(ctx, name, opts...)
}

// Environment variable names used as fallbacks when Config fields are zero-valued.
// These follow the OpenTelemetry SDK environment variable specification
// (https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/).
const (
	envOTLPEndpoint     = "OTEL_EXPORTER_OTLP_ENDPOINT"
	envOTLPProtocol     = "OTEL_EXPORTER_OTLP_PROTOCOL"
	envEnvironment      = "OTEL_ENVIRONMENT"
	envHeaders          = "OTEL_EXPORTER_OTLP_HEADERS"
	envInsecure         = "OTEL_EXPORTER_OTLP_INSECURE"
	envTracesSampler    = "OTEL_TRACES_SAMPLER"
	envTracesSamplerArg = "OTEL_TRACES_SAMPLER_ARG"
)

// Config holds all settings needed to initialize an OTLP trace exporter and sampler.
// Each field can be set explicitly or left zero to fall back to the corresponding
// OTEL_* environment variable (resolved in withDefaults). Explicit values always
// take precedence over environment variables.
type Config struct {
	// ServiceName (required) identifies this service in trace backends (e.g. "auth-service").
	// Falls back to the serviceName argument passed to InitProvider.
	ServiceName string

	// Environment (optional; default: OTEL_ENVIRONMENT or "production") is the deployment
	// environment (e.g. production, staging).
	Environment constants.PlatformMode

	// Endpoint (optional; default: OTEL_EXPORTER_OTLP_ENDPOINT) is the OTLP collector URL
	// (e.g. "localhost:4318"). For HTTP, a full URL with scheme is parsed to extract host,
	// path, and insecure mode automatically.
	Endpoint string

	// Protocol (optional; default: OTEL_EXPORTER_OTLP_PROTOCOL or "http") selects the OTLP
	// transport: "grpc" or "http/protobuf".
	Protocol constants.Protocol

	// Insecure (optional; default: OTEL_EXPORTER_OTLP_INSECURE) disables TLS for the
	// exporter connection.
	Insecure bool

	// Headers (optional; default: OTEL_EXPORTER_OTLP_HEADERS) are sent with every export
	// request (e.g. auth tokens). Env var format: comma-separated "key=value" pairs.
	Headers map[string]string

	// Sampler (optional; default: OTEL_TRACES_SAMPLER) selects the sampling strategy.
	// Supported values: "parentbased_traceidratio", "traceidratio", "always_on", "always_off".
	// When empty, a parent-based sampler with priority root sampling is used.
	Sampler string

	// SamplerArg (optional; default: OTEL_TRACES_SAMPLER_ARG or "0.1") is passed to
	// ratio-based samplers as the sampling probability (0.0-1.0).
	SamplerArg string
}

// defaultHeaders returns the first map if it has entries, otherwise falls back to
// the second. Used by withDefaults to prefer explicit Config.Headers over
// environment-parsed headers.
func defaultHeaders(a, b map[string]string) map[string]string {
	if len(a) > 0 {
		return a
	}
	return b
}

// withDefaults returns a new Config with zero-value fields populated from
// environment variables (via getenv). Resolution order for each field:
//
//  1. Explicit value set on the receiver.
//  2. Value from the corresponding OTEL_* environment variable.
//  3. Hardcoded default (e.g. HTTP protocol, production environment, 10% sampling).
//
// A nil receiver is treated as an entirely empty Config. The original struct is
// never mutated; a fresh copy is always returned.
func (c *Config) withDefaults(serviceName string, getenv func(string) string) *Config {
	if c == nil {
		c = &Config{}
	}

	endpoint := env.GetEnv(envOTLPEndpoint, getenv)
	headers := parseHeaders(getenv(envHeaders))
	sampler := env.GetEnv(envTracesSampler, getenv)
	samplerArg := env.GetEnv(envTracesSamplerArg, getenv)
	protocol := strings.ToLower(env.GetEnv(envOTLPProtocol, getenv))
	environment := env.GetEnv(envEnvironment, getenv)
	insecure := isTruthy(getenv(envInsecure))

	return &Config{
		ServiceName: cmp.Or(c.ServiceName, serviceName),
		Environment: cmp.Or(cmp.Or(c.Environment, constants.PlatformMode(environment)), constants.PlatformModeProduction),
		Endpoint:    cmp.Or(c.Endpoint, endpoint),
		Protocol:    cmp.Or(cmp.Or(c.Protocol, constants.Protocol(protocol)), constants.ProtocolHTTP),
		Insecure:    cmp.Or(c.Insecure, insecure),
		Headers:     defaultHeaders(c.Headers, headers),
		Sampler:     cmp.Or(c.Sampler, sampler),
		SamplerArg:  cmp.Or(c.SamplerArg, samplerArg),
	}
}

// validate checks that required fields are present and that enum-typed fields
// contain recognized values. Must be called after withDefaults, which fills in
// defaults — calling validate on a raw zero-value Config will fail on ServiceName.
func (c *Config) validate() error {
	if c == nil {
		return errConfigNil
	}
	if c.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}
	if !c.Environment.IsValid() {
		return fmt.Errorf("environment is invalid: %s", c.Environment)
	}
	if !c.Protocol.IsValid() {
		return fmt.Errorf("protocol is invalid: %s", c.Protocol)
	}
	return nil
}

// InitProvider sets up the global OpenTelemetry TracerProvider and TextMapPropagator
// for a service. It creates an OTLP exporter, configures sampling, and registers
// everything with the otel global. Call this once at service startup. The provided
// context is used for exporter and resource initialization so that startup can be
// cancelled promptly (e.g. on SIGTERM).
//
// The returned function flushes pending spans and shuts down the provider; call it
// during graceful shutdown (typically deferred in main).
func InitProvider(ctx context.Context, serviceName string, getenv func(string) string) (func(context.Context) error, error) {
	cfg := new(Config).withDefaults(serviceName, getenv)
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	traceExporter, err := cfg.newExporter(ctx)
	if err != nil {
		return nil, err
	}

	traceProvider, err := newTraceProvider(ctx, cfg, traceExporter)
	if err != nil {
		return nil, err
	}
	otel.SetTracerProvider(traceProvider)

	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	return traceProvider.Shutdown, nil
}

// GetTracer returns a named tracer from the global TracerProvider. Each package
// should call this once at init time with its package path to get a tracer for
// creating spans (e.g. tracing.GetTracer("auth-service/internal/service")).
func GetTracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}

// WorkerTracerProvider wraps an independent TracerProvider for background workers
// (e.g. outbox processors, scheduled jobs). Using a separate provider causes worker
// traces to appear under a distinct service name ("{service}-worker") in the trace
// backend, keeping them visually separated from request-driven traces.
type WorkerTracerProvider struct {
	provider *sdktrace.TracerProvider
}

// NewWorkerTracerProvider creates a new TracerProvider configured identically to the
// global provider but registered under "{baseServiceName}-worker". The provided context
// is used for exporter and resource initialization. The caller is responsible for
// calling Shutdown on the returned provider during graceful shutdown.
func NewWorkerTracerProvider(ctx context.Context, baseServiceName string, getenv func(string) string) (*WorkerTracerProvider, error) {
	cfg := new(Config).withDefaults(baseServiceName, getenv)
	cfg.ServiceName = baseServiceName + "-worker"

	exporter, err := cfg.newExporter(ctx)
	if err != nil {
		return nil, err
	}

	provider, err := newTraceProvider(ctx, cfg, exporter)
	if err != nil {
		return nil, err
	}

	return &WorkerTracerProvider{provider: provider}, nil
}

// Tracer returns a named tracer from the worker's independent TracerProvider. Spans
// created with this tracer appear under the "{service}-worker" service name in the
// trace backend, visually separating background work from request-driven traces.
func (w *WorkerTracerProvider) Tracer(name string) trace.Tracer {
	return w.provider.Tracer(name)
}

// Shutdown flushes any pending spans and releases the worker provider's resources.
// The provided context controls the flush deadline; if it expires, pending spans
// may be lost. Typically called via [WorkerTracerProvider.DeferClose].
func (w *WorkerTracerProvider) Shutdown(ctx context.Context) error {
	return w.provider.Shutdown(ctx)
}

// DeferClose returns a closure that shuts down the worker provider with a 5-second
// timeout and logs any error. The double-parenthesis invocation pattern defers the
// returned closure so it runs at function exit:
//
//	defer workerTracer.DeferClose()()
func (w *WorkerTracerProvider) DeferClose() func() {
	return DeferShutdown(w.Shutdown)
}

// defaultShutdownTimeout is the maximum time DeferShutdown waits for the shutdown
// function to complete before abandoning the flush.
const defaultShutdownTimeout = 5 * time.Second

// DeferShutdown wraps a shutdown function (typically TracerProvider.Shutdown) with a
// 5-second timeout context and slog error logging. It returns a closure suitable for
// use with defer:
//
//	tracerShutdown, err := tracing.InitProvider(ctx, serviceName, getenv)
//	// ...
//	defer tracing.DeferShutdown(tracerShutdown)()
//
// This intentionally creates a fresh context from context.Background() rather than
// reusing the caller's startup context. Shutdown runs because the startup context was
// cancelled (e.g. by SIGTERM), so reusing it would cause the span flush to return
// context.Canceled immediately without sending data. Unlike local cleanup (Close,
// Stop), the trace provider's Shutdown performs network I/O to flush buffered spans
// to the OTLP collector, so it needs a live context with its own deadline.
func DeferShutdown(shutdown func(context.Context) error) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			slog.Error("failed to shutdown tracer", "error", err)
		}
	}
}

// newExporter creates an OTLP span exporter based on the configured protocol.
// For gRPC it uses otlptracegrpc; for HTTP (the default) it uses otlptracehttp
// with gzip compression. The endpoint, TLS, and header settings from the config
// are applied to the underlying client.
func (c *Config) newExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	if c == nil {
		return nil, errConfigNil
	}

	switch c.Protocol {
	case constants.ProtocolGRPC:
		var clientOpts []otlptracegrpc.Option

		if c.Endpoint != "" {
			clientOpts = append(clientOpts, otlptracegrpc.WithEndpoint(c.Endpoint))
		}

		if c.Insecure {
			clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
		}

		if len(c.Headers) > 0 {
			clientOpts = append(clientOpts, otlptracegrpc.WithHeaders(c.Headers))
		}

		client := otlptracegrpc.NewClient(clientOpts...)
		return otlptrace.New(ctx, client)

	default:
		clientOpts := []otlptracehttp.Option{
			otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
		}

		if c.Endpoint != "" {
			applyHTTPEndpoint(c.Endpoint, &clientOpts)
		}

		if c.Insecure {
			clientOpts = append(clientOpts, otlptracehttp.WithInsecure())
		}

		if len(c.Headers) > 0 {
			clientOpts = append(clientOpts, otlptracehttp.WithHeaders(c.Headers))
		}

		client := otlptracehttp.NewClient(clientOpts...)
		return otlptrace.New(ctx, client)
	}
}

// applyHTTPEndpoint parses the endpoint string and adds the appropriate options.
// If the endpoint contains a scheme (e.g. "http://localhost:4318/v1/traces"), it is
// decomposed into host, URL path, and insecure flag. Plain "host:port" strings are
// passed through directly as the endpoint.
func applyHTTPEndpoint(endpoint string, opts *[]otlptracehttp.Option) {
	if strings.Contains(endpoint, "://") {
		parsed, err := url.Parse(endpoint)
		if err != nil {
			return
		}

		if parsed.Host != "" {
			*opts = append(*opts, otlptracehttp.WithEndpoint(parsed.Host))
		}

		if parsed.Scheme == "http" {
			*opts = append(*opts, otlptracehttp.WithInsecure())
		}

		if trimmedPath := strings.TrimPrefix(parsed.Path, "/"); trimmedPath != "" {
			*opts = append(*opts, otlptracehttp.WithURLPath("/"+trimmedPath))
		}
		return
	}

	*opts = append(*opts, otlptracehttp.WithEndpoint(endpoint))
}

// newPropagator creates a composite propagator that injects and extracts both
// W3C TraceContext (traceparent/tracestate headers) and Baggage headers. This
// ensures trace context is carried across HTTP and gRPC service boundaries.
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// newTraceProvider builds an SDK TracerProvider that batches spans to the given
// exporter. It tags every span with the service name and deployment environment
// as OpenTelemetry semantic convention resource attributes, and applies the
// configured sampler to control trace volume.
func newTraceProvider(ctx context.Context, cfg *Config, exporter sdktrace.SpanExporter) (*sdktrace.TracerProvider, error) {
	if cfg == nil {
		return nil, errConfigNil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.DeploymentEnvironmentKey.String(string(cfg.Environment)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	sampler := newSampler(cfg)

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	return traceProvider, nil
}

// newSampler builds a trace sampler from the config's Sampler/SamplerArg fields.
//
// Supported sampler values (case-insensitive):
//   - "" (empty):                 parent-based with priorityRootSampler (default)
//   - "parentbased_traceidratio": standard parent-based ratio sampling
//   - "traceidratio":             ratio sampling ignoring parent decisions
//   - "always_on":                sample every trace
//   - "always_off":               drop every trace
//
// For ratio-based samplers, SamplerArg is parsed as a float64 between 0.0 and 1.0.
// If missing or invalid, it defaults to 0.1 (10% of traces).
func newSampler(cfg *Config) sdktrace.Sampler {
	const defaultRatio = 0.1

	sampler := strings.ToLower(strings.TrimSpace(cfg.Sampler))
	arg := strings.TrimSpace(cfg.SamplerArg)

	parseRatio := func(defaultVal float64) float64 {
		if arg == "" {
			return defaultVal
		}
		if v, err := strconv.ParseFloat(arg, 64); err == nil && v >= 0 && v <= 1 {
			return v
		}
		return defaultVal
	}

	switch sampler {
	case "":
		ratio := parseRatio(defaultRatio)
		baseRoot := sdktrace.TraceIDRatioBased(ratio)
		return sdktrace.ParentBased(newPriorityRootSampler(baseRoot))
	case "parentbased_traceidratio":
		ratio := parseRatio(defaultRatio)
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	case "traceidratio":
		ratio := parseRatio(defaultRatio)
		return sdktrace.TraceIDRatioBased(ratio)
	case "always_on":
		return sdktrace.AlwaysSample()
	case "always_off":
		return sdktrace.NeverSample()
	default:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(defaultRatio))
	}
}

// priorityRootSampler wraps a base sampler and overrides its decision for
// "priority" root spans — HTTP request spans that aren't health checks. These
// spans are always sampled regardless of the ratio, ensuring real user-facing
// requests are never dropped while still allowing the base sampler to thin out
// high-volume internal traffic. Only applies to root spans (no parent); child
// spans inherit their parent's sampling decision via the outer ParentBased sampler.
type priorityRootSampler struct {
	base sdktrace.Sampler
}

// newPriorityRootSampler returns a sampler that always records user-facing HTTP
// root spans and delegates everything else to base.
func newPriorityRootSampler(base sdktrace.Sampler) sdktrace.Sampler {
	return &priorityRootSampler{base: base}
}

// ShouldSample forces RecordAndSample for priority root spans (user-facing HTTP
// requests excluding /healthz). All other spans — child spans, non-HTTP root spans,
// and health checks — are delegated to the base sampler.
func (s *priorityRootSampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	parentSpanCtx := trace.SpanContextFromContext(parameters.ParentContext)
	if !parentSpanCtx.IsValid() {
		if isPriorityRootSpan(parameters.Name) {
			return sdktrace.SamplingResult{
				Decision:   sdktrace.RecordAndSample,
				Tracestate: parentSpanCtx.TraceState(),
			}
		}
	}

	return s.base.ShouldSample(parameters)
}

// Description returns a human-readable sampler description for debug output,
// including the wrapped base sampler's description.
func (s *priorityRootSampler) Description() string {
	return "priorityRootSampler(" + s.base.Description() + ")"
}

// isPriorityRootSpan returns true for HTTP request spans that represent real
// user traffic (i.e. "HTTP GET /api/..." but not "HTTP GET /healthz").
func isPriorityRootSpan(name string) bool {
	if !strings.HasPrefix(name, "HTTP ") {
		return false
	}

	return !strings.Contains(name, " /healthz")
}

// isTruthy returns true for common boolean-truthy string representations
// ("1", "t", "true", "yes", "y", "on"). Used to parse the OTEL_EXPORTER_OTLP_INSECURE
// environment variable.
func isTruthy(val string) bool {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "1", "t", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

// parseHeaders parses a comma-separated list of "key=value" pairs into a map,
// matching the OTEL_EXPORTER_OTLP_HEADERS format. Whitespace around keys, values,
// and pairs is trimmed. Malformed entries (missing "=" or empty key/value) are
// silently skipped. Returns nil if no valid headers are found.
func parseHeaders(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	headers := map[string]string{}

	pairs := strings.SplitSeq(raw, ",")
	for pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		if key == "" || value == "" {
			continue
		}
		headers[key] = value
	}

	if len(headers) == 0 {
		return nil
	}

	return headers
}

// RecordControllerError annotates a span with error information from an API controller.
// For structured APIErrors it records a rich "api.error" event with code, type, message,
// and optional fields (param, doc URL, internal error). For all other errors it falls
// back to the standard span.RecordError. In both cases the span status is set to Error.
func RecordControllerError(span trace.Span, err error) {
	if err == nil {
		return
	}

	if apiErr, ok := err.(*apierror.APIError); ok {
		attrs := []attribute.KeyValue{
			attribute.String("error.code", string(apiErr.Code)),
			attribute.String("error.type", string(apiErr.Type)),
			attribute.String("error.public_message", apiErr.PublicMessage),
		}
		if apiErr.InternalMessage != "" {
			attrs = append(attrs, attribute.String("error.internal_message", apiErr.InternalMessage))
		}
		if apiErr.Param != "" {
			attrs = append(attrs, attribute.String("error.param", apiErr.Param))
		}
		if apiErr.DocURL != "" {
			attrs = append(attrs, attribute.String("error.doc_url", apiErr.DocURL))
		}
		if apiErr.Internal != nil {
			attrs = append(attrs, attribute.String("error.internal_error", apiErr.Internal.Error()))
		}

		span.AddEvent("api.error", trace.WithAttributes(attrs...))
		span.SetStatus(codes.Error, apiErr.PublicMessage)
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
