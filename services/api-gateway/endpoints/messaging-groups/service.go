package messaginggroupep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/notification"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

// MessagingGroupSvc backs the reusable-roster endpoints via the notification-service ChatService
// gRPC client.
type MessagingGroupSvc interface {
	CreateMessagingGroup(ctx context.Context, req *CreateMessagingGroupRequest) (*apiresource.MessagingGroup, *apierror.APIError)
	ListMessagingGroups(ctx context.Context, req *ListMessagingGroupsRequest) (*apiresource.List[apiresource.MessagingGroup], *apierror.APIError)
	GetMessagingGroup(ctx context.Context, req *GetMessagingGroupRequest) (*apiresource.MessagingGroup, *apierror.APIError)
	UpdateMessagingGroup(ctx context.Context, req *UpdateMessagingGroupRequest) (*apiresource.MessagingGroup, *apierror.APIError)
	DeleteMessagingGroup(ctx context.Context, req *DeleteMessagingGroupRequest) (*apiresource.EmptyResource, *apierror.APIError)
	AddMessagingGroupMember(ctx context.Context, req *AddMessagingGroupMemberRequest) (*apiresource.MessagingGroup, *apierror.APIError)
	RemoveMessagingGroupMember(ctx context.Context, req *RemoveMessagingGroupMemberRequest) (*apiresource.MessagingGroup, *apierror.APIError)
}

type MessagingGroupSvcConfig struct {
	// ChatClient (required) is the notification-service ChatService gRPC client.
	ChatClient pb.ChatServiceClient
}

type messagingGroupSvcImpl struct {
	chatClient pb.ChatServiceClient
}

var messagingGroupSvcTracer = tracing.GetTracer("api-gateway.endpoints.messaging-groups.service")

func (c *MessagingGroupSvcConfig) validate() error {
	if c.ChatClient == nil {
		return fmt.Errorf("messaging groups endpoint service: chat client is required")
	}
	return nil
}

func NewMessagingGroupSvc(config *MessagingGroupSvcConfig) MessagingGroupSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &messagingGroupSvcImpl{chatClient: config.ChatClient}
}

