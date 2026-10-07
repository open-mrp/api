package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	servicemock "github.com/open-mrp/api/services/core-service/internal/domain/mock/service"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/pagination"
)

const customerExportSheet = "Customers"

type CustomerExportTestSuite struct {
	suite.Suite
	svc               *customerSvcImpl
	repoFactory       *factorymock.MockRepoFactory
	customerRepo      *repositorymock.MockCustomerRepo
	accountStatusRepo *repositorymock.MockAccountStatusRepo
	accountUserRepo   *repositorymock.MockAccountUserRepo
	idempotencyMed    *mediatormock.MockIdempotencyMed
	jobSvc            *servicemock.MockJobSvc
	outbox            *capturingOutboxRepo
	ctrl              *gomock.Controller
}

func (suite *CustomerExportTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.customerRepo = repositorymock.NewMockCustomerRepo(suite.ctrl)
	suite.accountStatusRepo = repositorymock.NewMockAccountStatusRepo(suite.ctrl)
	suite.accountUserRepo = repositorymock.NewMockAccountUserRepo(suite.ctrl)
	suite.outbox = &capturingOutboxRepo{}

	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewCustomerRepo().Return(suite.customerRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewAccountStatusRepo().Return(suite.accountStatusRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewAccountUserRepo().Return(suite.accountUserRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(suite.outbox).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	mediatorFactory := factorymock.NewMockMediatorFactory(suite.ctrl)
	mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: suite.idempotencyMed}).AnyTimes()

	suite.jobSvc = servicemock.NewMockJobSvc(suite.ctrl)
	jobSvcFactory := factorymock.NewMockJobSvcFactory(suite.ctrl)
	jobSvcFactory.EXPECT().Build(gomock.Any()).Return(suite.jobSvc).AnyTimes()

	suite.svc = NewCustomerSvc(&CustomerSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
	}).(*customerSvcImpl)
}

func (suite *CustomerExportTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestCustomerExportTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CustomerExportTestSuite))
}

func (suite *CustomerExportTestSuite) expectAccountStatuses() {
	suite.accountStatusRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(&domain.ListAccountStatusesResult{
		AccountStatuses: []*domain.AccountStatus{
			{Code: string(constants.AccountStatusCodeNormal), Name: "Normal"},
			{Code: string(constants.AccountStatusCodeHoldShipment), Name: "Hold Shipment"},
		},
	}, nil).Times(1)
}

// renders the workbook the export worker would build for the account the caller acts for
func (suite *CustomerExportTestSuite) build(ctx context.Context, filters domain.ListCustomersParams) (*domain.Export, *apierror.APIError) {
	raw, err := json.Marshal(filters)
	suite.Require().NoError(err)
	return suite.svc.BuildExportCustomers(ctx, "ac_owner", raw)
}

func onePage(customers ...*domain.Customer) *domain.ListCustomersResult {
	return &domain.ListCustomersResult{Items: customers}
}

func fullCustomer() *domain.Customer {
	return &domain.Customer{
		ID:                      "ac_harbor",
		Name:                    "Harbor Supply",
		Number:                  "8841",
		Status:                  constants.AccountStatusCodeHoldShipment,
		CommissionPolicy:        constants.CommissionPolicyExempt,
		FreightPolicy:           constants.FreightPolicyBilled,
		Note:                    new("Ships Tuesdays"),
		Email:                   new("orders@harbor.test"),
		Phone:                   new("555-0100"),
		TypeGroupName:           new("Distributors"),
		DefaultSalesRepName:     new("Pat Lee"),
		DefaultPriorityName:     new("High"),
		DefaultPaymentTermName:  new("Net 30"),
		DefaultShippingTermName: new("FOB Origin"),
		DefaultCarrierName:      new("Freight Co"),
		ParentAccountID:         new("ac_parent"),
		BillToAddress: &domain.CustomerAddress{Geolocation: &domain.CustomerGeolocation{
			StreetLine1: new("1 Main St"), StreetLine2: new("Suite 5"), Locality: new("Springfield"), State: new("IL"), PostalCode: new("62701"), Country: "US",
		}},
		ShipToAddress: &domain.CustomerAddress{Geolocation: &domain.CustomerGeolocation{
			StreetLine1: new("9 Dock Rd"), Locality: new("Springfield"), State: new("IL"), PostalCode: new("62702"), Country: "US",
		}},
		CreatedAt: time.Date(2026, 3, 1, 15, 0, 0, 0, time.UTC),
	}
}

