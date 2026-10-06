package resourceloaders

import (
	"context"
	"testing"
	"time"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// stubAccessClient answers the batch reads the access loaders make. The embedded interface panics on anything else.
type stubAccessClient struct {
	pb.CoreServiceClient
	customerAccess []*pb.CustomerProductLineAccessInfo
	groupAccess    []*pb.AccountGroupProductLineAccessInfo
	customers      []*pb.CustomerProto
	groups         []*pb.AccountGroupInfo
	groupsErr      error
	productLines   []*pb.ProductLineInfo
	productLineErr error
}

func (s stubAccessClient) BatchGetCustomerProductLineAccessByIDs(context.Context, *pb.BatchGetCustomerProductLineAccessByIDsRequest, ...grpc.CallOption) (*pb.BatchGetCustomerProductLineAccessByIDsResponse, error) {
	return &pb.BatchGetCustomerProductLineAccessByIDsResponse{Items: s.customerAccess}, nil
}

func (s stubAccessClient) BatchGetAccountGroupProductLineAccessByIDs(context.Context, *pb.BatchGetAccountGroupProductLineAccessByIDsRequest, ...grpc.CallOption) (*pb.BatchGetAccountGroupProductLineAccessByIDsResponse, error) {
	return &pb.BatchGetAccountGroupProductLineAccessByIDsResponse{Items: s.groupAccess}, nil
}

func (s stubAccessClient) BatchGetCustomersByIDs(context.Context, *pb.BatchGetCustomersByIDsRequest, ...grpc.CallOption) (*pb.BatchGetCustomersByIDsResponse, error) {
	return &pb.BatchGetCustomersByIDsResponse{Customers: s.customers}, nil
}

func (s stubAccessClient) BatchGetAccountGroupsByIDs(context.Context, *pb.BatchGetAccountGroupsByIDsRequest, ...grpc.CallOption) (*pb.BatchGetAccountGroupsByIDsResponse, error) {
	if s.groupsErr != nil {
		return nil, s.groupsErr
	}
	return &pb.BatchGetAccountGroupsByIDsResponse{AccountGroups: s.groups}, nil
}

func (s stubAccessClient) BatchGetProductLinesByIDs(context.Context, *pb.BatchGetProductLinesByIDsRequest, ...grpc.CallOption) (*pb.BatchGetProductLinesByIDsResponse, error) {
	if s.productLineErr != nil {
		return nil, s.productLineErr
	}
	return &pb.BatchGetProductLinesByIDsResponse{ProductLines: s.productLines}, nil
}

var accessTestCreatedAt = time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC)

func useAccessClient(t *testing.T, client stubAccessClient) {
	original := coreClient
	t.Cleanup(func() { coreClient = original })
	coreClient = client
}

func accessTestLine(id string) *pb.ProductLineInfo {
	return &pb.ProductLineInfo{Id: id, Name: "Line " + id, CommissionPolicy: "commission_applied", FreightPolicy: "billed_freight", CreatedAt: timestamppb.New(accessTestCreatedAt)}
}

func noProductLineAccess() error {
	return contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("You do not have permission to read product lines."))
}

func TestLoadCustomerProductLineAccess_EmbedsTheRealRecords(t *testing.T) {
	useAccessClient(t, stubAccessClient{
		customerAccess: []*pb.CustomerProductLineAccessInfo{{
			CustomerId:   "ac_buyer",
			ProductLines: []*pb.ProductLineAccessInfo{{Id: "pl_b"}, {Id: "pl_a"}},
		}},
		customers:    []*pb.CustomerProto{{Id: "ac_buyer", Name: "Buyer", Number: "C-1", Status: "normal", CreatedAt: timestamppb.New(accessTestCreatedAt)}},
		productLines: []*pb.ProductLineInfo{accessTestLine("pl_a"), accessTestLine("pl_b")},
	})

	out, apiErr := LoadCustomerProductLineAccess(context.Background(), []string{"ac_buyer"})
	if apiErr != nil {
		t.Fatalf("LoadCustomerProductLineAccess: %v", apiErr)
	}
	access := out["ac_buyer"].(*apiresource.CustomerProductLineAccess)
	if access.Customer == nil || access.Customer.Object != constants.ObjectTypeCustomer || access.Customer.Status != "normal" || !access.Customer.CreatedAt.Equal(accessTestCreatedAt) {
		t.Fatalf("customer = %+v, want the real customer record", access.Customer)
	}
	if access.ProductLines == nil || len(access.ProductLines.Data) != 2 {
		t.Fatalf("product lines = %+v, want both granted lines", access.ProductLines)
	}
	first := access.ProductLines.Data[0]
	if first.ID != "pl_b" || first.CommissionPolicy != constants.CommissionPolicy("commission_applied") || !first.CreatedAt.Equal(accessTestCreatedAt) {
		t.Fatalf("first line = %+v, want the real pl_b in grant order", first)
	}
}

