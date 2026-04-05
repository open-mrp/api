-- 0001_static_types.sql
-- Seeds all lookup/enum tables with static type data.
-- Only includes tables that exist as actual DB tables (not string enums in columns).

-- action_type (prefix: axtp)
INSERT IGNORE INTO action_type (id, code, name, created_at, updated_at) VALUES
    ('axtp_01seedscan0000000000', 'scan', 'Scan', NOW(), NOW()),
    ('axtp_01seedusercorrection', 'user_correction', 'User Correction', NOW(), NOW()),
    ('axtp_01seeduseraction0000', 'user_action', 'User Action', NOW(), NOW()),
    ('axtp_01seedsystemaction00', 'system_action', 'System Action', NOW(), NOW()),
    ('axtp_01seedcreaterecord00', 'create_record', 'Create Record', NOW(), NOW()),
    ('axtp_01seedupdaterecord00', 'update_record', 'Update Record', NOW(), NOW()),
    ('axtp_01seeddeleterecord00', 'delete_record', 'Delete Record', NOW(), NOW()),
    ('axtp_01seedfetchrecord000', 'fetch_record', 'Fetch Record', NOW(), NOW());

-- transaction_type (prefix: txtp)
INSERT IGNORE INTO transaction_type (id, code, name, is_commission_affected, created_at, updated_at) VALUES
    ('txtp_01seedpayment000000', 'payment', 'Payment', 0, NOW(), NOW()),
    ('txtp_01seedcreditmemo000', 'credit_memo', 'Credit Memo', 0, NOW(), NOW()),
    ('txtp_01seedadjustment000', 'adjustment', 'Adjustment', 0, NOW(), NOW()),
    ('txtp_01seedrebate0000000', 'rebate', 'Rebate', 0, NOW(), NOW());

-- adjustment_type (prefix: ajtp)
INSERT IGNORE INTO adjustment_type (id, code, name, created_at, updated_at) VALUES
    ('ajtp_01seeddiscount00000', 'discount', 'Discount', NOW(), NOW()),
    ('ajtp_01seedshipdiscrep00', 'shipping_discrepancy', 'Shipping Discrepancy', NOW(), NOW()),
    ('ajtp_01seedshortpayment0', 'short_payment', 'Short Payment', NOW(), NOW()),
    ('ajtp_01seedwriteoff00000', 'write_off', 'Write Off', NOW(), NOW()),
    ('ajtp_01seedfee0000000000', 'fee', 'Fee', NOW(), NOW()),
    ('ajtp_01seedrefund0000000', 'refund', 'Refund', NOW(), NOW());

-- color (prefix: cl)
INSERT IGNORE INTO color (id, code, name, created_at, updated_at) VALUES
    ('cl_01seedblue00000000000', 'blue', 'Blue', NOW(), NOW()),
    ('cl_01seedbrown0000000000', 'brown', 'Brown', NOW(), NOW()),
    ('cl_01seeddefault00000000', 'default', 'Default', NOW(), NOW()),
    ('cl_01seedgray00000000000', 'gray', 'Gray', NOW(), NOW()),
    ('cl_01seedgreen0000000000', 'green', 'Green', NOW(), NOW()),
    ('cl_01seedorange000000000', 'orange', 'Orange', NOW(), NOW()),
    ('cl_01seedpink00000000000', 'pink', 'Pink', NOW(), NOW()),
    ('cl_01seedpurple000000000', 'purple', 'Purple', NOW(), NOW()),
    ('cl_01seedred000000000000', 'red', 'Red', NOW(), NOW()),
    ('cl_01seedyellow000000000', 'yellow', 'Yellow', NOW(), NOW());

-- delivery_status (prefix: dvss)
INSERT IGNORE INTO delivery_status (id, code, name, created_at, updated_at) VALUES
    ('dvss_01seedall000000000', 'all', 'All', NOW(), NOW()),
    ('dvss_01seedaccepted0000', 'accepted', 'Accepted', NOW(), NOW()),
    ('dvss_01seedrejected0000', 'rejected', 'Rejected', NOW(), NOW());

