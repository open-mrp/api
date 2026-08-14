package accountpriceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to export a customer's price list.
type ExportPriceListRequest struct {
	// ID of the customer whose prices are listed.
	CustomerID string `json:"customer_id" validate:"required"`
}

var sampleExportPriceListRequest = &ExportPriceListRequest{
	CustomerID: apiresource.SampleCustomerID,
}

func (*ExportPriceListRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportPriceListRequest)
}

// Starts a customer's price list and returns the job that tracks it.
//
// The document covers every product the customer may order, grouped by product line and then by the SKUs that share a price, with the attributes that vary shown as columns. Prices are calculated by the same engine that prices a sales order, so they include the customer's contracted prices and any volume discount they qualify for; a volume break becomes its own price column only where it actually changes a price.
//
// Pricing a whole catalog takes too long to hold a request open for, so the PDF is rendered in the background. Poll the returned job and download the file it names once it completes.
type ExportPriceListEndpoint struct{}

func (e *ExportPriceListEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportPriceListRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportPriceListRequest, *apiresource.Job]{
		Title:             "Export Price List",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-prices/actions/export-price-list",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportPriceListRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(AccountPriceSvc).ExportPriceList
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
