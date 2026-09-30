package messaging

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPurgeInBatches(t *testing.T) {
	t.Run("stops at the first short batch", func(t *testing.T) {
		remaining := int64(450)
		var limits []int32
		total, err := purgeInBatches(t.Context(), 200, 10, func(_ context.Context, limit int32) (int64, error) {
			limits = append(limits, limit)
			n := min(remaining, int64(limit))
			remaining -= n
			return n, nil
		})
		require.NoError(t, err)
		require.Equal(t, int64(450), total)
		require.Equal(t, []int32{200, 200, 200}, limits)
	})

	t.Run("stops after max batches with a backlog left", func(t *testing.T) {
		calls := 0
		total, err := purgeInBatches(t.Context(), 10, 3, func(context.Context, int32) (int64, error) {
			calls++
			return 10, nil
		})
		require.NoError(t, err)
		require.Equal(t, 3, calls)
		require.Equal(t, int64(30), total)
	})

	t.Run("returns the rows deleted before an error", func(t *testing.T) {
		calls := 0
		boom := errors.New("boom")
		total, err := purgeInBatches(t.Context(), 10, 5, func(context.Context, int32) (int64, error) {
			calls++
			if calls == 2 {
				return 0, boom
			}
			return 10, nil
		})
		require.ErrorIs(t, err, boom)
		require.Equal(t, int64(10), total)
	})

	t.Run("stops between batches when the context ends", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		calls := 0
		total, err := purgeInBatches(ctx, 10, 5, func(context.Context, int32) (int64, error) {
			calls++
			cancel()
			return 10, nil
		})
		require.NoError(t, err)
		require.Equal(t, 1, calls)
		require.Equal(t, int64(10), total)
	})
}
