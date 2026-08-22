package regsessionep

import (
	"context"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/api-gateway/internal/cookie"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/auth"
	"github.com/open-mrp/api/shared/ptrutil"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RegistrationSessionSvc interface {
	CreateSession(ctx context.Context, req *CreateRegistrationSessionRequest) (*apiresource.CreateSessionResponse, *apierror.APIError)
	ResendVerificationEmail(ctx context.Context, req *ResendEmailRequest) (*apiresource.EmptyResource, *apierror.APIError)
	VerifyToken(ctx context.Context, req *VerifyTokenRequest) (*apiresource.RegistrationSession, *apierror.APIError)
	GetSession(ctx context.Context, req *RetrieveSessionRequest) (*apiresource.RegistrationSession, *apierror.APIError)
	CreateUser(ctx context.Context, req *CreateUserRequest) (*apiresource.CreateUserResponse, *apierror.APIError)
	UpdateSession(ctx context.Context, req *UpdateSessionRequest) (*apiresource.RegistrationSession, *apierror.APIError)
	ListSessions(ctx context.Context, req *apiresource.PaginationRequest) (*apiresource.List[apiresource.RegistrationSession], *apierror.APIError)
	SetupBilling(ctx context.Context, req *SetupBillingRequest) (*apiresource.SetupBillingResponse, *apierror.APIError)
	ConfirmPayment(ctx context.Context, req *ConfirmPaymentRequest) (*apiresource.ConfirmPaymentResponse, *apierror.APIError)
	CompleteRegistration(ctx context.Context, req *CompleteRegistrationRequest) (*apiresource.CompleteRegistrationResponse, *apierror.APIError)
}

type RegistrationSessionSvcConfig struct {
	// AuthClient (required) is the auth-service gRPC client.
	AuthClient pb.AuthServiceClient
}

type registrationSessionSvcImpl struct {
	authClient pb.AuthServiceClient
}

var registrationSessionSvcTracer = tracing.GetTracer("api-gateway.endpoints.registration_sessions.service")

func (c *RegistrationSessionSvcConfig) validate() error {
	if c.AuthClient == nil {
		return fmt.Errorf("registration session endpoint service: auth client is required")
	}
	return nil
}

func NewRegistrationSessionSvc(config *RegistrationSessionSvcConfig) RegistrationSessionSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &registrationSessionSvcImpl{
		authClient: config.AuthClient,
	}
}

