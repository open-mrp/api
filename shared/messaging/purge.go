package messaging

import (
	"context"
	"time"
)

const (
	// defaultPurgeBatchSize keeps each retention DELETE small. At 1000 rows the outbox purge reached a 1s p99, holding row locks and lagging replicas for its whole duration.
	defaultPurgeBatchSize = 200

	// defaultPurgeMaxBatches bounds one purge run, so a large backlog drains over a few ticks instead of in one long run.
	defaultPurgeMaxBatches = 100

	// purgeBatchPause lets replicas apply one batch before the next is written.
	purgeBatchPause = 50 * time.Millisecond
)

// purgeInBatches calls purge with batchSize until a batch comes back short (caught up), maxBatches have run, or ctx is done. Returns the rows deleted, including those deleted before an error.
func purgeInBatches(ctx context.Context, batchSize int32, maxBatches int, purge func(ctx context.Context, limit int32) (int64, error)) (int64, error) {
	var total int64
	for i := range maxBatches {
		if i > 0 {
			select {
			case <-ctx.Done():
				return total, nil
			case <-time.After(purgeBatchPause):
			}
		}
		deleted, err := purge(ctx, batchSize)
		total += deleted
		if err != nil {
			return total, err
		}
		if deleted < int64(batchSize) {
			return total, nil
		}
	}
	return total, nil
}
