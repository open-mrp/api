-- ============================================================================
-- Sandbox Account Seed Data
-- ============================================================================
-- This script populates a new sandbox account with tutorial data for an
-- expanded sock manufacturing company. It covers products, materials, parts,
-- BOMs, production steps, departments, machines, inventory, and more.
--
-- Assumptions:
--   - All static/global lookup tables already exist (unit_type, item_type,
--     product_type, item_category_type, scanning_station_type, color,
--     storage_location_type, action_type, label_type, label_size).
--   - The account record itself is created separately before running this.
--
-- ID Convention: {prefix}_{uuid_hex_12} (generated at runtime via UUID())
-- ============================================================================

-- ============================================================================
-- VARIABLE REFERENCE MAP
-- ============================================================================
-- All IDs are generated at runtime via UUID(). Each {prefix}_seed{N} listed
-- below maps to the MySQL user variable @{prefix}{N}.
-- Example: un_seed000000001 → @un1, it_seed000000021 → @it21
--
-- UNITS (@un...)
--   @un1  Each
--   @un2  Pair
--   @un3  Dozen
--   @un4  Dollar
--   un_seed000000005  Hour
--   un_seed000000006  Day
--   un_seed000000007  Minute
--   un_seed000000008  Pound
--   un_seed000000009  Gram
--
-- UNIT GROUPS (ungp_seed...)
--   ungp_seed000000001  Socks Units          (base: Pair)
--   ungp_seed000000002  Sellable Socks       (base: Pair)
--   ungp_seed000000003  Yarn Units           (base: Pound)
--   ungp_seed000000004  Chemical Units       (base: Gram)
--   ungp_seed000000005  Each Units           (base: Each)
--   ungp_seed000000006  Time Units           (base: Hour)
--   ungp_seed000000007  Currency Units       (base: Dollar)
--
-- UNIT CONVERSIONS (ungpun_seed...)
--   ungpun_seed000000001 .. ungpun_seed000000015
--
-- PROPERTIES (prop_seed...)
--   prop_seed000000001  Color
--   prop_seed000000002  Size
--   prop_seed000000003  Twist
--   prop_seed000000004  Denier
--   prop_seed000000005  Fiber Content
--
-- ATTRIBUTES (attr_seed...)
--   attr_seed000000001  White         (Color)
--   attr_seed000000002  Black         (Color)
--   attr_seed000000003  Navy          (Color)
--   attr_seed000000004  Heather Gray  (Color)
--   attr_seed000000005  Beige         (Color)
--   attr_seed000000006  S             (Size)
--   attr_seed000000007  M             (Size)
--   attr_seed000000008  L             (Size)
--   attr_seed000000009  XL            (Size)
--   attr_seed000000010  S-Twist       (Twist)
--   attr_seed000000011  Z-Twist       (Twist)
--   attr_seed000000012  40D           (Denier)
--   attr_seed000000013  70D           (Denier)
--   attr_seed000000014  Cotton        (Fiber)
--   attr_seed000000015  Nylon         (Fiber)
--   attr_seed000000016  Spandex       (Fiber)
--
-- CATEGORIES (itmcat_seed...)
--   itmcat_seed000000001  Crew Socks    (product)
--   itmcat_seed000000002  Ankle Socks   (product)
--   itmcat_seed000000003  Yarn          (material)
--   itmcat_seed000000004  Dyes          (material)
--   itmcat_seed000000005  Chemicals     (material)
--   itmcat_seed000000006  Packaging     (material)
--   itmcat_seed000000007  Shipping      (product)
--   itmcat_seed000000008  Credit        (product)
--
-- PRODUCT LINES (pdln_seed...)
--   pdln_seed000000001  Crew Collection
--   pdln_seed000000002  Ankle Collection
--   pdln_seed000000003  Athletic
--   pdln_seed000000004  Shipping
--   pdln_seed000000005  Credit
--
-- ITEMS (it_seed...)
--   -- Products (Crew Socks) it_seed000000001..009
--   it_seed000000001  FG-CRW-S-WHT   Crew Sock Small White
--   it_seed000000002  FG-CRW-S-BLK   Crew Sock Small Black
--   it_seed000000003  FG-CRW-S-NAV   Crew Sock Small Navy
--   it_seed000000004  FG-CRW-M-WHT   Crew Sock Medium White
--   it_seed000000005  FG-CRW-M-BLK   Crew Sock Medium Black
--   it_seed000000006  FG-CRW-M-NAV   Crew Sock Medium Navy
--   it_seed000000007  FG-CRW-L-WHT   Crew Sock Large White
--   it_seed000000008  FG-CRW-L-BLK   Crew Sock Large Black
--   it_seed000000009  FG-CRW-L-NAV   Crew Sock Large Navy
--   -- Products (Ankle Socks) it_seed000000010..018
--   it_seed000000010  FG-ANK-S-WHT   Ankle Sock Small White
--   it_seed000000011  FG-ANK-S-BLK   Ankle Sock Small Black
--   it_seed000000012  FG-ANK-S-NAV   Ankle Sock Small Navy
--   it_seed000000013  FG-ANK-M-WHT   Ankle Sock Medium White
--   it_seed000000014  FG-ANK-M-BLK   Ankle Sock Medium Black
--   it_seed000000015  FG-ANK-M-NAV   Ankle Sock Medium Navy
--   it_seed000000016  FG-ANK-L-WHT   Ankle Sock Large White
--   it_seed000000017  FG-ANK-L-BLK   Ankle Sock Large Black
--   it_seed000000018  FG-ANK-L-NAV   Ankle Sock Large Navy
--   -- Products (System) it_seed000000019..020
--   it_seed000000019  Shipping        Shipping
--   it_seed000000020  Credit      Credit
--   -- Materials it_seed000000021..032
--   it_seed000000021  RM-YRN-COT70   Cotton Yarn 70D
--   it_seed000000022  RM-YRN-COT40   Cotton Yarn 40D
--   it_seed000000023  RM-YRN-NYL40   Nylon Yarn 40D
--   it_seed000000024  RM-ELS-SPX     Spandex Elastic Thread
--   it_seed000000025  RM-DYE-BLK     Black Reactive Dye
--   it_seed000000026  RM-DYE-NAV     Navy Reactive Dye
--   it_seed000000027  RM-DYE-BGE     Beige Reactive Dye
--   it_seed000000028  RM-CHM-SOF     Fabric Softener
--   it_seed000000029  RM-CHM-DET     Industrial Detergent
--   it_seed000000030  RM-PKG-BX12    Corrugated Box 12-Pack
--   it_seed000000031  RM-PKG-BX06    Corrugated Box 6-Pack
--   it_seed000000032  RM-PKG-LBL     Product Label
--   -- Parts (WIP) it_seed000000033..047
--   it_seed000000033  WIP-KNT-S      Knitted Sock Blank Small
--   it_seed000000034  WIP-KNT-M      Knitted Sock Blank Medium
--   it_seed000000035  WIP-KNT-L      Knitted Sock Blank Large
--   it_seed000000036  WIP-LNK-S      Linked Sock Small
--   it_seed000000037  WIP-LNK-M      Linked Sock Medium
--   it_seed000000038  WIP-LNK-L      Linked Sock Large
--   it_seed000000039  WIP-WSH-S      Scoured Sock Small
--   it_seed000000040  WIP-WSH-M      Scoured Sock Medium
--   it_seed000000041  WIP-WSH-L      Scoured Sock Large
--   it_seed000000042  WIP-DYE-S-BLK  Dyed Sock Small Black
--   it_seed000000043  WIP-DYE-S-NAV  Dyed Sock Small Navy
--   it_seed000000044  WIP-DYE-M-BLK  Dyed Sock Medium Black
--   it_seed000000045  WIP-DYE-M-NAV  Dyed Sock Medium Navy
--   it_seed000000046  WIP-DYE-L-BLK  Dyed Sock Large Black
--   it_seed000000047  WIP-DYE-L-NAV  Dyed Sock Large Navy
--
-- PRODUCTS (pd_seed...)
--   pd_seed000000001..009  Crew Socks (S/M/L x WHT/BLK/NAV)
--   pd_seed000000010..018  Ankle Socks (S/M/L x WHT/BLK/NAV)
--   pd_seed000000019       Shipping
--   pd_seed000000020       Credit
--
-- MATERIALS (mat_seed...)
--   mat_seed000000001..012  (12 raw materials)
--
-- PARTS (prt_seed...)
--   prt_seed000000001..015  (15 WIP parts)
--
-- RATES (rt_seed...)
--   rt_seed000000001..060   Item rates (3 per item x 20 products)
--   rt_seed000000061..096   Item rates (3 per item x 12 materials)
--   rt_seed000000097..141   Item rates (3 per item x 15 parts)
--   rt_seed000000142..195   Production step rates (3 per step x 18 steps)
--
-- QUANTITIES (qty_seed...)
--   qty_seed000000001..024  Material quantities (2 per material x 12)
--   qty_seed000000025..099  Consumption/production quantities
--
-- DEPARTMENTS (dept_seed...)
--   dept_seed000000001  Knitting
--   dept_seed000000002  Linking
--   dept_seed000000003  Wet Processing
--   dept_seed000000004  Dyeing
--   dept_seed000000005  Finishing
--   dept_seed000000006  Packaging
--
-- STORAGE LOCATIONS (stloc_seed...)
--   stloc_seed000000001  Main Building
--   stloc_seed000000002  Knitting Floor
--   stloc_seed000000003  Linking Floor
--   stloc_seed000000004  Wet Processing Area
--   stloc_seed000000005  Dye House
--   stloc_seed000000006  Finishing Area
--   stloc_seed000000007  Pack & Ship
--   stloc_seed000000008  Raw Materials Storage
--
-- SCANNING STATIONS (scst_seed...)
--   scst_seed000000001..006  (1 per department)
--
-- MACHINES (mach_seed...)
--   mach_seed000000001..012  (2 per department)
--
-- PRODUCTION STEPS (pdst_seed...)
--   pdst_seed000000001..003  Knit S/M/L
--   pdst_seed000000004..006  Link S/M/L
--   pdst_seed000000007..009  Wash S/M/L
--   pdst_seed000000010..015  Dye S/M/L x BLK/NAV
--   pdst_seed000000016..018  Pack S/M/L
--
-- INVENTORY LOGS (invlog_seed...)
--   invlog_seed000000001..047  (1 per item)
--
-- INVENTORY CHANGE LOGS (invcl_seed...)
--   invcl_seed000000001..047  (1 per item)
--
-- PRODUCTIONS – PACK FIX (prod_seed...)
--   @prod16  Pack S -> FG-CRW-S-WHT
--   @prod17  Pack M -> FG-CRW-M-WHT
--   @prod18  Pack L -> FG-CRW-L-WHT
--   @qty212..214  Pack production quantities (1 pair each)
--
--
-- ACCOUNT GROUP (acgrp_seed...)
--   @acgrp1  Wholesale
--
-- CUSTOMER ACCOUNTS (cust_seed...)
--   @cust1  Global Manufacturing Solutions (CUST-001)
--   @cust2  Pacific Coast Distributors     (CUST-002)
--   @cust3  Northeast Medical Supplies     (CUST-003)
--
-- GEOLOCATIONS (geo_seed...)
--   @geo1..6  (2 per customer: billing + shipping)
--
-- ADDRESSES (caddr_seed...)
--   @caddr1..6  (2 per customer: billing + shipping)
--
-- ACCOUNT ADDRESSES (acadr_seed...)
--   @acadr1..6  (linking customer accounts to addresses)
--
-- ACCOUNT RELATIONS (acrel_seed...)
--   @acrel1..3  (sandbox -> customer, role=customer)
--
-- SALES ORDERS (so_seed...)
--   @so1  EST-001  estimate   Cust 1
--   @so2  ORD-001  issued     Cust 1
--   @so3  ORD-002  issued     Cust 2
--   @so4  ORD-003  issued     Cust 2  (packed shipment)
--   @so5  ORD-004  fulfilled  Cust 1  (shipped + invoiced)
--   @so6  ORD-005  fulfilled  Cust 3  (shipped + invoiced)
--
-- SALES ORDER LINES (sol_seed...)
--   @sol1..18  (3 per order: 2 product + 1 shipping)
--
-- ORDER LINE RATES (rt_seed...)
--   @rt196..213  unit_price rates (18, one per line)
--   @rt214..225  unit_cost rates  (12, product lines only)
--
-- ORDER LINE QUANTITIES (qty_seed...)
--   @qty215..232  (18, one per line)
--
-- PICKS (pick_seed...)
--   @pick1  PICK-001  ORD-001  open
--   @pick2  PICK-002  ORD-002  open
--   @pick3  PICK-003  ORD-003  finished+packed
--   @pick4  PICK-004  ORD-004  finished+packed
--   @pick5  PICK-005  ORD-005  finished+packed
--
-- PICK LINES (pkl_seed...)
--   @pkl1..10   (2 per pick)
--   @qty233..242  pick line quantities
--
-- SHIPMENTS (shp_seed...)
--   @shp1  SHP-001  ORD-003  packed
--   @shp2  SHP-002  ORD-004  shipped
--   @shp3  SHP-003  ORD-005  shipped
--
-- SHIPMENT LINES (shpl_seed...)
--   @shpl1..6   (2 per shipment)
--   @qty243..248  shipment line quantities
--
-- SHIPPING CASES (shpc_seed...)
--   @shpc1  SHP-001 CASE-001  (no tracking)
--   @shpc2  SHP-002 CASE-001  (FEDEX tracking)
--   @shpc3  SHP-003 CASE-001  (FEDEX tracking)
--   @shpc4  SHP-003 CASE-002  (FEDEX tracking)
--   @qty249..256  case freight amounts + weights
--
-- INVOICES (inv_seed...)
--   @inv1  INV-001  ORD-004
--   @inv2  INV-002  ORD-005
--
-- INVOICE LINES (invl_seed...)
--   @invl1..4   (2 per invoice)
--   @qty257..260  invoice line quantities
--
-- LOOKUPS OF ROWS CREATED WITH THE ACCOUNT
--   @acus1    the sandbox admin account_user
--   @ownadr1  the account's own billing address
--   @ownadr2  the account's own shipping address
--
-- PRODUCTION SHIFTS (pnsf_seed...)
--   @pnsf1..2  Day, Swing
--
-- SUPPLIERS
--   @supp1    Carolina Yarn Mills   (SUP-001)
--   @supp2    Atlantic Packaging Co (SUP-002)
--   @sgeo1..2, @saddr1..2, @sacadr1..2  geolocation, address, account_address
--   @acrel4..5  sandbox -> supplier, role=supplier
--   @suml1..4   supplier part numbers for 3 yarns and 1 carton
--
-- PURCHASE ORDERS (share the sales_order table, type purchase_order)
--   @po1  PO-001  fulfilled, received in full
--   @po2  PO-002  issued, still open
--   @poln1..4   lines, @qpo1..4 quantities, @rtpo1..4 prices
--
-- RECEIVING + INBOUND
--   @rcor1..2     RCV-001 closed, RCV-002 open
--   @rcorln1..4   receiving lines, first two stocked
--   @dv1..2       DLV-001, DLV-002
--   @dvln1..4     delivery lines, @qdv1..4 quantities, @rtdv1..4 unit costs
--   @inrp1..4     inventory receipts, the costed layers the deliveries created
--
-- PRODUCTION RUNS + BATCHES
--   @pnrn1..3   PR-1001 small, PR-1002 medium, PR-1003 large
--   @bt1..12    4 batches per run: knit, link, wash, pack
--   @qbt1..12   batch quantities in pairs
--
-- HISTORICAL DEMAND (the trailing-12 window the schedule plans from)
--   @hso1..3    ORD-H001 7 months back, ORD-H002 4 months, ORD-H003 2 months
--   @hsol1..6   lines, @qhso1..6 quantities, @rthso1..6 prices, @rthsc1..6 costs
--   @dho1..11   ORD-D01..D11, one per complete month solidly inside the demand window
--   @dhq1..33   line quantities (3 per order), @dhr1..33 per-line unit prices
--
-- MACHINE DOWNTIME (mcdt_seed...)
--   @mcdt1..5   breakdown, changeover, material shortage, minor stop, quality hold
--
-- SALES TARGETS + CUSTOMER PRICING
--   @ta1..2     quarter and year revenue targets, @qta1..2 amounts
--   @acpr1..3   negotiated product line prices, @rtacpr1..3 rates
--
-- PRODUCTION SCHEDULE SETTINGS
--   @acpnscsd1     knitting named as the planning constraint, scheduling enabled
--   @pnscrrsd1..3  the three knitting machines, written explicitly
--
-- ============================================================================


-- ============================================================================
-- ID GENERATION
-- ============================================================================
-- Each execution generates unique IDs using MySQL UUID() to ensure sandbox
-- isolation. Variables are referenced throughout the script.
-- ============================================================================

SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;

