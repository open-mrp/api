package resourceloaders

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var departmentLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.department")

func LoadDepartments(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, departmentLoaderTracer, "loader.departments.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetDepartmentsByIDsResponse, error) {
			return coreClient.BatchGetDepartmentsByIDs(ctx, &pb.BatchGetDepartmentsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Departments))
	for _, d := range resp.Departments {
		out[d.Id] = departmentFromProto(d)

		var locationID string
		if d.LocationId != nil {
			locationID = *d.LocationId
		}
		meta.Set(constants.ObjectTypeDepartment, d.Id, "location_id", locationID)

		stations := make([]apiresource.ScanningStation, len(d.ScanningStations))
		for i, s := range d.ScanningStations {
			stations[i] = *ScanningStationFromLightProto(s)
		}
		meta.Set(constants.ObjectTypeDepartment, d.Id, "scanning_stations", stations)

		machines := make([]apiresource.Machine, len(d.Machines))
		for i, m := range d.Machines {
			machines[i] = apiresource.Machine{
				ID:           m.Id,
				Object:       constants.ObjectTypeMachine,
				Name:         m.Name,
				SerialNumber: m.SerialNumber,
				CreatedAt:    grpcutil.TimestampToTime(m.CreatedAt),
				UpdatedAt:    grpcutil.TimestampToTime(m.UpdatedAt),
			}
		}
		meta.Set(constants.ObjectTypeDepartment, d.Id, "machines", machines)
	}
	return out, nil
}

func departmentFromProto(d *pb.DepartmentInfo) *apiresource.Department {
	return &apiresource.Department{
		ID:        d.Id,
		Object:    constants.ObjectTypeDepartment,
		Name:      d.Name,
		Notes:     d.Notes,
		LaborRate: departmentRateFromProto(d.LaborRate),
		CreatedAt: grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(d.UpdatedAt),
	}
}

// embeddedDepartmentRateTimestamp marks a rate rendered inline by its parent, mirroring the production-step convention.
var embeddedDepartmentRateTimestamp = time.Unix(1, 0).UTC()

func departmentRateFromProto(r *pb.DepartmentRateInfo) *apiresource.Rate {
	if r == nil {
		return nil
	}
	return &apiresource.Rate{
		ID:     r.Id,
		Object: constants.ObjectTypeRate,
		Value:  r.Value,
		// numerator_unit / denominator_unit left nil: expandable, loaded with real data via ?include=; never fabricated. display_value carries the rate.
		DisplayValue: apiresource.FormatRateDisplayValue(r.Value, r.NumeratorUnitAbbreviation, r.NumeratorUnitType, r.DenominatorUnitAbbreviation),
		CreatedAt:    embeddedDepartmentRateTimestamp,
		UpdatedAt:    embeddedDepartmentRateTimestamp,
	}
}
