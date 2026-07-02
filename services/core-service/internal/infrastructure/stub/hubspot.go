package stub

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
)

// HubspotClientFactory is a no-op factory for use in test mode.
type HubspotClientFactory struct{}

func (f *HubspotClientFactory) Build(_ string) domain.HubspotClient {
	return &hubspotClient{}
}

type hubspotClient struct{}

func (s *hubspotClient) EnsureDealProperties(_ context.Context) *apierror.APIError {
	return nil
}

func (s *hubspotClient) SearchCompaniesByDomain(_ context.Context, _ string) ([]domain.HubspotCompany, *apierror.APIError) {
	return nil, nil
}

func (s *hubspotClient) SearchCompaniesByName(_ context.Context, _ string) ([]domain.HubspotCompany, *apierror.APIError) {
	return nil, nil
}

func (s *hubspotClient) ListCompanies(_ context.Context, _ string) ([]domain.HubspotCompany, string, *apierror.APIError) {
	return nil, "", nil
}

func (s *hubspotClient) CreateCompany(_ context.Context, company domain.HubspotCompany) (*domain.HubspotCompany, *apierror.APIError) {
	return &company, nil
}

func (s *hubspotClient) UpdateCompany(_ context.Context, _ string, _ domain.HubspotCompany) *apierror.APIError {
	return nil
}

func (s *hubspotClient) UpsertContactByEmail(_ context.Context, contact domain.HubspotContact) (*domain.HubspotContact, *apierror.APIError) {
	return &contact, nil
}

func (s *hubspotClient) SearchDealBySalesOrderID(_ context.Context, _ string) (*domain.HubspotDeal, *apierror.APIError) {
	return nil, nil
}

func (s *hubspotClient) CreateDeal(_ context.Context, deal domain.HubspotDeal) (*domain.HubspotDeal, *apierror.APIError) {
	return &deal, nil
}

func (s *hubspotClient) UpdateDeal(_ context.Context, _ string, _ domain.HubspotDeal) *apierror.APIError {
	return nil
}

func (s *hubspotClient) Associate(_ context.Context, _, _, _, _ string) *apierror.APIError {
	return nil
}
