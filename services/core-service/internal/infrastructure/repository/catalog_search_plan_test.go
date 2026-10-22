//go:build plans

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// TestCatalogSearchMatches_AnswerAsTheLike requires an item or product list read from the search's ngram
// matches to return exactly the page the LIKE over every item returns: terms in the SKU and in the
// description, mixed case, terms whose 2-grams the index cannot hold, and a term nothing matches.
func TestCatalogSearchMatches_AnswerAsTheLike(t *testing.T) {
	ensureCatalogCorpus(t)
	db := planDB(t)
	ctx := context.Background()
	items := &itemRepoImpl{queries: sqlc.New(db)}
	products := &productRepoImpl{queries: sqlc.New(db)}

	narrowed := 0
	for _, term := range []string{planCatRareSearch, "plc-00001234", "00001234", "0000123", "item 002669", "CATALOG ITEM 00266", "26", "zz-nothing"} {
		matches, ok, err := catalogSearchMatches(ctx, db, planCatAccount, term)
		require.NoError(t, err, term)
		if !ok {
			continue
		}
		narrowed++
		q := term

		params := domain.ListItemsParams{AccountID: planCatAccount, Query: &q, Limit: 25}
		like := newItemListQuery(params, nil, nil)
		orderBy, orderArgs := like.page(nil, "i.created_at", "i.id")
		want, err := items.list(ctx, like, orderBy, orderArgs, params.Limit+1)
		require.NoError(t, err, term)
		if len(matches) == 0 {
			// The list answers an empty page without reading the items.
			require.Empty(t, want, "items for %q", term)
			continue
		}
		byID := newItemListQuery(params, nil, matches)
		orderBy, orderArgs = byID.page(nil, "i.created_at", "i.id")
		got, err := items.list(ctx, byID, orderBy, orderArgs, params.Limit+1)
		require.NoError(t, err, term)
		require.Equal(t, want, got, "items for %q", term)

		pparams := domain.ListProductsFullParams{AccountID: planCatAccount, Query: &q, Limit: 25}
		plike := newProductListQuery(pparams, nil, nil)
		orderBy, orderArgs = plike.page(nil, "p.created_at", "p.id")
		pwant, err := products.list(ctx, plike, orderBy, orderArgs, pparams.Limit+1)
		require.NoError(t, err, term)
		pbyID := newProductListQuery(pparams, nil, matches)
		orderBy, orderArgs = pbyID.page(nil, "p.created_at", "p.id")
		pgot, err := products.list(ctx, pbyID, orderBy, orderArgs, pparams.Limit+1)
		require.NoError(t, err, term)
		require.Equal(t, pwant, pgot, "products for %q", term)
	}
	require.GreaterOrEqual(t, narrowed, 5, "the ngram keys narrowed the terms")
}
