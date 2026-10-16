package grpc

import (
	"context"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *gRPCHandler) ListRoles(ctx context.Context, req *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListRolesParams{
		Limit:    req.Limit,
		Includes: req.Includes,
	}
	if req.Cursor != "" {
		params.Cursor = &req.Cursor
	}
	if req.Query != "" {
		params.Query = &req.Query
	}
	if len(req.RoleTypeCodes) > 0 {
		params.RoleTypes = req.RoleTypeCodes
	}

	result, apiErr := h.roleSvc.ListRoles(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	roles := make([]*pb.RoleDetail, len(result.Roles))
	for i, r := range result.Roles {
		roles[i] = roleWithPermissionsToProto(r)
	}

	return &pb.ListRolesResponse{
		Roles: roles,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetRole(ctx context.Context, req *pb.GetRoleRequest) (*pb.GetRoleResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.roleSvc.GetRole(ctx, req.Id, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetRoleResponse{
		Role: roleWithPermissionsToProto(result),
	}, nil
}

func (h *gRPCHandler) CreateRole(ctx context.Context, req *pb.CreateRoleRequest) (*pb.CreateRoleResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateRoleParams{
		Name:    req.Name,
		Access:  roleAccessFromProto(req.Access),
		Revokes: req.Revokes,
	}
	if req.TemplateCode != nil {
		t := constants.RoleTemplate(*req.TemplateCode)
		params.Template = &t
	}
	if req.HasGrants {
		grants := req.Grants
		params.Grants = &grants
	}
	result, apiErr := h.roleSvc.CreateRole(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateRoleResponse{
		Role: roleWithPermissionsToProto(result),
	}, nil
}

func (h *gRPCHandler) UpdateRole(ctx context.Context, req *pb.UpdateRoleRequest) (*pb.UpdateRoleResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateRoleParams{
		RoleID: req.Id,
	}
	if req.Name != "" {
		params.Name = &req.Name
	}
	params.Access = roleAccessFromProto(req.Access)
	if req.HasGrants {
		grants := req.Grants
		params.Grants = &grants
	}
	if req.HasRevokes {
		revokes := req.Revokes
		params.Revokes = &revokes
	}

	result, apiErr := h.roleSvc.UpdateRole(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateRoleResponse{
		Role: roleWithPermissionsToProto(result),
	}, nil
}

func (h *gRPCHandler) DeleteRole(ctx context.Context, req *pb.DeleteRoleRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.roleSvc.DeleteRole(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) BatchGetRolesByIDs(ctx context.Context, req *pb.BatchGetRolesByIDsRequest) (*pb.BatchGetRolesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	results, apiErr := h.roleSvc.BatchGetRolesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	roles := make([]*pb.RoleDetail, len(results))
	for i, rwp := range results {
		roles[i] = roleWithPermissionsToProto(rwp)
	}

	return &pb.BatchGetRolesByIDsResponse{
		Roles: roles,
	}, nil
}

func roleWithPermissionsToProto(rwp *domain.RoleWithPermissions) *pb.RoleDetail {
	if rwp == nil {
		return nil
	}

	detail := roleToProto(&rwp.Role, nil)
	def := rwp.Definition()
	detail.Access = map[string]string{}
	for area, level := range def.Access {
		detail.Access[string(area)] = string(level)
	}
	detail.Grants = def.Grants
	detail.Revokes = def.Revokes
	detail.EffectivePermissions = types.EffectivePermissions(def.Access, def.Grants, def.Revokes)
	detail.TemplateCode = rwp.Template

	if rwp.Permissions != nil {
		perms := make([]*pb.RolePermissionDetail, len(rwp.Permissions))
		for i, p := range rwp.Permissions {
			perms[i] = rolePermissionToProto(p)
		}
		detail.Permissions = perms
	}

	return detail
}

func roleToProto(r *domain.Role, permissions []*pb.RolePermissionDetail) *pb.RoleDetail {
	if r == nil {
		return nil
	}

	var accountID string
	if r.AccountID != nil {
		accountID = *r.AccountID
	}

	return &pb.RoleDetail{
		Id:           r.ID,
		Name:         r.Name,
		RoleTypeCode: r.RoleType,
		AccountId:    accountID,
		CreatedAt:    timestamppb.New(r.CreatedAt),
		UpdatedAt:    timestamppb.New(r.UpdatedAt),
		Permissions:  permissions,
	}
}

func rolePermissionToProto(rp *domain.RolePermission) *pb.RolePermissionDetail {
	if rp == nil {
		return nil
	}

	return &pb.RolePermissionDetail{
		Id:             rp.ID,
		PermissionCode: rp.PermissionCode,
		Create:         rp.Create,
		Read:           rp.Read,
		Update:         rp.Update,
		Delete:         rp.Delete,
		CreatedAt:      timestamppb.New(rp.CreatedAt),
		UpdatedAt:      timestamppb.New(rp.UpdatedAt),
	}
}

func roleAccessFromProto(in map[string]string) types.RoleAccess {
	if len(in) == 0 {
		return nil
	}
	out := types.RoleAccess{}
	for k, v := range in {
		out[constants.PermissionArea(k)] = constants.AccessLevel(v)
	}
	return out
}
