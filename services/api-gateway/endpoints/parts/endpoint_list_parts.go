package partep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list parts.
type ListPartsRequest struct {
	apiresource.PaginationRequest
	// Filter by category IDs.
	CategoryIDs []string `query:"category_ids"`
	// Filter by attribute IDs.
	AttributeIDs []string `query:"attribute_ids"`
	// Filter parts created on or after this date.
	StartDate *time.Time `query:"start_date"`
	// Filter parts created on or before this date.
	EndDate *time.Time `query:"end_date"`
}

// Returns a paginated list of parts for the current account.
type ListPartsEndpoint struct{}

func (e *ListPartsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPartsRequest, *apiresource.List[apiresource.Part]] {
	return (&apiendpoint.APIEndpoint[*ListPartsRequest, *apiresource.List[apiresource.Part]]{
		Title:             "List Parts",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/parts",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPartsRequest) (*apiresource.List[apiresource.Part], *apierror.APIError) {
			return svc.(PartSvc).ListParts
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePart,
			Fields:     []string{"item", "item.category", "item.category.properties", "item.category.unit_group", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
