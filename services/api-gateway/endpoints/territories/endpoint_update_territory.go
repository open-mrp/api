package territoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update a territory.
type UpdateTerritoryRequest struct {
	// Account ID.
	AccountID string `path:"account_id" validate:"required"`
	// Territory ID.
	TerritoryID string `path:"id" validate:"required"`
	// State this territory covers.
	State field.Optional[string] `json:"state,omitzero" validate:"omitempty,max=255"`
	// Start of ZIP code range (501-99999).
	StartZipcode field.Optional[int32] `json:"start_zipcode,omitzero"`
	// End of ZIP code range (501-99999).
	EndZipcode field.Optional[int32] `json:"end_zipcode,omitzero"`
	// Sales rep (account user) ID.
	SalesRepID field.Optional[string] `json:"sales_rep_id,omitzero" validate:"omitempty"`
	// Product line ID.
	ProductLineID field.Optional[string] `json:"product_line_id,omitzero" validate:"omitempty"`
	// Set to true to remove the product line.
	ClearProductLine field.Optional[bool] `json:"clear_product_line,omitzero"`
	// Set to true to remove the start ZIP code.
	ClearStartZipcode field.Optional[bool] `json:"clear_start_zipcode,omitzero"`
	// Set to true to remove the end ZIP code.
	ClearEndZipcode field.Optional[bool] `json:"clear_end_zipcode,omitzero"`
}

var sampleUpdateState = "CA"

var sampleUpdateTerritoryRequest = &UpdateTerritoryRequest{
	State: field.Some(sampleUpdateState),
}

func (*UpdateTerritoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateTerritoryRequest)
}

// Partially updates a territory.
type UpdateTerritoryEndpoint struct{}

func (e *UpdateTerritoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateTerritoryRequest, *apiresource.Territory] {
	return (&apiendpoint.APIEndpoint[*UpdateTerritoryRequest, *apiresource.Territory]{
		Title:             "Update Territory",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/accounts/{account_id}/territories/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeTerritory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateTerritoryRequest) (*apiresource.Territory, *apierror.APIError) {
			return svc.(TerritorySvc).UpdateTerritory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTerritory,
			Fields:     []string{"sales_rep", "sales_rep.user", "product_line"},
		}),
	})
}
