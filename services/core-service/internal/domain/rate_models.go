package domain

import "github.com/open-mrp/api/shared/constants"

type UpdateRateParams struct {
	RateID            string
	Value             *string
	NumeratorUnitID   *string
	DenominatorUnitID *string
	ObjectID          *string
	ObjectType        *constants.ObjectType
}
