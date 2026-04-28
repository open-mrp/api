package auditeventsep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/platform"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuditEventSvc interface {
	ListAuditEvents(ctx context.Context, req *ListAuditEventsRequest) (*apiresource.List[apiresource.AuditEvent], *apierror.APIError)
	ListAuditEventResourceTypes(ctx context.Context, req *ListAuditEventResourceTypesRequest) (*apiresource.List[constants.ObjectType], *apierror.APIError)
	GetAuditEvent(ctx context.Context, req *GetAuditEventRequest) (*apiresource.AuditEvent, *apierror.APIError)
}

type AuditEventSvcConfig struct {
	AuditClient pb.AuditServiceClient
}

type auditEventSvcImpl struct {
	auditClient pb.AuditServiceClient
}

var auditEventSvcTracer = tracing.GetTracer("api-gateway.endpoints.audit_events.service")

func (c *AuditEventSvcConfig) validate() error {
	if c.AuditClient == nil {
		return fmt.Errorf("audit events endpoint service: audit client is required")
	}
	return nil
}

func NewAuditEventSvc(config *AuditEventSvcConfig) AuditEventSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &auditEventSvcImpl{
		auditClient: config.AuditClient,
	}
}

func requireInternalAdmin(ctx context.Context) *apierror.APIError {
	identity, apiErr := httptransport.GetIdentity(ctx)
	if apiErr != nil {
		return apiErr
	}
	if !identity.IsInternalUser() || !identity.IsAdmin() {
		return apierror.NewAuthorizationError("Only internal administrators can access audit events.")
	}
	return nil
}

func (m *auditEventSvcImpl) ListAuditEvents(ctx context.Context, req *ListAuditEventsRequest) (*apiresource.List[apiresource.AuditEvent], *apierror.APIError) {
	if apiErr := requireInternalAdmin(ctx); apiErr != nil {
		return nil, apiErr
	}

	pbReq := &pb.ListAuditEventsRequest{
		ResourceTypes: stringsFromObjectTypes(req.ResourceTypes),
		ResourceIds:   req.ResourceIDs,
		ActorIds:      req.ActorIDs,
		Actions:       stringsFromActions(req.Actions),
		Query:         req.Query,
		Cursor:        req.Cursor,
		Limit:         req.Limit,
		Includes:      appctx.GetRequestedIncludeKeys(ctx),
	}

	if req.StartDate != nil && !req.StartDate.IsZero() {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil && !req.EndDate.IsZero() {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, auditEventSvcTracer, "service.audit_events.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAuditEventsResponse, error) {
			return m.auditClient.ListAuditEvents(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return AuditEventListPresenter(resp), nil
}

func (m *auditEventSvcImpl) ListAuditEventResourceTypes(ctx context.Context, _ *ListAuditEventResourceTypesRequest) (*apiresource.List[constants.ObjectType], *apierror.APIError) {
	if apiErr := requireInternalAdmin(ctx); apiErr != nil {
		return nil, apiErr
	}

	values := constants.ObjectType("").EnumValues()
	types := make([]constants.ObjectType, len(values))
	for i, v := range values {
		types[i] = constants.ObjectType(v)
	}
	return apiresource.NewList(types, apiresource.PageInfo{}), nil
}

func (m *auditEventSvcImpl) GetAuditEvent(ctx context.Context, req *GetAuditEventRequest) (*apiresource.AuditEvent, *apierror.APIError) {
	if apiErr := requireInternalAdmin(ctx); apiErr != nil {
		return nil, apiErr
	}

	pbReq := &pb.GetAuditEventRequest{
		Id:       req.ID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, auditEventSvcTracer, "service.audit_events.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAuditEventResponse, error) {
			return m.auditClient.GetAuditEvent(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return AuditEventPresenter(resp.AuditEvent), nil
}

func stringsFromObjectTypes(ots []constants.ObjectType) []string {
	if len(ots) == 0 {
		return nil
	}
	out := make([]string, len(ots))
	for i, ot := range ots {
		out[i] = string(ot)
	}
	return out
}

func stringsFromActions(actions []constants.AuditAction) []string {
	if len(actions) == 0 {
		return nil
	}
	out := make([]string, len(actions))
	for i, a := range actions {
		out[i] = string(a)
	}
	return out
}
