package db

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func NewPgPool(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, err
	}
	config.MaxConns = 50
	config.MinConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	if _, err := registerPoolMetrics(otel.Meter("github.com/open-mrp/api/services/agent-service/internal/infrastructure/db"), pool.Stat); err != nil {
		slog.Warn("db: registering connection pool metrics failed", "error", err)
	}

	return pool, nil
}

// registerPoolMetrics reports pgxpool.Stat as the db.client.pool.* instruments on every metrics collection, the Postgres counterpart to the MySQL pools' otelsql db.sql.connection.* metrics.
func registerPoolMetrics(meter metric.Meter, stat func() *pgxpool.Stat) (metric.Registration, error) {
	connections, err := meter.Int64ObservableGauge("db.client.pool.connections",
		metric.WithDescription("Pool connections by state."))
	if err != nil {
		return nil, err
	}
	maxConnections, err := meter.Int64ObservableGauge("db.client.pool.max_connections",
		metric.WithDescription("Maximum connections the pool will open."))
	if err != nil {
		return nil, err
	}
	acquireWaits, err := meter.Int64ObservableCounter("db.client.pool.acquire_waits",
		metric.WithDescription("Acquires that had to wait because the pool had no idle connection."))
	if err != nil {
		return nil, err
	}
	acquireWaitDuration, err := meter.Float64ObservableCounter("db.client.pool.acquire_wait_duration",
		metric.WithDescription("Total time acquires spent waiting because the pool had no idle connection."),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	system := metric.WithAttributes(semconv.DBSystemPostgreSQL)
	stateAttrs := func(state string) metric.ObserveOption {
		return metric.WithAttributes(semconv.DBSystemPostgreSQL, attribute.String("state", state))
	}

	return meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		s := stat()
		o.ObserveInt64(connections, int64(s.IdleConns()), stateAttrs("idle"))
		o.ObserveInt64(connections, int64(s.AcquiredConns()), stateAttrs("acquired"))
		o.ObserveInt64(connections, int64(s.ConstructingConns()), stateAttrs("constructing"))
		o.ObserveInt64(maxConnections, int64(s.MaxConns()), system)
		o.ObserveInt64(acquireWaits, s.EmptyAcquireCount(), system)
		o.ObserveFloat64(acquireWaitDuration, s.EmptyAcquireWaitTime().Seconds(), system)
		return nil
	}, connections, maxConnections, acquireWaits, acquireWaitDuration)
}
