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

var sysPropertyLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.sys_property")

// LoadSysProperties fetches sys properties by ID via BatchGetSysPropertiesByIDs. Inline SysPropertyType is built from denormalized proto fields. Account-scoped.
func LoadSysProperties(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, sysPropertyLoaderTracer, "loader.sys_properties.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetSysPropertiesByIDsResponse, error) {
			return coreClient.BatchGetSysPropertiesByIDs(ctx, &pb.BatchGetSysPropertiesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.SysProperties))
	for _, sp := range resp.SysProperties {
		out[sp.Id] = sysPropertyFromProto(sp)
	}
	return out, nil
}

func sysPropertyFromProto(sp *pb.SysPropertyInfo) *apiresource.SysProperty {
	return &apiresource.SysProperty{
		ID:     sp.Id,
		Object: constants.ObjectTypeSysProperty,
		Type: &apiresource.SysPropertyType{
			ID:     sp.TypeId,
			Object: constants.ObjectTypeSysPropertyType,
			Name:   sp.TypeName,
			Code:   sp.TypeCode,
		},
		Value:     sp.Value,
		CreatedAt: grpcutil.TimestampToTime(sp.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(sp.UpdatedAt),
	}
}
