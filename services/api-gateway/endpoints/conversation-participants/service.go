package participantep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/chatmap"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/notification"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

// ParticipantSvc backs the conversation participant endpoints via the notification-service
// ChatService gRPC client.
type ParticipantSvc interface {
	AddParticipant(ctx context.Context, req *AddParticipantRequest) (*apiresource.Conversation, *apierror.APIError)
	RemoveParticipant(ctx context.Context, req *RemoveParticipantRequest) (*apiresource.EmptyResource, *apierror.APIError)
	UpdateParticipantRole(ctx context.Context, req *UpdateParticipantRoleRequest) (*apiresource.Conversation, *apierror.APIError)
	AddAgentParticipant(ctx context.Context, req *AddAgentParticipantRequest) (*apiresource.ConversationParticipant, *apierror.APIError)
	RemoveAgentParticipant(ctx context.Context, req *RemoveAgentParticipantRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ParticipantSvcConfig struct {
	// ChatClient (required) is the notification-service ChatService gRPC client.
	ChatClient pb.ChatServiceClient
}

type participantSvcImpl struct {
	chatClient pb.ChatServiceClient
}

var participantSvcTracer = tracing.GetTracer("api-gateway.endpoints.conversation-participants.service")

func (c *ParticipantSvcConfig) validate() error {
	if c.ChatClient == nil {
		return fmt.Errorf("conversation participants endpoint service: chat client is required")
	}
	return nil
}

func NewParticipantSvc(config *ParticipantSvcConfig) ParticipantSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &participantSvcImpl{chatClient: config.ChatClient}
}

func (s *participantSvcImpl) AddParticipant(ctx context.Context, req *AddParticipantRequest) (*apiresource.Conversation, *apierror.APIError) {
	pbReq := &pb.AddParticipantRequest{ConversationId: req.ConversationID, AccountUserId: req.AccountUserID}
	if r, ok := req.Role.Value(); ok {
		s := string(r)
		pbReq.Role = &s
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, participantSvcTracer, "service.conversations.add_participant", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ConversationInfo, error) {
			return s.chatClient.AddParticipant(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := chatmap.ConversationFromProto(resp)
	chatmap.StashConversationMeta(ctx, resp, &result)
	hydrateConversationParticipants(ctx, &result)
	return &result, nil
}

// hydrateConversationParticipants fills participant actor names on a conversation when the caller
// requested ?include=participants, reading the stashed participants list from LoadMeta. Best-effort.
func hydrateConversationParticipants(ctx context.Context, conv *apiresource.Conversation) {
	if !resourcekit.RequestedIncludeSet(ctx)["participants"] {
		return
	}
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeConversation, conv.ID, "participants")
	if !ok {
		return
	}
	list := v.(*apiresource.List[apiresource.ConversationParticipant])
	var actors []*apiresource.Actor
	for i := range list.Data {
		if a := list.Data[i].Actor; a != nil {
			actors = append(actors, a)
		}
	}
	resourceloaders.HydrateActorNames(ctx, actors)
}

func (s *participantSvcImpl) RemoveParticipant(ctx context.Context, req *RemoveParticipantRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, rpcErr := grpcutil.CallRPC(ctx, participantSvcTracer, "service.conversations.remove_participant", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChatAck, error) {
			return s.chatClient.RemoveParticipant(ctx, &pb.RemoveParticipantRequest{ConversationId: req.ConversationID, ParticipantId: req.ParticipantID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return &apiresource.EmptyResource{}, nil
}

func (s *participantSvcImpl) UpdateParticipantRole(ctx context.Context, req *UpdateParticipantRoleRequest) (*apiresource.Conversation, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, participantSvcTracer, "service.conversations.update_participant_role", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ConversationInfo, error) {
			return s.chatClient.UpdateParticipantRole(ctx, &pb.UpdateParticipantRoleRequest{ConversationId: req.ConversationID, ParticipantId: req.ParticipantID, Role: string(req.Role)}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := chatmap.ConversationFromProto(resp)
	chatmap.StashConversationMeta(ctx, resp, &result)
	hydrateConversationParticipants(ctx, &result)
	return &result, nil
}

func (s *participantSvcImpl) AddAgentParticipant(ctx context.Context, req *AddAgentParticipantRequest) (*apiresource.ConversationParticipant, *apierror.APIError) {
	pbReq := &pb.AddAgentParticipantRequest{
		ConversationId:  req.ConversationID,
		AgentConfigId:   req.AgentConfigID,
		TriggerKeywords: req.TriggerKeywords,
	}
	if tp, ok := req.TriggerPolicy.Value(); ok {
		s := string(tp)
		pbReq.TriggerPolicy = &s
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, participantSvcTracer, "service.conversations.add_agent_participant", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ParticipantInfo, error) {
			return s.chatClient.AddAgentParticipant(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := chatmap.ParticipantFromProto(resp)
	hydrateAgentActor(ctx, &result)
	return &result, nil
}

// hydrateAgentActor fills the agent participant's display name + handle (slug) from agent-service,
// since the chat service stores agents by agent_config_id only. Best-effort.
func hydrateAgentActor(ctx context.Context, p *apiresource.ConversationParticipant) {
	a := p.Actor
	if a == nil || a.Type != constants.ActorTypeAgent || a.ID == "" {
		return
	}
	names, apiErr := resourceloaders.LoadAgentDefinitionNames(ctx, []string{a.ID})
	if apiErr != nil {
		return
	}
	if n, ok := names[a.ID]; ok {
		if n.Name != "" {
			name := n.Name
			a.Name = &name
		}
		if n.Slug != "" {
			slug := n.Slug
			a.Handle = &slug
		}
	}
}

func (s *participantSvcImpl) RemoveAgentParticipant(ctx context.Context, req *RemoveAgentParticipantRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, rpcErr := grpcutil.CallRPC(ctx, participantSvcTracer, "service.conversations.remove_agent_participant", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChatAck, error) {
			return s.chatClient.RemoveAgentParticipant(ctx, &pb.RemoveAgentParticipantRequest{ConversationId: req.ConversationID, ParticipantId: req.ParticipantID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return &apiresource.EmptyResource{}, nil
}
