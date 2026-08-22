package salestargetep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to create a sales target.
type CreateSalesTargetRequest struct {
	// The account user (sales rep) the target is for.
	//
	// Must be an active account user in your account.
	SalesRepID string `path:"id" validate:"required"`
	// Start of the period the target applies to (inclusive).
	StartDate time.Time `json:"starts_at"`
	// End of the period the target applies to.
	EndDate time.Time `json:"ends_at"`
	// The revenue goal for the period, as a decimal string (e.g. `50000.00`).
	AmountValue string `json:"amount_value"`
	// The unit the goal is denominated in, typically a currency unit.
	AmountUnitID string `json:"amount_unit_id" validate:"max=191"`
}

var sampleCreateSalesTargetRequest = &CreateSalesTargetRequest{
	StartDate:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	EndDate:      time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
	AmountValue:  "50000.00",
	AmountUnitID: apiresource.SampleUnitID,
}

func (*CreateSalesTargetRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSalesTargetRequest)
}

// Creates a revenue goal for a sales rep covering a given period.
//
// The sales rep must be an active account user in your account, otherwise the request returns a not-found error. Periods are not checked for overlap, so a rep can hold several targets covering the same dates; use the upsert endpoint to change an existing target rather than adding another.
type CreateSalesTargetEndpoint struct{}

func (e *CreateSalesTargetEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSalesTargetRequest, *apiresource.SalesTarget] {
	return (&apiendpoint.APIEndpoint[*CreateSalesTargetRequest, *apiresource.SalesTarget]{
		Title:             "Create Sales Target",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-users/{id}/sales-targets",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSalesTarget,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSalesTargets, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSalesTargetRequest) (*apiresource.SalesTarget, *apierror.APIError) {
			return svc.(SalesTargetSvc).CreateSalesTarget
		},
	})
}
