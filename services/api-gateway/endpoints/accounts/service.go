package accountep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type AccountSvc interface {
	GetAccount(ctx context.Context, req *GetAccountRequest) (*apiresource.Account, *apierror.APIError)
	GetAccountBySlug(ctx context.Context, req *GetAccountBySlugRequest) (*apiresource.PublicAccount, *apierror.APIError)
	UpdateAccount(ctx context.Context, req *UpdateAccountRequest) (*apiresource.Account, *apierror.APIError)
	UploadAccountPhoto(ctx context.Context, req *UploadAccountPhotoRequest) (*apiresource.AccountPhotoUploadResult, *apierror.APIError)
	GetAccountLogoURL(ctx context.Context, req *GetAccountLogoURLRequest) (*apiresource.AccountLogoURL, *apierror.APIError)
}

type AccountSvcConfig struct {
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

func (m *accountSvcImpl) GetAccount(ctx context.Context, req *GetAccountRequest) (*apiresource.Account, *apierror.APIError) {
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

	result := AccountPresenter(resp.Account)
	return &result, nil
}

func (m *accountSvcImpl) GetAccountBySlug(ctx context.Context, req *GetAccountBySlugRequest) (*apiresource.PublicAccount, *apierror.APIError) {
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

	result := PublicAccountPresenter(resp.Account)
	return &result, nil
}

func (m *accountSvcImpl) UpdateAccount(ctx context.Context, req *UpdateAccountRequest) (*apiresource.Account, *apierror.APIError) {
	pbReq := &pb.UpdateAccountRequest{
		Id:              req.AccountID,
		Name:            req.Name,
		SupportEmail:    req.SupportEmail,
		PhoneNumber:     req.PhoneNumber,
		Slug:            req.Slug,
		WebsiteUrl:      req.WebsiteURL,
		FacebookHandle:  req.FacebookHandle,
		InstagramHandle: req.InstagramHandle,
		LinkedinHandle:  req.LinkedInHandle,
		TwitterHandle:   req.TwitterHandle,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountSvcTracer, "service.accounts.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAccountResponse, error) {
			return m.coreClient.UpdateAccount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := AccountPresenter(resp.Account)
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
		URL: resp.Url,
	}, nil
}
