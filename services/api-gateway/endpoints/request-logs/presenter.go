package requestlogep

import (
	"context"
	"encoding/json"
	"slices"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/platform"
)

func RequestLogPresenter(rl *pb.RequestLogInfo, permissions map[string]bool) apiresource.RequestLog {
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
		QueryJSON:        rawMessageFromOptionalString(rl.QueryJson),
		StatusCode:       rl.StatusCode,
		LatencyUs:        rl.LatencyUs,
		APIVersion:       rl.ApiVersion,
		ClientIP:         rl.ClientIp,
		UserAgent:        rl.UserAgent,
		Referrer:         rl.Referrer,
		ErrorCode:        rl.ErrorCode,
		ErrorMessage:     rl.ErrorMessage,
		OccurredAt:       grpcutil.TimestampToTime(rl.OccurredAt),
		CreatedAt:        grpcutil.TimestampToTime(rl.CreatedAt),
		IdempotencyKey:   rl.IdempotencyKey,
		RequestBodyJSON:  rawMessageFromOptionalString(rl.BodyJson),
		ResponseBodyJSON: rawMessageFromOptionalString(rl.ResponseJson),
	}

	if rl.AccountId != nil && rl.AccountName != nil {
		result.Account = &apiresource.Account{
			ID:     *rl.AccountId,
			Object: constants.ObjectTypeAccount,
			Name:   *rl.AccountName,
		}
		if rl.AccountCreatedAt != nil {
			result.Account.CreatedAt = rl.AccountCreatedAt.AsTime()
		}
		if rl.AccountUpdatedAt != nil {
			result.Account.UpdatedAt = rl.AccountUpdatedAt.AsTime()
		}
	}

	if rl.Actor != nil {
		actorType := constants.ActorType(rl.Actor.ActorType)
		var handle *string
		switch actorType {
		case constants.ActorTypeAPIKey:
			handle = rl.Actor.RedactedValue
		case constants.ActorTypeUser:
			handle = rl.Actor.Email
		}

		actor := apiresource.NewActor(rl.Actor.Id, actorType, rl.Actor.Name, handle)
		if rl.Actor.RoleId != nil && rl.Actor.RoleName != nil && rl.Actor.RoleTypeCode != nil {
			rolePermissions := rolePermissionsFromMap(permissions)
			actor.Role = &apiresource.Role{
				ID:          *rl.Actor.RoleId,
				Object:      constants.ObjectTypeRole,
				Name:        *rl.Actor.RoleName,
				TypeCode:    constants.RoleType(*rl.Actor.RoleTypeCode),
				Permissions: &rolePermissions,
				Owner:       apiresource.SystemOwner(),
			}
		}
		result.Actor = actor
	}

	return result
}

func rolePermissionsFromMap(permissions map[string]bool) []string {
	if len(permissions) == 0 {
		return nil
	}

	result := make([]string, 0, len(permissions))
	for permission, allowed := range permissions {
		if allowed {
			result = append(result, permission)
		}
	}
	slices.Sort(result)
	return result
}

func RequestLogListPresenter(ctx context.Context, resp *pb.ListRequestLogsResponse, permResolver func(roleID *string) map[string]bool) *apiresource.List[apiresource.RequestLog] {
	if resp == nil {
		return apiresource.NewList[apiresource.RequestLog](nil, apiresource.PageInfo{})
	}

	logs := make([]apiresource.RequestLog, len(resp.RequestLogs))
	for i, rl := range resp.RequestLogs {
		var roleID *string
		if rl.Actor != nil {
			roleID = rl.Actor.RoleId
		}
		log := RequestLogPresenter(rl, permResolver(roleID))
		log.QueryJSON = nil
		log.RequestBodyJSON = nil
		log.ResponseBodyJSON = nil
		logs[i] = log
	}

	return apiresource.NewList(logs, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}

func rawMessageFromOptionalString(s *string) json.RawMessage {
	if s == nil || *s == "" {
		return nil
	}
	return json.RawMessage(*s)
}
