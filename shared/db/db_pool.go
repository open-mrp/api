package db

import (
	"cmp"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
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
	if !strings.Contains(config.DBURI, "interpolateParams=") {
		params = append(params, "interpolateParams=false")
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

	return db, nil
}

func spanFilter(ctx context.Context, method otelsql.Method, _ string, _ []driver.NamedValue) bool {
	return trace.SpanFromContext(ctx).SpanContext().IsValid() && method != otelsql.MethodConnResetSession
}
