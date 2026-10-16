package pricelistep

import (
	"context"
	"net/http"

	"google.golang.org/grpc"

	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
)

// Request to export the prices a customer pays.
type ExportCustomerPricesRequest struct {
	// ID of the customer whose prices are listed.
	CustomerID string `json:"customer_id" validate:"required"`
}

var sampleExportCustomerPricesRequest = &ExportCustomerPricesRequest{CustomerID: apiresource.SampleCustomerID}

func (*ExportCustomerPricesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportCustomerPricesRequest)
}

// Exports the prices a customer pays as a PDF.
//
// The document covers every item the customer may order, priced by the same price lists as an order line, per the item's default order unit; a tier becomes its own price column where it changes a price. It renders in the background: poll the returned job and download the file it names once it completes.
type ExportCustomerPricesEndpoint struct{}

func (e *ExportCustomerPricesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportCustomerPricesRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*ExportCustomerPricesRequest, *apiresource.AsyncJob]{
		Title:             "Export Customer Prices",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             priceListsRoute + "/actions/export-customer-prices",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		ObjectType:        constants.ObjectTypeAsyncJob,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportCustomerPricesRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(PriceListSvc).ExportCustomerPrices
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string { return "/v1/core/async-jobs/" + resp.ID },
	})
}

func (s *priceListSvcImpl) ExportCustomerPrices(ctx context.Context, req *ExportCustomerPricesRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	resp, apiErr := call(ctx, "service.price_lists.export_customer_prices", func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportPriceListResponse, error) {
		return s.exportClient.ExportPriceList(ctx, &pb.ExportPriceListRequest{CustomerAccountId: req.CustomerID}, opts...)
	})
	if apiErr != nil {
		return nil, apiErr
	}
	return jobep.JobFromProto(ctx, resp.GetJob()), nil
}
