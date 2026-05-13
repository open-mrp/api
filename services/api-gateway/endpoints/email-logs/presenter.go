package emaillogep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func EmailLogPresenter(el *pb.EmailLogInfo) apiresource.EmailLog {
	if el == nil {
		return apiresource.EmailLog{}
	}

	result := apiresource.EmailLog{
		ID:           el.Id,
		Object:       constants.ObjectTypeEmailLog,
		SendStatus:   emailSendStatus(el.HasSent),
		Recipients:   el.Recipients,
		Subject:      el.Subject,
		Filename:     el.Filename,
		SESMessageID: el.SesMessageId,
		CreatedAt:    grpcutil.TimestampToTime(el.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(el.UpdatedAt),
	}

	if result.Recipients == nil {
		result.Recipients = []string{}
	}

	if el.SentBy != nil && el.SentBy.Id != "" {
		result.SentBy = apiresource.NewActor(
			el.SentBy.Id,
			constants.ActorType(el.SentBy.ActorType),
			el.SentBy.Name,
			el.SentBy.Handle,
		)
	}

	return result
}

func emailSendStatus(hasSent bool) constants.EmailSendStatus {
	if hasSent {
		return constants.EmailSendStatusSent
	}
	return constants.EmailSendStatusPending
}

func EmailLogListPresenter(resp *pb.ListEmailLogsResponse) *apiresource.List[apiresource.EmailLog] {
	if resp == nil {
		return apiresource.NewList[apiresource.EmailLog](nil, apiresource.PageInfo{})
	}

	emailLogs := make([]apiresource.EmailLog, len(resp.EmailLogs))
	for i, el := range resp.EmailLogs {
		emailLogs[i] = EmailLogPresenter(el)
	}

	return apiresource.NewList(emailLogs, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
