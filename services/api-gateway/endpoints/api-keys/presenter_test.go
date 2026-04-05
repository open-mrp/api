package apikeyep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func newFullAPIKeyInfo() *pb.APIKeyInfo {
	roleID := "rl_01abc"
	roleName := "Admin"
	roleTypeCode := "admin"

	return &pb.APIKeyInfo{
		Id:            "apke_01abc",
		Name:          "Production Key",
		RedactedValue: "aug_sk_prod_****abcd",
		RoleId:        &roleID,
		RoleName:      &roleName,
		RoleTypeCode:  &roleTypeCode,
		CreatedAt:     timestamppb.Now(),
		UpdatedAt:     timestamppb.Now(),
		LastUsedAt:    timestamppb.Now(),
		ExpiresAt:     timestamppb.Now(),
	}
}

var testPerms = map[string]bool{"customer:read": true}

func TestAPIKeyPresenter(t *testing.T) {
	t.Parallel()
	result := APIKeyPresenter(newFullAPIKeyInfo(), testPerms)
	resourcetest.ValidateResourceStruct(t, "APIKey", result)
}

func TestAPIKeyPresenter_RoleWithoutTypeCode(t *testing.T) {
	t.Parallel()
	roleID := "rl_01abc"
	roleName := "Admin"

	key := &pb.APIKeyInfo{
		Id:            "apke_01abc",
		Name:          "Production Key",
		RedactedValue: "aug_sk_prod_****abcd",
		RoleId:        &roleID,
		RoleName:      &roleName,
		RoleTypeCode:  nil,
		CreatedAt:     timestamppb.Now(),
		UpdatedAt:     timestamppb.Now(),
	}

	result := APIKeyPresenter(key, nil)
	resourcetest.ValidateResourceStruct(t, "APIKey(RoleWithoutTypeCode)", result)
}

func TestAPIKeyCreatedPresenter(t *testing.T) {
	t.Parallel()
	resp := &pb.CreateAPIKeyResponse{
		ApiKeySecret: "aug_sk_prod_fullsecret",
		ApiKey:       newFullAPIKeyInfo(),
	}

	result := APIKeyCreatedPresenter(resp, testPerms)
	resourcetest.ValidateResourceStruct(t, "CreatedAPIKey", result)
}

func TestAPIKeyRotatedPresenter(t *testing.T) {
	t.Parallel()
	resp := &pb.RotateAPIKeyResponse{
		ApiKeySecret: "aug_sk_prod_rotatedsecret",
		ApiKey:       newFullAPIKeyInfo(),
	}

	result := APIKeyRotatedPresenter(resp, testPerms)
	resourcetest.ValidateResourceStruct(t, "RotatedAPIKey", result)
}
