package service

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repomock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
)

// The seam between the purchase order document and the send command. The automatic issue path and
// the manual resend both go through this assembler, and it is where the supplier's copy regressed
// once before: the send was built from a letterhead-only parameter map, so the template rendered a
// header over an empty table and no PDF was attached at all.

func poEmailRepos(t *testing.T, recipients []string) domain.RepoFactory {
	t.Helper()
	ctrl := gomock.NewController(t)

	order, lines := poFixture()

	poRepo := repomock.NewMockPurchaseOrderRepo(ctrl)
	poRepo.EXPECT().GetSubmissionRecipients(gomock.Any(), gomock.Any()).Return(recipients, nil).AnyTimes()
	poRepo.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(order, nil).AnyTimes()
	poRepo.EXPECT().GetLines(gomock.Any(), gomock.Any()).Return(lines, nil).AnyTimes()

	accountRepo := repomock.NewMockAccountRepo(ctrl)
	accountRepo.EXPECT().GetByID(gomock.Any(), gomock.Any()).
		Return(&domain.Account{ID: "ac_1", Name: "Augno Manufacturing"}, nil).AnyTimes()

	salesOrderRepo := repomock.NewMockSalesOrderRepo(ctrl)
	salesOrderRepo.EXPECT().GetAccountOriginAddress(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewPurchaseOrderRepo().Return(poRepo).AnyTimes()
	repos.EXPECT().NewAccountRepo().Return(accountRepo).AnyTimes()
	repos.EXPECT().NewSalesOrderRepo().Return(salesOrderRepo).AnyTimes()

	return repos
}

func TestBuildPurchaseOrderSubmissionEmail(t *testing.T) {
	t.Parallel()

	repos := poEmailRepos(t, []string{"orders@fastener.example"})
	emailData, apiErr := buildPurchaseOrderSubmissionEmail(context.Background(), repos, BrandingAssets{}, "ac_1", "po_1")
	require.Nil(t, apiErr)
	require.NotNil(t, emailData)

	t.Run("addressed to the submission recipients under the legacy subject", func(t *testing.T) {
		require.Equal(t, []string{"orders@fastener.example"}, emailData.To)
		require.Equal(t, "Purchase Order 000417", emailData.Subject)
		require.Equal(t, constants.EmailTemplatePurchaseOrderSubmission, emailData.TemplateID)
	})

	t.Run("params carry the whole document, not just a letterhead", func(t *testing.T) {
		// The regression this guards: a letterhead-only map renders a header over an empty table.
		require.Equal(t, "000417", emailData.Params["order_number"])
		require.Equal(t, "$10,262.50", emailData.Params["order_total"])
		require.Equal(t, "04/18/2026", emailData.Params["requested_delivery_date"])
		require.Equal(t, true, emailData.Params["has_ship_to"])

		lines, ok := emailData.Params["lines"].([]map[string]any)
		require.True(t, ok, "lines must reach the template")
		require.Len(t, lines, 2)
		require.Equal(t, "$8.5000 / pr", lines[0]["unit_price"])
	})

	t.Run("the purchase order PDF is attached", func(t *testing.T) {
		require.NotNil(t, emailData.AttachmentData)
		require.NotNil(t, emailData.AttachmentFilename)
		require.Equal(t, "purchase-order-000417.pdf", *emailData.AttachmentFilename)
		require.Equal(t, "application/pdf", *emailData.AttachmentContentType)

		pdfBytes, err := base64.StdEncoding.DecodeString(*emailData.AttachmentData)
		require.NoError(t, err, "the attachment must be base64, which is what the sender decodes")

		// It is the real document, not an empty page.
		runs := pdfText(t, pdfBytes)
		for _, want := range []string{"PURCHASE ORDER", "000417", "WSHR-M6", "$10,262.50"} {
			require.True(t, pdfContains(runs, want), "PDF missing %q\n%s", want, pdfJoined(runs))
		}
	})
}

// An order nobody is set up to receive produces no email at all, so the caller can flag it settled
// without addressing a message to nobody.
func TestBuildPurchaseOrderSubmissionEmailWithoutRecipients(t *testing.T) {
	t.Parallel()

	repos := poEmailRepos(t, nil)
	emailData, apiErr := buildPurchaseOrderSubmissionEmail(context.Background(), repos, BrandingAssets{}, "ac_1", "po_1")
	require.Nil(t, apiErr)
	require.Nil(t, emailData)
}
