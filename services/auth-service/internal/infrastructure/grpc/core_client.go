package grpc

import (
	"context"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/rpc"
	"github.com/augno/api/shared/tracing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const coreServiceName = "core-service"

var coreClientTracer = tracing.GetTracer("auth-service.core_client")

type AuthCoreClient struct {
	grpcConn *contracts.GRPCClientConn
	client   pb.CoreServiceClient
}

func NewAuthCoreClient(url string) (*AuthCoreClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: coreServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &AuthCoreClient{
		grpcConn: grpcConn,
		client:   pb.NewCoreServiceClient(grpcConn.Conn()),
	}, nil
}

func (c *AuthCoreClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *AuthCoreClient) Close() error {
	return c.grpcConn.Close()
}

func prepareCtx(ctx context.Context) context.Context {
	return rpc.PrepareServiceCallCtx(ctx)
}

func (c *AuthCoreClient) GetAccountContext(ctx context.Context, accountID string) (*domain.AccountContext, *apierror.APIError) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.get_account_context", coreServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountContextResponse, error) {
			return c.client.GetAccountContext(ctx, &pb.GetAccountContextRequest{
				AccountId: accountID,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.AccountContext{
		AccountID:          accountID,
		OwnerAccountID:     resp.OwnerAccountId,
		AccountMode:        convertAccountModeFromProto(resp.AccountMode),
		SubscriptionStatus: resp.SubscriptionStatus,
	}, nil
}

func (c *AuthCoreClient) GetUserAccountAccess(ctx context.Context, userID, accountID string) (*domain.AccountUserAccess, bool, *apierror.APIError) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.get_user_account_access", coreServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetUserAccountAccessResponse, error) {
			return c.client.GetUserAccountAccess(ctx, &pb.GetUserAccountAccessRequest{
				UserId:    userID,
				AccountId: accountID,
			}, opts...)
		})
	if apiErr != nil {
		return nil, false, apiErr
	}

	if !resp.HasAccess || resp.Access == nil {
		return nil, false, nil
	}

	return &domain.AccountUserAccess{
		AccountUserID: resp.Access.AccountUserId,
		AccountID:     resp.Access.AccountId,
		RoleID:        resp.Access.RoleId,
		RoleType:      resp.Access.RoleTypeCode,
		RoleName:      resp.Access.RoleName,
		Permissions:   resp.Access.Permissions,
	}, true, nil
}

func (c *AuthCoreClient) GetAccountRelationByUserID(ctx context.Context, targetAccountID, actorAccountID, userID string) (*domain.AuthAccountRelation, bool, *apierror.APIError) {
	ctx = prepareCtx(ctx)

	req := &pb.GetAccountRelationRequest{
		OwnerAccountId: targetAccountID,
		Lookup:         &pb.GetAccountRelationRequest_UserId{UserId: userID},
	}
	if actorAccountID != "" {
		req.ActorAccountId = &actorAccountID
	}

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.get_account_relation_by_user_id", coreServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountRelationResponse, error) {
			return c.client.GetAccountRelation(ctx, req, opts...)
		})
	if apiErr != nil {
		return nil, false, apiErr
	}

	if !resp.HasRelation || resp.Relation == nil {
		return nil, false, nil
	}

	roleCode, ok := types.ParseIdentityRelationType(resp.Relation.RoleCode)
	if !ok {
		return nil, false, apierror.NewInternalError(nil, "invalid account relation role code")
	}

	return &domain.AuthAccountRelation{
		ID:                      resp.Relation.Id,
		CounterpartyAccountID:   resp.Relation.CounterpartyAccountId,
		AccountRelationRoleCode: roleCode,
		IsOwnerSide:             resp.Relation.IsOwnerSide,
	}, true, nil
}

