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

// CreateTerritoryRequest is the request to create a new territory.
type CreateTerritoryRequest struct {
	// The ID of the account to create the territory for.
	AccountID string `path:"account_id" validate:"required"`
	// The state this territory covers.
	State string `json:"state" validate:"required"`
	// The start of the zipcode range (501-99999).
	StartZipcode *int32 `json:"start_zipcode,omitempty"`
	// The end of the zipcode range (501-99999).
	EndZipcode *int32 `json:"end_zipcode,omitempty"`
	// The ID of the sales rep (account user) assigned to this territory.
	SalesRepID string `json:"sales_rep_id" validate:"required"`
	// The ID of the product line this territory is scoped to.
	ProductLineID *string `json:"product_line_id,omitempty"`
}

var sampleCreateStartZipcode int32 = 10001
var sampleCreateEndZipcode int32 = 10999

var sampleCreateTerritoryRequest = &CreateTerritoryRequest{
	State:        "NY",
	StartZipcode: &sampleCreateStartZipcode,
	EndZipcode:   &sampleCreateEndZipcode,
	SalesRepID:   "au_01jm4r6700f8nwq3v5hx2d9ktp",
}

func (*CreateTerritoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateTerritoryRequest)
}

type CreateTerritoryEndpoint struct{}

func (e *CreateTerritoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateTerritoryRequest, *apiresource.Territory] {
	return &apiendpoint.APIEndpoint[*CreateTerritoryRequest, *apiresource.Territory]{
		Title:             "Create Territory",
		Description:       "Creates a new territory for the specified account.",
		Method:            http.MethodPost,
		Route:             "/v1/sales/accounts/{account_id}/territories",
		Request:           &CreateTerritoryRequest{},
		Response:          &apiresource.Territory{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateTerritoryRequest) (*apiresource.Territory, *apierror.APIError) {
			return svc.(TerritorySvc).CreateTerritory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTerritory,
			Fields:     []string{"sales_rep", "product_line"},
		}),
	}
}
