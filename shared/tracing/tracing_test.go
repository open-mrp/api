package tracing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestRecordControllerErrorAPIErrorAddsApiErrorEvent(t *testing.T) {
	t.Parallel()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	tracer := tp.Tracer("test")

	_, span := tracer.Start(context.Background(), "service-span")
	apiErr := &apierror.APIError{
		Code:          apierror.ErrorCodeInvalidCredentials,
		Type:          apierror.ErrorTypeInvalidRequest,
		PublicMessage: "This refresh token has been revoked.",
	}

	RecordControllerError(span, apiErr)
	span.End()

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)

	recorded := spans[0]
	require.Equal(t, codes.Error, recorded.Status().Code)
	require.Equal(t, apiErr.PublicMessage, recorded.Status().Description)

	foundEvent := false
	for _, event := range recorded.Events() {
		if event.Name != "api.error" {
			continue
		}
		foundEvent = true
		attrMap := attrsToMap(event.Attributes)
		require.Equal(t, string(apiErr.Code), attrMap["error.code"])
		require.Equal(t, string(apiErr.Type), attrMap["error.type"])
		require.Equal(t, apiErr.PublicMessage, attrMap["error.public_message"])
	}
	require.True(t, foundEvent, "expected api.error event")
}

func TestRecordControllerErrorNilDoesNothing(t *testing.T) {
	t.Parallel()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	tracer := tp.Tracer("test")

	_, span := tracer.Start(context.Background(), "no-error")
	RecordControllerError(span, nil)
	span.End()

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, codes.Unset, spans[0].Status().Code)
	require.Empty(t, spans[0].Events())
}

func TestRecordControllerErrorNonAPIErrorRecordsException(t *testing.T) {
	t.Parallel()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	tracer := tp.Tracer("test")

	_, span := tracer.Start(context.Background(), "generic-error")
	err := errors.New("something bad happened")
	RecordControllerError(span, err)
	span.End()

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)

	recorded := spans[0]
	require.Equal(t, codes.Error, recorded.Status().Code)
	require.Equal(t, err.Error(), recorded.Status().Description)

	foundException := false
	for _, event := range recorded.Events() {
		if event.Name == "exception" {
			foundException = true
			break
		}
	}
	require.True(t, foundException, "expected exception event for generic error")
}

func attrsToMap(attrs []attribute.KeyValue) map[string]string {
	result := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		result[string(attr.Key)] = attr.Value.AsString()
	}
	return result
}

func TestWithNoTraceDisablesTracing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	require.True(t, appctx.ShouldTrace(ctx), "default context should allow tracing")

	noTraceCtx := appctx.WithNoTrace(ctx)
	require.False(t, appctx.ShouldTrace(noTraceCtx), "WithNoTrace context should not allow tracing")
}

func TestStartSpanReturnsNoopWhenNoTrace(t *testing.T) {
	t.Parallel()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	tracer := tp.Tracer("test")

	// With tracing enabled, span should be recorded
	ctx := context.Background()
	_, span := StartSpan(ctx, tracer, "traced-span")
	span.End()

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1, "should record span when tracing is enabled")
	require.Equal(t, "traced-span", spans[0].Name())

	// With tracing disabled, span should not be recorded
	noTraceCtx := appctx.WithNoTrace(ctx)
	_, noopSpan := StartSpan(noTraceCtx, tracer, "untraced-span")
	noopSpan.End()

	// Should still only have 1 span (the first one)
	spans = spanRecorder.Ended()
	require.Len(t, spans, 1, "should not record span when tracing is disabled")
}

