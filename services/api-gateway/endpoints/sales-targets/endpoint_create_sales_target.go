package salestargetep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a sales target.
type CreateSalesTargetRequest struct {
	// ID of the account user (sales rep) the target is for.
	SalesRepID string `path:"id" validate:"required"`
	// Start of the period the target applies to (inclusive).
	StartDate time.Time `json:"start_date"`
	// End of the period the target applies to.
	EndDate time.Time `json:"end_date"`
	// Goal amount for the period, as a decimal string (e.g. `50000.00`).
	AmountValue string `json:"amount_value"`
	// ID of the unit the amount is denominated in (typically a currency unit).
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

// Creates a sales target for an account user.
type CreateSalesTargetEndpoint struct{}

func (e *CreateSalesTargetEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSalesTargetRequest, *apiresource.SalesTarget] {
	return (&apiendpoint.APIEndpoint[*CreateSalesTargetRequest, *apiresource.SalesTarget]{
		Title:             "Create Sales Target",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-users/{id}/sales-targets",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
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
