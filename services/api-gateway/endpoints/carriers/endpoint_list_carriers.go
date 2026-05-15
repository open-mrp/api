package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list carriers.
type ListCarriersRequest struct {
	apiresource.PaginationRequest
}

type ListCarriersEndpoint struct{}

func (e *ListCarriersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCarriersRequest, *apiresource.List[apiresource.Carrier]] {
	return &apiendpoint.APIEndpoint[*ListCarriersRequest, *apiresource.List[apiresource.Carrier]]{
		Title:             "List Carriers",
		Description:       "Returns a paginated list of carriers for the current account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers",
		Request:           &ListCarriersRequest{},
		Response:          &apiresource.List[apiresource.Carrier]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCarriersRequest) (*apiresource.List[apiresource.Carrier], *apierror.APIError) {
			return svc.(CarrierSvc).ListCarriers
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCarrier,
			Fields:     []string{"owner", "owner.account", "service_levels"},
		}),
	}
}
