package constants

// ObjectType is a string that indicates what type of object a given object is.
type ObjectType string

const (
	// ObjectTypeAccount indicates that the object is an account.
	ObjectTypeAccount ObjectType = "account"
	// ObjectTypeActor indicates that the object is an actor.
	ObjectTypeActor ObjectType = "actor"
	// ObjectTypeEntity indicates that the object is a polymorphic entity reference.
	ObjectTypeEntity ObjectType = "entity"
	// ObjectTypeRecord indicates that the object is a lightweight reference to a business record.
	ObjectTypeRecord ObjectType = "record"
	// ObjectTypeFreight indicates that the object is a freight (carrier selection and billing) sub-resource.
	ObjectTypeFreight ObjectType = "freight"
	// ObjectTypeSalesOrderTotals indicates that the object is a sales order totals sub-resource.
	ObjectTypeSalesOrderTotals ObjectType = "sales_order_totals"
	// ObjectTypeSalesOrderStageTotal indicates that the object is a sales order per-stage total (amount + completion) sub-resource.
	ObjectTypeSalesOrderStageTotal ObjectType = "sales_order_stage_total"
	// ObjectTypeSalesOrderRelated indicates that the object groups records related to a sales order.
	ObjectTypeSalesOrderRelated ObjectType = "sales_order_related"
	// ObjectTypeOrderContact indicates that the object groups a sales order's email recipients by notification purpose.
	ObjectTypeOrderContact ObjectType = "order_contact"
	// ObjectTypeUser indicates that the object is a user.
	ObjectTypeUser ObjectType = "user"
	// ObjectTypeAddress indicates that the object is an address.
	ObjectTypeAddress ObjectType = "address"
	// ObjectTypeAPIKey indicates that the object is an API key.
	ObjectTypeAPIKey ObjectType = "api_key"
	// ObjectTypeCreatedAPIKey indicates a one-time API key creation response including the secret.
	ObjectTypeCreatedAPIKey ObjectType = "created_api_key"
	// ObjectTypeRefreshToken indicates that the object is a refresh token.
	ObjectTypeRefreshToken ObjectType = "refresh_token"
	// ObjectTypeList indicates that the object is a list.
	ObjectTypeList ObjectType = "list"
	// ObjectTypeSandbox indicates that the object is a sandbox.
	ObjectTypeSandbox ObjectType = "sandbox"
	// ObjectTypeRegistrationSession indicates that the object is a registration session.
	ObjectTypeRegistrationSession ObjectType = "registration_session"
	// ObjectTypePricingPlan indicates that the object is a pricing plan.
	ObjectTypePricingPlan ObjectType = "pricing_plan"
	// ObjectTypeAccountPlan indicates that the object is a resolved account plan.
	ObjectTypeAccountPlan ObjectType = "account_plan"
	// ObjectTypePlanChange indicates that the object is a plan change.
	ObjectTypePlanChange ObjectType = "plan_change"
	// ObjectTypeEnterpriseInquiry indicates that the object is an enterprise inquiry.
	ObjectTypeEnterpriseInquiry ObjectType = "enterprise_inquiry"
	// ObjectTypeRequestLog indicates that the object is a request log.
	ObjectTypeRequestLog ObjectType = "request_log"
	// ObjectTypeAuditEvent indicates that the object is an audit event record.
	ObjectTypeAuditEvent ObjectType = "audit_event"
	// ObjectTypeAuditFieldChange indicates that the object is an audit field change.
	ObjectTypeAuditFieldChange ObjectType = "audit_field_change"
	// ObjectTypeRole indicates that the object is a role.
	ObjectTypeRole ObjectType = "role"
	// ObjectTypeUnit indicates that the object is a unit.
	ObjectTypeUnit ObjectType = "unit"
	// ObjectTypeAccountAffiliation indicates that the object is an account affiliation.
	ObjectTypeAccountAffiliation ObjectType = "account_affiliation"
	// ObjectTypeAgentDefinition indicates that the object is an agent definition.
	ObjectTypeAgentDefinition ObjectType = "agent_definition"
	// ObjectTypeAvailableTool indicates that the object is an available tool.
	ObjectTypeAvailableTool ObjectType = "available_tool"
	// ObjectTypeAgentDefinitionTool indicates that the object is an agent definition tool.
	ObjectTypeAgentDefinitionTool ObjectType = "agent_definition_tool"
	// ObjectTypeAgentAccountStatus indicates that the object is an agent account status.
	ObjectTypeAgentAccountStatus ObjectType = "agent_account_status"
	// ObjectTypeAgentRun indicates that the object is an agent run.
	ObjectTypeAgentRun ObjectType = "agent_run"
	// ObjectTypeAgentAction indicates that the object is an agent action.
	ObjectTypeAgentAction ObjectType = "agent_action"
	// ObjectTypeAgentRunStep indicates that the object is an agent run step.
	ObjectTypeAgentRunStep ObjectType = "agent_run_step"
	// ObjectTypeAgentTokenUsage indicates that the object is an agent token usage record.
	ObjectTypeAgentTokenUsage ObjectType = "agent_token_usage"
	// ObjectTypeAgentMemory indicates that the object is an agent memory.
	ObjectTypeAgentMemory ObjectType = "agent_memory"
	// ObjectTypeNotification indicates that the object is an in-app notification.
	ObjectTypeNotification ObjectType = "notification"
	// ObjectTypeNotificationUnreadCount indicates that the object is an unread-count summary.
	ObjectTypeNotificationUnreadCount ObjectType = "notification_unread_count"
	// ObjectTypeNotificationSendResult indicates that the object is a notification-send acknowledgement.
	ObjectTypeNotificationSendResult ObjectType = "notification_send_result"
	// ObjectTypeNotificationUnreadSummary indicates that the object is a cross-account unread summary.
	ObjectTypeNotificationUnreadSummary ObjectType = "notification_unread_summary"
	// ObjectTypeAnnouncement indicates that the object is a broadcast announcement.
	ObjectTypeAnnouncement ObjectType = "announcement"
	// ObjectTypeConversation indicates that the object is a conversation.
	ObjectTypeConversation ObjectType = "conversation"
	// ObjectTypeSupportCase indicates that the object is a customer-facing support case (an audience=customer conversation). Used to route notification links to the support inbox rather than team messages.
	ObjectTypeSupportCase ObjectType = "support_case"
	// ObjectTypeConversationParticipant indicates that the object is a conversation participant.
	ObjectTypeConversationParticipant ObjectType = "conversation_participant"
	// ObjectTypeReadCursor indicates that the object is a participant's read cursor (read receipts).
	ObjectTypeReadCursor ObjectType = "read_cursor"
	// ObjectTypeChatMessage indicates that the object is a chat message within a conversation.
	ObjectTypeChatMessage ObjectType = "chat_message"
	// ObjectTypeNotificationUnreadSummaryAccount indicates that the object is one account's unread tally in a cross-account summary.
	ObjectTypeNotificationUnreadSummaryAccount ObjectType = "notification_unread_summary_account"
	// ObjectTypeMessagingBlock indicates that the object is a 1:1 messaging block.
	ObjectTypeMessagingBlock ObjectType = "messaging_block"
	// ObjectTypeNotificationPreference indicates that the object is a per-user notification preference.
	ObjectTypeNotificationPreference ObjectType = "notification_preference"
	// ObjectTypeMessageAttachment indicates that the object is a message attachment.
	ObjectTypeMessageAttachment ObjectType = "message_attachment"
	// ObjectTypeAttachmentUploadTarget indicates that the object is a presigned attachment upload target.
	ObjectTypeAttachmentUploadTarget ObjectType = "attachment_upload_target"
	// ObjectTypeScheduledMessage indicates that the object is a scheduled message.
	ObjectTypeScheduledMessage ObjectType = "scheduled_message"
	// ObjectTypeMessagingContact indicates that the object is a messageable directory contact.
	ObjectTypeMessagingContact ObjectType = "messaging_contact"
	// ObjectTypeMessageReport indicates that the object is an abuse report for a message/conversation.
	ObjectTypeMessageReport ObjectType = "message_report"
	// ObjectTypeToolGroup indicates that the object is a tool group.
	ObjectTypeToolGroup ObjectType = "tool_group"
	// ObjectTypeModel indicates that the object is an LLM model available to agents.
	ObjectTypeModel ObjectType = "model"
	// ObjectTypePaymentTerm indicates that the object is a payment term.
	ObjectTypePaymentTerm ObjectType = "payment_term"
	// ObjectTypeShippingTerm indicates that the object is a shipping term.
	ObjectTypeShippingTerm ObjectType = "shipping_term"
	// ObjectTypeQuantity indicates that the object is a quantity.
	ObjectTypeQuantity ObjectType = "quantity"
	// ObjectTypeAccountGroup indicates that the object is an account group.
	ObjectTypeAccountGroup ObjectType = "account_group"
	// ObjectTypeSupportRoute indicates that the object is a support route: the group conversation handling a relationship's inbound support.
	ObjectTypeSupportRoute ObjectType = "support_route"
	// ObjectTypeReplyDraft indicates that the object is a customer-reply draft on an external case.
	ObjectTypeReplyDraft ObjectType = "reply_draft"
	// ObjectTypeConversationLink indicates that the object is a business-record link on a conversation.
	ObjectTypeConversationLink ObjectType = "conversation_link"
	// ObjectTypeMessagingGroup indicates that the object is a reusable messaging roster (a named member set that seeds conversations).
	ObjectTypeMessagingGroup ObjectType = "messaging_group"
	// ObjectTypeMessagingGroupMember indicates that the object is a member of a reusable messaging roster.
	ObjectTypeMessagingGroupMember ObjectType = "messaging_group_member"
	// ObjectTypeSupportAvailability indicates that the object reports whether a customer can contact support.
	ObjectTypeSupportAvailability ObjectType = "support_availability"
	// ObjectTypeAccountStatus indicates that the object is an account status.
	ObjectTypeAccountStatus ObjectType = "account_status"
	// ObjectTypeGeolocation indicates that the object is a geolocation.
	ObjectTypeGeolocation ObjectType = "geolocation"
	// ObjectTypeAccountUser indicates that the object is an account user.
	ObjectTypeAccountUser ObjectType = "account_user"
	// ObjectTypeDepartment indicates that the object is a department.
	ObjectTypeDepartment ObjectType = "department"
	// ObjectTypeAccountIntegration indicates that the object is an account integration.
	ObjectTypeAccountIntegration ObjectType = "account_integration"
	// ObjectTypeHubspotSyncJob indicates that the object is a HubSpot backfill sync job.
	ObjectTypeHubspotSyncJob ObjectType = "hubspot_sync_job"
	// ObjectTypeHubspotSyncReport indicates that the object is the dry-run report embedded in a HubSpot sync job.
	ObjectTypeHubspotSyncReport ObjectType = "hubspot_sync_report"
	// ObjectTypeHubspotCompanyReview indicates that the object is a HubSpot company-match review.
	ObjectTypeHubspotCompanyReview ObjectType = "hubspot_company_review"
	// ObjectTypeHubspotCompanyCandidate indicates that the object is a candidate HubSpot company match within a review.
	ObjectTypeHubspotCompanyCandidate ObjectType = "hubspot_company_candidate"
	// ObjectTypeHubspotSyncRecord indicates that the object is a mapping from an Augno record to its HubSpot counterpart.
	ObjectTypeHubspotSyncRecord ObjectType = "hubspot_sync_record"
	// ObjectTypeAccountPrice indicates that the object is an account price.
	ObjectTypeAccountPrice ObjectType = "account_price"
	// ObjectTypeProductLine indicates that the object is a product line.
	ObjectTypeProductLine ObjectType = "product_line"
	// ObjectTypeItemCategory indicates that the object is an item category.
	ObjectTypeItemCategory ObjectType = "item_category"
	// ObjectTypeAttribute indicates that the object is an attribute.
	ObjectTypeAttribute ObjectType = "attribute"
	// ObjectTypeRate indicates that the object is a rate.
	ObjectTypeRate ObjectType = "rate"
	// ObjectTypeAccountGroupProductLineAccess indicates that the object is an account group product line access.
	ObjectTypeAccountGroupProductLineAccess ObjectType = "account_group_product_line_access"
	// ObjectTypeSalesTarget indicates that the object is a sales target.
	ObjectTypeSalesTarget ObjectType = "sales_target"
	// ObjectTypeAdjustmentType indicates that the object is an adjustment type.
	ObjectTypeAdjustmentType ObjectType = "adjustment_type"
	// ObjectTypeAccountBranding indicates that the object is an account branding record.
	ObjectTypeAccountBranding ObjectType = "account_branding"
	// ObjectTypeAccountPortal indicates that the object is an account portal record.
	ObjectTypeAccountPortal ObjectType = "account_portal"
	// ObjectTypeAccountLogoURL indicates that the object is an account logo URL response.
	ObjectTypeAccountLogoURL ObjectType = "account_logo_url"
	// ObjectTypeAccountFaviconURL indicates that the object is an account favicon URL response.
	ObjectTypeAccountFaviconURL ObjectType = "account_favicon_url"
	// ObjectTypePublicAccount indicates that the object is a public account record.
	ObjectTypePublicAccount ObjectType = "public_account"
	// ObjectTypePortalProfile indicates that the object is an authenticated seller portal profile (identity + letterhead address).
	ObjectTypePortalProfile ObjectType = "portal_profile"
	// ObjectTypePortalRegistrationSession indicates that the object is a buyer's customer-portal registration session.
	ObjectTypePortalRegistrationSession ObjectType = "portal_registration_session"
	// ObjectTypePortalRegistrationSessionData indicates that the object is the scratch form data of a portal registration session.
	ObjectTypePortalRegistrationSessionData ObjectType = "portal_registration_session_data"
	// ObjectTypeProperty indicates that the object is a property.
	ObjectTypeProperty ObjectType = "property"
	// ObjectTypeCarrier indicates that the object is a carrier.
	ObjectTypeCarrier ObjectType = "carrier"
	// ObjectTypeServiceLevel indicates that the object is a service level.
	ObjectTypeServiceLevel ObjectType = "service_level"
	// ObjectTypeItem indicates that the object is an item.
	ObjectTypeItem ObjectType = "item"
	// ObjectTypeItemLotDefault indicates that the object is the lot an item is made in.
	ObjectTypeItemLotDefault ObjectType = "item_lot_default"
	// ObjectTypeItemInventory indicates that the object is an item's inventory data.
	ObjectTypeItemInventory ObjectType = "item_inventory"
	// ObjectTypeProduct indicates that the object is a product.
	ObjectTypeProduct ObjectType = "product"
	// ObjectTypeBatch indicates that the object is a batch.
	ObjectTypeBatch ObjectType = "batch"
	// ObjectTypeBatchFlowNode indicates that the object is a batch flow node.
	ObjectTypeBatchFlowNode ObjectType = "batch_flow_node"
	// ObjectTypeScanningConsumption indicates that the object is a scanning consumption.
	ObjectTypeScanningConsumption ObjectType = "scanning_consumption"
	// ObjectTypeOpenBatchSummary indicates that the object is an open batch summary.
	ObjectTypeOpenBatchSummary ObjectType = "open_batch_summary"
	// ObjectTypeScanningProductionStepInfo indicates that the object is a scanning production step info.
	ObjectTypeScanningProductionStepInfo ObjectType = "scanning_production_step_info"
	// ObjectTypeScanningStation indicates that the object is a scanning station.
	ObjectTypeScanningStation ObjectType = "scanning_station"
	// ObjectTypeProductionStep indicates that the object is a production step.
	ObjectTypeProductionStep ObjectType = "production_step"
	// ObjectTypeProductionRun indicates that the object is a production run.
	ObjectTypeProductionRun ObjectType = "production_run"
	// ObjectTypeMachine indicates that the object is a machine.
	ObjectTypeMachine ObjectType = "machine"
	// ObjectTypeMachineStatus indicates that the object is a machine's current work status.
	ObjectTypeMachineStatus ObjectType = "machine_status"
	// ObjectTypeMachineDowntimeEvent indicates that the object is a machine downtime event.
	ObjectTypeMachineDowntimeEvent ObjectType = "machine_downtime_event"

	// ObjectTypeDemandOverride indicates that the object is a demand override.
	ObjectTypeDemandOverride ObjectType = "demand_override"

	// ObjectTypeDemandOverrideType indicates that the object is a demand override type.
	ObjectTypeDemandOverrideType ObjectType = "demand_override_type"
	// ObjectTypeMachineDowntimeReason indicates that the object is a machine downtime reason.
	ObjectTypeMachineDowntimeReason ObjectType = "machine_downtime_reason"
	// ObjectTypeProductionSchedulePreview indicates that the object is a production schedule preview.
	ObjectTypeProductionSchedulePreview ObjectType = "production_schedule_preview"
	// ObjectTypeProductionScheduleRegeneratePreview indicates that the object is a production schedule regenerate preview.
	ObjectTypeProductionScheduleRegeneratePreview ObjectType = "production_schedule_regenerate_preview"
	// ObjectTypeProductionSchedule indicates that the object is a production schedule.
	ObjectTypeProductionSchedule ObjectType = "production_schedule"
	// ObjectTypeProductionScheduleLine indicates that the object is a production schedule line.
	ObjectTypeProductionScheduleLine ObjectType = "production_schedule_line"

	// ObjectTypeProductionScheduleDeviation indicates that the object is a production schedule deviation.
	ObjectTypeProductionScheduleDeviation ObjectType = "production_schedule_deviation"

	// ObjectTypeProductionScheduleDerivedLine indicates that the object is a derived production schedule line.
	ObjectTypeProductionScheduleDerivedLine ObjectType = "production_schedule_derived_line"

	// ObjectTypeProductionScheduleSettings indicates that the object is the account's production schedule settings.
	ObjectTypeProductionScheduleSettings ObjectType = "production_schedule_settings"

	// ObjectTypeProductionScheduleResourceSetting indicates that the object is a per-resource scheduling override.
	ObjectTypeProductionScheduleResourceSetting ObjectType = "production_schedule_resource_setting"

	// ObjectTypeScheduleDeviationType indicates that the object is a schedule deviation type.
	ObjectTypeScheduleDeviationType ObjectType = "schedule_deviation_type"
	// ObjectTypeProductionScheduleFinishedPolicy indicates that the object is a production schedule finished-goods policy.
	ObjectTypeProductionScheduleFinishedPolicy ObjectType = "production_schedule_finished_policy"
	// ObjectTypeProductionScheduleWeekRelease indicates that the object is the production run created from one week of a schedule.
	ObjectTypeProductionScheduleWeekRelease ObjectType = "production_schedule_week_release"
	// ObjectTypeProductionScheduleWeekReleasePreview indicates that the object is a preview of releasing one week of a schedule.
	ObjectTypeProductionScheduleWeekReleasePreview ObjectType = "production_schedule_week_release_preview"
	// ObjectTypeProductionScheduleItemPolicy indicates that the object is a production schedule item policy.
	ObjectTypeProductionScheduleItemPolicy ObjectType = "production_schedule_item_policy"
	// ObjectTypeChildAccount indicates that the object is a child account.
	ObjectTypeChildAccount ObjectType = "child_account"
	// ObjectTypeUnitGroup indicates that the object is a unit group.
	ObjectTypeUnitGroup ObjectType = "unit_group"
	// ObjectTypeUnitGroupUnit indicates that the object is a unit group unit conversion.
	ObjectTypeUnitGroupUnit ObjectType = "unit_group_unit"
	// ObjectTypeConsumption indicates that the object is a consumption.
	ObjectTypeConsumption ObjectType = "consumption"
	// ObjectTypeCustomerProductLineAccess indicates that the object is a customer product line access.
	ObjectTypeCustomerProductLineAccess ObjectType = "customer_product_line_access"
	// ObjectTypeCustomer indicates that the object is a customer.
	ObjectTypeCustomer ObjectType = "customer"
	// ObjectTypeFrequentlyOrderedProduct indicates that the object is a frequently ordered product.
	ObjectTypeFrequentlyOrderedProduct ObjectType = "frequently_ordered_product"
	// ObjectTypePriority indicates that the object is a priority.
	ObjectTypePriority ObjectType = "priority"
	// ObjectTypeDelivery indicates that the object is a delivery.
	ObjectTypeDelivery ObjectType = "delivery"
	// ObjectTypeDeliveryLine indicates that the object is a delivery line.
	ObjectTypeDeliveryLine ObjectType = "delivery_line"
	// ObjectTypeSalesOrder indicates that the object is a sales order.
	ObjectTypeSalesOrder ObjectType = "sales_order"
	// ObjectTypeLocation indicates that the object is a location.
	ObjectTypeLocation ObjectType = "location"
	// ObjectTypeLocationType indicates that the object is a location type.
	ObjectTypeLocationType ObjectType = "location_type"
	// ObjectTypeLot indicates that the object is a lot.
	ObjectTypeLot ObjectType = "lot"
	// ObjectTypeEmailLog indicates that the object is an email log.
	ObjectTypeEmailLog ObjectType = "email_log"
	// ObjectTypeEmailDomain indicates that the object is a customer-owned sending/receiving domain registered with the email bridge.
	ObjectTypeEmailDomain ObjectType = "email_domain"
	// ObjectTypeEmailInbox indicates that the object is a routable email inbox bound to chat conversations.
	ObjectTypeEmailInbox ObjectType = "email_inbox"
	// ObjectTypePortalDomain indicates that the object is a customer-supplied custom domain serving the account's customer portal.
	ObjectTypePortalDomain ObjectType = "portal_domain"
	// ObjectTypeDNSRecord indicates that the object is a DNS record the customer must publish for a portal domain.
	ObjectTypeDNSRecord ObjectType = "dns_record"
	// ObjectTypeInventoryChangeLog indicates that the object is an inventory change log.
	ObjectTypeInventoryChangeLog ObjectType = "inventory_change_log"
	// ObjectTypeInvoice indicates that the object is an invoice.
	ObjectTypeInvoice ObjectType = "invoice"
	// ObjectTypeInvoiceSummary indicates that the object is an invoice summary.
	ObjectTypeInvoiceSummary ObjectType = "invoice_summary"
	// ObjectTypeInvoiceLine indicates that the object is an invoice line.
	ObjectTypeInvoiceLine ObjectType = "invoice_line"
	// ObjectTypeInvoiceAllocation indicates that the object is an invoice allocation.
	ObjectTypeInvoiceAllocation ObjectType = "invoice_allocation"
	// ObjectTypeInvoiceForPayment indicates that the object is an invoice for payment.
	ObjectTypeInvoiceForPayment ObjectType = "invoice_for_payment"
	// ObjectTypeShipment indicates that the object is a shipment.
	ObjectTypeShipment ObjectType = "shipment"
	// ObjectTypeShipmentSummary indicates that the object is a shipment summary.
	ObjectTypeShipmentSummary ObjectType = "shipment_summary"
	// ObjectTypeShipmentLine indicates that the object is a shipment line.
	ObjectTypeShipmentLine ObjectType = "shipment_line"
	// ObjectTypeShippingCase indicates that the object is a shipping case.
	ObjectTypeShippingCase ObjectType = "shipping_case"
	// ObjectTypeShippingCaseLabelURL indicates that the object is a shipping case label URL response.
	ObjectTypeShippingCaseLabelURL ObjectType = "shipping_case_label_url"
	// ObjectTypeSettlement indicates that the object is a settlement.
	ObjectTypeSettlement ObjectType = "settlement"
	// ObjectTypeSettlementSummary indicates that the object is a settlement summary.
	ObjectTypeSettlementSummary ObjectType = "settlement_summary"
	// ObjectTypeRolePermission indicates that the object is a role permission.
	ObjectTypeRolePermission ObjectType = "role_permission"
	// ObjectTypeRegistrationFlow indicates that the object is a registration flow.
	ObjectTypeRegistrationFlow ObjectType = "registration_flow"
	// ObjectTypeRegistrationFlowOption indicates that the object is a registration flow option.
	ObjectTypeRegistrationFlowOption ObjectType = "registration_flow_option"
	// ObjectTypeTransaction indicates that the object is a transaction.
	ObjectTypeTransaction ObjectType = "transaction"
	// ObjectTypeTransactionSummary indicates that the object is a transaction summary.
	ObjectTypeTransactionSummary ObjectType = "transaction_summary"
	// ObjectTypeTransactionMethod indicates that the object is a transaction method.
	ObjectTypeTransactionMethod ObjectType = "transaction_method"
	// ObjectTypeTransactionType indicates that the object is a transaction type.
	ObjectTypeTransactionType ObjectType = "transaction_type"
	// ObjectTypeTransactionAllocation indicates that the object is a transaction allocation.
	ObjectTypeTransactionAllocation ObjectType = "transaction_allocation"
	// ObjectTypeUsageItem indicates that the object is a usage item.
	ObjectTypeUsageItem ObjectType = "usage_item"
	// ObjectTypeAccountUsageResponse indicates that the object is an account usage response.
	ObjectTypeAccountUsageResponse ObjectType = "account_usage_response"
	// ObjectTypeSubscriptionInfo indicates that the object is a subscription info.
	ObjectTypeSubscriptionInfo ObjectType = "subscription_info"
	// ObjectTypeBillingPortalSessionResponse indicates that the object is a billing portal session response.
	ObjectTypeBillingPortalSessionResponse ObjectType = "billing_portal_session_response"
	// ObjectTypeSwitchPlanResponse indicates that the object is a switch plan response.
	ObjectTypeSwitchPlanResponse ObjectType = "switch_plan_response"
	// ObjectTypeEnsureBillingCustomerResponse indicates that the object is an ensure billing customer response.
	ObjectTypeEnsureBillingCustomerResponse ObjectType = "ensure_billing_customer_response"
	// ObjectTypeSpendingCapResponse indicates that the object is a spending cap response.
	ObjectTypeSpendingCapResponse ObjectType = "spending_cap_response"
	// ObjectTypeAgentSpendInfo indicates that the object is an agent spend info.
	ObjectTypeAgentSpendInfo ObjectType = "agent_spend_info"
	// ObjectTypeWebhookResponse indicates that the object is a webhook response.
	ObjectTypeWebhookResponse ObjectType = "webhook_response"
	// ObjectTypeAddressSuggestion indicates that the object is an address suggestion.
	ObjectTypeAddressSuggestion ObjectType = "address_suggestion"
	// ObjectTypeAddressComponents indicates that the object is an address components record.
	ObjectTypeAddressComponents ObjectType = "address_components"
	// ObjectTypeAddressDetailsResult indicates that the object is an address details result.
	ObjectTypeAddressDetailsResult ObjectType = "address_details_result"
	// ObjectTypeContactMatch indicates that the object is a contact match (an account user found by email on a related account).
	ObjectTypeContactMatch ObjectType = "contact_match"
	// ObjectTypeValidatedAddress indicates that the object is a validated address.
	ObjectTypeValidatedAddress ObjectType = "validated_address"
	// ObjectTypePlanLimit indicates that the object is a plan limit.
	ObjectTypePlanLimit ObjectType = "plan_limit"
	// ObjectTypePlanChangeProration indicates that the object is a plan change proration.
	ObjectTypePlanChangeProration ObjectType = "plan_change_proration"
	// ObjectTypePlanChangeLineItem indicates that the object is a plan change line item.
	ObjectTypePlanChangeLineItem ObjectType = "plan_change_line_item"
	// ObjectTypeSetupBillingResponse indicates that the object is a setup billing response.
	ObjectTypeSetupBillingResponse ObjectType = "setup_billing_response"
	// ObjectTypeConfirmPaymentResponse indicates that the object is a confirm payment response.
	ObjectTypeConfirmPaymentResponse ObjectType = "confirm_payment_response"
	// ObjectTypeOAuthResponse indicates that the object is an OAuth response.
	ObjectTypeOAuthResponse ObjectType = "oauth_response"
	// ObjectTypeOAuthStatusResponse indicates that the object is an OAuth status response.
	ObjectTypeOAuthStatusResponse ObjectType = "oauth_status_response"
	// ObjectTypeStripePublishableKey indicates that the object is a Stripe publishable key.
	ObjectTypeStripePublishableKey ObjectType = "stripe_publishable_key"
	// ObjectTypeStripeStatus indicates that the object is a Stripe status.
	ObjectTypeStripeStatus ObjectType = "stripe_status"
	// ObjectTypeHealthcheck indicates that the object is a healthcheck.
	ObjectTypeHealthcheck ObjectType = "healthcheck"
	// ObjectTypeAgentDefinitionConfig indicates that the object is an agent definition config.
	ObjectTypeAgentDefinitionConfig ObjectType = "agent_definition_config"
	// ObjectTypeTriggerConfig indicates that the object is a trigger config.
	ObjectTypeTriggerConfig ObjectType = "trigger_config"
	// ObjectTypeCustomerContactInfo indicates that the object is customer contact info.
	ObjectTypeCustomerContactInfo ObjectType = "customer_contact_info"
	// ObjectTypeCustomerFreightPreferences indicates that the object is customer freight preferences.
	ObjectTypeCustomerFreightPreferences ObjectType = "customer_freight_preferences"
	// ObjectTypeCustomerDefaults indicates that the object is customer defaults.
	ObjectTypeCustomerDefaults ObjectType = "customer_defaults"
	// ObjectTypeCustomerNotificationPreferences indicates that the object is customer notification preferences.
	ObjectTypeCustomerNotificationPreferences ObjectType = "customer_notification_preferences"
	// ObjectTypeOrderNotificationRecipient indicates that the object is a default order-notification recipient for a customer.
	ObjectTypeOrderNotificationRecipient ObjectType = "order_notification_recipient"
	// ObjectTypeOrderDiscount indicates that the object is an order discount.
	ObjectTypeOrderDiscount ObjectType = "order_discount"
	// ObjectTypeSalesOrderLine indicates that the object is a sales order line.
	ObjectTypeSalesOrderLine ObjectType = "sales_order_line"
	// ObjectTypeSalesOrderType indicates that the object is a sales order type.
	ObjectTypeSalesOrderType ObjectType = "sales_order_type"
	// ObjectTypeSalesOrderStatus indicates that the object is a sales order status.
	ObjectTypeSalesOrderStatus ObjectType = "sales_order_status"
	// ObjectTypeMaterial indicates that the object is a material.
	ObjectTypeMaterial ObjectType = "material"
	// ObjectTypeSupplierMaterial indicates that the object is a supplier material.
	ObjectTypeSupplierMaterial ObjectType = "supplier_material"
	// ObjectTypePart indicates that the object is a part.
	ObjectTypePart ObjectType = "part"
	// ObjectTypePermissionGroup indicates that the object is a permission group.
	ObjectTypePermissionGroup ObjectType = "permission_group"
	// ObjectTypePermission indicates that the object is a permission.
	ObjectTypePermission ObjectType = "permission"
	// ObjectTypePick indicates that the object is a pick.
	ObjectTypePick ObjectType = "pick"
	// ObjectTypePickLine indicates that the object is a pick line.
	ObjectTypePickLine ObjectType = "pick_line"
	// ObjectTypeProductType indicates that the object is a product type.
	ObjectTypeProductType ObjectType = "product_type"
	// ObjectTypeProduction indicates that the object is a production output.
	ObjectTypeProduction ObjectType = "production"
	// ObjectTypeProductionFlow indicates that the object is a production flow.
	ObjectTypeProductionFlow ObjectType = "production_flow"
	// ObjectTypeMap indicates that the object is a map.
	ObjectTypeMap ObjectType = "map"
	// ObjectTypePurchaseOrder indicates that the object is a purchase order.
	ObjectTypePurchaseOrder ObjectType = "purchase_order"
	// ObjectTypePurchaseOrderLine indicates that the object is a purchase order line.
	ObjectTypePurchaseOrderLine ObjectType = "purchase_order_line"
	// ObjectTypeSupplier indicates that the object is a supplier.
	ObjectTypeSupplier ObjectType = "supplier"
	// ObjectTypeSupplierSummary indicates that the object is a supplier summary.
	ObjectTypeSupplierSummary ObjectType = "supplier_summary"
	// ObjectTypeReceivableEntry indicates that the object is a receivable entry.
	ObjectTypeReceivableEntry ObjectType = "receivable_entry"
	// ObjectTypeReceivingOrder indicates that the object is a receiving order.
	ObjectTypeReceivingOrder ObjectType = "receiving_order"
	// ObjectTypeReceivingOrderLine indicates that the object is a receiving order line.
	ObjectTypeReceivingOrderLine ObjectType = "receiving_order_line"
	// ObjectTypeEmailContact indicates that the object is an email contact.
	ObjectTypeEmailContact ObjectType = "email_contact"
	// ObjectTypeAllocationEntry indicates that the object is an allocation entry.
	ObjectTypeAllocationEntry ObjectType = "allocation_entry"
	// ObjectTypeOpenCreditEntry indicates that the object is an open credit entry.
	ObjectTypeOpenCreditEntry ObjectType = "open_credit_entry" // #nosec G101 -- constant name, not a credential
	// ObjectTypeVolumeDiscount indicates that the object is a volume discount.
	ObjectTypeVolumeDiscount ObjectType = "volume_discount"
	// ObjectTypeVolumeDiscountTier indicates that the object is a volume discount tier.
	ObjectTypeVolumeDiscountTier ObjectType = "volume_discount_tier"
	// ObjectTypeAnalyzeDeliveriesResponse indicates that the object is an analyze deliveries response.
	ObjectTypeAnalyzeDeliveriesResponse ObjectType = "analyze_deliveries_response"
	// ObjectTypeAnalyzeManufacturingResponse indicates that the object is an analyze manufacturing response.
	ObjectTypeAnalyzeManufacturingResponse ObjectType = "analyze_manufacturing_response"
	// ObjectTypeAnalyzeManufacturingBatchResponse indicates that the object is an analyze manufacturing batch response.
	ObjectTypeAnalyzeManufacturingBatchResponse ObjectType = "analyze_manufacturing_batch_response"
	// ObjectTypeAnalyzeQuarterlyOrdersResponse indicates that the object is an analyze quarterly orders response.
	ObjectTypeAnalyzeQuarterlyOrdersResponse ObjectType = "analyze_quarterly_orders_response"
	// ObjectTypeAnalyzeNewCustomersResponse indicates that the object is an analyze new customers response.
	ObjectTypeAnalyzeNewCustomersResponse ObjectType = "analyze_new_customers_response"
	// ObjectTypeAnalyzeDemandForecastResponse indicates that the object is an analyze demand forecast response.
	ObjectTypeAnalyzeDemandForecastResponse ObjectType = "analyze_demand_forecast_response"
	// ObjectTypeAnalyzeOeeResponse indicates that the object is an analyze OEE response.
	ObjectTypeAnalyzeOeeResponse ObjectType = "analyze_oee_response"
	// ObjectTypeAnalyzeOeeTrendResponse indicates that the object is an analyze OEE trend response.
	ObjectTypeAnalyzeOeeTrendResponse ObjectType = "analyze_oee_trend_response"

	// ObjectTypeAnalyzeScheduleAttainmentResponse indicates that the object is a schedule attainment analysis response.
	ObjectTypeAnalyzeScheduleAttainmentResponse ObjectType = "analyze_schedule_attainment_response"
	// ObjectTypeCatalogProductLine indicates that the object is a catalog product line.
	ObjectTypeCatalogProductLine ObjectType = "catalog_product_line"
	// ObjectTypeCatalogCategory indicates that the object is a catalog category.
	ObjectTypeCatalogCategory ObjectType = "catalog_category"
	// ObjectTypeCatalogProduct indicates that the object is a catalog product.
	ObjectTypeCatalogProduct ObjectType = "catalog_product"
	// ObjectTypeCatalogProperty indicates that the object is a catalog property.
	ObjectTypeCatalogProperty ObjectType = "catalog_property"
	// ObjectTypeCatalogAttribute indicates that the object is a catalog attribute.
	ObjectTypeCatalogAttribute ObjectType = "catalog_attribute"
	// ObjectTypeDCLocation indicates that the object is a DC location.
	ObjectTypeDCLocation ObjectType = "dc_location"
	// ObjectTypeEDIRun indicates that the object is an EDI run.
	ObjectTypeEDIRun ObjectType = "edi_run"
	// ObjectTypeInventoryItem indicates that the object is an inventory item.
	ObjectTypeInventoryItem ObjectType = "inventory_item"
	// ObjectTypeAnalyzeWeeksOfSalesResponse indicates that the object is a weeks of sales analytics response.
	ObjectTypeAnalyzeWeeksOfSalesResponse ObjectType = "analyze_weeks_of_sales_response"
	// ObjectTypeBulkReconcileItemsResponse indicates that the object is a bulk reconcile items response.
	ObjectTypeBulkReconcileItemsResponse ObjectType = "bulk_reconcile_items_response"
	// ObjectTypeSysProperty indicates that the object is a system property.
	ObjectTypeSysProperty ObjectType = "sys_property"
	// ObjectTypeSysPropertyType indicates that the object is a system property type.
	ObjectTypeSysPropertyType ObjectType = "sys_property_type"
	// ObjectTypeSysPropertyValue indicates that the object is a system property value.
	ObjectTypeSysPropertyValue ObjectType = "sys_property_value"
	// ObjectTypeTerritory indicates that the object is a territory.
	ObjectTypeTerritory ObjectType = "territory"
	// ObjectTypeTenancy indicates that the object is a tenancy.
	ObjectTypeTenancy ObjectType = "tenancy"
	// ObjectTypeCheckoutSession indicates that the object is a checkout session.
	ObjectTypeCheckoutSession ObjectType = "checkout_session"
	// ObjectTypeEstimateRateResult indicates that the object is an estimate rate result.
	ObjectTypeEstimateRateResult ObjectType = "estimate_rate_result"
	// ObjectTypeRateShopOption indicates that the object is a rate shop option.
	ObjectTypeRateShopOption ObjectType = "rate_shop_option"
	// ObjectTypeRateShopResult indicates that the object is a rate shop result.
	ObjectTypeRateShopResult ObjectType = "rate_shop_result"
	// ObjectTypeOwner indicates that the object is a resource owner.
	ObjectTypeOwner ObjectType = "owner"
	// ObjectTypeCreatedBy indicates that the object describes who created a resource.
	ObjectTypeCreatedBy ObjectType = "created_by"
	// ObjectTypeMessage indicates a simple human-readable status message payload.
	ObjectTypeMessage ObjectType = "message"
	// ObjectTypeAccountPhotoUploadResult indicates that the object is an account photo upload result.
	ObjectTypeAccountPhotoUploadResult ObjectType = "account_photo_upload_result"
	// ObjectTypeUserPhotoUploadResult indicates that the object is a user photo upload result.
	ObjectTypeUserPhotoUploadResult ObjectType = "user_photo_upload_result"
	// ObjectTypeUserPhotoURL indicates that the object is a user photo URL response.
	ObjectTypeUserPhotoURL ObjectType = "user_photo_url"
	// ObjectTypeBatchLot indicates that the object is a batch lot.
	ObjectTypeBatchLot ObjectType = "batch_lot"
	// ObjectTypeCheckDuplicateResult indicates that the object is a duplicate check result.
	ObjectTypeCheckDuplicateResult ObjectType = "check_duplicate_result"
	// ObjectTypeItemTrendPoint indicates that the object is an item trend data point.
	ObjectTypeItemTrendPoint ObjectType = "item_trend_point"
	// ObjectTypePackPickResponse indicates that the object is a pack pick response.
	ObjectTypePackPickResponse ObjectType = "pack_pick_response"
	// ObjectTypePickShipmentsResponse indicates that the object is a pick shipments response.
	ObjectTypePickShipmentsResponse ObjectType = "pick_shipments_response"
	// ObjectTypeTenancyPendingRegistration indicates that the object is a pending registration on a tenancy.
	ObjectTypeTenancyPendingRegistration ObjectType = "tenancy_pending_registration"
	// ObjectTypeInvoiceAllocationEntry indicates that the object is an invoice allocation entry on an open credit.
	ObjectTypeInvoiceAllocationEntry ObjectType = "invoice_allocation_entry"
	// ObjectTypeAllocationCustomer indicates that the object is a minimal customer sub-resource on an allocation entry.
	ObjectTypeAllocationCustomer ObjectType = "allocation_customer"
	// ObjectTypeCheckoutSalesOrderResponse indicates that the object is a sales order checkout response.
	ObjectTypeCheckoutSalesOrderResponse ObjectType = "checkout_sales_order"
	// ObjectTypeSalesOrderPriceQuote indicates that the object is a sales order line price quote.
	ObjectTypeSalesOrderPriceQuote ObjectType = "sales_order_price_quote"
	// ObjectTypeJob indicates that the object is a job.
	ObjectTypeJob ObjectType = "job"
	// ObjectTypeSalesOrderFreightQuote indicates that the object is a sales order freight (shipping) charge quote.
	ObjectTypeSalesOrderFreightQuote ObjectType = "sales_order_freight_quote"
	// ObjectTypeSalesOrderPriceQuoteLine indicates that the object is a single priced line within a sales order price quote.
	ObjectTypeSalesOrderPriceQuoteLine ObjectType = "sales_order_price_quote_line"
	// ObjectTypeSalesOrderQuoteRate indicates that the object is a per-unit rate on a sales order quote (a lightweight, unpersisted variant of a rate).
	ObjectTypeSalesOrderQuoteRate ObjectType = "sales_order_quote_rate"
	// ObjectTypePackList indicates that the object is an assembled pack-list document for a shipment.
	ObjectTypePackList ObjectType = "pack_list"
	// ObjectTypePackListParty indicates that the object is a bill-to or ship-to party on a pack list.
	ObjectTypePackListParty ObjectType = "pack_list_party"
	// ObjectTypePackListLineItem indicates that the object is a packed line item on a pack list.
	ObjectTypePackListLineItem ObjectType = "pack_list_line_item"
	// ObjectTypePackListBackOrder indicates that the object is a back-ordered line on a pack list.
	ObjectTypePackListBackOrder ObjectType = "pack_list_back_order"
	// ObjectTypePackListCase indicates that the object is a shipping case on a pack list.
	ObjectTypePackListCase ObjectType = "pack_list_case"
)

