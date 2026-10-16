package requestlogep

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/platform"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type RequestLogSvc interface {
	ListRequestLogs(ctx context.Context, req *ListRequestLogsRequest) (*apiresource.List[apiresource.RequestLog], *apierror.APIError)
	GetRequestLog(ctx context.Context, req *RetrieveRequestLogRequest) (*apiresource.RequestLog, *apierror.APIError)
	GetRequestLogBodies(ctx context.Context, req *RetrieveRequestLogBodiesRequest) (*apiresource.RequestLogBodies, *apierror.APIError)
}

type RequestLogSvcConfig struct {
	// LoggingClient (required) is the platform-service logging gRPC client.
	LoggingClient pb.LoggingServiceClient
}

type requestLogSvcImpl struct {
	loggingClient pb.LoggingServiceClient
}

var requestLogSvcTracer = tracing.GetTracer("api-gateway.endpoints.request_logs.service")

func (c *RequestLogSvcConfig) validate() error {
	if c.LoggingClient == nil {
		return fmt.Errorf("request log endpoint service: logging client is required")
	}
	return nil
}

func NewRequestLogSvc(config *RequestLogSvcConfig) RequestLogSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &requestLogSvcImpl{
		loggingClient: config.LoggingClient,
	}
}

// logIncludes are what every read of a log needs from platform-service: the actor and query parameters, which are always rendered, and the account when it is expanded.
func logIncludes(ctx context.Context) []string {
	return append(include.FilterIncludes(ctx, "account"), "actor", "query_params")
}

func (m *requestLogSvcImpl) ListRequestLogs(ctx context.Context, req *ListRequestLogsRequest) (*apiresource.List[apiresource.RequestLog], *apierror.APIError) {
	pbReq := &pb.ListRequestLogsRequest{
		Query:             req.Query,
		Methods:           stringsOf(req.Methods),
		StatusCodes:       req.StatusCodes,
		StatusCodeClasses: req.StatusCodeClasses,
		ErrorCodes:        stringsOf(req.ErrorCodes),
		ExcludeErrorCodes: stringsOf(req.ExcludeErrorCodes),
		ActorAccountIds:   req.ActorAccountIDs,
		TargetAccountIds:  req.TargetAccountIDs,
		ActorIds:          req.ActorIDs,
		ActorTypes:        stringsOf(req.ActorTypes),
		NormalizedRoutes:  req.NormalizedRoutes,
		Hosts:             req.Hosts,
		MinLatencyUs:      req.MinLatencyUs,
		IdempotencyKey:    req.IdempotencyKey,
		Cursor:            req.Cursor,
		Limit:             req.Limit,
		Includes:          logIncludes(ctx),
	}

	if req.OccurredAfter != nil && !req.OccurredAfter.IsZero() {
		pbReq.StartDate = timestamppb.New(*req.OccurredAfter)
	}
	if req.OccurredBefore != nil && !req.OccurredBefore.IsZero() {
		pbReq.EndDate = timestamppb.New(*req.OccurredBefore)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, requestLogSvcTracer, "service.request_logs.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListRequestLogsResponse, error) {
			return m.loggingClient.ListRequestLogs(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	if resp == nil {
		return apiresource.NewList[apiresource.RequestLog](nil, apiresource.PageInfo{}), nil
	}

	meta := include.GetLoadMeta(ctx)
	logs := make([]apiresource.RequestLog, len(resp.RequestLogs))
	actors := make([]*apiresource.ActorRef, 0, len(resp.RequestLogs))
	for i, rl := range resp.RequestLogs {
		logs[i] = resourceloaders.RequestLogFromProto(rl)
		stashRequestLogMeta(meta, rl)
		actors = append(actors, logs[i].Actor)
	}
	resourceloaders.ResolveActorRefs(ctx, actors)

	return apiresource.NewList(logs, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *requestLogSvcImpl) GetRequestLog(ctx context.Context, req *RetrieveRequestLogRequest) (*apiresource.RequestLog, *apierror.APIError) {
	includes := logIncludes(ctx)
	includes = append(includes, include.FilterIncludes(ctx, "request_body", "response_body")...)
	rl, apiErr := m.getRequestLog(ctx, req.ID, includes)
	if apiErr != nil {
		return nil, apiErr
	}

	result := resourceloaders.RequestLogFromProto(rl)
	stashRequestLogMeta(include.GetLoadMeta(ctx), rl)
	resourceloaders.ResolveActorRefs(ctx, []*apiresource.ActorRef{result.Actor})
	return &result, nil
}

func (m *requestLogSvcImpl) GetRequestLogBodies(ctx context.Context, req *RetrieveRequestLogBodiesRequest) (*apiresource.RequestLogBodies, *apierror.APIError) {
	rl, apiErr := m.getRequestLog(ctx, req.ID, []string{"request_body", "response_body"})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.RequestLogBodies{
		Object:       constants.ObjectTypeRequestLogBodies,
		RequestBody:  bodyOrEmpty(rl.BodyJson),
		ResponseBody: bodyOrEmpty(rl.ResponseJson),
	}, nil
}

func (m *requestLogSvcImpl) getRequestLog(ctx context.Context, id string, includes []string) (*pb.RequestLogInfo, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, requestLogSvcTracer, "service.request_logs.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetRequestLogResponse, error) {
			return m.loggingClient.GetRequestLog(ctx, &pb.GetRequestLogRequest{Id: id, Includes: includes}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return resp.RequestLog, nil
}

func bodyOrEmpty(s *string) json.RawMessage {
	if body := rawMessageFromString(s); body != nil {
		return body
	}
	return json.RawMessage(`{}`)
}

// stashRequestLogMeta preheats the log's expandable account and preview.9's bodies into the load meta.
func stashRequestLogMeta(meta *include.LoadMeta, rl *pb.RequestLogInfo) {
	if rl == nil {
		return
	}

	if rl.AccountId != nil && rl.AccountName != nil {
		account := &apiresource.Account{
			ID:     *rl.AccountId,
			Object: constants.ObjectTypeAccount,
			Name:   *rl.AccountName,
		}
		if rl.AccountCreatedAt != nil {
			account.CreatedAt = rl.AccountCreatedAt.AsTime()
		}
		if rl.AccountUpdatedAt != nil {
			account.UpdatedAt = rl.AccountUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypeRequestLog, rl.Id, "account", account)
	}

	if body := rawMessageFromString(rl.BodyJson); body != nil {
		meta.Set(constants.ObjectTypeRequestLog, rl.Id, "request_body", body)
	}
	if body := rawMessageFromString(rl.ResponseJson); body != nil {
		meta.Set(constants.ObjectTypeRequestLog, rl.Id, "response_body", body)
	}
}

func rawMessageFromString(s *string) json.RawMessage {
	if s == nil || *s == "" {
		return nil
	}
	if json.Valid([]byte(*s)) {
		return json.RawMessage(*s)
	}
	encoded, err := json.Marshal(*s)
	if err != nil {
		return nil
	}
	return json.RawMessage(encoded)
}

func stringsOf[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}
