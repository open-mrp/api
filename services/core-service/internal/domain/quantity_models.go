package domain

import "github.com/augno/api/shared/constants"

type UpdateQuantityParams struct {
	QuantityID string
	Value      *string
	UnitID     *string
	ObjectID   *string
	ObjectType *constants.ObjectType
}
