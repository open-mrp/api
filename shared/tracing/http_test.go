package tracing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestWrapGatewayHandler_Filter(t *testing.T) {
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