-- inventory_receipt_status (no explicit prefix, use irst)
INSERT IGNORE INTO inventory_receipt_status (id, code, name, created_at, updated_at) VALUES
    ('irst_01seedavailable000', 'available', 'Available', NOW(), NOW()),
    ('irst_01seedallocated000', 'allocated', 'Allocated', NOW(), NOW());

-- inventory_issue_status (prefix: iiss)
INSERT IGNORE INTO InventoryIssueStatus (id, code, name, created_at, updated_at) VALUES
    ('iiss_01seedopen00000000', 'open', 'Open', NOW(), NOW()),
    ('iiss_01seedreserved0000', 'reserved', 'Reserved', NOW(), NOW()),
    ('iiss_01seedclosed000000', 'closed', 'Closed', NOW(), NOW());

-- account_status (prefix: acss)
INSERT IGNORE INTO account_status (id, code, name, created_at, updated_at) VALUES
    ('acss_01seedholdall00000', 'hold_all', 'Hold All', NOW(), NOW()),
    ('acss_01seedholdshipment', 'hold_shipment', 'Hold Shipment', NOW(), NOW()),
    ('acss_01seednormal000000', 'normal', 'Normal', NOW(), NOW()),
    ('acss_01seedpreferred000', 'preferred', 'Preferred', NOW(), NOW());

-- product_type (prefix: pdtp)
INSERT IGNORE INTO product_type (id, code, name, created_at, updated_at) VALUES
    ('pdtp_01seedsale00000000', 'sale', 'Sale', NOW(), NOW()),
    ('pdtp_01seedservice00000', 'service', 'Service', NOW(), NOW()),
    ('pdtp_01seedshipping0000', 'shipping', 'Shipping', NOW(), NOW()),
    ('pdtp_01seedcredit000000', 'credit', 'Credit', NOW(), NOW()),
    ('pdtp_01seedreturn000000', 'return', 'Return', NOW(), NOW()),
    ('pdtp_01seedtax000000000', 'tax', 'Tax', NOW(), NOW());

-- priority (prefix: pi)
INSERT IGNORE INTO priority (id, code, name, created_at, updated_at) VALUES
    ('pi_01seedlow000000000000', 'low', 'Low', NOW(), NOW()),
    ('pi_01seednormal0000000000', 'normal', 'Normal', NOW(), NOW()),
    ('pi_01seedhigh00000000000', 'high', 'High', NOW(), NOW());

-- item_type (prefix: ittp)
INSERT IGNORE INTO item_type (id, code, name, created_at, updated_at) VALUES
    ('ittp_01seedproduct00000', 'product', 'Product', NOW(), NOW()),
    ('ittp_01seedmaterial0000', 'material', 'Material', NOW(), NOW()),
    ('ittp_01seedpart00000000', 'part', 'Part', NOW(), NOW());

-- scanning_station_type (prefix: sgsntp)
INSERT IGNORE INTO scanning_station_type (id, code, name, created_at, updated_at) VALUES
    ('sgsntp_01seedinitbatch', 'init_batch', 'Init Batch', NOW(), NOW()),
    ('sgsntp_01seedmergebatch', 'merge_batch', 'Merge Batch', NOW(), NOW()),
    ('sgsntp_01seedmovebatch0', 'move_batch', 'Move Batch', NOW(), NOW()),
    ('sgsntp_01seedsplitbatch', 'split_batch', 'Split Batch', NOW(), NOW());

-- shipment_status (prefix: shss)
INSERT IGNORE INTO shipment_status (id, code, name, created_at, updated_at) VALUES
    ('shss_01seedall000000000', 'all', 'All', NOW(), NOW()),
    ('shss_01seedpacked000000', 'packed', 'Packed', NOW(), NOW()),
    ('shss_01seedshipped00000', 'shipped', 'Shipped', NOW(), NOW());

