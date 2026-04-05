package propertyep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func PropertyPresenter(p *pb.PropertyInfo, includes map[string]bool) apiresource.Property {
	if p == nil {
		return apiresource.Property{}
	}

	prop := apiresource.Property{
		ID:        p.Id,
		Object:    constants.ObjectTypeProperty,
		Name:      p.Name,
		CreatedAt: grpcutil.TimestampToTime(p.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(p.UpdatedAt),
	}

	if includes["attributes"] && p.Attributes != nil {
		attrs := make([]apiresource.Attribute, len(p.Attributes))
		for i, a := range p.Attributes {
			attrs[i] = AttributePresenter(a)
		}
		prop.Attributes = apiresource.NewList(attrs, apiresource.PageInfo{})
	}

	return prop
}

func PropertyListPresenter(resp *pb.ListPropertiesResponse, includeKeys []string) *apiresource.List[apiresource.Property] {
	if resp == nil {
		return apiresource.NewList[apiresource.Property](nil, apiresource.PageInfo{})
	}

	includes := make(map[string]bool, len(includeKeys))
	for _, k := range includeKeys {
		includes[k] = true
	}

	properties := make([]apiresource.Property, len(resp.Properties))
	for i, p := range resp.Properties {
		properties[i] = PropertyPresenter(p, includes)
	}

	return apiresource.NewList(properties, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func AttributePresenter(a *pb.AttributeInfo) apiresource.Attribute {
	if a == nil {
		return apiresource.Attribute{}
	}

	return apiresource.Attribute{
		ID:        a.Id,
		Object:    constants.ObjectTypeAttribute,
		Value:     a.Value,
		ColorCode: constants.Color(a.ColorCode),
		SortOrder: a.SortOrder,
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}
}

func AttributeListPresenter(resp *pb.ListAttributesResponse) *apiresource.List[apiresource.Attribute] {
	if resp == nil {
		return apiresource.NewList[apiresource.Attribute](nil, apiresource.PageInfo{})
	}

	attributes := make([]apiresource.Attribute, len(resp.Attributes))
	for i, a := range resp.Attributes {
		attributes[i] = AttributePresenter(a)
	}

	return apiresource.NewList(attributes, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
