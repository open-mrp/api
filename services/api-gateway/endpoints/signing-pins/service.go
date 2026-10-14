package signingpinep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/auth"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const samplePin = "2580"

const sampleNewPin = "8642"

type SigningPinSvc interface {
	SetSigningPin(ctx context.Context, req *SetSigningPinRequest) (*apiresource.EmptyResource, *apierror.APIError)
	SetTemporarySigningPin(ctx context.Context, req *SetTemporarySigningPinRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ClearSigningPin(ctx context.Context, req *ClearSigningPinRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ChangeSigningPin(ctx context.Context, req *ChangeSigningPinRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type SigningPinSvcConfig struct {
	// AuthClient (required) is the auth-service gRPC client.
	AuthClient pb.AuthServiceClient
}

func (c *SigningPinSvcConfig) validate() error {
	if c.AuthClient == nil {
		return fmt.Errorf("signing pin endpoint service: auth client is required")
	}
	return nil
}

type signingPinSvcImpl struct {
	authClient pb.AuthServiceClient
}

var signingPinSvcTracer = tracing.GetTracer("api-gateway.endpoints.signing_pins.service")

func NewSigningPinSvc(config *SigningPinSvcConfig) SigningPinSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &signingPinSvcImpl{authClient: config.AuthClient}
}

func (s *signingPinSvcImpl) SetSigningPin(ctx context.Context, req *SetSigningPinRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, signingPinSvcTracer, "service.signing_pins.set", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return s.authClient.SetSigningPin(ctx, &pb.SetSigningPinRequest{Pin: req.Pin}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

func (s *signingPinSvcImpl) SetTemporarySigningPin(ctx context.Context, req *SetTemporarySigningPinRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, signingPinSvcTracer, "service.signing_pins.set_temporary", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return s.authClient.SetTemporarySigningPin(ctx, &pb.SetTemporarySigningPinRequest{
				AccountUserId: req.AccountUserID,
				Pin:           req.Pin,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

func (s *signingPinSvcImpl) ClearSigningPin(ctx context.Context, req *ClearSigningPinRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, signingPinSvcTracer, "service.signing_pins.clear", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return s.authClient.ClearSigningPin(ctx, &pb.ClearSigningPinRequest{AccountUserId: req.AccountUserID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

func (s *signingPinSvcImpl) ChangeSigningPin(ctx context.Context, req *ChangeSigningPinRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, signingPinSvcTracer, "service.signing_pins.change", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return s.authClient.ChangeSigningPin(ctx, &pb.ChangeSigningPinRequest{
				AccountUserId: req.AccountUserID,
				CurrentPin:    req.CurrentPin,
				NewPin:        req.NewPin,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}