-- sys_property_type (prefix: sypptp)
INSERT IGNORE INTO sys_property_type (id, code, name, created_at, updated_at) VALUES
    ('sypptp_01seedtxnumber0', 'transaction_number', 'Transaction Number', NOW(), NOW()),
    ('sypptp_01seedslnumber0', 'settlement_number', 'Settlement Number', NOW(), NOW()),
    ('sypptp_01seedsonumber0', 'sales_order_number', 'Sales Order Number', NOW(), NOW()),
    ('sypptp_01seedponumber0', 'purchase_order_number', 'Purchase Order Number', NOW(), NOW()),
    ('sypptp_01seedsupnumber', 'supplier_number', 'Supplier Number', NOW(), NOW()),
    ('sypptp_01seedcustnumbr', 'customer_number', 'Customer Number', NOW(), NOW()),
    ('sypptp_01seedsscccount', 'sscc_count', 'Sscc Count', NOW(), NOW()),
    ('sypptp_01seedprnumber0', 'production_run_number', 'Production Run Number', NOW(), NOW());

-- storage_location_type
INSERT IGNORE INTO storage_location_type (id, code, name, created_at, updated_at) VALUES
    ('sltp_01seedbuilding0000', 'building', 'Building', NOW(), NOW()),
    ('sltp_01seedsection00000', 'section', 'Section', NOW(), NOW()),
    ('sltp_01seedaisle0000000', 'aisle', 'Aisle', NOW(), NOW()),
    ('sltp_01seedrack00000000', 'rack', 'Rack', NOW(), NOW()),
    ('sltp_01seedshelf0000000', 'shelf', 'Shelf', NOW(), NOW()),
    ('sltp_01seedbin000000000', 'bin', 'Bin', NOW(), NOW());

-- transaction_method (prefix: txmd)
INSERT IGNORE INTO transaction_method (id, code, name, created_at, updated_at) VALUES
    ('txmd_01seedcash00000000', 'cash', 'Cash', NOW(), NOW()),
    ('txmd_01seedcheck0000000', 'check', 'Check', NOW(), NOW()),
    ('txmd_01seedcreditcard00', 'credit_card', 'Credit Card', NOW(), NOW()),
    ('txmd_01seedgiftcard0000', 'gift_card', 'Gift Card', NOW(), NOW()),
    ('txmd_01seedach000000000', 'ach', 'Ach', NOW(), NOW());

-- unit_type
INSERT IGNORE INTO unit_type (id, code, name, created_at, updated_at) VALUES
    ('untp_01seedcurrency0000', 'currency', 'Currency', NOW(), NOW()),
    ('untp_01seedquantity0000', 'quantity', 'Quantity', NOW(), NOW()),
    ('untp_01seedtime00000000', 'time', 'Time', NOW(), NOW()),
    ('untp_01seedmass00000000', 'mass', 'Mass', NOW(), NOW()),
    ('untp_01seedvolume000000', 'volume', 'Volume', NOW(), NOW()),
    ('untp_01seedlength000000', 'length', 'Length', NOW(), NOW()),
    ('untp_01seedtemperature0', 'temperature', 'Temperature', NOW(), NOW()),
    ('untp_01seedarea00000000', 'area', 'Area', NOW(), NOW());