func bareCustomer() *domain.Customer {
	return &domain.Customer{
		ID:               "ac_bare",
		Name:             "Bare Goods",
		Number:           "C-12",
		Status:           constants.AccountStatusCodeNormal,
		CommissionPolicy: constants.CommissionPolicyApplied,
		FreightPolicy:    constants.FreightPolicyFree,
		CreatedAt:        time.Date(2026, 2, 14, 9, 0, 0, 0, time.UTC),
	}
}

func (suite *CustomerExportTestSuite) TestExport_MatchesTheDashboardsFile() {
	ctx := internalIdentityCtx("ac_owner")
	suite.expectAccountStatuses()
	suite.customerRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(onePage(fullCustomer(), bareCustomer()), nil)
	suite.customerRepo.EXPECT().ListContacts(gomock.Any(), []string{"ac_harbor", "ac_bare"}).Return([]domain.CustomerContact{
		{CustomerAccountID: "ac_harbor", Name: new("Jane Doe"), Email: new("jane@harbor.test")},
		{CustomerAccountID: "ac_harbor", Name: new("Sam Roe")},
		{CustomerAccountID: "ac_harbor"},
		{CustomerAccountID: "ac_harbor", Email: new("dock@harbor.test")},
	}, nil)

	export, apiErr := suite.build(ctx, domain.ListCustomersParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(2), export.RowCount)

	rows := exportedSheetRows(suite.T(), export, customerExportSheet)
	suite.Require().Len(rows, 3, "a header and two customers")
	suite.Equal([]string{
		"Customer Number", "Name", "Email", "Phone", "Status", "Customer Group", "Sales Rep", "Priority",
		"Payment Term", "Shipping Term", "Carrier", "Default Billing", "Default Shipping", "Contacts",
		"Commission Exempt", "Freight Exempt", "Parent Account", "Note", "Created At",
	}, rows[0])

	suite.Equal([]string{
		"08841", "Harbor Supply", "orders@harbor.test", "555-0100", "Hold Shipment", "Distributors", "Pat Lee", "High",
		"Net 30", "FOB Origin", "Freight Co",
		"1 Main St, Suite 5, Springfield, IL 62701 US",
		"9 Dock Rd, Springfield, IL 62702 US",
		"Jane Doe (jane@harbor.test); Sam Roe; dock@harbor.test",
		"Yes", "No", "Yes", "Ships Tuesdays", "03/01/2026",
	}, rows[1])

	// Blank cells mid-row read back as empty strings; the row ends at its last filled column.
	suite.Equal([]string{
		"C-12", "Bare Goods", "", "", "Normal", "", "", "",
		"", "", "", "", "", "",
		"No", "Yes", "No", "", "02/14/2026",
	}, rows[2])
}

func (suite *CustomerExportTestSuite) TestExport_EmptyAccountYieldsHeaderOnlyFile() {
	ctx := internalIdentityCtx("ac_owner")
	suite.expectAccountStatuses()
	suite.customerRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(onePage(), nil)
	suite.customerRepo.EXPECT().ListContacts(gomock.Any(), []string{}).Return(nil, nil)

	export, apiErr := suite.build(ctx, domain.ListCustomersParams{})
	suite.Require().Nil(apiErr)

	rows := exportedSheetRows(suite.T(), export, customerExportSheet)
	suite.Require().Len(rows, 1)
	suite.Equal("Customer Number", rows[0][0])
}

