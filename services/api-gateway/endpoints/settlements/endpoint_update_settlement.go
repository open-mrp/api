package settlementep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateSettlementRequest is the request to update a settlement.
type UpdateSettlementRequest struct {
	// The ID of the settlement to update.
	SettlementID string `path:"id" validate:"required"`
	// The new settlement number.
	Number *string `json:"number"`
	// The new note for this settlement.
	Note *string `json:"note"`
	// The ID of the responsible user for this settlement.
	ResponsibleUserID *string `json:"responsible_user_id" nullable:"true"`
}

var sampleUpdateSettlementNote = "Partial payment applied"
var sampleUpdateSettlementUserID = apiresource.SampleUserID
var sampleUpdateSettlementRequest = &UpdateSettlementRequest{
	Note:              &sampleUpdateSettlementNote,
	ResponsibleUserID: &sampleUpdateSettlementUserID,
}

func (*UpdateSettlementRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSettlementRequest)
}

type UpdateSettlementEndpoint struct{}

func (e *UpdateSettlementEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSettlementRequest, *apiresource.Settlement] {
	return &apiendpoint.APIEndpoint[*UpdateSettlementRequest, *apiresource.Settlement]{
		Title:             "Update Settlement",
		Description:       "Partially updates a settlement's number, note, or responsible user.",
		Method:            http.MethodPatch,
		Route:             "/v1/finance/settlements/{id}",
		Request:           &UpdateSettlementRequest{},
		Response:          &apiresource.Settlement{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSettlementRequest) (*apiresource.Settlement, *apierror.APIError) {
			return svc.(SettlementSvc).UpdateSettlement
		},
	}
}
