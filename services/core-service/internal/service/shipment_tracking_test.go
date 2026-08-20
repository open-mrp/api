package service

import (
	"context"
	"strings"
	"testing"

	"github.com/augno/api/services/core-service/internal/domain"
	clientmock "github.com/augno/api/services/core-service/internal/domain/mock/client"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSandboxTrackingNumber(t *testing.T) {
	// Deterministic per shipment so a retried ship reuses the same value.
	a := sandboxTrackingNumber("sh_01k0a87w33emw8pmkz1mf86cg1")
	b := sandboxTrackingNumber("sh_01k0a87w33emw8pmkz1mf86cg1")
	assert.Equal(t, a, b, "same shipment id must yield the same tracking number")
	assert.True(t, strings.HasPrefix(a, "SANDBOX-"), "sandbox tracking must carry the SANDBOX- prefix")
	assert.NotEqual(t, a, sandboxTrackingNumber("sh_different0000"), "different shipments differ")

	// A short id is handled without panicking.
	assert.True(t, strings.HasPrefix(sandboxTrackingNumber("sh_1"), "SANDBOX-"))
}

func TestResolveShipmentTracking_Sandbox(t *testing.T) {
	existing := "EXISTING-TRACK-123"
	shipment := &domain.Shipment{ID: "sh_track_test_0001", AccountID: "ac_test", MasterTrackingNumber: &existing}

	ctrl := gomock.NewController(t)
	accountRepo := repositorymock.NewMockAccountRepo(ctrl)
	accountRepo.EXPECT().GetAccountContext(gomock.Any(), shipment.AccountID).
		Return(&domain.AccountContext{AccountID: shipment.AccountID, IsSandbox: true}, nil)
	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	repoFactory.EXPECT().NewAccountRepo().Return(accountRepo).AnyTimes()

	// The factory has no expectations: a sandbox ship must never reach Shippo.
	shippoFactory := clientmock.NewMockShippoClientFactory(ctrl)

	svc := &shipmentSvcImpl{repos: repoFactory, shippoFactory: shippoFactory}

	got, apiErr := svc.resolveShipmentTracking(context.Background(), shipment, "idk_test")
	require.Nil(t, apiErr)
	require.NotNil(t, got)
	assert.Equal(t, sandboxTrackingNumber(shipment.ID), *got)
	assert.NotEqual(t, existing, *got)
}

func TestResolveShipmentTracking_NonSandboxBuysLabels(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	accountRepo := repositorymock.NewMockAccountRepo(ctrl)
	accountRepo.EXPECT().GetAccountContext(gomock.Any(), testLabelAccountID).
		Return(&domain.AccountContext{AccountID: testLabelAccountID, IsSandbox: false}, nil)
	h.repoFactory.EXPECT().NewAccountRepo().Return(accountRepo).AnyTimes()

	h.expectSuccessfulPurchase(&domain.LabelResult{
		MasterTrackingNumber: "1Z-MASTER",
		Packages: []domain.LabelPackage{
			{TrackingNumber: "1Z-CASE-1", LabelURL: "https://shippo/label1.png", ShippoTransactionID: "txn_1"},
		},
	})

	got, apiErr := h.svc.resolveShipmentTracking(context.Background(), h.shipment, "idk_test")
	require.Nil(t, apiErr)
	assert.Nil(t, got, "a live purchase persists the master tracking itself")
}
