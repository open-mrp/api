package emaillogep

import (
	"context"
	"fmt"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/chatmap"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/recordstub"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	if apiErr := req.CreatedRange.Validate(); apiErr != nil {
		return nil, apiErr
	}
	pbReq := &pb.ListEmailLogsRequest{
		Cursor:     req.Cursor,
		Limit:      req.Limit,
		Query:      req.Query,
		RelatedIds: req.RelatedIDs,
	}
	if req.CreatedAfter != nil {
		pbReq.CreatedAfter = timestamppb.New(*req.CreatedAfter)
	}
	if req.CreatedBefore != nil {
		pbReq.CreatedBefore = timestamppb.New(*req.CreatedBefore)
	}
	for _, s := range req.SendStatuses {
		pbReq.SendStatuses = append(pbReq.SendStatuses, string(s))
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

	emailLogs := make([]apiresource.EmailLog, len(resp.EmailLogs))
	actors := make([]*apiresource.ActorRef, len(resp.EmailLogs))
	for i, el := range resp.EmailLogs {
		emailLogs[i] = emailLogFromProto(ctx, el)
		actors[i] = emailLogs[i].SentBy
	}
	resourceloaders.ResolveActorRefs(ctx, actors)

	return apiresource.NewList(emailLogs, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *emailLogSvcImpl) GetEmailLog(ctx context.Context, req *RetrieveEmailLogRequest) (*apiresource.EmailLog, *apierror.APIError) {
	pbReq := &pb.GetEmailLogRequest{Id: req.EmailLogID}

	resp, apiErr := grpcutil.CallRPC(ctx, emailLogSvcTracer, "service.email_logs.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetEmailLogResponse, error) {
			return m.coreClient.GetEmailLog(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := emailLogFromProto(ctx, resp.EmailLog)
	resourceloaders.ResolveActorRefs(ctx, []*apiresource.ActorRef{result.SentBy})
	return &result, nil
}

// emailLogFromProto maps an email log and stashes its related record for the `related.record` include.
func emailLogFromProto(ctx context.Context, el *pb.EmailLogInfo) apiresource.EmailLog {
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
		SentBy:     emailLogSentBy(el.SentBy),
		Related:    &apiresource.EmailLogRelated{Object: constants.ObjectTypeEmailLogRelated},
		CreatedAt:  grpcutil.TimestampToTime(el.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(el.UpdatedAt),
	}

	if result.Recipients == nil {
		result.Recipients = []string{}
	}
	if el.GetRelatedType() != "" && el.GetRelatedId() != "" {
		include.GetLoadMeta(ctx).Set(constants.ObjectTypeEmailLog, el.Id, chatmap.MetaRelatedRecord,
			recordstub.Ref{Type: constants.RecordType(el.GetRelatedType()), ID: el.GetRelatedId()})
	}

	return result
}

// emailLogSentBy is the email's sender. An email logged without one was sent by the platform itself.
func emailLogSentBy(a *pb.EmailLogActor) *apiresource.ActorRef {
	if a != nil && a.Id != "" {
		if ref := resourceloaders.ActorRefFromRecorded(constants.ActorType(a.ActorType), a.Id, a.Name); ref != nil {
			return ref
		}
	}
	return resourceloaders.ActorRefFromRecorded(constants.ActorTypeSystem, "", nil)
}

func emailSendStatus(hasSent bool) constants.EmailSendStatus {
	if hasSent {
		return constants.EmailSendStatusSent
	}
	return constants.EmailSendStatusPending
}