-- unit (default/global units, account_id = NULL)
INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, ratio_numerator, ratio_denominator, offset_numerator, offset_denominator, is_base_unit, account_id, created_at, updated_at) VALUES
    ('celcius',        'Celsius',        '°C',   'temperature', 1, 1, 0, 1, 1, NULL, NOW(), NOW()),
    ('cup',            'Cup',            'cup',  'volume',      236.588, 1000000, 0, 1, 0, NULL, NOW(), NOW()),
    ('day',            'Day',            'day',  'time',        86400, 1, 0, 1, 0, NULL, NOW(), NOW()),
    ('dollar',         'Dollar',         '$',    'currency',    1, 1, 0, 1, 1, NULL, NOW(), NOW()),
    ('dozen',          'Dozen',          'dz',   'quantity',    12, 1, 0, 1, 0, NULL, NOW(), NOW()),
    ('each',           'Each',           'ea',   'quantity',    1, 1, 0, 1, 1, NULL, NOW(), NOW()),
    ('feet',           'Feet',           'ft',   'length',      304800, 1000000, 0, 1, 0, NULL, NOW(), NOW()),
    ('fluid_ounce',    'Fluid Ounce',    'fl oz','volume',      29573.5, 1000000, 0, 1, 0, NULL, NOW(), NOW()),
    ('gallon',         'Gallon',         'gal',  'volume',      3785410, 1000000, 0, 1, 0, NULL, NOW(), NOW()),
    ('grain',          'Grain',          'gr',   'mass',        64799, 1000000, 0, 1, 0, NULL, NOW(), NOW()),
    ('gram',           'Gram',           'g',    'mass',        1, 1, 0, 1, 1, NULL, NOW(), NOW()),
    ('hour',           'Hour',           'hr',   'time',        3600, 1, 0, 1, 0, NULL, NOW(), NOW()),
    ('liter',          'Liter',          'L',    'volume',      1, 1, 0, 1, 1, NULL, NOW(), NOW()),
    ('meter',          'Meter',          'm',    'length',      1, 1, 0, 1, 1, NULL, NOW(), NOW()),
    ('meters_squared', 'Square Meter',   'm²',   'area',        1, 1, 0, 1, 1, NULL, NOW(), NOW()),
    ('minute',         'Minute',         'min',  'time',        60, 1, 0, 1, 0, NULL, NOW(), NOW()),
    ('month',          'Month',          'mo',   'time',        2592000, 1, 0, 1, 0, NULL, NOW(), NOW()),
    ('pair',           'Pair',           'pr',   'quantity',    2, 1, 0, 1, 0, NULL, NOW(), NOW()),
    ('pound',          'Pound',          'lb',   'mass',        453592, 1000, 0, 1, 0, NULL, NOW(), NOW()),
    ('quarter',        'Quarter',        'qtr',  'time',        7776000, 1, 0, 1, 0, NULL, NOW(), NOW()),
    ('second',         'Second',         's',    'time',        1, 1, 0, 1, 1, NULL, NOW(), NOW()),
    ('week',           'Week',           'wk',   'time',        604800, 1, 0, 1, 0, NULL, NOW(), NOW()),
    ('yard',           'Yard',           'yd',   'length',      914400, 1000000, 0, 1, 0, NULL, NOW(), NOW());

-- onboarding_status (prefix: obss)
INSERT IGNORE INTO onboarding_status (id, code, name, created_at, updated_at) VALUES
    ('obss_01seedunclaimed000', 'unclaimed', 'Unclaimed', NOW(), NOW()),
    ('obss_01seedactive000000', 'active', 'Active', NOW(), NOW()),
    ('obss_01seedsuspended000', 'suspended', 'Suspended', NOW(), NOW()),
    ('obss_01seeddeactivated0', 'deactivated', 'Deactivated', NOW(), NOW());

-- role_type (prefix: rltp)
INSERT IGNORE INTO role_type (id, code, name, created_at, updated_at) VALUES
    ('rltp_01seedadmin0000000', 'admin', 'Admin', NOW(), NOW()),
    ('rltp_01seeduser00000000', 'user', 'User', NOW(), NOW()),
    ('rltp_01seedscanner00000', 'scanner', 'Scanner', NOW(), NOW()),
    ('rltp_01seedsalesrep0000', 'sales_rep', 'Sales Rep', NOW(), NOW());

