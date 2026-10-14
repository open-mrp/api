package router

import (
	"context"
	"testing"

	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"google.golang.org/grpc"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	pb "github.com/open-mrp/api/shared/proto/core"
)

type unitClient struct {
	pb.CoreServiceClient
}

func (unitClient) BatchGetUnitsByIDs(_ context.Context, req *pb.BatchGetUnitsByIDsRequest, _ ...grpc.CallOption) (*pb.BatchGetUnitsByIDsResponse, error) {
	units := make([]*pb.UnitInfo, 0, len(req.Ids))
	for _, id := range req.Ids {
		units = append(units, &pb.UnitInfo{Id: id, Name: "Kilogram", Abbreviation: "kg", Type: "mass"})
	}
	return &pb.BatchGetUnitsByIDsResponse{Units: units}, nil
}

// The wrapped handler's response leaves with its units loaded; the endpoint's own handler is unchanged otherwise.
func TestFillHandlerUnits_LoadsTheResponseUnits(t *testing.T) {
	resourceloaders.SetCoreClient(unitClient{})
	t.Cleanup(func() { resourceloaders.SetCoreClient(nil) })

	ep := &apiendpoint.APIEndpoint[*struct{}, *apiresource.Quantity]{
		ServiceHandler: func(any) apiendpoint.ServiceHandler[*struct{}, *apiresource.Quantity] {
			return func(context.Context, *struct{}) (*apiresource.Quantity, *apierror.APIError) {
				return apiresource.NewQuantity("3", "un_kg"), nil
			}
		},
	}
	fillHandlerUnits(ep)

	q, apiErr := ep.ServiceHandler(nil)(context.Background(), &struct{}{})
	if apiErr != nil {
		t.Fatalf("handler error: %v", apiErr)
	}
	if apiresource.IsUnitStub(q.Unit) || q.Unit.Name != "Kilogram" {
		t.Errorf("unit = %+v, want the loaded kilogram", q.Unit)
	}
}

// A handler's error passes through untouched.
func TestFillHandlerUnits_PassesErrorsThrough(t *testing.T) {
	ep := &apiendpoint.APIEndpoint[*struct{}, *apiresource.Quantity]{
		ServiceHandler: func(any) apiendpoint.ServiceHandler[*struct{}, *apiresource.Quantity] {
			return func(context.Context, *struct{}) (*apiresource.Quantity, *apierror.APIError) {
				return nil, apierror.NewNotFoundError("missing")
			}
		},
	}
	fillHandlerUnits(ep)

	if _, apiErr := ep.ServiceHandler(nil)(context.Background(), &struct{}{}); apiErr == nil || apiErr.Code != apierror.CodeResourceNotFound {
		t.Errorf("error = %v, want resource_not_found", apiErr)
	}
}