// The export reads the list itself, page by page, so the file holds exactly the customers the list
// shows for the same filters — and only the caller's account's, whatever the stored filters say.
func (suite *CustomerExportTestSuite) TestExport_WalksTheListWithItsFilters() {
	ctx := internalIdentityCtx("ac_owner")
	suite.expectAccountStatuses()
	query := "harbor"
	stored := domain.ListCustomersParams{
		AccountID:        "ac_attacker",
		Query:            &query,
		StatusCodes:      []string{string(constants.AccountStatusCodeNormal)},
		CustomerGroupIDs: []string{"acgp_dist"},
	}

	first, second := fullCustomer(), bareCustomer()
	third := bareCustomer()
	third.ID = "ac_third"
	next := "cur_2"
	gomock.InOrder(
		suite.customerRepo.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, p domain.ListCustomersParams) (*domain.ListCustomersResult, *apierror.APIError) {
				suite.Equal("ac_owner", p.AccountID, "the account is the caller's, never the request's")
				suite.Nil(p.Cursor)
				suite.Equal(int32(customerExportPageSize), p.Limit)
				suite.Empty(p.Includes)
				suite.Equal(stored.Query, p.Query)
				suite.Equal(stored.StatusCodes, p.StatusCodes)
				suite.Equal(stored.CustomerGroupIDs, p.CustomerGroupIDs)
				return &domain.ListCustomersResult{
					Items:    []*domain.Customer{first, second},
					PageInfo: pagination.PageInfo{HasNextPage: true, NextCursor: &next},
				}, nil
			}),
		suite.customerRepo.EXPECT().ListContacts(gomock.Any(), []string{"ac_harbor", "ac_bare"}).Return(nil, nil),
		suite.customerRepo.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, p domain.ListCustomersParams) (*domain.ListCustomersResult, *apierror.APIError) {
				suite.Require().NotNil(p.Cursor)
				suite.Equal(next, *p.Cursor)
				suite.Equal(stored.CustomerGroupIDs, p.CustomerGroupIDs, "every page carries the filters")
				return onePage(third), nil
			}),
		suite.customerRepo.EXPECT().ListContacts(gomock.Any(), []string{"ac_third"}).Return(nil, nil),
	)

	export, apiErr := suite.build(ctx, stored)
	suite.Require().Nil(apiErr)

	rows := exportedSheetRows(suite.T(), export, customerExportSheet)
	suite.Require().Len(rows, 4)
	suite.Equal("Harbor Supply", rows[1][1], "in list order")
	suite.Equal("Bare Goods", rows[2][1])
	suite.Equal("Bare Goods", rows[3][1])
}

// One page past the cap is all it takes to refuse the export; reading on would only spend memory.
func (suite *CustomerExportTestSuite) TestExport_StopsReadingOncePastTheRowLimit() {
	ctx := internalIdentityCtx("ac_owner")
	suite.expectAccountStatuses()
	page := make([]*domain.Customer, customerExportPageSize)
	for i := range page {
		page[i] = bareCustomer()
	}
	next := "cur_more"
	pagesToPassTheCap := domain.ExportRowLimit/customerExportPageSize + 1
	suite.customerRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(&domain.ListCustomersResult{
		Items:    page,
		PageInfo: pagination.PageInfo{HasNextPage: true, NextCursor: &next},
	}, nil).Times(pagesToPassTheCap)
	suite.customerRepo.EXPECT().ListContacts(gomock.Any(), gomock.Any()).Return(nil, nil).Times(pagesToPassTheCap)

	export, apiErr := suite.build(ctx, domain.ListCustomersParams{})
	suite.Nil(export)
	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorTypeInvalidRequest, apiErr.Type)
}

