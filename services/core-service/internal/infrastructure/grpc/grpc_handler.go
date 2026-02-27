package grpc

import (
	"context"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type gRPCHandler struct {
	pb.UnimplementedCoreServiceServer

	accountSvc domain.AccountSvc
	sandboxSvc domain.SandboxSvc
	unitSvc    domain.UnitSvc
}

func NewGRPCHandler(server *grpc.Server, accountSvc domain.AccountSvc, sandboxSvc domain.SandboxSvc, unitSvc domain.UnitSvc) *gRPCHandler {
	handler := &gRPCHandler{
		accountSvc: accountSvc,
		sandboxSvc: sandboxSvc,
		unitSvc:    unitSvc,
	}

	pb.RegisterCoreServiceServer(server, handler)
	return handler
}

func (h *gRPCHandler) GetAccountContext(ctx context.Context, req *pb.GetAccountContextRequest) (*pb.GetAccountContextResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	accountContext, apiErr := h.accountSvc.GetAccountContext(ctx, req.AccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	var accountMode pb.AccountMode
	switch accountContext.AccountMode {
	case constants.AccountModeProduction:
		accountMode = pb.AccountMode_ACCOUNT_MODE_PRODUCTION
	case constants.AccountModeSandbox:
		accountMode = pb.AccountMode_ACCOUNT_MODE_SANDBOX
	default:
		accountMode = pb.AccountMode_ACCOUNT_MODE_UNSPECIFIED
	}

	return &pb.GetAccountContextResponse{
		IsSandbox:          accountContext.IsSandbox,
		OwnerAccountId:     accountContext.OwnerAccountID,
		AccountMode:        accountMode,
		SubscriptionStatus: accountContext.SubscriptionStatus,
	}, nil
}

func (h *gRPCHandler) GetUserAccountAccess(ctx context.Context, req *pb.GetUserAccountAccessRequest) (*pb.GetUserAccountAccessResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	access, hasAccess, apiErr := h.accountSvc.GetUserAccountAccess(ctx, req.UserId, req.AccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	if !hasAccess {
		return &pb.GetUserAccountAccessResponse{
			HasAccess: false,
		}, nil
	}

	var lastUsedAt *timestamppb.Timestamp
	if access.LastUsedAt != nil {
		lastUsedAt = timestamppb.New(*access.LastUsedAt)
	}

	return &pb.GetUserAccountAccessResponse{
		HasAccess: true,
		Access: &pb.AccountUserAccess{
			AccountUserId: access.AccountUserID,
			AccountId:     access.AccountID,
			RoleId:        access.RoleID,
			RoleTypeCode:  access.RoleTypeCode,
			Permissions:   access.Permissions,
			LastUsedAt:    lastUsedAt,
		},
	}, nil
}

func (h *gRPCHandler) GetRolePermissions(ctx context.Context, req *pb.GetRolePermissionsRequest) (*pb.GetRolePermissionsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	permissions, apiErr := h.accountSvc.GetRolePermissions(ctx, req.RoleId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetRolePermissionsResponse{
		Permissions: permissions,
	}, nil
}

func (h *gRPCHandler) GetAccountRelation(ctx context.Context, req *pb.GetAccountRelationRequest) (*pb.GetAccountRelationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	var relation *domain.AccountRelation
	var apiErr *apierror.APIError

	switch lookup := req.Lookup.(type) {
	case *pb.GetAccountRelationRequest_UserId:
		relation, apiErr = h.accountSvc.GetAccountRelationByUserID(ctx, req.OwnerAccountId, lookup.UserId)
	case *pb.GetAccountRelationRequest_ApiKeyId:
		relation, apiErr = h.accountSvc.GetAccountRelationByAPIKeyID(ctx, req.OwnerAccountId, lookup.ApiKeyId)
	default:
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewParameterMissingError("Either user_id or api_key_id must be provided", "lookup"))
	}

	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	if relation == nil {
		return &pb.GetAccountRelationResponse{
			HasRelation: false,
			Relation:    nil,
		}, nil
	}

	return &pb.GetAccountRelationResponse{
		HasRelation: true,
		Relation: &pb.AccountRelation{
			Id:                    relation.ID,
			CounterpartyAccountId: relation.CounterpartyAccountID,
			RoleCode:              relation.RoleCode,
		},
	}, nil
}

func (h *gRPCHandler) MarkAccountUserUsed(ctx context.Context, req *pb.MarkAccountUserUsedRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.accountSvc.MarkAccountUserUsed(ctx, req.AccountUserId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) ListUserAccountAffiliations(ctx context.Context, req *pb.ListUserAccountAffiliationsRequest) (*pb.ListUserAccountAffiliationsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	affiliations, lastUsedAccountID, apiErr := h.accountSvc.ListUserAccountAffiliations(ctx, req.UserId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbAffiliations := make([]*pb.AccountAffiliation, len(affiliations))
	for i, aff := range affiliations {
		var lastUsedAt *timestamppb.Timestamp
		if aff.LastUsedAt != nil {
			lastUsedAt = timestamppb.New(*aff.LastUsedAt)
		}

		pbAffiliations[i] = &pb.AccountAffiliation{
			AccountId:   aff.AccountID,
			AccountName: aff.AccountName,
			RoleId:      aff.RoleID,
			RoleName:    aff.RoleName,
			LastUsedAt:  lastUsedAt,
		}
	}

	return &pb.ListUserAccountAffiliationsResponse{
		Affiliations:      pbAffiliations,
		LastUsedAccountId: lastUsedAccountID,
	}, nil
}

func (h *gRPCHandler) GetSandboxAccountByOwner(ctx context.Context, req *pb.GetSandboxAccountByOwnerRequest) (*pb.GetSandboxAccountByOwnerResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	sandboxAccountID, apiErr := h.sandboxSvc.GetSandboxAccountByOwner(ctx, req.OwnerAccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetSandboxAccountByOwnerResponse{
		SandboxAccountId: sandboxAccountID,
	}, nil
}

func (h *gRPCHandler) ListSandboxAccounts(ctx context.Context, req *pb.ListSandboxAccountsRequest) (*pb.ListSandboxAccountsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.sandboxSvc.ListSandboxAccounts(ctx, req.Cursor, req.Limit)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbSandboxes := make([]*pb.SandboxInfo, len(result.Sandboxes))
	for i, s := range result.Sandboxes {
		pbSandboxes[i] = &pb.SandboxInfo{
			Id:        s.TypeID,
			Name:      s.Name,
			AccountId: s.AccountID,
			CreatedAt: timestamppb.New(s.CreatedAt),
			UpdatedAt: timestamppb.New(s.UpdatedAt),
		}
	}

	return &pb.ListSandboxAccountsResponse{
		Sandboxes: pbSandboxes,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetAdminRole(ctx context.Context, req *emptypb.Empty) (*pb.GetAdminRoleResponse, error) {
	roleID, apiErr := h.accountSvc.GetAdminRole(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAdminRoleResponse{
		RoleId: roleID,
	}, nil
}

func (h *gRPCHandler) GetSandbox(ctx context.Context, req *pb.GetSandboxRequest) (*pb.GetSandboxResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	sandbox, apiErr := h.sandboxSvc.GetSandbox(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetSandboxResponse{
		Sandbox: &pb.SandboxInfo{
			Id:        sandbox.TypeID,
			Name:      sandbox.Name,
			AccountId: sandbox.AccountID,
			CreatedAt: timestamppb.New(sandbox.CreatedAt),
			UpdatedAt: timestamppb.New(sandbox.UpdatedAt),
		},
	}, nil
}

func (h *gRPCHandler) DeleteSandbox(ctx context.Context, req *pb.DeleteSandboxRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.sandboxSvc.DeleteSandbox(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) CreateSandbox(ctx context.Context, req *pb.CreateSandboxRequest) (*pb.CreateSandboxResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	var mode constants.SandboxMode
	switch req.Mode {
	case pb.SandboxMode_SANDBOX_MODE_SEEDED:
		mode = constants.SandboxModeSeeded
	default:
		mode = constants.SandboxModeBlank
	}

	sandbox, apiErr := h.sandboxSvc.CreateSandbox(ctx, req.Name, mode)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateSandboxResponse{
		Sandbox: &pb.SandboxInfo{
			Id:        sandbox.TypeID,
			Name:      sandbox.Name,
			AccountId: sandbox.AccountID,
			CreatedAt: timestamppb.New(sandbox.CreatedAt),
			UpdatedAt: timestamppb.New(sandbox.UpdatedAt),
		},
	}, nil
}

func (h *gRPCHandler) UpdateAccountSubscription(ctx context.Context, req *pb.UpdateAccountSubscriptionRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	var periodEnd *time.Time
	if req.CurrentPeriodEnd != nil {
		t := req.CurrentPeriodEnd.AsTime()
		periodEnd = &t
	}

	apiErr := h.accountSvc.UpdateAccountSubscription(ctx, req.AccountId, req.SubscriptionStatus, req.PlanCode, req.StripeSubscriptionId, periodEnd, req.StripeCustomerId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) ClearAccountStripeCustomer(ctx context.Context, req *pb.ClearAccountStripeCustomerRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.accountSvc.ClearAccountStripeCustomer(ctx, req.AccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) GetAccountByStripeCustomerID(ctx context.Context, req *pb.GetAccountByStripeCustomerIDRequest) (*pb.GetAccountByStripeCustomerIDResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	accountID, planCode, apiErr := h.accountSvc.GetAccountByStripeCustomerID(ctx, req.StripeCustomerId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAccountByStripeCustomerIDResponse{
		AccountId: accountID,
		PlanCode:  planCode,
	}, nil
}

func (h *gRPCHandler) CompleteRegistration(ctx context.Context, req *pb.CompleteRegistrationRequest) (*pb.CompleteRegistrationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	input := domain.CompleteRegistrationInput{
		UserID:           req.UserId,
		PlanCode:         req.PlanCode,
		StripeCustomerID: req.StripeCustomerId,
	}

	if req.StripeSubscriptionId != nil {
		input.StripeSubscriptionID = *req.StripeSubscriptionId
	}

	if req.AccountData != nil {
		input.AccountData = domain.RegistrationAccountData{
			AccountName: req.AccountData.AccountName,
		}

		if req.AccountData.BusinessAddress != nil {
			addr := req.AccountData.BusinessAddress
			input.BusinessAddress = &domain.RegistrationAddress{
				Line1:      derefStr(addr.Line1),
				Line2:      derefStr(addr.Line2),
				City:       derefStr(addr.City),
				State:      derefStr(addr.State),
				PostalCode: derefStr(addr.PostalCode),
				Country:    derefStr(addr.Country),
			}
		}
	}

	result, apiErr := h.accountSvc.CompleteRegistration(ctx, input)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CompleteRegistrationResponse{
		AccountId: result.AccountID,
		SandboxId: result.SandboxID,
	}, nil
}

func (h *gRPCHandler) ListUnits(ctx context.Context, req *pb.ListUnitsRequest) (*pb.ListUnitsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListUnitsParams{
		Cursor:       req.Cursor,
		Limit:        req.Limit,
		Query:        req.Query,
		Type:         req.Type,
		UnitGroupIDs: req.UnitGroupIds,
	}

	result, apiErr := h.unitSvc.ListUnits(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbUnits := make([]*pb.UnitInfo, len(result.Units))
	for i, u := range result.Units {
		pbUnits[i] = &pb.UnitInfo{
			Id:               u.ID,
			Name:             u.Name,
			Abbreviation:     u.Abbreviation,
			Type:             u.UnitDimensionCode,
			RatioNumerator:   u.RatioNumerator,
			RatioDenominator: u.RatioDenominator,
			OffsetNumerator:  u.OffsetNumerator,
			OffsetDenominator: u.OffsetDenominator,
			IsBaseUnit:       u.IsBaseUnit,
			IsInternal:       u.AccountID != nil,
			CreatedAt:        timestamppb.New(u.CreatedAt),
			UpdatedAt:        timestamppb.New(u.UpdatedAt),
		}
	}

	return &pb.ListUnitsResponse{
		Units: pbUnits,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
