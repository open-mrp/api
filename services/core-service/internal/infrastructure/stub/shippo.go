package stub

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
)

// ShippoClientFactory is a no-op factory for use in test mode.
type ShippoClientFactory struct{}

func (f *ShippoClientFactory) Build(_ string) domain.ShippoClient {
	return &shippoClient{}
}

type shippoClient struct{}

func (s *shippoClient) FindOrRegisterCarrierAccount(_ context.Context, _ string) (*domain.ShippoCarrierAccount, *apierror.APIError) {
	return &domain.ShippoCarrierAccount{}, nil
}

func (s *shippoClient) ConnectCarrierAccount(_ context.Context, _, _ string, _ map[string]string) (*domain.ShippoCarrierAccount, *apierror.APIError) {
	return &domain.ShippoCarrierAccount{}, nil
}

func (s *shippoClient) GetCarrierAccount(_ context.Context, _ string) (*domain.ShippoCarrierAccount, *apierror.APIError) {
	return &domain.ShippoCarrierAccount{}, nil
}

func (s *shippoClient) DeactivateCarrierAccount(_ context.Context, _ string) *apierror.APIError {
	return nil
}

func (s *shippoClient) GetCarrierServiceLevels(_ context.Context, _ string) ([]domain.ShippoServiceLevel, *apierror.APIError) {
	return nil, nil
}

func (s *shippoClient) InitiateOAuth(_ context.Context, _, _ string, _ *string) (string, *apierror.APIError) {
	return "https://stub.local/oauth", nil
}

func (s *shippoClient) FetchShippingRate(_ context.Context, _ domain.FetchShippingRateParams) (float64, *apierror.APIError) {
	return 0, nil
}

func (s *shippoClient) FetchAllShippingRates(_ context.Context, _ domain.FetchAllShippingRatesParams) ([]domain.ShippoRateOption, *apierror.APIError) {
	return nil, nil
}
