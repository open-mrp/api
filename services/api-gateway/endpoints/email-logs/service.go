package emaillogep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type EmailLogSvc interface {
	ListEmailLogs(ctx context.Context, req *ListEmailLogsRequest) (*apiresource.List[apiresource.EmailLog], *apierror.APIError)
	GetEmailLog(ctx context.Context, req *RetrieveEmailLogRequest) (*apiresource.EmailLog, *apierror.APIError)
}

type EmailLogSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type emailLogSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var emailLogSvcTracer = tracing.GetTracer("api-gateway.endpoints.email_logs.service")

func (c *EmailLogSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("email log endpoint service: core client is required")
	}
	return nil
}

func NewEmailLogSvc(config *EmailLogSvcConfig) EmailLogSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &emailLogSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *emailLogSvcImpl) ListEmailLogs(ctx context.Context, req *ListEmailLogsRequest) (*apiresource.List[apiresource.EmailLog], *apierror.APIError) {
	pbReq := &pb.ListEmailLogsRequest{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		Includes: resourcekit.FilterIncludes(ctx, "sent_by"),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, emailLogSvcTracer, "service.email_logs.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListEmailLogsResponse, error) {
			return m.coreClient.ListEmailLogs(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	if resp == nil {
		return apiresource.NewList[apiresource.EmailLog](nil, apiresource.PageInfo{}), nil
	}

	meta := resourcekit.GetLoadMeta(ctx)
	emailLogs := make([]apiresource.EmailLog, len(resp.EmailLogs))
	for i, el := range resp.EmailLogs {
		emailLogs[i] = emailLogFromProto(el)
		stashEmailLogMeta(meta, el)
	}

	return apiresource.NewList(emailLogs, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *emailLogSvcImpl) GetEmailLog(ctx context.Context, req *RetrieveEmailLogRequest) (*apiresource.EmailLog, *apierror.APIError) {
	pbReq := &pb.GetEmailLogRequest{
		Id:       req.EmailLogID,
		Includes: resourcekit.FilterIncludes(ctx, "sent_by"),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, emailLogSvcTracer, "service.email_logs.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetEmailLogResponse, error) {
			return m.coreClient.GetEmailLog(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := emailLogFromProto(resp.EmailLog)
	stashEmailLogMeta(meta, resp.EmailLog)
	return &result, nil
}

func emailLogFromProto(el *pb.EmailLogInfo) apiresource.EmailLog {
	if el == nil {
		return apiresource.EmailLog{}
	}

	result := apiresource.EmailLog{
		ID:         el.Id,
		Object:     constants.ObjectTypeEmailLog,
		SendStatus: emailSendStatus(el.HasSent),
		Recipients: el.Recipients,
		Subject:    el.Subject,
		Filename:   el.Filename,
		CreatedAt:  grpcutil.TimestampToTime(el.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(el.UpdatedAt),
	}

	if result.Recipients == nil {
		result.Recipients = []string{}
	}

	return result
}

func stashEmailLogMeta(meta *resourcekit.LoadMeta, el *pb.EmailLogInfo) {
	if el == nil || el.SentBy == nil || el.SentBy.Id == "" {
		return
	}
	actor := apiresource.NewActor(
		el.SentBy.Id,
		constants.ActorType(el.SentBy.ActorType),
		el.SentBy.Name,
		el.SentBy.Handle,
	)
	meta.Set(constants.ObjectTypeEmailLog, el.Id, "sent_by", actor)
}

func emailSendStatus(hasSent bool) constants.EmailSendStatus {
	if hasSent {
		return constants.EmailSendStatusSent
	}
	return constants.EmailSendStatusPending
}