func (m ObjectType) IsValid() bool {
	switch m {
	case ObjectTypeAccount, ObjectTypeActor, ObjectTypeEntity, ObjectTypeRecord, ObjectTypeFreight, ObjectTypeSalesOrderTotals, ObjectTypeSalesOrderStageTotal, ObjectTypeSalesOrderRelated, ObjectTypeOrderContact, ObjectTypeUser, ObjectTypeAddress, ObjectTypeAPIKey, ObjectTypeCreatedAPIKey, ObjectTypeRefreshToken, ObjectTypeList, ObjectTypeSandbox, ObjectTypeRegistrationSession, ObjectTypePricingPlan, ObjectTypePlanChange, ObjectTypeEnterpriseInquiry, ObjectTypeRequestLog, ObjectTypeAuditEvent, ObjectTypeAuditFieldChange, ObjectTypeRole, ObjectTypeUnit, ObjectTypeAccountAffiliation, ObjectTypeAgentDefinition, ObjectTypeAvailableTool, ObjectTypeAgentDefinitionTool, ObjectTypeAgentAccountStatus, ObjectTypeAgentRun, ObjectTypeAgentAction, ObjectTypeAgentRunStep, ObjectTypeAgentTokenUsage, ObjectTypeAgentMemory, ObjectTypeNotification, ObjectTypeNotificationUnreadCount, ObjectTypeNotificationSendResult, ObjectTypeNotificationUnreadSummary, ObjectTypeAnnouncement, ObjectTypeConversation, ObjectTypeSupportCase, ObjectTypeConversationParticipant, ObjectTypeReadCursor, ObjectTypeChatMessage, ObjectTypeNotificationUnreadSummaryAccount, ObjectTypeMessagingBlock, ObjectTypeNotificationPreference, ObjectTypeMessageAttachment, ObjectTypeAttachmentUploadTarget, ObjectTypeScheduledMessage, ObjectTypeMessagingContact, ObjectTypeMessageReport, ObjectTypeToolGroup, ObjectTypeModel, ObjectTypePaymentTerm, ObjectTypeShippingTerm, ObjectTypeQuantity, ObjectTypeAccountGroup, ObjectTypeSupportRoute, ObjectTypeSupportAvailability, ObjectTypeAccountStatus, ObjectTypeGeolocation, ObjectTypeAccountUser, ObjectTypeDepartment, ObjectTypeAccountIntegration, ObjectTypeAccountPrice, ObjectTypeProductLine, ObjectTypeItemCategory, ObjectTypeAttribute, ObjectTypeRate, ObjectTypeAccountGroupProductLineAccess, ObjectTypeSalesTarget, ObjectTypeAdjustmentType, ObjectTypeAccountBranding, ObjectTypeAccountPortal, ObjectTypeAccountLogoURL, ObjectTypeAccountFaviconURL, ObjectTypePublicAccount, ObjectTypeProperty, ObjectTypeCarrier, ObjectTypeServiceLevel, ObjectTypeItem, ObjectTypeItemInventory, ObjectTypeItemLotDefault, ObjectTypeProduct, ObjectTypeBatch, ObjectTypeBatchFlowNode, ObjectTypeScanningConsumption, ObjectTypeOpenBatchSummary, ObjectTypeScanningProductionStepInfo, ObjectTypeScanningStation, ObjectTypeProductionStep, ObjectTypeProductionRun, ObjectTypeMachine, ObjectTypeMachineStatus, ObjectTypeMachineDowntimeEvent, ObjectTypeMachineDowntimeReason, ObjectTypeDemandOverride, ObjectTypeDemandOverrideType, ObjectTypeProductionSchedulePreview, ObjectTypeProductionScheduleRegeneratePreview, ObjectTypeProductionSchedule, ObjectTypeProductionScheduleLine, ObjectTypeProductionScheduleItemPolicy, ObjectTypeProductionScheduleFinishedPolicy, ObjectTypeProductionScheduleWeekRelease, ObjectTypeProductionScheduleWeekReleasePreview, ObjectTypeProductionScheduleDeviation, ObjectTypeProductionScheduleDerivedLine, ObjectTypeProductionScheduleSettings, ObjectTypeProductionScheduleResourceSetting, ObjectTypeScheduleDeviationType, ObjectTypeChildAccount, ObjectTypeUnitGroup, ObjectTypeUnitGroupUnit, ObjectTypeConsumption, ObjectTypeCustomerProductLineAccess, ObjectTypeCustomer, ObjectTypeFrequentlyOrderedProduct, ObjectTypePriority, ObjectTypeDelivery, ObjectTypeDeliveryLine, ObjectTypeSalesOrder, ObjectTypeSalesOrderLine, ObjectTypeSalesOrderType, ObjectTypeLocation, ObjectTypeLocationType, ObjectTypeLot, ObjectTypeEmailLog, ObjectTypeEmailDomain, ObjectTypeEmailInbox, ObjectTypePortalDomain, ObjectTypeDNSRecord, ObjectTypeInventoryChangeLog, ObjectTypeInvoice, ObjectTypeInvoiceSummary, ObjectTypeInvoiceLine, ObjectTypeInvoiceAllocation, ObjectTypeInvoiceForPayment, ObjectTypeShipment, ObjectTypeShipmentSummary, ObjectTypeShipmentLine, ObjectTypeShippingCase, ObjectTypeShippingCaseLabelURL, ObjectTypeSettlement, ObjectTypeSettlementSummary, ObjectTypeRolePermission, ObjectTypeRegistrationFlow, ObjectTypeRegistrationFlowOption, ObjectTypeTransaction, ObjectTypeTransactionSummary, ObjectTypeTransactionMethod, ObjectTypeTransactionType, ObjectTypeTransactionAllocation, ObjectTypeUsageItem, ObjectTypeAccountUsageResponse, ObjectTypeSubscriptionInfo, ObjectTypeBillingPortalSessionResponse, ObjectTypeSwitchPlanResponse, ObjectTypeEnsureBillingCustomerResponse, ObjectTypeSpendingCapResponse, ObjectTypeAgentSpendInfo, ObjectTypeWebhookResponse, ObjectTypeAddressSuggestion, ObjectTypeAddressComponents, ObjectTypeAddressDetailsResult, ObjectTypeValidatedAddress, ObjectTypePlanLimit, ObjectTypePlanChangeProration, ObjectTypePlanChangeLineItem, ObjectTypeSetupBillingResponse, ObjectTypeConfirmPaymentResponse, ObjectTypeOAuthResponse, ObjectTypeOAuthStatusResponse, ObjectTypeStripePublishableKey, ObjectTypeStripeStatus, ObjectTypeHealthcheck, ObjectTypeAgentDefinitionConfig, ObjectTypeTriggerConfig, ObjectTypeCustomerContactInfo, ObjectTypeCustomerFreightPreferences, ObjectTypeCustomerDefaults, ObjectTypeCustomerNotificationPreferences, ObjectTypeOrderNotificationRecipient, ObjectTypeOrderDiscount, ObjectTypeSalesOrderStatus, ObjectTypeMaterial, ObjectTypeSupplierMaterial, ObjectTypePart, ObjectTypePermissionGroup, ObjectTypePermission, ObjectTypePick, ObjectTypePickLine, ObjectTypeProductType, ObjectTypeProduction, ObjectTypeProductionFlow, ObjectTypeMap, ObjectTypePurchaseOrder, ObjectTypePurchaseOrderLine, ObjectTypeSupplier, ObjectTypeSupplierSummary, ObjectTypeReceivableEntry, ObjectTypeReceivingOrder, ObjectTypeReceivingOrderLine, ObjectTypeEmailContact, ObjectTypeAllocationEntry, ObjectTypeOpenCreditEntry, ObjectTypeVolumeDiscount, ObjectTypeVolumeDiscountTier, ObjectTypeAnalyzeDeliveriesResponse, ObjectTypeAnalyzeManufacturingResponse, ObjectTypeAnalyzeManufacturingBatchResponse, ObjectTypeAnalyzeQuarterlyOrdersResponse, ObjectTypeAnalyzeNewCustomersResponse, ObjectTypeAnalyzeDemandForecastResponse, ObjectTypeAnalyzeOeeResponse, ObjectTypeAnalyzeOeeTrendResponse, ObjectTypeAnalyzeScheduleAttainmentResponse, ObjectTypeCatalogProductLine, ObjectTypeCatalogCategory, ObjectTypeCatalogProduct, ObjectTypeCatalogProperty, ObjectTypeCatalogAttribute, ObjectTypeDCLocation, ObjectTypeEDIRun, ObjectTypeInventoryItem, ObjectTypeAnalyzeWeeksOfSalesResponse, ObjectTypeBulkReconcileItemsResponse, ObjectTypeSysProperty, ObjectTypeSysPropertyType, ObjectTypeSysPropertyValue, ObjectTypeTerritory, ObjectTypeTenancy, ObjectTypeCheckoutSession, ObjectTypeEstimateRateResult, ObjectTypeRateShopOption, ObjectTypeRateShopResult, ObjectTypeOwner, ObjectTypeCreatedBy, ObjectTypeMessage, ObjectTypeAccountPhotoUploadResult, ObjectTypeUserPhotoUploadResult, ObjectTypeUserPhotoURL, ObjectTypeBatchLot, ObjectTypeCheckDuplicateResult, ObjectTypeItemTrendPoint, ObjectTypePackPickResponse, ObjectTypePickShipmentsResponse, ObjectTypeTenancyPendingRegistration, ObjectTypeInvoiceAllocationEntry, ObjectTypeAllocationCustomer, ObjectTypeAccountPlan, ObjectTypeCheckoutSalesOrderResponse, ObjectTypeSalesOrderPriceQuote, ObjectTypeSalesOrderFreightQuote, ObjectTypeSalesOrderPriceQuoteLine, ObjectTypeSalesOrderQuoteRate, ObjectTypeHubspotSyncJob, ObjectTypeHubspotSyncReport, ObjectTypeHubspotCompanyReview, ObjectTypeHubspotCompanyCandidate, ObjectTypeHubspotSyncRecord, ObjectTypeContactMatch, ObjectTypeReplyDraft, ObjectTypeConversationLink, ObjectTypeMessagingGroup, ObjectTypeMessagingGroupMember, ObjectTypePortalProfile, ObjectTypePortalRegistrationSession, ObjectTypePortalRegistrationSessionData, ObjectTypePackList, ObjectTypePackListParty, ObjectTypePackListLineItem, ObjectTypePackListBackOrder, ObjectTypePackListCase, ObjectTypeJob:
		return true
	default:
		return false
	}
}

