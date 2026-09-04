package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
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

// LoadPropertiesByID resolves the properties a set of attributes name. An attribute carries only its
// property's id, so a caller that wants to print the property's name has to fetch the record; the
// alternative — an attribute shipping a property that is an id and a blank name — reads as a
// property that has no name rather than as one nobody looked up.
func LoadPropertiesByID(ctx context.Context, ids ...string) (map[string]*apiresource.Property, *apierror.APIError) {
	unique := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return map[string]*apiresource.Property{}, nil
	}

	loaded, apiErr := LoadProperties(ctx, unique)
	if apiErr != nil {
		return nil, apiErr
	}

	out := make(map[string]*apiresource.Property, len(loaded))
	for id, v := range loaded {
		if p, ok := v.(*apiresource.Property); ok {
			out[id] = p
		}
	}
	return out, nil
}
