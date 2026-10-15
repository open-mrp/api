package productlineep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// UpsertProductLineInput is the input for a single product line in a bulk upsert operation.
type UpsertProductLineInput struct {
	// Display name, such as `Outdoor`. Matches an existing line by name (case-insensitive). A row matching a system line fails.
	Name string `json:"name" validate:"required,max=255"`
	// Whether sales of this line's products earn commission: `applied` or `exempt`.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// Whether this line's products add a freight charge: `billed` or `free`.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
	// Unit group the line's products are priced in, by `id` or `name`.
	UnitGroup field.Optional[apirequest.ObjectIdentifier] `json:"unit_group,omitzero" visibility:"internal"`
}

// BulkUpsertProductLinesRequest is the request to bulk upsert product lines.
type BulkUpsertProductLinesRequest struct {
	// Product lines to create or update, matched by name within the account.
	ProductLines []UpsertProductLineInput `json:"product_lines" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertProductLinesRequest = &BulkUpsertProductLinesRequest{
	ProductLines: []UpsertProductLineInput{
		{
			Name:             apiresource.SampleProductLineName,
			CommissionPolicy: constants.CommissionPolicyExempt,
			FreightPolicy:    constants.FreightPolicyBilled,
		},
	},
}

func (*BulkUpsertProductLinesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertProductLinesRequest)
}

// Creates or updates product lines in bulk.
//
// Rows are matched to existing lines by `name` (case-insensitive); a row matching a system line fails. The work runs in the background; the response is an async job to poll.
type BulkUpsertProductLinesEndpoint struct{}

func (e *BulkUpsertProductLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertProductLinesRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertProductLinesRequest, *apiresource.AsyncJob]{
		Title:             "Bulk Upsert Product Lines",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             productLinesBulkUpsertRoute,
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProductLines, Action: types.ActionCreate}, {Domain: types.PermissionDomainProductLines, Action: types.ActionUpdate}}, RequireAll: true},
		ObjectType:        constants.ObjectTypeAsyncJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAsyncJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertProductLinesRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(ProductLineSvc).BulkUpsertProductLines
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
