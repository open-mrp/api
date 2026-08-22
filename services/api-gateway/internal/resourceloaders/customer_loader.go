package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
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

// HydrateCustomerEntities fills in the display Name (customer name) and Handle (customer number) on entity references that point to a customer, which the bare entity_type/entity_id pair cannot supply. The entities are mutated in place via one batched BatchGetCustomersByIDs call. Best-effort: nil entries, non-customer entities, unresolved/deleted ids, or a failed customer fetch leave the entity as a plain id/type reference.
func HydrateCustomerEntities(ctx context.Context, entities []*apiresource.Entity) {
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	for _, e := range entities {
		if e == nil || e.Type != constants.ObjectTypeCustomer {
			continue
		}
		if _, ok := seen[e.ID]; ok {
			continue
		}
		seen[e.ID] = struct{}{}
		ids = append(ids, e.ID)
	}
	if len(ids) == 0 {
		return
	}
	loaded, apiErr := LoadCustomers(ctx, ids)
	if apiErr != nil {
		return
	}
	for _, e := range entities {
		if e == nil || e.Type != constants.ObjectTypeCustomer {
			continue
		}
		c, ok := loaded[e.ID].(*apiresource.Customer)
		if !ok {
			continue
		}
		name := c.Name
		number := c.Number
		e.Name = &name
		e.Handle = &number
	}
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
