package tracing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-mrp/api/shared/appctx"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestWrapGatewayHandler_Filter(t *testing.T) {
	// Not parallel: mutates global otel tracer provider; parallel tests would race.
	tests := []struct {
		name          string
		method        string
		path          string
		expectedSpans int
	}{
		{
			name:          "GET request is traced",
			method:        http.MethodGet,
			path:          "/api/v1/resource",
			expectedSpans: 1,
		},
		{
			name:          "POST request is traced",
			method:        http.MethodPost,
			path:          "/api/v1/resource",
			expectedSpans: 1,
		},
		{
			name:          "OPTIONS request is NOT traced",
			method:        http.MethodOptions,
			path:          "/api/v1/resource",
			expectedSpans: 0,
		},
		{
			name:          "healthz request is NOT traced",
			method:        http.MethodGet,
			path:          "/healthz",
			expectedSpans: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup span recorder
			spanRecorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))

			// Temporarily set the global tracer provider
			oldTP := otel.GetTracerProvider()
			otel.SetTracerProvider(tp)
			defer otel.SetTracerProvider(oldTP)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			instrumentedHandler := WrapGatewayHandler(handler)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			instrumentedHandler.ServeHTTP(rr, req)

			spans := spanRecorder.Ended()
			require.Len(t, spans, tt.expectedSpans)
		})
	}
}

func TestWrapGatewayHandlerSpanNameUsesRoutePattern(t *testing.T) {
	// Not parallel: mutates global otel tracer provider.
	origTP := otel.GetTracerProvider()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	handler := WrapGatewayHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/usr_123?expand=org", nil)
	req = req.WithContext(appctx.WithRoutePattern(req.Context(), "/api/v1/users/{id}"))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "HTTP GET /api/v1/users/{id}", spans[0].Name(), "path parameters must not leak into span names")

	attrs := spanAttrs(spans[0])
	require.Equal(t, "/api/v1/users/{id}", attrs[attrHTTPRoute].AsString())
	require.Equal(t, "/api/v1/users/usr_123", attrs[attrHTTPPath].AsString())
	require.Equal(t, "expand=org", attrs[attrHTTPQuery].AsString())
}

func TestWrapGatewayHandlerSpanNameFallsBackToPath(t *testing.T) {
	// Not parallel: mutates global otel tracer provider.
	origTP := otel.GetTracerProvider()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	handler := WrapGatewayHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/unmatched", nil))

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "HTTP GET /unmatched", spans[0].Name())
}

func TestWrapGatewayHandlerRecordsStatusCode(t *testing.T) {
	// Not parallel: mutates global otel tracer provider.
	tests := []struct {
		name    string
		write   func(w http.ResponseWriter)
		expects int
	}{
		{
			name:    "explicit 500",
			write:   func(w http.ResponseWriter) { w.WriteHeader(http.StatusInternalServerError) },
			expects: http.StatusInternalServerError,
		},
		{
			name:    "explicit 400",
			write:   func(w http.ResponseWriter) { w.WriteHeader(http.StatusBadRequest) },
			expects: http.StatusBadRequest,
		},
		{
			name:    "implicit 200 via Write",
			write:   func(w http.ResponseWriter) { _, _ = w.Write([]byte("ok")) },
			expects: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origTP := otel.GetTracerProvider()
			spanRecorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
			otel.SetTracerProvider(tp)
			defer otel.SetTracerProvider(origTP)

			handler := WrapGatewayHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.write(w)
			}))
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/resource", nil))

			spans := spanRecorder.Ended()
			require.Len(t, spans, 1)
			require.Equal(t, int64(tt.expects), spanAttrs(spans[0])[attrHTTPStatusCode].AsInt64())
		})
	}
}

func TestWrapGatewayHandlerPropagatesPanic(t *testing.T) {
	// Not parallel: mutates global otel tracer provider.
	origTP := otel.GetTracerProvider()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(tracetest.NewSpanRecorder()))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	handler := WrapGatewayHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	require.PanicsWithValue(t, "boom", func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/resource", nil))
	}, "panics must reach the recovery middleware, not be swallowed here")
}

func TestWrapGatewayHandlerExtractsIncomingTraceContext(t *testing.T) {
	// Not parallel: mutates global otel tracer provider and propagator.
	origTP := otel.GetTracerProvider()
	origProp := otel.GetTextMapPropagator()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(newPropagator())
	defer func() {
		otel.SetTracerProvider(origTP)
		otel.SetTextMapPropagator(origProp)
	}()

	handler := WrapGatewayHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/resource", nil)
	req.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "0af7651916cd43dd8448eb211c80319c", spans[0].SpanContext().TraceID().String())
	require.Equal(t, "b7ad6b7169203331", spans[0].Parent().SpanID().String())
}

func spanAttrs(span sdktrace.ReadOnlySpan) map[string]attribute.Value {
	result := make(map[string]attribute.Value)
	for _, attr := range span.Attributes() {
		result[string(attr.Key)] = attr.Value
	}
	return result
}
