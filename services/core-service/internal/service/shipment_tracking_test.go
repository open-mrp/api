package service

import (
	"context"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSandboxTrackingNumber(t *testing.T) {
	// Deterministic per shipment so a retried ship reuses the same value.
	a := sandboxTrackingNumber("sh_01k0a87w33emw8pmkz1mf86cg1")
	b := sandboxTrackingNumber("sh_01k0a87w33emw8pmkz1mf86cg1")
	assert.Equal(t, a, b, "same shipment id must yield the same tracking number")
	assert.True(t, strings.HasPrefix(a, "SANDBOX-"), "sandbox tracking must carry the SANDBOX- prefix")
	assert.NotEqual(t, a, sandboxTrackingNumber("sh_different0000"), "different shipments differ")

	// A short id is handled without panicking.
	assert.True(t, strings.HasPrefix(sandboxTrackingNumber("sh_1"), "SANDBOX-"))
}

// Stubs the carrier and service level a live quote needs, for an account with no customer exemptions in play.
func (h *labelHarness) expectQuotableCarrier() {
	h.expectShippoCarrier()
	token := "ups_ground"
	h.serviceLevels.EXPECT().Get(gomock.Any(), testLabelAccountID, "crop_ground").
		Return(&domain.ServiceLevel{ID: "crop_ground", ServiceLevelToken: &token}, nil)
}

func estimateParams() domain.EstimateRateParams {
	return domain.EstimateRateParams{
		AccountID:      testLabelAccountID,
		CarrierID:      "car_ups",
		ServiceLevelID: "crop_ground",
		FromAddress:    *testOrigin(),
		ToAddress:      domain.ShippingAddress{Street1: "185 Berry St", City: "San Francisco", State: "CA", Zip: "94107", Country: "US"},
		Parcels:        []domain.Parcel{{Weight: "5"}},
	}
}

func TestEstimateShippingRate_InactiveIntegrationIsRefused(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	h.expectQuotableCarrier()
	h.expectShippoCredentials(false)

	// The Shippo client carries no expectations: an inactive integration is never quoted against.
	_, apiErr := estimateShippingRate(context.Background(), h.repoFactory, h.svc.shippoFactory, testEncryptionKey(), estimateParams())
	require.NotNil(t, apiErr)
	assert.Equal(t, "Shippo integration is inactive.", apiErr.PublicMessage)
}

// A carrier that answers without a rate is unavailable; quoting it as 0 would post free freight.
func TestEstimateShippingRate_MissingRateIsUnavailableNotFree(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	h.expectQuotableCarrier()
	h.expectShippoCredentials(true)
	h.shippoClient.EXPECT().FetchShippingRate(gomock.Any(), gomock.Any()).
		Return(0.0, domain.NewShippingRateUnavailableError("no rate"))

	rate, apiErr := estimateShippingRate(context.Background(), h.repoFactory, h.svc.shippoFactory, testEncryptionKey(), estimateParams())
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeSvcUnavailable, apiErr.Code)
	assert.True(t, apiErr.IsTransient, "the carrier may answer on a retry")
	assert.Zero(t, rate)
}

func TestAccountShippoClient_NoIntegrationSkipsTheCarrier(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	h.integrationRepo.EXPECT().HasIntegration(gomock.Any(), testLabelAccountID, constants.IntegrationCodeShippo).Return(false, nil)

	client, apiErr := accountShippoClient(context.Background(), h.repoFactory, h.svc.shippoFactory, testEncryptionKey(), testLabelAccountID)
	require.Nil(t, apiErr)
	assert.Nil(t, client)
}
