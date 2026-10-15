package shippingtermep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/tracing"
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
	// CoreClient (required) is the core-service gRPC client.
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
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		IsActive: constants.IsActiveFilter(req.Statuses),
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
	if apiErr := exclusiveFreeShipping(req.FreeShipping.IsSet(), req.MinimumOrderValue.IsSet() || len(req.FreeShippingServiceLevelIDs) > 0); apiErr != nil {
		return nil, apiErr
	}
	pbReq := &pb.CreateShippingTermRequest{
		Name:                        req.Name,
		Type:                        string(req.Type),
		Status:                      archiveStatusPtr(req.Status),
		FreeShippingServiceLevelIds: req.FreeShippingServiceLevelIDs,
		FlatRate:                    apirequest.OptionalMoneyToProto(req.FlatRate),
		MinimumOrderValue:           apirequest.OptionalMoneyToProto(req.MinimumOrderValue),
	}
	if fs, ok := req.FreeShipping.Value(); ok {
		pbReq.MinimumOrderValue = fs.MinimumOrderValue.ToProto()
		pbReq.FreeShippingServiceLevelIds = fs.ServiceLevelIDs
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
	if apiErr := exclusiveFreeShipping(req.FreeShipping.WasProvided(), req.MinimumOrderValue.WasProvided() || req.FreeShippingServiceLevelIDs.WasProvided()); apiErr != nil {
		return nil, apiErr
	}
	pbReq := &pb.UpdateShippingTermRequest{
		Id:     req.ShippingTermID,
		Name:   req.Name.Ptr(),
		Status: archiveStatusPtr(req.Status),
	}
	if t, ok := req.Type.Value(); ok {
		ts := string(t)
		pbReq.Type = &ts
	}
	pbReq.FlatRate = apirequest.MoneyFieldToProto(req.FlatRate)
	pbReq.MinimumOrderValue = apirequest.MoneyFieldToProto(req.MinimumOrderValue)
	pbReq.FreeShippingServiceLevelIds = fieldpb.StringListSliceClearableToProto(req.FreeShippingServiceLevelIDs)
	switch {
	case req.FreeShipping.IsClear():
		pbReq.MinimumOrderValue = apirequest.MoneyFieldToProto(field.Clear[apirequest.MoneyInput]())
		pbReq.FreeShippingServiceLevelIds = fieldpb.StringListSliceClearableToProto(field.Clear[[]string]())
	case req.FreeShipping.IsSet():
		fs, _ := req.FreeShipping.Value()
		pbReq.MinimumOrderValue = apirequest.MoneyFieldToProto(field.Set(fs.MinimumOrderValue))
		levels := field.Clear[[]string]()
		if len(fs.ServiceLevelIDs) > 0 {
			levels = field.Set(fs.ServiceLevelIDs)
		}
		pbReq.FreeShippingServiceLevelIds = fieldpb.StringListSliceClearableToProto(levels)
	}

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
		return nil, apierror.NewNotFoundError("Shipping term not found.")
	}
	return v.(*apiresource.ShippingTerm), nil
}

// exclusiveFreeShipping refuses free_shipping sent beside the separate threshold fields preview.9 callers send.
func exclusiveFreeShipping(section, legacy bool) *apierror.APIError {
	if section && legacy {
		return apierror.NewFieldError("free_shipping", apierror.CodeInvalidFormat, "Send free shipping in `free_shipping` only.")
	}
	return nil
}

// archiveStatusPtr is a request's optional status as core takes it.
func archiveStatusPtr(status field.Optional[constants.ArchiveStatus]) *string {
	v, ok := status.Value()
	if !ok {
		return nil
	}
	s := string(v)
	return &s
}