func (m ObjectType) EnumValues() []string {
	return []string{string(ObjectTypeAccount), string(ObjectTypeActor), string(ObjectTypeEntity), string(ObjectTypeRecord), string(ObjectTypeFreight), string(ObjectTypeSalesOrderTotals), string(ObjectTypeSalesOrderStageTotal), string(ObjectTypeSalesOrderRelated), string(ObjectTypeOrderContact), string(ObjectTypeUser), string(ObjectTypeAddress), string(ObjectTypeAPIKey), string(ObjectTypeCreatedAPIKey), string(ObjectTypeRefreshToken), string(ObjectTypeList), string(ObjectTypeSandbox), string(ObjectTypeRegistrationSession), string(ObjectTypePricingPlan), string(ObjectTypeAccountPlan), string(ObjectTypePlanChange), string(ObjectTypeEnterpriseInquiry), string(ObjectTypeRequestLog), string(ObjectTypeAuditEvent), string(ObjectTypeAuditFieldChange), string(ObjectTypeRole), string(ObjectTypeUnit), string(ObjectTypeAccountAffiliation), string(ObjectTypeAgentDefinition), string(ObjectTypeAvailableTool), string(ObjectTypeAgentDefinitionTool), string(ObjectTypeAgentAccountStatus), string(ObjectTypeAgentRun), string(ObjectTypeAgentAction), string(ObjectTypeAgentRunStep), string(ObjectTypeAgentTokenUsage), string(ObjectTypeAgentMemory), string(ObjectTypeNotification), string(ObjectTypeNotificationUnreadCount), string(ObjectTypeNotificationSendResult), string(ObjectTypeNotificationUnreadSummary), string(ObjectTypeAnnouncement), string(ObjectTypeConversation), string(ObjectTypeSupportCase), string(ObjectTypeConversationParticipant), string(ObjectTypeReadCursor), string(ObjectTypeChatMessage), string(ObjectTypeNotificationUnreadSummaryAccount), string(ObjectTypeMessagingBlock), string(ObjectTypeNotificationPreference), string(ObjectTypeMessageAttachment), string(ObjectTypeAttachmentUploadTarget), string(ObjectTypeScheduledMessage), string(ObjectTypeMessagingContact), string(ObjectTypeMessageReport), string(ObjectTypeToolGroup), string(ObjectTypeModel), string(ObjectTypePaymentTerm), string(ObjectTypeShippingTerm), string(ObjectTypeQuantity), string(ObjectTypeAccountGroup), string(ObjectTypeSupportRoute), string(ObjectTypeSupportAvailability), string(ObjectTypeAccountStatus), string(ObjectTypeGeolocation), string(ObjectTypeAccountUser), string(ObjectTypeDepartment), string(ObjectTypeAccountIntegration), string(ObjectTypeAccountPrice), string(ObjectTypeProductLine), string(ObjectTypeItemCategory), string(ObjectTypeAttribute), string(ObjectTypeRate), string(ObjectTypeAccountGroupProductLineAccess), string(ObjectTypeSalesTarget), string(ObjectTypeAdjustmentType), string(ObjectTypeAccountBranding), string(ObjectTypeAccountPortal), string(ObjectTypeAccountLogoURL), string(ObjectTypeAccountFaviconURL), string(ObjectTypePublicAccount), string(ObjectTypeProperty), string(ObjectTypeCarrier), string(ObjectTypeServiceLevel), string(ObjectTypeItem), string(ObjectTypeItemLotDefault), string(ObjectTypeItemInventory), string(ObjectTypeProduct), string(ObjectTypeBatch), string(ObjectTypeBatchFlowNode), string(ObjectTypeScanningConsumption), string(ObjectTypeOpenBatchSummary), string(ObjectTypeScanningProductionStepInfo), string(ObjectTypeScanningStation), string(ObjectTypeProductionStep), string(ObjectTypeProductionRun), string(ObjectTypeMachine), string(ObjectTypeMachineStatus), string(ObjectTypeMachineDowntimeEvent), string(ObjectTypeDemandOverride), string(ObjectTypeDemandOverrideType), string(ObjectTypeMachineDowntimeReason), string(ObjectTypeProductionSchedulePreview), string(ObjectTypeProductionScheduleRegeneratePreview), string(ObjectTypeProductionSchedule), string(ObjectTypeProductionScheduleLine), string(ObjectTypeProductionScheduleDeviation), string(ObjectTypeProductionScheduleDerivedLine), string(ObjectTypeProductionScheduleSettings), string(ObjectTypeProductionScheduleResourceSetting), string(ObjectTypeScheduleDeviationType), string(ObjectTypeProductionScheduleFinishedPolicy), string(ObjectTypeProductionScheduleWeekRelease), string(ObjectTypeProductionScheduleWeekReleasePreview), string(ObjectTypeProductionScheduleItemPolicy), string(ObjectTypeChildAccount), string(ObjectTypeUnitGroup), string(ObjectTypeUnitGroupUnit), string(ObjectTypeConsumption), string(ObjectTypeCustomerProductLineAccess), string(ObjectTypeCustomer), string(ObjectTypeFrequentlyOrderedProduct), string(ObjectTypePriority), string(ObjectTypeDelivery), string(ObjectTypeDeliveryLine), string(ObjectTypeSalesOrder), string(ObjectTypeLocation), string(ObjectTypeLocationType), string(ObjectTypeLot), string(ObjectTypeEmailLog), string(ObjectTypeEmailDomain), string(ObjectTypeEmailInbox), string(ObjectTypePortalDomain), string(ObjectTypeDNSRecord), string(ObjectTypeInventoryChangeLog), string(ObjectTypeInvoice), string(ObjectTypeInvoiceSummary), string(ObjectTypeInvoiceLine), string(ObjectTypeInvoiceAllocation), string(ObjectTypeInvoiceForPayment), string(ObjectTypeShipment), string(ObjectTypeShipmentSummary), string(ObjectTypeShipmentLine), string(ObjectTypeShippingCase), string(ObjectTypeShippingCaseLabelURL), string(ObjectTypeSettlement), string(ObjectTypeSettlementSummary), string(ObjectTypeRolePermission), string(ObjectTypeRegistrationFlow), string(ObjectTypeRegistrationFlowOption), string(ObjectTypeTransaction), string(ObjectTypeTransactionSummary), string(ObjectTypeTransactionMethod), string(ObjectTypeTransactionType), string(ObjectTypeTransactionAllocation), string(ObjectTypeUsageItem), string(ObjectTypeAccountUsageResponse), string(ObjectTypeSubscriptionInfo), string(ObjectTypeBillingPortalSessionResponse), string(ObjectTypeSwitchPlanResponse), string(ObjectTypeEnsureBillingCustomerResponse), string(ObjectTypeSpendingCapResponse), string(ObjectTypeAgentSpendInfo), string(ObjectTypeWebhookResponse), string(ObjectTypeAddressSuggestion), string(ObjectTypeAddressComponents), string(ObjectTypeAddressDetailsResult), string(ObjectTypeValidatedAddress), string(ObjectTypePlanLimit), string(ObjectTypePlanChangeProration), string(ObjectTypePlanChangeLineItem), string(ObjectTypeSetupBillingResponse), string(ObjectTypeConfirmPaymentResponse), string(ObjectTypeOAuthResponse), string(ObjectTypeOAuthStatusResponse), string(ObjectTypeStripePublishableKey), string(ObjectTypeStripeStatus), string(ObjectTypeHealthcheck), string(ObjectTypeAgentDefinitionConfig), string(ObjectTypeTriggerConfig), string(ObjectTypeCustomerContactInfo), string(ObjectTypeCustomerFreightPreferences), string(ObjectTypeCustomerDefaults), string(ObjectTypeCustomerNotificationPreferences), string(ObjectTypeOrderNotificationRecipient), string(ObjectTypeOrderDiscount), string(ObjectTypeSalesOrderLine), string(ObjectTypeSalesOrderType), string(ObjectTypeSalesOrderStatus), string(ObjectTypeMaterial), string(ObjectTypeSupplierMaterial), string(ObjectTypePart), string(ObjectTypePermissionGroup), string(ObjectTypePermission), string(ObjectTypePick), string(ObjectTypePickLine), string(ObjectTypeProductType), string(ObjectTypeProduction), string(ObjectTypeProductionFlow), string(ObjectTypeMap), string(ObjectTypePurchaseOrder), string(ObjectTypePurchaseOrderLine), string(ObjectTypeSupplier), string(ObjectTypeSupplierSummary), string(ObjectTypeReceivableEntry), string(ObjectTypeReceivingOrder), string(ObjectTypeReceivingOrderLine), string(ObjectTypeEmailContact), string(ObjectTypeAllocationEntry), string(ObjectTypeOpenCreditEntry), string(ObjectTypeVolumeDiscount), string(ObjectTypeVolumeDiscountTier), string(ObjectTypeAnalyzeDeliveriesResponse), string(ObjectTypeAnalyzeManufacturingResponse), string(ObjectTypeAnalyzeManufacturingBatchResponse), string(ObjectTypeAnalyzeQuarterlyOrdersResponse), string(ObjectTypeAnalyzeNewCustomersResponse), string(ObjectTypeAnalyzeDemandForecastResponse), string(ObjectTypeAnalyzeOeeResponse), string(ObjectTypeAnalyzeOeeTrendResponse), string(ObjectTypeAnalyzeScheduleAttainmentResponse), string(ObjectTypeCatalogProductLine), string(ObjectTypeCatalogCategory), string(ObjectTypeCatalogProduct), string(ObjectTypeCatalogProperty), string(ObjectTypeCatalogAttribute), string(ObjectTypeDCLocation), string(ObjectTypeEDIRun), string(ObjectTypeInventoryItem), string(ObjectTypeAnalyzeWeeksOfSalesResponse), string(ObjectTypeBulkReconcileItemsResponse), string(ObjectTypeSysProperty), string(ObjectTypeSysPropertyType), string(ObjectTypeSysPropertyValue), string(ObjectTypeTerritory), string(ObjectTypeTenancy), string(ObjectTypeCheckoutSession), string(ObjectTypeEstimateRateResult), string(ObjectTypeRateShopOption), string(ObjectTypeRateShopResult), string(ObjectTypeOwner), string(ObjectTypeCreatedBy), string(ObjectTypeMessage), string(ObjectTypeAccountPhotoUploadResult), string(ObjectTypeUserPhotoUploadResult), string(ObjectTypeUserPhotoURL), string(ObjectTypeBatchLot), string(ObjectTypeCheckDuplicateResult), string(ObjectTypeItemTrendPoint), string(ObjectTypePackPickResponse), string(ObjectTypePickShipmentsResponse), string(ObjectTypeTenancyPendingRegistration), string(ObjectTypeInvoiceAllocationEntry), string(ObjectTypeAllocationCustomer), string(ObjectTypeCheckoutSalesOrderResponse), string(ObjectTypeSalesOrderPriceQuote), string(ObjectTypeSalesOrderFreightQuote), string(ObjectTypeSalesOrderPriceQuoteLine), string(ObjectTypeSalesOrderQuoteRate), string(ObjectTypeHubspotSyncJob), string(ObjectTypeHubspotSyncReport), string(ObjectTypeHubspotCompanyReview), string(ObjectTypeHubspotCompanyCandidate), string(ObjectTypeHubspotSyncRecord), string(ObjectTypeContactMatch), string(ObjectTypeReplyDraft), string(ObjectTypeConversationLink), string(ObjectTypeMessagingGroup), string(ObjectTypeMessagingGroupMember), string(ObjectTypePortalProfile), string(ObjectTypePortalRegistrationSession), string(ObjectTypePortalRegistrationSessionData), string(ObjectTypePackList), string(ObjectTypePackListParty), string(ObjectTypePackListLineItem), string(ObjectTypePackListBackOrder), string(ObjectTypePackListCase), string(ObjectTypeJob)}
}
