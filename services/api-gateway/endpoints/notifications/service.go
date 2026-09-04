package notificationep

import (
	"context"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/notification"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NotificationSvc backs the in-app notification (bell) endpoints via the notification-service MessagingService gRPC client.
type NotificationSvc interface {
	SendNotification(ctx context.Context, req *SendNotificationRequest) (*apiresource.NotificationSendResult, *apierror.APIError)
	ListNotifications(ctx context.Context, req *ListNotificationsRequest) (*apiresource.List[apiresource.Notification], *apierror.APIError)
	GetNotification(ctx context.Context, req *RetrieveNotificationRequest) (*apiresource.Notification, *apierror.APIError)
	GetUnreadCount(ctx context.Context, req *UnreadCountRequest) (*apiresource.NotificationUnreadCount, *apierror.APIError)
	GetUnreadSummary(ctx context.Context, req *UnreadSummaryRequest) (*apiresource.NotificationUnreadSummary, *apierror.APIError)
	MarkSeen(ctx context.Context, req *MarkNotificationRequest) (*apiresource.Notification, *apierror.APIError)
	MarkRead(ctx context.Context, req *MarkNotificationRequest) (*apiresource.Notification, *apierror.APIError)
	MarkDismissed(ctx context.Context, req *MarkNotificationRequest) (*apiresource.Notification, *apierror.APIError)
	MarkAllSeen(ctx context.Context, req *MarkAllSeenRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type NotificationSvcConfig struct {
	// MessagingClient (required) is the notification-service MessagingService gRPC client.
	MessagingClient pb.MessagingServiceClient
}

type notificationSvcImpl struct {
	messagingClient pb.MessagingServiceClient
}

var notificationSvcTracer = tracing.GetTracer("api-gateway.endpoints.notifications.service")

func (c *NotificationSvcConfig) validate() error {
	if c.MessagingClient == nil {
		return fmt.Errorf("notification endpoint service: messaging client is required")
	}
	return nil
}

func NewNotificationSvc(config *NotificationSvcConfig) NotificationSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &notificationSvcImpl{messagingClient: config.MessagingClient}
}

func (s *notificationSvcImpl) SendNotification(ctx context.Context, req *SendNotificationRequest) (*apiresource.NotificationSendResult, *apierror.APIError) {
	if req.Target.ID == "" {
		return nil, apierror.NewParameterMissingError("A target is required.", "target")
	}
	if !req.Target.Type.IsValid() {
		return nil, apierror.NewParameterInvalidError("The target type is not supported.", "target.type")
	}

	pbReq := &pb.SendNotificationRequest{
		Category:       string(req.Category),
		TargetType:     string(req.Target.Type),
		TargetId:       req.Target.ID,
		Title:          req.Title,
		Body:           req.Body.Ptr(),
		LinkResourceId: req.LinkResourceID.Ptr(),
	}
	if p, ok := req.Priority.Value(); ok {
		ps := string(p)
		pbReq.Priority = &ps
	}
	if lt, ok := req.LinkResourceType.Value(); ok {
		lts := string(lt)
		pbReq.LinkResourceType = &lts
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, notificationSvcTracer, "service.notifications.send", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SendNotificationResponse, error) {
			return s.messagingClient.SendNotification(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return &apiresource.NotificationSendResult{
		Object:   constants.ObjectTypeNotificationSendResult,
		Enqueued: resp.Enqueued,
	}, nil
}

func (s *notificationSvcImpl) ListNotifications(ctx context.Context, req *ListNotificationsRequest) (*apiresource.List[apiresource.Notification], *apierror.APIError) {
	pbReq := &pb.ListNotificationsRequest{
		Limit:       req.Limit,
		Cursor:      req.Cursor,
		Search:      req.Query,
		SenderIds:   req.SenderIDs,
		SenderTypes: senderTypesToStrings(req.SenderTypes),
	}
	if req.Category != nil {
		c := string(*req.Category)
		pbReq.Category = &c
	}
	if req.Status != nil {
		s := string(*req.Status)
		pbReq.Status = &s
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, notificationSvcTracer, "service.notifications.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListNotificationsResponse, error) {
			return s.messagingClient.ListNotifications(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	items := make([]apiresource.Notification, len(resp.Notifications))
	for i, n := range resp.Notifications {
		items[i] = notificationFromProto(ctx, n)
	}

	pageInfo := apiresource.PageInfo{}
	if resp.PageInfo != nil {
		pageInfo = grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)
	}
	return apiresource.NewList(items, pageInfo), nil
}

func (s *notificationSvcImpl) GetNotification(ctx context.Context, req *RetrieveNotificationRequest) (*apiresource.Notification, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, notificationSvcTracer, "service.notifications.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.NotificationInfo, error) {
			return s.messagingClient.GetNotification(ctx, &pb.GetNotificationRequest{Id: req.NotificationID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := notificationFromProto(ctx, resp)
	return &result, nil
}

func (s *notificationSvcImpl) GetUnreadCount(ctx context.Context, _ *UnreadCountRequest) (*apiresource.NotificationUnreadCount, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, notificationSvcTracer, "service.notifications.unread_count", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetUnreadCountResponse, error) {
			return s.messagingClient.GetUnreadCount(ctx, &pb.GetUnreadCountRequest{}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return &apiresource.NotificationUnreadCount{
		Object:        constants.ObjectTypeNotificationUnreadCount,
		Notifications: resp.Notifications,
		Conversations: resp.Conversations,
		Total:         resp.Total,
	}, nil
}

func (s *notificationSvcImpl) GetUnreadSummary(ctx context.Context, _ *UnreadSummaryRequest) (*apiresource.NotificationUnreadSummary, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, notificationSvcTracer, "service.notifications.unread_summary", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetUnreadSummaryResponse, error) {
			return s.messagingClient.GetUnreadSummary(ctx, &pb.GetUnreadSummaryRequest{}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	accounts := make([]apiresource.NotificationUnreadSummaryAccount, 0, len(resp.Accounts))
	for _, a := range resp.Accounts {
		accounts = append(accounts, apiresource.NotificationUnreadSummaryAccount{
			Object:  constants.ObjectTypeNotificationUnreadSummaryAccount,
			Account: apiresource.NewEntity(a.AccountId, constants.ObjectTypeAccount, nil, nil),
			Unread:  a.Unread,
		})
	}
	return &apiresource.NotificationUnreadSummary{
		Object:   constants.ObjectTypeNotificationUnreadSummary,
		Total:    resp.Total,
		Accounts: apiresource.NewList(accounts, apiresource.PageInfo{}),
	}, nil
}

func (s *notificationSvcImpl) MarkSeen(ctx context.Context, req *MarkNotificationRequest) (*apiresource.Notification, *apierror.APIError) {
	return s.callMark(ctx, "service.notifications.mark_seen", req.NotificationID, s.messagingClient.MarkNotificationSeen)
}

func (s *notificationSvcImpl) MarkRead(ctx context.Context, req *MarkNotificationRequest) (*apiresource.Notification, *apierror.APIError) {
	return s.callMark(ctx, "service.notifications.mark_read", req.NotificationID, s.messagingClient.MarkNotificationRead)
}

func (s *notificationSvcImpl) MarkDismissed(ctx context.Context, req *MarkNotificationRequest) (*apiresource.Notification, *apierror.APIError) {
	return s.callMark(ctx, "service.notifications.mark_dismissed", req.NotificationID, s.messagingClient.MarkNotificationDismissed)
}

type markRPC func(context.Context, *pb.MarkNotificationRequest, ...grpc.CallOption) (*pb.NotificationInfo, error)

func (s *notificationSvcImpl) callMark(ctx context.Context, span, notificationID string, rpc markRPC) (*apiresource.Notification, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, notificationSvcTracer, span, domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.NotificationInfo, error) {
			return rpc(ctx, &pb.MarkNotificationRequest{Id: notificationID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := notificationFromProto(ctx, resp)
	return &result, nil
}

func (s *notificationSvcImpl) MarkAllSeen(ctx context.Context, _ *MarkAllSeenRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, rpcErr := grpcutil.CallRPC(ctx, notificationSvcTracer, "service.notifications.mark_all_seen", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MarkAllNotificationsSeenResponse, error) {
			return s.messagingClient.MarkAllNotificationsSeen(ctx, &pb.MarkAllNotificationsSeenRequest{}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return &apiresource.EmptyResource{}, nil
}

func notificationFromProto(ctx context.Context, n *pb.NotificationInfo) apiresource.Notification {
	if n == nil {
		return apiresource.Notification{}
	}
	result := apiresource.Notification{
		ID:          n.Id,
		Object:      constants.ObjectTypeNotification,
		Category:    constants.NotificationCategory(n.Category),
		Status:      notificationStatus(n),
		Title:       n.Title,
		Body:        n.Body,
		Priority:    constants.NotificationPriority(n.Priority),
		SeenAt:      tsToPtr(n.SeenAt),
		ReadAt:      tsToPtr(n.ReadAt),
		DismissedAt: tsToPtr(n.DismissedAt),
		CreatedAt:   tsToTime(n.CreatedAt),
		UpdatedAt:   tsToTime(n.UpdatedAt),
	}
	if n.ChangeCount != nil {
		cc := int64(*n.ChangeCount)
		result.ChangeCount = &cc
	}
	stashNotificationMeta(ctx, &result, n)
	return result
}

// stashNotificationMeta stashes the notification's expandable sub-objects (the `sender` actor and the
// `resource` link) into the request LoadMeta so the include resolver can surface each on
// ?include=sender / ?include=resource. The fields are left nil on the base resource.
func stashNotificationMeta(ctx context.Context, d *apiresource.Notification, n *pb.NotificationInfo) {
	if sender := senderFromProto(n.SenderType, n.SenderId, n.SenderName); sender != nil {
		resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeNotification, d.ID, "sender", sender)
	}

	var resource *apiresource.Entity
	if n.LinkResourceType != nil && n.LinkResourceId != nil {
		resource = apiresource.NewEntity(*n.LinkResourceId, constants.ObjectType(*n.LinkResourceType), nil, nil)
	} else if n.ConversationId != nil && *n.ConversationId != "" {
		// Chat notifications carry no explicit link resource; surface the conversation so
		// the client can open the thread on click.
		resource = apiresource.NewEntity(*n.ConversationId, constants.ObjectTypeConversation, nil, nil)
	}
	if resource != nil {
		resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeNotification, d.ID, "resource", resource)
	}
}

// senderFromProto builds the unified Actor for a notification's sender. System and unattributed
// notifications are represented by a null sender field (rather than an actor with a system type), so
// we only emit an actor for user/group/agent/apikey attributions, which always carry an id.
func senderFromProto(senderType, senderID, senderName *string) *apiresource.Actor {
	if senderType == nil || *senderType == "" {
		return nil
	}
	actorType := constants.ActorTypeFromSenderType(constants.NotificationSenderType(*senderType))
	if actorType == "" {
		return nil
	}
	if senderID == nil || *senderID == "" {
		return nil
	}
	return apiresource.NewActor(*senderID, actorType, senderName, nil)
}

// senderTypesToStrings flattens the typed sender-type filter into wire strings.
func senderTypesToStrings(types []constants.NotificationSenderType) []string {
	if len(types) == 0 {
		return nil
	}
	out := make([]string, 0, len(types))
	for _, t := range types {
		out = append(out, string(t))
	}
	return out
}

// notificationStatus derives the lifecycle status from the seen/read/dismissed timestamps.
func notificationStatus(n *pb.NotificationInfo) constants.NotificationStatus {
	switch {
	case n.DismissedAt != nil:
		return constants.NotificationStatusDismissed
	case n.ReadAt != nil:
		return constants.NotificationStatusRead
	case n.SeenAt != nil:
		return constants.NotificationStatusSeen
	default:
		return constants.NotificationStatusUnseen
	}
}

func tsToTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

func tsToPtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}
