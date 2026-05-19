package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/patch"
)

type AccountGroup struct {
	ID                   string
	OwnerAccountID       string  `audit:"account_id"`
	Name                 string  `audit:"name"`
	Description          *string `audit:"description"`
	CommissionPolicyCode string  `audit:"commission_policy_code"`
	FreightPolicyCode    string  `audit:"freight_policy_code"`
	AccountGroupTypeCode string  `audit:"account_group_type_code"`
	RegistrationFlowID   *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ListAccountGroupsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
	Type      *string
}

type ListAccountGroupsResult struct {
	AccountGroups []*AccountGroup
	PageInfo      pagination.PageInfo
}

type CreateAccountGroupParams struct {
	AccountID            string
	Name                 string
	Description          *string
	AccountGroupTypeCode string
	CommissionPolicyCode string
	FreightPolicyCode    string
}

type UpdateAccountGroupParams struct {
	AccountID            string
	AccountGroupID       string
	Name                 *string
	Description          patch.Field[string]
	CommissionPolicyCode *string
	FreightPolicyCode    *string
}

type DeleteAccountGroupParams struct {
	AccountID      string
	AccountGroupID string
}
