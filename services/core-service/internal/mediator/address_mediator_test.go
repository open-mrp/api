package mediator

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

const (
	inlineSupplierAccountID = "ac_supplier"
	inlineAddressID         = "ad_dock"
)

type AddressMedTestSuite struct {
	suite.Suite
	ctrl    *gomock.Controller
	address *repositorymock.MockAddressRepo
	outbox  *recordingOutboxRepo
	med     domain.AddressMed
	ctx     context.Context
}

func (s *AddressMedTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.address = repositorymock.NewMockAddressRepo(s.ctrl)
	s.outbox = &recordingOutboxRepo{}
	factory := factorymock.NewMockRepoFactory(s.ctrl)
	factory.EXPECT().NewAddressRepo().Return(s.address).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(s.outbox).AnyTimes()
	s.med = NewAddressMed(&AddressMedConfig{Repos: factory})

	accountID := "ac_buyer"
	s.ctx = appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor:  &types.IdentityActor{RelationType: types.IdentityRelationTypeInternal, ID: "usr_buyer", AccountID: &accountID},
	})
}

func (s *AddressMedTestSuite) TearDownTest() { s.ctrl.Finish() }

func TestAddressMedTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AddressMedTestSuite))
}

func strPtr(v string) *string { return &v }

func (s *AddressMedTestSuite) storedDock() *domain.Address {
	calendarID := "occd_dock"
	return &domain.Address{
		ID:                inlineAddressID,
		Name:              "Dock",
		Phone:             strPtr("555-0100"),
		ReceiveCalendarID: &calendarID,
		Geolocation:       &domain.Geolocation{StreetLine1: strPtr("9 Spindle Way"), Locality: strPtr("Los Angeles"), Country: "US"},
	}
}

func (s *AddressMedTestSuite) auditActions() []constants.AuditAction {
	var actions []constants.AuditAction
	for _, msg := range s.outbox.messages {
		var evt audit.PublishedEvent
		s.Require().NoError(json.Unmarshal(msg.Payload.Data, &evt))
		actions = append(actions, evt.Action)
	}
	return actions
}

func (s *AddressMedTestSuite) TestSaveWithoutIDCreatesInTheGivenAccount() {
	var created domain.CreateAddressParams
	s.address.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, addressID, _, _ string, params domain.CreateAddressParams) (*domain.Address, *apierror.APIError) {
			created = params
			return &domain.Address{ID: addressID, Name: params.Name, Geolocation: &domain.Geolocation{Country: params.Country}}, nil
		})

	saved, apiErr := s.med.Save(s.ctx, inlineSupplierAccountID, domain.InlineAddressParams{
		Name:    strPtr("  Receiving  "),
		Phone:   field.Clear[string](),
		Country: strPtr("US"),
	}, "bill_to_address")

	s.Require().Nil(apiErr)
	s.NotEmpty(saved.ID)
	s.Equal(inlineSupplierAccountID, created.AccountID, "a new address belongs to the account the record saves it in")
	s.Equal("Receiving", created.Name)
	s.Nil(created.Phone, "a cleared field on a new address is simply absent")
	s.Equal([]constants.AuditAction{constants.AuditActionCreate}, s.auditActions())
}

func (s *AddressMedTestSuite) TestSaveWithoutIDNeedsANameAndCountry() {
	s.address.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	for _, tc := range []struct {
		input domain.InlineAddressParams
		code  apierror.ErrorCode
		param string
	}{
		{domain.InlineAddressParams{Country: strPtr("US")}, apierror.ErrorCodeMissingField, "ship_to_address.name"},
		{domain.InlineAddressParams{Name: strPtr("Dock")}, apierror.ErrorCodeMissingField, "ship_to_address.country"},
		{domain.InlineAddressParams{Name: strPtr("Dock"), Country: strPtr(" ")}, apierror.ErrorCodeMissingField, "ship_to_address.country"},
		{domain.InlineAddressParams{Name: strPtr("  "), Country: strPtr("US")}, apierror.ErrorCodeValidationFailed, "ship_to_address.name"},
	} {
		_, apiErr := s.med.Save(s.ctx, inlineSupplierAccountID, tc.input, "ship_to_address")
		s.Require().NotNil(apiErr)
		s.Equal(tc.code, apiErr.Code)
		s.Equal(tc.param, apiErr.Param)
	}
	s.Empty(s.outbox.messages)
}

