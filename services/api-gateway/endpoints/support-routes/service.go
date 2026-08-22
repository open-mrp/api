package supportrouteep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/notification"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

// SupportRouteSvc backs the support-route endpoints via the notification-service ChatService gRPC client.
type SupportRouteSvc interface {
	SetSupportRoute(ctx context.Context, req *SetSupportRouteRequest) (*apiresource.SupportRoute, *apierror.APIError)
	ClearSupportRoute(ctx context.Context, req *ClearSupportRouteRequest) (*apiresource.EmptyResource, *apierror.APIError)
	GetSupportRoute(ctx context.Context, req *GetSupportRouteRequest) (*apiresource.SupportRoute, *apierror.APIError)
}

type SupportRouteSvcConfig struct {
	// ChatClient (required) is the notification-service ChatService gRPC client.
	ChatClient pb.ChatServiceClient
}

type supportRouteSvcImpl struct {
	chatClient pb.ChatServiceClient
}

var supportRouteSvcTracer = tracing.GetTracer("api-gateway.endpoints.support-routes.service")

func (c *SupportRouteSvcConfig) validate() error {
	if c.ChatClient == nil {
		return fmt.Errorf("support route endpoint service: chat client is required")
	}
	return nil
}

func NewSupportRouteSvc(config *SupportRouteSvcConfig) SupportRouteSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &supportRouteSvcImpl{chatClient: config.ChatClient}
}

func supportRouteFromProto(r *pb.SupportRouteInfo) *apiresource.SupportRoute {
	out := &apiresource.SupportRoute{
		ID:                r.Id,
		Object:            constants.ObjectTypeSupportRoute,
		GroupConversation: apiresource.NewEntity(r.GroupConversationId, constants.ObjectTypeConversation, nil, nil),
		CreatedAt:         grpcutil.TimestampToTime(r.CreatedAt),
		UpdatedAt:         grpcutil.TimestampToTime(r.UpdatedAt),
	}
	// The empty-string scope is the account-level default; surface it as a null relation rather than "".
	if r.RelationAccountId != "" {
		out.RelationAccount = apiresource.NewEntity(r.RelationAccountId, constants.ObjectTypeAccount, nil, nil)
	}
	return out
}

func (s *supportRouteSvcImpl) SetSupportRoute(ctx context.Context, req *SetSupportRouteRequest) (*apiresource.SupportRoute, *apierror.APIError) {
	relationAccountID := ""
	if v, ok := req.RelationAccountID.Value(); ok {
		relationAccountID = v
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, supportRouteSvcTracer, "service.support_routes.set", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SupportRouteInfo, error) {
			return s.chatClient.SetSupportRoute(ctx, &pb.SetSupportRouteRequest{
				RelationAccountId:   relationAccountID,
				GroupConversationId: req.GroupConversationID,
			}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return supportRouteFromProto(resp), nil
}

func (s *supportRouteSvcImpl) ClearSupportRoute(ctx context.Context, req *ClearSupportRouteRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	relationAccountID := ""
	if v, ok := req.RelationAccountID.Value(); ok {
		relationAccountID = v
	}
	_, rpcErr := grpcutil.CallRPC(ctx, supportRouteSvcTracer, "service.support_routes.clear", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChatAck, error) {
			return s.chatClient.ClearSupportRoute(ctx, &pb.ClearSupportRouteRequest{RelationAccountId: relationAccountID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return &apiresource.EmptyResource{}, nil
}

func (s *supportRouteSvcImpl) GetSupportRoute(ctx context.Context, req *GetSupportRouteRequest) (*apiresource.SupportRoute, *apierror.APIError) {
	relationAccountID := ""
	if req.RelationAccountID != nil {
		relationAccountID = *req.RelationAccountID
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, supportRouteSvcTracer, "service.support_routes.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SupportRouteInfo, error) {
			return s.chatClient.GetSupportRoute(ctx, &pb.GetSupportRouteRequest{RelationAccountId: relationAccountID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return supportRouteFromProto(resp), nil
}
