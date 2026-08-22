package domain

import (
	"time"

	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/pagination"
)

type AccountGroup struct {
	ID                   string
	OwnerAccountID       string  `audit:"account_id"`
	Name                 string  `audit:"name"`
	Description          *string `audit:"description"`
	CommissionPolicyCode string  `audit:"commission_policy_code"`
	FreightPolicyCode    string  `audit:"freight_policy_code"`
	AccountGroupTypeCode string  `audit:"account_group_type_code"`
	// DefaultLeadTimeDays is inherited by every customer in the group that has not set its own. Nil falls through to the account default.
	DefaultLeadTimeDays *int32 `audit:"default_lead_time_days"`
	RegistrationFlowID  *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
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
	DefaultLeadTimeDays  *int32
}

type UpdateAccountGroupParams struct {
	AccountID            string
	AccountGroupID       string
	Name                 *string
	Description          field.Clearable[string]
	CommissionPolicyCode *string
	FreightPolicyCode    *string
	DefaultLeadTimeDays  field.Clearable[int32]
}

type DeleteAccountGroupParams struct {
	AccountID      string
	AccountGroupID string
}
