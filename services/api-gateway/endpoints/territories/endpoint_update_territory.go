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

// Request to partially update a territory.
type UpdateTerritoryRequest struct {
	// Account ID.
	AccountID string `path:"account_id" validate:"required"`
	// Territory ID.
	TerritoryID string `path:"id" validate:"required"`
	// State this territory covers.
	State *string `json:"state,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Start of ZIP code range (501-99999).
	StartZipcode *int32 `json:"start_zipcode,omitempty" nullable:"false"`
	// End of ZIP code range (501-99999).
	EndZipcode *int32 `json:"end_zipcode,omitempty" nullable:"false"`
	// Sales rep (account user) ID.
	SalesRepID *string `json:"sales_rep_id,omitempty" nullable:"false" validate:"omitempty"`
	// Product line ID.
	ProductLineID *string `json:"product_line_id,omitempty" nullable:"false" validate:"omitempty"`
	// Set to true to remove the product line.
	ClearProductLine *bool `json:"clear_product_line,omitempty" nullable:"false"`
	// Set to true to remove the start ZIP code.
	ClearStartZipcode *bool `json:"clear_start_zipcode,omitempty" nullable:"false"`
	// Set to true to remove the end ZIP code.
	ClearEndZipcode *bool `json:"clear_end_zipcode,omitempty" nullable:"false"`
}

var sampleUpdateState = "CA"

var sampleUpdateTerritoryRequest = &UpdateTerritoryRequest{
	State: &sampleUpdateState,
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateTerritoryRequest) (*apiresource.Territory, *apierror.APIError) {
			return svc.(TerritorySvc).UpdateTerritory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTerritory,
			Fields:     []string{"sales_rep", "product_line"},
		}),
	})
}