func TestConfigWithDefaults(t *testing.T) {
	t.Parallel()
	fullEnv := map[string]string{
		envOTLPEndpoint:     "env-collector:4318",
		envOTLPProtocol:     "GRPC",
		envEnvironment:      "development",
		envHeaders:          "env-key=env-value",
		envInsecure:         "true",
		envTracesSampler:    "always_on",
		envTracesSamplerArg: "0.9",
	}

	tests := []struct {
		name     string
		cfg      *Config
		env      map[string]string
		expected *Config
	}{
		{
			name: "explicit fields beat environment",
			cfg: &Config{
				ServiceName: "explicit-service",
				Environment: constants.PlatformModeTest,
				Endpoint:    "explicit:4318",
				Protocol:    constants.ProtocolHTTP,
				Headers:     map[string]string{"explicit": "header"},
				Sampler:     "always_off",
				SamplerArg:  "0.25",
			},
			env: fullEnv,
			expected: &Config{
				ServiceName: "explicit-service",
				Environment: constants.PlatformModeTest,
				Endpoint:    "explicit:4318",
				Protocol:    constants.ProtocolHTTP,
				Insecure:    true,
				Headers:     map[string]string{"explicit": "header"},
				Sampler:     "always_off",
				SamplerArg:  "0.25",
			},
		},
		{
			name: "environment beats hardcoded defaults",
			cfg:  &Config{},
			env:  fullEnv,
			expected: &Config{
				ServiceName: "svc",
				Environment: constants.PlatformModeDevelopment,
				Endpoint:    "env-collector:4318",
				Protocol:    constants.ProtocolGRPC,
				Insecure:    true,
				Headers:     map[string]string{"env-key": "env-value"},
				Sampler:     "always_on",
				SamplerArg:  "0.9",
			},
		},
		{
			name: "empty environment falls back to hardcoded defaults",
			cfg:  &Config{},
			env:  nil,
			expected: &Config{
				ServiceName: "svc",
				Environment: constants.PlatformModeProduction,
				Protocol:    constants.ProtocolHTTP,
			},
		},
		{
			name: "nil receiver behaves as empty config",
			cfg:  nil,
			env:  nil,
			expected: &Config{
				ServiceName: "svc",
				Environment: constants.PlatformModeProduction,
				Protocol:    constants.ProtocolHTTP,
			},
		},
		{
			name: "insecure env is parsed with isTruthy",
			cfg:  &Config{},
			env:  map[string]string{envInsecure: " YES "},
			expected: &Config{
				ServiceName: "svc",
				Environment: constants.PlatformModeProduction,
				Protocol:    constants.ProtocolHTTP,
				Insecure:    true,
			},
		},
		{
			name: "non-truthy insecure env leaves TLS on",
			cfg:  &Config{},
			env:  map[string]string{envInsecure: "0"},
			expected: &Config{
				ServiceName: "svc",
				Environment: constants.PlatformModeProduction,
				Protocol:    constants.ProtocolHTTP,
			},
		},
		{
			// Insecure has no "unset" state, so an explicit false cannot override a truthy env var.
			name: "explicit insecure false cannot override truthy env",
			cfg:  &Config{Insecure: false},
			env:  map[string]string{envInsecure: "on"},
			expected: &Config{
				ServiceName: "svc",
				Environment: constants.PlatformModeProduction,
				Protocol:    constants.ProtocolHTTP,
				Insecure:    true,
			},
		},
		{
			name: "malformed header pairs are skipped",
			cfg:  &Config{},
			env:  map[string]string{envHeaders: " a=1 , b = 2 ,,no-equals,=novalue,emptyvalue= ,c=d=e "},
			expected: &Config{
				ServiceName: "svc",
				Environment: constants.PlatformModeProduction,
				Protocol:    constants.ProtocolHTTP,
				Headers:     map[string]string{"a": "1", "b": "2", "c": "d=e"},
			},
		},
		{
			name: "header env with no usable pairs yields nil",
			cfg:  &Config{},
			env:  map[string]string{envHeaders: " , =x, y= "},
			expected: &Config{
				ServiceName: "svc",
				Environment: constants.PlatformModeProduction,
				Protocol:    constants.ProtocolHTTP,
			},
		},
		{
			name: "explicit service name beats the InitProvider argument",
			cfg:  &Config{ServiceName: "cfg-service"},
			env:  nil,
			expected: &Config{
				ServiceName: "cfg-service",
				Environment: constants.PlatformModeProduction,
				Protocol:    constants.ProtocolHTTP,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, tt.cfg.withDefaults("svc", fakeGetenv(tt.env)))
		})
	}
}

