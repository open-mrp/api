package db

import (
	"context"
	"testing"

	"github.com/XSAM/otelsql"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestSpanFilter(t *testing.T) {
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test")
	ctxWithSpan, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	tests := []struct {
		name     string
		ctx      context.Context
		method   otelsql.Method
		expected bool
	}{
		{
			name:     "valid context and normal method",
			ctx:      ctxWithSpan,
			method:   otelsql.MethodConnQuery,
			expected: true,
		},
		{
			name:     "valid context and reset session method",
			ctx:      ctxWithSpan,
			method:   otelsql.MethodConnResetSession,
			expected: false,
		},
		{
			name:     "invalid context and normal method",
			ctx:      context.Background(),
			method:   otelsql.MethodConnQuery,
			expected: false,
		},
		{
			name:     "invalid context and reset session method",
			ctx:      context.Background(),
			method:   otelsql.MethodConnResetSession,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := spanFilter(tt.ctx, tt.method, "", nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}
