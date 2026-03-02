package requestlogep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/platform"
)

func RequestLogPresenter(rl *pb.RequestLogInfo) apiresource.RequestLog {
	if rl == nil {
		return apiresource.RequestLog{}
	}

	result := apiresource.RequestLog{
		ID:               rl.Id,
		Object:           constants.ObjectTypeRequestLog,
		Method:           rl.Method,
		Host:             rl.Host,
		Path:             rl.Path,
		NormalizedRoute:  rl.NormalizedRoute,
		QueryJSON:        rl.QueryJson,
		StatusCode:       rl.StatusCode,
		LatencyUs:        rl.LatencyUs,
		APIVersion:       rl.ApiVersion,
		IdentityType:     rl.IdentityType,
		ClientIP:         rl.ClientIp,
		UserAgent:        rl.UserAgent,
		Referrer:         rl.Referrer,
		ErrorCode:        rl.ErrorCode,
		ErrorMessage:     rl.ErrorMessage,
		OccurredAt:       grpcutil.TimestampToTime(rl.OccurredAt),
		CreatedAt:        grpcutil.TimestampToTime(rl.CreatedAt),
		IdempotencyKey:   rl.IdempotencyKey,
		RequestBodyJSON:  rl.BodyJson,
		ResponseBodyJSON: rl.ResponseJson,
	}

	if rl.AccountId != nil && rl.AccountName != nil {
		result.Account = &apiresource.LightAccount{
			ID:         *rl.AccountId,
			ObjectType: constants.ObjectTypeAccount,
			Name:       *rl.AccountName,
		}
	}

	if rl.Actor != nil {
		actor := &apiresource.RequestLogActor{
			ID:            rl.Actor.Id,
			ObjectType:    constants.ObjectType(rl.Actor.ObjectType),
			Name:          rl.Actor.Name,
			Email:         rl.Actor.Email,
			RedactedValue: rl.Actor.RedactedValue,
		}
		if rl.Actor.RoleId != nil && rl.Actor.RoleName != nil {
			role := &apiresource.LightRole{
				ID:         *rl.Actor.RoleId,
				ObjectType: constants.ObjectTypeRole,
				Name:       *rl.Actor.RoleName,
			}
			if rl.Actor.RoleTypeCode != nil {
				rtc := constants.RoleTypeCode(*rl.Actor.RoleTypeCode)
				role.RoleTypeCode = &rtc
			}
			actor.Role = role
		}
		result.Actor = actor
	}

	return result
}

func RequestLogListItemPresenter(rl *pb.RequestLogInfo) apiresource.RequestLogListItem {
	if rl == nil {
		return apiresource.RequestLogListItem{}
	}

	result := apiresource.RequestLogListItem{
		ID:              rl.Id,
		Object:          constants.ObjectTypeRequestLog,
		Method:          rl.Method,
		Host:            rl.Host,
		Path:            rl.Path,
		NormalizedRoute: rl.NormalizedRoute,
		QueryJSON:       rl.QueryJson,
		StatusCode:      rl.StatusCode,
		LatencyUs:       rl.LatencyUs,
		APIVersion:      rl.ApiVersion,
		IdentityType:    rl.IdentityType,
		ClientIP:        rl.ClientIp,
		UserAgent:       rl.UserAgent,
		Referrer:        rl.Referrer,
		ErrorCode:       rl.ErrorCode,
		ErrorMessage:    rl.ErrorMessage,
		OccurredAt:      grpcutil.TimestampToTime(rl.OccurredAt),
		CreatedAt:       grpcutil.TimestampToTime(rl.CreatedAt),
		IdempotencyKey:  rl.IdempotencyKey,
	}

	if rl.AccountId != nil && rl.AccountName != nil {
		result.Account = &apiresource.LightAccount{
			ID:         *rl.AccountId,
			ObjectType: constants.ObjectTypeAccount,
			Name:       *rl.AccountName,
		}
	}

	if rl.Actor != nil {
		actor := &apiresource.RequestLogActor{
			ID:            rl.Actor.Id,
			ObjectType:    constants.ObjectType(rl.Actor.ObjectType),
			Name:          rl.Actor.Name,
			Email:         rl.Actor.Email,
			RedactedValue: rl.Actor.RedactedValue,
		}
		if rl.Actor.RoleId != nil && rl.Actor.RoleName != nil {
			role := &apiresource.LightRole{
				ID:         *rl.Actor.RoleId,
				ObjectType: constants.ObjectTypeRole,
				Name:       *rl.Actor.RoleName,
			}
			if rl.Actor.RoleTypeCode != nil {
				rtc := constants.RoleTypeCode(*rl.Actor.RoleTypeCode)
				role.RoleTypeCode = &rtc
			}
			actor.Role = role
		}
		result.Actor = actor
	}

	return result
}

func RequestLogListPresenter(resp *pb.ListRequestLogsResponse) *apiresource.List[apiresource.RequestLogListItem] {
	if resp == nil {
		return apiresource.NewList[apiresource.RequestLogListItem](nil, apiresource.PageInfo{})
	}

	logs := make([]apiresource.RequestLogListItem, len(resp.RequestLogs))
	for i, rl := range resp.RequestLogs {
		logs[i] = RequestLogListItemPresenter(rl)
	}

	return apiresource.NewList(logs, mapProtoPageInfo(resp.PageInfo))
}

func mapProtoPageInfo(pi *pb.PageInfo) apiresource.PageInfo {
	if pi == nil {
		return apiresource.PageInfo{}
	}
	return apiresource.PageInfo{
		NextCursor:  pi.NextCursor,
		PrevCursor:  pi.PrevCursor,
		HasNextPage: pi.HasNextPage,
		HasPrevPage: pi.HasPrevPage,
	}
}
