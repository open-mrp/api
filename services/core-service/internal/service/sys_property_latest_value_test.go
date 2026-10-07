package service

import (
	"context"
	"strconv"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"go.uber.org/mock/gomock"
)

const latestValueAccountID = "acct_latest_value"

func latestValueCtx() context.Context {
	accountID := latestValueAccountID
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_latest_value",
			AccountID:    &accountID,
			Permissions:  map[string]bool{"system_properties:update": true},
		},
	})
}

func newLatestValueSvc(t *testing.T) (domain.SysPropertySvc, *repositorymock.MockSysPropertyRepo) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockSysPropertyRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewSysPropertyRepo().Return(repo).AnyTimes()
	return NewSysPropertySvc(&SysPropertySvcConfig{
		Repos:           repos,
		MediatorFactory: factorymock.NewMockMediatorFactory(ctrl),
		TxManager:       &stubTxManager{factory: repos},
	}), repo
}

// takenNumbers answers TakenNumbers from a fixed set of numbers in use, counting the queries.
func takenNumbers(inUse map[int]bool, queries *int) func(context.Context, string, constants.SysPropertyTypeCode, []string) ([]string, *apierror.APIError) {
	return func(_ context.Context, _ string, _ constants.SysPropertyTypeCode, candidates []string) ([]string, *apierror.APIError) {
		*queries++
		var taken []string
		for _, c := range candidates {
			if n, _ := strconv.Atoi(c); inUse[n] {
				taken = append(taken, c)
			}
		}
		return taken, nil
	}
}

func supplierCounter(value int32) *domain.SysProperty {
	return &domain.SysProperty{ID: "sypp_supplier", TypeCode: constants.SysPropertyTypeCodeSupplierNumber, Value: value}
}

func TestGetLatestSysPropertyValue_skipsEveryNumberInUse(t *testing.T) {
	t.Parallel()
	svc, repo := newLatestValueSvc(t)
	var queries int
	repo.EXPECT().GetByTypeCode(gomock.Any(), latestValueAccountID, constants.SysPropertyTypeCodeSupplierNumber).Return(supplierCounter(100), nil)
	repo.EXPECT().TakenNumbers(gomock.Any(), latestValueAccountID, constants.SysPropertyTypeCodeSupplierNumber, gomock.Any()).
		DoAndReturn(takenNumbers(map[int]bool{100: true, 101: true, 102: true}, &queries)).AnyTimes()
	repo.EXPECT().UpdateValue(gomock.Any(), latestValueAccountID, "sypp_supplier", int32(103)).Return(supplierCounter(103), nil)

	got, apiErr := svc.GetLatestSysPropertyValue(latestValueCtx(), constants.SysPropertyTypeCodeSupplierNumber)
	if apiErr != nil {
		t.Fatal(apiErr)
	}
	if got != "103" {
		t.Fatalf("got %s, want 103: the first number after the taken run", got)
	}
	if queries != 1 {
		t.Fatalf("a short run of taken numbers took %d queries, want 1", queries)
	}
}

// A free counter is handed out as it is and not written, so opening a form takes no number.
func TestGetLatestSysPropertyValue_keepsAFreeNumber(t *testing.T) {
	t.Parallel()
	svc, repo := newLatestValueSvc(t)
	var queries int
	repo.EXPECT().GetByTypeCode(gomock.Any(), latestValueAccountID, constants.SysPropertyTypeCodeSupplierNumber).Return(supplierCounter(100), nil)
	repo.EXPECT().TakenNumbers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(takenNumbers(map[int]bool{99: true, 101: true}, &queries)).AnyTimes()

	got, apiErr := svc.GetLatestSysPropertyValue(latestValueCtx(), constants.SysPropertyTypeCodeSupplierNumber)
	if apiErr != nil {
		t.Fatal(apiErr)
	}
	if got != "100" {
		t.Fatalf("got %s, want 100", got)
	}
}

