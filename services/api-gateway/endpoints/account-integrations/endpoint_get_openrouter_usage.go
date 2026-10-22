package accountintegrationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve the account's OpenRouter spend.
type GetOpenRouterUsageRequest struct{}

// Returns what the account has spent through its OpenRouter integration today, this week, this month and in total, with its monthly spend limit.
//
// Spend is read live from OpenRouter for the integration's API key. Fails with `404` if the account has no OpenRouter integration, and with `400` if OpenRouter rejects the stored key.
type GetOpenRouterUsageEndpoint struct{}

func (e *GetOpenRouterUsageEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetOpenRouterUsageRequest, *apiresource.OpenRouterUsage] {
	return (&apiendpoint.APIEndpoint[*GetOpenRouterUsageRequest, *apiresource.OpenRouterUsage]{
		Title:             "Get OpenRouter Usage",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/settings/integrations/openrouter/usage",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		RequiredRoleType:  constants.RoleTypeAdmin,
		Preview:           true,
		ObjectType:        constants.ObjectTypeOpenRouterUsage,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetOpenRouterUsageRequest) (*apiresource.OpenRouterUsage, *apierror.APIError) {
			return svc.(AccountIntegrationSvc).GetOpenRouterUsage
		},
	})
}
