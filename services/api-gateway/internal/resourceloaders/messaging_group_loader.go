package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/chatmap"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	pb "github.com/open-mrp/api/shared/proto/notification"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
)

var messagingGroupLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.messaging_group")

// LoadMessagingGroups loads rosters by ID, with their members, for a field that includes one. A roster deleted after the including row was read is left out.
func LoadMessagingGroups(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	out := make(map[string]any, len(ids))
	var actors []*apiresource.Actor
	for _, id := range ids {
		groupID := id
		info, apiErr := grpcutil.CallRPC(ctx, messagingGroupLoaderTracer, "loader.messaging_groups.get", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.MessagingGroupInfo, error) {
				return chatClient.GetMessagingGroup(ctx, &pb.GetMessagingGroupRequest{GroupId: groupID}, opts...)
			})
		if apiErr != nil {
			if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
				continue
			}
			return nil, apiErr
		}
		group := chatmap.MessagingGroupFromProto(info)
		for i := range group.Members.Data {
			if a := group.Members.Data[i].Actor; a != nil {
				actors = append(actors, a)
			}
		}
		out[groupID] = &group
	}
	HydrateActorNames(ctx, actors)
	return out, nil
}

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
	embedCtx := include.WithIncludeReads(ctx)
	loaded := make(map[string]*pb.MessagingGroupInfo)
	for _, g := range groups {
		if g == nil || g.ID == "" {
			continue
		}
		if _, ok := loaded[g.ID]; ok {
			continue
		}
		info, apiErr := grpcutil.CallRPC(embedCtx, messagingGroupLoaderTracer, "loader.messaging_groups.get", domain.ServiceName,
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
