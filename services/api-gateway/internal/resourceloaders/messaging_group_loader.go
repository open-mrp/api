package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	pb "github.com/augno/api/shared/proto/notification"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var messagingGroupLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.messaging_group")

// HydrateMessagingGroups fills the roster's identity (Name + timestamps) on messaging-group references
// that the chat service stashes by id only. The groups are mutated in place. Members are deliberately
// left nil: the conversation's group is provenance only (members were snapshotted at creation and are
// not driven by the roster thereafter), so exposing the roster's current members here would mislead.
//
// The notification-service exposes no batch get for rosters, so this resolves each distinct id via a
// GetMessagingGroup call. Best-effort: nil entries, duplicates, and unresolved/deleted ids leave the
// reference as a bare id+object stub.
func HydrateMessagingGroups(ctx context.Context, groups []*apiresource.MessagingGroup) {
	if len(groups) == 0 {
		return
	}
	loaded := make(map[string]*pb.MessagingGroupInfo)
	for _, g := range groups {
		if g == nil || g.ID == "" {
			continue
		}
		if _, ok := loaded[g.ID]; ok {
			continue
		}
		info, apiErr := grpcutil.CallRPC(ctx, messagingGroupLoaderTracer, "loader.messaging_groups.get", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.MessagingGroupInfo, error) {
				return chatClient.GetMessagingGroup(ctx, &pb.GetMessagingGroupRequest{GroupId: g.ID}, opts...)
			})
		if apiErr != nil || info == nil {
			continue
		}
		loaded[g.ID] = info
	}

	for _, g := range groups {
		if g == nil {
			continue
		}
		info, ok := loaded[g.ID]
		if !ok {
			continue
		}
		g.Name = info.Name
		g.CreatedAt = grpcutil.TimestampToTime(info.CreatedAt)
		g.UpdatedAt = grpcutil.TimestampToTime(info.UpdatedAt)
	}
}
