package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	publishermock "github.com/open-mrp/api/services/core-service/internal/domain/mock/publisher"
	"github.com/open-mrp/api/shared/appctx"
	"go.uber.org/mock/gomock"
)

// A number that is only whitespace is empty once trimmed, which no record can carry: it is refused
// on the field rather than answered "not a duplicate". The repos are mocks with no expectations, so
// any lookup fails the test.
func TestCheckDuplicate_refusesAWhitespaceOnlyNumber(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repos := factorymock.NewMockRepoFactory(ctrl)
	svc := NewUtilsSvc(&UtilsSvcConfig{
		Repos:                 repos,
		MediatorFactory:       factorymock.NewMockMediatorFactory(ctrl),
		TxManager:             &stubTxManager{factory: repos},
		NotificationPublisher: publishermock.NewMockNotificationPublisher(ctrl),
	})
	accountID := "acct_check_duplicate"
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_check_duplicate",
			AccountID:    &accountID,
			Permissions:  map[string]bool{"sales_orders:read": true},
		},
	})

	for _, number := range []string{" ", "   ", "\t\n"} {
		_, apiErr := svc.CheckDuplicate(ctx, domain.CheckDuplicateParams{Type: domain.DuplicateCheckTypeOrderNumber, RecordNumber: number})
		if apiErr == nil || apiErr.Param != "record_number" {
			t.Fatalf("%q: got %v, want an error naming record_number", number, apiErr)
		}
	}
}
