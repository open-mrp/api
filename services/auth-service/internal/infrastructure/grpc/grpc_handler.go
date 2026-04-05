package grpc

import (
	"context"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/auth"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type gRPCHandler struct {
	pb.UnimplementedAuthServiceServer

	authSvc                domain.AuthSvc
	userSvc                domain.UserSvc
	tokenSvc               domain.TokenSvc
	passwordSvc            domain.PasswordSvc
	apiKeySvc              domain.APIKeySvc
	docAPIKeySvc           domain.DocAPIKeySvc
	registrationSessionSvc domain.RegistrationSessionSvc
}

func NewGRPCHandler(server *grpc.Server, authSvc domain.AuthSvc, userSvc domain.UserSvc, tokenSvc domain.TokenSvc, passwordSvc domain.PasswordSvc, apiKeySvc domain.APIKeySvc, docAPIKeySvc domain.DocAPIKeySvc, registrationSessionSvc domain.RegistrationSessionSvc) *gRPCHandler {
	handler := &gRPCHandler{
		authSvc:                authSvc,
		userSvc:                userSvc,
		tokenSvc:               tokenSvc,
		passwordSvc:            passwordSvc,
		apiKeySvc:              apiKeySvc,
		docAPIKeySvc:           docAPIKeySvc,
		registrationSessionSvc: registrationSessionSvc,
	}

	pb.RegisterAuthServiceServer(server, handler)
	return handler
}

func (h *gRPCHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.userSvc.Login(ctx, req.Identifier, req.Password)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User:         result.User.ToProto(),
	}, nil
}

func (h *gRPCHandler) ValidateCredential(ctx context.Context, cred *pb.Credential) (*pb.Identity, error) {
	if cred == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.authSvc.ValidateCredential(ctx, cred.Token, cred.TargetAccountId, cred.ActorAccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return result.ToProto(), nil
}

func (h *gRPCHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.LoginResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.userSvc.Register(ctx, domain.RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User:         result.User.ToProto(),
	}, nil
}

func (h *gRPCHandler) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.tokenSvc.RefreshToken(ctx, req.RefreshToken)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.RefreshTokenResponse{
		AccessToken: result.AccessToken,
	}, nil
}

func (h *gRPCHandler) RequestPasswordReset(ctx context.Context, req *pb.RequestPasswordResetRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.passwordSvc.RequestPasswordReset(ctx, req.Identifier, req.AccountSlug)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.LoginResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.passwordSvc.ResetPassword(ctx, req.Token, req.Password)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User:         result.User.ToProto(),
	}, nil
}

