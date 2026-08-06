package productlineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpsertProductLineInput is the input for a single product line in a bulk upsert operation.
type UpsertProductLineInput struct {
	// Display name of the product line, matched case-insensitively against existing lines.
	// A row matching a system product line fails — system lines cannot be modified.
	Name string `json:"name" validate:"required,max=255"`
	// Unit group to associate with this product line, referenced by `id` or `name`.
	UnitGroup apirequest.ObjectIdentifier `json:"unit_group" validate:"required"`
	// Default commission policy for products in this product line.
	//
	// - `commission_exempt`: no commission applies to these products.
	// - `commission_applied`: commission applies to these products, unless overridden elsewhere.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// Default freight policy for products in this product line.
	//
	// - `free_freight`: these products do not incur a freight charge.
	// - `billed_freight`: freight is billed for these products, unless overridden elsewhere.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
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
			UnitGroup:        apirequest.ObjectIdentifier{ID: apiresource.SampleUnitGroupID},
			CommissionPolicy: constants.CommissionPolicyExempt,
			FreightPolicy:    constants.FreightPolicyBilled,
		},
	},
}

func (*BulkUpsertProductLinesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertProductLinesRequest)
}

// Creates or updates multiple product lines for the account, matched by name
// (case-insensitive), then writes asynchronously — 202 with a job to poll.
type BulkUpsertProductLinesEndpoint struct{}

func (e *BulkUpsertProductLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertProductLinesRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertProductLinesRequest, *apiresource.Job]{
		Title:             "Bulk Upsert Product Lines",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/product-lines/actions/bulk-upsert",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertProductLinesRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(ProductLineSvc).BulkUpsertProductLines
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
