package blockep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/chatmap"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/notification"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

// BlockSvc backs the message block endpoints via the notification-service ChatService gRPC client.
type BlockSvc interface {
	Block(ctx context.Context, req *BlockRequest) (*apiresource.MessagingBlock, *apierror.APIError)
	Unblock(ctx context.Context, req *UnblockRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ListBlocks(ctx context.Context, req *ListBlocksRequest) (*apiresource.List[apiresource.MessagingBlock], *apierror.APIError)
}

type BlockSvcConfig struct {
	// ChatClient (required) is the notification-service ChatService gRPC client.
	ChatClient pb.ChatServiceClient
}

type blockSvcImpl struct {
	chatClient pb.ChatServiceClient
}

var blockSvcTracer = tracing.GetTracer("api-gateway.endpoints.message-blocks.service")

func (c *BlockSvcConfig) validate() error {
	if c.ChatClient == nil {
		return fmt.Errorf("message blocks endpoint service: chat client is required")
	}
	return nil
}

func NewBlockSvc(config *BlockSvcConfig) BlockSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &blockSvcImpl{chatClient: config.ChatClient}
}

func (s *blockSvcImpl) Block(ctx context.Context, req *BlockRequest) (*apiresource.MessagingBlock, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, blockSvcTracer, "service.conversations.block", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BlockInfo, error) {
			return s.chatClient.Block(ctx, &pb.BlockRequest{BlockedAccountUserId: req.BlockedAccountUserID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	// blocked_user is fetched by id on ?include=blocked_user; stash the FK for the resolver.
	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeMessagingBlock, resp.Id, "blocked_user_id", resp.BlockedAccountUserId)
	return &apiresource.MessagingBlock{
		ID:        resp.Id,
		Object:    constants.ObjectTypeMessagingBlock,
		CreatedAt: chatmap.TsToTime(resp.CreatedAt),
	}, nil
}

func (s *blockSvcImpl) Unblock(ctx context.Context, req *UnblockRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, rpcErr := grpcutil.CallRPC(ctx, blockSvcTracer, "service.conversations.unblock", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChatAck, error) {
			return s.chatClient.Unblock(ctx, &pb.UnblockRequest{BlockedAccountUserId: req.BlockedAccountUserID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return &apiresource.EmptyResource{}, nil
}

func (s *blockSvcImpl) ListBlocks(ctx context.Context, _ *ListBlocksRequest) (*apiresource.List[apiresource.MessagingBlock], *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, blockSvcTracer, "service.conversations.list_blocks", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListBlocksResponse, error) {
			return s.chatClient.ListBlocks(ctx, &pb.ListBlocksRequest{}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	meta := resourcekit.GetLoadMeta(ctx)
	items := make([]apiresource.MessagingBlock, 0, len(resp.Blocks))
	for _, b := range resp.Blocks {
		// blocked_user is fetched by id on ?include=blocked_user; stash the FK for the resolver.
		meta.Set(constants.ObjectTypeMessagingBlock, b.Id, "blocked_user_id", b.BlockedAccountUserId)
		items = append(items, apiresource.MessagingBlock{
			ID:        b.Id,
			Object:    constants.ObjectTypeMessagingBlock,
			CreatedAt: chatmap.TsToTime(b.CreatedAt),
		})
	}
	return apiresource.NewList(items, apiresource.PageInfo{}), nil
}
