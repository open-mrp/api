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

var attributeLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.attribute")

func LoadAttributes(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, attributeLoaderTracer, "loader.attributes.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetAttributesByIDsResponse, error) {
			return coreClient.BatchGetAttributesByIDs(ctx, &pb.BatchGetAttributesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Attributes))
	for _, a := range resp.Attributes {
		out[a.Id] = AttributeFromProto(a)
	}
	return out, nil
}

func AttributeFromProto(a *pb.AttributeInfo) *apiresource.Attribute {
	return &apiresource.Attribute{
		ID:        a.Id,
		Object:    constants.ObjectTypeAttribute,
		Value:     a.Value,
		ColorCode: constants.Color(a.ColorCode),
		SortOrder: a.SortOrder,
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}
}
