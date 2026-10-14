package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

func TestProbeSQL(t *testing.T) {
	q, err := probeSQL(domain.ReferenceProbe{Label: "sales orders", Table: "sales_order", Column: "payment_term_id"})
	require.NoError(t, err)
	assert.Equal(t, "SELECT 1 FROM `sales_order` WHERE `payment_term_id` = ? LIMIT 1", q)

	q, err = probeSQL(domain.ReferenceProbe{Label: "price list rules", Table: "price_list_rule", Column: "scope_id", Where: "scope_type = 'item'"})
	require.NoError(t, err)
	assert.Equal(t, "SELECT 1 FROM `price_list_rule` WHERE `scope_id` = ? AND (scope_type = 'item') LIMIT 1", q)

	for _, bad := range []domain.ReferenceProbe{
		{Table: "sales_order", Column: "payment_term_id"},
		{Label: "x", Table: "sales_order`", Column: "id"},
		{Label: "x", Table: "sales_order", Column: "id = id OR 1"},
		{Label: "x", Table: "Sales", Column: "id"},
	} {
		_, err := probeSQL(bad)
		assert.Error(t, err, "%+v", bad)
	}
}
