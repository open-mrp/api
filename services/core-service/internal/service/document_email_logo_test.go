package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"

	"go.uber.org/mock/gomock"
)

// account_branding.logo_url holds an object key, and a mail client given the bare key renders a
// broken image. These cover the emails that carry the merchant's logo.

const (
	storedLogoKey = "ac_1/logo.png"
	signedLogoURL = "https://s3.example.com/ac_1/logo.png?signature=abc"
)

func logoAccount() *domain.Account {
	return &domain.Account{
		ID:       "ac_1",
		Name:     "Acme Company",
		Branding: &domain.AccountBranding{LogoURL: poPtr(storedLogoKey)},
	}
}

func signingBranding(t *testing.T) BrandingAssets {
	t.Helper()
	return NewBrandingAssets(&stubBrandingStore{object: redPNG(t), presignedURL: signedLogoURL}, "account-photos")
}

func TestFetchAccountLogo_SignsTheStoredKey(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repos := factorymock.NewMockRepoFactory(ctrl)
	accounts := repositorymock.NewMockAccountRepo(ctrl)
	repos.EXPECT().NewAccountRepo().Return(accounts).AnyTimes()
	accounts.EXPECT().GetByID(gomock.Any(), "ac_1").Return(logoAccount(), nil)

	logo := fetchAccountLogo(context.Background(), repos, signingBranding(t), "ac_1")

	if logo.URL != signedLogoURL {
		t.Errorf("URL = %q, want the signed link, not the stored key", logo.URL)
	}
	if len(logo.Image) == 0 {
		t.Error("Image is empty, want the logo bytes for the PDF")
	}
}

func TestBuildInvoiceEmail_CarriesTheSignedLogoAndSKUs(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repos := factorymock.NewMockRepoFactory(ctrl)
	accounts := repositorymock.NewMockAccountRepo(ctrl)
	invoices := repositorymock.NewMockInvoiceRepo(ctrl)
	orders := repositorymock.NewMockSalesOrderRepo(ctrl)
	customers := repositorymock.NewMockCustomerRepo(ctrl)
	repos.EXPECT().NewAccountRepo().Return(accounts).AnyTimes()
	repos.EXPECT().NewInvoiceRepo().Return(invoices).AnyTimes()
	repos.EXPECT().NewSalesOrderRepo().Return(orders).AnyTimes()
	repos.EXPECT().NewCustomerRepo().Return(customers).AnyTimes()

	invoice, lines, _ := invoiceFixture()
	invoice.ID = "iv_1"
	// No order behind it, so the letterhead comes straight off the account's branding row.
	invoice.OrderID = ""
	for _, l := range lines {
		l.PricingQuantityRatioNumerator, l.PricingQuantityRatioDenominator = "1", "1"
		l.PricingPriceRatioNumerator, l.PricingPriceRatioDenominator = "1", "1"
	}

	accounts.EXPECT().GetByID(gomock.Any(), "ac_1").Return(logoAccount(), nil).AnyTimes()
	invoices.EXPECT().Get(gomock.Any(), gomock.Any()).Return(invoice, nil)
	invoices.EXPECT().GetLines(gomock.Any(), "iv_1").Return(lines, nil)
	invoices.EXPECT().GetEmailRecipients(gomock.Any(), "iv_1").Return(nil, nil)
	orders.EXPECT().GetAccountOriginAddress(gomock.Any(), "ac_1").Return(nil, nil)
	customers.EXPECT().Get(gomock.Any(), "ac_1", gomock.Any(), gomock.Any()).Return(nil, nil)

	email, apiErr := buildInvoiceEmail(context.Background(), repos, signingBranding(t), "ac_1", "iv_1")
	if apiErr != nil {
		t.Fatalf("buildInvoiceEmail: %v", apiErr)
	}

	if got := email.params["logo_url"]; got != signedLogoURL {
		t.Errorf("logo_url = %v, want the signed link, not the stored key", got)
	}

	rows, ok := email.params["lines"].([]map[string]any)
	if !ok || len(rows) != 2 {
		t.Fatalf("lines = %#v", email.params["lines"])
	}
	for i, want := range []string{"SOCK-CREW-BLK", "SOCK-ANK-WHT"} {
		if rows[i]["sku"] != want {
			t.Errorf("lines[%d].sku = %v, want %q", i, rows[i]["sku"], want)
		}
	}
}