// A role that may not read product lines sees the lines as null, not as an empty grant.
func TestLoadCustomerProductLineAccess_UnreadableLinesAreNull(t *testing.T) {
	useAccessClient(t, stubAccessClient{
		customerAccess: []*pb.CustomerProductLineAccessInfo{{CustomerId: "ac_buyer", ProductLines: []*pb.ProductLineAccessInfo{{Id: "pl_a"}}}},
		customers:      []*pb.CustomerProto{{Id: "ac_buyer", Name: "Buyer"}},
		productLineErr: noProductLineAccess(),
	})

	out, apiErr := LoadCustomerProductLineAccess(context.Background(), []string{"ac_buyer"})
	if apiErr != nil {
		t.Fatalf("LoadCustomerProductLineAccess: %v", apiErr)
	}
	access := out["ac_buyer"].(*apiresource.CustomerProductLineAccess)
	if access.ProductLines != nil {
		t.Fatalf("product lines = %+v, want null", access.ProductLines)
	}
	if access.Customer == nil {
		t.Fatal("the customer is still readable")
	}
}

func TestLoadAccountGroupProductLineAccess_EmbedsTheRealGroup(t *testing.T) {
	useAccessClient(t, stubAccessClient{
		groupAccess:  []*pb.AccountGroupProductLineAccessInfo{{AccountGroupId: "ag_retail", ProductLines: []*pb.ProductLineAccessInfo{{Id: "pl_a"}}}},
		groups:       []*pb.AccountGroupInfo{{Id: "ag_retail", Name: "Retail", Type: "type_group", CreatedAt: timestamppb.New(accessTestCreatedAt)}},
		productLines: []*pb.ProductLineInfo{accessTestLine("pl_a")},
	})

	out, apiErr := LoadAccountGroupProductLineAccess(context.Background(), []string{"ag_retail"})
	if apiErr != nil {
		t.Fatalf("LoadAccountGroupProductLineAccess: %v", apiErr)
	}
	access := out["ag_retail"].(*apiresource.AccountGroupProductLineAccess)
	if access.AccountGroup == nil || access.AccountGroup.Type != constants.AccountGroupType("type_group") || !access.AccountGroup.CreatedAt.Equal(accessTestCreatedAt) {
		t.Fatalf("account group = %+v, want the real group record", access.AccountGroup)
	}
	if access.ProductLines == nil || len(access.ProductLines.Data) != 1 || access.ProductLines.Data[0].ID != "pl_a" {
		t.Fatalf("product lines = %+v, want pl_a", access.ProductLines)
	}
}

func TestLoadAccountGroupProductLineAccess_UnreadableGroupIsNull(t *testing.T) {
	useAccessClient(t, stubAccessClient{
		groupAccess:  []*pb.AccountGroupProductLineAccessInfo{{AccountGroupId: "ag_retail", ProductLines: []*pb.ProductLineAccessInfo{{Id: "pl_a"}}}},
		groupsErr:    contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("You do not have permission to read customer groups.")),
		productLines: []*pb.ProductLineInfo{accessTestLine("pl_a")},
	})

	out, apiErr := LoadAccountGroupProductLineAccess(context.Background(), []string{"ag_retail"})
	if apiErr != nil {
		t.Fatalf("LoadAccountGroupProductLineAccess: %v", apiErr)
	}
	if group := out["ag_retail"].(*apiresource.AccountGroupProductLineAccess).AccountGroup; group != nil {
		t.Fatalf("account group = %+v, want null", group)
	}
}