func TestConfigWithDefaultsDoesNotMutateReceiver(t *testing.T) {
	t.Parallel()
	cfg := &Config{}
	resolved := cfg.withDefaults("svc", fakeGetenv(map[string]string{envOTLPEndpoint: "collector:4318"}))

	require.Equal(t, &Config{}, cfg)
	require.Equal(t, "collector:4318", resolved.Endpoint)
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cfg         *Config
		expectedErr string
	}{
		{
			name: "valid production http",
			cfg:  &Config{ServiceName: "svc", Environment: constants.PlatformModeProduction, Protocol: constants.ProtocolHTTP},
		},
		{
			name: "valid development grpc",
			cfg:  &Config{ServiceName: "svc", Environment: constants.PlatformModeDevelopment, Protocol: constants.ProtocolGRPC},
		},
		{
			name: "valid test environment",
			cfg:  &Config{ServiceName: "svc", Environment: constants.PlatformModeTest, Protocol: constants.ProtocolHTTP},
		},
		{
			name:        "nil config",
			cfg:         nil,
			expectedErr: errConfigNil.Error(),
		},
		{
			name:        "missing service name",
			cfg:         &Config{Environment: constants.PlatformModeProduction, Protocol: constants.ProtocolHTTP},
			expectedErr: "tracing: service name is required",
		},
		{
			// The OTEL spec's "http/protobuf" is not a member of constants.Protocol, so it aborts startup.
			name:        "otel spec protocol value is rejected",
			cfg:         &Config{ServiceName: "svc", Environment: constants.PlatformModeProduction, Protocol: constants.Protocol("http/protobuf")},
			expectedErr: "tracing: protocol is invalid: http/protobuf",
		},
		{
			// "staging" is not a member of constants.PlatformMode, so it aborts startup.
			name:        "staging environment is rejected",
			cfg:         &Config{ServiceName: "svc", Environment: constants.PlatformMode("staging"), Protocol: constants.ProtocolHTTP},
			expectedErr: "tracing: environment is invalid: staging",
		},
		{
			name:        "empty environment",
			cfg:         &Config{ServiceName: "svc", Protocol: constants.ProtocolHTTP},
			expectedErr: "tracing: environment is invalid: ",
		},
		{
			name:        "empty protocol",
			cfg:         &Config{ServiceName: "svc", Environment: constants.PlatformModeProduction},
			expectedErr: "tracing: protocol is invalid: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.validate()
			if tt.expectedErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.expectedErr)
		})
	}
}

func TestIsTruthy(t *testing.T) {
	t.Parallel()
	truthy := []string{"1", "t", "true", "TRUE", " yes ", "y", "on"}
	falsy := []string{"", "0", "f", "false", "no", "off", "enabled", "2"}

	for _, val := range truthy {
		require.True(t, isTruthy(val), val)
	}
	for _, val := range falsy {
		require.False(t, isTruthy(val), val)
	}
}

func TestNewSampler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		sampler     string
		samplerArg  string
		contains    string
		notContains string
	}{
		{
			name:     "default keeps priority root sampling at 10%",
			contains: "priorityRootSampler(TraceIDRatioBased{0.1})",
		},
		{
			name:       "default honors a valid sampler arg",
			samplerArg: "0.5",
			contains:   "priorityRootSampler(TraceIDRatioBased{0.5})",
		},
		{
			name:       "ratio above 1 falls back to the default",
			samplerArg: "1.5",
			contains:   "priorityRootSampler(TraceIDRatioBased{0.1})",
		},
		{
			name:       "negative ratio falls back to the default",
			samplerArg: "-0.5",
			contains:   "priorityRootSampler(TraceIDRatioBased{0.1})",
		},
		{
			name:       "unparseable ratio falls back to the default",
			samplerArg: "not-a-number",
			contains:   "priorityRootSampler(TraceIDRatioBased{0.1})",
		},
		{
			name:       "zero ratio is honored",
			samplerArg: "0",
			contains:   "priorityRootSampler(TraceIDRatioBased{0})",
		},
		{
			name:        "parentbased_traceidratio drops priority root sampling",
			sampler:     "  PARENTBASED_TRACEIDRATIO  ",
			samplerArg:  " 0.25 ",
			contains:    "root:TraceIDRatioBased{0.25}",
			notContains: "priorityRootSampler",
		},
		{
			name:        "traceidratio ignores parent decisions",
			sampler:     "traceidratio",
			samplerArg:  "0.3",
			contains:    "TraceIDRatioBased{0.3}",
			notContains: "ParentBased",
		},
		{
			name:     "always_on",
			sampler:  "always_on",
			contains: "AlwaysOnSampler",
		},
		{
			name:     "always_off",
			sampler:  "always_off",
			contains: "AlwaysOffSampler",
		},
		{
			// Unrecognized values fall through to plain parent-based ratio sampling and ignore SamplerArg.
			name:        "unrecognized sampler falls back to parent-based 10%",
			sampler:     "parentbased_always_on",
			samplerArg:  "0.9",
			contains:    "root:TraceIDRatioBased{0.1}",
			notContains: "priorityRootSampler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			description := newSampler(&Config{Sampler: tt.sampler, SamplerArg: tt.samplerArg}).Description()
			require.Contains(t, description, tt.contains)
			if tt.notContains != "" {
				require.NotContains(t, description, tt.notContains)
			}
		})
	}
}

