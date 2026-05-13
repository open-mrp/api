package propertyep

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PropertySvc interface {
	ListProperties(ctx context.Context, req *ListPropertiesRequest) (*apiresource.List[apiresource.Property], *apierror.APIError)
	GetProperty(ctx context.Context, req *RetrievePropertyRequest) (*apiresource.Property, *apierror.APIError)
	CreateProperty(ctx context.Context, req *CreatePropertyRequest) (*apiresource.Property, *apierror.APIError)
	UpdateProperty(ctx context.Context, req *UpdatePropertyRequest) (*apiresource.Property, *apierror.APIError)
	DeleteProperty(ctx context.Context, req *DeletePropertyRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ListAttributes(ctx context.Context, req *ListAttributesRequest) (*apiresource.List[apiresource.Attribute], *apierror.APIError)
	GetAttribute(ctx context.Context, req *RetrieveAttributeRequest) (*apiresource.Attribute, *apierror.APIError)
	CreateAttribute(ctx context.Context, req *CreateAttributeRequest) (*apiresource.Attribute, *apierror.APIError)
	UpdateAttribute(ctx context.Context, req *UpdateAttributeRequest) (*apiresource.Attribute, *apierror.APIError)
	DeleteAttribute(ctx context.Context, req *DeleteAttributeRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type PropertySvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type propertySvcImpl struct {
	coreClient pb.CoreServiceClient
}

var propertySvcTracer = tracing.GetTracer("api-gateway.endpoints.properties.service")

func (c *PropertySvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("property endpoint service: core client is required")
	}
	return nil
}

func NewPropertySvc(config *PropertySvcConfig) PropertySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &propertySvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *propertySvcImpl) ListProperties(ctx context.Context, req *ListPropertiesRequest) (*apiresource.List[apiresource.Property], *apierror.APIError) {
	pbReq := &pb.ListPropertiesRequest{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, propertySvcTracer, "service.properties.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPropertiesResponse, error) {
			return m.coreClient.ListProperties(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	includes := appctx.GetRequestedIncludeKeys(ctx)
	return PropertyListPresenter(resp, includes), nil
}

func (m *propertySvcImpl) GetProperty(ctx context.Context, req *RetrievePropertyRequest) (*apiresource.Property, *apierror.APIError) {
	pbReq := &pb.GetPropertyRequest{
		Id:       req.PropertyID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, propertySvcTracer, "service.properties.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPropertyResponse, error) {
			return m.coreClient.GetProperty(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	includes := make(map[string]bool)
	for _, k := range appctx.GetRequestedIncludeKeys(ctx) {
		includes[k] = true
	}
	result := PropertyPresenter(resp.Property, includes)
	return &result, nil
}

func (m *propertySvcImpl) CreateProperty(ctx context.Context, req *CreatePropertyRequest) (*apiresource.Property, *apierror.APIError) {
	pbReq := &pb.CreatePropertyRequest{
		Name:     req.Name,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, propertySvcTracer, "service.properties.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreatePropertyResponse, error) {
			return m.coreClient.CreateProperty(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	includes := make(map[string]bool)
	for _, k := range appctx.GetRequestedIncludeKeys(ctx) {
		includes[k] = true
	}
	result := PropertyPresenter(resp.Property, includes)
	return &result, nil
}

func (m *propertySvcImpl) UpdateProperty(ctx context.Context, req *UpdatePropertyRequest) (*apiresource.Property, *apierror.APIError) {
	pbReq := &pb.UpdatePropertyRequest{
		Id:       req.PropertyID,
		Name:     req.Name,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, propertySvcTracer, "service.properties.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdatePropertyResponse, error) {
			return m.coreClient.UpdateProperty(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	includes := make(map[string]bool)
	for _, k := range appctx.GetRequestedIncludeKeys(ctx) {
		includes[k] = true
	}
	result := PropertyPresenter(resp.Property, includes)
	return &result, nil
}

func (m *propertySvcImpl) DeleteProperty(ctx context.Context, req *DeletePropertyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeletePropertyRequest{
		Id: req.PropertyID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, propertySvcTracer, "service.properties.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteProperty(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *propertySvcImpl) ListAttributes(ctx context.Context, req *ListAttributesRequest) (*apiresource.List[apiresource.Attribute], *apierror.APIError) {
	pbReq := &pb.ListAttributesRequest{
		PropertyId: req.PropertyID,
		Cursor:     req.Cursor,
		Limit:      req.Limit,
		Query:      req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, propertySvcTracer, "service.attributes.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAttributesResponse, error) {
			return m.coreClient.ListAttributes(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AttributeListPresenter(resp), nil
}

func (m *propertySvcImpl) GetAttribute(ctx context.Context, req *RetrieveAttributeRequest) (*apiresource.Attribute, *apierror.APIError) {
	pbReq := &pb.GetAttributeRequest{
		PropertyId: req.PropertyID,
		Id:         req.AttributeID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, propertySvcTracer, "service.attributes.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAttributeResponse, error) {
			return m.coreClient.GetAttribute(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := AttributePresenter(resp.Attribute)
	return &result, nil
}

func (m *propertySvcImpl) CreateAttribute(ctx context.Context, req *CreateAttributeRequest) (*apiresource.Attribute, *apierror.APIError) {
	var colorCode constants.Color
	if req.ColorCode != nil {
		colorCode = *req.ColorCode
	} else {
		colorCode = randomColor()
	}

	var sortOrder int32 = -1
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	pbReq := &pb.CreateAttributeRequest{
		PropertyId: req.PropertyID,
		Value:      req.Value,
		ColorCode:  string(colorCode),
		SortOrder:  sortOrder,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, propertySvcTracer, "service.attributes.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateAttributeResponse, error) {
			return m.coreClient.CreateAttribute(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := AttributePresenter(resp.Attribute)
	return &result, nil
}

func (m *propertySvcImpl) UpdateAttribute(ctx context.Context, req *UpdateAttributeRequest) (*apiresource.Attribute, *apierror.APIError) {
	var colorCode *string
	if req.ColorCode != nil {
		s := string(*req.ColorCode)
		colorCode = &s
	}

	pbReq := &pb.UpdateAttributeRequest{
		PropertyId: req.PropertyID,
		Id:         req.AttributeID,
		Value:      req.Value,
		ColorCode:  colorCode,
		SortOrder:  req.SortOrder,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, propertySvcTracer, "service.attributes.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAttributeResponse, error) {
			return m.coreClient.UpdateAttribute(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := AttributePresenter(resp.Attribute)
	return &result, nil
}

func (m *propertySvcImpl) DeleteAttribute(ctx context.Context, req *DeleteAttributeRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteAttributeRequest{
		PropertyId: req.PropertyID,
		Id:         req.AttributeID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, propertySvcTracer, "service.attributes.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteAttribute(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

var assignableColors = []constants.Color{
	constants.ColorBlue,
	constants.ColorBrown,
	constants.ColorGray,
	constants.ColorGreen,
	constants.ColorOrange,
	constants.ColorPink,
	constants.ColorPurple,
	constants.ColorRed,
	constants.ColorYellow,
}

func randomColor() constants.Color {
	var b [8]byte
	_, _ = rand.Read(b[:])
	idx := binary.LittleEndian.Uint64(b[:]) % uint64(len(assignableColors))
	return assignableColors[idx]
}
