package territoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateTerritoryRequest is the request to partially update a territory.
type UpdateTerritoryRequest struct {
	// The ID of the account that owns the territory.
	AccountID string `path:"account_id" validate:"required"`
	// The ID of the territory to update.
	TerritoryID string `path:"id" validate:"required"`
	// The state this territory covers.
	State *string `json:"state,omitempty"`
	// The start of the zipcode range (501-99999).
	StartZipcode *int32 `json:"start_zipcode,omitempty"`
	// The end of the zipcode range (501-99999).
	EndZipcode *int32 `json:"end_zipcode,omitempty"`
	// The ID of the sales rep (account user) assigned to this territory.
	SalesRepID *string `json:"sales_rep_id,omitempty" nullable:"true"`
	// The ID of the product line this territory is scoped to.
	ProductLineID *string `json:"product_line_id,omitempty" nullable:"true"`
	// Set to true to remove the product line from this territory.
	ClearProductLine *bool `json:"clear_product_line,omitempty"`
	// Set to true to remove the start zipcode from this territory.
	ClearStartZipcode *bool `json:"clear_start_zipcode,omitempty"`
	// Set to true to remove the end zipcode from this territory.
	ClearEndZipcode *bool `json:"clear_end_zipcode,omitempty"`
}

var sampleUpdateState = "CA"

var sampleUpdateTerritoryRequest = &UpdateTerritoryRequest{
	State: &sampleUpdateState,
}

func (*UpdateTerritoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateTerritoryRequest)
}

type UpdateTerritoryEndpoint struct{}

func (e *UpdateTerritoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateTerritoryRequest, *apiresource.Territory] {
	return &apiendpoint.APIEndpoint[*UpdateTerritoryRequest, *apiresource.Territory]{
		Title:             "Update Territory",
		Description:       "Partially updates a territory.",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/accounts/{account_id}/territories/{id}",
		ContentType:       "application/json",
		Request:           &UpdateTerritoryRequest{},
		Response:          &apiresource.Territory{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateTerritoryRequest) (*apiresource.Territory, *apierror.APIError) {
			return svc.(TerritorySvc).UpdateTerritory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTerritory,
			Fields:     []string{"sales_rep", "product_line"},
		}),
	}
}