func TestPriorityRootSamplerShouldSample(t *testing.T) {
	t.Parallel()
	traceID, err := trace.TraceIDFromHex("0af7651916cd43dd8448eb211c80319c")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("b7ad6b7169203331")
	require.NoError(t, err)

	parented := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	}))

	tests := []struct {
		name     string
		ctx      context.Context
		spanName string
		expected sdktrace.SamplingDecision
	}{
		{
			name:     "user-facing http root span is forced",
			ctx:      context.Background(),
			spanName: "HTTP GET /api/v1/x",
			expected: sdktrace.RecordAndSample,
		},
		{
			name:     "health check root span delegates",
			ctx:      context.Background(),
			spanName: "HTTP GET /healthz",
			expected: sdktrace.Drop,
		},
		{
			name:     "non-http root span delegates",
			ctx:      context.Background(),
			spanName: "rabbitmq.consume notification.cmd.send_email",
			expected: sdktrace.Drop,
		},
		{
			name:     "http prefix must be exact",
			ctx:      context.Background(),
			spanName: "HTTPS GET /api/v1/x",
			expected: sdktrace.Drop,
		},
		{
			name:     "child span delegates even for user-facing names",
			ctx:      parented,
			spanName: "HTTP GET /api/v1/x",
			expected: sdktrace.Drop,
		},
	}

	// A base that never samples makes delegation observable.
	sampler := newPriorityRootSampler(sdktrace.NeverSample())
	require.Equal(t, "priorityRootSampler(AlwaysOffSampler)", sampler.Description())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := sampler.ShouldSample(sdktrace.SamplingParameters{
				ParentContext: tt.ctx,
				TraceID:       traceID,
				Name:          tt.spanName,
			})
			require.Equal(t, tt.expected, result.Decision)
		})
	}
}

func TestApplyHTTPEndpoint(t *testing.T) {
	t.Parallel()
	// otlptracehttp options wrap an internal config that is unreachable from outside that package, so the option count is the only observable signal that each branch fired.
	tests := []struct {
		name            string
		endpoint        string
		expectedOptions int
	}{
		{
			name:            "bare host and port",
			endpoint:        "collector:4318",
			expectedOptions: 1,
		},
		{
			name:            "http scheme adds insecure",
			endpoint:        "http://collector:4318",
			expectedOptions: 2,
		},
		{
			name:            "https scheme stays secure",
			endpoint:        "https://collector:4318",
			expectedOptions: 1,
		},
		{
			name:            "signal path is applied",
			endpoint:        "http://collector:4318/v1/traces",
			expectedOptions: 3,
		},
		{
			name:            "base path is applied",
			endpoint:        "https://collector.example.com/otlp",
			expectedOptions: 2,
		},
		{
			name:            "root path adds no path option",
			endpoint:        "https://collector.example.com/",
			expectedOptions: 1,
		},
		{
			name:            "unparseable url yields no options",
			endpoint:        "http://[::1",
			expectedOptions: 0,
		},
		{
			name:            "empty host yields no endpoint option",
			endpoint:        "http:///v1/traces",
			expectedOptions: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var opts []otlptracehttp.Option
			applyHTTPEndpoint(tt.endpoint, &opts)
			require.Len(t, opts, tt.expectedOptions)
		})
	}
}

func TestInitProviderRejectsInvalidConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		env         map[string]string
		serviceName string
		expectedErr string
	}{
		{
			name:        "unknown environment",
			env:         map[string]string{envEnvironment: "staging"},
			serviceName: "svc",
			expectedErr: "tracing: environment is invalid: staging",
		},
		{
			name:        "unknown protocol",
			env:         map[string]string{envOTLPProtocol: "http/protobuf"},
			serviceName: "svc",
			expectedErr: "tracing: protocol is invalid: http/protobuf",
		},
		{
			name:        "missing service name",
			env:         nil,
			serviceName: "",
			expectedErr: "tracing: service name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			shutdown, err := InitProvider(context.Background(), tt.serviceName, fakeGetenv(tt.env))
			require.EqualError(t, err, tt.expectedErr)
			require.Nil(t, shutdown)
		})
	}
}

func TestInitProviderSucceedsWithUnreachableCollector(t *testing.T) {
	// Not parallel: mutates the global otel tracer provider and propagator.
	origTP := otel.GetTracerProvider()
	origProp := otel.GetTextMapPropagator()
	defer func() {
		otel.SetTracerProvider(origTP)
		otel.SetTextMapPropagator(origProp)
	}()

	shutdown, err := InitProvider(context.Background(), "svc", fakeGetenv(map[string]string{
		envEnvironment:   string(constants.PlatformModeTest),
		envOTLPProtocol:  string(constants.ProtocolHTTP),
		envOTLPEndpoint:  "http://127.0.0.1:1/v1/traces",
		envTracesSampler: "always_off",
	}))
	require.NoError(t, err, "the exporter connects lazily, so a down collector must not block startup")
	require.NotNil(t, shutdown)

	require.ElementsMatch(t, []string{"traceparent", "tracestate", "baggage"}, otel.GetTextMapPropagator().Fields())
	require.NotEqual(t, origTP, otel.GetTracerProvider())

	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()
	require.NoError(t, shutdown(ctx))
}

func TestDeferShutdownUsesFreshContext(t *testing.T) {
	t.Parallel()
	// The context must be inspected while the flush is in flight; DeferShutdown cancels it on return.
	var err error
	var deadline time.Time
	var hasDeadline bool
	// A shutdown error is logged and swallowed, never propagated into the defer chain.
	DeferShutdown(func(ctx context.Context) error {
		err = ctx.Err()
		deadline, hasDeadline = ctx.Deadline()
		return errors.New("flush failed")
	})()

	require.NoError(t, err, "shutdown must not inherit an already-cancelled startup context")
	require.True(t, hasDeadline, "shutdown needs its own deadline to bound the final flush")
	require.WithinDuration(t, time.Now().Add(defaultShutdownTimeout), deadline, 2*time.Second)
}

func TestRecordControllerErrorRecordsOptionalAPIErrorFields(t *testing.T) {
	t.Parallel()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	_, span := tp.Tracer("test").Start(context.Background(), "service-span")

	apiErr := &apierror.APIError{
		Code:            apierror.ErrorCodeParameterInvalid,
		Type:            apierror.ErrorTypeInvalidRequest,
		PublicMessage:   "Invalid parameter.",
		InternalMessage: "quantity must be positive",
		Param:           "quantity",
		DocURL:          "https://docs.example.com/errors/parameter_invalid",
		Internal:        errors.New("underlying failure"),
	}
	RecordControllerError(span, apiErr)
	span.End()

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	require.Len(t, spans[0].Events(), 1)

	attrs := attrsToMap(spans[0].Events()[0].Attributes)
	require.Equal(t, "quantity", attrs["error.param"])
	require.Equal(t, apiErr.DocURL, attrs["error.doc_url"])
	require.Equal(t, "underlying failure", attrs["error.internal_error"])
	require.Equal(t, apiErr.InternalMessage, attrs["error.internal_message"])
}

func fakeGetenv(env map[string]string) func(string) string {
	return func(key string) string {
		return env[key]
	}
}
