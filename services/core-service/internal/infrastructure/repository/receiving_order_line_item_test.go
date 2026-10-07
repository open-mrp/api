package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
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

// A receiving or delivery line shows the item it received. A purchase order line that orders a product
// names none of its own, so the line reports the product's.
func TestReceivingAndDeliveryLines_ShowTheProductsItemForAProductOrderedLine(t *testing.T) {
	t.Parallel()

	named := sql.NullString{String: "it_material", Valid: true}
	productItem := sql.NullString{String: "it_product", Valid: true}
	for name, tc := range map[string]struct {
		item, productItem sql.NullString
		want              *string
	}{
		"names an item":     {named, sql.NullString{}, &named.String},
		"orders a product":  {sql.NullString{}, productItem, &productItem.String},
		"names an empty id": {sql.NullString{Valid: true}, productItem, &productItem.String},
		"names neither":     {sql.NullString{}, sql.NullString{}, nil},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			listed := mapReceivingOrderLineRow(sqlc.ListReceivingOrderLinesByOrderIDsRow{OrderLineItemID: tc.item, OrderLineProductItemID: tc.productItem})
			assert.Equal(t, tc.want, listed.ReceivedItemID(), "a receiving order's line")
			got := mapGetReceivingOrderLineRow(sqlc.GetReceivingOrderLineRow{OrderLineItemID: tc.item, OrderLineProductItemID: tc.productItem})
			assert.Equal(t, tc.want, got.ReceivedItemID(), "a receiving order line on its own")
			delivered := mapDeliveryLineRow(sqlc.ListDeliveryLinesRow{ItemID: tc.item, ProductItemID: tc.productItem})
			assert.Equal(t, tc.want, delivered.ItemID, "a delivery's line")
		})
	}
}

// The item filter matches a line that orders a product made from the item as well as one naming it.
// The products are looked up first, so the list's key can drive either branch.
func TestReceivingOrderList_ItemFilterMatchesTheItemsProducts(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT p.id FROM product p WHERE p.item_id IN").WithArgs("it_1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("pd_1"))
	args := make([]driver.Value, 27)
	for i := range args {
		args[i] = sqlmock.AnyArg()
	}
	// After the account, search, status and item-filter flag: the item, then the product it makes.
	args[14], args[15] = sql.NullString{String: "it_1", Valid: true}, sql.NullString{String: "pd_1", Valid: true}
	mock.ExpectQuery(`OR \(sol.product_id IN \(\?\) AND COALESCE\(sol.item_id, ''\) = ''\)`).WithArgs(args...).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	_, apiErr := NewReceivingOrderRepo(sqlc.New(db)).List(context.Background(), domain.ListReceivingOrdersParams{
		AccountID: "ac_1", ItemIDs: []string{"it_1"}, Limit: 25,
	})
	require.Nil(t, apiErr)
	require.NoError(t, mock.ExpectationsWereMet())
}
