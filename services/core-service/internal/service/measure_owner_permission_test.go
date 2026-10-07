package service

import (
	"testing"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
)

// A measure takes the update permission of every resource it belongs to, and one that belongs to nothing in the account reads as not found.
func TestMeasureOwnerPermission(t *testing.T) {
	t.Parallel()
	svc := &measureSvcImpl{}
	item, step := constants.ObjectTypeItem, constants.ObjectTypeProductionStep

	cases := []struct {
		name   string
		owners []constants.ObjectType
		holds  []string
		want   apierror.ErrorCode
	}{
		{"an item's measure with items:update", []constants.ObjectType{item}, []string{"items:update"}, ""},
		{"an item's measure with production_steps:update", []constants.ObjectType{item}, []string{"production_steps:update"}, apierror.ErrorCodeInsufficientPerms},
		{"a step's measure with production_steps:update", []constants.ObjectType{step}, []string{"production_steps:update"}, ""},
		{"a step's measure with items:update", []constants.ObjectType{step}, []string{"items:update"}, apierror.ErrorCodeInsufficientPerms},
		{"a department's measure with departments:update", []constants.ObjectType{constants.ObjectTypeDepartment}, []string{"departments:update"}, ""},
		{"a measure two resources share with one of their permissions", []constants.ObjectType{item, step}, []string{"items:update"}, apierror.ErrorCodeInsufficientPerms},
		{"a measure two resources share with both permissions", []constants.ObjectType{item, step}, []string{"items:update", "production_steps:update"}, ""},
		{"a measure nothing in the account holds", nil, []string{"items:update", "production_steps:update"}, apierror.ErrorCodeResourceNotFound},
	}
	for _, tc := range cases {
		identity := mustIdentity(t, sellerCtx("", tc.holds...))
		apiErr := svc.checkOwnerPermission(identity, tc.owners, "Rate not found.")
		if tc.want == "" {
			assert.Nil(t, apiErr, tc.name)
			continue
		}
		if assert.NotNil(t, apiErr, tc.name) {
			assert.Equal(t, tc.want, apiErr.Code, tc.name)
		}
	}
}
