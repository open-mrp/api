package db

import (
	"cmp"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/XSAM/otelsql"
	_ "github.com/go-sql-driver/mysql"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultConnectionMaxLifetime = time.Minute * 30
	defaultConnectionMaxIdleTime = time.Minute * 10
	defaultMaxOpenConnections    = 50
	defaultMaxIdleConnections    = 50
	defaultTracingEnabled        = true
	defaultWarmConnections       = 4
	defaultWarmInterval          = time.Minute
)

// Config represents the configuration for the database connection pool.
type Config struct {
	// DBURI (required) is the database connection URI.
	DBURI string

	// TracingEnabled (optional; default: true) specifies whether tracing is enabled. The zero value (false) is treated as "unset" by WithDefaults and replaced with true, so tracing cannot be disabled via this config.
	TracingEnabled bool

	// ConnectionMaxLifetime (optional; default: 30m) is the maximum lifetime of a connection.
	ConnectionMaxLifetime time.Duration

	// ConnectionMaxIdleTime (optional; default: 10m) is the maximum idle time of a connection.
	ConnectionMaxIdleTime time.Duration

	// MaxOpenConnections (optional; default: 50) is the maximum number of open connections.
	MaxOpenConnections int

	// MaxIdleConnections (optional; default: 50) is the maximum number of idle connections.
	MaxIdleConnections int

	// WarmConnections (optional; default: 4; negative disables) is how many connections are kept open off the request path, so a request after a quiet spell does not pay the TLS connect.
	WarmConnections int

	// WarmInterval (optional; default: 1m) is how often the warm connections are pinged; keep it under the 350s AWS NAT idle timeout.
	WarmInterval time.Duration
}

// WithDefaults returns a new Config with all zero-value optional fields replaced by production defaults. It is safe to call on a nil receiver. The original Config is not mutated; a copy is always returned.
func (c *Config) WithDefaults() *Config {
	if c == nil {
		c = &Config{}
	}

	return &Config{
		DBURI:                 c.DBURI,
		TracingEnabled:        cmp.Or(c.TracingEnabled, defaultTracingEnabled),
		ConnectionMaxLifetime: cmp.Or(c.ConnectionMaxLifetime, defaultConnectionMaxLifetime),
		ConnectionMaxIdleTime: cmp.Or(c.ConnectionMaxIdleTime, defaultConnectionMaxIdleTime),
		MaxOpenConnections:    cmp.Or(c.MaxOpenConnections, defaultMaxOpenConnections),
		MaxIdleConnections:    cmp.Or(c.MaxIdleConnections, defaultMaxIdleConnections),
		WarmConnections:       cmp.Or(c.WarmConnections, defaultWarmConnections),
		WarmInterval:          cmp.Or(c.WarmInterval, defaultWarmInterval),
	}
}

// validate checks that the Config fields form a coherent database connection pool configuration.
func (c *Config) validate() error {
	if c == nil {
		return fmt.Errorf("db: config is nil")
	}
	if c.DBURI == "" {
		return fmt.Errorf("db: database URI is empty")
	}
	if c.WarmConnections > c.MaxIdleConnections {
		return fmt.Errorf("db: warm connections (%d) exceed max idle connections (%d), so the pool would close them", c.WarmConnections, c.MaxIdleConnections)
	}
	return nil
}

// NewDbPool creates a new instrumented SQL database connection pool for MySQL with default parameters and tracing.
func NewDbPool(config *Config) (*sql.DB, error) {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}

	params := []string{}

	if !strings.Contains(config.DBURI, "parseTime=") {
		params = append(params, "parseTime=true")
	}
	if !strings.Contains(config.DBURI, "loc=") {
		params = append(params, "loc=UTC")
	}
	if !strings.Contains(config.DBURI, "time_zone=") {
		params = append(params, "time_zone='%2B00:00'")
	}
	// Client-side interpolation sends each query in one round trip instead of prepare + execute.
	if !strings.Contains(config.DBURI, "interpolateParams=") {
		params = append(params, "interpolateParams=true")
	}

	if len(params) > 0 {
		paramString := strings.Join(params, "&")
		if strings.Contains(config.DBURI, "?") {
			config.DBURI += "&" + paramString
		} else {
			config.DBURI += "?" + paramString
		}
	}

	var db *sql.DB
	var err error
	if config.TracingEnabled {
		db, err = otelsql.Open("mysql", config.DBURI,
			otelsql.WithTracerProvider(otel.GetTracerProvider()),
			otelsql.WithAttributes(semconv.DBSystemMySQL),
			otelsql.WithSpanOptions(otelsql.SpanOptions{
				Ping:           true,
				DisableErrSkip: true,
				SpanFilter:     spanFilter,
			}),
			otelsql.WithSpanNameFormatter(func(ctx context.Context, method otelsql.Method, query string) string {
				return string(method)
			}),
		)
	} else {
		db, err = sql.Open("mysql", config.DBURI)
		if err != nil {
			return nil, fmt.Errorf("failed to open database connection pool: %w", err)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database connection pool: %w", err)
	}

	db.SetConnMaxLifetime(config.ConnectionMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnectionMaxIdleTime)
	db.SetMaxOpenConns(config.MaxOpenConnections)
	db.SetMaxIdleConns(config.MaxIdleConnections)

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if config.WarmConnections > 0 {
		go keepWarm(db, config.WarmConnections, config.WarmInterval)
	}

	return db, nil
}

// keepWarm runs until the pool is closed. Checking out a dead or expired connection here, rather
// than in a request, is what moves the reconnect off the request path.
func keepWarm(db *sql.DB, n int, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for ok := warm(db, n, every); ok; ok = warm(db, n, every) {
		<-ticker.C
	}
}

// warm holds n connections at once, so the pool opens any it is missing, and pings each. It
// reports false once the pool is closed.
func warm(db *sql.DB, n int, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conns := make([]*sql.Conn, 0, n)
	defer func() {
		for _, c := range conns {
			_ = c.Close()
		}
	}()
	for range n {
		c, err := db.Conn(ctx)
		if err != nil {
			if isPoolClosed(err) {
				return false
			}
			slog.WarnContext(ctx, "db: warming connection failed", "error", err)
			return true
		}
		conns = append(conns, c)
		// A failed ping marks the connection bad, so it is discarded on Close and reopened next round.
		_ = c.PingContext(ctx)
	}
	return true
}

// database/sql does not export the error it returns once DB.Close has been called.
func isPoolClosed(err error) bool {
	return err != nil && err.Error() == "sql: database is closed"
}

func spanFilter(ctx context.Context, method otelsql.Method, _ string, _ []driver.NamedValue) bool {
	return trace.SpanFromContext(ctx).SpanContext().IsValid() && method != otelsql.MethodConnResetSession
}
