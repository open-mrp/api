package constants

// SysPropertyTypeCode represents the code for a system property type.
type SysPropertyTypeCode string

const (
	// SysPropertyTypeCodeTransactionNumber is the type code for transaction numbers.
	SysPropertyTypeCodeTransactionNumber SysPropertyTypeCode = "transaction_number"
	// SysPropertyTypeCodeSettlementNumber is the type code for settlement numbers.
	SysPropertyTypeCodeSettlementNumber SysPropertyTypeCode = "settlement_number"
	// SysPropertyTypeCodeSalesOrderNumber is the type code for sales order numbers.
	SysPropertyTypeCodeSalesOrderNumber SysPropertyTypeCode = "sales_order_number"
	// SysPropertyTypeCodePurchaseOrderNumber is the type code for purchase order numbers.
	SysPropertyTypeCodePurchaseOrderNumber SysPropertyTypeCode = "purchase_order_number"
	// SysPropertyTypeCodeSupplierNumber is the type code for supplier numbers.
	SysPropertyTypeCodeSupplierNumber SysPropertyTypeCode = "supplier_number"
	// SysPropertyTypeCodeCustomerNumber is the type code for customer numbers.
	SysPropertyTypeCodeCustomerNumber SysPropertyTypeCode = "customer_number"
	// SysPropertyTypeCodeSsccCount is the type code for SSCC counts.
	SysPropertyTypeCodeSsccCount SysPropertyTypeCode = "sscc_count"
	// SysPropertyTypeCodeProductionRunNumber is the type code for production run numbers.
	SysPropertyTypeCodeProductionRunNumber SysPropertyTypeCode = "production_run_number"
)

func (m SysPropertyTypeCode) IsValid() bool {
	switch m {
	case SysPropertyTypeCodeTransactionNumber, SysPropertyTypeCodeSettlementNumber, SysPropertyTypeCodeSalesOrderNumber, SysPropertyTypeCodePurchaseOrderNumber, SysPropertyTypeCodeSupplierNumber, SysPropertyTypeCodeCustomerNumber, SysPropertyTypeCodeSsccCount, SysPropertyTypeCodeProductionRunNumber:
		return true
	default:
		return false
	}
}

func (m SysPropertyTypeCode) EnumValues() []string {
	return []string{
		string(SysPropertyTypeCodeTransactionNumber),
		string(SysPropertyTypeCodeSettlementNumber),
		string(SysPropertyTypeCodeSalesOrderNumber),
		string(SysPropertyTypeCodePurchaseOrderNumber),
		string(SysPropertyTypeCodeSupplierNumber),
		string(SysPropertyTypeCodeCustomerNumber),
		string(SysPropertyTypeCodeSsccCount),
		string(SysPropertyTypeCodeProductionRunNumber),
	}
}
