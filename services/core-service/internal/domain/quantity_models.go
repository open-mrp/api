package domain

import "github.com/open-mrp/api/shared/constants"

type UpdateQuantityParams struct {
	QuantityID string
	Value      *string
	UnitID     *string
	ObjectID   *string
	ObjectType *constants.ObjectType
}
