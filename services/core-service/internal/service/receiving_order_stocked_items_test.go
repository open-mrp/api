package service

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
)

func stockedItemsSvc(t *testing.T, lines ...domain.ReceivingOrderLineUnitPrice) *receivingOrderSvcImpl {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockReceivingOrderRepo(ctrl)
	repo.EXPECT().GetLineUnitPrices(gomock.Any(), "rcor_1").Return(lines, nil)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewReceivingOrderRepo().Return(repo).AnyTimes()
	return &receivingOrderSvcImpl{repos: repos}
}

func stockingOf(lineItems ...domain.StockingLineItem) domain.StockReceivingOrderParams {
	return domain.StockReceivingOrderParams{ReceivingOrderID: "rcor_1", Data: domain.StockingData{LineItems: lineItems}}
}

var oneAllocation = []domain.StorageAllocation{{Quantity: domain.ReceivedQuantity{Value: decimal.NewFromInt(4), UnitID: "un_pr"}}}

// The items a stocking locks and asks demand to be covered for are the ones its allocations land on.
func TestStockedItemIDs_AreTheAllocatedLinesItems(t *testing.T) {
	t.Parallel()

	svc := stockedItemsSvc(t,
		domain.ReceivingOrderLineUnitPrice{ReceivingOrderLineID: "rcorln_a", ItemID: "it_a"},
		domain.ReceivingOrderLineUnitPrice{ReceivingOrderLineID: "rcorln_b", ItemID: "it_b"},
		domain.ReceivingOrderLineUnitPrice{ReceivingOrderLineID: "rcorln_c", ItemID: "it_c"},
	)
	got, apiErr := svc.stockedItemIDs(context.Background(), stockingOf(
		domain.StockingLineItem{ReceivingOrderLineID: "rcorln_a", Allocations: oneAllocation},
		domain.StockingLineItem{ReceivingOrderLineID: "rcorln_b"},
		domain.StockingLineItem{ReceivingOrderLineID: "rcorln_c", Allocations: oneAllocation},
	))
	require.Nil(t, apiErr)
	assert.Equal(t, []string{"it_a", "it_c"}, got, "a line only refused or counted puts nothing away")
}

// A line that resolves to no item would book its receipts, change log and lot against an empty item id,
// stock no item's on-hand counts. Stocking is refused before anything is written.
func TestStockedItemIDs_RefuseALineWithNoItem(t *testing.T) {
	t.Parallel()

	for name, lineItem := range map[string]domain.StockingLineItem{
		"allocated": {ReceivingOrderLineID: "rcorln_none", Allocations: oneAllocation},
		"lot only":  {ReceivingOrderLineID: "rcorln_none", LotNumber: new("LOT-1")},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svc := stockedItemsSvc(t, domain.ReceivingOrderLineUnitPrice{ReceivingOrderLineID: "rcorln_none"})
			_, apiErr := svc.stockedItemIDs(context.Background(), stockingOf(lineItem))
			require.NotNil(t, apiErr)
		})
	}
}
