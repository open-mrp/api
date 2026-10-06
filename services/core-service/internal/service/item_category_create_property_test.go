package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type categoryPropertyHarness struct {
	svc        domain.ItemCategorySvc
	categories *repositorymock.MockItemCategoryRepo
	properties *repositorymock.MockPropertyRepo
	idem       *mediatormock.MockIdempotencyMed
	outbox     *capturingOutboxRepo
}

func newCategoryPropertyHarness(t *testing.T) *categoryPropertyHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	h := &categoryPropertyHarness{
		categories: repositorymock.NewMockItemCategoryRepo(ctrl),
		properties: repositorymock.NewMockPropertyRepo(ctrl),
		idem:       mediatormock.NewMockIdempotencyMed(ctrl),
		outbox:     &capturingOutboxRepo{},
	}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewItemCategoryRepo().Return(h.categories).AnyTimes()
	repos.EXPECT().NewPropertyRepo().Return(h.properties).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(h.outbox).AnyTimes()
	meds := factorymock.NewMockMediatorFactory(ctrl)
	meds.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: h.idem}).AnyTimes()
	h.svc = NewItemCategorySvc(&ItemCategorySvcConfig{
		Repos:           repos,
		MediatorFactory: meds,
		JobSvcFactory:   NewJobSvcFactory(),
		TxManager:       &stubTxManager{factory: repos},
	})
	return h
}

func (h *categoryPropertyHarness) startsFresh() {
	h.idem.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).Return(&domain.IdempotencyKey{TypeID: "idk_prop", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil)
}

func (h *categoryPropertyHarness) audited(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, msg := range h.outbox.messages {
		var payload struct {
			Action       string `json:"action"`
			ResourceType string `json:"resource_type"`
			ResourceID   string `json:"resource_id"`
		}
		require.NoError(t, json.Unmarshal(msg.Payload.Data, &payload))
		out = append(out, payload.Action+" "+payload.ResourceType+" "+payload.ResourceID)
	}
	return out
}

// categoryEditorCtx is a non-admin user holding exactly perms.
func categoryEditorCtx(perms ...string) context.Context {
	account := "ac_cat"
	roleType := string(constants.RoleTypeCustom)
	held := map[string]bool{}
	for _, p := range perms {
		held[p] = true
	}
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: account},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_cat",
			AccountID:    &account,
			RoleType:     &roleType,
			Permissions:  held,
		},
	})
}

func (h *categoryPropertyHarness) categoryWith(properties ...*domain.ItemCategoryProperty) {
	h.categories.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.ItemCategoryFull{ID: "ic_socks", UnitGroupID: "ug_pairs"}, nil)
	h.categories.EXPECT().GetProperties(gomock.Any(), "ic_socks").Return(properties, nil)
	h.categories.EXPECT().GetUnitGroup(gomock.Any(), "ug_pairs", nil).Return(&domain.ItemCategoryUnitGroup{}, nil)
}

