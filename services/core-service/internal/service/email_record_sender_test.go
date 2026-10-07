package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	publishermock "github.com/open-mrp/api/services/core-service/internal/domain/mock/publisher"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

// A manual send is attributed to whoever sent it, as a statement is; only the platform's own sends go unattributed.
func TestEmailRecord_InvoiceIsAttributedToItsSender(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	accounts := repositorymock.NewMockAccountRepo(ctrl)
	invoices := repositorymock.NewMockInvoiceRepo(ctrl)
	orders := repositorymock.NewMockSalesOrderRepo(ctrl)
	customers := repositorymock.NewMockCustomerRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewAccountRepo().Return(accounts).AnyTimes()
	repos.EXPECT().NewInvoiceRepo().Return(invoices).AnyTimes()
	repos.EXPECT().NewSalesOrderRepo().Return(orders).AnyTimes()
	repos.EXPECT().NewCustomerRepo().Return(customers).AnyTimes()
	idempotency := mediatormock.NewMockIdempotencyMed(ctrl)
	meds := factorymock.NewMockMediatorFactory(ctrl)
	meds.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: idempotency}).AnyTimes()
	publisher := publishermock.NewMockNotificationPublisher(ctrl)
	svc := NewUtilsSvc(&UtilsSvcConfig{
		Repos:                 repos,
		MediatorFactory:       meds,
		TxManager:             &stubTxManager{factory: repos},
		NotificationPublisher: publisher,
	})

	invoice, lines, _ := invoiceFixture()
	invoice.ID = "iv_1"
	invoice.OrderID = ""
	for _, l := range lines {
		l.PricingQuantityRatioNumerator, l.PricingQuantityRatioDenominator = "1", "1"
		l.PricingPriceRatioNumerator, l.PricingPriceRatioDenominator = "1", "1"
	}
	accounts.EXPECT().GetByID(gomock.Any(), invoiceSellerID).Return(&domain.Account{ID: invoiceSellerID, Name: "Seller Co"}, nil).AnyTimes()
	invoices.EXPECT().Get(gomock.Any(), domain.GetInvoiceParams{AccountID: invoiceSellerID, InvoiceID: "iv_1"}).Return(invoice, nil).AnyTimes()
	invoices.EXPECT().GetLines(gomock.Any(), "iv_1").Return(lines, nil)
	invoices.EXPECT().GetEmailRecipients(gomock.Any(), invoiceSellerID, "iv_1").Return([]string{"ap@example.com"}, nil).AnyTimes()
	orders.EXPECT().GetAccountOriginAddress(gomock.Any(), invoiceSellerID).Return(nil, nil)
	customers.EXPECT().Get(gomock.Any(), invoiceSellerID, gomock.Any(), gomock.Any()).Return(nil, nil)
	invoices.EXPECT().MarkEmailSent(gomock.Any(), invoiceSellerID, "iv_1").Return(nil)
	idempotency.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_1", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil)
	idempotency.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_1", gomock.Any()).Return(nil)

	var sent messaging.EmailSendData
	publisher.EXPECT().PublishSendEmail(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, data messaging.EmailSendData) *apierror.APIError {
			sent = data
			return nil
		})

	ctx := invoiceActorCtx(types.IdentityRelationTypeInternal, invoiceSellerID, invoiceSellerID, "invoices:read")
	require.Nil(t, svc.EmailRecord(ctx, domain.EmailRecordParams{Type: domain.EmailRecordTypeInvoice, ID: "iv_1"}))

	assert.Equal(t, []string{"ap@example.com"}, sent.To)
	if assert.NotNil(t, sent.SentByID) {
		assert.Equal(t, "us_actor", *sent.SentByID)
	}
}
