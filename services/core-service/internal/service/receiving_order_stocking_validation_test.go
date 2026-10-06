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
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pricing"
)

type stockingValidationFixture struct {
	repos     *factorymock.MockRepoFactory
	locations *repositorymock.MockLocationRepo
	pricing   *repositorymock.MockPricingRepo
	receiving *repositorymock.MockReceivingOrderRepo
}

func newStockingValidationFixture(t *testing.T) stockingValidationFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	f := stockingValidationFixture{
		repos:     factorymock.NewMockRepoFactory(ctrl),
		locations: repositorymock.NewMockLocationRepo(ctrl),
		pricing:   repositorymock.NewMockPricingRepo(ctrl),
		receiving: repositorymock.NewMockReceivingOrderRepo(ctrl),
	}
	f.repos.EXPECT().NewLocationRepo().Return(f.locations).AnyTimes()
	f.repos.EXPECT().NewPricingRepo().Return(f.pricing).AnyTimes()
	f.repos.EXPECT().NewReceivingOrderRepo().Return(f.receiving).AnyTimes()
	return f
}

// Puts a pair away at each location, from a line received four pairs of it_sock.
func stockFourPairsAt(f stockingValidationFixture, locationIDs ...string) *apierror.APIError {
	itemID := "it_sock"
	line := &domain.ReceivingOrderLine{ID: "rcorln_1", QuantityValue: "4", QuantityUnitID: "un_pr", OrderLineItemID: &itemID}
	allocations := make([]domain.StorageAllocation, len(locationIDs))
	for i, id := range locationIDs {
		allocations[i] = domain.StorageAllocation{LocationID: &id, Quantity: domain.ReceivedQuantity{Value: decimal.NewFromInt(1), UnitID: "un_pr"}}
	}
	data := domain.StockingData{LineItems: []domain.StockingLineItem{{ReceivingOrderLineID: line.ID, Allocations: allocations}}}
	return validateStockingData(context.Background(), f.repos, "ac_1", data, []*domain.ReceivingOrderLine{line}, map[string]struct{}{line.ID: {}})
}

// A location the account does not have, unknown or another tenant's, is absent from the account-scoped lookup and refused before units are read.
func TestValidateStockingData_RefusesALocationOutsideTheAccount(t *testing.T) {
	t.Parallel()

	f := newStockingValidationFixture(t)
	f.locations.EXPECT().GetByIDs(gomock.Any(), "ac_1", []string{"sglc_own", "sglc_elsewhere"}).
		Return([]*domain.Location{{ID: "sglc_own"}}, nil)

	apiErr := stockFourPairsAt(f, "sglc_own", "sglc_elsewhere", "sglc_own")
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeValidationFailed, apiErr.Code)
	assert.Equal(t, "line_items.allocations.location_id", apiErr.Param)
}

// The account's own locations pass, looked up once however many allocations name them.
func TestValidateStockingData_AcceptsTheAccountsOwnLocations(t *testing.T) {
	t.Parallel()

	f := newStockingValidationFixture(t)
	f.locations.EXPECT().GetByIDs(gomock.Any(), "ac_1", []string{"sglc_a", "sglc_b"}).
		Return([]*domain.Location{{ID: "sglc_b"}, {ID: "sglc_a"}}, nil)
	f.pricing.EXPECT().ItemQuantityUnits(gomock.Any(), "ac_1", []string{"it_sock"}).
		Return(map[string]map[string]struct{}{"it_sock": {"un_pr": {}}}, nil)
	f.pricing.EXPECT().ProductQuantityUnits(gomock.Any(), "ac_1", gomock.Any()).Return(nil, nil)
	f.receiving.EXPECT().GetUnitRatios(gomock.Any(), []string{"un_pr"}).
		Return(map[string]pricing.UnitRatio{"un_pr": {Numerator: decimal.NewFromInt(1), Denominator: decimal.NewFromInt(1)}}, nil)

	assert.Nil(t, stockFourPairsAt(f, "sglc_a", "sglc_b", "sglc_a"))
}