// A long run is searched in growing batches, so it costs a handful of queries, not one per number.
func TestGetLatestSysPropertyValue_longRunTakesFewQueries(t *testing.T) {
	t.Parallel()
	svc, repo := newLatestValueSvc(t)
	inUse := map[int]bool{}
	for n := 1; n <= 5000; n++ {
		inUse[n] = true
	}
	var queries int
	repo.EXPECT().GetByTypeCode(gomock.Any(), gomock.Any(), gomock.Any()).Return(supplierCounter(1), nil)
	repo.EXPECT().TakenNumbers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(takenNumbers(inUse, &queries)).AnyTimes()
	repo.EXPECT().UpdateValue(gomock.Any(), gomock.Any(), gomock.Any(), int32(5001)).Return(supplierCounter(5001), nil)

	got, apiErr := svc.GetLatestSysPropertyValue(latestValueCtx(), constants.SysPropertyTypeCodeSupplierNumber)
	if apiErr != nil {
		t.Fatal(apiErr)
	}
	if got != "5001" {
		t.Fatalf("got %s, want 5001", got)
	}
	if queries > 10 {
		t.Fatalf("5000 taken numbers took %d queries", queries)
	}
}

// The search is bounded: a counter far below the numbers already issued is refused with a message to
// move it, and the counter is left where it is.
func TestGetLatestSysPropertyValue_boundedSearch(t *testing.T) {
	t.Parallel()
	svc, repo := newLatestValueSvc(t)
	var queries int
	repo.EXPECT().GetByTypeCode(gomock.Any(), gomock.Any(), gomock.Any()).Return(supplierCounter(1), nil)
	repo.EXPECT().TakenNumbers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ constants.SysPropertyTypeCode, candidates []string) ([]string, *apierror.APIError) {
			queries++
			return candidates, nil
		}).AnyTimes()

	_, apiErr := svc.GetLatestSysPropertyValue(latestValueCtx(), constants.SysPropertyTypeCodeSupplierNumber)
	if apiErr == nil || apiErr.Code != apierror.ErrorCodeResourceConflict {
		t.Fatalf("got %v, want resource_conflict", apiErr)
	}
	if queries > 20 {
		t.Fatalf("the bounded search took %d queries", queries)
	}
}

// The search never steps past the largest value the counter can hold.
func TestGetLatestSysPropertyValue_stopsAtTheCounterMaximum(t *testing.T) {
	t.Parallel()
	svc, repo := newLatestValueSvc(t)
	repo.EXPECT().GetByTypeCode(gomock.Any(), gomock.Any(), gomock.Any()).Return(supplierCounter(2147483646), nil)
	last := []string{"2147483646", "2147483647"}
	repo.EXPECT().TakenNumbers(gomock.Any(), gomock.Any(), gomock.Any(), last).Return(last, nil)

	_, apiErr := svc.GetLatestSysPropertyValue(latestValueCtx(), constants.SysPropertyTypeCodeSupplierNumber)
	if apiErr == nil || apiErr.Code != apierror.ErrorCodeResourceConflict {
		t.Fatalf("got %v, want resource_conflict", apiErr)
	}
}

// A counter read for the first time starts at the first number no record already carries.
func TestGetLatestSysPropertyValue_createsTheCounterAtTheFirstFreeNumber(t *testing.T) {
	t.Parallel()
	svc, repo := newLatestValueSvc(t)
	var queries int
	repo.EXPECT().GetByTypeCode(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, apierror.NewResourceNotFoundError("System property not found."))
	repo.EXPECT().TakenNumbers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(takenNumbers(map[int]bool{1: true, 2: true}, &queries)).AnyTimes()
	repo.EXPECT().Create(gomock.Any(), gomock.Any(), latestValueAccountID, constants.SysPropertyTypeCodeSupplierNumber, int32(3)).Return(supplierCounter(3), nil)

	got, apiErr := svc.GetLatestSysPropertyValue(latestValueCtx(), constants.SysPropertyTypeCodeSupplierNumber)
	if apiErr != nil {
		t.Fatal(apiErr)
	}
	if got != "3" {
		t.Fatalf("got %s, want 3", got)
	}
}

func TestGetLatestSysPropertyValue_ssccCountIsReturnedAsIs(t *testing.T) {
	t.Parallel()
	svc, repo := newLatestValueSvc(t)
	repo.EXPECT().GetByTypeCode(gomock.Any(), gomock.Any(), constants.SysPropertyTypeCodeSsccCount).
		Return(&domain.SysProperty{ID: "sypp_sscc", TypeCode: constants.SysPropertyTypeCodeSsccCount, Value: 7}, nil)

	got, apiErr := svc.GetLatestSysPropertyValue(latestValueCtx(), constants.SysPropertyTypeCodeSsccCount)
	if apiErr != nil {
		t.Fatal(apiErr)
	}
	if got != "7" {
		t.Fatalf("got %s, want 7", got)
	}
}
