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

// Request to create a territory.
type CreateTerritoryRequest struct {
	// Account ID.
	AccountID string `path:"account_id" validate:"required"`
	// State this territory covers.
	State string `json:"state" validate:"required,max=255"`
	// Start of ZIP code range (501-99999).
	StartZipcode *int32 `json:"start_zipcode,omitempty"`
	// End of ZIP code range (501-99999).
	EndZipcode *int32 `json:"end_zipcode,omitempty"`
	// Sales rep (account user) ID.
	SalesRepID string `json:"sales_rep_id" validate:"required"`
	// Product line ID.
	ProductLineID *string `json:"product_line_id,omitempty" validate:"omitempty"`
}

var sampleCreateStartZipcode int32 = 10001
var sampleCreateEndZipcode int32 = 10999

var sampleCreateTerritoryRequest = &CreateTerritoryRequest{
	State:        "NY",
	StartZipcode: &sampleCreateStartZipcode,
	EndZipcode:   &sampleCreateEndZipcode,
	SalesRepID:   apiresource.SampleAccountUserID,
}

func (*CreateTerritoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateTerritoryRequest)
}

// Creates a territory.
type CreateTerritoryEndpoint struct{}

func (e *CreateTerritoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateTerritoryRequest, *apiresource.Territory] {
	return (&apiendpoint.APIEndpoint[*CreateTerritoryRequest, *apiresource.Territory]{
		Title:             "Create Territory",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/accounts/{account_id}/territories",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeTerritory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateTerritoryRequest) (*apiresource.Territory, *apierror.APIError) {
			return svc.(TerritorySvc).CreateTerritory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTerritory,
			Fields:     []string{"sales_rep", "product_line"},
		}),
	})
}
