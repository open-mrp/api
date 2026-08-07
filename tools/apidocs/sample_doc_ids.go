package main

import (
	"regexp"
	"strings"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

// augnoIDPattern matches primary-key style docs IDs: short prefix + nano segment (length >= 12).
var augnoIDPattern = regexp.MustCompile(`^[a-z]{2,15}_[0-9a-z]{12,}$`)

// isAugnoLikeDocID reports whether s looks like an Augno type id for docs validation. The nano segment must contain at least one digit so strings like "order_acknowledgement" are excluded.
func isAugnoLikeDocID(s string) bool {
	if !augnoIDPattern.MatchString(s) {
		return false
	}
	i := strings.IndexByte(s, '_')
	if i < 0 || i+1 >= len(s) {
		return false
	}
	for _, r := range s[i+1:] {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

// documentedAugnoIDs is every Sample*ID used in OpenAPI examples that matches our type-id shape. Strings in request/response examples matching isAugnoLikeDocID must appear here or the completeness test fails.
func documentedAugnoIDs() map[string]struct{} {
	m := map[string]struct{}{
		apiresource.SampleAccountID:                           {},
		apiresource.SampleMachineDowntimeEventID:              {},
		apiresource.SampleMachineDowntimeReasonID:             {},
		apiresource.SampleDemandOverrideID:                    {},
		apiresource.SampleDemandOverrideTypeID:                {},
		apiresource.SampleProductionScheduleID:                {},
		apiresource.SampleProductionScheduleLineID:            {},
		apiresource.SampleProductionScheduleItemPolicyID:      {},
		apiresource.SampleProductionScheduleDeviationID:       {},
		apiresource.SampleProductionScheduleDerivedLineID:     {},
		apiresource.SampleProductionScheduleResourceSettingID: {},
		apiresource.SampleProductionScheduleItemSettingID:     {},
		apiresource.SampleProductionScheduleFinishedPolicyID:  {},
		apiresource.SampleScheduleDeviationTypeID:             {},
		apiresource.SampleAccountBrandingID:                   {},
		apiresource.SampleAccountPortalID:                     {},
		apiresource.SampleAccountIntegrationID:                {},
		apiresource.SampleAccountGroupID:                      {},
		apiresource.SampleAccountPriceID:                      {},
		apiresource.SampleAccountStatusID:                     {},
		apiresource.SampleAccountUserID:                       {},
		apiresource.SampleAdjustmentTypeID:                    {},
		apiresource.SampleAgentActionID:                       {},
		apiresource.SampleAgentDefinitionID:                   {},
		apiresource.SampleAgentDefinitionToolID:               {},
		apiresource.SampleAgentMemoryID:                       {},
		apiresource.SampleAgentRunID:                          {},
		apiresource.SampleAgentRunStepID:                      {},
		apiresource.SampleAnnouncementID:                      {},
		apiresource.SampleNotificationID:                      {},
		apiresource.SampleConversationID:                      {},
		apiresource.SampleConversationParticipantID:           {},
		apiresource.SampleConversationLinkID:                  {},
		apiresource.SampleMessageID:                           {},
		apiresource.SampleMessagingBlockID:                    {},
		apiresource.SampleNotificationPreferenceID:            {},
		apiresource.SampleMessageAttachmentID:                 {},
		apiresource.SampleAllocationEntryID:                   {},
		apiresource.SampleAPIKeyID:                            {},
		apiresource.SampleAttributeID:                         {},
		apiresource.SampleAuditEventID:                        {},
		string(apiresource.SampleAvailableToolSlug):           {},
		apiresource.SampleBatchID:                             {},
		apiresource.SampleCarrierID:                           {},
		apiresource.SampleChildAccountRelationID:              {},
		apiresource.SampleConsumptionID:                       {},
		apiresource.SampleCRUDAddressID:                       {},
		apiresource.SampleCustomerID:                          {},
		apiresource.SampleCustomerRelationID:                  {},
		apiresource.SampleDCLocationID:                        {},
		apiresource.SampleDeliveryID:                          {},
		apiresource.SampleDeliveryLineID:                      {},
		apiresource.SampleDepartmentID:                        {},
		apiresource.SampleEmailLogID:                          {},
		apiresource.SampleEmailDomainID:                       {},
		apiresource.SamplePortalDomainID:                      {},
		apiresource.SampleEmailInboxID:                        {},
		apiresource.SampleMessagingGroupID:                    {},
		apiresource.SampleMessagingGroupMemberID:              {},
		apiresource.SampleSupportRouteID:                      {},
		apiresource.SampleEnterpriseInquiryID:                 {},
		apiresource.SampleEmailContactID:                      {},
		apiresource.SampleGeolocationID:                       {},
		apiresource.SampleHubspotCompanyReviewID:              {},
		apiresource.SampleHubspotSyncJobID:                    {},
		apiresource.SampleHubspotSyncRecordID:                 {},
		apiresource.SampleInvoiceID:                           {},
		apiresource.SampleInvoiceLineID:                       {},
		apiresource.SampleItemCategoryID:                      {},
		apiresource.SampleItemID:                              {},
		apiresource.SampleInventoryChangeLogID:                {},
		apiresource.SampleJobID:                               {},
		apiresource.SampleLocationChildID:                     {},
		apiresource.SampleLocationTypeID:                      {},
		apiresource.SampleLotID:                               {},
		apiresource.SampleMachineID:                           {},
		apiresource.SampleMaterialID:                          {},
		apiresource.SampleOpenCreditEntryID:                   {},
		apiresource.SampleOrderDiscountID:                     {},
		apiresource.SamplePartID:                              {},
		apiresource.SamplePlanTypeIDFree:                      {},
		apiresource.SamplePlanTypeIDStarter:                   {},
		apiresource.SamplePlanTypeIDPro:                       {},
		apiresource.SamplePaymentTermID:                       {},
		apiresource.SamplePermissionGroupID:                   {},
		apiresource.SamplePermissionID:                        {},
		apiresource.SamplePickID:                              {},
		apiresource.SamplePickLineID:                          {},
		apiresource.SamplePriorityID:                          {},
		apiresource.SampleProductID:                           {},
		apiresource.SampleProductLineID:                       {},
		apiresource.SampleProductTypeID:                       {},
		apiresource.SampleProductionID:                        {},
		apiresource.SampleProductionRunID:                     {},
		apiresource.SampleProductionStepID:                    {},
		apiresource.SamplePropertyID:                          {},
		apiresource.SamplePurchaseOrderID:                     {},
		apiresource.SamplePurchaseOrderLineID:                 {},
		apiresource.SampleQuantityID:                          {},
		apiresource.SampleRateID:                              {},
		apiresource.SampleReceivingOrderID:                    {},
		apiresource.SampleReceivingOrderLineID:                {},
		apiresource.SampleRegistrationFlowID:                  {},
		apiresource.SampleRegistrationFlowOptionID:            {},
		apiresource.SampleRequestLogID:                        {},
		apiresource.SampleRoleID:                              {},
		apiresource.SampleRolePermissionID:                    {},
		apiresource.SampleSalesOrderID:                        {},
		apiresource.SampleSalesOrderLineID:                    {},
		apiresource.SampleSalesOrderLineID2:                   {},
		apiresource.SampleSalesOrderStatusID:                  {},
		apiresource.SampleSalesTargetID:                       {},
		apiresource.SampleSandboxID:                           {},
		apiresource.SampleScanningStationID:                   {},
		apiresource.SampleServiceLevelID:                      {},
		apiresource.SampleSettlementID:                        {},
		apiresource.SampleSettlementSummaryID:                 {},
		apiresource.SampleLocationID:                          {},
		apiresource.SampleShippingCaseID:                      {},
		apiresource.SampleShipmentID:                          {},
		apiresource.SampleShipmentLineID:                      {},
		apiresource.SampleShippingTermID:                      {},
		apiresource.SampleSupplierID:                          {},
		apiresource.SampleSupplierMaterialID:                  {},
		apiresource.SampleSysPropertyID:                       {},
		apiresource.SampleSysPropertyTypeID:                   {},
		apiresource.SampleTerritoryID:                         {},
		apiresource.SampleToolGroupID:                         {},
		apiresource.SampleTransactionDetailID:                 {},
		apiresource.SampleTransactionMethodID:                 {},
		apiresource.SampleTransactionTypeID:                   {},
		apiresource.SampleUnitGroupID:                         {},
		apiresource.SampleUnitGroupUnitID:                     {},
		apiresource.SampleUnitID:                              {},
		apiresource.SampleUserID:                              {},
		apiresource.SampleVolumeDiscountID:                    {},
		apiresource.SampleVolumeDiscountTierID:                {},
		apiresource.SampleEDIRunID:                            {},
		apiresource.SampleRegistrationSessionID:               {},
		apiresource.SamplePortalRegistrationSessionID:         {},
		apiresource.SampleCheckoutSessionID:                   {},
		apiresource.SampleAddressID:                           {},
	}
	return m
}