func (c *AuthCoreClient) GetAccountRelationByAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*domain.AuthAccountRelation, bool, *apierror.APIError) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.get_account_relation_by_api_key_id", coreServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountRelationResponse, error) {
			return c.client.GetAccountRelation(ctx, &pb.GetAccountRelationRequest{
				OwnerAccountId: ownerAccountID,
				Lookup:         &pb.GetAccountRelationRequest_ApiKeyId{ApiKeyId: apiKeyID},
			}, opts...)
		})
	if apiErr != nil {
		return nil, false, apiErr
	}

	if !resp.HasRelation || resp.Relation == nil {
		return nil, false, nil
	}

	roleCode, ok := types.ParseIdentityRelationType(resp.Relation.RoleCode)
	if !ok {
		return nil, false, apierror.NewInternalError(nil, "invalid account relation role code")
	}

	return &domain.AuthAccountRelation{
		ID:                      resp.Relation.Id,
		CounterpartyAccountID:   resp.Relation.CounterpartyAccountId,
		AccountRelationRoleCode: roleCode,
		IsOwnerSide:             resp.Relation.IsOwnerSide,
	}, true, nil
}

func (c *AuthCoreClient) MarkAccountUserUsed(ctx context.Context, accountUserID string) *apierror.APIError {
	ctx = prepareCtx(ctx)

	_, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.mark_account_user_used", coreServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return c.client.MarkAccountUserUsed(ctx, &pb.MarkAccountUserUsedRequest{
				AccountUserId: accountUserID,
			}, opts...)
		})

	return apiErr
}

func (c *AuthCoreClient) GetRolePermissions(ctx context.Context, roleID string) (map[string]bool, *apierror.APIError) {
	if roleID == "" {
		return map[string]bool{}, nil
	}

	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.get_role_permissions", coreServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetRolePermissionsResponse, error) {
			return c.client.GetRolePermissions(ctx, &pb.GetRolePermissionsRequest{
				RoleId: roleID,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return resp.Permissions, nil
}

func (c *AuthCoreClient) GetSandboxAccountByOwner(ctx context.Context, ownerAccountID string) (string, *apierror.APIError) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.get_sandbox_account_by_owner", coreServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetSandboxAccountByOwnerResponse, error) {
			return c.client.GetSandboxAccountByOwner(ctx, &pb.GetSandboxAccountByOwnerRequest{
				OwnerAccountId: ownerAccountID,
			}, opts...)
		})
	if apiErr != nil {
		return "", apiErr
	}

	return resp.SandboxAccountId, nil
}

func (c *AuthCoreClient) GetAdminRole(ctx context.Context) (string, *apierror.APIError) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.get_admin_role", coreServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAdminRoleResponse, error) {
			return c.client.GetAdminRole(ctx, &emptypb.Empty{}, opts...)
		})
	if apiErr != nil {
		return "", apiErr
	}

	return resp.RoleId, nil
}

func (c *AuthCoreClient) CompleteRegistration(ctx context.Context, input domain.CompleteAccountRegistrationInput) (*domain.CompleteRegistrationOutput, *apierror.APIError) {
	ctx = prepareCtx(ctx)

	pbReq := &pb.CompleteRegistrationRequest{
		UserId:           input.UserID,
		PlanCode:         input.PlanCode,
		StripeCustomerId: input.StripeCustomerID,
		AccountData: &pb.RegistrationAccountData{
			AccountName: input.AccountName,
		},
	}

	if input.BusinessAddress != nil {
		pbReq.AccountData.BusinessAddress = &pb.RegistrationAddress{
			Line1:      &input.BusinessAddress.Line1,
			Line2:      &input.BusinessAddress.Line2,
			City:       &input.BusinessAddress.City,
			State:      &input.BusinessAddress.State,
			PostalCode: &input.BusinessAddress.PostalCode,
			Country:    &input.BusinessAddress.Country,
		}
	}

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.complete_registration", coreServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CompleteRegistrationResponse, error) {
			return c.client.CompleteRegistration(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.CompleteRegistrationOutput{
		AccountID: resp.AccountId,
		SandboxID: resp.SandboxId,
	}, nil
}

func convertAccountModeFromProto(m pb.AccountMode) constants.AccountMode {
	switch m {
	case pb.AccountMode_ACCOUNT_MODE_PRODUCTION:
		return constants.AccountModeProduction
	case pb.AccountMode_ACCOUNT_MODE_SANDBOX:
		return constants.AccountModeSandbox
	default:
		return constants.AccountModeProduction
	}
}
