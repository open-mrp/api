package grpc

import (
	"context"

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
}

func NewGRPCHandler(server *grpc.Server, accountSvc domain.AccountSvc) *gRPCHandler {
	handler := &gRPCHandler{
		accountSvc: accountSvc,
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
		IsSandbox:      accountContext.IsSandbox,
		OwnerAccountId: accountContext.OwnerAccountID,
		AccountMode:    accountMode,
	}, nil
}

func (h *gRPCHandler) GetUserAccountAccess(ctx context.Context, req *pb.GetUserAccountAccessRequest) (*pb.GetUserAccountAccessResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	access, apiErr := h.accountSvc.GetUserAccountAccess(ctx, req.UserId, req.AccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	if access == nil {
		return &pb.GetUserAccountAccessResponse{
			HasAccess: false,
			Access:    nil,
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
