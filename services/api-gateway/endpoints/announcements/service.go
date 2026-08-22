package announcementep

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

// AnnouncementSvc backs the broadcast announcement endpoints via the notification-service MessagingService gRPC client.
type AnnouncementSvc interface {
	ListAnnouncements(ctx context.Context, req *ListAnnouncementsRequest) (*apiresource.List[apiresource.Announcement], *apierror.APIError)
	GetAnnouncement(ctx context.Context, req *RetrieveAnnouncementRequest) (*apiresource.Announcement, *apierror.APIError)
	MarkAnnouncementSeen(ctx context.Context, req *MarkAnnouncementRequest) (*apiresource.Announcement, *apierror.APIError)
	MarkAnnouncementRead(ctx context.Context, req *MarkAnnouncementRequest) (*apiresource.Announcement, *apierror.APIError)
	MarkAnnouncementDismissed(ctx context.Context, req *MarkAnnouncementRequest) (*apiresource.Announcement, *apierror.APIError)
}

type AnnouncementSvcConfig struct {
	// MessagingClient (required) is the notification-service MessagingService gRPC client.
	MessagingClient pb.MessagingServiceClient
}

type announcementSvcImpl struct {
	messagingClient pb.MessagingServiceClient
}

var announcementSvcTracer = tracing.GetTracer("api-gateway.endpoints.announcements.service")

func (c *AnnouncementSvcConfig) validate() error {
	if c.MessagingClient == nil {
		return fmt.Errorf("announcement endpoint service: messaging client is required")
	}
	return nil
}

func NewAnnouncementSvc(config *AnnouncementSvcConfig) AnnouncementSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &announcementSvcImpl{messagingClient: config.MessagingClient}
}

func (s *announcementSvcImpl) ListAnnouncements(ctx context.Context, req *ListAnnouncementsRequest) (*apiresource.List[apiresource.Announcement], *apierror.APIError) {
	pbReq := &pb.ListAnnouncementsRequest{
		Limit:  req.Limit,
		Cursor: req.Cursor,
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, announcementSvcTracer, "service.announcements.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAnnouncementsResponse, error) {
			return s.messagingClient.ListAnnouncements(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	items := make([]apiresource.Announcement, len(resp.Announcements))
	for i, a := range resp.Announcements {
		items[i] = announcementFromProto(ctx, a)
	}

	pageInfo := apiresource.PageInfo{}
	if resp.PageInfo != nil {
		pageInfo = grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)
	}
	return apiresource.NewList(items, pageInfo), nil
}

func (s *announcementSvcImpl) GetAnnouncement(ctx context.Context, req *RetrieveAnnouncementRequest) (*apiresource.Announcement, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, announcementSvcTracer, "service.announcements.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnnouncementInfo, error) {
			return s.messagingClient.GetAnnouncement(ctx, &pb.GetAnnouncementRequest{Id: req.AnnouncementID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := announcementFromProto(ctx, resp)
	return &result, nil
}

func (s *announcementSvcImpl) MarkAnnouncementSeen(ctx context.Context, req *MarkAnnouncementRequest) (*apiresource.Announcement, *apierror.APIError) {
	return s.callMarkAnnouncement(ctx, "service.announcements.mark_seen", req.AnnouncementID, s.messagingClient.MarkAnnouncementSeen)
}

func (s *announcementSvcImpl) MarkAnnouncementRead(ctx context.Context, req *MarkAnnouncementRequest) (*apiresource.Announcement, *apierror.APIError) {
	return s.callMarkAnnouncement(ctx, "service.announcements.mark_read", req.AnnouncementID, s.messagingClient.MarkAnnouncementRead)
}

func (s *announcementSvcImpl) MarkAnnouncementDismissed(ctx context.Context, req *MarkAnnouncementRequest) (*apiresource.Announcement, *apierror.APIError) {
	return s.callMarkAnnouncement(ctx, "service.announcements.mark_dismissed", req.AnnouncementID, s.messagingClient.MarkAnnouncementDismissed)
}

type markAnnouncementRPC func(context.Context, *pb.MarkAnnouncementRequest, ...grpc.CallOption) (*pb.AnnouncementInfo, error)

func (s *announcementSvcImpl) callMarkAnnouncement(ctx context.Context, span, announcementID string, rpc markAnnouncementRPC) (*apiresource.Announcement, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, announcementSvcTracer, span, domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnnouncementInfo, error) {
			return rpc(ctx, &pb.MarkAnnouncementRequest{Id: announcementID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := announcementFromProto(ctx, resp)
	return &result, nil
}

func announcementFromProto(ctx context.Context, a *pb.AnnouncementInfo) apiresource.Announcement {
	if a == nil {
		return apiresource.Announcement{}
	}
	result := apiresource.Announcement{
		ID:          a.Id,
		Object:      constants.ObjectTypeAnnouncement,
		Scope:       constants.AnnouncementScope(a.Scope),
		Category:    constants.NotificationCategory(a.Category),
		Status:      announcementStatus(a),
		Title:       a.Title,
		Body:        a.Body,
		Priority:    constants.NotificationPriority(a.Priority),
		PublishAt:   tsToTime(a.PublishAt),
		ExpiresAt:   tsToPtr(a.ExpiresAt),
		SeenAt:      tsToPtr(a.SeenAt),
		ReadAt:      tsToPtr(a.ReadAt),
		DismissedAt: tsToPtr(a.DismissedAt),
		CreatedAt:   tsToTime(a.CreatedAt),
		UpdatedAt:   tsToTime(a.UpdatedAt),
	}
	if a.LinkResourceType != nil && a.LinkResourceId != nil {
		// resource is expandable: left nil on the base resource, stashed for ?include=resource.
		resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeAnnouncement, result.ID, "resource",
			apiresource.NewEntity(*a.LinkResourceId, constants.ObjectType(*a.LinkResourceType), nil, nil))
	}
	return result
}

// announcementStatus derives the caller's lifecycle status from their receipt timestamps.
func announcementStatus(a *pb.AnnouncementInfo) constants.NotificationStatus {
	switch {
	case a.DismissedAt != nil:
		return constants.NotificationStatusDismissed
	case a.ReadAt != nil:
		return constants.NotificationStatusRead
	case a.SeenAt != nil:
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
