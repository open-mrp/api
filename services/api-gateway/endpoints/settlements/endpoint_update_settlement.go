package settlementep

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

// Request to partially update a settlement.
type UpdateSettlementRequest struct {
	// Settlement ID.
	SettlementID string `path:"id" validate:"required"`
	// New settlement number.
	//
	// Must be unique within the account.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Note for this settlement.
	Note field.Optional[string] `json:"note,omitzero"`
	// ID of the user responsible for this settlement.
	//
	// Accepts either an account user ID or a user ID; the value is resolved to an account user in the current account.
	ResponsibleUserID field.Optional[string] `json:"responsible_user_id,omitzero" validate:"omitempty"`
}

var sampleUpdateSettlementNote = "Partial payment applied"
var sampleUpdateSettlementUserID = apiresource.SampleUserID
var sampleUpdateSettlementRequest = &UpdateSettlementRequest{
	Note:              field.Some(sampleUpdateSettlementNote),
	ResponsibleUserID: field.Some(sampleUpdateSettlementUserID),
}

func (*UpdateSettlementRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSettlementRequest)
}

// Partially updates a settlement's number, note, or responsible user.
type UpdateSettlementEndpoint struct{}

func (e *UpdateSettlementEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSettlementRequest, *apiresource.Settlement] {
	return (&apiendpoint.APIEndpoint[*UpdateSettlementRequest, *apiresource.Settlement]{
		Title:               "Update Settlement",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/finance/settlements/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSettlements, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeSettlement,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSettlementRequest) (*apiresource.Settlement, *apierror.APIError) {
			return svc.(SettlementSvc).UpdateSettlement
		},
	})
}
