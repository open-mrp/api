package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/chatmap"
	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/notification"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var conversationLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.conversation")

// LoadConversations fetches conversations by ID via BatchGetConversations (access-gated server-side)
// and builds base Conversation references, stashing their expandable sub-objects (participants, topic, last_message) so deeper includes on an expanded conversation resolve. Display-name hydration is best-effort and only applied at the top level by the conversations endpoint.
func LoadConversations(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, conversationLoaderTracer, "loader.conversations.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetConversationsResponse, error) {
			return chatClient.BatchGetConversations(ctx, &pb.BatchGetConversationsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Conversations))
	for _, c := range resp.Conversations {
		conv := chatmap.ConversationFromProto(c)
		chatmap.StashConversationMeta(ctx, c, &conv)
		out[c.Id] = &conv
	}
	return out, nil
}