func TestCreateItemCategoryProperty_CreatesAndAttachesUnderTheCategoryPermission(t *testing.T) {
	t.Parallel()
	h := newCategoryPropertyHarness(t)
	h.startsFresh()
	created := &domain.Property{ID: "pp_size", Name: "Size", AccountID: "ac_cat"}

	gomock.InOrder(
		h.categories.EXPECT().IsInAccount(gomock.Any(), "ac_cat", "ic_socks").Return(true, nil),
		h.categories.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.ItemCategoryFull{ID: "ic_socks", UnitGroupID: "ug_pairs"}, nil),
		h.categories.EXPECT().GetProperties(gomock.Any(), "ic_socks").Return(nil, nil),
		h.categories.EXPECT().GetUnitGroup(gomock.Any(), "ug_pairs", nil).Return(&domain.ItemCategoryUnitGroup{}, nil),
		h.properties.EXPECT().ExistsByName(gomock.Any(), "ac_cat", "Size", nil).Return(false, nil),
		h.categories.EXPECT().PropertyExistsByNameInCategory(gomock.Any(), "ac_cat", "ic_socks", "Size", nil).Return(false, nil),
		h.properties.EXPECT().Create(gomock.Any(), gomock.Any(), domain.CreatePropertyParams{AccountID: "ac_cat", Name: "Size"}).Return(created, nil),
		h.categories.EXPECT().AddProperty(gomock.Any(), domain.AddItemCategoryPropertyParams{AccountID: "ac_cat", ItemCategoryID: "ic_socks", PropertyID: "pp_size"}).Return(nil),
		h.categories.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.ItemCategoryFull{ID: "ic_socks", UnitGroupID: "ug_pairs"}, nil),
		h.categories.EXPECT().GetProperties(gomock.Any(), "ic_socks").Return([]*domain.ItemCategoryProperty{{ID: "pp_size", Name: "Size"}}, nil),
		h.categories.EXPECT().GetUnitGroup(gomock.Any(), "ug_pairs", nil).Return(&domain.ItemCategoryUnitGroup{}, nil),
		h.idem.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_prop", created).Return(nil),
	)

	got, apiErr := h.svc.CreateItemCategoryProperty(categoryEditorCtx("item_categories:update"), domain.CreateItemCategoryPropertyParams{ItemCategoryID: "ic_socks", Name: "Size"})

	require.Nil(t, apiErr)
	assert.Equal(t, "pp_size", got.ID)
	assert.Equal(t, []string{"create property pp_size", "update item_category ic_socks"}, h.audited(t), "the events the separate create and attach published")
}

func TestCreateItemCategoryProperty_RefusesATakenNameWithoutWriting(t *testing.T) {
	t.Parallel()

	for name, inCategory := range map[string]bool{"taken in the account": false, "taken in the category": true} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := newCategoryPropertyHarness(t)
			h.startsFresh()
			h.categories.EXPECT().IsInAccount(gomock.Any(), "ac_cat", "ic_socks").Return(true, nil)
			h.categoryWith()
			h.properties.EXPECT().ExistsByName(gomock.Any(), "ac_cat", "Color", nil).Return(!inCategory, nil)
			if inCategory {
				h.categories.EXPECT().PropertyExistsByNameInCategory(gomock.Any(), "ac_cat", "ic_socks", "Color", nil).Return(true, nil)
			}
			h.idem.EXPECT().CacheErrorResponse(gomock.Any(), "idk_prop", gomock.Any()).DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError { return apiErr })

			_, apiErr := h.svc.CreateItemCategoryProperty(categoryEditorCtx("item_categories:update"), domain.CreateItemCategoryPropertyParams{ItemCategoryID: "ic_socks", Name: "Color"})

			require.NotNil(t, apiErr)
			assert.Equal(t, apierror.ErrorCodeResourceConflict, apiErr.Code)
			assert.Equal(t, "name", apiErr.Param)
			assert.Empty(t, h.audited(t))
		})
	}
}

func TestCreateItemCategoryProperty_RefusesBeforeTouchingTheCategory(t *testing.T) {
	t.Parallel()

	t.Run("without item_categories:update", func(t *testing.T) {
		t.Parallel()
		h := newCategoryPropertyHarness(t)
		_, apiErr := h.svc.CreateItemCategoryProperty(categoryEditorCtx("properties:create"), domain.CreateItemCategoryPropertyParams{ItemCategoryID: "ic_socks", Name: "Size"})
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code)
		assert.Contains(t, apiErr.PublicMessage, "item_categories:update")
	})

	t.Run("a default category", func(t *testing.T) {
		t.Parallel()
		h := newCategoryPropertyHarness(t)
		_, apiErr := h.svc.CreateItemCategoryProperty(categoryEditorCtx("item_categories:update"), domain.CreateItemCategoryPropertyParams{ItemCategoryID: domain.DefaultCategoryShipping, Name: "Size"})
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code)
	})

	t.Run("a category outside the account", func(t *testing.T) {
		t.Parallel()
		h := newCategoryPropertyHarness(t)
		h.categories.EXPECT().IsInAccount(gomock.Any(), "ac_cat", "ic_elsewhere").Return(false, nil)
		_, apiErr := h.svc.CreateItemCategoryProperty(categoryEditorCtx("item_categories:update"), domain.CreateItemCategoryPropertyParams{ItemCategoryID: "ic_elsewhere", Name: "Size"})
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code)
	})
}
