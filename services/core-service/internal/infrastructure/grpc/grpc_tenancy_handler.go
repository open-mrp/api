package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func tenancyRoleToProto(r *domain.TenancyRole) *pb.TenancyRoleProto {
	if r == nil {
		return nil
	}
	return &pb.TenancyRoleProto{
		Id:           r.ID,
		Name:         r.Name,
		RoleTypeCode: r.RoleType,
		Permissions:  r.Permissions,
		CreatedAt:    timestamppb.New(r.CreatedAt),
		UpdatedAt:    timestamppb.New(r.UpdatedAt),
	}
}

func tenancyAccountPlanToProto(p *domain.TenancyAccountPlan) *pb.TenancyAccountPlanProto {
	if p == nil {
		return nil
	}

	limits := make([]*pb.TenancyAccountPlanLimitProto, 0, len(p.Limits))
	for key, value := range p.Limits {
		entry := &pb.TenancyAccountPlanLimitProto{Key: key}
		if value != nil {
			v := *value
			entry.Value = &v
		}
		limits = append(limits, entry)
	}

	features := make(map[string]bool, len(p.Features))
	for key, enabled := range p.Features {
		features[key] = enabled
	}

	proto := &pb.TenancyAccountPlanProto{
		TypeId:       p.TypeID,
		Name:         p.Name,
		PlanTypeCode: p.PlanTypeCode,
		Version:      p.Version,
		PricePerSeat: p.PricePerSeat,
		Limits:       limits,
		Features:     features,
	}
	if p.PricePerMonth != nil {
		proto.PricePerMonth = p.PricePerMonth
	}
	if p.SeatMinimum != nil {
		proto.SeatMinimum = p.SeatMinimum
	}
	return proto
}

func tenancyPendingRegistrationToProto(pr *domain.TenancyPendingRegistration) *pb.TenancyPendingRegistrationProto {
	if pr == nil {
		return nil
	}
	return &pb.TenancyPendingRegistrationProto{
		SessionId: pr.SessionID,
		PlanCode:  pr.PlanCode,
		Step:      pr.Step,
		CreatedAt: timestamppb.New(pr.CreatedAt),
	}
}

func tenancyCurrentAccountToProto(ca *domain.TenancyCurrentAccount) *pb.TenancyCurrentAccountProto {
	if ca == nil {
		return nil
	}
	proto := &pb.TenancyCurrentAccountProto{
		Id:                       ca.ID,
		Name:                     ca.Name,
		Type:                     ca.Type,
		OnboardingStatus:         ca.OnboardingStatus,
		PlanCode:                 ca.PlanCode,
		Role:                     tenancyRoleToProto(ca.Role),
		InternalStripeCustomerId: ca.InternalStripeCustomerID,
		AccountPlan:              tenancyAccountPlanToProto(ca.AccountPlan),
	}
	if ca.Slug != nil {
		proto.Slug = ca.Slug
	}
	return proto
}

func tenancyAccountSummaryToProto(s domain.TenancySandbox) *pb.TenancyAccountSummaryProto {
	return &pb.TenancyAccountSummaryProto{
		Id:   s.ID,
		Name: s.Name,
	}
}

func tenancyOwnerAccountToProto(o *domain.TenancyOwnerAccount) *pb.TenancyAccountSummaryProto {
	if o == nil {
		return nil
	}
	return &pb.TenancyAccountSummaryProto{
		Id:   o.ID,
		Name: o.Name,
	}
}

func tenancyOtherAccountToProto(o domain.TenancyOtherAccount) *pb.TenancyOtherAccountProto {
	return &pb.TenancyOtherAccountProto{
		Id:   o.ID,
		Name: o.Name,
		Type: o.Type,
	}
}

func tenancyToProto(t *domain.Tenancy) *pb.GetTenancyResponse {
	if t == nil {
		return nil
	}

	resp := &pb.GetTenancyResponse{
		HasTenancy:          t.HasTenancy,
		CurrentAccount:      tenancyCurrentAccountToProto(t.CurrentAccount),
		OwnerAccount:        tenancyOwnerAccountToProto(t.OwnerAccount),
		PendingRegistration: tenancyPendingRegistrationToProto(t.PendingRegistration),
	}

	if t.Sandboxes != nil {
		resp.Sandboxes = make([]*pb.TenancyAccountSummaryProto, len(t.Sandboxes))
		for i, s := range t.Sandboxes {
			resp.Sandboxes[i] = tenancyAccountSummaryToProto(s)
		}
	}

	if t.OtherAccounts != nil {
		resp.OtherAccounts = make([]*pb.TenancyOtherAccountProto, len(t.OtherAccounts))
		for i, o := range t.OtherAccounts {
			resp.OtherAccounts[i] = tenancyOtherAccountToProto(o)
		}
	}

	return resp
}

func currentUserToProto(u *domain.UserRecord) *pb.GetCurrentUserResponse {
	if u == nil {
		return nil
	}

	resp := &pb.GetCurrentUserResponse{
		Id:        u.ID,
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}

	if u.Name != nil {
		resp.Name = u.Name
	}
	if u.Email != nil {
		resp.Email = u.Email
	}
	if u.Username != nil {
		resp.Username = u.Username
	}
	if u.ImageURL != nil {
		resp.ImageUrl = u.ImageURL
	}
	if u.EmailVerified != nil {
		resp.EmailVerifiedAt = timestamppb.New(*u.EmailVerified)
	}

	return resp
}

func (h *gRPCHandler) GetTenancy(ctx context.Context, req *pb.GetTenancyRequest) (*pb.GetTenancyResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	tenancy, apiErr := h.tenancySvc.GetTenancy(ctx, req.UserId, req.TargetAccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return tenancyToProto(tenancy), nil
}

func (h *gRPCHandler) SwitchAccount(ctx context.Context, req *pb.SwitchTenancyAccountRequest) (*pb.GetTenancyResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	tenancy, apiErr := h.tenancySvc.SwitchAccount(ctx, req.UserId, req.AccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return tenancyToProto(tenancy), nil
}

func (h *gRPCHandler) GetCurrentUser(ctx context.Context, req *pb.GetCurrentUserRequest) (*pb.GetCurrentUserResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	user, apiErr := h.tenancySvc.GetCurrentUser(ctx, req.UserId, req.TargetAccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return currentUserToProto(user), nil
}

func (h *gRPCHandler) ListCustomerAccountsForUser(ctx context.Context, req *pb.ListCustomerAccountsForUserRequest) (*pb.ListCustomerAccountsForUserResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	accounts, apiErr := h.tenancySvc.ListCustomerAccountsForUser(ctx, req.UserId, req.VendorAccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbAccounts := make([]*pb.CustomerAccountSummaryProto, len(accounts))
	for i, a := range accounts {
		pbAccounts[i] = &pb.CustomerAccountSummaryProto{
			Id:   a.ID,
			Name: a.Name,
		}
	}

	return &pb.ListCustomerAccountsForUserResponse{
		Accounts: pbAccounts,
	}, nil
}
