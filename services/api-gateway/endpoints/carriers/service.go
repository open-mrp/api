package carrierep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type CarrierSvc interface {
	ListCarriers(ctx context.Context, req *ListCarriersRequest) (*apiresource.List[apiresource.Carrier], *apierror.APIError)
	GetCarrier(ctx context.Context, req *RetrieveCarrierRequest) (*apiresource.Carrier, *apierror.APIError)
	CreateCarrier(ctx context.Context, req *CreateCarrierRequest) (*apiresource.Carrier, *apierror.APIError)
	UpdateCarrier(ctx context.Context, req *UpdateCarrierRequest) (*apiresource.Carrier, *apierror.APIError)
	DeleteCarrier(ctx context.Context, req *DeleteCarrierRequest) (*apiresource.EmptyResource, *apierror.APIError)
	InitiateOAuth(ctx context.Context, req *InitiateOAuthRequest) (*apiresource.OAuthResponse, *apierror.APIError)
	GetOAuthStatus(ctx context.Context, req *GetOAuthStatusRequest) (*apiresource.OAuthStatusResponse, *apierror.APIError)
	SyncOptions(ctx context.Context, req *SyncOptionsRequest) (*apiresource.Carrier, *apierror.APIError)
}

type CarrierSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type carrierSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var carrierSvcTracer = tracing.GetTracer("api-gateway.endpoints.carriers.service")

func (c *CarrierSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("carrier endpoint service: core client is required")
	}
	return nil
}

func NewCarrierSvc(config *CarrierSvcConfig) CarrierSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &carrierSvcImpl{
		coreClient: config.CoreClient,
	}
}

// ListCarriers returns a paginated list of carriers. The existing
// ListCarriers gRPC is used purely for pagination + filtering; the returned
// carrier IDs are then handed to LoadCarriers so each item is built via the
// resourcekit loader (clean apiresource + LoadMeta populated). The V2
// include resolver runs in APIEndpoint.Execute against the resulting list.
func (m *carrierSvcImpl) ListCarriers(ctx context.Context, req *ListCarriersRequest) (*apiresource.List[apiresource.Carrier], *apierror.APIError) {
	pbReq := &pb.ListCarriersRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, carrierSvcTracer, "service.carriers.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListCarriersResponse, error) {
			return m.coreClient.ListCarriers(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.Carriers))
	for i, c := range resp.Carriers {
		ids[i] = c.Id
	}
	loaded, apiErr := resourceloaders.LoadCarriers(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	carriers := make([]apiresource.Carrier, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			carriers = append(carriers, *(v.(*apiresource.Carrier)))
		}
	}
	return apiresource.NewList(carriers, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

// GetCarrier returns a carrier by ID using the resourcekit loader. The
// returned carrier is a clean *apiresource.Carrier with sub-resources left
// nil; APIEndpoint.Execute runs the include resolver afterwards to populate
// owner / owner.account / service_levels per the client's ?include[]=
// params (filtered by the endpoint's per-route allow-list).
func (m *carrierSvcImpl) GetCarrier(ctx context.Context, req *RetrieveCarrierRequest) (*apiresource.Carrier, *apierror.APIError) {
	return loadCarrierByID(ctx, req.CarrierID)
}

func (m *carrierSvcImpl) CreateCarrier(ctx context.Context, req *CreateCarrierRequest) (*apiresource.Carrier, *apierror.APIError) {
	isPortalEnabled := true
	if v, ok := req.CustomerPortalVisibility.Value(); ok {
		isPortalEnabled = v == constants.CustomerPortalVisibilityVisible
	}

	var code *string
	if v, ok := req.Code.Value(); ok {
		s := string(v)
		code = &s
	}

	pbReq := &pb.CreateCarrierRequest{
		Name:            req.Name,
		Code:            code,
		AccountNumber:   req.AccountNumber.Ptr(),
		IsPortalEnabled: isPortalEnabled,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, carrierSvcTracer, "service.carriers.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateCarrierResponse, error) {
			return m.coreClient.CreateCarrier(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadCarrierByID(ctx, resp.Carrier.Id)
}

func (m *carrierSvcImpl) UpdateCarrier(ctx context.Context, req *UpdateCarrierRequest) (*apiresource.Carrier, *apierror.APIError) {
	var isPortalEnabled *bool
	if v, ok := req.CustomerPortalVisibility.Value(); ok {
		enabled := v == constants.CustomerPortalVisibilityVisible
		isPortalEnabled = &enabled
	}

	pbReq := &pb.UpdateCarrierRequest{
		Id:              req.CarrierID,
		Name:            req.Name.Ptr(),
		IsPortalEnabled: isPortalEnabled,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, carrierSvcTracer, "service.carriers.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateCarrierResponse, error) {
			return m.coreClient.UpdateCarrier(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadCarrierByID(ctx, resp.Carrier.Id)
}

// loadCarrierByID wraps the single-ID load pattern used after every
// mutation. The mutation RPC returns a CarrierInfo whose ID is the only
// thing we care about; the resourcekit loader fetches the fresh apiresource
// with LoadMeta so the include resolver can populate sub-resources.
func loadCarrierByID(ctx context.Context, id string) (*apiresource.Carrier, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadCarriers(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Carrier not found.")
	}
	return v.(*apiresource.Carrier), nil
}

func (m *carrierSvcImpl) DeleteCarrier(ctx context.Context, req *DeleteCarrierRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteCarrierRequest{
		Id: req.CarrierID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, carrierSvcTracer, "service.carriers.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteCarrier(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *carrierSvcImpl) InitiateOAuth(ctx context.Context, req *InitiateOAuthRequest) (*apiresource.OAuthResponse, *apierror.APIError) {
	pbReq := &pb.InitiateCarrierOAuthRequest{
		CarrierId:   req.CarrierID,
		RedirectUri: req.RedirectURI,
		State:       req.State.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, carrierSvcTracer, "service.carriers.initiate_oauth", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.InitiateCarrierOAuthResponse, error) {
			return m.coreClient.InitiateCarrierOAuth(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.OAuthResponse{Object: constants.ObjectTypeOAuthResponse, OAuthURL: resp.OauthUrl}, nil
}

func (m *carrierSvcImpl) GetOAuthStatus(ctx context.Context, req *GetOAuthStatusRequest) (*apiresource.OAuthStatusResponse, *apierror.APIError) {
	pbReq := &pb.GetCarrierOAuthStatusRequest{
		CarrierId: req.CarrierID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, carrierSvcTracer, "service.carriers.get_oauth_status", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetCarrierOAuthStatusResponse, error) {
			return m.coreClient.GetCarrierOAuthStatus(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.OAuthStatusResponse{Object: constants.ObjectTypeOAuthStatusResponse, Status: resp.Status}, nil
}

func (m *carrierSvcImpl) SyncOptions(ctx context.Context, req *SyncOptionsRequest) (*apiresource.Carrier, *apierror.APIError) {
	pbReq := &pb.SyncServiceLevelsRequest{
		CarrierId: req.CarrierID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, carrierSvcTracer, "service.carriers.sync_options", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SyncServiceLevelsResponse, error) {
			return m.coreClient.SyncServiceLevels(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadCarrierByID(ctx, resp.Carrier.Id)
}
