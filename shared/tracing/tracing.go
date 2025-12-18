package tracing

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/augno/api/shared/contracts"

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
)

const (
	envOTLPEndpoint     = "OTEL_EXPORTER_OTLP_ENDPOINT"
	envOTLPProtocol     = "OTEL_EXPORTER_OTLP_PROTOCOL"
	envServiceName      = "OTEL_SERVICE_NAME"
	envEnvironment      = "OTEL_ENVIRONMENT"
	envHeaders          = "OTEL_EXPORTER_OTLP_HEADERS"
	envInsecure         = "OTEL_EXPORTER_OTLP_INSECURE"
	envTracesSampler    = "OTEL_TRACES_SAMPLER"
	envTracesSamplerArg = "OTEL_TRACES_SAMPLER_ARG"
)

type Protocol string

const (
	ProtocolHTTP Protocol = "http"
	ProtocolGRPC Protocol = "grpc"
)

type Config struct {
	ServiceName string
	Environment string
	Endpoint    string
	Protocol    Protocol
	Insecure    bool
	Headers     map[string]string
	Sampler     string
	SamplerArg  string
}

func loadConfigFromEnv(getenv func(string) string) Config {
	serviceName := strings.TrimSpace(getenv(envServiceName))

	env := strings.TrimSpace(getenv(envEnvironment))
	if env == "" {
		env = "development"
	}

	endpoint := strings.TrimSpace(getenv(envOTLPEndpoint))

	protocol := Protocol(strings.ToLower(strings.TrimSpace(getenv(envOTLPProtocol))))
	if protocol == "" {
		protocol = ProtocolHTTP
	}

	headers := parseHeaders(getenv(envHeaders))
	sampler := strings.TrimSpace(getenv(envTracesSampler))
	samplerArg := strings.TrimSpace(getenv(envTracesSamplerArg))

	return Config{
		ServiceName: serviceName,
		Environment: env,
		Endpoint:    endpoint,
		Protocol:    protocol,
		Insecure:    isTruthy(getenv(envInsecure)),
		Headers:     headers,
		Sampler:     sampler,
		SamplerArg:  samplerArg,
	}
}

func InitTracer(cfg Config) (func(context.Context) error, error) {
	cfg.Protocol = cfg.Protocol.normalize()

	traceExporter, err := newExporter(cfg)
	if err != nil {
		return nil, err
	}

	traceProvider, err := newTraceProvider(cfg, traceExporter)
	if err != nil {
		return nil, err
	}
	otel.SetTracerProvider(traceProvider)

	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	return traceProvider.Shutdown, nil
}

func InitProvider(serviceName string) (func(context.Context) error, error) {
	cfg := loadConfigFromEnv(os.Getenv)
	if cfg.ServiceName == "" {
		cfg.ServiceName = serviceName
	}
	return InitTracer(cfg)
}

func GetTracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}

func newExporter(cfg Config) (sdktrace.SpanExporter, error) {
	endpoint := cfg.Endpoint

	switch cfg.Protocol {
	case ProtocolGRPC:
		return newGRPCExporter(endpoint, cfg)
	default:
		return newHTTPExporter(endpoint, cfg)
	}
}

func newHTTPExporter(endpoint string, cfg Config) (sdktrace.SpanExporter, error) {
	clientOpts := []otlptracehttp.Option{
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
	}

	if endpoint != "" {
		applyHTTPEndpoint(endpoint, &clientOpts)
	}

	if cfg.Insecure {
		clientOpts = append(clientOpts, otlptracehttp.WithInsecure())
	}

	if len(cfg.Headers) > 0 {
		clientOpts = append(clientOpts, otlptracehttp.WithHeaders(cfg.Headers))
	}

	client := otlptracehttp.NewClient(clientOpts...)
	return otlptrace.New(context.Background(), client)
}

func newGRPCExporter(endpoint string, cfg Config) (sdktrace.SpanExporter, error) {
	clientOpts := []otlptracegrpc.Option{}

	if endpoint != "" {
		clientOpts = append(clientOpts, otlptracegrpc.WithEndpoint(endpoint))
	}

	if cfg.Insecure {
		clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
	}

	if len(cfg.Headers) > 0 {
		clientOpts = append(clientOpts, otlptracegrpc.WithHeaders(cfg.Headers))
	}

	client := otlptracegrpc.NewClient(clientOpts...)
	return otlptrace.New(context.Background(), client)
}

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

func (p Protocol) normalize() Protocol {
	switch Protocol(strings.ToLower(string(p))) {
	case ProtocolGRPC:
		return ProtocolGRPC
	default:
		return ProtocolHTTP
	}
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTraceProvider(cfg Config, exporter sdktrace.SpanExporter) (*sdktrace.TracerProvider, error) {
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
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

func newSampler(cfg Config) sdktrace.Sampler {
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

type priorityRootSampler struct {
	base sdktrace.Sampler
}

func newPriorityRootSampler(base sdktrace.Sampler) sdktrace.Sampler {
	return &priorityRootSampler{base: base}
}

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

func (s *priorityRootSampler) Description() string {
	return "priorityRootSampler(" + s.base.Description() + ")"
}

func isPriorityRootSpan(name string) bool {
	if !strings.HasPrefix(name, "HTTP ") {
		return false
	}

	return !strings.Contains(name, " /healthz")
}

func isTruthy(val string) bool {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "1", "t", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func parseHeaders(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	headers := map[string]string{}

	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
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

func RecordControllerError(span trace.Span, err error) {
	if err == nil {
		return
	}

	if apiErr, ok := err.(*contracts.APIError); ok {
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
