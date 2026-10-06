package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// Stocking writes its receipts, change log and lot on the item this read names. A purchase order line
// that orders a product leaves its own item column empty, so the item has to come from the product, or
// the stock is booked against no item at all.
func TestGetLineUnitPrices_NamesTheItemTheLineRestocks(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	columns := []string{"receiving_order_line_id", "item_id", "product_item_id", "unit_price_value",
		"unit_price_numerator_unit_id", "unit_price_denominator_unit_id", "quantity_unit_id"}
	mock.ExpectQuery("LEFT JOIN product p ON p.id = sol.product_id").WithArgs("rcor_1").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow("rcorln_item", "it_material", nil, "6", "dollar", "un_lb", "un_lb").
			AddRow("rcorln_product", nil, "it_product", "9.5", "dollar", "un_pr", "un_pr").
			AddRow("rcorln_neither", nil, nil, "1", "dollar", "un_pr", "un_pr"))

	got, apiErr := NewReceivingOrderRepo(sqlc.New(db)).GetLineUnitPrices(context.Background(), "rcor_1")
	require.Nil(t, apiErr)
	require.Len(t, got, 3)

	assert.Equal(t, "it_material", got[0].ItemID, "a line that names an item restocks it")
	assert.Equal(t, "it_product", got[1].ItemID, "a line that orders a product restocks the product's item")
	assert.Empty(t, got[2].ItemID, "a line with neither names nothing, for the caller to refuse")
	require.NoError(t, mock.ExpectationsWereMet())
}
