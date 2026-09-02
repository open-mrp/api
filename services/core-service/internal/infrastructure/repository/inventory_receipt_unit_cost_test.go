package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
)

// A receipt is valued by converting its quantity into its cost's denominator, so a layer is worth what
// its stock is worth only while that denominator is the unit the item is counted in. The layer has no
// unit of its own to fall back on — it copies the item's cost whole — so whatever the rollup last wrote
// onto the item is what every receipt after it is valued at. That is the propagation half of the
// mislabelled-cost incident: one item wrong, and $35k of stock on the books that was never there.
//
// The rollup's half is covered by the item service tests. This pins the copy: substituting the
// receipt's own quantity unit for the item's, or dropping the denominator, has to fail here rather
// than in a month's valuation.
func TestCreateInventoryReceipt_LayerCarriesTheItemsOwnCostDenominator(t *testing.T) {
	t.Parallel()

	const (
		accountID    = "ac_cost"
		itemID       = "itm_hotmitt"
		cartonUnitID = "un_ct8ea"
		usdUnitID    = "un_usd"
		costPerCase  = "100.564364000000000000000000000000"
	)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO inventory_item_lock").WithArgs(itemID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO quantity").
		WithArgs(sqlmock.AnyArg(), "50", cartonUnitID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("FROM item i").WithArgs(itemID, accountID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "value", "numerator_unit_id", "denominator_unit_id"}).
			AddRow("rt_item_cost", costPerCase, usdUnitID, cartonUnitID))
	// The assertion: value, numerator and denominator all land on the layer exactly as the item holds
	// them. Fifty cartons then value at fifty times the cost, not at four hundred times it.
	mock.ExpectExec("INSERT INTO rate").
		WithArgs(sqlmock.AnyArg(), costPerCase, usdUnitID, cartonUnitID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO inventory_receipt").
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := NewInventoryMutationRepo(sqlc.New(db))

	scope, apiErr := ledgerlock.Acquire(context.Background(), repo, []string{itemID})
	if apiErr != nil {
		t.Fatalf("Acquire: %v", apiErr)
	}

	apiErr = repo.CreateInventoryReceipt(context.Background(), scope, domain.CreateInventoryReceiptParams{
		AccountID: accountID,
		ItemID:    itemID,
		Measure:   decimal.NewFromInt(50),
		UnitID:    cartonUnitID,
	})
	if apiErr != nil {
		t.Fatalf("CreateInventoryReceipt: %v", apiErr)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
