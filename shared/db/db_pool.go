package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strings"
	"time"

	"github.com/XSAM/otelsql"
	_ "github.com/go-sql-driver/mysql"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

func NewDbPool(dbURL string) (*sql.DB, error) {
	if dbURL != "" {
		params := []string{}

		if !strings.Contains(dbURL, "parseTime=") {
			params = append(params, "parseTime=true")
		}
		if !strings.Contains(dbURL, "loc=") {
			params = append(params, "loc=UTC")
		}
		if !strings.Contains(dbURL, "time_zone=") {
			params = append(params, "time_zone=UTC")
		}
		if !strings.Contains(dbURL, "interpolateParams=") {
			params = append(params, "interpolateParams=false")
		}

		if len(params) > 0 {
			paramString := strings.Join(params, "&")
			if strings.Contains(dbURL, "?") {
				dbURL += "&" + paramString
			} else {
				dbURL += "?" + paramString
			}
		}
	}

	db, err := otelsql.Open("mysql", dbURL,
		otelsql.WithTracerProvider(otel.GetTracerProvider()),
		otelsql.WithAttributes(semconv.DBSystemMySQL),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			Ping:           true,
			DisableErrSkip: true,
			SpanFilter:     SpanFilter,
		}),
		otelsql.WithSpanNameFormatter(func(ctx context.Context, method otelsql.Method, query string) string {
			return string(method)
		}),
	)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(time.Minute * 30)
	db.SetConnMaxIdleTime(time.Minute * 10)
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(50)

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func SpanFilter(ctx context.Context, method otelsql.Method, _ string, _ []driver.NamedValue) bool {
	return trace.SpanFromContext(ctx).SpanContext().IsValid() && method != otelsql.MethodConnResetSession
}