func (s *AddressMedTestSuite) TestSaveWithAnIDOutsideTheAccountIsNotFoundOnTheID() {
	s.address.EXPECT().IsInAccount(gomock.Any(), inlineSupplierAccountID, "ad_elsewhere").Return(false, nil)
	s.address.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

	_, apiErr := s.med.Save(s.ctx, inlineSupplierAccountID, domain.InlineAddressParams{ID: strPtr("ad_elsewhere"), Name: strPtr("x")}, "billing_address")

	s.Require().NotNil(apiErr)
	s.True(apierror.IsNotFound(apiErr))
	s.Equal("billing_address.id", apiErr.Param)
	s.Empty(s.outbox.messages)
}

func (s *AddressMedTestSuite) TestSaveWithAnIDUpdatesOnlyTheFieldsSent() {
	stored := s.storedDock()
	s.address.EXPECT().IsInAccount(gomock.Any(), inlineSupplierAccountID, inlineAddressID).Return(true, nil)
	s.address.EXPECT().Get(gomock.Any(), domain.GetAddressParams{AccountID: inlineSupplierAccountID, AddressID: inlineAddressID}).Return(stored, nil)

	var written domain.UpdateAddressParams
	s.address.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.UpdateAddressParams) (*domain.Address, *apierror.APIError) {
			written = params
			updated := *stored
			updated.Name = *params.Name
			updated.Phone = nil
			return &updated, nil
		})

	_, apiErr := s.med.Save(s.ctx, inlineSupplierAccountID, domain.InlineAddressParams{
		ID:    strPtr(inlineAddressID),
		Name:  strPtr("Dock 2"),
		Phone: field.Clear[string](),
	}, "shipping_address")

	s.Require().Nil(apiErr)
	s.Equal(inlineSupplierAccountID, written.AccountID)
	s.Equal(inlineAddressID, written.AddressID)
	s.True(written.Phone.IsClear(), "null clears the phone")
	calendar, ok := written.ReceiveCalendarID.Value()
	s.True(ok, "the receiving calendar the request left out is written back")
	s.Equal("occd_dock", calendar)
	s.Equal([]constants.AuditAction{constants.AuditActionUpdate}, s.auditActions())
}

func (s *AddressMedTestSuite) TestPreviewAppliesTheInlineFieldsWithoutWriting() {
	s.address.EXPECT().IsInAccount(gomock.Any(), inlineSupplierAccountID, inlineAddressID).Return(true, nil)
	s.address.EXPECT().Get(gomock.Any(), gomock.Any()).Return(s.storedDock(), nil)
	s.address.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

	got, apiErr := s.med.Preview(s.ctx, inlineSupplierAccountID, domain.InlineAddressParams{
		ID:         strPtr(inlineAddressID),
		State:      strPtr("CA"),
		PostalCode: strPtr("90001"),
	}, "ship_to_address")

	s.Require().Nil(apiErr)
	s.Equal("Dock", got.Name)
	s.Equal("555-0100", *got.Phone)
	s.Equal("9 Spindle Way", *got.Geolocation.StreetLine1)
	s.Equal("CA", *got.Geolocation.State)
	s.Equal("90001", *got.Geolocation.PostalCode)
	s.Empty(s.outbox.messages)
}

func (s *AddressMedTestSuite) TestPreviewOfANewAddressIsBuiltFromItsFields() {
	got, apiErr := s.med.Preview(s.ctx, inlineSupplierAccountID, domain.InlineAddressParams{
		Name:       strPtr("Dock"),
		IsDropShip: func() *bool { v := true; return &v }(),
		State:      strPtr("OH"),
		Country:    strPtr("US"),
	}, "ship_to_address")

	s.Require().Nil(apiErr)
	s.Equal("Dock", got.Name)
	s.True(got.IsDropShip)
	s.Equal("OH", *got.Geolocation.State)
	s.Equal("US", got.Geolocation.Country)
}
