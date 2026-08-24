package conversationep

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/chatmap"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/notification"
	"google.golang.org/grpc"
)

func (s *conversationSvcImpl) SetWorkflowStatus(ctx context.Context, req *SetWorkflowStatusRequest) (*apiresource.Conversation, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.set_workflow_status", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ConversationInfo, error) {
			return s.chatClient.UpdateConversationWorkflow(ctx, &pb.UpdateConversationWorkflowRequest{
				ConversationId: req.ConversationID,
				WorkflowStatus: string(req.WorkflowStatus),
			}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := s.toResource(ctx, resp)
	return &result, nil
}

func (s *conversationSvcImpl) AssignConversation(ctx context.Context, req *AssignConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.assign", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ConversationInfo, error) {
			return s.chatClient.AssignConversation(ctx, &pb.AssignConversationRequest{
				ConversationId:       req.ConversationID,
				AssigneeResourceType: req.AssigneeResourceType.Ptr().StringPtr(),
				AssigneeResourceId:   req.AssigneeResourceID.Ptr(),
			}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := s.toResource(ctx, resp)
	return &result, nil
}

// ReportConversation files an abuse report (notification-service) and returns the refreshed conversation.
func (s *conversationSvcImpl) ReportConversation(ctx context.Context, req *ReportConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
	_, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.report", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MessageReportInfo, error) {
			return s.chatClient.ReportConversation(ctx, &pb.ReportConversationRequest{
				ConversationId: req.ConversationID,
				MessageId:      req.MessageID.Ptr(),
				Reason:         req.Reason,
			}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return s.GetConversation(ctx, &RetrieveConversationRequest{ConversationID: req.ConversationID})
}

// listInbox lists external customer-service cases for the support inbox (the inbox-filter branch of
// the unified conversations list).
func (s *conversationSvcImpl) listInbox(ctx context.Context, req *ListConversationsRequest) (*apiresource.List[apiresource.Conversation], *apierror.APIError) {
	pbReq := &pb.ListInboxRequest{
		Cursor:             req.Cursor,
		Limit:              req.Limit,
		AssigneeResourceId: req.AssigneeResourceID,
		Unassigned:         req.Unassigned,
		IncludeArchived:    req.IncludeArchived,
	}
	if req.WorkflowStatus != nil {
		ws := string(*req.WorkflowStatus)
		pbReq.WorkflowStatus = &ws
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.list_inbox", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListConversationsResponse, error) {
			return s.chatClient.ListInbox(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	items := s.toResourceList(ctx, resp.Conversations)
	pageInfo := apiresource.PageInfo{}
	if resp.PageInfo != nil {
		pageInfo = grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)
	}
	return apiresource.NewList(items, pageInfo), nil
}

// listByResource lists conversations anchored to a business record (the topic-filter branch of the
// unified conversations list).
func (s *conversationSvcImpl) listByResource(ctx context.Context, req *ListConversationsRequest) (*apiresource.List[apiresource.Conversation], *apierror.APIError) {
	resourceType := ""
	if req.TopicResourceType != nil {
		resourceType = string(*req.TopicResourceType)
	}
	resourceID := ""
	if req.TopicResourceID != nil {
		resourceID = *req.TopicResourceID
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.list_by_resource", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListConversationsResponse, error) {
			return s.chatClient.ListConversationsByResource(ctx, &pb.ListConversationsByResourceRequest{
				ResourceType: resourceType,
				ResourceId:   resourceID,
				Limit:        req.Limit,
			}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	items := s.toResourceList(ctx, resp.Conversations)
	return apiresource.NewList(items, apiresource.PageInfo{}), nil
}

func (s *conversationSvcImpl) AddConversationLink(ctx context.Context, req *AddConversationLinkRequest) (*apiresource.ConversationLink, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.add_link", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ConversationLinkInfo, error) {
			return s.chatClient.AddConversationLink(ctx, &pb.AddConversationLinkRequest{
				ConversationId: req.ConversationID,
				ResourceType:   string(req.ResourceType),
				ResourceId:     req.ResourceID,
			}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	link := chatmap.ConversationLinkFromProto(resp)
	chatmap.StashConversationLinkMeta(ctx, resp, &link)
	return &link, nil
}

func (s *conversationSvcImpl) RemoveConversationLink(ctx context.Context, req *RemoveConversationLinkRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.remove_link", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChatAck, error) {
			return s.chatClient.RemoveConversationLink(ctx, &pb.RemoveConversationLinkRequest{
				ConversationId: req.ConversationID,
				LinkId:         req.LinkID,
			}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return &apiresource.EmptyResource{}, nil
}

func (s *conversationSvcImpl) ListConversationLinks(ctx context.Context, req *ListConversationLinksRequest) (*apiresource.List[apiresource.ConversationLink], *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.list_links", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListConversationLinksResponse, error) {
			return s.chatClient.ListConversationLinks(ctx, &pb.ListConversationLinksRequest{ConversationId: req.ConversationID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	items := make([]apiresource.ConversationLink, 0, len(resp.Links))
	for _, l := range resp.Links {
		link := chatmap.ConversationLinkFromProto(l)
		chatmap.StashConversationLinkMeta(ctx, l, &link)
		items = append(items, link)
	}
	return apiresource.NewList(items, apiresource.PageInfo{}), nil
}
