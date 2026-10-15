package partep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve a part by ID.
type RetrievePartRequest struct {
	// ID of the part to retrieve.
	ItemID string `path:"id" validate:"required"`
}

// Returns a part by ID.
type RetrievePartEndpoint struct{}

func (e *RetrievePartEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrievePartRequest, *apiresource.Part] {
	return (&apiendpoint.APIEndpoint[*RetrievePartRequest, *apiresource.Part]{
		Title:             "Retrieve Part",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/parts/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainParts, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrievePartRequest) (*apiresource.Part, *apierror.APIError) {
			return svc.(PartSvc).GetPart
		},
		ObjectType: constants.ObjectTypePart,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePart,
			Fields:     []string{"item", "item.category", "item.category.properties", "item.category.unit_group", "item.attributes"},
		}),
	})
}