func (m *registrationSessionSvcImpl) CreateSession(ctx context.Context, req *CreateRegistrationSessionRequest) (*apiresource.CreateSessionResponse, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, registrationSessionSvcTracer, "service.registration_sessions.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateRegistrationSessionResponse, error) {
			return m.authClient.CreateRegistrationSession(ctx, &pb.CreateRegistrationSessionRequest{
				Email:    req.Email,
				PlanCode: string(req.PlanCode),
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.CreateSessionResponse{
		ID:     resp.SessionId,
		Object: constants.ObjectTypeRegistrationSession,
	}, nil
}

func (m *registrationSessionSvcImpl) ResendVerificationEmail(ctx context.Context, req *ResendEmailRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, registrationSessionSvcTracer, "service.registration_sessions.resend_verification_email", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.authClient.ResendVerificationEmail(ctx, &pb.ResendVerificationEmailRequest{
				SessionId: req.SessionID,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *registrationSessionSvcImpl) GetSession(ctx context.Context, req *RetrieveSessionRequest) (*apiresource.RegistrationSession, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, registrationSessionSvcTracer, "service.registration_sessions.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetRegistrationSessionResponse, error) {
			return m.authClient.GetRegistrationSession(ctx, &pb.GetRegistrationSessionRequest{
				SessionId: req.SessionID,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return mapProtoToRegistrationSession(resp.Session), nil
}

func (m *registrationSessionSvcImpl) VerifyToken(ctx context.Context, req *VerifyTokenRequest) (*apiresource.RegistrationSession, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, registrationSessionSvcTracer, "service.registration_sessions.verify_token", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.VerifyRegistrationTokenResponse, error) {
			return m.authClient.VerifyRegistrationToken(ctx, &pb.VerifyRegistrationTokenRequest{
				Token: req.Token,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return mapProtoToRegistrationSession(resp.Session), nil
}

func (m *registrationSessionSvcImpl) CreateUser(ctx context.Context, req *CreateUserRequest) (*apiresource.CreateUserResponse, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, registrationSessionSvcTracer, "service.registration_sessions.create_user", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateUserForRegistrationResponse, error) {
			return m.authClient.CreateUserForRegistration(ctx, &pb.CreateUserForRegistrationRequest{
				SessionId: req.SessionID,
				Name:      req.Name,
				Password:  req.Password,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	appctx.AddCookies(ctx, cookie.MakeAuthCookies(ctx, resp.AccessToken, resp.RefreshToken))

	return &apiresource.CreateUserResponse{
		ID:     resp.UserId,
		Object: constants.ObjectTypeUser,
	}, nil
}

func (m *registrationSessionSvcImpl) UpdateSession(ctx context.Context, req *UpdateSessionRequest) (*apiresource.RegistrationSession, *apierror.APIError) {
	pbReq := &pb.UpdateRegistrationSessionRequest{
		SessionId: req.SessionID,
	}

	if step, ok := req.Step.Value(); ok {
		s := string(step)
		pbReq.Step = &s
	}

	if sessionData, ok := req.SessionData.Value(); ok {
		pbReq.SessionData = &pb.RegistrationSessionData{
			UserName:                 sessionData.UserName.Ptr(),
			AccountName:              sessionData.AccountName.Ptr(),
			BillingAddressLine1:      sessionData.BillingAddressLine1.Ptr(),
			BillingAddressLine2:      sessionData.BillingAddressLine2.Ptr(),
			BillingAddressCity:       sessionData.BillingAddressCity.Ptr(),
			BillingAddressState:      sessionData.BillingAddressState.Ptr(),
			BillingAddressPostalCode: sessionData.BillingAddressPostalCode.Ptr(),
			BillingAddressCountry:    sessionData.BillingAddressCountry.Ptr(),
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, registrationSessionSvcTracer, "service.registration_sessions.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateRegistrationSessionResponse, error) {
			return m.authClient.UpdateRegistrationSession(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return mapProtoToRegistrationSession(resp.Session), nil
}

func (m *registrationSessionSvcImpl) ListSessions(ctx context.Context, req *apiresource.PaginationRequest) (*apiresource.List[apiresource.RegistrationSession], *apierror.APIError) {
	pbReq := &pb.ListRegistrationSessionsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, registrationSessionSvcTracer, "service.registration_sessions.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListRegistrationSessionsResponse, error) {
			return m.authClient.ListRegistrationSessions(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	sessions := make([]apiresource.RegistrationSession, len(resp.Sessions))
	for i, s := range resp.Sessions {
		sessions[i] = *mapProtoToRegistrationSession(s)
	}

	return apiresource.NewList(sessions, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *registrationSessionSvcImpl) SetupBilling(ctx context.Context, req *SetupBillingRequest) (*apiresource.SetupBillingResponse, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, registrationSessionSvcTracer, "service.registration_sessions.setup_billing", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SetupRegistrationBillingResponse, error) {
			return m.authClient.SetupRegistrationBilling(ctx, &pb.SetupRegistrationBillingRequest{
				SessionId: req.SessionID,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.SetupBillingResponse{
		Object:           constants.ObjectTypeSetupBillingResponse,
		StripeCustomerID: resp.StripeCustomerId,
		ClientSecret:     resp.ClientSecret,
		PublishableKey:   resp.PublishableKey,
	}, nil
}

func (m *registrationSessionSvcImpl) ConfirmPayment(ctx context.Context, req *ConfirmPaymentRequest) (*apiresource.ConfirmPaymentResponse, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, registrationSessionSvcTracer, "service.registration_sessions.confirm_payment", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ConfirmRegistrationPaymentResponse, error) {
			return m.authClient.ConfirmRegistrationPayment(ctx, &pb.ConfirmRegistrationPaymentRequest{
				SessionId:     req.SessionID,
				SetupIntentId: req.SetupIntentID,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.ConfirmPaymentResponse{
		Object:          constants.ObjectTypeConfirmPaymentResponse,
		Status:          resp.Status,
		PaymentMethodID: resp.PaymentMethodId,
	}, nil
}

func (m *registrationSessionSvcImpl) CompleteRegistration(ctx context.Context, req *CompleteRegistrationRequest) (*apiresource.CompleteRegistrationResponse, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, registrationSessionSvcTracer, "service.registration_sessions.complete_registration", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CompleteRegistrationResponse, error) {
			return m.authClient.CompleteRegistration(ctx, &pb.CompleteRegistrationRequest{
				SessionId: req.SessionID,
			}, opts...)
		}, grpcutil.WithTimeout(grpcutil.BillingOperationTimeout))

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.CompleteRegistrationResponse{
		ID:     resp.AccountId,
		Object: constants.ObjectTypeAccount,
	}, nil
}

func mapProtoToRegistrationSession(s *pb.RegistrationSessionInfo) *apiresource.RegistrationSession {
	if s == nil {
		return nil
	}

	// Build user
	user := apiresource.RegistrationSessionUser{
		Object: constants.ObjectTypeUser,
		Email:  s.Email,
		ID:     s.UserId,
	}
	if s.IsEmailVerified && s.UpdatedAt != nil {
		t := s.UpdatedAt.AsTime()
		user.EmailVerifiedAt = &t
	}
	if s.SessionData != nil && s.SessionData.UserName != nil && *s.SessionData.UserName != "" {
		user.Name = s.SessionData.UserName
	}

	// Build account (only if account_id or session data has account info)
	var account *apiresource.RegistrationSessionAccount
	if s.AccountId != nil || (s.SessionData != nil && s.SessionData.AccountName != nil && *s.SessionData.AccountName != "") {
		accountName := ""
		if s.SessionData != nil && s.SessionData.AccountName != nil {
			accountName = *s.SessionData.AccountName
		}

		addr := apiresource.RegistrationSessionAddress{
			Object: constants.ObjectTypeAddress,
		}
		if s.SessionData != nil {
			addr.Line1 = new(ptrutil.ValOrDefault(s.SessionData.BillingAddressLine1, ""))
			addr.Line2 = new(ptrutil.ValOrDefault(s.SessionData.BillingAddressLine2, ""))
			addr.City = new(ptrutil.ValOrDefault(s.SessionData.BillingAddressCity, ""))
			addr.State = new(ptrutil.ValOrDefault(s.SessionData.BillingAddressState, ""))
			addr.PostalCode = new(ptrutil.ValOrDefault(s.SessionData.BillingAddressPostalCode, ""))
			addr.Country = new(ptrutil.ValOrDefault(s.SessionData.BillingAddressCountry, ""))
		}

		account = &apiresource.RegistrationSessionAccount{
			ID:             s.AccountId,
			Object:         constants.ObjectTypeAccount,
			Name:           accountName,
			BillingAddress: addr,
		}
	}

	var completedAt *time.Time
	if s.CompletedAt != nil {
		t := s.CompletedAt.AsTime()
		completedAt = &t
	}

	return &apiresource.RegistrationSession{
		ID:                      s.Id,
		Object:                  constants.ObjectTypeRegistrationSession,
		PlanCode:                s.PlanCode,
		Step:                    constants.RegistrationStep(s.Step),
		StripeCustomerID:        s.StripeCustomerId,
		StripeCheckoutSessionID: s.StripeCheckoutSessionId,
		PaymentCompleted:        s.PaymentCompleted,
		User:                    user,
		Account:                 account,
		CompletedAt:             completedAt,
		CreatedAt:               s.CreatedAt.AsTime(),
		UpdatedAt:               s.UpdatedAt.AsTime(),
	}
}
