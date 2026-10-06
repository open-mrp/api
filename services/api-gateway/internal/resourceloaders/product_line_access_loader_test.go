package resourceloaders

import (
	"context"
	"testing"
	"time"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// stubAccessClient answers the batch reads the access loaders make. The embedded interface panics on anything else.
type stubAccessClient struct {
	pb.CoreServiceClient
	calls          *[]accessCall
	customerAccess []*pb.CustomerProductLineAccessInfo
	groupAccess    []*pb.AccountGroupProductLineAccessInfo
	customers      []*pb.CustomerProto
	groups         []*pb.AccountGroupInfo
	groupsErr      error
	productLines   []*pb.ProductLineInfo
	productLineErr error
}

// accessCall is one batch read the access loaders made and whether it went out as an include read.
type accessCall struct {
	rpc          string
	includeReads bool
}

func (s stubAccessClient) record(ctx context.Context, rpc string) {
	if s.calls == nil {
		return
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	identity, _ := contracts.GetIdentityFromMetadata(md)
	*s.calls = append(*s.calls, accessCall{rpc: rpc, includeReads: identity != nil && identity.IncludeReads})
}

func (s stubAccessClient) BatchGetCustomerProductLineAccessByIDs(ctx context.Context, _ *pb.BatchGetCustomerProductLineAccessByIDsRequest, _ ...grpc.CallOption) (*pb.BatchGetCustomerProductLineAccessByIDsResponse, error) {
	s.record(ctx, "access")
	return &pb.BatchGetCustomerProductLineAccessByIDsResponse{Items: s.customerAccess}, nil
}

func (s stubAccessClient) BatchGetAccountGroupProductLineAccessByIDs(ctx context.Context, _ *pb.BatchGetAccountGroupProductLineAccessByIDsRequest, _ ...grpc.CallOption) (*pb.BatchGetAccountGroupProductLineAccessByIDsResponse, error) {
	s.record(ctx, "access")
	return &pb.BatchGetAccountGroupProductLineAccessByIDsResponse{Items: s.groupAccess}, nil
}

func (s stubAccessClient) BatchGetCustomersByIDs(ctx context.Context, _ *pb.BatchGetCustomersByIDsRequest, _ ...grpc.CallOption) (*pb.BatchGetCustomersByIDsResponse, error) {
	s.record(ctx, "customers")
	return &pb.BatchGetCustomersByIDsResponse{Customers: s.customers}, nil
}

func (s stubAccessClient) BatchGetAccountGroupsByIDs(ctx context.Context, _ *pb.BatchGetAccountGroupsByIDsRequest, _ ...grpc.CallOption) (*pb.BatchGetAccountGroupsByIDsResponse, error) {
	s.record(ctx, "account_groups")
	if s.groupsErr != nil {
		return nil, s.groupsErr
	}
	return &pb.BatchGetAccountGroupsByIDsResponse{AccountGroups: s.groups}, nil
}

func (s stubAccessClient) BatchGetProductLinesByIDs(ctx context.Context, _ *pb.BatchGetProductLinesByIDsRequest, _ ...grpc.CallOption) (*pb.BatchGetProductLinesByIDsResponse, error) {
	s.record(ctx, "product_lines")
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

func callerContext() context.Context {
	account := "ac_seller"
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: account},
		Actor:  &types.IdentityActor{RelationType: types.IdentityRelationTypeInternal, ID: "us_reader", AccountID: &account},
	})
}

func assertEmbedsReadAsIncluded(t *testing.T, calls []accessCall, embeds ...string) {
	t.Helper()
	seen := map[string]bool{}
	for _, c := range calls {
		seen[c.rpc] = true
		if c.rpc == "access" && c.includeReads {
			t.Error("the access records themselves were read as an include")
		}
		if c.rpc != "access" && !c.includeReads {
			t.Errorf("the embedded %s were read without the include-reads flag", c.rpc)
		}
	}
	for _, rpc := range append([]string{"access"}, embeds...) {
		if !seen[rpc] {
			t.Errorf("%s was never read", rpc)
		}
	}
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

// Whoever may read the access reads the customer and product lines it embeds.
func TestLoadCustomerProductLineAccess_EmbedsReadAsIncluded(t *testing.T) {
	var calls []accessCall
	useAccessClient(t, stubAccessClient{
		calls:          &calls,
		customerAccess: []*pb.CustomerProductLineAccessInfo{{CustomerId: "ac_buyer", ProductLines: []*pb.ProductLineAccessInfo{{Id: "pl_a"}}}},
		customers:      []*pb.CustomerProto{{Id: "ac_buyer", Name: "Buyer"}},
		productLines:   []*pb.ProductLineInfo{accessTestLine("pl_a")},
	})

	out, apiErr := LoadCustomerProductLineAccess(callerContext(), []string{"ac_buyer"})
	if apiErr != nil {
		t.Fatalf("LoadCustomerProductLineAccess: %v", apiErr)
	}
	access := out["ac_buyer"].(*apiresource.CustomerProductLineAccess)
	if access.Customer == nil || access.ProductLines == nil || len(access.ProductLines.Data) != 1 {
		t.Fatalf("access = %+v, want the customer and its line", access)
	}
	assertEmbedsReadAsIncluded(t, calls, "customers", "product_lines")
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

func TestLoadAccountGroupProductLineAccess_EmbedsReadAsIncluded(t *testing.T) {
	var calls []accessCall
	useAccessClient(t, stubAccessClient{
		calls:        &calls,
		groupAccess:  []*pb.AccountGroupProductLineAccessInfo{{AccountGroupId: "ag_retail", ProductLines: []*pb.ProductLineAccessInfo{{Id: "pl_a"}}}},
		groups:       []*pb.AccountGroupInfo{{Id: "ag_retail", Name: "Retail"}},
		productLines: []*pb.ProductLineInfo{accessTestLine("pl_a")},
	})

	out, apiErr := LoadAccountGroupProductLineAccess(callerContext(), []string{"ag_retail"})
	if apiErr != nil {
		t.Fatalf("LoadAccountGroupProductLineAccess: %v", apiErr)
	}
	access := out["ag_retail"].(*apiresource.AccountGroupProductLineAccess)
	if access.AccountGroup == nil || access.ProductLines == nil {
		t.Fatalf("access = %+v, want the group and its line", access)
	}
	assertEmbedsReadAsIncluded(t, calls, "account_groups", "product_lines")
}

// An embed that still fails fails the request: an access record never shows a null group or lines in place of an error.
func TestLoadAccountGroupProductLineAccess_EmbedFailureFailsTheRequest(t *testing.T) {
	useAccessClient(t, stubAccessClient{
		groupAccess:  []*pb.AccountGroupProductLineAccessInfo{{AccountGroupId: "ag_retail", ProductLines: []*pb.ProductLineAccessInfo{{Id: "pl_a"}}}},
		groupsErr:    contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("You do not have permission to read customer groups.")),
		productLines: []*pb.ProductLineInfo{accessTestLine("pl_a")},
	})

	if _, apiErr := LoadAccountGroupProductLineAccess(context.Background(), []string{"ag_retail"}); apiErr == nil {
		t.Fatal("the group's refusal was swallowed")
	}
}
