//go:build plans

package repository

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
)

// TestProductionRunBatches_ReadsWhatItReturns holds ListBatchesByProductionRun to reading about what
// each of its statements returns (lookupPlanSuite). The run's batch flow is walked whole, by design:
// every statement is a lookup by run or by batch, and none may scan beyond its own rows.
func TestProductionRunBatches_ReadsWhatItReturns(t *testing.T) {
	ensureBatchCorpus(t)
	str := func(s string) *string { return &s }
	runScope := string(constants.ProductionRunBatchScopeRun)
	list := func(p domain.ListBatchesByProductionRunParams) func(ctx context.Context, q *sqlc.Queries) error {
		p.AccountID = planBatchAccount
		if p.Limit == 0 {
			p.Limit = 25
		}
		return func(ctx context.Context, q *sqlc.Queries) error {
			repo := NewProductionRunRepo(q)
			// The service resolves the run before listing its batches.
			if _, apiErr := repo.Get(ctx, domain.GetProductionRunParams{ProductionRunID: p.ProductionRunID, AccountID: p.AccountID}); apiErr != nil {
				return apiErr
			}
			if _, apiErr := repo.ListBatchesByRun(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		}
	}
	big, rare := planBatchRunID(planBatchBigRun), planBatchRunID(planBatchRareRun)
	lookupPlanSuite{
		tables: []string{"batch", "production_run"},
		cases: []lookupPlanCase{
			{"big/flow", list(domain.ListBatchesByProductionRunParams{ProductionRunID: big})},
			{"big/run", list(domain.ListBatchesByProductionRunParams{ProductionRunID: big, Scope: &runScope})},
			{"big/search", list(domain.ListBatchesByProductionRunParams{ProductionRunID: big, SearchQuery: str(planBatchRareSKU)})},
			{"typical/flow", list(domain.ListBatchesByProductionRunParams{ProductionRunID: planBatchRunID(planBatchRuns / 2)})},
			{"rare/flow", list(domain.ListBatchesByProductionRunParams{ProductionRunID: rare})},
		},
	}.run(t)
}
