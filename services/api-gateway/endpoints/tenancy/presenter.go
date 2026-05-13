package tenancyep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func tenancyAccountPlanPresenter(p *pb.TenancyAccountPlanProto) *apiresource.TenancyAccountPlan {
	if p == nil {
		return nil
	}

	limits := make(map[string]*int32, len(p.Limits))
	for _, entry := range p.Limits {
		if entry == nil {
			continue
		}
		if entry.Value != nil {
			v := *entry.Value
			limits[entry.Key] = &v
		} else {
			limits[entry.Key] = nil
		}
	}

	features := make(map[string]bool, len(p.Features))
	for key, enabled := range p.Features {
		features[key] = enabled
	}

	return &apiresource.TenancyAccountPlan{
		TypeID:        p.TypeId,
		Object:        constants.ObjectTypeAccountPlan,
		Name:          p.Name,
		PlanTypeCode:  p.PlanTypeCode,
		Version:       p.Version,
		PricePerSeat:  p.PricePerSeat,
		PricePerMonth: p.PricePerMonth,
		SeatMinimum:   p.SeatMinimum,
		Limits:        limits,
		Features:      features,
	}
}

func tenancyPendingRegistrationPresenter(pr *pb.TenancyPendingRegistrationProto) *apiresource.TenancyPendingRegistration {
	if pr == nil {
		return nil
	}
	return &apiresource.TenancyPendingRegistration{
		SessionID: pr.SessionId,
		PlanCode:  pr.PlanCode,
		Step:      pr.Step,
		CreatedAt: grpcutil.TimestampToTime(pr.CreatedAt),
	}
}

func TenancyPresenter(resp *pb.GetTenancyResponse) *apiresource.Tenancy {
	if resp == nil {
		return &apiresource.Tenancy{
			Object:        constants.ObjectTypeTenancy,
			Sandboxes:     []apiresource.TenancySandboxAccount{},
			OtherAccounts: []apiresource.TenancyOtherAccount{},
		}
	}

	result := &apiresource.Tenancy{
		Object:              constants.ObjectTypeTenancy,
		Sandboxes:           make([]apiresource.TenancySandboxAccount, 0, len(resp.Sandboxes)),
		OtherAccounts:       make([]apiresource.TenancyOtherAccount, 0, len(resp.OtherAccounts)),
		PendingRegistration: tenancyPendingRegistrationPresenter(resp.PendingRegistration),
	}

	if resp.CurrentAccount != nil {
		ca := &apiresource.TenancyCurrentAccount{
			ID:                       resp.CurrentAccount.Id,
			Object:                   constants.ObjectTypeAccount,
			Name:                     resp.CurrentAccount.Name,
			Type:                     resp.CurrentAccount.Type,
			OnboardingStatus:         resp.CurrentAccount.OnboardingStatus,
			Plan:                     resp.CurrentAccount.PlanCode,
			Slug:                     resp.CurrentAccount.Slug,
			InternalStripeCustomerID: resp.CurrentAccount.InternalStripeCustomerId,
			AccountPlan:              tenancyAccountPlanPresenter(resp.CurrentAccount.AccountPlan),
		}

		if resp.CurrentAccount.Role != nil {
			accountID := resp.CurrentAccount.Id
			permissions := resp.CurrentAccount.Role.Permissions
			ca.Role = &apiresource.Role{
				ID:          resp.CurrentAccount.Role.Id,
				Object:      constants.ObjectTypeRole,
				Name:        resp.CurrentAccount.Role.Name,
				TypeCode:    constants.RoleType(resp.CurrentAccount.Role.RoleTypeCode),
				Owner:       apiresource.NewOwner(&accountID),
				Permissions: &permissions,
				CreatedAt:   grpcutil.TimestampToTime(resp.CurrentAccount.Role.CreatedAt),
				UpdatedAt:   grpcutil.TimestampToTime(resp.CurrentAccount.Role.UpdatedAt),
			}
		}

		result.CurrentAccount = ca
	}

	for _, s := range resp.Sandboxes {
		result.Sandboxes = append(result.Sandboxes, apiresource.TenancySandboxAccount{
			ID:     s.Id,
			Object: constants.ObjectTypeAccount,
			Name:   s.Name,
		})
	}

	if resp.OwnerAccount != nil {
		result.OwnerAccount = &apiresource.TenancyOwnerAccount{
			ID:     resp.OwnerAccount.Id,
			Object: constants.ObjectTypeAccount,
			Name:   resp.OwnerAccount.Name,
		}
	}

	for _, o := range resp.OtherAccounts {
		result.OtherAccounts = append(result.OtherAccounts, apiresource.TenancyOtherAccount{
			ID:     o.Id,
			Object: constants.ObjectTypeAccount,
			Name:   o.Name,
			Type:   o.Type,
		})
	}

	return result
}

func CurrentUserPresenter(resp *pb.GetCurrentUserResponse) *apiresource.User {
	if resp == nil {
		return &apiresource.User{}
	}

	return &apiresource.User{
		ID:              resp.Id,
		Object:          constants.ObjectTypeUser,
		Email:           resp.Email,
		Name:            resp.Name,
		Username:        resp.Username,
		EmailVerifiedAt: grpcutil.TimestampToTimePtr(resp.EmailVerifiedAt),
		ImageUrl:        resp.ImageUrl,
		CreatedAt:       grpcutil.TimestampToTime(resp.CreatedAt),
		UpdatedAt:       grpcutil.TimestampToTime(resp.UpdatedAt),
	}
}

func CustomerAccountListPresenter(resp *pb.ListCustomerAccountsForUserResponse) *apiresource.List[apiresource.CustomerAccountSummary] {
	if resp == nil {
		return apiresource.NewList[apiresource.CustomerAccountSummary](nil, apiresource.PageInfo{})
	}

	accounts := make([]apiresource.CustomerAccountSummary, len(resp.Accounts))
	for i, a := range resp.Accounts {
		accounts[i] = apiresource.CustomerAccountSummary{
			ID:     a.Id,
			Object: constants.ObjectTypeAccount,
			Name:   a.Name,
		}
	}

	return apiresource.NewList(accounts, apiresource.PageInfo{})
}
