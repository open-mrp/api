package requestlogep

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	corepb "github.com/augno/api/shared/proto/core"
	pb "github.com/augno/api/shared/proto/platform"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type RequestLogSvc interface {
	ListRequestLogs(ctx context.Context, req *ListRequestLogsRequest) (*apiresource.List[apiresource.RequestLog], *apierror.APIError)
	GetRequestLog(ctx context.Context, req *RetrieveRequestLogRequest) (*apiresource.RequestLog, *apierror.APIError)
}

type RequestLogSvcConfig struct {
	LoggingClient pb.LoggingServiceClient
	CoreClient    corepb.CoreServiceClient
}

type requestLogSvcImpl struct {
	loggingClient pb.LoggingServiceClient
	coreClient    corepb.CoreServiceClient
}

var requestLogSvcTracer = tracing.GetTracer("api-gateway.endpoints.request_logs.service")

func (c *RequestLogSvcConfig) validate() error {
	if c.LoggingClient == nil {
		return fmt.Errorf("request log endpoint service: logging client is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("request log endpoint service: core client is required")
	}
	return nil
}

func NewRequestLogSvc(config *RequestLogSvcConfig) RequestLogSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &requestLogSvcImpl{
		loggingClient: config.LoggingClient,
		coreClient:    config.CoreClient,
	}
}

func (m *requestLogSvcImpl) resolveRolePermissions(ctx context.Context, roleID *string) map[string]bool {
	if roleID == nil || !appctx.IsIncludeRequested(ctx, "actor.role.permissions") {
		return nil
	}
	resp, err := m.coreClient.GetRolePermissions(ctx, &corepb.GetRolePermissionsRequest{RoleId: *roleID})
	if err != nil {
		return nil
	}
	return resp.Permissions
}

func requireInternalAdmin(ctx context.Context) *apierror.APIError {
	identity, apiErr := httptransport.GetIdentity(ctx)
	if apiErr != nil {
		return apiErr
	}
	if !identity.IsInternalUser() || !identity.IsAdmin() {
		return apierror.NewAuthorizationError("Only internal administrators can access request logs.")
	}
	return nil
}

func (m *requestLogSvcImpl) ListRequestLogs(ctx context.Context, req *ListRequestLogsRequest) (*apiresource.List[apiresource.RequestLog], *apierror.APIError) {
	if apiErr := requireInternalAdmin(ctx); apiErr != nil {
		return nil, apiErr
	}

	requestedIncludes := expandedRequestLogIncludeKeys(appctx.GetRequestedIncludeKeys(ctx))

	methods := make([]string, len(req.Methods))
	for i, m := range req.Methods {
		methods[i] = string(m)
	}

	errorCodes := make([]string, len(req.ErrorCodes))
	for i, ec := range req.ErrorCodes {
		errorCodes[i] = string(ec)
	}

	actorTypes := make([]string, len(req.ActorTypes))
	for i, at := range req.ActorTypes {
		actorTypes[i] = string(at)
	}

	pbReq := &pb.ListRequestLogsRequest{
		Query:            req.Query,
		Methods:          methods,
		StatusCodes:      req.StatusCodes,
		ErrorCodes:       errorCodes,
		AccountIds:       req.AccountIDs,
		ActorIds:         req.ActorIDs,
		ActorTypes:       actorTypes,
		NormalizedRoutes: req.NormalizedRoutes,
		Hosts:            req.Hosts,
		MinLatencyUs:     req.MinLatencyUs,
		IdempotencyKey:   req.IdempotencyKey,
		Cursor:           req.Cursor,
		Limit:            req.Limit,
		Includes:         requestedIncludes,
	}

	if req.StartDate != nil && !req.StartDate.IsZero() {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil && !req.EndDate.IsZero() {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, requestLogSvcTracer, "service.request_logs.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListRequestLogsResponse, error) {
			return m.loggingClient.ListRequestLogs(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return RequestLogListPresenter(resp, func(roleID *string) map[string]bool {
		return m.resolveRolePermissions(ctx, roleID)
	}), nil
}

func (m *requestLogSvcImpl) GetRequestLog(ctx context.Context, req *RetrieveRequestLogRequest) (*apiresource.RequestLog, *apierror.APIError) {
	if apiErr := requireInternalAdmin(ctx); apiErr != nil {
		return nil, apiErr
	}

	requestedIncludes := expandedRequestLogIncludeKeys(appctx.GetRequestedIncludeKeys(ctx))

	pbReq := &pb.GetRequestLogRequest{
		Id:       req.ID,
		Includes: requestedIncludes,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, requestLogSvcTracer, "service.request_logs.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetRequestLogResponse, error) {
			return m.loggingClient.GetRequestLog(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	var roleID *string
	if resp.RequestLog != nil && resp.RequestLog.Actor != nil {
		roleID = resp.RequestLog.Actor.RoleId
	}
	perms := m.resolveRolePermissions(ctx, roleID)
	result := RequestLogPresenter(resp.RequestLog, perms)
	return &result, nil
}

func expandedRequestLogIncludeKeys(includes []string) []string {
	if includes == nil {
		return nil
	}

	expanded := make(map[string]bool, len(includes))
	for _, include := range includes {
		expanded[include] = true
		parts := strings.Split(include, ".")
		for i := 1; i < len(parts); i++ {
			expanded[strings.Join(parts[:i], ".")] = true
		}
	}

	keys := make([]string, 0, len(expanded))
	for key := range expanded {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
