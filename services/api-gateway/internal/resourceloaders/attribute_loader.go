package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/include"
	"github.com/open-mrp/apikit/tracing"
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
		StashAttributeProperty(ctx, a.Id, a.PropertyId, nil)
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
		Status:    constants.ArchiveStatus(a.Status),
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}
}

// StashAttributeProperty records which property an attribute belongs to for its `property` include, and the property
// itself when it was read with the attribute's parent, so the include needs no separate read.
func StashAttributeProperty(ctx context.Context, attributeID, propertyID string, property *apiresource.Property) {
	if propertyID == "" {
		return
	}
	meta := include.GetLoadMeta(ctx)
	meta.Set(constants.ObjectTypeAttribute, attributeID, "property_id", propertyID)
	if property != nil {
		meta.Set(constants.ObjectTypeAttribute, attributeID, "property", property)
	}
}