func (h *gRPCHandler) RevokeRefreshToken(ctx context.Context, req *pb.RevokeRefreshTokenRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.tokenSvc.RevokeRefreshToken(ctx, req.RefreshToken)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) UpdatePassword(ctx context.Context, req *pb.UpdatePasswordRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.passwordSvc.UpdatePassword(ctx, req.OldPassword, req.NewPassword)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) CreateAPIKey(ctx context.Context, req *pb.CreateAPIKeyRequest) (*pb.CreateAPIKeyResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		if !t.IsZero() {
			expiresAt = &t
		}
	}

	result, apiErr := h.apiKeySvc.CreateAPIKey(ctx, domain.CreateAPIKeyInput{
		RoleID:    req.RoleId,
		Name:      req.Name,
		ExpiresAt: expiresAt,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateAPIKeyResponse{
		ApiKeySecret: result.APIKeySecret,
		ApiKey:       result.APIKey.ToProto(),
	}, nil
}

func (h *gRPCHandler) ListAPIKeys(ctx context.Context, req *pb.ListAPIKeysRequest) (*pb.ListAPIKeysResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	statuses := make([]constants.APIKeyStatus, len(req.Statuses))
	for i, s := range req.Statuses {
		statuses[i] = constants.APIKeyStatus(s)
	}

	result, apiErr := h.apiKeySvc.ListAPIKeys(ctx, req.Cursor, req.Limit, req.Query, statuses, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbKeys := make([]*pb.APIKeyInfo, len(result.APIKeys))
	for i, key := range result.APIKeys {
		pbKeys[i] = key.ToProto()
	}

	return &pb.ListAPIKeysResponse{
		ApiKeys: pbKeys,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetAPIKey(ctx context.Context, req *pb.GetAPIKeyRequest) (*pb.GetAPIKeyResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.apiKeySvc.GetAPIKey(ctx, req.ApiKeyId, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAPIKeyResponse{
		ApiKey: result.ToProto(),
	}, nil
}

func (h *gRPCHandler) RevokeAPIKey(ctx context.Context, req *pb.RevokeAPIKeyRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.apiKeySvc.RevokeAPIKey(ctx, domain.RevokeAPIKeyInput{
		APIKeyID: req.ApiKeyId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) RotateAPIKey(ctx context.Context, req *pb.RotateAPIKeyRequest) (*pb.RotateAPIKeyResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	input := domain.RotateAPIKeyInput{
		APIKeyID: req.ApiKeyId,
	}
	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		if !t.IsZero() {
			input.ExpiresAt = &t
		}
	}

	result, apiErr := h.apiKeySvc.RotateAPIKey(ctx, input)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.RotateAPIKeyResponse{
		ApiKeySecret: result.APIKeySecret,
		ApiKey:       result.APIKey.ToProto(),
	}, nil
}

func (h *gRPCHandler) GetOrCreateDocAPIKey(ctx context.Context, req *pb.GetOrCreateDocAPIKeyRequest) (*pb.GetOrCreateDocAPIKeyResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.docAPIKeySvc.GetOrCreateDocAPIKey(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetOrCreateDocAPIKeyResponse{
		ApiKeySecret: result.APIKeySecret,
		ApiKey:       result.APIKey.ToProto(),
	}, nil
}

func (h *gRPCHandler) CreateRegistrationSession(ctx context.Context, req *pb.CreateRegistrationSessionRequest) (*pb.CreateRegistrationSessionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.registrationSessionSvc.CreateSession(ctx, domain.CreateRegistrationSessionInput{
		Email:    req.Email,
		PlanCode: req.PlanCode,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateRegistrationSessionResponse{
		SessionId: result.SessionID,
	}, nil
}

func (h *gRPCHandler) CreateUserForRegistration(ctx context.Context, req *pb.CreateUserForRegistrationRequest) (*pb.CreateUserForRegistrationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.registrationSessionSvc.CreateUserForSession(ctx, domain.CreateUserForRegistrationInput{
		SessionID: req.SessionId,
		Name:      req.Name,
		Password:  req.Password,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateUserForRegistrationResponse{
		UserId:       result.UserID,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}, nil
}

func (h *gRPCHandler) GetRegistrationSession(ctx context.Context, req *pb.GetRegistrationSessionRequest) (*pb.GetRegistrationSessionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.registrationSessionSvc.GetSession(ctx, req.SessionId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetRegistrationSessionResponse{
		Session: result.ToProto(),
	}, nil
}

func (h *gRPCHandler) VerifyRegistrationToken(ctx context.Context, req *pb.VerifyRegistrationTokenRequest) (*pb.VerifyRegistrationTokenResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.registrationSessionSvc.VerifyToken(ctx, req.Token)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.VerifyRegistrationTokenResponse{
		Session: result.ToProto(),
	}, nil
}

func (h *gRPCHandler) UpdateRegistrationSession(ctx context.Context, req *pb.UpdateRegistrationSessionRequest) (*pb.UpdateRegistrationSessionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	// Only pass step and session_data — sensitive fields (stripe_customer_id,
	// payment_completed, stripe_subscription_id) are ignored. Payment state
	// should only be modified via SetupBilling.
	input := domain.UpdateRegistrationSessionInput{
		SessionID: req.SessionId,
	}

	if req.Step != nil {
		step := constants.RegistrationStep(*req.Step)
		input.Step = &step
	}

	if req.SessionData != nil {
		input.SessionData = &domain.UpdateRegistrationSessionData{
			UserName:                 req.SessionData.UserName,
			AccountName:              req.SessionData.AccountName,
			BillingAddressLine1:      req.SessionData.BillingAddressLine1,
			BillingAddressLine2:      req.SessionData.BillingAddressLine2,
			BillingAddressCity:       req.SessionData.BillingAddressCity,
			BillingAddressState:      req.SessionData.BillingAddressState,
			BillingAddressPostalCode: req.SessionData.BillingAddressPostalCode,
			BillingAddressCountry:    req.SessionData.BillingAddressCountry,
		}
	}

	result, apiErr := h.registrationSessionSvc.UpdateSession(ctx, input)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateRegistrationSessionResponse{
		Session: result.ToProto(),
	}, nil
}

func (h *gRPCHandler) ListRegistrationSessions(ctx context.Context, req *pb.ListRegistrationSessionsRequest) (*pb.ListRegistrationSessionsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.registrationSessionSvc.ListSessions(ctx, domain.ListRegistrationSessionsInput{
		Cursor: req.Cursor,
		Limit:  req.Limit,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbSessions := make([]*pb.RegistrationSessionInfo, len(result.Sessions))
	for i, s := range result.Sessions {
		pbSessions[i] = s.ToProto()
	}

	return &pb.ListRegistrationSessionsResponse{
		Sessions: pbSessions,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) ResendVerificationEmail(ctx context.Context, req *pb.ResendVerificationEmailRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.registrationSessionSvc.ResendVerificationEmail(ctx, req.SessionId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) CompleteRegistration(ctx context.Context, req *pb.CompleteRegistrationRequest) (*pb.CompleteRegistrationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.registrationSessionSvc.CompleteRegistration(ctx, req.SessionId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CompleteRegistrationResponse{
		AccountId: result.AccountID,
		SandboxId: result.SandboxID,
	}, nil
}

func (h *gRPCHandler) SetupRegistrationBilling(ctx context.Context, req *pb.SetupRegistrationBillingRequest) (*pb.SetupRegistrationBillingResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.registrationSessionSvc.SetupBilling(ctx, domain.SetupBillingInput{
		SessionID: req.SessionId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.SetupRegistrationBillingResponse{
		StripeCustomerId: result.StripeCustomerID,
		ClientSecret:     result.ClientSecret,
		PublishableKey:   result.PublishableKey,
	}, nil
}

func (h *gRPCHandler) ConfirmRegistrationPayment(ctx context.Context, req *pb.ConfirmRegistrationPaymentRequest) (*pb.ConfirmRegistrationPaymentResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.registrationSessionSvc.ConfirmPayment(ctx, domain.ConfirmPaymentInput{
		SessionID:     req.SessionId,
		SetupIntentID: req.SetupIntentId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ConfirmRegistrationPaymentResponse{
		Status:          result.Status,
		PaymentMethodId: result.PaymentMethodID,
	}, nil
}
