package transactionep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func TransactionDetailPresenter(d *pb.TransactionInfo) apiresource.TransactionDetail {
	if d == nil {
		return apiresource.TransactionDetail{}
	}

	tx := apiresource.TransactionDetail{
		ID:               d.Id,
		Object:           constants.ObjectTypeTransaction,
		Number:           d.Number,
		IsFullyAllocated: d.IsFullyAllocated,
		StripePaymentID:  d.StripePaymentId,
		AllocationCount:  d.AllocationCount,
		CreatedAt:        grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(d.UpdatedAt),
	}

	tx.Amount = &apiresource.Quantity{
		ID:           d.AmountId,
		Object:       constants.ObjectTypeQuantity,
		Value:        d.AmountValue,
		DisplayValue: apiresource.FormatDisplayValue(d.AmountValue, d.AmountUnitAbbreviation, string(constants.UnitTypeCurrency)),
		Unit: &apiresource.Unit{
			ID:     d.AmountUnitId,
			Object: constants.ObjectTypeUnit,
		},
	}

	if d.CustomerId != nil {
		customer := &apiresource.Customer{
			ID:               *d.CustomerId,
			Object:           constants.ObjectTypeCustomer,
			EDIStatus:        constants.EDIStatusDisabled,
			RelationshipType: constants.CustomerRelationshipTypeStandalone,
			CreatedAt:        grpcutil.TimestampToTime(d.CustomerCreatedAt),
			UpdatedAt:        grpcutil.TimestampToTime(d.CustomerUpdatedAt),
		}
		if d.CustomerName != nil {
			customer.Name = *d.CustomerName
		}
		if d.CustomerNumber != nil {
			customer.Number = *d.CustomerNumber
		}
		if d.CustomerStatus != nil {
			customer.Status = constants.AccountStatusCode(*d.CustomerStatus)
		} else {
			customer.Status = constants.AccountStatusCodeNormal
		}
		if d.CustomerCommissionPolicy != nil {
			customer.CommissionPolicy = constants.CommissionPolicy(*d.CustomerCommissionPolicy)
		} else {
			customer.CommissionPolicy = constants.CommissionPolicyApplied
		}
		tx.Customer = customer
	}

	if d.ResponsibleUserId != nil {
		user := &apiresource.AccountUser{
			ID:        *d.ResponsibleUserId,
			Object:    constants.ObjectTypeAccountUser,
			Name:      d.ResponsibleUserName,
			CreatedAt: grpcutil.TimestampToTime(d.ResponsibleUserCreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(d.ResponsibleUserUpdatedAt),
		}
		if d.ResponsibleUserStatus != nil {
			user.Status = constants.AccountUserStatus(*d.ResponsibleUserStatus)
		} else {
			user.Status = constants.AccountUserStatusActive
		}
		tx.ResponsibleUser = user
	}

	tx.Note = d.Note

	tx.TransactionType = &apiresource.TransactionType{
		ID:     d.TransactionTypeId,
		Object: constants.ObjectTypeTransactionType,
		Name:   d.TransactionTypeName,
		Code:   constants.TransactionType(d.TransactionTypeCode),
	}

	if d.TransactionMethodId != nil {
		tx.TransactionMethod = &apiresource.TransactionMethod{
			ID:     *d.TransactionMethodId,
			Object: constants.ObjectTypeTransactionMethod,
		}
		if d.TransactionMethodName != nil {
			tx.TransactionMethod.Name = *d.TransactionMethodName
		}
		if d.TransactionMethodCode != nil {
			tx.TransactionMethod.Code = constants.TransactionMethod(*d.TransactionMethodCode)
		}
	}

	if d.AdjustmentTypeId != nil {
		tx.AdjustmentType = &apiresource.AdjustmentType{
			ID:     *d.AdjustmentTypeId,
			Object: constants.ObjectTypeAdjustmentType,
		}
		if d.AdjustmentTypeName != nil {
			tx.AdjustmentType.Name = *d.AdjustmentTypeName
		}
		if d.AdjustmentTypeCode != nil {
			tx.AdjustmentType.Code = constants.AdjustmentType(*d.AdjustmentTypeCode)
		}
	}

	if d.Allocations != nil {
		allocations := make([]apiresource.TransactionAllocation, len(d.Allocations))
		for i, a := range d.Allocations {
			allocations[i] = TransactionAllocationPresenter(a)
		}
		tx.Allocations = apiresource.NewList(allocations, apiresource.PageInfo{})
	}

	return tx
}

func TransactionSummaryPresenter(d *pb.TransactionSummaryInfo) apiresource.TransactionSummary {
	if d == nil {
		return apiresource.TransactionSummary{}
	}

	ts := apiresource.TransactionSummary{
		ID:               d.Id,
		Object:           constants.ObjectTypeTransactionSummary,
		Number:           d.Number,
		IsFullyAllocated: d.IsFullyAllocated,
		AllocationCount:  d.AllocationCount,
		CreatedAt:        grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(d.UpdatedAt),
	}

	ts.Amount = &apiresource.Quantity{
		ID:           d.AmountId,
		Object:       constants.ObjectTypeQuantity,
		Value:        d.AmountValue,
		DisplayValue: apiresource.FormatDisplayValue(d.AmountValue, d.AmountUnitAbbreviation, string(constants.UnitTypeCurrency)),
		Unit: &apiresource.Unit{
			ID:     d.AmountUnitId,
			Object: constants.ObjectTypeUnit,
		},
	}

	if d.CustomerId != nil {
		customer := &apiresource.Customer{
			ID:               *d.CustomerId,
			Object:           constants.ObjectTypeCustomer,
			EDIStatus:        constants.EDIStatusDisabled,
			RelationshipType: constants.CustomerRelationshipTypeStandalone,
			CreatedAt:        grpcutil.TimestampToTime(d.CustomerCreatedAt),
			UpdatedAt:        grpcutil.TimestampToTime(d.CustomerUpdatedAt),
		}
		if d.CustomerName != nil {
			customer.Name = *d.CustomerName
		}
		if d.CustomerNumber != nil {
			customer.Number = *d.CustomerNumber
		}
		if d.CustomerStatus != nil {
			customer.Status = constants.AccountStatusCode(*d.CustomerStatus)
		} else {
			customer.Status = constants.AccountStatusCodeNormal
		}
		if d.CustomerCommissionPolicy != nil {
			customer.CommissionPolicy = constants.CommissionPolicy(*d.CustomerCommissionPolicy)
		} else {
			customer.CommissionPolicy = constants.CommissionPolicyApplied
		}
		ts.Customer = customer
	}

	ts.TransactionType = &apiresource.TransactionType{
		ID:     d.TransactionTypeId,
		Object: constants.ObjectTypeTransactionType,
		Name:   d.TransactionTypeName,
		Code:   constants.TransactionType(d.TransactionTypeCode),
	}

	if d.TransactionMethodId != nil {
		ts.TransactionMethod = &apiresource.TransactionMethod{
			ID:     *d.TransactionMethodId,
			Object: constants.ObjectTypeTransactionMethod,
		}
		if d.TransactionMethodName != nil {
			ts.TransactionMethod.Name = *d.TransactionMethodName
		}
		if d.TransactionMethodCode != nil {
			ts.TransactionMethod.Code = constants.TransactionMethod(*d.TransactionMethodCode)
		}
	}

	if d.AdjustmentTypeId != nil {
		ts.AdjustmentType = &apiresource.AdjustmentType{
			ID:     *d.AdjustmentTypeId,
			Object: constants.ObjectTypeAdjustmentType,
		}
		if d.AdjustmentTypeName != nil {
			ts.AdjustmentType.Name = *d.AdjustmentTypeName
		}
		if d.AdjustmentTypeCode != nil {
			ts.AdjustmentType.Code = constants.AdjustmentType(*d.AdjustmentTypeCode)
		}
	}

	return ts
}

func TransactionAllocationPresenter(a *pb.TransactionAllocationInfo) apiresource.TransactionAllocation {
	if a == nil {
		return apiresource.TransactionAllocation{}
	}

	alloc := apiresource.TransactionAllocation{
		ID:     a.Id,
		Object: constants.ObjectTypeTransactionAllocation,
		Amount: &apiresource.Quantity{
			ID:           a.AmountId,
			Object:       constants.ObjectTypeQuantity,
			Value:        a.AmountValue,
			DisplayValue: apiresource.FormatDisplayValue(a.AmountValue, a.AmountUnitAbbreviation, string(constants.UnitTypeCurrency)),
			Unit: &apiresource.Unit{
				ID:     a.AmountUnitId,
				Object: constants.ObjectTypeUnit,
			},
		},
		Note: a.Note,
		Transaction: &apiresource.TransactionDetail{
			ID:     a.TransactionId,
			Object: constants.ObjectTypeTransaction,
		},
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}

	if a.InvoiceId != nil {
		alloc.Invoice = &apiresource.InvoiceSummary{
			ID:     *a.InvoiceId,
			Object: constants.ObjectTypeInvoiceSummary,
		}
		if a.InvoiceNumber != nil {
			alloc.Invoice.Number = *a.InvoiceNumber
		}
	}

	return alloc
}

func TransactionListPresenter(resp *pb.ListTransactionsResponse) *apiresource.List[apiresource.TransactionSummary] {
	if resp == nil {
		return apiresource.NewList[apiresource.TransactionSummary](nil, apiresource.PageInfo{})
	}

	transactions := make([]apiresource.TransactionSummary, len(resp.Transactions))
	for i, d := range resp.Transactions {
		transactions[i] = TransactionSummaryPresenter(d)
	}

	return apiresource.NewList(transactions, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func AccountTransactionListPresenter(resp *pb.ListAccountTransactionsResponse) *apiresource.List[apiresource.TransactionDetail] {
	if resp == nil {
		return apiresource.NewList[apiresource.TransactionDetail](nil, apiresource.PageInfo{})
	}

	transactions := make([]apiresource.TransactionDetail, len(resp.Transactions))
	for i, d := range resp.Transactions {
		transactions[i] = TransactionDetailPresenter(d)
	}

	return apiresource.NewList(transactions, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func AdjustmentTypePresenter(at *pb.AdjustmentTypeInfo) apiresource.AdjustmentType {
	if at == nil {
		return apiresource.AdjustmentType{}
	}

	return apiresource.AdjustmentType{
		ID:        at.Id,
		Object:    constants.ObjectTypeAdjustmentType,
		Name:      at.Name,
		Code:      constants.AdjustmentType(at.Code),
		Owner:     apiresource.SystemOwner(),
		CreatedAt: grpcutil.TimestampToTime(at.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(at.UpdatedAt),
	}
}

func AdjustmentTypeListPresenter(resp *pb.ListAdjustmentTypesResponse) *apiresource.List[apiresource.AdjustmentType] {
	if resp == nil {
		return apiresource.NewList[apiresource.AdjustmentType](nil, apiresource.PageInfo{})
	}

	adjustmentTypes := make([]apiresource.AdjustmentType, len(resp.AdjustmentTypes))
	for i, at := range resp.AdjustmentTypes {
		adjustmentTypes[i] = AdjustmentTypePresenter(at)
	}

	return apiresource.NewList(adjustmentTypes, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