-- permission_group (prefix: pmgp)
INSERT IGNORE INTO permission_group (id, code, name, created_at, updated_at) VALUES
    ('pmgp_01seedself00000000', 'self', 'Self', NOW(), NOW()),
    ('pmgp_01seedadmin0000000', 'admin', 'Admin', NOW(), NOW()),
    ('pmgp_01seedcustomers000', 'customers', 'Customers', NOW(), NOW()),
    ('pmgp_01seedcustportals0', 'customer_portals', 'Customer Portals', NOW(), NOW()),
    ('pmgp_01seeddepartments0', 'departments', 'Departments', NOW(), NOW()),
    ('pmgp_01seededi000000000', 'edi', 'Edi', NOW(), NOW()),
    ('pmgp_01seedinventory000', 'inventory', 'Inventory', NOW(), NOW()),
    ('pmgp_01seedinvoices0000', 'invoices', 'Invoices', NOW(), NOW()),
    ('pmgp_01seeditems0000000', 'items', 'Items', NOW(), NOW()),
    ('pmgp_01seedlogs00000000', 'logs', 'Logs', NOW(), NOW()),
    ('pmgp_01seedpayments0000', 'payments', 'Payments', NOW(), NOW()),
    ('pmgp_01seedpicking00000', 'picking', 'Picking', NOW(), NOW()),
    ('pmgp_01seedpricing00000', 'pricing', 'Pricing', NOW(), NOW()),
    ('pmgp_01seedproduction00', 'production', 'Production', NOW(), NOW()),
    ('pmgp_01seedproducts0000', 'products', 'Products', NOW(), NOW()),
    ('pmgp_01seedpurchasing00', 'purchasing', 'Purchasing', NOW(), NOW()),
    ('pmgp_01seedreports00000', 'reports', 'Reports', NOW(), NOW()),
    ('pmgp_01seedsalesorders0', 'sales_orders', 'Sales Orders', NOW(), NOW()),
    ('pmgp_01seedsalesreps000', 'sales_reps', 'Sales Reps', NOW(), NOW()),
    ('pmgp_01seedscanning0000', 'scanning', 'Scanning', NOW(), NOW()),
    ('pmgp_01seedshipping0000', 'shipping', 'Shipping', NOW(), NOW()),
    ('pmgp_01seedteams0000000', 'teams', 'Teams', NOW(), NOW()),
    ('pmgp_01seedunits0000000', 'units', 'Units', NOW(), NOW());

-- inventory_policy_type
INSERT IGNORE INTO inventory_policy_type (id, code, name, created_at, updated_at) VALUES
    ('iptp_01seedfifo00000000', 'fifo', 'Fifo', NOW(), NOW()),
    ('iptp_01seedlifo00000000', 'lifo', 'Lifo', NOW(), NOW());

-- item_category_type (prefix: itcgtp)
INSERT IGNORE INTO item_category_type (id, code, name, created_at, updated_at) VALUES
    ('itcgtp_01seedmatcat000', 'material_category', 'Material Category', NOW(), NOW()),
    ('itcgtp_01seedprodcat00', 'product_category', 'Product Category', NOW(), NOW());

-- sales_order_status (prefix: orss)
INSERT IGNORE INTO sales_order_status (id, code, name, created_at, updated_at) VALUES
    ('orss_01seedall000000000', 'all', 'All', NOW(), NOW()),
    ('orss_01seedestimate0000', 'estimate', 'Estimate', NOW(), NOW()),
    ('orss_01seedissued000000', 'issued', 'Issued', NOW(), NOW()),
    ('orss_01seedfulfilled000', 'fulfilled', 'Fulfilled', NOW(), NOW());

-- sales_order_type (prefix: ortp)
INSERT IGNORE INTO sales_order_type (id, code, name, created_at, updated_at) VALUES
    ('sales_order', 'sales_order', 'Sales Order', NOW(), NOW()),
    ('purchase_order', 'purchase_order', 'Purchase Order', NOW(), NOW());

-- shipping_term (global/shared, no account_id)
INSERT IGNORE INTO shipping_term (id, name, is_freight_exempt, is_carrier_rate, created_at, updated_at) VALUES
    ('prepaid', 'Free Shipping', 0, 1, NOW(), NOW()),
    ('prepaid_billed', 'Freight Charged', 0, 1, NOW(), NOW()),
    ('freight_collect', 'Freight Collect', 0, 1, NOW(), NOW());