func (s *messagingGroupSvcImpl) CreateMessagingGroup(ctx context.Context, req *CreateMessagingGroupRequest) (*apiresource.MessagingGroup, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, messagingGroupSvcTracer, "service.messaging_groups.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MessagingGroupInfo, error) {
			return s.chatClient.CreateMessagingGroup(ctx, &pb.CreateMessagingGroupRequest{
				Name:                 req.Name,
				MemberAccountUserIds: req.MemberAccountUserIDs,
				MemberAgentConfigIds: req.MemberAgentConfigIDs,
			}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return s.hydratedGroup(ctx, resp), nil
}

func (s *messagingGroupSvcImpl) ListMessagingGroups(ctx context.Context, _ *ListMessagingGroupsRequest) (*apiresource.List[apiresource.MessagingGroup], *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, messagingGroupSvcTracer, "service.messaging_groups.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListMessagingGroupsResponse, error) {
			return s.chatClient.ListMessagingGroups(ctx, &pb.ListMessagingGroupsRequest{}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	groups := make([]apiresource.MessagingGroup, 0, len(resp.Groups))
	var actors []*apiresource.Actor
	for _, g := range resp.Groups {
		group := messagingGroupFromProto(g)
		groups = append(groups, group)
		actors = append(actors, memberActors(&groups[len(groups)-1])...)
	}
	resourceloaders.HydrateActorNames(ctx, actors)
	list := apiresource.NewList(groups, apiresource.PageInfo{})
	return list, nil
}

func (s *messagingGroupSvcImpl) GetMessagingGroup(ctx context.Context, req *GetMessagingGroupRequest) (*apiresource.MessagingGroup, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, messagingGroupSvcTracer, "service.messaging_groups.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MessagingGroupInfo, error) {
			return s.chatClient.GetMessagingGroup(ctx, &pb.GetMessagingGroupRequest{GroupId: req.GroupID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return s.hydratedGroup(ctx, resp), nil
}

func (s *messagingGroupSvcImpl) UpdateMessagingGroup(ctx context.Context, req *UpdateMessagingGroupRequest) (*apiresource.MessagingGroup, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, messagingGroupSvcTracer, "service.messaging_groups.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MessagingGroupInfo, error) {
			return s.chatClient.UpdateMessagingGroup(ctx, &pb.UpdateMessagingGroupRequest{GroupId: req.GroupID, Name: req.Name}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return s.hydratedGroup(ctx, resp), nil
}

func (s *messagingGroupSvcImpl) DeleteMessagingGroup(ctx context.Context, req *DeleteMessagingGroupRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, rpcErr := grpcutil.CallRPC(ctx, messagingGroupSvcTracer, "service.messaging_groups.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChatAck, error) {
			return s.chatClient.DeleteMessagingGroup(ctx, &pb.DeleteMessagingGroupRequest{GroupId: req.GroupID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return &apiresource.EmptyResource{}, nil
}

func (s *messagingGroupSvcImpl) AddMessagingGroupMember(ctx context.Context, req *AddMessagingGroupMemberRequest) (*apiresource.MessagingGroup, *apierror.APIError) {
	pbReq := &pb.AddMessagingGroupMemberRequest{GroupId: req.GroupID, MemberType: string(req.MemberType)}
	if v, ok := req.AccountUserID.Value(); ok {
		pbReq.AccountUserId = &v
	}
	if v, ok := req.AgentConfigID.Value(); ok {
		pbReq.AgentConfigId = &v
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, messagingGroupSvcTracer, "service.messaging_groups.add_member", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MessagingGroupInfo, error) {
			return s.chatClient.AddMessagingGroupMember(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return s.hydratedGroup(ctx, resp), nil
}

func (s *messagingGroupSvcImpl) RemoveMessagingGroupMember(ctx context.Context, req *RemoveMessagingGroupMemberRequest) (*apiresource.MessagingGroup, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, messagingGroupSvcTracer, "service.messaging_groups.remove_member", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MessagingGroupInfo, error) {
			return s.chatClient.RemoveMessagingGroupMember(ctx, &pb.RemoveMessagingGroupMemberRequest{GroupId: req.GroupID, MemberId: req.MemberID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return s.hydratedGroup(ctx, resp), nil
}

// hydratedGroup maps a group proto to its resource and fills member display names (users + agents).
func (s *messagingGroupSvcImpl) hydratedGroup(ctx context.Context, g *pb.MessagingGroupInfo) *apiresource.MessagingGroup {
	group := messagingGroupFromProto(g)
	resourceloaders.HydrateActorNames(ctx, memberActors(&group))
	return &group
}

// memberActors collects the member actors of a group so their names can be hydrated in one batch.
func memberActors(g *apiresource.MessagingGroup) []*apiresource.Actor {
	if g.Members == nil {
		return nil
	}
	actors := make([]*apiresource.Actor, 0, len(g.Members.Data))
	for i := range g.Members.Data {
		if a := g.Members.Data[i].Actor; a != nil {
			actors = append(actors, a)
		}
	}
	return actors
}

// messagingGroupFromProto maps a roster proto to its API resource. Member actors carry id + type
// only; display names are hydrated separately.
func messagingGroupFromProto(g *pb.MessagingGroupInfo) apiresource.MessagingGroup {
	members := make([]apiresource.MessagingGroupMember, 0, len(g.Members))
	for _, m := range g.Members {
		var actor *apiresource.Actor
		switch m.MemberType {
		case string(constants.ActorTypeAgent):
			if m.AgentConfigId != nil {
				actor = apiresource.NewActor(*m.AgentConfigId, constants.ActorTypeAgent, nil, nil)
			}
		default:
			if m.AccountUserId != nil {
				actor = apiresource.NewActor(*m.AccountUserId, constants.ActorTypeUser, nil, nil)
			}
		}
		members = append(members, apiresource.MessagingGroupMember{
			ID:     m.Id,
			Object: constants.ObjectTypeMessagingGroupMember,
			Actor:  actor,
		})
	}
	return apiresource.MessagingGroup{
		ID:        g.Id,
		Object:    constants.ObjectTypeMessagingGroup,
		Name:      g.Name,
		Members:   apiresource.NewList(members, apiresource.PageInfo{}),
		CreatedAt: g.CreatedAt.AsTime(),
		UpdatedAt: g.UpdatedAt.AsTime(),
	}
}
