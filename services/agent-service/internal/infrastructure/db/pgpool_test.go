package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestRegisterPoolMetrics(t *testing.T) {
	t.Parallel()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	// MinConns is zero, so the pool never dials the unreachable host.
	config, err := pgxpool.ParseConfig("postgres://user:pass@127.0.0.1:1/db")
	require.NoError(t, err)
	config.MaxConns = 9
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	_, err = registerPoolMetrics(provider.Meter("test"), pool.Stat)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	byName := map[string]metricdata.Metrics{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			byName[m.Name] = m
		}
	}

	postgres := attribute.String("db.system", "postgresql")

	connections, ok := byName["db.client.pool.connections"].Data.(metricdata.Gauge[int64])
	require.True(t, ok)
	states := map[string]int64{}
	for _, dp := range connections.DataPoints {
		state, _ := dp.Attributes.Value("state")
		system, _ := dp.Attributes.Value("db.system")
		require.Equal(t, postgres.Value, system)
		states[state.AsString()] = dp.Value
	}
	require.Equal(t, map[string]int64{"idle": 0, "acquired": 0, "constructing": 0}, states)

	maxConns, ok := byName["db.client.pool.max_connections"].Data.(metricdata.Gauge[int64])
	require.True(t, ok)
	require.Len(t, maxConns.DataPoints, 1)
	require.Equal(t, int64(9), maxConns.DataPoints[0].Value)

	waits, ok := byName["db.client.pool.acquire_waits"].Data.(metricdata.Sum[int64])
	require.True(t, ok)
	require.True(t, waits.IsMonotonic)
	require.Equal(t, metricdata.CumulativeTemporality, waits.Temporality)

	waitDuration, ok := byName["db.client.pool.acquire_wait_duration"].Data.(metricdata.Sum[float64])
	require.True(t, ok)
	require.True(t, waitDuration.IsMonotonic)
	require.Equal(t, "s", byName["db.client.pool.acquire_wait_duration"].Unit)
}
