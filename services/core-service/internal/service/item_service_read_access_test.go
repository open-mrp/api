package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	itemReadSellerID   = "ac_seller"
	itemReadCustomerID = "ac_customer"
)

// itemReadCtx is a user of actorAccount reaching target as relation, holding exactly perms.
func itemReadCtx(relation types.IdentityRelationType, actorAccount, target string, perms ...string) context.Context {
	granted := make(map[string]bool, len(perms))
	for _, p := range perms {
		granted[p] = true
	}
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: target},
		Actor: &types.IdentityActor{
			RelationType: relation,
			ID:           "us_actor",
			AccountID:    &actorAccount,
			Permissions:  granted,
		},
	})
}

func newItemReadSvc(t *testing.T) (domain.ItemSvc, *repositorymock.MockItemRepo, *mediatormock.MockReadAccessMed) {
	t.Helper()
	ctrl := gomock.NewController(t)
	itemRepo := repositorymock.NewMockItemRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewItemRepo().Return(itemRepo).AnyTimes()
	readAccess := mediatormock.NewMockReadAccessMed(ctrl)
	meds := factorymock.NewMockMediatorFactory(ctrl)
	meds.EXPECT().Build(gomock.Any()).Return(domain.Mediators{ReadAccess: readAccess}).AnyTimes()
	return NewItemSvc(&ItemSvcConfig{Repos: repos, MediatorFactory: meds, TxManager: &stubTxManager{factory: repos}}), itemRepo, readAccess
}

// An item record, cost included, is the seller's own: a portal user is refused it whatever permissions it holds, before anything is read.
func TestItemSvc_GetItemRefusesAnyoneButTheSellersOwnUsers(t *testing.T) {
	t.Parallel()

	actors := map[string]context.Context{
		"customer portal user":    itemReadCtx(types.IdentityRelationTypeCustomer, itemReadCustomerID, itemReadSellerID, "items:read"),
		"supplier portal user":    itemReadCtx(types.IdentityRelationTypeSupplier, itemReadCustomerID, itemReadSellerID, "items:read"),
		"seller staff at a buyer": itemReadCtx(types.IdentityRelationTypeInternal, itemReadSellerID, itemReadCustomerID, "items:read", "customers:read"),
		"seller staff without it": itemReadCtx(types.IdentityRelationTypeInternal, itemReadSellerID, itemReadSellerID, "products:read"),
	}
	for name, ctx := range actors {
		svc, _, _ := newItemReadSvc(t)
		_, apiErr := svc.GetItem(ctx, "itm_1", nil)
		if assert.NotNil(t, apiErr, name) {
			assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code, name)
		}
	}
}

func TestItemSvc_GetItemReadsTheSellersItem(t *testing.T) {
	t.Parallel()

	includes := []string{"unit_value", "unit_cost", "burn_rate", "attributes"}
	svc, itemRepo, _ := newItemReadSvc(t)
	itemRepo.EXPECT().Get(gomock.Any(), domain.GetItemParams{AccountID: itemReadSellerID, ItemID: "itm_1", Includes: includes}).
		Return(&domain.Item{ID: "itm_1"}, nil)

	item, apiErr := svc.GetItem(itemReadCtx(types.IdentityRelationTypeInternal, itemReadSellerID, itemReadSellerID, "items:read"), "itm_1", includes)
	require.Nil(t, apiErr)
	assert.Equal(t, "itm_1", item.ID)
}

func TestItemSvc_GetItemNotFoundNamesTheItem(t *testing.T) {
	t.Parallel()

	svc, itemRepo, _ := newItemReadSvc(t)
	itemRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, apierror.NewResourceNotFoundError("Resource not found."))

	_, apiErr := svc.GetItem(itemReadCtx(types.IdentityRelationTypeInternal, itemReadSellerID, itemReadSellerID, "items:read"), "itm_missing", nil)
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code)
	assert.Equal(t, "Item not found.", apiErr.PublicMessage)
}

// The include loader's batch read is what lets a portal user see the items on its own orders and invoices.
func TestItemSvc_BatchGetItemsAdmitsACounterpartyResolvingIncludes(t *testing.T) {
	t.Parallel()

	svc, itemRepo, readAccess := newItemReadSvc(t)
	readAccess.EXPECT().CheckCounterpartyReadAccess(gomock.Any(), itemReadCustomerID, itemReadSellerID).Return(nil)
	itemRepo.EXPECT().GetByIDs(gomock.Any(), itemReadSellerID, []string{"itm_1"}).Return([]*domain.Item{{ID: "itm_1"}}, nil)

	items, apiErr := svc.BatchGetItemsByIDs(itemReadCtx(types.IdentityRelationTypeCustomer, itemReadCustomerID, itemReadSellerID), []string{"itm_1"})
	require.Nil(t, apiErr)
	require.Len(t, items, 1)
}
