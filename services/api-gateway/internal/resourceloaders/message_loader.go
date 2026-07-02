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

var messageLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.message")

// LoadMessages fetches chat messages by ID via BatchGetMessages (access-gated server-side) and builds base Message references, stashing their expandable sub-objects (sender, author, resource, attachments) so deeper includes on an expanded message resolve. Used for a message's reply_to and a conversation's last_message expansions. Display-name hydration is best-effort and applied at the top level by the messages endpoint.
func LoadMessages(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, messageLoaderTracer, "loader.messages.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetMessagesResponse, error) {
			return chatClient.BatchGetMessages(ctx, &pb.BatchGetMessagesRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Messages))
	for _, m := range resp.Messages {
		msg := chatmap.MessageFromProto(m)
		chatmap.StashMessageMeta(ctx, m, &msg)
		out[m.Id] = &msg
	}
	return out, nil
}
