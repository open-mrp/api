package db

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/XSAM/otelsql"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestSpanFilter(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result := spanFilter(tt.ctx, tt.method, "", nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigWithDefaults(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver yields production defaults", func(t *testing.T) {
		t.Parallel()
		var cfg *Config
		got := cfg.WithDefaults()

		assert.Equal(t, "", got.DBURI)
		assert.True(t, got.TracingEnabled)
		assert.Equal(t, defaultConnectionMaxLifetime, got.ConnectionMaxLifetime)
		assert.Equal(t, defaultConnectionMaxIdleTime, got.ConnectionMaxIdleTime)
		assert.Equal(t, defaultMaxOpenConnections, got.MaxOpenConnections)
		assert.Equal(t, defaultMaxIdleConnections, got.MaxIdleConnections)
	})

	t.Run("zero fields are replaced and the URI is carried over", func(t *testing.T) {
		t.Parallel()
		got := (&Config{DBURI: "user:pass@tcp(host:3306)/app"}).WithDefaults()

		assert.Equal(t, "user:pass@tcp(host:3306)/app", got.DBURI)
		assert.Equal(t, defaultMaxOpenConnections, got.MaxOpenConnections)
	})

	t.Run("explicit values are kept", func(t *testing.T) {
		t.Parallel()
		got := (&Config{
			DBURI:                 "user:pass@tcp(host:3306)/app",
			ConnectionMaxLifetime: time.Minute,
			ConnectionMaxIdleTime: 2 * time.Minute,
			MaxOpenConnections:    7,
			MaxIdleConnections:    3,
		}).WithDefaults()

		assert.Equal(t, time.Minute, got.ConnectionMaxLifetime)
		assert.Equal(t, 2*time.Minute, got.ConnectionMaxIdleTime)
		assert.Equal(t, 7, got.MaxOpenConnections)
		assert.Equal(t, 3, got.MaxIdleConnections)
	})

	// Documented as unset-means-on: false is indistinguishable from the zero value, so tracing
	// cannot be turned off through this config.
	t.Run("tracing cannot be disabled", func(t *testing.T) {
		t.Parallel()
		assert.True(t, (&Config{TracingEnabled: false}).WithDefaults().TracingEnabled)
	})

	t.Run("the caller's config is not mutated", func(t *testing.T) {
		t.Parallel()
		original := &Config{DBURI: "user:pass@tcp(host:3306)/app"}
		got := original.WithDefaults()

		assert.NotSame(t, original, got)
		assert.Zero(t, original.MaxOpenConnections)
		assert.False(t, original.TracingEnabled)
		assert.Zero(t, original.ConnectionMaxLifetime)
	})
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	t.Run("nil config is rejected", func(t *testing.T) {
		t.Parallel()
		var cfg *Config
		assert.Error(t, cfg.validate())
	})

	t.Run("empty URI is rejected", func(t *testing.T) {
		t.Parallel()
		assert.ErrorContains(t, (&Config{}).validate(), "database URI is empty")
	})

	t.Run("a URI is all that is required", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, (&Config{DBURI: "user:pass@tcp(host:3306)/app"}).validate())
	})
}

// The pool is built before anything is dialed, so a misconfigured service fails at startup
// with the reason rather than on its first query.
func TestNewDbPool_RejectsBadConfigBeforeConnecting(t *testing.T) {
	t.Parallel()

	t.Run("nil config", func(t *testing.T) {
		t.Parallel()
		pool, err := NewDbPool(nil)
		assert.Nil(t, pool)
		assert.ErrorContains(t, err, "database URI is empty")
	})

	t.Run("empty URI", func(t *testing.T) {
		t.Parallel()
		pool, err := NewDbPool(&Config{MaxOpenConnections: 5})
		assert.Nil(t, pool)
		assert.ErrorContains(t, err, "database URI is empty")
	})

	t.Run("unparseable DSN", func(t *testing.T) {
		t.Parallel()
		pool, err := NewDbPool(&Config{DBURI: "not-a-dsn"})
		assert.Nil(t, pool)
		assert.ErrorContains(t, err, "failed to open database connection pool")
	})
}

// The appended parameters decide parseTime, loc and time_zone for every service, so a DSN the
// driver then rejects takes the whole service down at startup. Dialing is intercepted, so the
// two failures are distinguishable without a database: the driver parses the DSN before it
// dials, meaning "failed to open" is a rejected DSN and "failed to ping" is a DSN the driver
// accepted.
func TestNewDbPool_AppendsDSNParameters(t *testing.T) {
	t.Parallel()

	errBlockedDial := errors.New("dial intercepted")
	mysql.RegisterDialContext("dsntest", func(context.Context, string) (net.Conn, error) {
		return nil, errBlockedDial
	})

	t.Run("a DSN with no query string stays parseable", func(t *testing.T) {
		t.Parallel()
		_, err := NewDbPool(&Config{DBURI: "user:pass@dsntest(host:3306)/app"})
		assert.ErrorIs(t, err, errBlockedDial)
		assert.ErrorContains(t, err, "failed to ping database")
	})

	// The separator: an existing query string has to be extended with "&".
	t.Run("a DSN that already has parameters stays parseable", func(t *testing.T) {
		t.Parallel()
		_, err := NewDbPool(&Config{DBURI: "user:pass@dsntest(host:3306)/app?readTimeout=1s"})
		assert.ErrorIs(t, err, errBlockedDial)
		assert.ErrorContains(t, err, "failed to ping database")
	})

	// What the previous case would look like if the parameters were joined with "?" instead:
	// the driver folds the second "?" into the preceding value and rejects the DSN.
	t.Run("a second question mark would be rejected by the driver", func(t *testing.T) {
		t.Parallel()
		_, err := NewDbPool(&Config{DBURI: "user:pass@dsntest(host:3306)/app?readTimeout=1s?parseTime=true"})
		assert.ErrorContains(t, err, "failed to open database connection pool")
		assert.NotErrorIs(t, err, errBlockedDial, "a rejected DSN never reaches the dialer")
	})

	// Nothing is appended when the DSN already sets every parameter, so the DSN is handed to the
	// driver unchanged.
	t.Run("a fully parameterized DSN stays parseable", func(t *testing.T) {
		t.Parallel()
		_, err := NewDbPool(&Config{
			DBURI: "user:pass@dsntest(host:3306)/app?parseTime=true&loc=UTC&time_zone=%27%2B00%3A00%27&interpolateParams=false",
		})
		assert.ErrorIs(t, err, errBlockedDial)
		assert.ErrorContains(t, err, "failed to ping database")
	})

	t.Run("the caller's URI is not mutated", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{DBURI: "user:pass@dsntest(host:3306)/app"}
		_, _ = NewDbPool(cfg)
		assert.Equal(t, "user:pass@dsntest(host:3306)/app", cfg.DBURI)
	})
}
