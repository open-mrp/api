package fieldpb

import (
	"fmt"
	"time"

	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/timeutil"
)

// DateToProto is a business day as the RPC carries it.
func DateToProto(d timeutil.Date) *pb.Date {
	return &pb.Date{Year: int32(d.Year), Month: int32(d.Month), Day: int32(d.Day)} // #nosec G115 - calendar fields
}

// DatePtrToProto is DateToProto for an optional date; nil stays nil.
func DatePtrToProto(d *timeutil.Date) *pb.Date {
	if d == nil {
		return nil
	}
	return DateToProto(*d)
}

// DateFromProto is the RPC's date, or nil when it was not sent. A day that does not exist is an error, never
// rolled over into the next month.
func DateFromProto(p *pb.Date) (*timeutil.Date, error) {
	if p == nil {
		return nil, nil
	}
	d := timeutil.NewDate(int(p.Year), time.Month(p.Month), int(p.Day))
	if d.Year != int(p.Year) || int32(d.Month) != p.Month || int32(d.Day) != p.Day { // #nosec G115 - calendar fields
		return nil, fmt.Errorf("invalid date %04d-%02d-%02d", p.Year, p.Month, p.Day)
	}
	return &d, nil
}

// DateClearableToProto is a clearable date field as the RPC carries it; nil when unset.
func DateClearableToProto(f field.Clearable[timeutil.Date]) *pb.DatePatch {
	if f.IsUnset() {
		return nil
	}
	if f.IsClear() {
		return &pb.DatePatch{Clear: true}
	}
	d, _ := f.Value()
	return &pb.DatePatch{Value: DateToProto(d)}
}

// DateClearableFromProto is the RPC's date patch as a clearable field. A patch with neither clear nor a value
// clears, matching the other patch types.
func DateClearableFromProto(p *pb.DatePatch) (field.Clearable[timeutil.Date], error) {
	if p == nil {
		return field.Unset[timeutil.Date](), nil
	}
	if p.Clear || p.Value == nil {
		return field.Clear[timeutil.Date](), nil
	}
	d, err := DateFromProto(p.Value)
	if err != nil {
		return field.Unset[timeutil.Date](), err
	}
	return field.Set(*d), nil
}
