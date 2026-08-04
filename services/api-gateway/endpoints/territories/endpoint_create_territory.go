package territoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create a territory.
type CreateTerritoryRequest struct {
	// ID of your account, which owns the territory.
	AccountID string `path:"account_id" validate:"required"`
	// State this territory covers (e.g. `NY`).
	//
	// A territory created without a ZIP code range is matched by comparing this value exactly against the ship-to address's state, so use the same format your addresses use.
	State string `json:"state" validate:"required,max=255"`
	// Inclusive start of the ZIP code range this territory covers (`501`-`99999`).
	//
	// Omit both ZIP code fields to cover the entire state.
	StartZipcode field.Optional[int32] `json:"start_zipcode,omitzero"`
	// Inclusive end of the ZIP code range this territory covers (`501`-`99999`).
	//
	// Dropped when no start ZIP code is supplied. Supplying a start without an end creates a territory that matches that single ZIP code.
	EndZipcode field.Optional[int32] `json:"end_zipcode,omitzero"`
	// ID of the account user to credit as the sales rep on orders matching this territory.
	SalesRepID string `json:"sales_rep_id" validate:"required"`
	// ID of the product line this territory is associated with.
	//
	// Sales rep auto-assignment matches on ZIP code and state only, so this records what the territory covers rather than narrowing which orders it matches.
	ProductLineID field.Optional[string] `json:"product_line_id,omitzero" validate:"omitempty"`
}

var sampleCreateStartZipcode int32 = 10001
var sampleCreateEndZipcode int32 = 10999

var sampleCreateTerritoryRequest = &CreateTerritoryRequest{
	State:        "NY",
	StartZipcode: field.Some(sampleCreateStartZipcode),
	EndZipcode:   field.Some(sampleCreateEndZipcode),
	SalesRepID:   apiresource.SampleAccountUserID,
}

func (*CreateTerritoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateTerritoryRequest)
}

// Creates a territory that assigns a sales rep to a state or ZIP code range.
//
// The territory takes effect for sales orders created afterwards that do not name a sales rep explicitly and whose customer has no default sales rep.
type CreateTerritoryEndpoint struct{}

func (e *CreateTerritoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateTerritoryRequest, *apiresource.Territory] {
	return (&apiendpoint.APIEndpoint[*CreateTerritoryRequest, *apiresource.Territory]{
		Title:               "Create Territory",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/sales/accounts/{account_id}/territories",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeTerritory,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainTerritories, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateTerritoryRequest) (*apiresource.Territory, *apierror.APIError) {
			return svc.(TerritorySvc).CreateTerritory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTerritory,
			Fields:     []string{"sales_rep", "sales_rep.user", "product_line"},
		}),
	})
}
