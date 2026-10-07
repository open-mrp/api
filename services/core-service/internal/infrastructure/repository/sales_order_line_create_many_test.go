package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// CreateMany is the batched insert used when a new order's lines are created. It must issue exactly one
// INSERT per backing table (quantity, rate, sales_order_line) regardless of line count — the whole point
// of the change — and number the lines 1..N in slice order. A line carrying a unit cost contributes a
// second rate row; a line without one does not. This pins all three: the per-table batching, the 1..N
// numbering, and the conditional cost rate.
func TestCreateMany_BatchesOneInsertPerTable(t *testing.T) {
	t.Parallel()

	const (
		orderID = "or_1"
		usd     = "un_usd"
		each    = "un_ea"
	)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	cost := "6"
	costNum := usd
	costDen := each

	lines := []domain.CreateSalesOrderLineParams{
		{
			SalesOrderID:               orderID,
			ProductID:                  "prod_1",
			ItemID:                     strPtr("itm_1"),
			ProductSKU:                 "SKU-1",
			QuantityValue:              "2",
			QuantityUnitID:             each,
			UnitPriceValue:             "10",
			UnitPriceNumeratorUnitID:   usd,
			UnitPriceDenominatorUnitID: each,
			UnitCostValue:              &cost,
			UnitCostNumeratorUnitID:    &costNum,
			UnitCostDenominatorUnitID:  &costDen,
			Metadata:                   map[string]string{"edi_line_item_id": "00010"},
		},
		{
			SalesOrderID:               orderID,
			ProductID:                  "prod_2",
			ItemID:                     strPtr("itm_2"),
			ProductSKU:                 "SKU-2",
			QuantityValue:              "3",
			QuantityUnitID:             each,
			UnitPriceValue:             "20",
			UnitPriceNumeratorUnitID:   usd,
			UnitPriceDenominatorUnitID: each,
		},
	}

	mock.ExpectExec("INSERT INTO quantity").
		WithArgs(
			sqlmock.AnyArg(), "2", each,
			sqlmock.AnyArg(), "3", each,
		).
		WillReturnResult(sqlmock.NewResult(0, 2))

	mock.ExpectExec("INSERT INTO rate").
		WithArgs(
			sqlmock.AnyArg(), "10", usd, each,
			sqlmock.AnyArg(), cost, usd, each,
			sqlmock.AnyArg(), "20", usd, each,
		).
		WillReturnResult(sqlmock.NewResult(0, 3))

	mock.ExpectExec("INSERT INTO sales_order_line").
		WithArgs(
			sqlmock.AnyArg(), "SKU-1", sqlmock.AnyArg(), int32(1), "prod_1", "itm_1", orderID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), []byte(`{"edi_line_item_id":"00010"}`),
			sqlmock.AnyArg(), "SKU-2", sqlmock.AnyArg(), int32(2), "prod_2", "itm_2", orderID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), []byte(nil),
		).
		WillReturnResult(sqlmock.NewResult(0, 2))

	repo := NewSalesOrderLineRepo(sqlc.New(db))

	if apiErr := repo.CreateMany(context.Background(), lines); apiErr != nil {
		t.Fatalf("CreateMany: %v", apiErr)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func strPtr(s string) *string { return &s }
