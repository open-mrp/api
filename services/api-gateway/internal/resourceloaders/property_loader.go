package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var propertyLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.property")

func LoadProperties(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, propertyLoaderTracer, "loader.properties.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetPropertiesByIDsResponse, error) {
			return coreClient.BatchGetPropertiesByIDs(ctx, &pb.BatchGetPropertiesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Properties))
	for _, p := range resp.Properties {
		out[p.Id] = PropertyFromProto(p)

		attrs := make([]apiresource.Attribute, len(p.Attributes))
		for i, a := range p.Attributes {
			attrs[i] = *AttributeFromProto(a)
		}
		meta.Set(constants.ObjectTypeProperty, p.Id, "attributes_list",
			apiresource.NewList(attrs, apiresource.PageInfo{}))
	}
	return out, nil
}

func PropertyFromProto(p *pb.PropertyInfo) *apiresource.Property {
	return &apiresource.Property{
		ID:        p.Id,
		Object:    constants.ObjectTypeProperty,
		Name:      p.Name,
		CreatedAt: grpcutil.TimestampToTime(p.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(p.UpdatedAt),
	}
}
