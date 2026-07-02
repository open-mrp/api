package grpc

import (
	"context"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/notification"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type messagingGRPCHandler struct {
	pb.UnimplementedMessagingServiceServer
	messagingSvc domain.MessagingSvc
}

// NewMessagingGRPCHandler registers the MessagingService (in-app notifications) handler.
func NewMessagingGRPCHandler(server *grpc.Server, messagingSvc domain.MessagingSvc) *messagingGRPCHandler {
	handler := &messagingGRPCHandler{messagingSvc: messagingSvc}
	pb.RegisterMessagingServiceServer(server, handler)
	return handler
}

func (h *messagingGRPCHandler) SendNotification(ctx context.Context, req *pb.SendNotificationRequest) (*pb.SendNotificationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	enqueued, apiErr := h.messagingSvc.SendNotification(ctx, domain.SendNotificationInput{
		Category:         req.Category,
		TargetType:       req.TargetType,
		TargetID:         req.TargetId,
		Title:            req.Title,
		Body:             req.Body,
		TemplateKey:      req.TemplateKey,
		TemplateParams:   req.TemplateParams,
		LinkResourceType: req.LinkResourceType,
		LinkResourceID:   req.LinkResourceId,
		Priority:         req.Priority,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.SendNotificationResponse{Enqueued: enqueued}, nil
}

func (h *messagingGRPCHandler) ListNotifications(ctx context.Context, req *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	page, apiErr := h.messagingSvc.ListNotifications(ctx, domain.ListNotificationsInput{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		Category:    req.Category,
		Status:      req.Status,
		Search:      req.Search,
		SenderIDs:   req.SenderIds,
		SenderTypes: req.SenderTypes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	notifications := make([]*pb.NotificationInfo, 0, len(page.Notifications))
	for _, n := range page.Notifications {
		notifications = append(notifications, toProtoNotification(n))
	}
	return &pb.ListNotificationsResponse{
		Notifications: notifications,
		PageInfo: &pb.PageInfo{
			NextCursor:  page.NextCursor,
			HasNextPage: page.HasNextPage,
		},
	}, nil
}

func (h *messagingGRPCHandler) GetNotification(ctx context.Context, req *pb.GetNotificationRequest) (*pb.NotificationInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	n, apiErr := h.messagingSvc.GetNotification(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return toProtoNotification(n), nil
}

func (h *messagingGRPCHandler) GetUnreadCount(ctx context.Context, req *pb.GetUnreadCountRequest) (*pb.GetUnreadCountResponse, error) {
	counts, apiErr := h.messagingSvc.GetUnreadCount(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.GetUnreadCountResponse{
		Notifications: counts.Notifications,
		Conversations: counts.Conversations,
		Total:         counts.Total,
	}, nil
}

func (h *messagingGRPCHandler) GetUnreadSummary(ctx context.Context, _ *pb.GetUnreadSummaryRequest) (*pb.GetUnreadSummaryResponse, error) {
	summary, apiErr := h.messagingSvc.GetUnreadSummary(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	accounts := make([]*pb.UnreadSummaryAccount, 0, len(summary.Accounts))
	for _, a := range summary.Accounts {
		accounts = append(accounts, &pb.UnreadSummaryAccount{AccountId: a.AccountID, Unread: a.Unread})
	}
	return &pb.GetUnreadSummaryResponse{Total: summary.Total, Accounts: accounts}, nil
}

func (h *messagingGRPCHandler) MarkNotificationSeen(ctx context.Context, req *pb.MarkNotificationRequest) (*pb.NotificationInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	n, apiErr := h.messagingSvc.MarkSeen(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return toProtoNotification(n), nil
}

func (h *messagingGRPCHandler) MarkNotificationRead(ctx context.Context, req *pb.MarkNotificationRequest) (*pb.NotificationInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	n, apiErr := h.messagingSvc.MarkRead(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return toProtoNotification(n), nil
}

func (h *messagingGRPCHandler) MarkNotificationDismissed(ctx context.Context, req *pb.MarkNotificationRequest) (*pb.NotificationInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	n, apiErr := h.messagingSvc.MarkDismissed(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return toProtoNotification(n), nil
}

func (h *messagingGRPCHandler) MarkAllNotificationsSeen(ctx context.Context, req *pb.MarkAllNotificationsSeenRequest) (*pb.MarkAllNotificationsSeenResponse, error) {
	updated, apiErr := h.messagingSvc.MarkAllSeen(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.MarkAllNotificationsSeenResponse{Updated: updated}, nil
}

func (h *messagingGRPCHandler) ListAnnouncements(ctx context.Context, req *pb.ListAnnouncementsRequest) (*pb.ListAnnouncementsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	page, apiErr := h.messagingSvc.ListAnnouncements(ctx, domain.ListAnnouncementsInput{
		Cursor: req.Cursor,
		Limit:  req.Limit,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	announcements := make([]*pb.AnnouncementInfo, 0, len(page.Announcements))
	for _, a := range page.Announcements {
		announcements = append(announcements, toProtoAnnouncement(a))
	}
	return &pb.ListAnnouncementsResponse{
		Announcements: announcements,
		PageInfo: &pb.PageInfo{
			NextCursor:  page.NextCursor,
			HasNextPage: page.HasNextPage,
		},
	}, nil
}

func (h *messagingGRPCHandler) GetAnnouncement(ctx context.Context, req *pb.GetAnnouncementRequest) (*pb.AnnouncementInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	a, apiErr := h.messagingSvc.GetAnnouncement(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return toProtoAnnouncement(a), nil
}

func (h *messagingGRPCHandler) MarkAnnouncementSeen(ctx context.Context, req *pb.MarkAnnouncementRequest) (*pb.AnnouncementInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	a, apiErr := h.messagingSvc.MarkAnnouncementSeen(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return toProtoAnnouncement(a), nil
}

func (h *messagingGRPCHandler) MarkAnnouncementRead(ctx context.Context, req *pb.MarkAnnouncementRequest) (*pb.AnnouncementInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	a, apiErr := h.messagingSvc.MarkAnnouncementRead(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return toProtoAnnouncement(a), nil
}

func (h *messagingGRPCHandler) MarkAnnouncementDismissed(ctx context.Context, req *pb.MarkAnnouncementRequest) (*pb.AnnouncementInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	a, apiErr := h.messagingSvc.MarkAnnouncementDismissed(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return toProtoAnnouncement(a), nil
}

func (h *messagingGRPCHandler) ListNotificationPreferences(ctx context.Context, _ *pb.ListNotificationPreferencesRequest) (*pb.ListNotificationPreferencesResponse, error) {
	prefs, apiErr := h.messagingSvc.ListNotificationPreferences(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.NotificationPreferenceInfo, 0, len(prefs))
	for _, p := range prefs {
		out = append(out, toProtoNotificationPreference(p))
	}
	return &pb.ListNotificationPreferencesResponse{Preferences: out}, nil
}

func (h *messagingGRPCHandler) UpsertNotificationPreference(ctx context.Context, req *pb.UpsertNotificationPreferenceRequest) (*pb.NotificationPreferenceInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	pref, apiErr := h.messagingSvc.UpsertNotificationPreference(ctx, domain.UpsertNotificationPreferenceInput{
		Category:     req.Category,
		InAppEnabled: req.InAppEnabled,
		EmailEnabled: req.EmailEnabled,
		PushEnabled:  req.PushEnabled,
		Digest:       req.Digest,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return toProtoNotificationPreference(pref), nil
}

func toProtoNotificationPreference(p *domain.NotificationPreference) *pb.NotificationPreferenceInfo {
	return &pb.NotificationPreferenceInfo{
		Id:            p.ID,
		AccountId:     p.AccountID,
		AccountUserId: p.AccountUserID,
		Category:      p.Category,
		InAppEnabled:  p.InAppEnabled,
		EmailEnabled:  p.EmailEnabled,
		PushEnabled:   p.PushEnabled,
		Digest:        p.Digest,
		CreatedAt:     timestamppb.New(p.CreatedAt),
		UpdatedAt:     timestamppb.New(p.UpdatedAt),
	}
}

func toProtoAnnouncement(a *domain.Announcement) *pb.AnnouncementInfo {
	info := &pb.AnnouncementInfo{
		Id:               a.ID,
		Scope:            a.Scope,
		AccountId:        a.AccountID,
		Category:         a.Category,
		TemplateKey:      a.TemplateKey,
		TemplateParams:   a.TemplateParams,
		Title:            a.Title,
		Body:             a.Body,
		LinkResourceType: a.LinkResourceType,
		LinkResourceId:   a.LinkResourceID,
		Priority:         a.Priority,
		PublishAt:        timestamppb.New(a.PublishAt),
		CreatedAt:        timestamppb.New(a.CreatedAt),
		UpdatedAt:        timestamppb.New(a.UpdatedAt),
	}
	if a.ExpiresAt != nil {
		info.ExpiresAt = timestamppb.New(*a.ExpiresAt)
	}
	if a.SeenAt != nil {
		info.SeenAt = timestamppb.New(*a.SeenAt)
	}
	if a.ReadAt != nil {
		info.ReadAt = timestamppb.New(*a.ReadAt)
	}
	if a.DismissedAt != nil {
		info.DismissedAt = timestamppb.New(*a.DismissedAt)
	}
	return info
}

func toProtoNotification(n *domain.Notification) *pb.NotificationInfo {
	info := &pb.NotificationInfo{
		Id:                     n.ID,
		AccountId:              n.AccountID,
		RecipientAccountUserId: n.RecipientAccountUserID,
		Category:               n.Category,
		SourceMessageId:        n.SourceMessageID,
		ConversationId:         n.ConversationID,
		Title:                  n.Title,
		Body:                   n.Body,
		TemplateKey:            n.TemplateKey,
		TemplateParams:         n.TemplateParams,
		LinkResourceType:       n.LinkResourceType,
		LinkResourceId:         n.LinkResourceID,
		SenderType:             n.SenderType,
		SenderId:               n.SenderID,
		SenderName:             n.SenderName,
		Priority:               n.Priority,
		CreatedAt:              timestamppb.New(n.CreatedAt),
		UpdatedAt:              timestamppb.New(n.UpdatedAt),
	}
	if n.SeenAt != nil {
		info.SeenAt = timestamppb.New(*n.SeenAt)
	}
	if n.ReadAt != nil {
		info.ReadAt = timestamppb.New(*n.ReadAt)
	}
	if n.DismissedAt != nil {
		info.DismissedAt = timestamppb.New(*n.DismissedAt)
	}
	return info
}
