package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var unitLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.unit")

// LoadUnits fetches units by ID via BatchGetUnitsByIDs.
// Stashes owner_account_id in LoadMeta so the SubField closures can build the Owner shell and (on owner.account) write the loaded Account.
func LoadUnits(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, unitLoaderTracer, "loader.units.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetUnitsByIDsResponse, error) {
			return coreClient.BatchGetUnitsByIDs(ctx, &pb.BatchGetUnitsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Units))
	for _, u := range resp.Units {
		out[u.Id] = unitFromProto(u)
		var accountID string
		if u.AccountId != nil {
			accountID = *u.AccountId
		}
		meta.Set(constants.ObjectTypeUnit, u.Id, "owner_account_id", accountID)
	}
	return out, nil
}

func unitFromProto(u *pb.UnitInfo) *apiresource.Unit {
	return &apiresource.Unit{
		ID:                u.Id,
		Object:            constants.ObjectTypeUnit,
		Name:              u.Name,
		Abbreviation:      u.Abbreviation,
		Type:              constants.UnitType(u.Type),
		RatioNumerator:    db.TrimDecimal(u.RatioNumerator),
		RatioDenominator:  db.TrimDecimal(u.RatioDenominator),
		OffsetNumerator:   db.TrimDecimal(u.OffsetNumerator),
		OffsetDenominator: db.TrimDecimal(u.OffsetDenominator),
		IsBaseUnit:        u.IsBaseUnit,
		CreatedAt:         grpcutil.TimestampToTime(u.CreatedAt),
		UpdatedAt:         grpcutil.TimestampToTime(u.UpdatedAt),
	}
}

// LoadUnitsByID resolves units into their API resources, keyed by id. Ids are
// deduplicated and blanks dropped, so callers can pass whatever their proto handed
// them.
//
// Use it wherever a response carries unit ids but the unit is not behind an
// include — a computed figure has to arrive with the unit it is counted in, or the
// client has nothing to render it against.
func LoadUnitsByID(ctx context.Context, ids ...string) (map[string]*apiresource.Unit, *apierror.APIError) {
	unique := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return map[string]*apiresource.Unit{}, nil
	}

	loaded, apiErr := LoadUnits(ctx, unique)
	if apiErr != nil {
		return nil, apiErr
	}

	out := make(map[string]*apiresource.Unit, len(loaded))
	for id, v := range loaded {
		if u, ok := v.(*apiresource.Unit); ok {
			out[id] = u
		}
	}
	return out, nil
}
