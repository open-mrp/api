package accountep

import (
	"context"
	"fmt"

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

type AccountSvc interface {
	GetAccount(ctx context.Context, req *RetrieveAccountRequest) (*apiresource.Account, *apierror.APIError)
	GetAccountBySlug(ctx context.Context, req *RetrieveAccountBySlugRequest) (*apiresource.PublicAccount, *apierror.APIError)
	GetPortalProfileBySlug(ctx context.Context, req *RetrievePortalProfileRequest) (*apiresource.PortalProfile, *apierror.APIError)
	UpdateAccount(ctx context.Context, req *UpdateAccountRequest) (*apiresource.Account, *apierror.APIError)
	UploadAccountPhoto(ctx context.Context, req *UploadAccountPhotoRequest) (*apiresource.AccountPhotoUploadResult, *apierror.APIError)
	GetAccountLogoURL(ctx context.Context, req *GetAccountLogoURLRequest) (*apiresource.AccountLogoURL, *apierror.APIError)
}

type AccountSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type accountSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var accountSvcTracer = tracing.GetTracer("api-gateway.endpoints.accounts.service")

func (c *AccountSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("account endpoint service: core client is required")
	}
	return nil
}

func NewAccountSvc(config *AccountSvcConfig) AccountSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &accountSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *accountSvcImpl) GetAccount(ctx context.Context, req *RetrieveAccountRequest) (*apiresource.Account, *apierror.APIError) {
	pbReq := &pb.GetAccountRequest{
		Id: req.AccountID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountSvcTracer, "service.accounts.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountResponse, error) {
			return m.coreClient.GetAccount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := accountFromProto(resp.Account)
	stashAccountMeta(meta, resp.Account)
	return &result, nil
}

func (m *accountSvcImpl) GetAccountBySlug(ctx context.Context, req *RetrieveAccountBySlugRequest) (*apiresource.PublicAccount, *apierror.APIError) {
	pbReq := &pb.GetAccountBySlugRequest{
		Slug: req.Slug,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountSvcTracer, "service.accounts.get_by_slug", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountBySlugResponse, error) {
			return m.coreClient.GetAccountBySlug(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := publicAccountFromProto(resp.Account)
	return &result, nil
}

func (m *accountSvcImpl) GetPortalProfileBySlug(ctx context.Context, req *RetrievePortalProfileRequest) (*apiresource.PortalProfile, *apierror.APIError) {
	pbReq := &pb.GetPortalProfileBySlugRequest{
		Slug: req.Slug,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountSvcTracer, "service.accounts.get_portal_profile_by_slug", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPortalProfileBySlugResponse, error) {
			return m.coreClient.GetPortalProfileBySlug(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := portalProfileFromProto(resp.Profile)
	return &result, nil
}

func portalProfileFromProto(p *pb.PortalProfileInfo) apiresource.PortalProfile {
	if p == nil {
		return apiresource.PortalProfile{}
	}

	return apiresource.PortalProfile{
		ID:           p.Id,
		Object:       constants.ObjectTypePortalProfile,
		Name:         p.Name,
		Slug:         p.Slug,
		LogoURL:      p.LogoUrl,
		SupportEmail: p.SupportEmail,
		Address:      portalProfileAddressFromProto(p.Address),
	}
}

func portalProfileAddressFromProto(a *pb.AddressInfo) *apiresource.Address {
	if a == nil {
		return nil
	}
	addressType := constants.AddressTypeStandard
	if a.IsDropShip {
		addressType = constants.AddressTypeDropShip
	}
	return &apiresource.Address{
		ID:          a.Id,
		Object:      constants.ObjectTypeAddress,
		Name:        a.Name,
		Phone:       a.Phone,
		Email:       a.Email,
		Type:        addressType,
		Geolocation: portalProfileGeolocationFromProto(a.Geolocation),
		CreatedAt:   grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(a.UpdatedAt),
	}
}

func portalProfileGeolocationFromProto(g *pb.GeolocationInfo) *apiresource.Geolocation {
	if g == nil {
		return nil
	}
	return &apiresource.Geolocation{
		ID:          g.Id,
		Object:      constants.ObjectTypeGeolocation,
		StreetLine1: g.StreetLine_1,
		StreetLine2: g.StreetLine_2,
		Locality:    g.Locality,
		State:       g.State,
		PostalCode:  g.PostalCode,
		Country:     g.Country,
	}
}

func (m *accountSvcImpl) UpdateAccount(ctx context.Context, req *UpdateAccountRequest) (*apiresource.Account, *apierror.APIError) {
	pbReq := &pb.UpdateAccountRequest{
		Id:              req.AccountID,
		Name:            req.Name.Ptr(),
		SupportEmail:    req.SupportEmail.Ptr(),
		PhoneNumber:     req.PhoneNumber.Ptr(),
		Slug:            req.Slug.Ptr(),
		WebsiteUrl:      req.WebsiteURL.Ptr(),
		FacebookHandle:  req.FacebookHandle.Ptr(),
		InstagramHandle: req.InstagramHandle.Ptr(),
		LinkedinHandle:  req.LinkedInHandle.Ptr(),
		TwitterHandle:   req.TwitterHandle.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountSvcTracer, "service.accounts.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAccountResponse, error) {
			return m.coreClient.UpdateAccount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := accountFromProto(resp.Account)
	stashAccountMeta(meta, resp.Account)
	return &result, nil
}

func (m *accountSvcImpl) UploadAccountPhoto(ctx context.Context, req *UploadAccountPhotoRequest) (*apiresource.AccountPhotoUploadResult, *apierror.APIError) {
	pbReq := &pb.UploadAccountPhotoRequest{
		Id:          req.AccountID,
		File:        req.RawBody,
		ContentType: req.ContentType,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountSvcTracer, "service.accounts.upload_photo", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UploadAccountPhotoResponse, error) {
			return m.coreClient.UploadAccountPhoto(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.AccountPhotoUploadResult{
		Object:  constants.ObjectTypeAccountPhotoUploadResult,
		Success: resp.Success,
	}, nil
}

func (m *accountSvcImpl) GetAccountLogoURL(ctx context.Context, req *GetAccountLogoURLRequest) (*apiresource.AccountLogoURL, *apierror.APIError) {
	pbReq := &pb.GetAccountLogoURLRequest{
		Id: req.AccountID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountSvcTracer, "service.accounts.get_logo_url", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountLogoURLResponse, error) {
			return m.coreClient.GetAccountLogoURL(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.AccountLogoURL{
		Object: constants.ObjectTypeAccountLogoURL,
		URL:    resp.Url,
	}, nil
}

func accountFromProto(a *pb.AccountInfo) apiresource.Account {
	if a == nil {
		return apiresource.Account{}
	}

	return apiresource.Account{
		ID:        a.Id,
		Object:    constants.ObjectTypeAccount,
		Name:      a.Name,
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}
}

func stashAccountMeta(meta *resourcekit.LoadMeta, a *pb.AccountInfo) {
	if a == nil {
		return
	}

	if a.Branding != nil {
		meta.Set(constants.ObjectTypeAccount, a.Id, "branding", &apiresource.AccountBranding{
			ID:              a.Branding.Id,
			Object:          constants.ObjectTypeAccountBranding,
			SupportEmail:    a.Branding.SupportEmail,
			PhoneNumber:     a.Branding.PhoneNumber,
			LogoURL:         a.Branding.LogoUrl,
			FacebookHandle:  a.Branding.FacebookHandle,
			InstagramHandle: a.Branding.InstagramHandle,
			LinkedInHandle:  a.Branding.LinkedinHandle,
			TwitterHandle:   a.Branding.TwitterHandle,
			WebsiteURL:      a.Branding.WebsiteUrl,
			CreatedAt:       grpcutil.TimestampToTime(a.Branding.CreatedAt),
			UpdatedAt:       grpcutil.TimestampToTime(a.Branding.UpdatedAt),
		})
	}

	if a.Portal != nil {
		meta.Set(constants.ObjectTypeAccount, a.Id, "portal", &apiresource.AccountPortal{
			ID:        a.Portal.Id,
			Object:    constants.ObjectTypeAccountPortal,
			Slug:      a.Portal.Slug,
			CreatedAt: grpcutil.TimestampToTime(a.Portal.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(a.Portal.UpdatedAt),
		})
	}
}

func publicAccountFromProto(a *pb.PublicAccountInfo) apiresource.PublicAccount {
	if a == nil {
		return apiresource.PublicAccount{}
	}

	return apiresource.PublicAccount{
		ID:           a.Id,
		Object:       constants.ObjectTypePublicAccount,
		Name:         a.Name,
		Slug:         a.Slug,
		SupportEmail: a.SupportEmail,
		LogoURL:      a.LogoUrl,
	}
}
