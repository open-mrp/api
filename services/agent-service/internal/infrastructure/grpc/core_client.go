package grpc

import (
	"context"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/shared/actor"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/rpc"
	"github.com/open-mrp/apikit/tracing"

	grpclib "google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const coreServiceName = "core-service"

var coreClientTracer = tracing.GetTracer("agent-service.core_client")

type AgentCoreClient struct {
	grpcConn *contracts.GRPCClientConn
	client   pb.CoreServiceClient
	notes    pb.CoreNoteServiceClient
}

func NewAgentCoreClient(url string) (*AgentCoreClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: coreServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &AgentCoreClient{
		grpcConn: grpcConn,
		client:   pb.NewCoreServiceClient(grpcConn.Conn()),
		notes:    pb.NewCoreNoteServiceClient(grpcConn.Conn()),
	}, nil
}

func (c *AgentCoreClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *AgentCoreClient) Close() error {
	return c.grpcConn.Close()
}

func prepareCtx(ctx context.Context) context.Context {
	return rpc.PrepareServiceCallCtx(ctx)
}

func (c *AgentCoreClient) GetAccountContext(ctx context.Context, accountID string) (*domain.AccountContext, error) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.get_account_context", coreServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*pb.GetAccountContextResponse, error) {
			return c.client.GetAccountContext(ctx, &pb.GetAccountContextRequest{
				AccountId: accountID,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := &domain.AccountContext{
		IsSandbox:                    resp.IsSandbox,
		PlanCode:                     resp.PlanCode,
		AgentMonthlySpendingCapCents: resp.AgentMonthlySpendingCapCents,
	}
	if resp.OwnerAccountId != nil {
		result.OwnerAccountID = *resp.OwnerAccountId
	}

	return result, nil
}

func (c *AgentCoreClient) GetRolePermissions(ctx context.Context, roleID string) (map[string]bool, error) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.get_role_permissions", coreServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*pb.GetRolePermissionsResponse, error) {
			return c.client.GetRolePermissions(ctx, &pb.GetRolePermissionsRequest{
				RoleId: roleID,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return resp.Permissions, nil
}

func (c *AgentCoreClient) GetRoleInfo(ctx context.Context, roleID string) (*domain.RoleInfo, error) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.get_role_info", coreServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*pb.GetRoleInfoResponse, error) {
			return c.client.GetRoleInfo(ctx, &pb.GetRoleInfoRequest{
				RoleId: roleID,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.RoleInfo{
		ID:        resp.RoleId,
		RoleType:  resp.RoleTypeCode,
		AccountID: resp.AccountId,
	}, nil
}

func (c *AgentCoreClient) GetUserAccess(ctx context.Context, userID, accountID string) (*domain.UserAccess, error) {
	ctx = prepareCtx(ctx)
	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.get_user_account_access", coreServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*pb.GetUserAccountAccessResponse, error) {
			return c.client.GetUserAccountAccess(ctx, &pb.GetUserAccountAccessRequest{
				UserId:    userID,
				AccountId: accountID,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	if !resp.HasAccess || resp.Access == nil {
		return nil, nil
	}
	return &domain.UserAccess{
		RoleType:    resp.Access.GetRoleTypeCode(),
		Permissions: resp.Access.Permissions,
	}, nil
}

func (c *AgentCoreClient) RecallNotes(ctx context.Context, accountID string, recordIDs []string, limit int32) ([]domain.RecalledNote, error) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.recall_notes", coreServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*pb.RecallNotesResponse, error) {
			return c.notes.RecallNotes(ctx, &pb.RecallNotesRequest{AccountId: accountID, RecordIds: recordIDs, Limit: limit}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make([]domain.RecalledNote, len(resp.Notes))
	for i, n := range resp.Notes {
		out[i] = domain.RecalledNote{Type: n.Type, Importance: n.Importance, Body: n.Body}
		if n.Record != nil {
			out[i].RecordType, out[i].RecordID = n.Record.Type, n.Record.Id
		}
	}
	return out, nil
}

func (c *AgentCoreClient) ImportNotes(ctx context.Context, notes []domain.ImportedNote) (int64, error) {
	ctx = prepareCtx(ctx)

	in := make([]*pb.ImportNoteInput, len(notes))
	for i, n := range notes {
		p := &pb.ImportNoteInput{
			Id:         n.ID,
			AccountId:  n.AccountID,
			Body:       n.Body,
			Type:       n.Type,
			Importance: n.Importance,
			Actor:      actor.System().ToProto(),
			CreatedAt:  timestamppb.New(n.CreatedAt),
			UpdatedAt:  timestamppb.New(n.UpdatedAt),
		}
		if n.RecordID != "" {
			p.Record = &pb.NoteRecordRef{Type: n.RecordType, Id: n.RecordID}
		}
		if n.ExpiresAt != nil {
			p.ExpiresAt = timestamppb.New(*n.ExpiresAt)
		}
		in[i] = p
	}
	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.import_notes", coreServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*pb.ImportNotesResponse, error) {
			return c.notes.ImportNotes(ctx, &pb.ImportNotesRequest{Notes: in}, opts...)
		})
	if apiErr != nil {
		return 0, apiErr
	}
	return resp.Inserted, nil
}
