package repository

import (
	"context"
	"fmt"
	"time"

	agentdb "github.com/augno/api/services/agent-service/internal/infrastructure/db"
	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/lease"
	"github.com/augno/api/shared/tracing"
)

var leaseRepoTracer = tracing.GetTracer("agent-service.lease_repository")

type leaseRepoImpl struct {
	queries *sqlc.Queries
}

// NewLeaseRepo creates a lease repository backed by the agent-service's PostgreSQL connection.
func NewLeaseRepo(queries *sqlc.Queries) lease.Repo {
	return &leaseRepoImpl{queries: queries}
}

func (r *leaseRepoImpl) Acquire(ctx context.Context, name, holder string, ttl time.Duration) (bool, error) {
	ctx, span := tracing.StartSpan(ctx, leaseRepoTracer, "repository.lease.acquire")
	defer span.End()

	affected, err := r.queries.AcquireTaskLease(ctx, sqlc.AcquireTaskLeaseParams{
		Name:    name,
		Holder:  holder,
		Column3: agentdb.PgText(fmt.Sprintf("%d", int64(ttl/time.Second))),
	})
	if err != nil {
		span.RecordError(err)
		return false, err
	}
	return affected > 0, nil
}

func (r *leaseRepoImpl) Renew(ctx context.Context, name, holder string, ttl time.Duration) (bool, error) {
	ctx, span := tracing.StartSpan(ctx, leaseRepoTracer, "repository.lease.renew")
	defer span.End()

	affected, err := r.queries.RenewTaskLease(ctx, sqlc.RenewTaskLeaseParams{
		Name:    name,
		Holder:  holder,
		Column1: agentdb.PgText(fmt.Sprintf("%d", int64(ttl/time.Second))),
	})
	if err != nil {
		span.RecordError(err)
		return false, err
	}
	return affected > 0, nil
}

func (r *leaseRepoImpl) Release(ctx context.Context, name, holder string) error {
	ctx, span := tracing.StartSpan(ctx, leaseRepoTracer, "repository.lease.release")
	defer span.End()

	if err := r.queries.ReleaseTaskLease(ctx, sqlc.ReleaseTaskLeaseParams{
		Name:   name,
		Holder: holder,
	}); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}
