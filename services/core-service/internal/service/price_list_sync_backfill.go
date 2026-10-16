package service

import (
	"context"

	"github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db/backfill"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

// PriceListSyncBackfillName is the price-list mirror backfill's row in backfill_progress.
const PriceListSyncBackfillName = "price_list_legacy_sync"

// PriceListSyncBackfill mirrors every account's legacy account prices, volume discounts and price groups into price
// lists once, walking accounts by id. Each account mirrors in a short transaction of its own; an account already
// mirrored (by this backfill or by a write that needed its lists) is skipped, so a re-run batch changes nothing.
type PriceListSyncBackfill struct {
	read domain.LegacyPricingRepo
	tx   TransactionManager
}

// NewPriceListSyncBackfill pages account IDs from read (a replica) and mirrors on the primary through tx.
func NewPriceListSyncBackfill(read domain.LegacyPricingRepo, tx TransactionManager) *PriceListSyncBackfill {
	return &PriceListSyncBackfill{read: read, tx: tx}
}

// Batch is a backfill.Batch over account ids.
func (b *PriceListSyncBackfill) Batch(ctx context.Context, m *backfill.Meter, cursor string, limit int) (string, int, bool, error) {
	var ids []string
	if err := m.Time(func() error {
		var apiErr *apierror.APIError
		ids, apiErr = b.read.AccountIDsAfter(ctx, cursor, int32(limit)) //nolint:gosec // the runner caps the batch size
		if apiErr != nil {
			return apiErr
		}
		return nil
	}); err != nil {
		return cursor, 0, false, err
	}
	if len(ids) == 0 {
		return cursor, 0, true, nil
	}
	for _, accountID := range ids {
		if err := m.Time(func() error {
			if apiErr := b.tx.WithTx(ctx, func(txCtx context.Context, repos domain.RepoFactory) *apierror.APIError {
				return ensurePriceListsSynced(txCtx, repos, accountID)
			}); apiErr != nil {
				return apiErr
			}
			return nil
		}); err != nil {
			return cursor, 0, false, err
		}
	}
	return ids[len(ids)-1], len(ids), len(ids) < limit, nil
}
