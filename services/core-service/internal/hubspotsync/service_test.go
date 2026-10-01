package hubspotsync

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	clientmock "github.com/open-mrp/api/services/core-service/internal/domain/mock/client"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
)

// TestLinkOrderContact pins that order sync only links contacts already in HubSpot. Creating contacts from orders polluted curated portals, so a regression back to UpsertContactByEmail must fail here (the strict mock rejects any unexpected call).
func TestLinkOrderContact(t *testing.T) {
	const (
		accountID = "ac_seller"
		companyID = "co_1"
		contactID = "ct_1"
	)
	billTo := "ap@clinic.com"
	customer := &domain.Customer{ID: "ac_buyer", Name: "Clinic", Email: strptr("owner@clinic.com")}

	tests := []struct {
		name   string
		order  *domain.SalesOrder
		setup  func(client *clientmock.MockHubspotClient, syncRepo *repositorymock.MockHubspotSyncRepo)
		wantID string
	}{
		{
			name:  "existing contact is promoted, associated, and mapped",
			order: &domain.SalesOrder{BillToEmail: &billTo},
			setup: func(client *clientmock.MockHubspotClient, syncRepo *repositorymock.MockHubspotSyncRepo) {
				client.EXPECT().SearchContactByEmail(gomock.Any(), billTo).Return(&domain.HubspotContact{ID: contactID, Email: billTo}, nil)
				client.EXPECT().UpdateContact(gomock.Any(), contactID, domain.HubspotContact{Lifecycle: lifecycleCustomer}).Return(nil)
				client.EXPECT().Associate(gomock.Any(), objectTypeContacts, contactID, objectTypeCompanies, companyID).Return(nil)
				syncRepo.EXPECT().UpsertRecord(gomock.Any(), domain.UpsertHubspotSyncRecordParams{
					AccountID:   accountID,
					AugnoType:   openMRPTypeContact,
					AugnoID:     customer.ID,
					HubspotType: objectTypeContacts,
					HubspotID:   contactID,
				}).Return(nil)
			},
			wantID: contactID,
		},
		{
			name:  "missing contact is skipped, not created",
			order: &domain.SalesOrder{BillToEmail: &billTo},
			setup: func(client *clientmock.MockHubspotClient, _ *repositorymock.MockHubspotSyncRepo) {
				client.EXPECT().SearchContactByEmail(gomock.Any(), billTo).Return(nil, nil)
			},
			wantID: "",
		},
		{
			name:  "falls back to the customer email when the order has no bill-to email",
			order: &domain.SalesOrder{},
			setup: func(client *clientmock.MockHubspotClient, _ *repositorymock.MockHubspotSyncRepo) {
				client.EXPECT().SearchContactByEmail(gomock.Any(), "owner@clinic.com").Return(nil, nil)
			},
			wantID: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			client := clientmock.NewMockHubspotClient(ctrl)
			syncRepo := repositorymock.NewMockHubspotSyncRepo(ctrl)
			repos := factorymock.NewMockRepoFactory(ctrl)
			repos.EXPECT().NewHubspotSyncRepo().Return(syncRepo).AnyTimes()
			tt.setup(client, syncRepo)

			s := &service{repos: repos}
			got, apiErr := s.linkOrderContact(context.Background(), client, accountID, tt.order, customer, companyID, true)
			if apiErr != nil {
				t.Fatalf("linkOrderContact() error = %v", apiErr)
			}
			if got != tt.wantID {
				t.Errorf("linkOrderContact() = %q, want %q", got, tt.wantID)
			}
		})
	}
}

func TestLinkOrderContact_NoEmailMakesNoCalls(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := clientmock.NewMockHubspotClient(ctrl)
	s := &service{repos: factorymock.NewMockRepoFactory(ctrl)}

	got, apiErr := s.linkOrderContact(context.Background(), client, "ac_seller", &domain.SalesOrder{}, &domain.Customer{ID: "ac_buyer"}, "co_1", true)
	if apiErr != nil || got != "" {
		t.Fatalf("linkOrderContact() = (%q, %v), want (\"\", nil)", got, apiErr)
	}
}
