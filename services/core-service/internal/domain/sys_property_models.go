package domain

import (
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
)

type SysProperty struct {
	ID        string
	TypeID    string
	TypeCode  constants.SysPropertyTypeCode `audit:"type_code"`
	TypeName  string                        `audit:"type_name"`
	Value     int32                         `audit:"value"`
	AccountID string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListSysPropertiesParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
}

type ListSysPropertiesResult struct {
	SysProperties []*SysProperty
	PageInfo      pagination.PageInfo
}

type UpdateSysPropertyParams struct {
	AccountID string
	ID        string
	Value     *int32
}
