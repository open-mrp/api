package shippingtermep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/patch"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ShippingTermSvc interface {
	ListShippingTerms(ctx context.Context, req *ListShippingTermsRequest) (*apiresource.List[apiresource.ShippingTerm], *apierror.APIError)
	GetShippingTerm(ctx context.Context, req *RetrieveShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError)
	CreateShippingTerm(ctx context.Context, req *CreateShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError)
	UpdateShippingTerm(ctx context.Context, req *UpdateShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError)
	DeleteShippingTerm(ctx context.Context, req *DeleteShippingTermRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ShippingTermSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type shippingTermSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var shippingTermSvcTracer = tracing.GetTracer("api-gateway.endpoints.shipping-terms.service")

func (c *ShippingTermSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("shipping term endpoint service: core client is required")
	}
	return nil
}

func NewShippingTermSvc(config *ShippingTermSvcConfig) ShippingTermSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &shippingTermSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *shippingTermSvcImpl) ListShippingTerms(ctx context.Context, req *ListShippingTermsRequest) (*apiresource.List[apiresource.ShippingTerm], *apierror.APIError) {
	pbReq := &pb.ListShippingTermsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shippingTermSvcTracer, "service.shipping_terms.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListShippingTermsResponse, error) {
			return m.coreClient.ListShippingTerms(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.ShippingTerms))
	for i, st := range resp.ShippingTerms {
		ids[i] = st.Id
	}
	loaded, apiErr := resourceloaders.LoadShippingTerms(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.ShippingTerm, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.ShippingTerm)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *shippingTermSvcImpl) GetShippingTerm(ctx context.Context, req *RetrieveShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
	return loadShippingTermByID(ctx, req.ShippingTermID)
}

func (m *shippingTermSvcImpl) CreateShippingTerm(ctx context.Context, req *CreateShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
	pbReq := &pb.CreateShippingTermRequest{
		Name:                        req.Name,
		Type:                        string(req.Type),
		FreeShippingServiceLevelIds: req.FreeShippingServiceLevelIDs,
	}
	if q, ok := req.FlatRate.Value(); ok {
		pbReq.FlatRate = &pb.QuantityInput{
			Value:  q.Value,
			UnitId: q.UnitID,
		}
	}
	if q, ok := req.MinimumOrderValue.Value(); ok {
		pbReq.MinimumOrderValue = &pb.QuantityInput{
			Value:  q.Value,
			UnitId: q.UnitID,
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shippingTermSvcTracer, "service.shipping_terms.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateShippingTermResponse, error) {
			return m.coreClient.CreateShippingTerm(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadShippingTermByID(ctx, resp.ShippingTerm.Id)
}

func (m *shippingTermSvcImpl) UpdateShippingTerm(ctx context.Context, req *UpdateShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
	pbReq := &pb.UpdateShippingTermRequest{
		Id:   req.ShippingTermID,
		Name: req.Name,
	}
	if req.Type != nil {
		t := string(*req.Type)
		pbReq.Type = &t
	}
	pbReq.FlatRate = apirequest.QuantityFieldPtrToProto(req.FlatRate)
	pbReq.MinimumOrderValue = apirequest.QuantityFieldPtrToProto(req.MinimumOrderValue)
	pbReq.FreeShippingServiceLevelIds = patch.StringListSliceFieldPtrToProto(req.FreeShippingServiceLevelIDs)

	resp, apiErr := grpcutil.CallRPC(ctx, shippingTermSvcTracer, "service.shipping_terms.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateShippingTermResponse, error) {
			return m.coreClient.UpdateShippingTerm(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadShippingTermByID(ctx, resp.ShippingTerm.Id)
}

func (m *shippingTermSvcImpl) DeleteShippingTerm(ctx context.Context, req *DeleteShippingTermRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteShippingTermRequest{
		Id: req.ShippingTermID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, shippingTermSvcTracer, "service.shipping_terms.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteShippingTerm(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func loadShippingTermByID(ctx context.Context, id string) (*apiresource.ShippingTerm, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadShippingTerms(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Shipping term not found.")
	}
	return v.(*apiresource.ShippingTerm), nil
}
