package main

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/blobstore"
	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/db/backfill"
	"github.com/open-mrp/api/shared/lease"
)

// backfillRetryInterval is how often a pod tries to take up an unfinished backfill.
const backfillRetryInterval = 5 * time.Minute

// requestLogPayloadMaxBatch caps the batch: each row can carry two 256 KB bodies, and the batch
// holds them all in memory while it uploads.
const requestLogPayloadMaxBatch = 50

// startBackfills starts each unfinished backfill in the background. They run after the deploy, not as
// part of it, on one pod at a time, and resume from their cursors across restarts.
func startBackfills(ctx context.Context, cfg *config, primary *sqlc.Queries, payloads *blobstore.Store, leaseSvc *lease.Lease) error {
	// Nothing to move without a bucket to move it to.
	if payloads == nil {
		return nil
	}

	// A small pool of its own on the replica, so the backfill can never crowd out request traffic.
	replicaDB, err := db.NewDbPool(&db.Config{
		DBURI:              cfg.DBReplicaURL,
		Application:        domain.ServiceName + "-backfill",
		MaxOpenConnections: 2,
		MaxIdleConnections: 2,
		// Batches are paced well apart; a cold connect is nothing next to the pause between them.
		WarmConnections: -1,
	})
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		_ = replicaDB.Close()
	}()

	requestLogs := repository.NewRequestLogPayloadBackfill(sqlc.New(replicaDB), primary, payloads, time.Now().UTC())
	runner, err := backfill.New(&backfill.Config{
		Name:     repository.RequestLogPayloadBackfillName,
		Batch:    requestLogs.Batch,
		Progress: repository.NewBackfillProgressRepo(primary),
		MaxBatch: requestLogPayloadMaxBatch,
	})
	if err != nil {
		return err
	}
	go runner.Keep(ctx, leaseSvc, backfillRetryInterval)
	return nil
}
