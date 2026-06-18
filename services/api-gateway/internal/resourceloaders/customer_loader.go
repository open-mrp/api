package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var customerLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.customer")

// LoadCustomers fetches customers by ID via BatchGetCustomersByIDs and builds expandable Customer references with real header data. Nested sub-resources (addresses, defaults, …) are their own expandable relations and are not populated here.
func LoadCustomers(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, customerLoaderTracer, "loader.customers.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetCustomersByIDsResponse, error) {
			return coreClient.BatchGetCustomersByIDs(ctx, &pb.BatchGetCustomersByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Customers))
	for _, c := range resp.Customers {
		out[c.Id] = customerReferenceFromProto(c)
	}
	return out, nil
}

func customerReferenceFromProto(c *pb.CustomerProto) *apiresource.Customer {
	edi := constants.EDIStatusDisabled
	if c.IsEdiEnabled {
		edi = constants.EDIStatusEnabled
	}
	relationship := constants.CustomerRelationshipTypeStandalone
	if c.IsParentAccount {
		relationship = constants.CustomerRelationshipTypeParent
	} else if c.ParentAccount != nil {
		relationship = constants.CustomerRelationshipTypeChild
	}
	return &apiresource.Customer{
		ID:               c.Id,
		Object:           constants.ObjectTypeCustomer,
		Name:             c.Name,
		Number:           c.Number,
		Status:           constants.AccountStatusCode(c.Status),
		EDIStatus:        edi,
		RelationshipType: relationship,
		CommissionPolicy: constants.CommissionPolicy(c.CommissionPolicy),
		Note:             c.Note,
		CreatedAt:        grpcutil.TimestampToTime(c.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(c.UpdatedAt),
	}
}
