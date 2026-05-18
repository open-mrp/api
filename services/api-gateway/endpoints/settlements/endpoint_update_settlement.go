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
	// Settlement ID.
	SettlementID string `path:"id" validate:"required"`
	// Settlement number.
	Number *string `json:"number" nullable:"false" validate:"omitempty,max=255"`
	// Note for this settlement.
	Note *string `json:"note" nullable:"false"`
	// Responsible user ID.
	ResponsibleUserID *string `json:"responsible_user_id" nullable:"false" validate:"omitempty"`
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

// Partially updates a settlement's number, note, or responsible user.
type UpdateSettlementEndpoint struct{}

func (e *UpdateSettlementEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSettlementRequest, *apiresource.Settlement] {
	return (&apiendpoint.APIEndpoint[*UpdateSettlementRequest, *apiresource.Settlement]{
		Title:             "Update Settlement",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/finance/settlements/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSettlementRequest) (*apiresource.Settlement, *apierror.APIError) {
			return svc.(SettlementSvc).UpdateSettlement
		},
	})
}
