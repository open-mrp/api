package apirequest

import (
	"math/big"

	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/validate"
)

func init() {
	validate.RegisterWrappedTypes(field.Optional[QuantityInput]{}, field.Clearable[QuantityInput]{})
}

// DayUnitID is the system unit every lead time is counted in.
const DayUnitID = "day"

// maxLeadTimeDays bounds a lead time at ten years.
const maxLeadTimeDays = 3650

// LeadTimeDays is a lead time sent as a Quantity, in whole days. param names the field it was sent as.
func LeadTimeDays(in QuantityInput, param string) (int32, *apierror.APIError) {
	if in.UnitID != DayUnitID {
		return 0, apierror.NewInvalidFormatError(param+".unit_id", "A lead time is counted in days: use the `day` unit.")
	}
	v, ok := new(big.Rat).SetString(in.Value)
	if !ok || !v.IsInt() || v.Sign() < 0 || v.Num().Int64() > maxLeadTimeDays {
		return 0, apierror.NewInvalidFormatError(param+".value", "A lead time is a whole number of days from 0 to 3650.")
	}
	return int32(v.Num().Int64()), nil
}

// OptionalLeadTimeDays is an optional lead time in days, or nil when it was left out.
func OptionalLeadTimeDays(f field.Optional[QuantityInput], param string) (*int32, *apierror.APIError) {
	in, ok := f.Value()
	if !ok {
		return nil, nil
	}
	days, apiErr := LeadTimeDays(in, param)
	if apiErr != nil {
		return nil, apiErr
	}
	return &days, nil
}

// LeadTimeDaysPatch is a clearable lead time as the days patch core-service takes: nil when left out.
func LeadTimeDaysPatch(f field.Clearable[QuantityInput], param string) (*pb.Int32Patch, *apierror.APIError) {
	switch {
	case f.IsUnset():
		return nil, nil
	case f.IsClear():
		return &pb.Int32Patch{Clear: true}, nil
	}
	in, _ := f.Value()
	days, apiErr := LeadTimeDays(in, param)
	if apiErr != nil {
		return nil, apiErr
	}
	return &pb.Int32Patch{Value: &days}, nil
}
