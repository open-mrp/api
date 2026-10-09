package resourceloaders

import (
	"context"

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

var messageLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.message")

// LoadMessages fetches chat messages by ID via BatchGetMessages (access-gated server-side) and builds base Message references, stashing their expandable sub-objects (sender, author, resource, attachments) so deeper includes on an expanded message resolve. Used for a message's reply_to and a conversation's last_message expansions. Sender/author names are hydrated here: the stash shares its key with any top-level copy of the same message, so an unhydrated actor would also blank that copy's already-hydrated name.
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
	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Messages))
	var actors []*apiresource.Actor
	for _, m := range resp.Messages {
		msg := chatmap.MessageFromProto(m)
		chatmap.StashMessageMeta(ctx, m, &msg)
		out[m.Id] = &msg
		for _, key := range []string{"sender", "author"} {
			if v, ok := meta.Get(constants.ObjectTypeChatMessage, m.Id, key); ok {
				actors = append(actors, v.(*apiresource.Actor))
			}
		}
	}
	HydrateActorNames(ctx, actors)
	return out, nil
}
