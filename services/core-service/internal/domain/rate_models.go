package domain

import "github.com/augno/api/shared/constants"

type UpdateRateParams struct {
	RateID            string
	Value             *string
	NumeratorUnitID   *string
	DenominatorUnitID *string
	ObjectID          *string
	ObjectType        *constants.ObjectType
}
