package resourceloaders

import (
	"context"
	"strconv"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var unitGroupLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.unit_group")

// LoadUnitGroups fetches unit groups by ID via BatchGetUnitGroupsByIDs.
// Stashes owner_account_id, base_unit_id, and pre-built associated units
// data in LoadMeta for SubField closures.
func LoadUnitGroups(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, unitGroupLoaderTracer, "loader.unit_groups.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetUnitGroupsByIDsResponse, error) {
			return coreClient.BatchGetUnitGroupsByIDs(ctx, &pb.BatchGetUnitGroupsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.UnitGroups))
	for _, ug := range resp.UnitGroups {
		out[ug.Id] = unitGroupFromProto(ug)

		var accountID string
		if ug.AccountId != nil {
			accountID = *ug.AccountId
		}
		meta.Set(constants.ObjectTypeUnitGroup, ug.Id, "owner_account_id", accountID)

		if ug.BaseUnit != nil {
			meta.Set(constants.ObjectTypeUnitGroup, ug.Id, "base_unit_id", ug.BaseUnit.Id)
		}

		// Pre-build the associated units WITH inline Unit data for the
		// no-fetch associated_units SubField on UnitGroup. This preserves
		// backward compat: ?include[]=associated_units returns full
		// UnitGroupUnits with Unit details populated.
		if len(ug.UnitConversions) > 0 {
			items := make([]apiresource.UnitGroupUnit, 0, len(ug.UnitConversions))
			for _, c := range ug.UnitConversions {
				items = append(items, unitGroupUnitFromProtoWithUnit(c))
			}
			meta.Set(constants.ObjectTypeUnitGroup, ug.Id, "associated_units_data", items)
		}
	}
	return out, nil
}

func unitGroupFromProto(ug *pb.UnitGroupInfo) *apiresource.UnitGroup {
	var notes *string
	if ug.Notes != nil {
		notes = ug.Notes
	}
	return &apiresource.UnitGroup{
		ID:        ug.Id,
		Object:    constants.ObjectTypeUnitGroup,
		Name:      ug.Name,
		Notes:     notes,
		Type:      constants.UnitType(ug.Type),
		CreatedAt: grpcutil.TimestampToTime(ug.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(ug.UpdatedAt),
	}
}

var unitGroupUnitLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.unit_group_unit")

// LoadUnitGroupUnits fetches unit group units by ID via BatchGetUnitGroupUnitsByIDs.
// Returns flat UnitGroupUnits without Unit populated — the unit SubField on
// UnitGroupUnit handles loading Unit when ?include=unit is requested.
func LoadUnitGroupUnits(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, unitGroupUnitLoaderTracer, "loader.unit_group_units.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetUnitGroupUnitsByIDsResponse, error) {
			return coreClient.BatchGetUnitGroupUnitsByIDs(ctx, &pb.BatchGetUnitGroupUnitsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.UnitGroupUnits))
	for _, u := range resp.UnitGroupUnits {
		out[u.Id] = unitGroupUnitFromProtoFlat(u)
		meta.Set(constants.ObjectTypeUnitGroupUnit, u.Id, "unit_id", u.UnitId)
	}
	return out, nil
}

// unitGroupUnitFromProtoFlat maps a proto UnitGroupUnitInfo to an apiresource
// WITHOUT populating Unit. Used by standalone UnitGroupUnit endpoints.
func unitGroupUnitFromProtoFlat(u *pb.UnitGroupUnitInfo) *apiresource.UnitGroupUnit {
	discountPercentage, _ := strconv.ParseFloat(u.DiscountPercentage, 64)
	discountFixed, _ := strconv.ParseFloat(u.DiscountFixed, 64)

	visibility := constants.CustomerPortalVisibilityHidden
	if u.IsVisible {
		visibility = constants.CustomerPortalVisibilityVisible
	}

	return &apiresource.UnitGroupUnit{
		ID:                       u.Id,
		Object:                   constants.ObjectTypeUnitGroupUnit,
		DiscountPercentage:       discountPercentage,
		DiscountFixed:            discountFixed,
		CustomerPortalVisibility: visibility,
		CreatedAt:                grpcutil.TimestampToTime(u.CreatedAt),
		UpdatedAt:                grpcutil.TimestampToTime(u.UpdatedAt),
	}
}

// unitGroupUnitFromProtoWithUnit maps a proto UnitGroupUnitInfo WITH inline
// Unit details. Used by the UnitGroup's associated_units SubField to preserve
// backward compat where unit data is always returned.
func unitGroupUnitFromProtoWithUnit(u *pb.UnitGroupUnitInfo) apiresource.UnitGroupUnit {
	result := *unitGroupUnitFromProtoFlat(u)

	if u.Unit != nil {
		result.Unit = &apiresource.Unit{
			ID:                u.Unit.Id,
			Object:            constants.ObjectTypeUnit,
			Name:              u.Unit.Name,
			Abbreviation:      u.Unit.Abbreviation,
			Type:              constants.UnitType(u.Unit.Type),
			RatioNumerator:    db.TrimDecimal(u.Unit.RatioNumerator),
			RatioDenominator:  db.TrimDecimal(u.Unit.RatioDenominator),
			OffsetNumerator:   db.TrimDecimal(u.Unit.OffsetNumerator),
			OffsetDenominator: db.TrimDecimal(u.Unit.OffsetDenominator),
			IsBaseUnit:        u.Unit.IsBaseUnit,
			Owner:             ownerShellFromAccountID(u.Unit.AccountId),
			CreatedAt:         grpcutil.TimestampToTime(u.Unit.CreatedAt),
			UpdatedAt:         grpcutil.TimestampToTime(u.Unit.UpdatedAt),
		}
	}

	return result
}
