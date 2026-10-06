package service

import (
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// The quote names its carriers and service levels whole, so a role that may rate shop sees them without carriers:read.
func TestShipmentSvc_RateShopReturnsTheCarriersAndServiceLevelsItsOptionsName(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	carrierRepo := repositorymock.NewMockCarrierRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewCarrierRepo().Return(carrierRepo).AnyTimes()
	meds := factorymock.NewMockMediatorFactory(ctrl)
	meds.EXPECT().Build(gomock.Any()).Return(domain.Mediators{}).AnyTimes()
	svc := NewShipmentSvc(&ShipmentSvcConfig{Repos: repos, MediatorFactory: meds, TxManager: &stubTxManager{factory: repos}, DispatchLeases: newMemLeases()})

	ground := &domain.ServiceLevel{ID: "crop_ground", Name: "Ground", CarrierID: "cr_truck"}
	express := &domain.ServiceLevel{ID: "crop_express", Name: "Express", CarrierID: "cr_truck"}
	pickup := &domain.ServiceLevel{ID: "crop_pickup", Name: "Pickup", CarrierID: "cr_dock"}
	carrierRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(&domain.ListCarriersResult{Carriers: []*domain.Carrier{
		{ID: "cr_truck", Name: "Truck"},
		{ID: "cr_dock", Name: "Dock"},
	}}, nil)
	carrierRepo.EXPECT().ListOptionsByCarrierIDs(gomock.Any(), "ac_vendor", []string{"cr_truck", "cr_dock"}).Return(map[string][]*domain.ServiceLevel{
		"cr_truck": {ground, express},
		"cr_dock":  {pickup},
	}, nil)

	result, apiErr := svc.RateShop(shipmentInternalCtx("ac_vendor"), domain.RateShopParams{
		FromAddress: domain.ShippingAddress{Street1: "1 Mill Rd", City: "Hickory", State: "NC", Zip: "28601"},
		ToAddress:   domain.ShippingAddress{Street1: "2 Oak Ave", City: "Los Angeles", State: "CA", Zip: "90001"},
		Parcels:     []domain.Parcel{{Weight: "5", Length: "12", Width: "8", Height: "6"}},
	})
	require.Nil(t, apiErr)
	require.Len(t, result.Options, 3)

	carrierIDs := map[string]bool{}
	for _, c := range result.Carriers {
		carrierIDs[c.ID] = true
		assert.Nil(t, c.ServiceLevels, "the carrier does not repeat the service levels the options name")
	}
	assert.Equal(t, map[string]bool{"cr_truck": true, "cr_dock": true}, carrierIDs, "each named carrier, once")
	assert.ElementsMatch(t, []*domain.ServiceLevel{ground, express, pickup}, result.ServiceLevels)
}