SET @un1 = (SELECT id FROM unit WHERE name = 'Each' AND account_id = '@account_id');
SET @un2 = CONCAT('un_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @un3 = CONCAT('un_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @un4 = (SELECT id FROM unit WHERE name = 'Dollar' AND account_id = '@account_id');
SET @un5 = CONCAT('un_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @un6 = (SELECT id FROM unit WHERE name = 'Day' AND account_id = '@account_id');
SET @un7 = CONCAT('un_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @un8 = CONCAT('un_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @un9 = CONCAT('un_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @ungp1 = CONCAT('ungp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ungp2 = CONCAT('ungp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ungp3 = CONCAT('ungp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ungp4 = CONCAT('ungp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ungp5 = (SELECT id FROM unit_group WHERE name = 'Each Units' AND account_id = '@account_id');
SET @ungp6 = CONCAT('ungp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ungp7 = CONCAT('ungp_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @ungpun1 = CONCAT('ungpun_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ungpun2 = CONCAT('ungpun_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ungpun3 = CONCAT('ungpun_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ungpun4 = CONCAT('ungpun_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ungpun5 = CONCAT('ungpun_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ungpun6 = CONCAT('ungpun_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ungpun7 = CONCAT('ungpun_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ungpun9 = CONCAT('ungpun_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ungpun10 = CONCAT('ungpun_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ungpun11 = CONCAT('ungpun_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ungpun12 = CONCAT('ungpun_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @prop1 = CONCAT('pp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prop2 = CONCAT('pp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prop3 = CONCAT('pp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prop4 = CONCAT('pp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prop5 = CONCAT('pp_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @attr1 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @attr2 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @attr3 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @attr4 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @attr5 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @attr6 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @attr7 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @attr8 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @attr9 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @attr10 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @attr11 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @attr12 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @attr13 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @attr14 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @attr15 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @attr16 = CONCAT('at_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @itmcat1 = CONCAT('itcg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @itmcat2 = CONCAT('itcg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @itmcat3 = CONCAT('itcg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @itmcat4 = CONCAT('itcg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @itmcat5 = CONCAT('itcg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @itmcat6 = CONCAT('itcg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @itmcat7 = (SELECT id FROM item_category WHERE name = 'Shipping' AND account_id = '@account_id');
SET @itmcat8 = (SELECT id FROM item_category WHERE name = 'Credit' AND account_id = '@account_id');

SET @pdln1 = CONCAT('pdln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdln2 = CONCAT('pdln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdln3 = CONCAT('pdln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdln4 = (SELECT id FROM product_line WHERE name = 'Shipping' AND account_id = '@account_id');
SET @pdln5 = (SELECT id FROM product_line WHERE name = 'Credit' AND account_id = '@account_id');

SET @rt1 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt2 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt3 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt4 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt5 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt6 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt7 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt8 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt9 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt10 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt11 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt12 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt13 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt14 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt15 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt16 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt17 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt18 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt19 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt20 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt21 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt22 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt23 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt24 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt25 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt26 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt27 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt28 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt29 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt30 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt31 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt32 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt33 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt34 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt35 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt36 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt37 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt38 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt39 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt40 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt41 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt42 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt43 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt44 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt45 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt46 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt47 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt48 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt49 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt50 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt51 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt52 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt53 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt54 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
-- rt55-rt60 and it19/it20/pd19/pd20 are looked up below (system products created at account creation)
SET @rt61 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt62 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt63 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt64 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt65 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt66 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt67 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt68 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt69 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt70 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt71 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt72 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt73 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt74 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt75 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt76 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt77 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt78 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt79 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt80 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt81 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt82 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt83 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt84 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt85 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt86 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt87 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt88 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt89 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt90 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt91 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt92 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt93 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt94 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt95 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt96 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt97 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt98 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt99 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt100 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt101 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt102 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt103 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt104 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt105 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt106 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt107 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt108 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt109 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt110 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt111 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt112 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt113 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt114 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt115 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt116 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt117 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt118 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt119 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt120 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt121 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt122 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt123 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt124 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt125 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt126 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt127 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt128 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt129 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt130 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt131 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt132 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt133 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt134 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt135 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt136 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt137 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt138 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt139 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt140 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt141 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt142 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt143 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt144 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt145 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt146 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt147 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt148 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt149 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt150 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt151 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt152 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt153 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt154 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt155 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt156 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt157 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt158 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt159 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt160 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt161 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt162 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt163 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt164 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt165 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt166 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt167 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt168 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt169 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt170 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt171 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt172 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt173 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt174 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt175 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt176 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt177 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt178 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt179 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt180 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt181 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt182 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt183 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt184 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt185 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt186 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt187 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt188 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt189 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt190 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt191 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt192 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt193 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt194 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt195 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @qty1 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty2 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty3 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty4 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty5 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty6 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty7 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty8 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty9 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty10 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty11 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty12 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty13 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty14 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty15 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty16 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty17 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty18 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty19 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty20 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty21 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty22 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty23 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty24 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty25 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty26 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty27 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty28 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty29 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty30 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty31 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty32 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty33 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty34 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty35 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty36 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty37 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty38 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty39 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty40 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty41 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty42 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty43 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty44 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty45 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty46 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty47 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty48 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty49 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty50 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty51 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty52 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty53 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty54 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty55 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty56 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty57 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty58 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty59 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty60 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty61 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty62 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty63 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty64 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty65 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty66 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty67 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty68 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty69 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty70 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty71 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty72 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty73 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty74 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty75 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty76 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty77 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty78 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty79 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty80 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty81 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty82 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty83 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty84 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty85 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty86 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty87 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty88 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty89 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty90 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty91 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty92 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty93 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty94 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty95 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty96 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty97 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty98 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty99 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty100 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty101 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty102 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty103 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty104 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty105 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty106 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty107 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty108 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty109 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty110 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty111 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty112 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty113 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty114 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty115 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty116 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty117 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty118 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty119 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty120 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty121 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty122 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty123 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty124 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty125 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty126 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty127 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty128 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty129 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty130 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty131 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty132 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty133 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty134 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty135 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty136 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty137 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty138 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty139 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty140 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty141 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty142 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty143 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty144 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty145 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty146 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty147 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty148 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty149 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty150 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty151 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty152 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty153 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty154 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty155 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty156 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty157 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty158 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty159 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty160 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty161 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty162 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty163 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty164 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty165 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty166 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty167 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty168 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty169 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty170 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty171 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty172 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty173 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty174 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty175 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty176 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty177 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty178 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty179 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty180 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty181 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty182 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty183 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty184 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty185 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty186 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty187 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty188 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty189 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty190 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty191 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty192 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty193 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty194 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty195 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty196 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty197 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty198 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty199 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty200 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty201 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty202 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty203 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty204 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty205 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty206 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty207 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty208 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty209 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty210 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty211 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @it1 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it2 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it3 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it4 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it5 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it6 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it7 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it8 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it9 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it10 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it11 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it12 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it13 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it14 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it15 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it16 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it17 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it18 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it19 = (SELECT id FROM item WHERE sku = 'Shipping' AND account_id = '@account_id');
SET @it20 = (SELECT id FROM item WHERE sku = 'Credit' AND account_id = '@account_id');
SET @rt55 = (SELECT unit_value_id FROM item WHERE id = @it19);
SET @rt56 = (SELECT unit_cost_id FROM item WHERE id = @it19);
SET @rt57 = (SELECT burn_rate_id FROM item WHERE id = @it19);
SET @rt58 = (SELECT unit_value_id FROM item WHERE id = @it20);
SET @rt59 = (SELECT unit_cost_id FROM item WHERE id = @it20);
SET @rt60 = (SELECT burn_rate_id FROM item WHERE id = @it20);
SET @it21 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it22 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it23 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it24 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it25 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it26 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it27 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it28 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it29 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it30 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it31 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it32 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it33 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it34 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it35 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it36 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it37 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it38 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it39 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it40 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it41 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it42 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it43 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it44 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it45 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it46 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @it47 = CONCAT('it_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @pd1 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd2 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd3 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd4 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd5 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd6 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd7 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd8 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd9 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd10 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd11 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd12 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd13 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd14 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd15 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd16 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd17 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd18 = CONCAT('pd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pd19 = (SELECT p.id FROM product p JOIN item i ON p.item_id = i.id WHERE i.sku = 'Shipping' AND i.account_id = '@account_id');
SET @pd20 = (SELECT p.id FROM product p JOIN item i ON p.item_id = i.id WHERE i.sku = 'Credit' AND i.account_id = '@account_id');

SET @mat1 = CONCAT('ml_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mat2 = CONCAT('ml_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mat3 = CONCAT('ml_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mat4 = CONCAT('ml_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mat5 = CONCAT('ml_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mat6 = CONCAT('ml_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mat7 = CONCAT('ml_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mat8 = CONCAT('ml_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mat9 = CONCAT('ml_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mat10 = CONCAT('ml_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mat11 = CONCAT('ml_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mat12 = CONCAT('ml_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @prt1 = CONCAT('pt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prt2 = CONCAT('pt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prt3 = CONCAT('pt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prt4 = CONCAT('pt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prt5 = CONCAT('pt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prt6 = CONCAT('pt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prt7 = CONCAT('pt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prt8 = CONCAT('pt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prt9 = CONCAT('pt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prt10 = CONCAT('pt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prt11 = CONCAT('pt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prt12 = CONCAT('pt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prt13 = CONCAT('pt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prt14 = CONCAT('pt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prt15 = CONCAT('pt_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @dept1 = CONCAT('dp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dept2 = CONCAT('dp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dept3 = CONCAT('dp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dept4 = CONCAT('dp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dept5 = CONCAT('dp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dept6 = CONCAT('dp_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @stloc1 = CONCAT('lc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @stloc2 = CONCAT('lc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @stloc3 = CONCAT('lc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @stloc4 = CONCAT('lc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @stloc5 = CONCAT('lc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @stloc6 = CONCAT('lc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @stloc7 = CONCAT('lc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @stloc8 = CONCAT('lc_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @scst1 = CONCAT('sgsn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @scst2 = CONCAT('sgsn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @scst3 = CONCAT('sgsn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @scst4 = CONCAT('sgsn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @scst5 = CONCAT('sgsn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @scst6 = CONCAT('sgsn_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @mach1 = CONCAT('mc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mach2 = CONCAT('mc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mach3 = CONCAT('mc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mach4 = CONCAT('mc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mach5 = CONCAT('mc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mach6 = CONCAT('mc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mach7 = CONCAT('mc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mach8 = CONCAT('mc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mach9 = CONCAT('mc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mach10 = CONCAT('mc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mach11 = CONCAT('mc_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mach12 = CONCAT('mc_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @pdst1 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst2 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst3 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst4 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst5 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst6 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst7 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst8 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst9 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst10 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst11 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst12 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst13 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst14 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst15 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst16 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst17 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst18 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @cons1 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons2 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons3 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons4 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons5 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons6 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons7 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons8 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons9 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons10 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons11 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons12 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons13 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons14 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons15 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons16 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons17 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons18 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons19 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons20 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons21 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons22 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons23 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons24 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons25 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons26 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons27 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons28 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons29 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons30 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons31 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons32 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons33 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons34 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons35 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons36 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons37 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons38 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons39 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @prod1 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod2 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod3 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod4 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod5 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod6 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod7 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod8 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod9 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod10 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod11 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod12 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod13 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod14 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod15 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @invlog1 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog2 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog3 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog4 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog5 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog6 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog7 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog8 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog9 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog10 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog11 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog12 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog13 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog14 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog15 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog16 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog17 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog18 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog19 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog20 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog21 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog22 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog23 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog24 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog25 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog26 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog27 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog28 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog29 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog30 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog31 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog32 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog33 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog34 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog35 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog36 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog37 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog38 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog39 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog40 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog41 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog42 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog43 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog44 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog45 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog46 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invlog47 = CONCAT('inlg_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @invcl1 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl2 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl3 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl4 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl5 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl6 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl7 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl8 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl9 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl10 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl11 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl12 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl13 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl14 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl15 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl16 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl17 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl18 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl19 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl20 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl21 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl22 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl23 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl24 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl25 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl26 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl27 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl28 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl29 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl30 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl31 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl32 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl33 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl34 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl35 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl36 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl37 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl38 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl39 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl40 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl41 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl42 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl43 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl44 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl45 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl46 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invcl47 = CONCAT('inchlg_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Pack production fix
SET @prod16 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod17 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod18 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty212 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty213 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty214 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Pack step split: 9 pack steps instead of 3 (one per size x color)
SET @pdst19 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst20 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst21 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst22 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst23 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pdst24 = CONCAT('pnst_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt226 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt227 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt228 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt229 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt230 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt231 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt232 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt233 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt234 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt235 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt236 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt237 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt238 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt239 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt240 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt241 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt242 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt243 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons40 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons41 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons42 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons43 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons44 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons45 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons46 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons47 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons48 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons49 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons50 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons51 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons52 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons53 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons54 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons55 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons56 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons57 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons58 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons59 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cons60 = CONCAT('cp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod19 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod20 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod21 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod22 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod23 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @prod24 = CONCAT('pn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty261 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty262 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty263 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty264 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty265 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty266 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty267 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty268 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty269 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty270 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty271 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty272 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty273 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty274 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty275 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty276 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty277 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty278 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty279 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty280 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty281 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty282 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty283 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty284 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty285 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty286 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty287 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty288 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty289 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty290 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty291 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty292 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty293 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty294 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty295 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty296 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty297 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty298 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty299 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty300 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty301 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty302 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty303 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty304 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty305 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty306 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty307 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty308 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Account group
SET @acgrp1 = CONCAT('acgp_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Customer accounts
SET @cust1 = CONCAT('ac_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cust2 = CONCAT('ac_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @cust3 = CONCAT('ac_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Geolocations (2 per customer: billing + shipping)
SET @geo1 = CONCAT('gl_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @geo2 = CONCAT('gl_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @geo3 = CONCAT('gl_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @geo4 = CONCAT('gl_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @geo5 = CONCAT('gl_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @geo6 = CONCAT('gl_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Customer addresses
SET @caddr1 = CONCAT('acad_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @caddr2 = CONCAT('acad_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @caddr3 = CONCAT('acad_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @caddr4 = CONCAT('acad_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @caddr5 = CONCAT('acad_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @caddr6 = CONCAT('acad_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Account addresses
SET @acadr1 = CONCAT('acad_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @acadr2 = CONCAT('acad_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @acadr3 = CONCAT('acad_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @acadr4 = CONCAT('acad_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @acadr5 = CONCAT('acad_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @acadr6 = CONCAT('acad_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Payment terms
SET @pytm1 = CONCAT('pytm_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pytm2 = CONCAT('pytm_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Shipping terms
SET @shtm1 = CONCAT('shtm_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Account relations
SET @acrel1 = CONCAT('acre_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @acrel2 = CONCAT('acre_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @acrel3 = CONCAT('acre_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Sales orders
SET @so1 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @so2 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @so3 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @so4 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @so5 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @so6 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Sales order lines (3 per order x 6 = 18)
SET @sol1 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol2 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol3 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol4 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol5 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol6 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol7 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol8 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol9 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol10 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol11 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol12 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol13 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol14 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol15 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol16 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol17 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sol18 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Order line quantities (18)
SET @qty215 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty216 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty217 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty218 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty219 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty220 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty221 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty222 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty223 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty224 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty225 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty226 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty227 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty228 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty229 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty230 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty231 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty232 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Order line unit prices (18 rates) and unit costs (12 rates)
SET @rt196 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt197 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt198 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt199 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt200 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt201 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt202 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt203 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt204 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt205 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt206 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt207 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt208 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt209 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt210 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt211 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt212 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt213 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt214 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt215 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt216 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt217 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt218 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt219 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt220 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt221 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt222 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt223 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt224 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rt225 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Picks
SET @pick1 = CONCAT('pk_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pick2 = CONCAT('pk_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pick3 = CONCAT('pk_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pick4 = CONCAT('pk_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pick5 = CONCAT('pk_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Pick lines (2 per pick x 5 = 10)
SET @pkl1 = CONCAT('pkln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pkl2 = CONCAT('pkln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pkl3 = CONCAT('pkln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pkl4 = CONCAT('pkln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pkl5 = CONCAT('pkln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pkl6 = CONCAT('pkln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pkl7 = CONCAT('pkln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pkl8 = CONCAT('pkln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pkl9 = CONCAT('pkln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pkl10 = CONCAT('pkln_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Pick line quantities (10)
SET @qty233 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty234 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty235 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty236 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty237 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty238 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty239 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty240 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty241 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty242 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Shipments
SET @shp1 = CONCAT('sh_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @shp2 = CONCAT('sh_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @shp3 = CONCAT('sh_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Shipment lines (2 per shipment x 3 = 6)
SET @shpl1 = CONCAT('shln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @shpl2 = CONCAT('shln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @shpl3 = CONCAT('shln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @shpl4 = CONCAT('shln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @shpl5 = CONCAT('shln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @shpl6 = CONCAT('shln_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Shipment line quantities (6)
SET @qty243 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty244 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty245 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty246 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty247 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty248 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Shipping cases
SET @shpc1 = CONCAT('shcs_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @shpc2 = CONCAT('shcs_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @shpc3 = CONCAT('shcs_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @shpc4 = CONCAT('shcs_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Shipping case quantities (freight_amount + freight_weight per case = 8)
SET @qty249 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty250 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty251 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty252 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty253 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty254 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty255 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty256 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Invoices
SET @inv1 = CONCAT('iv_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @inv2 = CONCAT('iv_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Invoice lines (2 per invoice x 2 = 4)
SET @invl1 = CONCAT('ivln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invl2 = CONCAT('ivln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invl3 = CONCAT('ivln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @invl4 = CONCAT('ivln_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Invoice line quantities (4)
SET @qty257 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty258 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty259 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qty260 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- The sandbox admin, created with the account before this script runs. Owns the seeded
-- production runs, downtime reports and sales targets, all of which require a person.
SET @acus1 = (SELECT id FROM account_user WHERE account_id = '@account_id' ORDER BY created_at LIMIT 1);

-- The sandbox's own business address, for purchase orders it is the ship-to.
SET @ownadr1 = (SELECT default_billing_address_id FROM account WHERE id = '@account_id');
SET @ownadr2 = (SELECT default_shipping_address_id FROM account WHERE id = '@account_id');

-- Third knitting machine, so each knit size has one (see SECTION 20)
SET @mach13 = CONCAT('mc_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Production shifts (2)
SET @pnsf1 = CONCAT('pnsf_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pnsf2 = CONCAT('pnsf_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Supplier accounts, geolocations, addresses, links and relations (2 each)
SET @supp1 = CONCAT('ac_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @supp2 = CONCAT('ac_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sgeo1 = CONCAT('gl_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sgeo2 = CONCAT('gl_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @saddr1 = CONCAT('ad_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @saddr2 = CONCAT('ad_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sacadr1 = CONCAT('acad_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @sacadr2 = CONCAT('acad_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @acrel4 = CONCAT('acre_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @acrel5 = CONCAT('acre_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Supplier materials (4)
SET @suml1 = CONCAT('suml_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @suml2 = CONCAT('suml_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @suml3 = CONCAT('suml_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @suml4 = CONCAT('suml_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Purchase orders (2) + lines (4) + quantities + prices
SET @po1 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @po2 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @poln1 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @poln2 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @poln3 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @poln4 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qpo1 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qpo2 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qpo3 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qpo4 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rtpo1 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rtpo2 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rtpo3 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rtpo4 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Receiving orders (2) + lines (4)
SET @rcor1 = CONCAT('rcor_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rcor2 = CONCAT('rcor_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rcorln1 = CONCAT('rcorln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rcorln2 = CONCAT('rcorln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rcorln3 = CONCAT('rcorln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rcorln4 = CONCAT('rcorln_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Deliveries (2) + lines (4) + quantities + unit costs
SET @dv1 = CONCAT('dv_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dv2 = CONCAT('dv_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dvln1 = CONCAT('dvln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dvln2 = CONCAT('dvln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dvln3 = CONCAT('dvln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dvln4 = CONCAT('dvln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qdv1 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qdv2 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qdv3 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qdv4 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rtdv1 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rtdv2 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rtdv3 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rtdv4 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Inventory receipts (4)
SET @inrp1 = CONCAT('inrp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @inrp2 = CONCAT('inrp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @inrp3 = CONCAT('inrp_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @inrp4 = CONCAT('inrp_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Production runs (3) + batches (12) + batch quantities (12)
SET @pnrn1 = CONCAT('pnrn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pnrn2 = CONCAT('pnrn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pnrn3 = CONCAT('pnrn_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @bt1 = CONCAT('bt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @bt2 = CONCAT('bt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @bt3 = CONCAT('bt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @bt4 = CONCAT('bt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @bt5 = CONCAT('bt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @bt6 = CONCAT('bt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @bt7 = CONCAT('bt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @bt8 = CONCAT('bt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @bt9 = CONCAT('bt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @bt10 = CONCAT('bt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @bt11 = CONCAT('bt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @bt12 = CONCAT('bt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qbt1 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qbt2 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qbt3 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qbt4 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qbt5 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qbt6 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qbt7 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qbt8 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qbt9 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qbt10 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qbt11 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qbt12 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Historical demand orders (3) + lines (6) + quantities + prices + costs
SET @hso1 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @hso2 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @hso3 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @hsol1 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @hsol2 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @hsol3 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @hsol4 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @hsol5 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @hsol6 = CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qhso1 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qhso2 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qhso3 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qhso4 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qhso5 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qhso6 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rthso1 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rthso2 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rthso3 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rthso4 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rthso5 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rthso6 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rthsc1 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rthsc2 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rthsc3 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rthsc4 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rthsc5 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rthsc6 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Machine downtime events (5)
SET @mcdt1 = CONCAT('mcdt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mcdt2 = CONCAT('mcdt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mcdt3 = CONCAT('mcdt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mcdt4 = CONCAT('mcdt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @mcdt5 = CONCAT('mcdt_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Sales targets (2) + target amounts (2)
SET @ta1 = CONCAT('ta_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @ta2 = CONCAT('ta_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qta1 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @qta2 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Customer prices (3) + their rates (3)
SET @acpr1 = CONCAT('acpr_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @acpr2 = CONCAT('acpr_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @acpr3 = CONCAT('acpr_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rtacpr1 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rtacpr2 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @rtacpr3 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));

-- Production schedule settings + per-machine resource settings (3)
SET @acpnscsd1 = CONCAT('acpnscsd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pnscrrsd1 = CONCAT('pnscrrsd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pnscrrsd2 = CONCAT('pnscrrsd_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @pnscrrsd3 = CONCAT('pnscrrsd_', LEFT(REPLACE(UUID(), '-', ''), 12));


-- ============================================================================
-- SECTION 1: UNITS
-- ============================================================================

INSERT INTO `unit` (`id`, `name`, `abbreviation`, `account_id`, `unit_dimension_code`, `ratio_numerator`, `ratio_denominator`, `offset_numerator`, `offset_denominator`, `is_base_unit`) VALUES
  (@un2, 'Pair',   'pr',  '@account_id', 'quantity', 2, 1, 0, 1, false),
  (@un3, 'Dozen',  'dz',  '@account_id', 'quantity', 12, 1, 0, 1, false),
  (@un5, 'Hour',   'hr',  '@account_id', 'time', 1, 1, 0, 1, true),
  (@un7, 'Minute', 'min', '@account_id', 'time', 1, 60, 0, 1, false),
  (@un8, 'Pound',  'lb',  '@account_id', 'mass', 453592, 1000, 0, 1, false),
  (@un9, 'Gram',   'g',   '@account_id', 'mass', 1, 1, 0, 1, true);


-- ============================================================================
-- SECTION 2: UNIT GROUPS
-- ============================================================================

INSERT INTO `unit_group` (`id`, `name`, `notes`, `base_unit_id`, `account_id`, `unit_type_code`) VALUES
  (@ungp1, 'Socks Units',    NULL, @un2, '@account_id', 'quantity'),
  (@ungp2, 'Sellable Socks', NULL, @un2, '@account_id', 'quantity'),
  (@ungp3, 'Yarn Units',     NULL, @un8, '@account_id', 'mass'),
  (@ungp4, 'Chemical Units',  NULL, @un9, '@account_id', 'mass'),
  (@ungp6, 'Time Units',     NULL, @un5, '@account_id', 'time'),
  (@ungp7, 'Currency Units',  NULL, @un4, '@account_id', 'currency');


-- ============================================================================
-- SECTION 3: UNIT CONVERSIONS (unit_group_unit)
-- ============================================================================

INSERT INTO `unit_group_unit` (`id`, `unit_group_id`, `unit_id`, `discount_percentage`, `is_visible`) VALUES
  -- Socks Units: Each, Pair, Dozen
  (@ungpun1, @ungp1, @un1, 0, true),
  (@ungpun2, @ungp1, @un2, 0, true),
  (@ungpun3, @ungp1, @un3, 0, true),
  -- Sellable Socks: Pair
  (@ungpun4, @ungp2, @un2, 0, true),
  -- Yarn Units: Pound
  (@ungpun5, @ungp3, @un8, 0, true),
  -- Chemical Units: Gram, Pound
  (@ungpun6, @ungp4, @un9, 0, true),
  (@ungpun7, @ungp4, @un8, 0, true),
  -- Time Units: Hour, Day, Minute
  (@ungpun9, @ungp6, @un5, 0, true),
  (@ungpun10, @ungp6, @un6, 0, true),
  (@ungpun11, @ungp6, @un7, 0, true),
  -- Currency Units: Dollar
  (@ungpun12, @ungp7, @un4, 0, true);


-- ============================================================================
-- SECTION 4: PROPERTIES
-- ============================================================================

INSERT INTO `property` (`id`, `name`, `is_public`, `account_id`) VALUES
  (@prop1, 'Color',         true, '@account_id'),
  (@prop2, 'Size',          true, '@account_id'),
  (@prop3, 'Twist',         true, '@account_id'),
  (@prop4, 'Denier',        true, '@account_id'),
  (@prop5, 'Fiber Content', true, '@account_id');


-- ============================================================================
-- SECTION 5: ATTRIBUTES
-- ============================================================================

INSERT INTO `attribute` (`id`, `text`, `order`, `property_id`, `color_code`, `account_id`, `is_public`) VALUES
  -- Color attributes
  (@attr1, 'White',        1, @prop1, 'default', '@account_id', true),
  (@attr2, 'Black',        2, @prop1, 'gray',    '@account_id', true),
  (@attr3, 'Navy',         3, @prop1, 'blue',    '@account_id', true),
  (@attr4, 'Heather Gray', 4, @prop1, 'gray',    '@account_id', true),
  (@attr5, 'Beige',        5, @prop1, 'brown',   '@account_id', true),
  -- Size attributes
  (@attr6, 'S',            1, @prop2, 'blue',    '@account_id', true),
  (@attr7, 'M',            2, @prop2, 'green',   '@account_id', true),
  (@attr8, 'L',            3, @prop2, 'red',     '@account_id', true),
  (@attr9, 'XL',           4, @prop2, 'orange',  '@account_id', true),
  -- Twist attributes
  (@attr10, 'S-Twist',      1, @prop3, 'orange',  '@account_id', true),
  (@attr11, 'Z-Twist',      2, @prop3, 'red',     '@account_id', true),
  -- Denier attributes
  (@attr12, '40D',          1, @prop4, 'blue',    '@account_id', true),
  (@attr13, '70D',          2, @prop4, 'green',   '@account_id', true),
  -- Fiber Content attributes
  (@attr14, 'Cotton',       1, @prop5, 'green',   '@account_id', true),
  (@attr15, 'Nylon',        2, @prop5, 'purple',  '@account_id', true),
  (@attr16, 'Spandex',      3, @prop5, 'yellow',  '@account_id', true);


-- ============================================================================
-- SECTION 6: CATEGORIES (item_category)
-- ============================================================================

INSERT INTO `item_category` (`id`, `name`, `notes`, `account_id`, `item_category_type_code`, `unit_group_id`) VALUES
  (@itmcat1, 'Crew Socks', NULL, '@account_id', 'product_category',  @ungp1),
  (@itmcat2, 'Ankle Socks', NULL, '@account_id', 'product_category', @ungp1),
  (@itmcat3, 'Yarn',       NULL, '@account_id', 'material_category', @ungp3),
  (@itmcat4, 'Dyes',       NULL, '@account_id', 'material_category', @ungp4),
  (@itmcat5, 'Chemicals',  NULL, '@account_id', 'material_category', @ungp4),
  (@itmcat6, 'Packaging',  NULL, '@account_id', 'material_category', @ungp5);


-- ============================================================================
-- SECTION 7: CATEGORY-PROPERTY JOINS (_item_categories_properties)
-- ============================================================================
-- A = item_category id, B = property id

INSERT INTO `_item_categories_properties` (`A`, `B`) VALUES
  -- Crew Socks: Color, Size
  (@itmcat1, @prop1),
  (@itmcat1, @prop2),
  -- Ankle Socks: Color, Size
  (@itmcat2, @prop1),
  (@itmcat2, @prop2),
  -- Yarn: Twist, Denier, Fiber Content, Color
  (@itmcat3, @prop1),
  (@itmcat3, @prop3),
  (@itmcat3, @prop4),
  (@itmcat3, @prop5),
  -- Dyes: Color
  (@itmcat4, @prop1);


-- ============================================================================
-- SECTION 8: PRODUCT LINES
-- ============================================================================

INSERT INTO `product_line` (`id`, `name`, `description`, `notes`, `account_id`, `unit_group_id`, `is_commission_exempt`, `is_freight_exempt`) VALUES
  (@pdln1, 'Crew Collection',  'Classic crew-length socks',  NULL, '@account_id', @ungp2, false, false),
  (@pdln2, 'Ankle Collection', 'Low-cut ankle socks',        NULL, '@account_id', @ungp2, false, false),
  (@pdln3, 'Athletic',         'Performance athletic socks',  NULL, '@account_id', @ungp2, false, false);


-- ============================================================================
-- SECTION 9: RATES
-- ============================================================================
-- Each item has 3 rates: unit_value, unit_cost, burn_rate
-- Rate columns: id, value, numerator_unit_id, denominator_unit_id
-- Products: rt_seed000000001..060 (20 products x 3)
-- Materials: rt_seed000000061..096 (12 materials x 3)
-- Parts: rt_seed000000097..141 (15 parts x 3)

INSERT INTO `rate` (`id`, `value`, `numerator_unit_id`, `denominator_unit_id`) VALUES
  -- ── Product Rates ────────────────────────────────────────────────────────
  -- Crew Socks: S/M/L x WHT/BLK/NAV  (unit_value=$30/pr, unit_cost=$7/pr, burn=1pr/d)
  -- FG-CRW-S-WHT (it_seed000000001)
  (@rt1, 30,  @un4, @un2),  -- unit_value
  (@rt2, 7,   @un4, @un2),  -- unit_cost
  (@rt3, 1,   @un2, @un6),  -- burn_rate
  -- FG-CRW-S-BLK (it_seed000000002)
  (@rt4, 30,  @un4, @un2),
  (@rt5, 7,   @un4, @un2),
  (@rt6, 1,   @un2, @un6),
  -- FG-CRW-S-NAV (it_seed000000003)
  (@rt7, 30,  @un4, @un2),
  (@rt8, 7,   @un4, @un2),
  (@rt9, 1,   @un2, @un6),
  -- FG-CRW-M-WHT (it_seed000000004)
  (@rt10, 33,  @un4, @un2),
  (@rt11, 7.5, @un4, @un2),
  (@rt12, 1,   @un2, @un6),
  -- FG-CRW-M-BLK (it_seed000000005)
  (@rt13, 33,  @un4, @un2),
  (@rt14, 7.5, @un4, @un2),
  (@rt15, 1,   @un2, @un6),
  -- FG-CRW-M-NAV (it_seed000000006)
  (@rt16, 33,  @un4, @un2),
  (@rt17, 7.5, @un4, @un2),
  (@rt18, 1,   @un2, @un6),
  -- FG-CRW-L-WHT (it_seed000000007)
  (@rt19, 36,  @un4, @un2),
  (@rt20, 8,   @un4, @un2),
  (@rt21, 1,   @un2, @un6),
  -- FG-CRW-L-BLK (it_seed000000008)
  (@rt22, 36,  @un4, @un2),
  (@rt23, 8,   @un4, @un2),
  (@rt24, 1,   @un2, @un6),
  -- FG-CRW-L-NAV (it_seed000000009)
  (@rt25, 36,  @un4, @un2),
  (@rt26, 8,   @un4, @un2),
  (@rt27, 1,   @un2, @un6),

  -- Ankle Socks: S/M/L x WHT/BLK/NAV  (unit_value=$24/pr, unit_cost=$5.5/pr, burn=1.5pr/d)
  -- FG-ANK-S-WHT (it_seed000000010)
  (@rt28, 24,   @un4, @un2),
  (@rt29, 5.5,  @un4, @un2),
  (@rt30, 1.5,  @un2, @un6),
  -- FG-ANK-S-BLK (it_seed000000011)
  (@rt31, 24,   @un4, @un2),
  (@rt32, 5.5,  @un4, @un2),
  (@rt33, 1.5,  @un2, @un6),
  -- FG-ANK-S-NAV (it_seed000000012)
  (@rt34, 24,   @un4, @un2),
  (@rt35, 5.5,  @un4, @un2),
  (@rt36, 1.5,  @un2, @un6),
  -- FG-ANK-M-WHT (it_seed000000013)
  (@rt37, 27,   @un4, @un2),
  (@rt38, 6,    @un4, @un2),
  (@rt39, 1.5,  @un2, @un6),
  -- FG-ANK-M-BLK (it_seed000000014)
  (@rt40, 27,   @un4, @un2),
  (@rt41, 6,    @un4, @un2),
  (@rt42, 1.5,  @un2, @un6),
  -- FG-ANK-M-NAV (it_seed000000015)
  (@rt43, 27,   @un4, @un2),
  (@rt44, 6,    @un4, @un2),
  (@rt45, 1.5,  @un2, @un6),
  -- FG-ANK-L-WHT (it_seed000000016)
  (@rt46, 30,   @un4, @un2),
  (@rt47, 6.5,  @un4, @un2),
  (@rt48, 1.5,  @un2, @un6),
  -- FG-ANK-L-BLK (it_seed000000017)
  (@rt49, 30,   @un4, @un2),
  (@rt50, 6.5,  @un4, @un2),
  (@rt51, 1.5,  @un2, @un6),
  -- FG-ANK-L-NAV (it_seed000000018)
  (@rt52, 30,   @un4, @un2),
  (@rt53, 6.5,  @un4, @un2),
  (@rt54, 1.5,  @un2, @un6),

  -- ── Material Rates ───────────────────────────────────────────────────────
  -- RM-YRN-COT70 (it_seed000000021): value=0, cost=$6/lb, burn=10lb/d
  (@rt61, 0,    @un4, @un8),
  (@rt62, 6,    @un4, @un8),
  (@rt63, 10,   @un8, @un6),
  -- RM-YRN-COT40 (it_seed000000022): value=0, cost=$8/lb, burn=8lb/d
  (@rt64, 0,    @un4, @un8),
  (@rt65, 8,    @un4, @un8),
  (@rt66, 8,    @un8, @un6),
  -- RM-YRN-NYL40 (it_seed000000023): value=0, cost=$12/lb, burn=5lb/d
  (@rt67, 0,    @un4, @un8),
  (@rt68, 12,   @un4, @un8),
  (@rt69, 5,    @un8, @un6),
  -- RM-ELS-SPX (it_seed000000024): value=0, cost=$15/lb, burn=3lb/d
  (@rt70, 0,    @un4, @un8),
  (@rt71, 15,   @un4, @un8),
  (@rt72, 3,    @un8, @un6),
  -- RM-DYE-BLK (it_seed000000025): value=0, cost=$4/g, burn=2g/d
  (@rt73, 0,    @un4, @un9),
  (@rt74, 4,    @un4, @un9),
  (@rt75, 2,    @un9, @un6),
  -- RM-DYE-NAV (it_seed000000026): value=0, cost=$5/g, burn=2g/d
  (@rt76, 0,    @un4, @un9),
  (@rt77, 5,    @un4, @un9),
  (@rt78, 2,    @un9, @un6),
  -- RM-DYE-BGE (it_seed000000027): value=0, cost=$4/g, burn=1g/d
  (@rt79, 0,    @un4, @un9),
  (@rt80, 4,    @un4, @un9),
  (@rt81, 1,    @un9, @un6),
  -- RM-CHM-SOF (it_seed000000028): value=0, cost=$1/g, burn=10g/d
  (@rt82, 0,    @un4, @un9),
  (@rt83, 1,    @un4, @un9),
  (@rt84, 10,   @un9, @un6),
  -- RM-CHM-DET (it_seed000000029): value=0, cost=$0.80/g, burn=10g/d
  (@rt85, 0,    @un4, @un9),
  (@rt86, 0.80, @un4, @un9),
  (@rt87, 10,   @un9, @un6),
  -- RM-PKG-BX12 (it_seed000000030): value=0, cost=$0.50/ea, burn=5ea/d
  (@rt88, 0,    @un4, @un1),
  (@rt89, 0.50, @un4, @un1),
  (@rt90, 5,    @un1, @un6),
  -- RM-PKG-BX06 (it_seed000000031): value=0, cost=$0.35/ea, burn=5ea/d
  (@rt91, 0,    @un4, @un1),
  (@rt92, 0.35, @un4, @un1),
  (@rt93, 5,    @un1, @un6),
  -- RM-PKG-LBL (it_seed000000032): value=0, cost=$0.05/ea, burn=20ea/d
  (@rt94, 0,    @un4, @un1),
  (@rt95, 0.05, @un4, @un1),
  (@rt96, 20,   @un1, @un6),

  -- ── Part Rates ───────────────────────────────────────────────────────────
  -- All WIP parts: value=0, cost=0, burn=0
  -- WIP-KNT-S (it_seed000000033)
  (@rt97,  0, @un4, @un2),
  (@rt98,  0, @un4, @un2),
  (@rt99,  0, @un2, @un6),
  -- WIP-KNT-M (it_seed000000034)
  (@rt100, 0, @un4, @un2),
  (@rt101, 0, @un4, @un2),
  (@rt102, 0, @un2, @un6),
  -- WIP-KNT-L (it_seed000000035)
  (@rt103, 0, @un4, @un2),
  (@rt104, 0, @un4, @un2),
  (@rt105, 0, @un2, @un6),
  -- WIP-LNK-S (it_seed000000036)
  (@rt106, 0, @un4, @un2),
  (@rt107, 0, @un4, @un2),
  (@rt108, 0, @un2, @un6),
  -- WIP-LNK-M (it_seed000000037)
  (@rt109, 0, @un4, @un2),
  (@rt110, 0, @un4, @un2),
  (@rt111, 0, @un2, @un6),
  -- WIP-LNK-L (it_seed000000038)
  (@rt112, 0, @un4, @un2),
  (@rt113, 0, @un4, @un2),
  (@rt114, 0, @un2, @un6),
  -- WIP-WSH-S (it_seed000000039)
  (@rt115, 0, @un4, @un2),
  (@rt116, 0, @un4, @un2),
  (@rt117, 0, @un2, @un6),
  -- WIP-WSH-M (it_seed000000040)
  (@rt118, 0, @un4, @un2),
  (@rt119, 0, @un4, @un2),
  (@rt120, 0, @un2, @un6),
  -- WIP-WSH-L (it_seed000000041)
  (@rt121, 0, @un4, @un2),
  (@rt122, 0, @un4, @un2),
  (@rt123, 0, @un2, @un6),
  -- WIP-DYE-S-BLK (it_seed000000042)
  (@rt124, 0, @un4, @un2),
  (@rt125, 0, @un4, @un2),
  (@rt126, 0, @un2, @un6),
  -- WIP-DYE-S-NAV (it_seed000000043)
  (@rt127, 0, @un4, @un2),
  (@rt128, 0, @un4, @un2),
  (@rt129, 0, @un2, @un6),
  -- WIP-DYE-M-BLK (it_seed000000044)
  (@rt130, 0, @un4, @un2),
  (@rt131, 0, @un4, @un2),
  (@rt132, 0, @un2, @un6),
  -- WIP-DYE-M-NAV (it_seed000000045)
  (@rt133, 0, @un4, @un2),
  (@rt134, 0, @un4, @un2),
  (@rt135, 0, @un2, @un6),
  -- WIP-DYE-L-BLK (it_seed000000046)
  (@rt136, 0, @un4, @un2),
  (@rt137, 0, @un4, @un2),
  (@rt138, 0, @un2, @un6),
  -- WIP-DYE-L-NAV (it_seed000000047)
  (@rt139, 0, @un4, @un2),
  (@rt140, 0, @un4, @un2),
  (@rt141, 0, @un2, @un6);


-- ============================================================================
-- SECTION 10: QUANTITIES (for materials: order_point and lead_time)
-- ============================================================================
-- Quantity columns: id, value, unit_id
-- 2 per material x 12 = 24 quantities

INSERT INTO `quantity` (`id`, `value`, `unit_id`) VALUES
  -- RM-YRN-COT70: order_point=10lb, lead_time=30d
  (@qty1, 10,   @un8),
  (@qty2, 30,   @un6),
  -- RM-YRN-COT40: order_point=8lb, lead_time=30d
  (@qty3, 8,    @un8),
  (@qty4, 30,   @un6),
  -- RM-YRN-NYL40: order_point=5lb, lead_time=25d
  (@qty5, 5,    @un8),
  (@qty6, 25,   @un6),
  -- RM-ELS-SPX: order_point=3lb, lead_time=20d
  (@qty7, 3,    @un8),
  (@qty8, 20,   @un6),
  -- RM-DYE-BLK: order_point=500g, lead_time=15d
  (@qty9, 500,  @un9),
  (@qty10, 15,   @un6),
  -- RM-DYE-NAV: order_point=500g, lead_time=15d
  (@qty11, 500,  @un9),
  (@qty12, 15,   @un6),
  -- RM-DYE-BGE: order_point=300g, lead_time=15d
  (@qty13, 300,  @un9),
  (@qty14, 15,   @un6),
  -- RM-CHM-SOF: order_point=1000g, lead_time=10d
  (@qty15, 1000, @un9),
  (@qty16, 10,   @un6),
  -- RM-CHM-DET: order_point=1000g, lead_time=10d
  (@qty17, 1000, @un9),
  (@qty18, 10,   @un6),
  -- RM-PKG-BX12: order_point=200ea, lead_time=7d
  (@qty19, 200,  @un1),
  (@qty20, 7,    @un6),
  -- RM-PKG-BX06: order_point=200ea, lead_time=7d
  (@qty21, 200,  @un1),
  (@qty22, 7,    @un6),
  -- RM-PKG-LBL: order_point=500ea, lead_time=5d
  (@qty23, 500,  @un1),
  (@qty24, 5,    @un6);


-- ============================================================================
-- SECTION 11: ITEMS
-- ============================================================================
-- Item columns: id, sku, description, notes, unit_value_id, burn_rate_id, account_id, item_type_code, unit_cost_id, item_category_id, is_dirty
-- Rate mapping: for item N, unit_value = rt_seed(N*3-2), unit_cost = rt_seed(N*3-1), burn_rate = rt_seed(N*3)

INSERT INTO `item` (`id`, `sku`, `description`, `notes`, `unit_value_id`, `burn_rate_id`, `account_id`, `item_type_code`, `unit_cost_id`, `item_category_id`, `is_dirty`) VALUES
  -- ── Crew Sock Products ───────────────────────────────────────────────────
  (@it1, 'FG-CRW-S-WHT', 'Crew Sock Small White',   NULL, @rt1, @rt3, '@account_id', 'product', @rt2, @itmcat1, false),
  (@it2, 'FG-CRW-S-BLK', 'Crew Sock Small Black',   NULL, @rt4, @rt6, '@account_id', 'product', @rt5, @itmcat1, false),
  (@it3, 'FG-CRW-S-NAV', 'Crew Sock Small Navy',    NULL, @rt7, @rt9, '@account_id', 'product', @rt8, @itmcat1, false),
  (@it4, 'FG-CRW-M-WHT', 'Crew Sock Medium White',  NULL, @rt10, @rt12, '@account_id', 'product', @rt11, @itmcat1, false),
  (@it5, 'FG-CRW-M-BLK', 'Crew Sock Medium Black',  NULL, @rt13, @rt15, '@account_id', 'product', @rt14, @itmcat1, false),
  (@it6, 'FG-CRW-M-NAV', 'Crew Sock Medium Navy',   NULL, @rt16, @rt18, '@account_id', 'product', @rt17, @itmcat1, false),
  (@it7, 'FG-CRW-L-WHT', 'Crew Sock Large White',   NULL, @rt19, @rt21, '@account_id', 'product', @rt20, @itmcat1, false),
  (@it8, 'FG-CRW-L-BLK', 'Crew Sock Large Black',   NULL, @rt22, @rt24, '@account_id', 'product', @rt23, @itmcat1, false),
  (@it9, 'FG-CRW-L-NAV', 'Crew Sock Large Navy',    NULL, @rt25, @rt27, '@account_id', 'product', @rt26, @itmcat1, false),

  -- ── Ankle Sock Products ──────────────────────────────────────────────────
  (@it10, 'FG-ANK-S-WHT', 'Ankle Sock Small White',  NULL, @rt28, @rt30, '@account_id', 'product', @rt29, @itmcat2, false),
  (@it11, 'FG-ANK-S-BLK', 'Ankle Sock Small Black',  NULL, @rt31, @rt33, '@account_id', 'product', @rt32, @itmcat2, false),
  (@it12, 'FG-ANK-S-NAV', 'Ankle Sock Small Navy',   NULL, @rt34, @rt36, '@account_id', 'product', @rt35, @itmcat2, false),
  (@it13, 'FG-ANK-M-WHT', 'Ankle Sock Medium White', NULL, @rt37, @rt39, '@account_id', 'product', @rt38, @itmcat2, false),
  (@it14, 'FG-ANK-M-BLK', 'Ankle Sock Medium Black', NULL, @rt40, @rt42, '@account_id', 'product', @rt41, @itmcat2, false),
  (@it15, 'FG-ANK-M-NAV', 'Ankle Sock Medium Navy',  NULL, @rt43, @rt45, '@account_id', 'product', @rt44, @itmcat2, false),
  (@it16, 'FG-ANK-L-WHT', 'Ankle Sock Large White',  NULL, @rt46, @rt48, '@account_id', 'product', @rt47, @itmcat2, false),
  (@it17, 'FG-ANK-L-BLK', 'Ankle Sock Large Black',  NULL, @rt49, @rt51, '@account_id', 'product', @rt50, @itmcat2, false),
  (@it18, 'FG-ANK-L-NAV', 'Ankle Sock Large Navy',   NULL, @rt52, @rt54, '@account_id', 'product', @rt53, @itmcat2, false),

  -- ── Materials ────────────────────────────────────────────────────────────
  (@it21, 'RM-YRN-COT70', 'Cotton Yarn 70 Denier',      NULL, @rt61, @rt63, '@account_id', 'material', @rt62, @itmcat3, false),
  (@it22, 'RM-YRN-COT40', 'Cotton Yarn 40 Denier',      NULL, @rt64, @rt66, '@account_id', 'material', @rt65, @itmcat3, false),
  (@it23, 'RM-YRN-NYL40', 'Nylon Yarn 40 Denier',       NULL, @rt67, @rt69, '@account_id', 'material', @rt68, @itmcat3, false),
  (@it24, 'RM-ELS-SPX',   'Spandex Elastic Thread',     NULL, @rt70, @rt72, '@account_id', 'material', @rt71, @itmcat3, false),
  (@it25, 'RM-DYE-BLK',   'Black Reactive Dye',         NULL, @rt73, @rt75, '@account_id', 'material', @rt74, @itmcat4, false),
  (@it26, 'RM-DYE-NAV',   'Navy Reactive Dye',          NULL, @rt76, @rt78, '@account_id', 'material', @rt77, @itmcat4, false),
  (@it27, 'RM-DYE-BGE',   'Beige Reactive Dye',         NULL, @rt79, @rt81, '@account_id', 'material', @rt80, @itmcat4, false),
  (@it28, 'RM-CHM-SOF',   'Fabric Softener',            NULL, @rt82, @rt84, '@account_id', 'material', @rt83, @itmcat5, false),
  (@it29, 'RM-CHM-DET',   'Industrial Detergent',       NULL, @rt85, @rt87, '@account_id', 'material', @rt86, @itmcat5, false),
  (@it30, 'RM-PKG-BX12',  'Corrugated Box 12-Pack',     NULL, @rt88, @rt90, '@account_id', 'material', @rt89, @itmcat6, false),
  (@it31, 'RM-PKG-BX06',  'Corrugated Box 6-Pack',      NULL, @rt91, @rt93, '@account_id', 'material', @rt92, @itmcat6, false),
  (@it32, 'RM-PKG-LBL',   'Product Label',              NULL, @rt94, @rt96, '@account_id', 'material', @rt95, @itmcat6, false),

  -- ── Parts (WIP) ──────────────────────────────────────────────────────────
  (@it33, 'WIP-KNT-S',     'Knitted Sock Blank Small',   NULL, @rt97,  @rt99,  '@account_id', 'part', @rt98,  @itmcat1, false),
  (@it34, 'WIP-KNT-M',     'Knitted Sock Blank Medium',  NULL, @rt100, @rt102, '@account_id', 'part', @rt101, @itmcat1, false),
  (@it35, 'WIP-KNT-L',     'Knitted Sock Blank Large',   NULL, @rt103, @rt105, '@account_id', 'part', @rt104, @itmcat1, false),
  (@it36, 'WIP-LNK-S',     'Linked Sock Small',          NULL, @rt106, @rt108, '@account_id', 'part', @rt107, @itmcat1, false),
  (@it37, 'WIP-LNK-M',     'Linked Sock Medium',         NULL, @rt109, @rt111, '@account_id', 'part', @rt110, @itmcat1, false),
  (@it38, 'WIP-LNK-L',     'Linked Sock Large',          NULL, @rt112, @rt114, '@account_id', 'part', @rt113, @itmcat1, false),
  (@it39, 'WIP-WSH-S',     'Scoured Sock Small',         NULL, @rt115, @rt117, '@account_id', 'part', @rt116, @itmcat1, false),
  (@it40, 'WIP-WSH-M',     'Scoured Sock Medium',        NULL, @rt118, @rt120, '@account_id', 'part', @rt119, @itmcat1, false),
  (@it41, 'WIP-WSH-L',     'Scoured Sock Large',         NULL, @rt121, @rt123, '@account_id', 'part', @rt122, @itmcat1, false),
  (@it42, 'WIP-DYE-S-BLK', 'Dyed Sock Small Black',      NULL, @rt124, @rt126, '@account_id', 'part', @rt125, @itmcat1, false),
  (@it43, 'WIP-DYE-S-NAV', 'Dyed Sock Small Navy',       NULL, @rt127, @rt129, '@account_id', 'part', @rt128, @itmcat1, false),
  (@it44, 'WIP-DYE-M-BLK', 'Dyed Sock Medium Black',     NULL, @rt130, @rt132, '@account_id', 'part', @rt131, @itmcat1, false),
  (@it45, 'WIP-DYE-M-NAV', 'Dyed Sock Medium Navy',      NULL, @rt133, @rt135, '@account_id', 'part', @rt134, @itmcat1, false),
  (@it46, 'WIP-DYE-L-BLK', 'Dyed Sock Large Black',      NULL, @rt136, @rt138, '@account_id', 'part', @rt137, @itmcat1, false),
  (@it47, 'WIP-DYE-L-NAV', 'Dyed Sock Large Navy',       NULL, @rt139, @rt141, '@account_id', 'part', @rt140, @itmcat1, false);


-- ============================================================================
-- SECTION 12: ITEM-ATTRIBUTE JOINS (_item_attributes)
-- ============================================================================
-- A = attribute id, B = item id

INSERT INTO `_item_attributes` (`A`, `B`) VALUES
  -- Crew Socks: Size + Color
  (@attr6, @it1), (@attr1, @it1),  -- S, White
  (@attr6, @it2), (@attr2, @it2),  -- S, Black
  (@attr6, @it3), (@attr3, @it3),  -- S, Navy
  (@attr7, @it4), (@attr1, @it4),  -- M, White
  (@attr7, @it5), (@attr2, @it5),  -- M, Black
  (@attr7, @it6), (@attr3, @it6),  -- M, Navy
  (@attr8, @it7), (@attr1, @it7),  -- L, White
  (@attr8, @it8), (@attr2, @it8),  -- L, Black
  (@attr8, @it9), (@attr3, @it9),  -- L, Navy
  -- Ankle Socks: Size + Color
  (@attr6, @it10), (@attr1, @it10),  -- S, White
  (@attr6, @it11), (@attr2, @it11),  -- S, Black
  (@attr6, @it12), (@attr3, @it12),  -- S, Navy
  (@attr7, @it13), (@attr1, @it13),  -- M, White
  (@attr7, @it14), (@attr2, @it14),  -- M, Black
  (@attr7, @it15), (@attr3, @it15),  -- M, Navy
  (@attr8, @it16), (@attr1, @it16),  -- L, White
  (@attr8, @it17), (@attr2, @it17),  -- L, Black
  (@attr8, @it18), (@attr3, @it18),  -- L, Navy
  -- Materials: Yarn attributes
  (@attr13, @it21), (@attr14, @it21),  -- COT70: 70D, Cotton
  (@attr12, @it22), (@attr14, @it22),  -- COT40: 40D, Cotton
  (@attr12, @it23), (@attr15, @it23),  -- NYL40: 40D, Nylon
  (@attr16, @it24),                                               -- SPX: Spandex
  -- Materials: Dye color attributes
  (@attr2, @it25),  -- BLK dye: Black
  (@attr3, @it26),  -- NAV dye: Navy
  (@attr5, @it27),  -- BGE dye: Beige
  -- Parts: Size attributes
  (@attr6, @it33),  -- WIP-KNT-S: S
  (@attr7, @it34),  -- WIP-KNT-M: M
  (@attr8, @it35),  -- WIP-KNT-L: L
  (@attr6, @it36),  -- WIP-LNK-S: S
  (@attr7, @it37),  -- WIP-LNK-M: M
  (@attr8, @it38),  -- WIP-LNK-L: L
  (@attr6, @it39),  -- WIP-WSH-S: S
  (@attr7, @it40),  -- WIP-WSH-M: M
  (@attr8, @it41),  -- WIP-WSH-L: L
  (@attr6, @it42), (@attr2, @it42),  -- WIP-DYE-S-BLK: S, Black
  (@attr6, @it43), (@attr3, @it43),  -- WIP-DYE-S-NAV: S, Navy
  (@attr7, @it44), (@attr2, @it44),  -- WIP-DYE-M-BLK: M, Black
  (@attr7, @it45), (@attr3, @it45),  -- WIP-DYE-M-NAV: M, Navy
  (@attr8, @it46), (@attr2, @it46),  -- WIP-DYE-L-BLK: L, Black
  (@attr8, @it47), (@attr3, @it47);  -- WIP-DYE-L-NAV: L, Navy


-- ============================================================================
-- SECTION 13: PRODUCTS
-- ============================================================================

INSERT INTO `product` (`id`, `item_id`, `product_type_code`, `product_line_id`) VALUES
  -- Crew Socks
  (@pd1, @it1, 'sale', @pdln1),
  (@pd2, @it2, 'sale', @pdln1),
  (@pd3, @it3, 'sale', @pdln1),
  (@pd4, @it4, 'sale', @pdln1),
  (@pd5, @it5, 'sale', @pdln1),
  (@pd6, @it6, 'sale', @pdln1),
  (@pd7, @it7, 'sale', @pdln1),
  (@pd8, @it8, 'sale', @pdln1),
  (@pd9, @it9, 'sale', @pdln1),
  -- Ankle Socks
  (@pd10, @it10, 'sale', @pdln2),
  (@pd11, @it11, 'sale', @pdln2),
  (@pd12, @it12, 'sale', @pdln2),
  (@pd13, @it13, 'sale', @pdln2),
  (@pd14, @it14, 'sale', @pdln2),
  (@pd15, @it15, 'sale', @pdln2),
  (@pd16, @it16, 'sale', @pdln2),
  (@pd17, @it17, 'sale', @pdln2),
  (@pd18, @it18, 'sale', @pdln2);


-- ============================================================================
-- SECTION 14: MATERIALS
-- ============================================================================

INSERT INTO `material` (`id`, `item_id`, `order_point_id`, `lead_time_id`) VALUES
  (@mat1, @it21, @qty1, @qty2),
  (@mat2, @it22, @qty3, @qty4),
  (@mat3, @it23, @qty5, @qty6),
  (@mat4, @it24, @qty7, @qty8),
  (@mat5, @it25, @qty9, @qty10),
  (@mat6, @it26, @qty11, @qty12),
  (@mat7, @it27, @qty13, @qty14),
  (@mat8, @it28, @qty15, @qty16),
  (@mat9, @it29, @qty17, @qty18),
  (@mat10, @it30, @qty19, @qty20),
  (@mat11, @it31, @qty21, @qty22),
  (@mat12, @it32, @qty23, @qty24);


-- ============================================================================
-- SECTION 15: PARTS
-- ============================================================================

INSERT INTO `part` (`id`, `item_id`) VALUES
  (@prt1, @it33),
  (@prt2, @it34),
  (@prt3, @it35),
  (@prt4, @it36),
  (@prt5, @it37),
  (@prt6, @it38),
  (@prt7, @it39),
  (@prt8, @it40),
  (@prt9, @it41),
  (@prt10, @it42),
  (@prt11, @it43),
  (@prt12, @it44),
  (@prt13, @it45),
  (@prt14, @it46),
  (@prt15, @it47);


-- ============================================================================
-- SECTION 16: DEPARTMENTS
-- ============================================================================
-- NOTE: location_id will be updated after storage_locations are created

INSERT INTO `department` (`id`, `name`, `notes`, `account_id`, `location_id`) VALUES
  (@dept1, 'Knitting',       'Circular and flat knitting machines',       '@account_id', NULL),
  (@dept2, 'Linking',        'Toe closing and seaming operations',        '@account_id', NULL),
  (@dept3, 'Wet Processing', 'Scouring, bleaching, and pre-treatment',    '@account_id', NULL),
  (@dept4, 'Dyeing',         'Reactive dyeing and color matching',        '@account_id', NULL),
  (@dept5, 'Finishing',      'Boarding, pressing, and quality inspection', '@account_id', NULL),
  (@dept6, 'Packaging',      'Pairing, labeling, and boxing',             '@account_id', NULL);


-- ============================================================================
-- SECTION 17: STORAGE LOCATIONS
-- ============================================================================

INSERT INTO `storage_location` (`id`, `account_id`, `storage_location_type_code`, `name`, `parent_id`) VALUES
  (@stloc1, '@account_id', 'building', 'Main Building',          NULL),
  (@stloc2, '@account_id', 'section',  'Knitting Floor',         @stloc1),
  (@stloc3, '@account_id', 'section',  'Linking Floor',          @stloc1),
  (@stloc4, '@account_id', 'section',  'Wet Processing Area',    @stloc1),
  (@stloc5, '@account_id', 'section',  'Dye House',              @stloc1),
  (@stloc6, '@account_id', 'section',  'Finishing Area',         @stloc1),
  (@stloc7, '@account_id', 'section',  'Pack & Ship',            @stloc1),
  (@stloc8, '@account_id', 'section',  'Raw Materials Storage',  @stloc1);


-- ============================================================================
-- SECTION 18: UPDATE DEPARTMENTS WITH LOCATION IDS
-- ============================================================================

UPDATE `department` SET `location_id` = @stloc2 WHERE `id` = @dept1;
UPDATE `department` SET `location_id` = @stloc3 WHERE `id` = @dept2;
UPDATE `department` SET `location_id` = @stloc4 WHERE `id` = @dept3;
UPDATE `department` SET `location_id` = @stloc5 WHERE `id` = @dept4;
UPDATE `department` SET `location_id` = @stloc6 WHERE `id` = @dept5;
UPDATE `department` SET `location_id` = @stloc7 WHERE `id` = @dept6;


-- ============================================================================
-- SECTION 19: SCANNING STATIONS
-- ============================================================================

INSERT INTO `scanning_station` (`id`, `name`, `notes`, `department_id`, `account_id`, `scanning_station_type_code`, `label_size_code`, `label_type_code`, `material_check_required`) VALUES
  (@scst1, 'Knitting Station',       NULL, @dept1, '@account_id', 'init_batch', '1x4', 'tag', false),
  (@scst2, 'Linking Station',        NULL, @dept2, '@account_id', 'move_batch', '1x4', 'tag', false),
  (@scst3, 'Wet Processing Station', NULL, @dept3, '@account_id', 'move_batch', '1x4', 'tag', true),
  (@scst4, 'Dyeing Station',         NULL, @dept4, '@account_id', 'move_batch', '1x4', 'tag', true),
  (@scst5, 'Finishing Station',       NULL, @dept5, '@account_id', 'move_batch', '1x4', 'tag', false),
  (@scst6, 'Packaging Station',       NULL, @dept6, '@account_id', 'move_batch', '1x4', 'tag', false);


-- ============================================================================
-- SECTION 20: MACHINES
-- ============================================================================

-- The knitting machines carry a production step, the rest do not. Knitting is the
-- planning constraint (SECTION 56), and a campaign explodes into downstream department
-- work through its machine's own step: a constraint machine without one derives nothing
-- and is reported as a gap on every generated schedule. One machine per knit size is
-- what makes that coverage complete. Machines outside the constraint are planned as a
-- department pool and read their rate off the production step, so a step there would be
-- decoration.
INSERT INTO `machine` (`id`, `account_id`, `name`, `notes`, `serial_number`, `department_id`, `production_step_id`) VALUES
  -- Knitting (3 machines, one per size)
  (@mach1, '@account_id', 'Lonati GL616 #1',        NULL, 'LNT-KNT-001', @dept1, @pdst1),
  (@mach2, '@account_id', 'Lonati GL616 #2',        NULL, 'LNT-KNT-002', @dept1, @pdst2),
  (@mach13, '@account_id', 'Lonati GL616 #3',       NULL, 'LNT-KNT-003', @dept1, @pdst3),
  -- Linking (2 machines)
  (@mach3, '@account_id', 'Rosso RSM-08 #1',        NULL, 'RSO-LNK-001', @dept2, NULL),
  (@mach4, '@account_id', 'Rosso RSM-08 #2',        NULL, 'RSO-LNK-002', @dept2, NULL),
  -- Wet Processing (2 machines)
  (@mach5, '@account_id', 'Tonello G1 Washer #1',   NULL, 'TNL-WSH-001', @dept3, NULL),
  (@mach6, '@account_id', 'Tonello G1 Washer #2',   NULL, 'TNL-WSH-002', @dept3, NULL),
  -- Dyeing (2 machines)
  (@mach7, '@account_id', 'Thies iCone 200 #1',     NULL, 'THS-DYE-001', @dept4, NULL),
  (@mach8, '@account_id', 'Thies iCone 200 #2',     NULL, 'THS-DYE-002', @dept4, NULL),
  -- Finishing (2 machines)
  (@mach9, '@account_id', 'Cortese Boarding Press #1', NULL, 'CRT-FIN-001', @dept5, NULL),
  (@mach10, '@account_id', 'Cortese Boarding Press #2', NULL, 'CRT-FIN-002', @dept5, NULL),
  -- Packaging (2 machines)
  (@mach11, '@account_id', 'Sato CL4NX Label Printer',  NULL, 'SAT-PKG-001', @dept6, NULL),
  (@mach12, '@account_id', 'Lantech Q-300 Case Sealer',  NULL, 'LNT-PKG-002', @dept6, NULL);


-- ============================================================================
-- SECTION 21: PRODUCTION STEP RATES
-- ============================================================================
-- Each step needs 3 rates: labor_rate ($/hr), labor_time (min/pr), overhead_rate ($/hr)
-- rt_seed000000142..195 (18 steps x 3)

INSERT INTO `rate` (`id`, `value`, `numerator_unit_id`, `denominator_unit_id`) VALUES
  -- Knit S: labor=$14/hr, time=3min/pr, overhead=$8/hr
  (@rt142, 14, @un4, @un5),
  (@rt143, 3,  @un7, @un2),
  (@rt144, 8,  @un4, @un5),
  -- Knit M: labor=$14/hr, time=3.5min/pr, overhead=$8/hr
  (@rt145, 14,  @un4, @un5),
  (@rt146, 3.5, @un7, @un2),
  (@rt147, 8,   @un4, @un5),
  -- Knit L: labor=$14/hr, time=4min/pr, overhead=$8/hr
  (@rt148, 14, @un4, @un5),
  (@rt149, 4,  @un7, @un2),
  (@rt150, 8,  @un4, @un5),
  -- Link S: labor=$12/hr, time=5min/pr, overhead=$5/hr
  (@rt151, 12, @un4, @un5),
  (@rt152, 5,  @un7, @un2),
  (@rt153, 5,  @un4, @un5),
  -- Link M: labor=$12/hr, time=5.5min/pr, overhead=$5/hr
  (@rt154, 12,  @un4, @un5),
  (@rt155, 5.5, @un7, @un2),
  (@rt156, 5,   @un4, @un5),
  -- Link L: labor=$12/hr, time=6min/pr, overhead=$5/hr
  (@rt157, 12, @un4, @un5),
  (@rt158, 6,  @un7, @un2),
  (@rt159, 5,  @un4, @un5),
  -- Wash S: labor=$10/hr, time=2min/pr, overhead=$12/hr
  (@rt160, 10, @un4, @un5),
  (@rt161, 2,  @un7, @un2),
  (@rt162, 12, @un4, @un5),
  -- Wash M: labor=$10/hr, time=2.5min/pr, overhead=$12/hr
  (@rt163, 10,  @un4, @un5),
  (@rt164, 2.5, @un7, @un2),
  (@rt165, 12,  @un4, @un5),
  -- Wash L: labor=$10/hr, time=3min/pr, overhead=$12/hr
  (@rt166, 10, @un4, @un5),
  (@rt167, 3,  @un7, @un2),
  (@rt168, 12, @un4, @un5),
  -- Dye S BLK: labor=$15/hr, time=4min/pr, overhead=$10/hr
  (@rt169, 15, @un4, @un5),
  (@rt170, 4,  @un7, @un2),
  (@rt171, 10, @un4, @un5),
  -- Dye S NAV
  (@rt172, 15, @un4, @un5),
  (@rt173, 4,  @un7, @un2),
  (@rt174, 10, @un4, @un5),
  -- Dye M BLK
  (@rt175, 15,  @un4, @un5),
  (@rt176, 4.5, @un7, @un2),
  (@rt177, 10,  @un4, @un5),
  -- Dye M NAV
  (@rt178, 15,  @un4, @un5),
  (@rt179, 4.5, @un7, @un2),
  (@rt180, 10,  @un4, @un5),
  -- Dye L BLK
  (@rt181, 15, @un4, @un5),
  (@rt182, 5,  @un7, @un2),
  (@rt183, 10, @un4, @un5),
  -- Dye L NAV
  (@rt184, 15, @un4, @un5),
  (@rt185, 5,  @un7, @un2),
  (@rt186, 10, @un4, @un5),
  -- Pack S: labor=$10/hr, time=1min/pr, overhead=$3/hr
  (@rt187, 10, @un4, @un5),
  (@rt188, 1,  @un7, @un2),
  (@rt189, 3,  @un4, @un5),
  -- Pack M
  (@rt190, 10, @un4, @un5),
  (@rt191, 1,  @un7, @un2),
  (@rt192, 3,  @un4, @un5),
  -- Pack L
  (@rt193, 10, @un4, @un5),
  (@rt194, 1.5, @un7, @un2),
  (@rt195, 3,  @un4, @un5),
  -- Pack S BLK: labor=$10/hr, time=1min/pr, overhead=$3/hr
  (@rt226, 10, @un4, @un5),
  (@rt227, 1,  @un7, @un2),
  (@rt228, 3,  @un4, @un5),
  -- Pack S NAV
  (@rt229, 10, @un4, @un5),
  (@rt230, 1,  @un7, @un2),
  (@rt231, 3,  @un4, @un5),
  -- Pack M BLK
  (@rt232, 10, @un4, @un5),
  (@rt233, 1,  @un7, @un2),
  (@rt234, 3,  @un4, @un5),
  -- Pack M NAV
  (@rt235, 10, @un4, @un5),
  (@rt236, 1,  @un7, @un2),
  (@rt237, 3,  @un4, @un5),
  -- Pack L BLK
  (@rt238, 10, @un4, @un5),
  (@rt239, 1.5, @un7, @un2),
  (@rt240, 3,  @un4, @un5),
  -- Pack L NAV
  (@rt241, 10, @un4, @un5),
  (@rt242, 1.5, @un7, @un2),
  (@rt243, 3,  @un4, @un5);


-- ============================================================================
-- SECTION 22: CONSUMPTION & PRODUCTION QUANTITIES
-- ============================================================================
-- qty_seed000000025..117: consumption qty, waste qty, and production qty

INSERT INTO `quantity` (`id`, `value`, `unit_id`) VALUES
  -- ── Knit S (2 consumptions + 1 production) ──────────────────────────────
  (@qty25, 0.25,  @un8),  -- Cotton 70D qty
  (@qty26, 0.01,  @un8),  -- Cotton 70D waste
  (@qty27, 0.02,  @un8),  -- Spandex qty
  (@qty28, 0.001, @un8),  -- Spandex waste
  (@qty29, 1,     @un2),  -- Produces WIP-KNT-S
  -- ── Knit M ──────────────────────────────────────────────────────────────
  (@qty30, 0.30,  @un8),
  (@qty31, 0.015, @un8),
  (@qty32, 0.03,  @un8),
  (@qty33, 0.001, @un8),
  (@qty34, 1,     @un2),
  -- ── Knit L ──────────────────────────────────────────────────────────────
  (@qty35, 0.35,  @un8),
  (@qty36, 0.02,  @un8),
  (@qty37, 0.04,  @un8),
  (@qty38, 0.002, @un8),
  (@qty39, 1,     @un2),
  -- ── Link S (2 consumptions + 1 production) ─────────────────────────────
  (@qty40, 1,     @un2),  -- WIP-KNT-S qty
  (@qty41, 0,     @un2),  -- WIP-KNT-S waste
  (@qty42, 0.01,  @un8),  -- Nylon qty
  (@qty43, 0.001, @un8),  -- Nylon waste
  (@qty44, 1,     @un2),  -- Produces WIP-LNK-S
  -- ── Link M ──────────────────────────────────────────────────────────────
  (@qty45, 1,     @un2),
  (@qty46, 0,     @un2),
  (@qty47, 0.01,  @un8),
  (@qty48, 0.001, @un8),
  (@qty49, 1,     @un2),
  -- ── Link L ──────────────────────────────────────────────────────────────
  (@qty50, 1,     @un2),
  (@qty51, 0,     @un2),
  (@qty52, 0.01,  @un8),
  (@qty53, 0.001, @un8),
  (@qty54, 1,     @un2),
  -- ── Wash S (3 consumptions + 1 production) ─────────────────────────────
  (@qty55, 1,  @un2),  -- WIP-LNK-S qty
  (@qty56, 0,  @un2),  -- WIP-LNK-S waste
  (@qty57, 5,  @un9),  -- Detergent qty
  (@qty58, 0,  @un9),  -- Detergent waste
  (@qty59, 3,  @un9),  -- Softener qty
  (@qty60, 0,  @un9),  -- Softener waste
  (@qty61, 1,  @un2),  -- Produces WIP-WSH-S
  -- ── Wash M ──────────────────────────────────────────────────────────────
  (@qty62, 1,  @un2),
  (@qty63, 0,  @un2),
  (@qty64, 6,  @un9),
  (@qty65, 0,  @un9),
  (@qty66, 4,  @un9),
  (@qty67, 0,  @un9),
  (@qty68, 1,  @un2),
  -- ── Wash L ──────────────────────────────────────────────────────────────
  (@qty69, 1,  @un2),
  (@qty70, 0,  @un2),
  (@qty71, 7,  @un9),
  (@qty72, 0,  @un9),
  (@qty73, 5,  @un9),
  (@qty74, 0,  @un9),
  (@qty75, 1,  @un2),
  -- ── Dye S BLK (2 consumptions + 1 production) ─────────────────────────
  (@qty76, 1,   @un2),  -- WIP-WSH-S qty
  (@qty77, 0,   @un2),  -- WIP-WSH-S waste
  (@qty78, 2,   @un9),  -- Black Dye qty
  (@qty79, 0.1, @un9),  -- Black Dye waste
  (@qty80, 1,   @un2),  -- Produces WIP-DYE-S-BLK
  -- ── Dye S NAV ──────────────────────────────────────────────────────────
  (@qty81, 1,   @un2),
  (@qty82, 0,   @un2),
  (@qty83, 2,   @un9),
  (@qty84, 0.1, @un9),
  (@qty85, 1,   @un2),
  -- ── Dye M BLK ──────────────────────────────────────────────────────────
  (@qty86, 1,   @un2),
  (@qty87, 0,   @un2),
  (@qty88, 2.5, @un9),
  (@qty89, 0.1, @un9),
  (@qty90, 1,   @un2),
  -- ── Dye M NAV ──────────────────────────────────────────────────────────
  (@qty91, 1,   @un2),
  (@qty92, 0,   @un2),
  (@qty93, 2.5, @un9),
  (@qty94, 0.1, @un9),
  (@qty95, 1,   @un2),
  -- ── Dye L BLK ──────────────────────────────────────────────────────────
  (@qty96, 1,   @un2),
  (@qty97, 0,   @un2),
  (@qty98, 3,   @un9),
  (@qty99, 0.1, @un9),
  (@qty100, 1,   @un2),
  -- ── Dye L NAV ──────────────────────────────────────────────────────────
  (@qty101, 1,   @un2),
  (@qty102, 0,   @un2),
  (@qty103, 3,   @un9),
  (@qty104, 0.1, @un9),
  (@qty105, 1,   @un2),
  -- ── Pack S (2 consumptions, no production) ─────────────────────────────
  (@qty106, 1,     @un1),  -- Label qty
  (@qty107, 0,     @un1),  -- Label waste
  (@qty108, 0.167, @un1),  -- Box 6-Pack qty
  (@qty109, 0,     @un1),  -- Box waste
  -- ── Pack M ─────────────────────────────────────────────────────────────
  (@qty110, 1,     @un1),
  (@qty111, 0,     @un1),
  (@qty112, 0.167, @un1),
  (@qty113, 0,     @un1),
  -- ── Pack L ─────────────────────────────────────────────────────────────
  (@qty114, 1,     @un1),
  (@qty115, 0,     @un1),
  (@qty116, 0.083, @un1),
  (@qty117, 0,     @un1),
  -- Pack S BLK: Label + Box 6-Pack
  (@qty261, 1,     @un1),  -- Label qty
  (@qty262, 0,     @un1),  -- Label waste
  (@qty263, 0.167, @un1),  -- Box 6-Pack qty
  (@qty264, 0,     @un1),  -- Box waste
  -- Pack S NAV: Label + Box 6-Pack
  (@qty265, 1,     @un1),
  (@qty266, 0,     @un1),
  (@qty267, 0.167, @un1),
  (@qty268, 0,     @un1),
  -- Pack M BLK: Label + Box 6-Pack
  (@qty269, 1,     @un1),
  (@qty270, 0,     @un1),
  (@qty271, 0.167, @un1),
  (@qty272, 0,     @un1),
  -- Pack M NAV: Label + Box 6-Pack
  (@qty273, 1,     @un1),
  (@qty274, 0,     @un1),
  (@qty275, 0.167, @un1),
  (@qty276, 0,     @un1),
  -- Pack L BLK: Label + Box 12-Pack
  (@qty277, 1,     @un1),
  (@qty278, 0,     @un1),
  (@qty279, 0.083, @un1),
  (@qty280, 0,     @un1),
  -- Pack L NAV: Label + Box 12-Pack
  (@qty281, 1,     @un1),
  (@qty282, 0,     @un1),
  (@qty283, 0.083, @un1),
  (@qty284, 0,     @un1),
  -- ── Pack WIP consumptions (1 pair each) ──────────────────────────────
  -- Pack S WHT: WIP-WSH-S
  (@qty291, 1, @un2),
  (@qty292, 0, @un2),
  -- Pack M WHT: WIP-WSH-M
  (@qty293, 1, @un2),
  (@qty294, 0, @un2),
  -- Pack L WHT: WIP-WSH-L
  (@qty295, 1, @un2),
  (@qty296, 0, @un2),
  -- Pack S BLK: WIP-DYE-S-BLK
  (@qty297, 1, @un2),
  (@qty298, 0, @un2),
  -- Pack S NAV: WIP-DYE-S-NAV
  (@qty299, 1, @un2),
  (@qty300, 0, @un2),
  -- Pack M BLK: WIP-DYE-M-BLK
  (@qty301, 1, @un2),
  (@qty302, 0, @un2),
  -- Pack M NAV: WIP-DYE-M-NAV
  (@qty303, 1, @un2),
  (@qty304, 0, @un2),
  -- Pack L BLK: WIP-DYE-L-BLK
  (@qty305, 1, @un2),
  (@qty306, 0, @un2),
  -- Pack L NAV: WIP-DYE-L-NAV
  (@qty307, 1, @un2),
  (@qty308, 0, @un2);


-- ============================================================================
-- SECTION 23: PRODUCTION STEPS
-- ============================================================================

INSERT INTO `production_step` (`id`, `name`, `notes`, `account_id`, `scanning_station_id`, `allowances`, `labor_rate_id`, `labor_time_id`, `leveling_factor`, `overhead_rate_id`, `department_id`) VALUES
  (@pdst1, 'Knit Small',      NULL, '@account_id', @scst1, 0, @rt142, @rt143, 0, @rt144, @dept1),
  (@pdst2, 'Knit Medium',     NULL, '@account_id', @scst1, 0, @rt145, @rt146, 0, @rt147, @dept1),
  (@pdst3, 'Knit Large',      NULL, '@account_id', @scst1, 0, @rt148, @rt149, 0, @rt150, @dept1),
  (@pdst4, 'Link Small',      NULL, '@account_id', @scst2, 0, @rt151, @rt152, 0, @rt153, @dept2),
  (@pdst5, 'Link Medium',     NULL, '@account_id', @scst2, 0, @rt154, @rt155, 0, @rt156, @dept2),
  (@pdst6, 'Link Large',      NULL, '@account_id', @scst2, 0, @rt157, @rt158, 0, @rt159, @dept2),
  (@pdst7, 'Wash Small',      NULL, '@account_id', @scst3, 0, @rt160, @rt161, 0, @rt162, @dept3),
  (@pdst8, 'Wash Medium',     NULL, '@account_id', @scst3, 0, @rt163, @rt164, 0, @rt165, @dept3),
  (@pdst9, 'Wash Large',      NULL, '@account_id', @scst3, 0, @rt166, @rt167, 0, @rt168, @dept3),
  (@pdst10, 'Dye Small Black', NULL, '@account_id', @scst4, 0, @rt169, @rt170, 0, @rt171, @dept4),
  (@pdst11, 'Dye Small Navy',  NULL, '@account_id', @scst4, 0, @rt172, @rt173, 0, @rt174, @dept4),
  (@pdst12, 'Dye Medium Black', NULL, '@account_id', @scst4, 0, @rt175, @rt176, 0, @rt177, @dept4),
  (@pdst13, 'Dye Medium Navy',  NULL, '@account_id', @scst4, 0, @rt178, @rt179, 0, @rt180, @dept4),
  (@pdst14, 'Dye Large Black',  NULL, '@account_id', @scst4, 0, @rt181, @rt182, 0, @rt183, @dept4),
  (@pdst15, 'Dye Large Navy',   NULL, '@account_id', @scst4, 0, @rt184, @rt185, 0, @rt186, @dept4),
  (@pdst16, 'Pack S WHT',       NULL, '@account_id', @scst6, 0, @rt187, @rt188, 0, @rt189, @dept6),
  (@pdst17, 'Pack M WHT',       NULL, '@account_id', @scst6, 0, @rt190, @rt191, 0, @rt192, @dept6),
  (@pdst18, 'Pack L WHT',       NULL, '@account_id', @scst6, 0, @rt193, @rt194, 0, @rt195, @dept6),
  (@pdst19, 'Pack S BLK',       NULL, '@account_id', @scst6, 0, @rt226, @rt227, 0, @rt228, @dept6),
  (@pdst20, 'Pack S NAV',       NULL, '@account_id', @scst6, 0, @rt229, @rt230, 0, @rt231, @dept6),
  (@pdst21, 'Pack M BLK',       NULL, '@account_id', @scst6, 0, @rt232, @rt233, 0, @rt234, @dept6),
  (@pdst22, 'Pack M NAV',       NULL, '@account_id', @scst6, 0, @rt235, @rt236, 0, @rt237, @dept6),
  (@pdst23, 'Pack L BLK',       NULL, '@account_id', @scst6, 0, @rt238, @rt239, 0, @rt240, @dept6),
  (@pdst24, 'Pack L NAV',       NULL, '@account_id', @scst6, 0, @rt241, @rt242, 0, @rt243, @dept6);


-- ============================================================================
-- SECTION 24: CONSUMPTIONS
-- ============================================================================
-- consumption columns: id, item_id, quantity_id, production_step_id, waste_quantity_id, instructions

INSERT INTO `consumption` (`id`, `item_id`, `quantity_id`, `production_step_id`, `waste_quantity_id`, `instructions`) VALUES
  -- Knit S: Cotton 70D + Spandex
  (@cons1, @it21, @qty25, @pdst1, @qty26, NULL),
  (@cons2, @it24, @qty27, @pdst1, @qty28, NULL),
  -- Knit M: Cotton 70D + Spandex
  (@cons3, @it21, @qty30, @pdst2, @qty31, NULL),
  (@cons4, @it24, @qty32, @pdst2, @qty33, NULL),
  -- Knit L: Cotton 70D + Spandex
  (@cons5, @it21, @qty35, @pdst3, @qty36, NULL),
  (@cons6, @it24, @qty37, @pdst3, @qty38, NULL),
  -- Link S: WIP-KNT-S + Nylon
  (@cons7, @it33, @qty40, @pdst4, @qty41, NULL),
  (@cons8, @it23, @qty42, @pdst4, @qty43, NULL),
  -- Link M: WIP-KNT-M + Nylon
  (@cons9, @it34, @qty45, @pdst5, @qty46, NULL),
  (@cons10, @it23, @qty47, @pdst5, @qty48, NULL),
  -- Link L: WIP-KNT-L + Nylon
  (@cons11, @it35, @qty50, @pdst6, @qty51, NULL),
  (@cons12, @it23, @qty52, @pdst6, @qty53, NULL),
  -- Wash S: WIP-LNK-S + Detergent + Softener
  (@cons13, @it36, @qty55, @pdst7, @qty56, NULL),
  (@cons14, @it29, @qty57, @pdst7, @qty58, NULL),
  (@cons15, @it28, @qty59, @pdst7, @qty60, NULL),
  -- Wash M: WIP-LNK-M + Detergent + Softener
  (@cons16, @it37, @qty62, @pdst8, @qty63, NULL),
  (@cons17, @it29, @qty64, @pdst8, @qty65, NULL),
  (@cons18, @it28, @qty66, @pdst8, @qty67, NULL),
  -- Wash L: WIP-LNK-L + Detergent + Softener
  (@cons19, @it38, @qty69, @pdst9, @qty70, NULL),
  (@cons20, @it29, @qty71, @pdst9, @qty72, NULL),
  (@cons21, @it28, @qty73, @pdst9, @qty74, NULL),
  -- Dye S BLK: WIP-WSH-S + Black Dye
  (@cons22, @it39, @qty76, @pdst10, @qty77, NULL),
  (@cons23, @it25, @qty78, @pdst10, @qty79, NULL),
  -- Dye S NAV: WIP-WSH-S + Navy Dye
  (@cons24, @it39, @qty81, @pdst11, @qty82, NULL),
  (@cons25, @it26, @qty83, @pdst11, @qty84, NULL),
  -- Dye M BLK: WIP-WSH-M + Black Dye
  (@cons26, @it40, @qty86, @pdst12, @qty87, NULL),
  (@cons27, @it25, @qty88, @pdst12, @qty89, NULL),
  -- Dye M NAV: WIP-WSH-M + Navy Dye
  (@cons28, @it40, @qty91, @pdst13, @qty92, NULL),
  (@cons29, @it26, @qty93, @pdst13, @qty94, NULL),
  -- Dye L BLK: WIP-WSH-L + Black Dye
  (@cons30, @it41, @qty96, @pdst14, @qty97, NULL),
  (@cons31, @it25, @qty98, @pdst14, @qty99, NULL),
  -- Dye L NAV: WIP-WSH-L + Navy Dye
  (@cons32, @it41, @qty101, @pdst15, @qty102, NULL),
  (@cons33, @it26, @qty103, @pdst15, @qty104, NULL),
  -- Pack S WHT: WIP-WSH-S + Label + Box 6-Pack
  (@cons52, @it39, @qty291, @pdst16, @qty292, NULL),
  (@cons34, @it32, @qty106, @pdst16, @qty107, NULL),
  (@cons35, @it31, @qty108, @pdst16, @qty109, NULL),
  -- Pack M WHT: WIP-WSH-M + Label + Box 6-Pack
  (@cons53, @it40, @qty293, @pdst17, @qty294, NULL),
  (@cons36, @it32, @qty110, @pdst17, @qty111, NULL),
  (@cons37, @it31, @qty112, @pdst17, @qty113, NULL),
  -- Pack L WHT: WIP-WSH-L + Label + Box 12-Pack
  (@cons54, @it41, @qty295, @pdst18, @qty296, NULL),
  (@cons38, @it32, @qty114, @pdst18, @qty115, NULL),
  (@cons39, @it30, @qty116, @pdst18, @qty117, NULL),
  -- Pack S BLK: WIP-DYE-S-BLK + Label + Box 6-Pack
  (@cons55, @it42, @qty297, @pdst19, @qty298, NULL),
  (@cons40, @it32, @qty261, @pdst19, @qty262, NULL),
  (@cons41, @it31, @qty263, @pdst19, @qty264, NULL),
  -- Pack S NAV: WIP-DYE-S-NAV + Label + Box 6-Pack
  (@cons56, @it43, @qty299, @pdst20, @qty300, NULL),
  (@cons42, @it32, @qty265, @pdst20, @qty266, NULL),
  (@cons43, @it31, @qty267, @pdst20, @qty268, NULL),
  -- Pack M BLK: WIP-DYE-M-BLK + Label + Box 6-Pack
  (@cons57, @it44, @qty301, @pdst21, @qty302, NULL),
  (@cons44, @it32, @qty269, @pdst21, @qty270, NULL),
  (@cons45, @it31, @qty271, @pdst21, @qty272, NULL),
  -- Pack M NAV: WIP-DYE-M-NAV + Label + Box 6-Pack
  (@cons58, @it45, @qty303, @pdst22, @qty304, NULL),
  (@cons46, @it32, @qty273, @pdst22, @qty274, NULL),
  (@cons47, @it31, @qty275, @pdst22, @qty276, NULL),
  -- Pack L BLK: WIP-DYE-L-BLK + Label + Box 12-Pack
  (@cons59, @it46, @qty305, @pdst23, @qty306, NULL),
  (@cons48, @it32, @qty277, @pdst23, @qty278, NULL),
  (@cons49, @it30, @qty279, @pdst23, @qty280, NULL),
  -- Pack L NAV: WIP-DYE-L-NAV + Label + Box 12-Pack
  (@cons60, @it47, @qty307, @pdst24, @qty308, NULL),
  (@cons50, @it32, @qty281, @pdst24, @qty282, NULL),
  (@cons51, @it30, @qty283, @pdst24, @qty284, NULL);


-- ============================================================================
-- SECTION 25: PRODUCTIONS
-- ============================================================================
-- production columns: id, item_id, quantity_id, production_step_id

INSERT INTO `production` (`id`, `item_id`, `quantity_id`, `production_step_id`) VALUES
  -- Knit -> WIP-KNT
  (@prod1, @it33, @qty29, @pdst1),
  (@prod2, @it34, @qty34, @pdst2),
  (@prod3, @it35, @qty39, @pdst3),
  -- Link -> WIP-LNK
  (@prod4, @it36, @qty44, @pdst4),
  (@prod5, @it37, @qty49, @pdst5),
  (@prod6, @it38, @qty54, @pdst6),
  -- Wash -> WIP-WSH
  (@prod7, @it39, @qty61, @pdst7),
  (@prod8, @it40, @qty68, @pdst8),
  (@prod9, @it41, @qty75, @pdst9),
  -- Dye -> WIP-DYE
  (@prod10, @it42, @qty80, @pdst10),
  (@prod11, @it43, @qty85, @pdst11),
  (@prod12, @it44, @qty90, @pdst12),
  (@prod13, @it45, @qty95, @pdst13),
  (@prod14, @it46, @qty100, @pdst14),
  (@prod15, @it47, @qty105, @pdst15);


-- ============================================================================
-- SECTION 26: PARENT-CHILD PRODUCTION STEP LINKS
-- ============================================================================
-- Prisma implicit m2m: row (A, B) means B is in A's `in` field, A is in B's `out` field
-- So A = the downstream step, B = the upstream step (direction: B -> A)
-- Flow: Knit -> Link -> Wash -> Dye (BLK/NAV) -> Pack
--                              Wash -> Pack (white socks skip dye)

INSERT INTO `_parent_child_production_steps` (`A`, `B`) VALUES
  -- Knit -> Link
  (@pdst4, @pdst1),   -- Knit S -> Link S
  (@pdst5, @pdst2),   -- Knit M -> Link M
  (@pdst6, @pdst3),   -- Knit L -> Link L
  -- Link -> Wash
  (@pdst7, @pdst4),   -- Link S -> Wash S
  (@pdst8, @pdst5),   -- Link M -> Wash M
  (@pdst9, @pdst6),   -- Link L -> Wash L
  -- Wash -> Dye (for BLK/NAV socks)
  (@pdst10, @pdst7),  -- Wash S -> Dye S BLK
  (@pdst11, @pdst7),  -- Wash S -> Dye S NAV
  (@pdst12, @pdst8),  -- Wash M -> Dye M BLK
  (@pdst13, @pdst8),  -- Wash M -> Dye M NAV
  (@pdst14, @pdst9),  -- Wash L -> Dye L BLK
  (@pdst15, @pdst9),  -- Wash L -> Dye L NAV
  -- Wash -> Pack WHT (white socks skip dye)
  (@pdst16, @pdst7),  -- Wash S -> Pack S WHT
  (@pdst17, @pdst8),  -- Wash M -> Pack M WHT
  (@pdst18, @pdst9),  -- Wash L -> Pack L WHT
  -- Dye -> Pack BLK/NAV
  (@pdst19, @pdst10), -- Dye S BLK -> Pack S BLK
  (@pdst20, @pdst11), -- Dye S NAV -> Pack S NAV
  (@pdst21, @pdst12), -- Dye M BLK -> Pack M BLK
  (@pdst22, @pdst13), -- Dye M NAV -> Pack M NAV
  (@pdst23, @pdst14), -- Dye L BLK -> Pack L BLK
  (@pdst24, @pdst15); -- Dye L NAV -> Pack L NAV


-- ============================================================================
-- SECTION 27: INVENTORY QUANTITIES
-- ============================================================================
-- 2 quantities per item (1 for inventory_log, 1 for inventory_change_log)
-- All starting at 0 in the base unit of the item's unit group

INSERT INTO `quantity` (`id`, `value`, `unit_id`) VALUES
  -- Inventory log quantities (qty_seed000000118..164)
  (@qty118, 0, @un2),  -- it_seed000000001 FG-CRW-S-WHT
  (@qty119, 0, @un2),  -- it_seed000000002
  (@qty120, 0, @un2),  -- it_seed000000003
  (@qty121, 0, @un2),  -- it_seed000000004
  (@qty122, 0, @un2),  -- it_seed000000005
  (@qty123, 0, @un2),  -- it_seed000000006
  (@qty124, 0, @un2),  -- it_seed000000007
  (@qty125, 0, @un2),  -- it_seed000000008
  (@qty126, 0, @un2),  -- it_seed000000009
  (@qty127, 0, @un2),  -- it_seed000000010 FG-ANK-S-WHT
  (@qty128, 0, @un2),  -- it_seed000000011
  (@qty129, 0, @un2),  -- it_seed000000012
  (@qty130, 0, @un2),  -- it_seed000000013
  (@qty131, 0, @un2),  -- it_seed000000014
  (@qty132, 0, @un2),  -- it_seed000000015
  (@qty133, 0, @un2),  -- it_seed000000016
  (@qty134, 0, @un2),  -- it_seed000000017
  (@qty135, 0, @un2),  -- it_seed000000018
  (@qty136, 0, @un1),  -- it_seed000000019 Shipping
  (@qty137, 0, @un1),  -- it_seed000000020 Credit
  (@qty138, 0, @un8),  -- it_seed000000021 RM-YRN-COT70
  (@qty139, 0, @un8),  -- it_seed000000022 RM-YRN-COT40
  (@qty140, 0, @un8),  -- it_seed000000023 RM-YRN-NYL40
  (@qty141, 0, @un8),  -- it_seed000000024 RM-ELS-SPX
  (@qty142, 0, @un9),  -- it_seed000000025 RM-DYE-BLK
  (@qty143, 0, @un9),  -- it_seed000000026 RM-DYE-NAV
  (@qty144, 0, @un9),  -- it_seed000000027 RM-DYE-BGE
  (@qty145, 0, @un9),  -- it_seed000000028 RM-CHM-SOF
  (@qty146, 0, @un9),  -- it_seed000000029 RM-CHM-DET
  (@qty147, 0, @un1),  -- it_seed000000030 RM-PKG-BX12
  (@qty148, 0, @un1),  -- it_seed000000031 RM-PKG-BX06
  (@qty149, 0, @un1),  -- it_seed000000032 RM-PKG-LBL
  (@qty150, 0, @un2),  -- it_seed000000033 WIP-KNT-S
  (@qty151, 0, @un2),  -- it_seed000000034 WIP-KNT-M
  (@qty152, 0, @un2),  -- it_seed000000035 WIP-KNT-L
  (@qty153, 0, @un2),  -- it_seed000000036 WIP-LNK-S
  (@qty154, 0, @un2),  -- it_seed000000037 WIP-LNK-M
  (@qty155, 0, @un2),  -- it_seed000000038 WIP-LNK-L
  (@qty156, 0, @un2),  -- it_seed000000039 WIP-WSH-S
  (@qty157, 0, @un2),  -- it_seed000000040 WIP-WSH-M
  (@qty158, 0, @un2),  -- it_seed000000041 WIP-WSH-L
  (@qty159, 0, @un2),  -- it_seed000000042 WIP-DYE-S-BLK
  (@qty160, 0, @un2),  -- it_seed000000043 WIP-DYE-S-NAV
  (@qty161, 0, @un2),  -- it_seed000000044 WIP-DYE-M-BLK
  (@qty162, 0, @un2),  -- it_seed000000045 WIP-DYE-M-NAV
  (@qty163, 0, @un2),  -- it_seed000000046 WIP-DYE-L-BLK
  (@qty164, 0, @un2),  -- it_seed000000047 WIP-DYE-L-NAV
  -- Inventory change log quantities (qty_seed000000165..211)
  (@qty165, 0, @un2),
  (@qty166, 0, @un2),
  (@qty167, 0, @un2),
  (@qty168, 0, @un2),
  (@qty169, 0, @un2),
  (@qty170, 0, @un2),
  (@qty171, 0, @un2),
  (@qty172, 0, @un2),
  (@qty173, 0, @un2),
  (@qty174, 0, @un2),
  (@qty175, 0, @un2),
  (@qty176, 0, @un2),
  (@qty177, 0, @un2),
  (@qty178, 0, @un2),
  (@qty179, 0, @un2),
  (@qty180, 0, @un2),
  (@qty181, 0, @un2),
  (@qty182, 0, @un2),
  (@qty183, 0, @un1),
  (@qty184, 0, @un1),
  (@qty185, 0, @un8),
  (@qty186, 0, @un8),
  (@qty187, 0, @un8),
  (@qty188, 0, @un8),
  (@qty189, 0, @un9),
  (@qty190, 0, @un9),
  (@qty191, 0, @un9),
  (@qty192, 0, @un9),
  (@qty193, 0, @un9),
  (@qty194, 0, @un1),
  (@qty195, 0, @un1),
  (@qty196, 0, @un1),
  (@qty197, 0, @un2),
  (@qty198, 0, @un2),
  (@qty199, 0, @un2),
  (@qty200, 0, @un2),
  (@qty201, 0, @un2),
  (@qty202, 0, @un2),
  (@qty203, 0, @un2),
  (@qty204, 0, @un2),
  (@qty205, 0, @un2),
  (@qty206, 0, @un2),
  (@qty207, 0, @un2),
  (@qty208, 0, @un2),
  (@qty209, 0, @un2),
  (@qty210, 0, @un2),
  (@qty211, 0, @un2);


-- ============================================================================
-- SECTION 28: INVENTORY LOGS
-- ============================================================================

INSERT INTO `inventory_log` (`id`, `item_id`, `quantity_id`, `account_id`) VALUES
  (@invlog1, @it1, @qty118, '@account_id'),
  (@invlog2, @it2, @qty119, '@account_id'),
  (@invlog3, @it3, @qty120, '@account_id'),
  (@invlog4, @it4, @qty121, '@account_id'),
  (@invlog5, @it5, @qty122, '@account_id'),
  (@invlog6, @it6, @qty123, '@account_id'),
  (@invlog7, @it7, @qty124, '@account_id'),
  (@invlog8, @it8, @qty125, '@account_id'),
  (@invlog9, @it9, @qty126, '@account_id'),
  (@invlog10, @it10, @qty127, '@account_id'),
  (@invlog11, @it11, @qty128, '@account_id'),
  (@invlog12, @it12, @qty129, '@account_id'),
  (@invlog13, @it13, @qty130, '@account_id'),
  (@invlog14, @it14, @qty131, '@account_id'),
  (@invlog15, @it15, @qty132, '@account_id'),
  (@invlog16, @it16, @qty133, '@account_id'),
  (@invlog17, @it17, @qty134, '@account_id'),
  (@invlog18, @it18, @qty135, '@account_id'),
  (@invlog19, @it19, @qty136, '@account_id'),
  (@invlog20, @it20, @qty137, '@account_id'),
  (@invlog21, @it21, @qty138, '@account_id'),
  (@invlog22, @it22, @qty139, '@account_id'),
  (@invlog23, @it23, @qty140, '@account_id'),
  (@invlog24, @it24, @qty141, '@account_id'),
  (@invlog25, @it25, @qty142, '@account_id'),
  (@invlog26, @it26, @qty143, '@account_id'),
  (@invlog27, @it27, @qty144, '@account_id'),
  (@invlog28, @it28, @qty145, '@account_id'),
  (@invlog29, @it29, @qty146, '@account_id'),
  (@invlog30, @it30, @qty147, '@account_id'),
  (@invlog31, @it31, @qty148, '@account_id'),
  (@invlog32, @it32, @qty149, '@account_id'),
  (@invlog33, @it33, @qty150, '@account_id'),
  (@invlog34, @it34, @qty151, '@account_id'),
  (@invlog35, @it35, @qty152, '@account_id'),
  (@invlog36, @it36, @qty153, '@account_id'),
  (@invlog37, @it37, @qty154, '@account_id'),
  (@invlog38, @it38, @qty155, '@account_id'),
  (@invlog39, @it39, @qty156, '@account_id'),
  (@invlog40, @it40, @qty157, '@account_id'),
  (@invlog41, @it41, @qty158, '@account_id'),
  (@invlog42, @it42, @qty159, '@account_id'),
  (@invlog43, @it43, @qty160, '@account_id'),
  (@invlog44, @it44, @qty161, '@account_id'),
  (@invlog45, @it45, @qty162, '@account_id'),
  (@invlog46, @it46, @qty163, '@account_id'),
  (@invlog47, @it47, @qty164, '@account_id');


-- ============================================================================
-- SECTION 29: INVENTORY CHANGE LOGS
-- ============================================================================

INSERT INTO `inventory_change_log` (`id`, `item_id`, `quantity_id`, `action_type_code`, `account_id`, `inventory_log_id`) VALUES
  (@invcl1, @it1, @qty165, 'create_record', '@account_id', @invlog1),
  (@invcl2, @it2, @qty166, 'create_record', '@account_id', @invlog2),
  (@invcl3, @it3, @qty167, 'create_record', '@account_id', @invlog3),
  (@invcl4, @it4, @qty168, 'create_record', '@account_id', @invlog4),
  (@invcl5, @it5, @qty169, 'create_record', '@account_id', @invlog5),
  (@invcl6, @it6, @qty170, 'create_record', '@account_id', @invlog6),
  (@invcl7, @it7, @qty171, 'create_record', '@account_id', @invlog7),
  (@invcl8, @it8, @qty172, 'create_record', '@account_id', @invlog8),
  (@invcl9, @it9, @qty173, 'create_record', '@account_id', @invlog9),
  (@invcl10, @it10, @qty174, 'create_record', '@account_id', @invlog10),
  (@invcl11, @it11, @qty175, 'create_record', '@account_id', @invlog11),
  (@invcl12, @it12, @qty176, 'create_record', '@account_id', @invlog12),
  (@invcl13, @it13, @qty177, 'create_record', '@account_id', @invlog13),
  (@invcl14, @it14, @qty178, 'create_record', '@account_id', @invlog14),
  (@invcl15, @it15, @qty179, 'create_record', '@account_id', @invlog15),
  (@invcl16, @it16, @qty180, 'create_record', '@account_id', @invlog16),
  (@invcl17, @it17, @qty181, 'create_record', '@account_id', @invlog17),
  (@invcl18, @it18, @qty182, 'create_record', '@account_id', @invlog18),
  (@invcl19, @it19, @qty183, 'create_record', '@account_id', @invlog19),
  (@invcl20, @it20, @qty184, 'create_record', '@account_id', @invlog20),
  (@invcl21, @it21, @qty185, 'create_record', '@account_id', @invlog21),
  (@invcl22, @it22, @qty186, 'create_record', '@account_id', @invlog22),
  (@invcl23, @it23, @qty187, 'create_record', '@account_id', @invlog23),
  (@invcl24, @it24, @qty188, 'create_record', '@account_id', @invlog24),
  (@invcl25, @it25, @qty189, 'create_record', '@account_id', @invlog25),
  (@invcl26, @it26, @qty190, 'create_record', '@account_id', @invlog26),
  (@invcl27, @it27, @qty191, 'create_record', '@account_id', @invlog27),
  (@invcl28, @it28, @qty192, 'create_record', '@account_id', @invlog28),
  (@invcl29, @it29, @qty193, 'create_record', '@account_id', @invlog29),
  (@invcl30, @it30, @qty194, 'create_record', '@account_id', @invlog30),
  (@invcl31, @it31, @qty195, 'create_record', '@account_id', @invlog31),
  (@invcl32, @it32, @qty196, 'create_record', '@account_id', @invlog32),
  (@invcl33, @it33, @qty197, 'create_record', '@account_id', @invlog33),
  (@invcl34, @it34, @qty198, 'create_record', '@account_id', @invlog34),
  (@invcl35, @it35, @qty199, 'create_record', '@account_id', @invlog35),
  (@invcl36, @it36, @qty200, 'create_record', '@account_id', @invlog36),
  (@invcl37, @it37, @qty201, 'create_record', '@account_id', @invlog37),
  (@invcl38, @it38, @qty202, 'create_record', '@account_id', @invlog38),
  (@invcl39, @it39, @qty203, 'create_record', '@account_id', @invlog39),
  (@invcl40, @it40, @qty204, 'create_record', '@account_id', @invlog40),
  (@invcl41, @it41, @qty205, 'create_record', '@account_id', @invlog41),
  (@invcl42, @it42, @qty206, 'create_record', '@account_id', @invlog42),
  (@invcl43, @it43, @qty207, 'create_record', '@account_id', @invlog43),
  (@invcl44, @it44, @qty208, 'create_record', '@account_id', @invlog44),
  (@invcl45, @it45, @qty209, 'create_record', '@account_id', @invlog45),
  (@invcl46, @it46, @qty210, 'create_record', '@account_id', @invlog46),
  (@invcl47, @it47, @qty211, 'create_record', '@account_id', @invlog47);


-- ============================================================================
-- SECTION 30: PACK PRODUCTION FIX
-- ============================================================================
-- Pack steps (pdst16-18) were missing production records. Every production step
-- needs exactly one production (output item). Pack steps produce finished goods.

INSERT INTO `quantity` (`id`, `value`, `unit_id`) VALUES
  (@qty212, 1, @un2),  -- Pack S WHT produces 1 pair
  (@qty213, 1, @un2),  -- Pack M WHT produces 1 pair
  (@qty214, 1, @un2),  -- Pack L WHT produces 1 pair
  (@qty285, 1, @un2),  -- Pack S BLK produces 1 pair
  (@qty286, 1, @un2),  -- Pack S NAV produces 1 pair
  (@qty287, 1, @un2),  -- Pack M BLK produces 1 pair
  (@qty288, 1, @un2),  -- Pack M NAV produces 1 pair
  (@qty289, 1, @un2),  -- Pack L BLK produces 1 pair
  (@qty290, 1, @un2);  -- Pack L NAV produces 1 pair

INSERT INTO `production` (`id`, `item_id`, `quantity_id`, `production_step_id`) VALUES
  (@prod16, @it1, @qty212, @pdst16),  -- Pack S WHT -> FG-CRW-S-WHT
  (@prod17, @it4, @qty213, @pdst17),  -- Pack M WHT -> FG-CRW-M-WHT
  (@prod18, @it7, @qty214, @pdst18),  -- Pack L WHT -> FG-CRW-L-WHT
  (@prod19, @it2, @qty285, @pdst19),  -- Pack S BLK -> FG-CRW-S-BLK
  (@prod20, @it3, @qty286, @pdst20),  -- Pack S NAV -> FG-CRW-S-NAV
  (@prod21, @it5, @qty287, @pdst21),  -- Pack M BLK -> FG-CRW-M-BLK
  (@prod22, @it6, @qty288, @pdst22),  -- Pack M NAV -> FG-CRW-M-NAV
  (@prod23, @it8, @qty289, @pdst23),  -- Pack L BLK -> FG-CRW-L-BLK
  (@prod24, @it9, @qty290, @pdst24);  -- Pack L NAV -> FG-CRW-L-NAV


-- ============================================================================
-- SECTION 32: ACCOUNT GROUP
-- ============================================================================

INSERT INTO `account_group` (`id`, `owner_account_id`, `name`, `description`, `commission_status_code`, `freight_status_code`, `account_group_type_code`) VALUES
  (@acgrp1, '@account_id', 'Wholesale', 'Wholesale distribution customers', 'commission_applied', 'billed_freight', 'type_group');


-- ============================================================================
-- SECTION 35: CUSTOMER ACCOUNTS
-- ============================================================================

INSERT INTO `account` (`id`, `name`, `account_type_code`, `onboarding_status_code`) VALUES
  (@cust1, 'Global Manufacturing Solutions', 'company', 'unclaimed'),
  (@cust2, 'Pacific Coast Distributors',     'company', 'unclaimed'),
  (@cust3, 'Northeast Medical Supplies',     'company', 'unclaimed');


-- ============================================================================
-- SECTION 36: CUSTOMER GEOLOCATIONS + ADDRESSES
-- ============================================================================

INSERT INTO `geolocation` (`id`, `street_line_1`, `street_line_2`, `locality`, `state`, `postal_code`, `country`) VALUES
  -- Customer 1: Global Manufacturing Solutions
  (@geo1, '100 Industrial Blvd',  'Suite 200', 'Chicago',     'IL', '60601', 'US'),
  (@geo2, '100 Industrial Blvd',  'Dock 4',    'Chicago',     'IL', '60601', 'US'),
  -- Customer 2: Pacific Coast Distributors
  (@geo3, '2500 Harbor Dr',       NULL,         'Los Angeles', 'CA', '90731', 'US'),
  (@geo4, '2510 Warehouse Way',   'Bay 12',    'Los Angeles', 'CA', '90731', 'US'),
  -- Customer 3: Northeast Medical Supplies
  (@geo5, '75 Commerce St',       'Floor 3',   'Boston',      'MA', '02110', 'US'),
  (@geo6, '80 Distribution Ave',  NULL,         'Boston',      'MA', '02110', 'US');

INSERT INTO `address` (`id`, `name`, `phone`, `email`, `is_drop_ship`, `geolocation_id`) VALUES
  -- Customer 1 billing / shipping
  (@caddr1, 'Global Mfg - Billing',  '312-555-0100', 'ap@globalmfg.example.com',  false, @geo1),
  (@caddr2, 'Global Mfg - Shipping', '312-555-0101', NULL,                        false, @geo2),
  -- Customer 2 billing / shipping
  (@caddr3, 'Pacific Coast - Billing',  '310-555-0200', 'billing@paccoast.example.com', false, @geo3),
  (@caddr4, 'Pacific Coast - Shipping', '310-555-0201', NULL,                          false, @geo4),
  -- Customer 3 billing / shipping
  (@caddr5, 'NE Medical - Billing',  '617-555-0300', 'accounts@nemedical.example.com', false, @geo5),
  (@caddr6, 'NE Medical - Shipping', '617-555-0301', NULL,                             false, @geo6);


-- ============================================================================
-- SECTION 33: PAYMENT TERMS
-- ============================================================================

INSERT INTO `payment_term` (`id`, `is_active`, `name`, `account_id`) VALUES
  (@pytm1, 1, 'Net 30', '@account_id'),
  (@pytm2, 1, 'Net 60', '@account_id');


-- ============================================================================
-- SECTION 34: SHIPPING TERMS
-- ============================================================================

INSERT INTO `shipping_term` (`id`, `name`, `is_freight_exempt`, `is_carrier_rate`, `account_id`) VALUES
  (@shtm1, 'Prepaid', 0, 1, '@account_id');


-- ============================================================================
-- SECTION 37: ACCOUNT ADDRESSES + ACCOUNT RELATIONS
-- ============================================================================

INSERT INTO `account_address` (`id`, `account_id`, `address_id`) VALUES
  (@acadr1, @cust1, @caddr1),
  (@acadr2, @cust1, @caddr2),
  (@acadr3, @cust2, @caddr3),
  (@acadr4, @cust2, @caddr4),
  (@acadr5, @cust3, @caddr5),
  (@acadr6, @cust3, @caddr6);

UPDATE `account` SET default_billing_address_id = @caddr1, default_shipping_address_id = @caddr2 WHERE id = @cust1;
UPDATE `account` SET default_billing_address_id = @caddr3, default_shipping_address_id = @caddr4 WHERE id = @cust2;
UPDATE `account` SET default_billing_address_id = @caddr5, default_shipping_address_id = @caddr6 WHERE id = @cust3;

INSERT INTO `account_relation` (`id`, `owner_account_id`, `counterparty_account_id`, `account_relation_role_code`, `external_number`, `priority_code`, `account_group_id`, `payment_term_id`, `shipping_term_id`, `default_carrier_id`, `default_carrier_option_id`, `default_billing_address_id`, `default_shipping_address_id`, `default_sales_rep_id`, `account_status_code`, `commission_status_code`, `freight_status_code`) VALUES
  (@acrel1, '@account_id', @cust1, 'customer', 'CUST-001', 'normal', @acgrp1, @pytm1, @shtm1, 'delivery', NULL, @caddr1, @caddr2, @acus1, 'normal', 'commission_applied', 'billed_freight'),
  (@acrel2, '@account_id', @cust2, 'customer', 'CUST-002', 'normal', @acgrp1, @pytm1, @shtm1, 'delivery', NULL, @caddr3, @caddr4, @acus1, 'normal', 'commission_applied', 'billed_freight'),
  (@acrel3, '@account_id', @cust3, 'customer', 'CUST-003', 'normal', @acgrp1, @pytm2, @shtm1, 'delivery', NULL, @caddr5, @caddr6, @acus1, 'normal', 'commission_applied', 'billed_freight');


-- ============================================================================
-- SECTION 38: ORDER LINE RATES + QUANTITIES
-- ============================================================================
-- Unit prices (18 rates: 1 per line)
-- Unit costs (12 rates: product lines only, shipping lines have NULL cost)

INSERT INTO `rate` (`id`, `value`, `numerator_unit_id`, `denominator_unit_id`) VALUES
  -- EST-001 line prices
  (@rt196, 10,  @un4, @un2),  -- CRW-S-WHT unit_price $10/pr
  (@rt197, 9,   @un4, @un2),  -- ANK-M-BLK unit_price $9/pr
  (@rt198, 25,  @un4, @un1),  -- Shipping unit_price $25/ea
  -- ORD-001 line prices
  (@rt199, 12,  @un4, @un2),  -- CRW-L-NAV unit_price $12/pr
  (@rt200, 8,   @un4, @un2),  -- ANK-S-WHT unit_price $8/pr
  (@rt201, 35,  @un4, @un1),  -- Shipping unit_price $35/ea
  -- ORD-002 line prices
  (@rt202, 11,  @un4, @un2),  -- CRW-M-WHT unit_price $11/pr
  (@rt203, 12,  @un4, @un2),  -- CRW-L-BLK unit_price $12/pr
  (@rt204, 45,  @un4, @un1),  -- Shipping unit_price $45/ea
  -- ORD-003 line prices
  (@rt205, 10,  @un4, @un2),  -- ANK-L-NAV unit_price $10/pr
  (@rt206, 10,  @un4, @un2),  -- CRW-S-BLK unit_price $10/pr
  (@rt207, 30,  @un4, @un1),  -- Shipping unit_price $30/ea
  -- ORD-004 line prices
  (@rt208, 9,   @un4, @un2),  -- ANK-M-NAV unit_price $9/pr
  (@rt209, 11,  @un4, @un2),  -- CRW-M-BLK unit_price $11/pr
  (@rt210, 40,  @un4, @un1),  -- Shipping unit_price $40/ea
  -- ORD-005 line prices
  (@rt211, 12,  @un4, @un2),  -- CRW-L-WHT unit_price $12/pr
  (@rt212, 10,  @un4, @un2),  -- ANK-L-BLK unit_price $10/pr
  (@rt213, 25,  @un4, @un1),  -- Shipping unit_price $25/ea

  -- Unit costs (product lines only)
  -- EST-001
  (@rt214, 7,    @un4, @un2),  -- CRW-S-WHT cost
  (@rt215, 6,    @un4, @un2),  -- ANK-M-BLK cost
  -- ORD-001
  (@rt216, 8,    @un4, @un2),  -- CRW-L-NAV cost
  (@rt217, 5.5,  @un4, @un2),  -- ANK-S-WHT cost
  -- ORD-002
  (@rt218, 7.5,  @un4, @un2),  -- CRW-M-WHT cost
  (@rt219, 8,    @un4, @un2),  -- CRW-L-BLK cost
  -- ORD-003
  (@rt220, 6.5,  @un4, @un2),  -- ANK-L-NAV cost
  (@rt221, 7,    @un4, @un2),  -- CRW-S-BLK cost
  -- ORD-004
  (@rt222, 6,    @un4, @un2),  -- ANK-M-NAV cost
  (@rt223, 7.5,  @un4, @un2),  -- CRW-M-BLK cost
  -- ORD-005
  (@rt224, 8,    @un4, @un2),  -- CRW-L-WHT cost
  (@rt225, 6.5,  @un4, @un2);  -- ANK-L-BLK cost

INSERT INTO `quantity` (`id`, `value`, `unit_id`) VALUES
  -- EST-001 line quantities
  (@qty215, 24,  @un2),  -- 24 pairs CRW-S-WHT
  (@qty216, 36,  @un2),  -- 36 pairs ANK-M-BLK
  (@qty217, 1,   @un1),  -- 1 ea shipping
  -- ORD-001 line quantities
  (@qty218, 48,  @un2),  -- 48 pairs CRW-L-NAV
  (@qty219, 60,  @un2),  -- 60 pairs ANK-S-WHT
  (@qty220, 1,   @un1),  -- 1 ea shipping
  -- ORD-002 line quantities
  (@qty221, 72,  @un2),  -- 72 pairs CRW-M-WHT
  (@qty222, 36,  @un2),  -- 36 pairs CRW-L-BLK
  (@qty223, 1,   @un1),  -- 1 ea shipping
  -- ORD-003 line quantities
  (@qty224, 48,  @un2),  -- 48 pairs ANK-L-NAV
  (@qty225, 24,  @un2),  -- 24 pairs CRW-S-BLK
  (@qty226, 1,   @un1),  -- 1 ea shipping
  -- ORD-004 line quantities
  (@qty227, 36,  @un2),  -- 36 pairs ANK-M-NAV
  (@qty228, 48,  @un2),  -- 48 pairs CRW-M-BLK
  (@qty229, 1,   @un1),  -- 1 ea shipping
  -- ORD-005 line quantities
  (@qty230, 60,  @un2),  -- 60 pairs CRW-L-WHT
  (@qty231, 24,  @un2),  -- 24 pairs ANK-L-BLK
  (@qty232, 1,   @un1);  -- 1 ea shipping


-- ============================================================================
-- SECTION 39: SALES ORDERS + LINES
-- ============================================================================

INSERT INTO `sales_order` (`id`, `number`, `sales_order_status_code`, `sales_order_type_code`, `priority_code`, `buyer_account_id`, `seller_account_id`, `owner_account_id`, `billing_address_id`, `shipping_address_id`, `carrier_id`, `carrier_option_id`, `payment_term_id`, `shipping_term_id`, `sales_rep_id`, `issued_at`, `completed_at`) VALUES
  -- EST-001: estimate for Customer 1
  (@so1, 'EST-001', 'estimate',  'sales_order', 'normal', @cust1, '@account_id', '@account_id', @caddr1, @caddr2, 'delivery', NULL, @pytm1, @shtm1, @acus1, NULL, NULL),
  -- ORD-001: issued for Customer 1
  (@so2, 'ORD-001', 'issued',    'sales_order', 'normal', @cust1, '@account_id', '@account_id', @caddr1, @caddr2, 'delivery', NULL, @pytm1, @shtm1, @acus1, NOW() - INTERVAL 10 DAY, NULL),
  -- ORD-002: issued for Customer 2
  (@so3, 'ORD-002', 'issued',    'sales_order', 'normal', @cust2, '@account_id', '@account_id', @caddr3, @caddr4, 'delivery', NULL, @pytm1, @shtm1, @acus1, NOW() - INTERVAL 8 DAY, NULL),
  -- ORD-003: issued for Customer 2 (packed shipment)
  (@so4, 'ORD-003', 'issued',    'sales_order', 'high',   @cust2, '@account_id', '@account_id', @caddr3, @caddr4, 'delivery', NULL, @pytm1, @shtm1, @acus1, NOW() - INTERVAL 6 DAY, NULL),
  -- ORD-004: fulfilled for Customer 1
  (@so5, 'ORD-004', 'fulfilled', 'sales_order', 'normal', @cust1, '@account_id', '@account_id', @caddr1, @caddr2, 'delivery', NULL, @pytm1, @shtm1, @acus1, NOW() - INTERVAL 14 DAY, NOW() - INTERVAL 2 DAY),
  -- ORD-005: fulfilled for Customer 3
  (@so6, 'ORD-005', 'fulfilled', 'sales_order', 'normal', @cust3, '@account_id', '@account_id', @caddr5, @caddr6, 'delivery', NULL, @pytm2, @shtm1, @acus1, NOW() - INTERVAL 12 DAY, NOW() - INTERVAL 1 DAY);

INSERT INTO `sales_order_line` (`id`, `product_sku`, `product_description`, `line_item_number`, `product_id`, `item_id`, `sales_order_id`, `quantity_id`, `unit_price_id`, `unit_cost_id`) VALUES
  -- EST-001 lines
  (@sol1,  'FG-CRW-S-WHT', 'Crew Sock Small White',    1, @pd1,  @it1,  @so1, @qty215, @rt196, @rt214),
  (@sol2,  'FG-ANK-M-BLK', 'Ankle Sock Medium Black',  2, @pd14, @it14, @so1, @qty216, @rt197, @rt215),
  (@sol3,  'Shipping',     'Shipping',                  3, @pd19, @it19, @so1, @qty217, @rt198, NULL),
  -- ORD-001 lines
  (@sol4,  'FG-CRW-L-NAV', 'Crew Sock Large Navy',     1, @pd9,  @it9,  @so2, @qty218, @rt199, @rt216),
  (@sol5,  'FG-ANK-S-WHT', 'Ankle Sock Small White',   2, @pd10, @it10, @so2, @qty219, @rt200, @rt217),
  (@sol6,  'Shipping',     'Shipping',                  3, @pd19, @it19, @so2, @qty220, @rt201, NULL),
  -- ORD-002 lines
  (@sol7,  'FG-CRW-M-WHT', 'Crew Sock Medium White',   1, @pd4,  @it4,  @so3, @qty221, @rt202, @rt218),
  (@sol8,  'FG-CRW-L-BLK', 'Crew Sock Large Black',    2, @pd8,  @it8,  @so3, @qty222, @rt203, @rt219),
  (@sol9,  'Shipping',     'Shipping',                  3, @pd19, @it19, @so3, @qty223, @rt204, NULL),
  -- ORD-003 lines
  (@sol10, 'FG-ANK-L-NAV', 'Ankle Sock Large Navy',    1, @pd18, @it18, @so4, @qty224, @rt205, @rt220),
  (@sol11, 'FG-CRW-S-BLK', 'Crew Sock Small Black',    2, @pd2,  @it2,  @so4, @qty225, @rt206, @rt221),
  (@sol12, 'Shipping',     'Shipping',                  3, @pd19, @it19, @so4, @qty226, @rt207, NULL),
  -- ORD-004 lines
  (@sol13, 'FG-ANK-M-NAV', 'Ankle Sock Medium Navy',   1, @pd15, @it15, @so5, @qty227, @rt208, @rt222),
  (@sol14, 'FG-CRW-M-BLK', 'Crew Sock Medium Black',   2, @pd5,  @it5,  @so5, @qty228, @rt209, @rt223),
  (@sol15, 'Shipping',     'Shipping',                  3, @pd19, @it19, @so5, @qty229, @rt210, NULL),
  -- ORD-005 lines
  (@sol16, 'FG-CRW-L-WHT', 'Crew Sock Large White',    1, @pd7,  @it7,  @so6, @qty230, @rt211, @rt224),
  (@sol17, 'FG-ANK-L-BLK', 'Ankle Sock Large Black',   2, @pd17, @it17, @so6, @qty231, @rt212, @rt225),
  (@sol18, 'Shipping',     'Shipping',                  3, @pd19, @it19, @so6, @qty232, @rt213, NULL);


-- ============================================================================
-- SECTION 40: PICKS + PICK LINES
-- ============================================================================

INSERT INTO `quantity` (`id`, `value`, `unit_id`) VALUES
  -- Pick line quantities (match order line quantities for product lines)
  (@qty233, 48, @un2),  -- PICK-001: CRW-L-NAV
  (@qty234, 60, @un2),  -- PICK-001: ANK-S-WHT
  (@qty235, 72, @un2),  -- PICK-002: CRW-M-WHT
  (@qty236, 36, @un2),  -- PICK-002: CRW-L-BLK
  (@qty237, 48, @un2),  -- PICK-003: ANK-L-NAV
  (@qty238, 24, @un2),  -- PICK-003: CRW-S-BLK
  (@qty239, 36, @un2),  -- PICK-004: ANK-M-NAV
  (@qty240, 48, @un2),  -- PICK-004: CRW-M-BLK
  (@qty241, 60, @un2),  -- PICK-005: CRW-L-WHT
  (@qty242, 24, @un2);  -- PICK-005: ANK-L-BLK

INSERT INTO `pick` (`id`, `number`, `sales_order_id`, `account_id`, `finished_at`) VALUES
  (@pick1, 'PICK-001', @so2, '@account_id', NULL),                        -- ORD-001: open
  (@pick2, 'PICK-002', @so3, '@account_id', NULL),                        -- ORD-002: open
  (@pick3, 'PICK-003', @so4, '@account_id', NOW() - INTERVAL 4 DAY),     -- ORD-003: finished
  (@pick4, 'PICK-004', @so5, '@account_id', NOW() - INTERVAL 5 DAY),     -- ORD-004: finished
  (@pick5, 'PICK-005', @so6, '@account_id', NOW() - INTERVAL 3 DAY);     -- ORD-005: finished

INSERT INTO `pick_line` (`id`, `pick_id`, `quantity_id`, `sales_order_line_id`, `packed_at`) VALUES
  -- PICK-001 (open, not packed)
  (@pkl1, @pick1, @qty233, @sol4,  NULL),
  (@pkl2, @pick1, @qty234, @sol5,  NULL),
  -- PICK-002 (open, not packed)
  (@pkl3, @pick2, @qty235, @sol7,  NULL),
  (@pkl4, @pick2, @qty236, @sol8,  NULL),
  -- PICK-003 (finished + packed)
  (@pkl5, @pick3, @qty237, @sol10, NOW() - INTERVAL 4 DAY),
  (@pkl6, @pick3, @qty238, @sol11, NOW() - INTERVAL 4 DAY),
  -- PICK-004 (finished + packed)
  (@pkl7, @pick4, @qty239, @sol13, NOW() - INTERVAL 5 DAY),
  (@pkl8, @pick4, @qty240, @sol14, NOW() - INTERVAL 5 DAY),
  -- PICK-005 (finished + packed)
  (@pkl9, @pick5, @qty241, @sol16, NOW() - INTERVAL 3 DAY),
  (@pkl10, @pick5, @qty242, @sol17, NOW() - INTERVAL 3 DAY);

-- ============================================================================
-- SECTION 41: SHIPMENTS + SHIPMENT LINES
-- ============================================================================

INSERT INTO `quantity` (`id`, `value`, `unit_id`) VALUES
  -- Shipment line quantities
  (@qty243, 48, @un2),  -- SHP-001: ANK-L-NAV
  (@qty244, 24, @un2),  -- SHP-001: CRW-S-BLK
  (@qty245, 36, @un2),  -- SHP-002: ANK-M-NAV
  (@qty246, 48, @un2),  -- SHP-002: CRW-M-BLK
  (@qty247, 60, @un2),  -- SHP-003: CRW-L-WHT
  (@qty248, 24, @un2);  -- SHP-003: ANK-L-BLK

INSERT INTO `shipment` (`id`, `number`, `sales_order_id`, `carrier_id`, `carrier_option_id`, `shipping_address_id`, `shipment_status_code`, `account_id`, `invoice_id`, `shipped_at`, `master_tracking_number`) VALUES
  -- SHP-001: ORD-003, packed (not yet shipped)
  (@shp1, 'SHP-001', @so4, 'delivery', NULL, @caddr4, 'packed',  '@account_id', NULL,   NULL,                       NULL),
  -- SHP-002: ORD-004, shipped
  (@shp2, 'SHP-002', @so5, 'delivery', NULL, @caddr2, 'shipped', '@account_id', @inv1,  NOW() - INTERVAL 3 DAY,     'FEDEX-MTN-001'),
  -- SHP-003: ORD-005, shipped
  (@shp3, 'SHP-003', @so6, 'delivery', NULL, @caddr6, 'shipped', '@account_id', @inv2,  NOW() - INTERVAL 2 DAY,     'FEDEX-MTN-002');

INSERT INTO `shipment_line` (`id`, `shipment_id`, `sales_order_line_id`, `quantity_id`) VALUES
  -- SHP-001 lines (ORD-003 product lines)
  (@shpl1, @shp1, @sol10, @qty243),
  (@shpl2, @shp1, @sol11, @qty244),
  -- SHP-002 lines (ORD-004 product lines)
  (@shpl3, @shp2, @sol13, @qty245),
  (@shpl4, @shp2, @sol14, @qty246),
  -- SHP-003 lines (ORD-005 product lines)
  (@shpl5, @shp3, @sol16, @qty247),
  (@shpl6, @shp3, @sol17, @qty248);


-- ============================================================================
-- SECTION 42: SHIPPING CASES
-- ============================================================================

INSERT INTO `quantity` (`id`, `value`, `unit_id`) VALUES
  -- Shipping case freight amounts (dollars) and weights (pounds)
  (@qty249, 0,    @un4),   -- SHP-001 CASE-001 freight_amount (not yet rated)
  (@qty250, 15,   @un8),   -- SHP-001 CASE-001 freight_weight 15lb
  (@qty251, 45,   @un4),   -- SHP-002 CASE-001 freight_amount $45
  (@qty252, 20,   @un8),   -- SHP-002 CASE-001 freight_weight 20lb
  (@qty253, 30,   @un4),   -- SHP-003 CASE-001 freight_amount $30
  (@qty254, 18,   @un8),   -- SHP-003 CASE-001 freight_weight 18lb
  (@qty255, 25,   @un4),   -- SHP-003 CASE-002 freight_amount $25
  (@qty256, 12,   @un8);   -- SHP-003 CASE-002 freight_weight 12lb

INSERT INTO `shipping_case` (`id`, `number`, `shipment_id`, `carrier_id`, `account_id`, `freight_amount_id`, `freight_weight_id`, `tracking_number`, `shipped_at`) VALUES
  (@shpc1, 'CASE-001', @shp1, 'delivery', '@account_id', @qty249, @qty250, NULL,              NULL),
  (@shpc2, 'CASE-001', @shp2, 'delivery', '@account_id', @qty251, @qty252, 'FEDEX-123456789', NOW() - INTERVAL 3 DAY),
  (@shpc3, 'CASE-001', @shp3, 'delivery', '@account_id', @qty253, @qty254, 'FEDEX-789012345', NOW() - INTERVAL 2 DAY),
  (@shpc4, 'CASE-002', @shp3, 'delivery', '@account_id', @qty255, @qty256, 'FEDEX-789012346', NOW() - INTERVAL 2 DAY);


-- ============================================================================
-- SECTION 43: INVOICES + INVOICE LINES
-- ============================================================================

INSERT INTO `quantity` (`id`, `value`, `unit_id`) VALUES
  -- Invoice line quantities
  (@qty257, 36, @un2),  -- INV-001: ANK-M-NAV
  (@qty258, 48, @un2),  -- INV-001: CRW-M-BLK
  (@qty259, 60, @un2),  -- INV-002: CRW-L-WHT
  (@qty260, 24, @un2);  -- INV-002: ANK-L-BLK

INSERT INTO `invoice` (`id`, `number`, `sales_order_id`, `billing_address_id`, `account_id`, `has_been_sent`, `is_paid_in_full`) VALUES
  (@inv1, 'INV-001', @so5, @caddr1, '@account_id', true,  false),
  (@inv2, 'INV-002', @so6, @caddr5, '@account_id', true,  false);

INSERT INTO `invoice_line` (`id`, `invoice_id`, `quantity_id`, `sales_order_line_id`) VALUES
  -- INV-001 lines (ORD-004 product lines)
  (@invl1, @inv1, @qty257, @sol13),
  (@invl2, @inv1, @qty258, @sol14),
  -- INV-002 lines (ORD-005 product lines)
  (@invl3, @inv2, @qty259, @sol16),
  (@invl4, @inv2, @qty260, @sol17);


-- ============================================================================
-- SECTION 44: PRODUCTION SHIFTS
-- ============================================================================
-- Two shifts on the standard five-day week. The schedule's capacity comes from
-- shifts_per_day and hours_per_shift on the settings row, not from these, but the
-- floor's downtime and attainment views read a shift off the calendar.

INSERT INTO `production_shift` (`id`, `account_id`, `department_id`, `code`, `name`, `start_time`, `end_time`, `crosses_midnight`, `days_of_week`, `is_active`, `sort_order`) VALUES
  (@pnsf1, '@account_id', NULL, 'day',   'Day Shift',   '06:00', '14:00', 0, '1111100', 1, 1),
  (@pnsf2, '@account_id', NULL, 'swing', 'Swing Shift', '14:00', '22:00', 0, '1111100', 1, 2);


-- ============================================================================
-- SECTION 45: SUPPLIERS
-- ============================================================================
-- Receiving resolves a supplier through the purchase order's seller_account_id and
-- the owner's account_relation, so a purchase order without one of these is unreadable.

INSERT INTO `account` (`id`, `name`, `account_type_code`, `onboarding_status_code`) VALUES
  (@supp1, 'Carolina Yarn Mills',   'company', 'unclaimed'),
  (@supp2, 'Atlantic Packaging Co', 'company', 'unclaimed');

INSERT INTO `geolocation` (`id`, `street_line_1`, `street_line_2`, `locality`, `state`, `postal_code`, `country`) VALUES
  (@sgeo1, '900 Mill Rd',     NULL, 'Gastonia',  'NC', '28052', 'US'),
  (@sgeo2, '410 Carton Way',  NULL, 'Charlotte', 'NC', '28206', 'US');

INSERT INTO `address` (`id`, `name`, `phone`, `email`, `is_drop_ship`, `geolocation_id`) VALUES
  (@saddr1, 'Carolina Yarn Mills',   '704-555-0400', 'orders@carolinayarn.example.com', false, @sgeo1),
  (@saddr2, 'Atlantic Packaging Co', '704-555-0500', 'sales@atlanticpkg.example.com',   false, @sgeo2);

INSERT INTO `account_address` (`id`, `account_id`, `address_id`) VALUES
  (@sacadr1, @supp1, @saddr1),
  (@sacadr2, @supp2, @saddr2);

UPDATE `account` SET default_billing_address_id = @saddr1, default_shipping_address_id = @saddr1 WHERE id = @supp1;
UPDATE `account` SET default_billing_address_id = @saddr2, default_shipping_address_id = @saddr2 WHERE id = @supp2;

INSERT INTO `account_relation` (`id`, `owner_account_id`, `counterparty_account_id`, `account_relation_role_code`, `external_number`, `priority_code`, `payment_term_id`, `shipping_term_id`, `default_carrier_id`, `default_billing_address_id`, `default_shipping_address_id`, `account_status_code`, `commission_status_code`, `freight_status_code`) VALUES
  (@acrel4, '@account_id', @supp1, 'supplier', 'SUP-001', 'normal', @pytm1, @shtm1, 'delivery', @saddr1, @saddr1, 'normal', 'commission_applied', 'billed_freight'),
  (@acrel5, '@account_id', @supp2, 'supplier', 'SUP-002', 'normal', @pytm2, @shtm1, 'delivery', @saddr2, @saddr2, 'normal', 'commission_applied', 'billed_freight');


-- ============================================================================
-- SECTION 46: SUPPLIER MATERIALS
-- ============================================================================
-- The supplier's own part number for a material, which is what a buyer quotes back
-- to them and what an inbound document is matched on.

INSERT INTO `supplier_material` (`id`, `material_id`, `supplier_account_id`, `supplier_part_number`, `supplier_description`, `is_active`, `owner_account_id`) VALUES
  (@suml1, @mat1, @supp1, 'CYM-COT-70', 'Combed cotton 70 denier, cone wound', 1, '@account_id'),
  (@suml2, @mat2, @supp1, 'CYM-COT-40', 'Combed cotton 40 denier, cone wound', 1, '@account_id'),
  (@suml3, @mat3, @supp1, 'CYM-NYL-40', 'Textured nylon 40 denier',            1, '@account_id'),
  (@suml4, @mat10, @supp2, 'APC-BX-12', 'Corrugated shipper, 12-pack',         1, '@account_id');


-- ============================================================================
-- SECTION 47: PURCHASE ORDERS + LINES
-- ============================================================================
-- Purchase orders share the sales_order table, distinguished by type and by the
-- account on each side: here the sandbox is the buyer and the supplier the seller.
-- PO-001 is fully received, PO-002 is still open so the receiving and on-order views
-- both have work in them.

INSERT INTO `rate` (`id`, `value`, `numerator_unit_id`, `denominator_unit_id`) VALUES
  (@rtpo1, 4.85, @un4, @un8),
  (@rtpo2, 4.40, @un4, @un8),
  (@rtpo3, 6.20, @un4, @un8),
  (@rtpo4, 1.15, @un4, @un1);

INSERT INTO `quantity` (`id`, `value`, `unit_id`) VALUES
  (@qpo1, 800, @un8),
  (@qpo2, 600, @un8),
  (@qpo3, 400, @un8),
  (@qpo4, 1200, @un1);

INSERT INTO `sales_order` (`id`, `number`, `sales_order_status_code`, `sales_order_type_code`, `priority_code`, `buyer_account_id`, `seller_account_id`, `owner_account_id`, `billing_address_id`, `shipping_address_id`, `carrier_id`, `carrier_option_id`, `payment_term_id`, `shipping_term_id`, `issued_at`, `completed_at`) VALUES
  (@po1, 'PO-001', 'fulfilled', 'purchase_order', 'normal', '@account_id', @supp1, '@account_id', @ownadr1, @ownadr2, 'delivery', NULL, @pytm1, @shtm1, NOW() - INTERVAL 45 DAY, NOW() - INTERVAL 38 DAY),
  (@po2, 'PO-002', 'issued',    'purchase_order', 'normal', '@account_id', @supp2, '@account_id', @ownadr1, @ownadr2, 'delivery', NULL, @pytm2, @shtm1, NOW() - INTERVAL 9 DAY, NULL);

INSERT INTO `sales_order_line` (`id`, `product_sku`, `product_description`, `line_item_number`, `product_id`, `item_id`, `sales_order_id`, `quantity_id`, `unit_price_id`, `unit_cost_id`) VALUES
  (@poln1, 'RM-YRN-COT70', 'Cotton Yarn 70D',        1, NULL, @it21, @po1, @qpo1, @rtpo1, @rtpo1),
  (@poln2, 'RM-YRN-COT40', 'Cotton Yarn 40D',        2, NULL, @it22, @po1, @qpo2, @rtpo2, @rtpo2),
  (@poln3, 'RM-YRN-NYL40', 'Nylon Yarn 40D',         1, NULL, @it23, @po2, @qpo3, @rtpo3, @rtpo3),
  (@poln4, 'RM-PKG-BX12',  'Corrugated Box 12-Pack', 2, NULL, @it30, @po2, @qpo4, @rtpo4, @rtpo4);


-- ============================================================================
-- SECTION 48: RECEIVING ORDERS + LINES
-- ============================================================================
-- RCV-001 is closed against PO-001. RCV-002 is open with nothing stocked yet, which
-- is what puts PO-002 on the open receiving and products-on-order lists.

INSERT INTO `receiving_order` (`id`, `number`, `order_id`, `account_id`, `completed_at`) VALUES
  (@rcor1, 'RCV-001', @po1, '@account_id', NOW() - INTERVAL 38 DAY),
  (@rcor2, 'RCV-002', @po2, '@account_id', NULL);

INSERT INTO `receiving_order_line` (`id`, `receiving_order_id`, `quantity_id`, `sales_order_line_id`, `stocked_at`) VALUES
  (@rcorln1, @rcor1, @qpo1, @poln1, NOW() - INTERVAL 38 DAY),
  (@rcorln2, @rcor1, @qpo2, @poln2, NOW() - INTERVAL 38 DAY),
  (@rcorln3, @rcor2, @qpo3, @poln3, NULL),
  (@rcorln4, @rcor2, @qpo4, @poln4, NULL);


-- ============================================================================
-- SECTION 49: DELIVERIES + LINES
-- ============================================================================
-- A delivery is the inbound event against a purchase order. DLV-001 landed complete,
-- DLV-002 is a short first drop against the still-open PO-002.

INSERT INTO `rate` (`id`, `value`, `numerator_unit_id`, `denominator_unit_id`) VALUES
  (@rtdv1, 4.85, @un4, @un8),
  (@rtdv2, 4.40, @un4, @un8),
  (@rtdv3, 6.20, @un4, @un8),
  (@rtdv4, 1.15, @un4, @un1);

INSERT INTO `quantity` (`id`, `value`, `unit_id`) VALUES
  (@qdv1, 800, @un8),
  (@qdv2, 600, @un8),
  (@qdv3, 150, @un8),
  (@qdv4, 400, @un1);

INSERT INTO `delivery` (`id`, `number`, `sales_order_id`, `account_id`, `delivery_status_code`, `accepted_at`) VALUES
  (@dv1, 'DLV-001', @po1, '@account_id', 'accepted', NOW() - INTERVAL 38 DAY),
  (@dv2, 'DLV-002', @po2, '@account_id', 'accepted', NOW() - INTERVAL 3 DAY);

INSERT INTO `delivery_line` (`id`, `delivery_id`, `receiving_order_line_id`, `quantity_id`, `unit_cost_id`, `storage_location_id`, `accepted_at`) VALUES
  (@dvln1, @dv1, @rcorln1, @qdv1, @rtdv1, @stloc8, NOW() - INTERVAL 38 DAY),
  (@dvln2, @dv1, @rcorln2, @qdv2, @rtdv2, @stloc8, NOW() - INTERVAL 38 DAY),
  (@dvln3, @dv2, @rcorln3, @qdv3, @rtdv3, @stloc8, NOW() - INTERVAL 3 DAY),
  (@dvln4, @dv2, @rcorln4, @qdv4, @rtdv4, @stloc7, NOW() - INTERVAL 3 DAY);


-- ============================================================================
-- SECTION 50: INVENTORY RECEIPTS
-- ============================================================================
-- What the deliveries put on the floor. Receipts are the costed layers inventory is
-- consumed from, so a sandbox without them shows stock that has no cost behind it.

INSERT INTO `inventory_receipt` (`id`, `owner_account_id`, `holder_account_id`, `item_id`, `storage_location_id`, `received_at`, `quantity_id`, `unit_cost_id`, `order_id`, `status_code`) VALUES
  (@inrp1, '@account_id', '@account_id', @it21, @stloc8, NOW() - INTERVAL 38 DAY, @qdv1, @rtdv1, @po1, 'available'),
  (@inrp2, '@account_id', '@account_id', @it22, @stloc8, NOW() - INTERVAL 38 DAY, @qdv2, @rtdv2, @po1, 'available'),
  (@inrp3, '@account_id', '@account_id', @it23, @stloc8, NOW() - INTERVAL 3 DAY,  @qdv3, @rtdv3, @po2, 'available'),
  (@inrp4, '@account_id', '@account_id', @it30, @stloc7, NOW() - INTERVAL 3 DAY,  @qdv4, @rtdv4, @po2, 'available');


-- ============================================================================
-- SECTION 51: PRODUCTION RUNS + BATCHES
-- ============================================================================
-- Three completed runs, one per sock size, each walking the whole flow: knit, link,
-- wash, pack. This is the history the production schedule is built from. The solver
-- measures a run rate by joining batch to machine, pools a knit item's demand through
-- the batch genealogy to the finished good it becomes, and reads both from scans
-- inside the demand window, so a sandbox with no batches generates an empty plan.

INSERT INTO `production_run` (`id`, `number`, `account_id`, `responsible_user_id`, `started_at`, `completed_at`) VALUES
  (@pnrn1, 'PR-1001', '@account_id', @acus1, NOW() - INTERVAL 100 DAY, NOW() - INTERVAL 96 DAY),
  (@pnrn2, 'PR-1002', '@account_id', @acus1, NOW() - INTERVAL 70 DAY,  NOW() - INTERVAL 66 DAY),
  (@pnrn3, 'PR-1003', '@account_id', @acus1, NOW() - INTERVAL 40 DAY,  NOW() - INTERVAL 36 DAY);

INSERT INTO `quantity` (`id`, `value`, `unit_id`) VALUES
  (@qbt1, 600, @un2),  (@qbt2, 600, @un2),  (@qbt3, 588, @un2),  (@qbt4, 576, @un2),
  (@qbt5, 720, @un2),  (@qbt6, 720, @un2),  (@qbt7, 708, @un2),  (@qbt8, 696, @un2),
  (@qbt9, 480, @un2),  (@qbt10, 480, @un2), (@qbt11, 468, @un2), (@qbt12, 456, @un2);

INSERT INTO `batch` (`id`, `account_id`, `item_id`, `quantity_id`, `scanning_station_id`, `production_step_id`, `production_run_id`, `location_id`, `scanned_at`, `closed_at`) VALUES
  -- PR-1001, small
  (@bt1,  '@account_id', @it33, @qbt1,  @scst1, @pdst1,  @pnrn1, @stloc2, NOW() - INTERVAL 100 DAY, NOW() - INTERVAL 99 DAY),
  (@bt2,  '@account_id', @it36, @qbt2,  @scst2, @pdst4,  @pnrn1, @stloc3, NOW() - INTERVAL 99 DAY,  NOW() - INTERVAL 98 DAY),
  (@bt3,  '@account_id', @it39, @qbt3,  @scst3, @pdst7,  @pnrn1, @stloc4, NOW() - INTERVAL 98 DAY,  NOW() - INTERVAL 97 DAY),
  (@bt4,  '@account_id', @it1,  @qbt4,  @scst6, @pdst16, @pnrn1, @stloc7, NOW() - INTERVAL 96 DAY,  NOW() - INTERVAL 96 DAY),
  -- PR-1002, medium
  (@bt5,  '@account_id', @it34, @qbt5,  @scst1, @pdst2,  @pnrn2, @stloc2, NOW() - INTERVAL 70 DAY,  NOW() - INTERVAL 69 DAY),
  (@bt6,  '@account_id', @it37, @qbt6,  @scst2, @pdst5,  @pnrn2, @stloc3, NOW() - INTERVAL 69 DAY,  NOW() - INTERVAL 68 DAY),
  (@bt7,  '@account_id', @it40, @qbt7,  @scst3, @pdst8,  @pnrn2, @stloc4, NOW() - INTERVAL 68 DAY,  NOW() - INTERVAL 67 DAY),
  (@bt8,  '@account_id', @it4,  @qbt8,  @scst6, @pdst17, @pnrn2, @stloc7, NOW() - INTERVAL 66 DAY,  NOW() - INTERVAL 66 DAY),
  -- PR-1003, large
  (@bt9,  '@account_id', @it35, @qbt9,  @scst1, @pdst3,  @pnrn3, @stloc2, NOW() - INTERVAL 40 DAY,  NOW() - INTERVAL 39 DAY),
  (@bt10, '@account_id', @it38, @qbt10, @scst2, @pdst6,  @pnrn3, @stloc3, NOW() - INTERVAL 39 DAY,  NOW() - INTERVAL 38 DAY),
  (@bt11, '@account_id', @it41, @qbt11, @scst3, @pdst9,  @pnrn3, @stloc4, NOW() - INTERVAL 38 DAY,  NOW() - INTERVAL 37 DAY),
  (@bt12, '@account_id', @it7,  @qbt12, @scst6, @pdst18, @pnrn3, @stloc7, NOW() - INTERVAL 36 DAY,  NOW() - INTERVAL 36 DAY);

-- A = batch, B = machine. The constraint stage reads its run rate off this join and
-- plans machine by machine, so a knit scan with no machine is invisible to the solver.
INSERT INTO `_batches_machines` (`A`, `B`) VALUES
  (@bt1, @mach1),  (@bt2, @mach3),  (@bt3, @mach5),  (@bt4, @mach11),
  (@bt5, @mach2),  (@bt6, @mach4),  (@bt7, @mach6),  (@bt8, @mach11),
  (@bt9, @mach13), (@bt10, @mach3), (@bt11, @mach5), (@bt12, @mach12);

-- A = downstream, B = upstream, matching the Prisma orientation of _batch_flow. The
-- knit item carries no order demand of its own, so the plan pools it from the finished
-- good at the end of this chain.
INSERT INTO `_batch_flow` (`A`, `B`) VALUES
  (@bt2, @bt1),   (@bt3, @bt2),   (@bt4, @bt3),
  (@bt6, @bt5),   (@bt7, @bt6),   (@bt8, @bt7),
  (@bt10, @bt9),  (@bt11, @bt10), (@bt12, @bt11);


-- ============================================================================
-- SECTION 52: HISTORICAL DEMAND
-- ============================================================================
-- Fulfilled orders spread over completed months, for the finished goods the seeded runs
-- produce. Trailing-twelve demand deliberately ignores the current partial month, so the
-- orders in SECTION 39 -- all inside the last two weeks -- contribute nothing to a plan.
-- Without this history every item plans to zero. The three anchor orders here are kept for
-- their per-line costs. SECTION 52b layers a full year of monthly volume on top so the
-- plan is not a single frozen campaign. See that section for the sizing rationale.

INSERT INTO `rate` (`id`, `value`, `numerator_unit_id`, `denominator_unit_id`) VALUES
  (@rthso1, 11, @un4, @un2), (@rthso2, 11, @un4, @un2), (@rthso3, 12, @un4, @un2),
  (@rthso4, 12, @un4, @un2), (@rthso5, 11, @un4, @un2), (@rthso6, 12, @un4, @un2);

INSERT INTO `rate` (`id`, `value`, `numerator_unit_id`, `denominator_unit_id`) VALUES
  (@rthsc1, 6.10, @un4, @un2), (@rthsc2, 6.30, @un4, @un2), (@rthsc3, 6.55, @un4, @un2),
  (@rthsc4, 6.55, @un4, @un2), (@rthsc5, 6.10, @un4, @un2), (@rthsc6, 6.55, @un4, @un2);

INSERT INTO `quantity` (`id`, `value`, `unit_id`) VALUES
  (@qhso1, 480, @un2), (@qhso2, 360, @un2), (@qhso3, 600, @un2),
  (@qhso4, 420, @un2), (@qhso5, 540, @un2), (@qhso6, 480, @un2);

INSERT INTO `sales_order` (`id`, `number`, `sales_order_status_code`, `sales_order_type_code`, `priority_code`, `buyer_account_id`, `seller_account_id`, `owner_account_id`, `billing_address_id`, `shipping_address_id`, `carrier_id`, `carrier_option_id`, `payment_term_id`, `shipping_term_id`, `sales_rep_id`, `issued_at`, `completed_at`) VALUES
  (@hso1, 'ORD-H001', 'fulfilled', 'sales_order', 'normal', @cust1, '@account_id', '@account_id', @caddr1, @caddr2, 'delivery', NULL, @pytm1, @shtm1, @acus1, NOW() - INTERVAL 7 MONTH, NOW() - INTERVAL 7 MONTH + INTERVAL 12 DAY),
  (@hso2, 'ORD-H002', 'fulfilled', 'sales_order', 'normal', @cust2, '@account_id', '@account_id', @caddr3, @caddr4, 'delivery', NULL, @pytm1, @shtm1, @acus1, NOW() - INTERVAL 4 MONTH, NOW() - INTERVAL 4 MONTH + INTERVAL 9 DAY),
  (@hso3, 'ORD-H003', 'fulfilled', 'sales_order', 'normal', @cust3, '@account_id', '@account_id', @caddr5, @caddr6, 'delivery', NULL, @pytm2, @shtm1, @acus1, NOW() - INTERVAL 2 MONTH, NOW() - INTERVAL 2 MONTH + INTERVAL 11 DAY);

INSERT INTO `sales_order_line` (`id`, `product_sku`, `product_description`, `line_item_number`, `product_id`, `item_id`, `sales_order_id`, `quantity_id`, `unit_price_id`, `unit_cost_id`) VALUES
  (@hsol1, 'FG-CRW-S-WHT', 'Crew Sock Small White',  1, @pd1, @it1, @hso1, @qhso1, @rthso1, @rthsc1),
  (@hsol2, 'FG-CRW-M-WHT', 'Crew Sock Medium White', 2, @pd4, @it4, @hso1, @qhso2, @rthso2, @rthsc2),
  (@hsol3, 'FG-CRW-M-WHT', 'Crew Sock Medium White', 1, @pd4, @it4, @hso2, @qhso3, @rthso3, @rthsc3),
  (@hsol4, 'FG-CRW-L-WHT', 'Crew Sock Large White',  2, @pd7, @it7, @hso2, @qhso4, @rthso4, @rthsc4),
  (@hsol5, 'FG-CRW-S-WHT', 'Crew Sock Small White',  1, @pd1, @it1, @hso3, @qhso5, @rthso5, @rthsc5),
  (@hsol6, 'FG-CRW-L-WHT', 'Crew Sock Large White',  2, @pd7, @it7, @hso3, @qhso6, @rthso6, @rthsc6);


-- ----------------------------------------------------------------------------
-- SECTION 52b: EXPANDED MONTHLY DEMAND HISTORY
-- ----------------------------------------------------------------------------
-- One fulfilled order for each of the eleven completed months at NOW() - INTERVAL k MONTH
-- (k = 1..11), on the three finished goods the seeded knit runs produce (Crew White S/M/L,
-- the only products the batch genealogy pools onto WIP-KNT-S/M/L). The three orders above
-- alone leave weekly demand so far below a machine-week that a single campaign covers the
-- whole horizon and every flexible week reads idle. A year of realistic volume
-- (~42k/38k/30k pairs across S/M/L, roughly half of knitting capacity) is what makes the
-- solver plan recurring campaigns across the weeks instead of one frozen batch.
--
-- Eleven months rather than twelve on purpose: the demand window is planning-date minus
-- twelve months, so an order at exactly NOW() - INTERVAL 12 MONTH sits on the window edge
-- and drops out the moment the schedule is generated any later than the seed ran. k stops
-- at 11 so the oldest order (about eleven months back) still clears the window start by
-- roughly a month of slack -- the trailing year holds eleven complete months anyway once
-- the current partial month is set aside. Every month carries demand so the series is
-- dense and classifies as smooth, with quantities rising into the autumn peak. Customers
-- rotate so the history is not one buyer, and every order names the sales rep so it counts
-- toward the seeded targets. Lines carry a unit price but no cost -- the schedule reads
-- quantity only, and the costed history above already feeds the margin views.

SET @dho1 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dho2 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dho3 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dho4 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dho5 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dho6 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dho7 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dho8 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dho9 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dho10 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dho11 = CONCAT('or_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @dhq1 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq2 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq3 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq4 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq5 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq6 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq7 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq8 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq9 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq10 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq11 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq12 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq13 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq14 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq15 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq16 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq17 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq18 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq19 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq20 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq21 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq22 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq23 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq24 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq25 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq26 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq27 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq28 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq29 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq30 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq31 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq32 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhq33 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12));

SET @dhr1 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr2 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr3 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr4 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr5 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr6 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr7 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr8 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr9 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr10 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr11 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr12 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr13 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr14 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr15 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr16 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr17 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr18 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr19 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr20 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr21 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr22 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr23 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr24 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr25 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr26 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr27 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr28 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr29 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr30 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr31 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr32 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));
SET @dhr33 = CONCAT('rt_', LEFT(REPLACE(UUID(), '-', ''), 12));

INSERT INTO `rate` (`id`, `value`, `numerator_unit_id`, `denominator_unit_id`) VALUES
  (@dhr1, 11.50, @un4, @un2), (@dhr2, 11.00, @un4, @un2), (@dhr3, 12.00, @un4, @un2),
  (@dhr4, 11.50, @un4, @un2), (@dhr5, 11.00, @un4, @un2), (@dhr6, 12.00, @un4, @un2),
  (@dhr7, 11.50, @un4, @un2), (@dhr8, 11.00, @un4, @un2), (@dhr9, 12.00, @un4, @un2),
  (@dhr10, 11.50, @un4, @un2), (@dhr11, 11.00, @un4, @un2), (@dhr12, 12.00, @un4, @un2),
  (@dhr13, 11.50, @un4, @un2), (@dhr14, 11.00, @un4, @un2), (@dhr15, 12.00, @un4, @un2),
  (@dhr16, 11.50, @un4, @un2), (@dhr17, 11.00, @un4, @un2), (@dhr18, 12.00, @un4, @un2),
  (@dhr19, 11.50, @un4, @un2), (@dhr20, 11.00, @un4, @un2), (@dhr21, 12.00, @un4, @un2),
  (@dhr22, 11.50, @un4, @un2), (@dhr23, 11.00, @un4, @un2), (@dhr24, 12.00, @un4, @un2),
  (@dhr25, 11.50, @un4, @un2), (@dhr26, 11.00, @un4, @un2), (@dhr27, 12.00, @un4, @un2),
  (@dhr28, 11.50, @un4, @un2), (@dhr29, 11.00, @un4, @un2), (@dhr30, 12.00, @un4, @un2),
  (@dhr31, 11.50, @un4, @un2), (@dhr32, 11.00, @un4, @un2), (@dhr33, 12.00, @un4, @un2);

INSERT INTO `quantity` (`id`, `value`, `unit_id`) VALUES
  (@dhq1, 3800, @un2), (@dhq2, 3400, @un2), (@dhq3, 2600, @un2),
  (@dhq4, 3200, @un2), (@dhq5, 2900, @un2), (@dhq6, 2200, @un2),
  (@dhq7, 2800, @un2), (@dhq8, 2600, @un2), (@dhq9, 2000, @un2),
  (@dhq10, 3000, @un2), (@dhq11, 2800, @un2), (@dhq12, 2100, @un2),
  (@dhq13, 3600, @un2), (@dhq14, 3200, @un2), (@dhq15, 2500, @un2),
  (@dhq16, 4200, @un2), (@dhq17, 3700, @un2), (@dhq18, 2900, @un2),
  (@dhq19, 4800, @un2), (@dhq20, 4200, @un2), (@dhq21, 3300, @un2),
  (@dhq22, 5200, @un2), (@dhq23, 4500, @un2), (@dhq24, 3600, @un2),
  (@dhq25, 4600, @un2), (@dhq26, 4000, @un2), (@dhq27, 3200, @un2),
  (@dhq28, 3800, @un2), (@dhq29, 3400, @un2), (@dhq30, 2700, @un2),
  (@dhq31, 3400, @un2), (@dhq32, 3000, @un2), (@dhq33, 2400, @un2);

INSERT INTO `sales_order` (`id`, `number`, `sales_order_status_code`, `sales_order_type_code`, `priority_code`, `buyer_account_id`, `seller_account_id`, `owner_account_id`, `billing_address_id`, `shipping_address_id`, `carrier_id`, `carrier_option_id`, `payment_term_id`, `shipping_term_id`, `sales_rep_id`, `issued_at`, `completed_at`) VALUES
  (@dho1, 'ORD-D01', 'fulfilled', 'sales_order', 'normal', @cust1, '@account_id', '@account_id', @caddr1, @caddr2, 'delivery', NULL, @pytm1, @shtm1, @acus1, NOW() - INTERVAL 1 MONTH, NOW() - INTERVAL 1 MONTH + INTERVAL 10 DAY),
  (@dho2, 'ORD-D02', 'fulfilled', 'sales_order', 'normal', @cust2, '@account_id', '@account_id', @caddr3, @caddr4, 'delivery', NULL, @pytm1, @shtm1, @acus1, NOW() - INTERVAL 2 MONTH, NOW() - INTERVAL 2 MONTH + INTERVAL 10 DAY),
  (@dho3, 'ORD-D03', 'fulfilled', 'sales_order', 'normal', @cust3, '@account_id', '@account_id', @caddr5, @caddr6, 'delivery', NULL, @pytm2, @shtm1, @acus1, NOW() - INTERVAL 3 MONTH, NOW() - INTERVAL 3 MONTH + INTERVAL 10 DAY),
  (@dho4, 'ORD-D04', 'fulfilled', 'sales_order', 'normal', @cust1, '@account_id', '@account_id', @caddr1, @caddr2, 'delivery', NULL, @pytm1, @shtm1, @acus1, NOW() - INTERVAL 4 MONTH, NOW() - INTERVAL 4 MONTH + INTERVAL 10 DAY),
  (@dho5, 'ORD-D05', 'fulfilled', 'sales_order', 'normal', @cust2, '@account_id', '@account_id', @caddr3, @caddr4, 'delivery', NULL, @pytm1, @shtm1, @acus1, NOW() - INTERVAL 5 MONTH, NOW() - INTERVAL 5 MONTH + INTERVAL 10 DAY),
  (@dho6, 'ORD-D06', 'fulfilled', 'sales_order', 'normal', @cust3, '@account_id', '@account_id', @caddr5, @caddr6, 'delivery', NULL, @pytm2, @shtm1, @acus1, NOW() - INTERVAL 6 MONTH, NOW() - INTERVAL 6 MONTH + INTERVAL 10 DAY),
  (@dho7, 'ORD-D07', 'fulfilled', 'sales_order', 'normal', @cust1, '@account_id', '@account_id', @caddr1, @caddr2, 'delivery', NULL, @pytm1, @shtm1, @acus1, NOW() - INTERVAL 7 MONTH, NOW() - INTERVAL 7 MONTH + INTERVAL 10 DAY),
  (@dho8, 'ORD-D08', 'fulfilled', 'sales_order', 'normal', @cust2, '@account_id', '@account_id', @caddr3, @caddr4, 'delivery', NULL, @pytm1, @shtm1, @acus1, NOW() - INTERVAL 8 MONTH, NOW() - INTERVAL 8 MONTH + INTERVAL 10 DAY),
  (@dho9, 'ORD-D09', 'fulfilled', 'sales_order', 'normal', @cust3, '@account_id', '@account_id', @caddr5, @caddr6, 'delivery', NULL, @pytm2, @shtm1, @acus1, NOW() - INTERVAL 9 MONTH, NOW() - INTERVAL 9 MONTH + INTERVAL 10 DAY),
  (@dho10, 'ORD-D10', 'fulfilled', 'sales_order', 'normal', @cust1, '@account_id', '@account_id', @caddr1, @caddr2, 'delivery', NULL, @pytm1, @shtm1, @acus1, NOW() - INTERVAL 10 MONTH, NOW() - INTERVAL 10 MONTH + INTERVAL 10 DAY),
  (@dho11, 'ORD-D11', 'fulfilled', 'sales_order', 'normal', @cust2, '@account_id', '@account_id', @caddr3, @caddr4, 'delivery', NULL, @pytm1, @shtm1, @acus1, NOW() - INTERVAL 11 MONTH, NOW() - INTERVAL 11 MONTH + INTERVAL 10 DAY);

INSERT INTO `sales_order_line` (`id`, `product_sku`, `product_description`, `line_item_number`, `product_id`, `item_id`, `sales_order_id`, `quantity_id`, `unit_price_id`, `unit_cost_id`) VALUES
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-S-WHT', 'Crew Sock Small White', 1, @pd1, @it1, @dho1, @dhq1, @dhr1, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-M-WHT', 'Crew Sock Medium White', 2, @pd4, @it4, @dho1, @dhq2, @dhr2, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-L-WHT', 'Crew Sock Large White', 3, @pd7, @it7, @dho1, @dhq3, @dhr3, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-S-WHT', 'Crew Sock Small White', 1, @pd1, @it1, @dho2, @dhq4, @dhr4, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-M-WHT', 'Crew Sock Medium White', 2, @pd4, @it4, @dho2, @dhq5, @dhr5, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-L-WHT', 'Crew Sock Large White', 3, @pd7, @it7, @dho2, @dhq6, @dhr6, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-S-WHT', 'Crew Sock Small White', 1, @pd1, @it1, @dho3, @dhq7, @dhr7, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-M-WHT', 'Crew Sock Medium White', 2, @pd4, @it4, @dho3, @dhq8, @dhr8, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-L-WHT', 'Crew Sock Large White', 3, @pd7, @it7, @dho3, @dhq9, @dhr9, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-S-WHT', 'Crew Sock Small White', 1, @pd1, @it1, @dho4, @dhq10, @dhr10, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-M-WHT', 'Crew Sock Medium White', 2, @pd4, @it4, @dho4, @dhq11, @dhr11, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-L-WHT', 'Crew Sock Large White', 3, @pd7, @it7, @dho4, @dhq12, @dhr12, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-S-WHT', 'Crew Sock Small White', 1, @pd1, @it1, @dho5, @dhq13, @dhr13, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-M-WHT', 'Crew Sock Medium White', 2, @pd4, @it4, @dho5, @dhq14, @dhr14, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-L-WHT', 'Crew Sock Large White', 3, @pd7, @it7, @dho5, @dhq15, @dhr15, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-S-WHT', 'Crew Sock Small White', 1, @pd1, @it1, @dho6, @dhq16, @dhr16, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-M-WHT', 'Crew Sock Medium White', 2, @pd4, @it4, @dho6, @dhq17, @dhr17, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-L-WHT', 'Crew Sock Large White', 3, @pd7, @it7, @dho6, @dhq18, @dhr18, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-S-WHT', 'Crew Sock Small White', 1, @pd1, @it1, @dho7, @dhq19, @dhr19, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-M-WHT', 'Crew Sock Medium White', 2, @pd4, @it4, @dho7, @dhq20, @dhr20, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-L-WHT', 'Crew Sock Large White', 3, @pd7, @it7, @dho7, @dhq21, @dhr21, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-S-WHT', 'Crew Sock Small White', 1, @pd1, @it1, @dho8, @dhq22, @dhr22, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-M-WHT', 'Crew Sock Medium White', 2, @pd4, @it4, @dho8, @dhq23, @dhr23, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-L-WHT', 'Crew Sock Large White', 3, @pd7, @it7, @dho8, @dhq24, @dhr24, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-S-WHT', 'Crew Sock Small White', 1, @pd1, @it1, @dho9, @dhq25, @dhr25, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-M-WHT', 'Crew Sock Medium White', 2, @pd4, @it4, @dho9, @dhq26, @dhr26, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-L-WHT', 'Crew Sock Large White', 3, @pd7, @it7, @dho9, @dhq27, @dhr27, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-S-WHT', 'Crew Sock Small White', 1, @pd1, @it1, @dho10, @dhq28, @dhr28, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-M-WHT', 'Crew Sock Medium White', 2, @pd4, @it4, @dho10, @dhq29, @dhr29, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-L-WHT', 'Crew Sock Large White', 3, @pd7, @it7, @dho10, @dhq30, @dhr30, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-S-WHT', 'Crew Sock Small White', 1, @pd1, @it1, @dho11, @dhq31, @dhr31, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-M-WHT', 'Crew Sock Medium White', 2, @pd4, @it4, @dho11, @dhq32, @dhr32, NULL),
  (CONCAT('orln_', LEFT(REPLACE(UUID(), '-', ''), 12)), 'FG-CRW-L-WHT', 'Crew Sock Large White', 3, @pd7, @it7, @dho11, @dhq33, @dhr33, NULL);


-- ============================================================================
-- SECTION 53: MACHINE DOWNTIME EVENTS
-- ============================================================================
-- Closed events across the three OEE buckets, dated against the seeded runs so
-- availability, performance and quality all have something to report.

INSERT INTO `machine_downtime_event` (`id`, `account_id`, `machine_id`, `department_id`, `production_step_id`, `reason_code`, `started_at`, `ended_at`, `duration_seconds`, `shift_date`, `shift_code`, `item_id`, `production_run_id`, `note`, `reported_by_id`, `source_code`) VALUES
  (@mcdt1, '@account_id', @mach1,  @dept1, @pdst1, 'breakdown',         NOW() - INTERVAL 100 DAY, NOW() - INTERVAL 100 DAY + INTERVAL 45 MINUTE, 2700, DATE(NOW() - INTERVAL 100 DAY), 'day',   @it33, @pnrn1, 'Needle bar jam on cylinder 3', @acus1, 'manual'),
  (@mcdt2, '@account_id', @mach2,  @dept1, @pdst2, 'changeover',        NOW() - INTERVAL 70 DAY,  NOW() - INTERVAL 70 DAY + INTERVAL 35 MINUTE,  2100, DATE(NOW() - INTERVAL 70 DAY),  'day',   @it34, @pnrn2, 'Yarn color changeover',       @acus1, 'manual'),
  (@mcdt3, '@account_id', @mach13, @dept1, @pdst3, 'material_shortage', NOW() - INTERVAL 40 DAY,  NOW() - INTERVAL 40 DAY + INTERVAL 90 MINUTE,  5400, DATE(NOW() - INTERVAL 40 DAY),  'swing', @it35, @pnrn3, 'Waiting on cotton yarn delivery', @acus1, 'manual'),
  (@mcdt4, '@account_id', @mach5,  @dept3, @pdst7, 'minor_stop',        NOW() - INTERVAL 98 DAY,  NOW() - INTERVAL 98 DAY + INTERVAL 12 MINUTE,   720, DATE(NOW() - INTERVAL 98 DAY),  'day',   @it39, @pnrn1, 'Drum sensor fault, cleared on site', @acus1, 'manual'),
  (@mcdt5, '@account_id', @mach11, @dept6, @pdst17, 'quality_hold',     NOW() - INTERVAL 66 DAY,  NOW() - INTERVAL 66 DAY + INTERVAL 60 MINUTE,  3600, DATE(NOW() - INTERVAL 66 DAY),  'day',   @it4,  @pnrn2, 'Pairing check on boarded stock', @acus1, 'manual');


-- ============================================================================
-- SECTION 54: SALES TARGETS
-- ============================================================================
-- Quarter-to-date and year-to-date revenue targets for the sandbox admin.
--
-- A target only means something against orders attributed to the same rep: the quarterly
-- totals and the sales and order analytics all group on sales_order.sales_rep_id. That is
-- why every seeded sales order names this account_user, and why the customer relations
-- carry it as their default -- an order added later inherits a rep rather than landing
-- unattributed next to the seeded ones. Purchase orders have no sales rep and set none.

INSERT INTO `quantity` (`id`, `value`, `unit_id`) VALUES
  (@qta1, 250000, @un4),
  (@qta2, 900000, @un4);

INSERT INTO `target` (`id`, `start_date`, `end_date`, `sales_rep_id`, `account_id`, `amount_id`) VALUES
  (@ta1, DATE(NOW() - INTERVAL 3 MONTH), DATE(NOW() + INTERVAL 1 MONTH), @acus1, '@account_id', @qta1),
  (@ta2, DATE(NOW() - INTERVAL 9 MONTH), DATE(NOW() + INTERVAL 3 MONTH), @acus1, '@account_id', @qta2);


-- ============================================================================
-- SECTION 55: CUSTOMER PRICES
-- ============================================================================
-- A negotiated price per product line for the two wholesale customers, which is what
-- the pricing views compare a quoted line against.

INSERT INTO `rate` (`id`, `value`, `numerator_unit_id`, `denominator_unit_id`) VALUES
  (@rtacpr1, 9.50,  @un4, @un2),
  (@rtacpr2, 8.75,  @un4, @un2),
  (@rtacpr3, 10.25, @un4, @un2);

INSERT INTO `account_price` (`id`, `owner_account_id`, `recipient_account_id`, `product_line_id`, `unit_value_id`) VALUES
  (@acpr1, '@account_id', @cust1, @pdln1, @rtacpr1),
  (@acpr2, '@account_id', @cust1, @pdln2, @rtacpr2),
  (@acpr3, '@account_id', @cust2, @pdln1, @rtacpr3);


-- ============================================================================
-- SECTION 56: PRODUCTION SCHEDULE SETTINGS
-- ============================================================================
-- Knitting is the constraint: it sets the pace of the plant, and everything downstream
-- is derived from what it can turn out. Naming the department rather than the machines
-- means a machine added to it is planned without anyone ticking a box.
--
-- The per-machine rows below are the department's own machines, written explicitly so
-- the settings page shows what is being planned rather than an empty list. They carry
-- the defaults, so they change nothing on their own -- the absence of a row already
-- means planned -- and are the rows a user edits to take a machine out for a rebuild.
--
-- is_enabled is what turns scheduling on for the account. Seeded on, so a sandbox can
-- generate a schedule without visiting settings first.

INSERT INTO `account_production_schedule_setting` (`id`, `account_id`, `constraint_department_id`, `is_enabled`, `shifts_per_day`, `hours_per_shift`, `work_days_per_week`, `planning_horizon_weeks`, `frozen_weeks`, `default_lot_units`) VALUES
  (@acpnscsd1, '@account_id', @dept1, 1, 2, 8.000, 5, 13, 1, 60.000000);

INSERT INTO `production_schedule_resource_setting` (`id`, `account_id`, `scope_code`, `scope_ref_id`, `is_excluded`, `is_enabled`, `sort_order`) VALUES
  (@pnscrrsd1, '@account_id', 'machine', @mach1,  0, 1, 1),
  (@pnscrrsd2, '@account_id', 'machine', @mach2,  0, 1, 2),
  (@pnscrrsd3, '@account_id', 'machine', @mach13, 0, 1, 3);
