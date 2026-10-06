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

var supplierLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.supplier")

// LoadSuppliers reads the suppliers documents name, as the supplier list shows them.
func LoadSuppliers(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, supplierLoaderTracer, "loader.suppliers.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetSuppliersByIDsResponse, error) {
			return coreClient.BatchGetSuppliersByIDs(ctx, &pb.BatchGetSuppliersByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	out := make(map[string]any, len(resp.Suppliers))
	for _, s := range resp.Suppliers {
		supplier := SupplierFromSummaryProto(s)
		out[s.Id] = &supplier
	}
	return out, nil
}

// SupplierFromSummaryProto maps a supplier as the list reads it, leaving its expandable addresses unset.
func SupplierFromSummaryProto(s *pb.SupplierSummaryProto) apiresource.Supplier {
	if s == nil {
		return apiresource.Supplier{}
	}
	materialCount := s.MaterialCount
	return apiresource.Supplier{
		ID:            s.Id,
		Object:        constants.ObjectTypeSupplier,
		Name:          s.Name,
		Number:        s.Number,
		Note:          s.Note,
		MaterialCount: &materialCount,
		CreatedAt:     grpcutil.TimestampToTimePtr(s.CreatedAt),
		UpdatedAt:     grpcutil.TimestampToTimePtr(s.UpdatedAt),
	}
}