// The accept phase records only the list's filters: paging and includes belong to the export, and the
// account is the caller's at render time.
func (suite *CustomerExportTestSuite) TestExportCustomers_RecordsTheListFiltersOnAnExportJob() {
	ctx := idempotencyCtx(internalIdentityCtx("ac_owner"))
	suite.accountUserRepo.EXPECT().ResolveAccountUserID(gomock.Any(), gomock.Any(), gomock.Any()).Return("acus_1", nil).AnyTimes()
	suite.idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_1", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil)
	suite.idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_1", gomock.Any()).Return(nil)

	var recorded domain.CreateJobServiceParams
	suite.jobSvc.EXPECT().CreateJob(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, params domain.CreateJobServiceParams) (*domain.Job, *apierror.APIError) {
			recorded = params
			return &domain.Job{ID: "jb_1"}, nil
		})

	query, cursor := "harbor", "cur_x"
	isParent := true
	job, apiErr := suite.svc.ExportCustomers(ctx, domain.ListCustomersParams{
		AccountID:       "ac_attacker",
		Cursor:          &cursor,
		Limit:           5,
		Includes:        []string{"price_groups"},
		Query:           &query,
		SalesRepIDs:     []string{"acus_rep"},
		IsParentAccount: &isParent,
	})
	suite.Require().Nil(apiErr)
	suite.Equal("jb_1", job.ID)
	suite.Equal(constants.JobTypeExport, recorded.Type)
	suite.Equal(constants.ObjectTypeCustomer, recorded.ResourceType)

	var payload exportJobPayload
	suite.Require().NoError(json.Unmarshal(recorded.JobItems, &payload))
	suite.Equal("customers", payload.Slug)
	var filters domain.ListCustomersParams
	suite.Require().NoError(json.Unmarshal(payload.Filters, &filters))
	suite.Equal(domain.ListCustomersParams{Query: &query, SalesRepIDs: []string{"acus_rep"}, IsParentAccount: &isParent}, filters)

	suite.Require().Len(suite.outbox.messages, 1)
	suite.Equal(string(messaging.ExportCustomers.RoutingKey()), suite.outbox.messages[0].RoutingKey)
}

// Like the dashboard's export, only the account's own staff may export its customers, and only with customers:read.
func (suite *CustomerExportTestSuite) TestExportCustomers_RefusesWhoeverMayNotReadCustomers() {
	for name, ctx := range map[string]context.Context{
		"no identity":      context.Background(),
		"customer portal":  customerProductIdentityCtx("ac_customer", "ac_owner"),
		"no customer read": readOnlyProductIdentityCtx("ac_owner"),
	} {
		job, apiErr := suite.svc.ExportCustomers(ctx, domain.ListCustomersParams{})
		suite.Nil(job, name)
		suite.NotNil(apiErr, name)
	}
}

func TestCustomerAddressLine_JoinsLikeTheDashboard(t *testing.T) {
	t.Parallel()
	geo := func(g domain.CustomerGeolocation) *domain.CustomerAddress {
		return &domain.CustomerAddress{Geolocation: &g}
	}
	for name, tc := range map[string]struct {
		address *domain.CustomerAddress
		want    string
	}{
		"no address":        {nil, ""},
		"country only":      {geo(domain.CustomerGeolocation{Country: "US"}), "US"},
		"two parts":         {geo(domain.CustomerGeolocation{Locality: new("Springfield"), Country: "US"}), "Springfield, US"},
		"blank parts drop":  {geo(domain.CustomerGeolocation{StreetLine1: new(""), State: new("IL"), PostalCode: new("62701"), Country: "US"}), "IL 62701 US"},
		"comma in a street": {geo(domain.CustomerGeolocation{StreetLine1: new("Unit 4, 1 Main St"), Locality: new("Springfield"), State: new("IL"), PostalCode: new("62701"), Country: "US"}), "Unit 4, 1 Main St, Springfield, IL 62701 US"},
	} {
		if got := customerAddressLine(tc.address); got != tc.want {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}
