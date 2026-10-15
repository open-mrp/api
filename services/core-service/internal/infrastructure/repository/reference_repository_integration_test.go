//go:build integration

package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

var paymentTermProbes = []domain.ReferenceProbe{
	{Label: "customers", Table: "account_relation", Column: "payment_term_id"},
	{Label: "sales orders", Table: "sales_order", Column: "payment_term_id"},
}

func TestReferenceRepo_InUseBy(t *testing.T) {
	pool := testDB(t)
	repo := NewReferenceRepo(sqlc.New(pool))
	ctx := context.Background()

	var used string
	require.NoError(t, pool.QueryRow("SELECT payment_term_id FROM account_relation WHERE payment_term_id IS NOT NULL LIMIT 1").Scan(&used),
		"the seed gives a customer a payment term")
	var onOrders bool
	require.NoError(t, pool.QueryRow("SELECT EXISTS(SELECT 1 FROM sales_order WHERE payment_term_id = ?)", used).Scan(&onOrders))

	labels, apiErr := repo.InUseBy(ctx, used, paymentTermProbes)
	require.Nil(t, apiErr)
	want := []string{"customers"}
	if onOrders {
		want = append(want, "sales orders")
	}
	assert.Equal(t, want, labels)

	labels, apiErr = repo.InUseBy(ctx, "pt_never_referenced", paymentTermProbes)
	require.Nil(t, apiErr)
	assert.Empty(t, labels)

	narrowed := []domain.ReferenceProbe{{Label: "customers", Table: "account_relation", Column: "payment_term_id", Where: "1 = 0"}}
	labels, apiErr = repo.InUseBy(ctx, used, narrowed)
	require.Nil(t, apiErr)
	assert.Empty(t, labels, "Where narrows the probe")

	_, apiErr = repo.InUseBy(ctx, used, []domain.ReferenceProbe{{Label: "x", Table: "sales_order; DROP", Column: "id"}})
	assert.NotNil(t, apiErr, "a probe must name plain identifiers")
}

// explainProbe is the access type, index and Extra note MySQL plans for one probe.
func explainProbe(t *testing.T, pool *sql.DB, p domain.ReferenceProbe) (accessType string, key, extra sql.NullString) {
	t.Helper()
	q, err := probeSQL(p)
	require.NoError(t, err)
	rows, err := pool.Query("EXPLAIN "+q, "probe_id")
	require.NoError(t, err)
	defer rows.Close()
	cols, err := rows.Columns()
	require.NoError(t, err)
	require.True(t, rows.Next(), "EXPLAIN returned no plan")
	vals := make([]sql.NullString, len(cols))
	dest := make([]any, len(cols))
	for i := range vals {
		dest[i] = &vals[i]
	}
	require.NoError(t, rows.Scan(dest...))
	for i, c := range cols {
		switch c {
		case "type":
			accessType = vals[i].String
		case "key":
			key = vals[i]
		case "Extra":
			extra = vals[i]
		}
	}
	return accessType, key, extra
}

// probeIsIndexed reports whether a probe is an index lookup rather than a scan.
func probeIsIndexed(t *testing.T, pool *sql.DB, p domain.ReferenceProbe) bool {
	accessType, key, extra := explainProbe(t, pool, p)
	// A probe on a unique key is looked up while planning; when no row matches, the plan has no access type left.
	if extra.String == "no matching row in const table" {
		return true
	}
	switch accessType {
	case "const", "eq_ref", "ref", "ref_or_null", "range", "index_merge":
		return key.Valid
	default:
		return false
	}
}

// TestReferenceProbes_AreIndexed holds every registered resource_in_use probe to an index lookup, so a delete's
// reference check stays fast however large the referencing table grows.
func TestReferenceProbes_AreIndexed(t *testing.T) {
	pool := testDB(t)
	for _, refs := range append(domain.RegisteredReferences(), domain.References{Noun: "payment term", Probes: paymentTermProbes}) {
		for _, p := range refs.Probes {
			t.Run(refs.Noun+"/"+p.Table+"."+p.Column, func(t *testing.T) {
				assert.True(t, probeIsIndexed(t, pool, p), "probe %+v must use an index", p)
			})
		}
	}

	unindexed := domain.ReferenceProbe{Label: "addresses", Table: "address", Column: "phone"}
	assert.False(t, probeIsIndexed(t, pool, unindexed), "the check catches a probe on a column with no B-tree index")
}
