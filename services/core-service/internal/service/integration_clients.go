package service

import (
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/integrationhealth"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

// buildShippoClient is the account's Shippo client, recording each call's outcome on its Shippo integration.
func buildShippoClient(factory domain.ShippoClientFactory, repo domain.AccountIntegrationRepo, accountID, apiKey string) (domain.ShippoClient, *apierror.APIError) {
	client, apiErr := factory.Build(apiKey)
	if apiErr != nil {
		return nil, apiErr
	}
	return integrationhealth.Shippo(client, integrationhealth.ForAccount(repo, accountID, constants.IntegrationCodeShippo)), nil
}
