//go:build plans

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/pagination"
)

// A phrase's few matches are paged in memory from the columns read with them (phrasePage). Nothing but
// this compares that page with the SQL one it stands in for, so walk both through every filter the
// in-memory page applies, forward and back, on the corpus's ship-by ties.
func TestPickPhrasePage_MatchesTheSQLPage(t *testing.T) {
	ensureFulfillmentCorpus(t)
	db := planDB(t)
	repo := NewPickRepo(sqlc.New(db)).(*pickRepoImpl)
	ctx := context.Background()
	str := func(s string) *string { return &s }

	type variant struct {
		name  string
		apply func(*domain.ListPicksParams)
	}
	terms := []string{"1234", "Customer 055", "Customer 0012", "Customer 0599"}
	sorts := []variant{
		{"ship-by", func(*domain.ListPicksParams) {}},
		{"created", func(p *domain.ListPicksParams) { p.Sort = constants.PickSortCreatedAt }},
		{"created-window", func(p *domain.ListPicksParams) {
			p.Sort = constants.PickSortCreatedAt
			p.StartDate, p.EndDate = str("2023-06-01"), str("2025-06-30")
		}},
	}
	filters := []variant{
		{"all", func(*domain.ListPicksParams) {}},
		{"open", func(p *domain.ListPicksParams) { p.Status = str("open") }},
		{"closed", func(p *domain.ListPicksParams) { p.Status = str("closed") }},
		{"customers", func(p *domain.ListPicksParams) {
			p.CustomerIDs = []string{planFulCustomerID(12), planFulCustomerID(555), planFulCustomerID(planFulRareCustomer)}
		}},
	}

	inMemory := 0
	for _, term := range terms {
		for _, sort := range sorts {
			for _, filter := range filters {
				t.Run(term+"/"+sort.name+"/"+filter.name, func(t *testing.T) {
					params := domain.ListPicksParams{AccountID: planFulAccount, Limit: 40, Query: str(term)}
					sort.apply(&params)
					filter.apply(&params)

					// Forward three pages, then back one, each cursor taken from the page just read.
					directions := []pagination.Direction{"", pagination.DirectionForward, pagination.DirectionForward, pagination.DirectionBackward}
					var last, first string
					for step, dir := range directions {
						if dir != "" {
							anchor := last
							if dir == pagination.DirectionBackward {
								anchor = first
							}
							if anchor == "" {
								return
							}
							var at, shipBy time.Time
							require.NoError(t, db.QueryRow("SELECT created_at, ship_by_sort_date FROM pick WHERE id = ?", anchor).Scan(&at, &shipBy))
							if params.Sort != constants.PickSortCreatedAt {
								at = shipBy
							}
							c := pagination.EncodeStringCursor(pagination.StringCursor{OccurredAt: at, ID: anchor, Direction: dir})
							params.Cursor = &c
						}

						q, _, apiErr := repo.listQuery(ctx, params)
						require.Nil(t, apiErr)
						if q.PhraseMatches == nil {
							return
						}
						inMemory++
						got := q.phrasePage()
						q.PhraseMatches = nil
						want, apiErr := repo.listIDs(ctx, q)
						require.Nil(t, apiErr)
						if want == nil {
							want = []string{}
						}
						require.Equal(t, want, got, "page %d", step)

						if len(got) == 0 {
							return
						}
						ids := got[:min(len(got), int(params.Limit))]
						if dir == pagination.DirectionBackward {
							return
						}
						first, last = ids[0], ids[len(ids)-1]
					}
				})
			}
		}
	}
	require.Positive(t, inMemory, "no request paged its phrase in memory")
}
