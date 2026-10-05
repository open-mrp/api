# Changelog

## [2.20.0](https://github.com/open-mrp/api/compare/v2.19.1...v2.20.0) (2026-10-05)


### Features

* include department on machine list and allow clearing machine/department notes ([#249](https://github.com/open-mrp/api/issues/249)) ([646bfa4](https://github.com/open-mrp/api/commit/646bfa43dc27ceee60f6581e941f8adfd8f96c19))

## [2.19.1](https://github.com/open-mrp/api/compare/v2.19.0...v2.19.1) (2026-10-05)


### Bug Fixes

* return 400 not 500 when address validation region is unsupported ([#247](https://github.com/open-mrp/api/issues/247)) ([713faff](https://github.com/open-mrp/api/commit/713faff090a87f9e47e7e70ff81baf4404fe9ee8))

## [2.19.0](https://github.com/open-mrp/api/compare/v2.18.5...v2.19.0) (2026-10-05)


### Features

* **release:** split schema deploys over PlanetScale's 10-table limit and wait on in-flight deploys ([#245](https://github.com/open-mrp/api/issues/245)) ([5ec07fb](https://github.com/open-mrp/api/commit/5ec07fbc2e0092660dba6b593e6d5add73d927cd))

## [2.18.5](https://github.com/open-mrp/api/compare/v2.18.4...v2.18.5) (2026-10-05)


### Bug Fixes

* cost labor time in its own units, and weigh labor efficiency in standard hours ([#242](https://github.com/open-mrp/api/issues/242)) ([21f4bbb](https://github.com/open-mrp/api/commit/21f4bbb72016d1d0df4baf634a73f5600410ff0d))

## [2.18.4](https://github.com/open-mrp/api/compare/v2.18.3...v2.18.4) (2026-10-02)


### Performance Improvements

* **sales,analytics:** serve sales lists and analytics from keys, and plan-test them ([#236](https://github.com/open-mrp/api/issues/236)) ([64c9fca](https://github.com/open-mrp/api/commit/64c9fca0e40f51c85dad27e5d53628410fa47483))

## [2.18.3](https://github.com/open-mrp/api/compare/v2.18.2...v2.18.3) (2026-10-02)


### Performance Improvements

* **inventory,production:** serve inventory, production, and log lists from keys, and plan-test them ([#235](https://github.com/open-mrp/api/issues/235)) ([12e0b30](https://github.com/open-mrp/api/commit/12e0b304717b8e559f27fed389900e2872a3d581))

## [2.18.2](https://github.com/open-mrp/api/compare/v2.18.1...v2.18.2) (2026-10-02)


### Bug Fixes

* **migrations:** batch the shipment buyer backfill so Vitess doesn't kill it ([#238](https://github.com/open-mrp/api/issues/238)) ([b546d9f](https://github.com/open-mrp/api/commit/b546d9f4373812053beefa919017e22494040c5e))

## [2.18.1](https://github.com/open-mrp/api/compare/v2.18.0...v2.18.1) (2026-10-02)


### Bug Fixes

* **release:** recut the release's deploy request on every push while a release PR is open ([#237](https://github.com/open-mrp/api/issues/237)) ([fa6c5b3](https://github.com/open-mrp/api/commit/fa6c5b3e17a67800c8ca96bad93195eb2755e160))


### Performance Improvements

* serve every list and analytics query from an index, and plan-test them ([#232](https://github.com/open-mrp/api/issues/232)) ([bd92a36](https://github.com/open-mrp/api/commit/bd92a3617f438521e1b9f9d23cf0a8a9b3d3c1e2))

## [2.18.0](https://github.com/open-mrp/api/compare/v2.17.1...v2.18.0) (2026-10-01)


### Features

* **receiving-orders:** serve the dashboard's receiving screens from the Go API ([#230](https://github.com/open-mrp/api/issues/230)) ([74fec19](https://github.com/open-mrp/api/commit/74fec1938673a20f5f57cedf3569d769960af868))

## [2.17.1](https://github.com/open-mrp/api/compare/v2.17.0...v2.17.1) (2026-10-01)


### Bug Fixes

* **inventory:** ensure all reserved inventory is released on an order being closed short ([#228](https://github.com/open-mrp/api/issues/228)) ([77629ac](https://github.com/open-mrp/api/commit/77629ac1275d84d8eb4fc7950ac759de6c2f6772))

## [2.17.0](https://github.com/open-mrp/api/compare/v2.16.1...v2.17.0) (2026-10-01)


### Features

* **purchase-orders:** serve the dashboard's purchase orders from the Go API ([#223](https://github.com/open-mrp/api/issues/223)) ([37fc931](https://github.com/open-mrp/api/commit/37fc931270f6f7004a3d7f88f3e839d5809805eb))


### Bug Fixes

* **finance:** include allocations applied at the ends_at bound ([#225](https://github.com/open-mrp/api/issues/225)) ([2fa2e7a](https://github.com/open-mrp/api/commit/2fa2e7ade67b54692057f58034cd48d58fdc5fba))


### Performance Improvements

* **sales-facts:** start daily refresher passes at midnight Eastern ([#224](https://github.com/open-mrp/api/issues/224)) ([e772fb2](https://github.com/open-mrp/api/commit/e772fb2152bf3c496cebe91f0cdd4d234d040402))

## [2.16.1](https://github.com/open-mrp/api/compare/v2.16.0...v2.16.1) (2026-10-01)


### Bug Fixes

* **hubspot:** only link existing contacts on order sync ([#221](https://github.com/open-mrp/api/issues/221)) ([453f469](https://github.com/open-mrp/api/commit/453f4692149d5ad8b64a9fe76ee6408285b5002f))

## [2.16.0](https://github.com/open-mrp/api/compare/v2.15.2...v2.16.0) (2026-10-01)


### Features

* **proto:** enhance production run and batch structures with new fields and validation ([#219](https://github.com/open-mrp/api/issues/219)) ([081923d](https://github.com/open-mrp/api/commit/081923d3fd1b2c6262855d4697af52580041e230))

## [2.15.2](https://github.com/open-mrp/api/compare/v2.15.1...v2.15.2) (2026-09-30)


### Bug Fixes

* **refresher:** optimize buyer marking logic in sales fact refresher ([eec427f](https://github.com/open-mrp/api/commit/eec427fa22a4742d0ad42fbaaf8d0b8e4aa5a966))
* **tests:** update Pod B lease acquisition logic in sales_fact_refresher_loop_test to ensure correct retry behavior ([885589f](https://github.com/open-mrp/api/commit/885589f91d69e31a6f9baab794c92fa4671c6f7a))

## [2.15.1](https://github.com/open-mrp/api/compare/v2.15.0...v2.15.1) (2026-09-30)


### Code Refactoring

* **tests:** remove legacy rollup mark tests and related logic ([1c2bbfb](https://github.com/open-mrp/api/commit/1c2bbfbc1bbebe0889e139c38401663900374368))

## [2.15.0](https://github.com/open-mrp/api/compare/v2.14.0...v2.15.0) (2026-09-30)


### Features

* **analytics:** add ListNewCustomers endpoint and related data structures ([#215](https://github.com/open-mrp/api/issues/215)) ([44b8eaf](https://github.com/open-mrp/api/commit/44b8eaf83bd63d2d60e83e8496711e093c94cd1d))

## [2.14.0](https://github.com/open-mrp/api/compare/v2.13.0...v2.14.0) (2026-09-30)


### Features

* **analytics:** serve sales analytics from pre-priced sales_line_fact, on the read replica ([#213](https://github.com/open-mrp/api/issues/213)) ([c8da632](https://github.com/open-mrp/api/commit/c8da632db151d26354fafd95bf03c908c75b3ace))

## [2.13.0](https://github.com/open-mrp/api/compare/v2.12.0...v2.13.0) (2026-09-29)


### Features

* **analytics:** serve sales analytics from pre-priced sales_line_fact, on the read replica ([68ba4d0](https://github.com/open-mrp/api/commit/68ba4d0143bda438f6da0370318e900b3f0bed13))

## [2.12.0](https://github.com/open-mrp/api/compare/v2.11.1...v2.12.0) (2026-09-29)


### Features

* **observability:** export outbox, inbox and DB pool metrics over OTLP ([#210](https://github.com/open-mrp/api/issues/210)) ([0cd2ea0](https://github.com/open-mrp/api/commit/0cd2ea06665d6049c34b582199a19398aebeba12))

## [2.11.1](https://github.com/open-mrp/api/compare/v2.11.0...v2.11.1) (2026-09-28)


### Code Refactoring

* **db:** enhance clarity in SQL migration comments for decimal residue updates ([bd90a66](https://github.com/open-mrp/api/commit/bd90a669fadaa451dbcfcf855c646fe3271ba734))

## [2.11.0](https://github.com/open-mrp/api/compare/v2.10.4...v2.11.0) (2026-09-28)


### Features

* **resolver:** add test for Loader seeing earlier sub-populate ([#206](https://github.com/open-mrp/api/issues/206)) ([c72020c](https://github.com/open-mrp/api/commit/c72020c4346920a34005897079056b824287145a))


### Bug Fixes

* **api:** update endpoint descriptions and input schemas for clarity ([612450d](https://github.com/open-mrp/api/commit/612450d94ca4960eff27abbfdf08e04b9c192160))
* **inventory:** scan decimal residue ([#208](https://github.com/open-mrp/api/issues/208)) ([6c1e348](https://github.com/open-mrp/api/commit/6c1e3480a47ebf1d1c85b1f419e5dde0a9a23654))


### Performance Improvements

* **gateway:** resolve include loaders per level in parallel ([#202](https://github.com/open-mrp/api/issues/202)) ([c5a7631](https://github.com/open-mrp/api/commit/c5a76319674ccb454307a6239c79d05b63e61bfb))
* parallelize tenancy & pick reads, drop redundant products refetch ([#203](https://github.com/open-mrp/api/issues/203)) ([565d7bc](https://github.com/open-mrp/api/commit/565d7bc2136c6d07102eb4d8477a8448c15e5ff2))
* **sales-orders:** batch order line inserts on create ([#201](https://github.com/open-mrp/api/issues/201)) ([2ac9c8f](https://github.com/open-mrp/api/commit/2ac9c8fda743208ef74cb805694732b73523060e))
* **sales-orders:** send order acknowledgement email off the issue request path ([#200](https://github.com/open-mrp/api/issues/200)) ([3eef351](https://github.com/open-mrp/api/commit/3eef35179a771efdccb1c4c0652e4b23d8bcabc7))
* **sales-orders:** speed up the list endpoint ([#205](https://github.com/open-mrp/api/issues/205)) ([101a79a](https://github.com/open-mrp/api/commit/101a79ae90f1d7ea3e2ea586d2c95e3dd2b835e4))

## [2.10.4](https://github.com/open-mrp/api/compare/v2.10.3...v2.10.4) (2026-09-28)


### Bug Fixes

* **outbox:** don't fail on duplicate message, these messages should be acked ([#198](https://github.com/open-mrp/api/issues/198)) ([4415acd](https://github.com/open-mrp/api/commit/4415acd4bd75872087c1a9416b72558670dd6d9b))

## [2.10.3](https://github.com/open-mrp/api/compare/v2.10.2...v2.10.3) (2026-09-25)


### Performance Improvements

* **picks:** page the pick list by id, then hydrate the page ([#194](https://github.com/open-mrp/api/issues/194)) ([cc3ba05](https://github.com/open-mrp/api/commit/cc3ba050c78f8acb1aa6c63295f0847ff0264df9))

## [2.10.2](https://github.com/open-mrp/api/compare/v2.10.1...v2.10.2) (2026-09-25)


### Performance Improvements

* cut API latency toward sub-100ms p99 ([#193](https://github.com/open-mrp/api/issues/193)) ([6335c3a](https://github.com/open-mrp/api/commit/6335c3ad8da1164e398d0f1eef9043924ecbf880))

## [2.10.1](https://github.com/open-mrp/api/compare/v2.10.0...v2.10.1) (2026-09-24)


### Code Refactoring

* **redis:** implement asynchronous monitoring for Redis connection in analytics cache ([#190](https://github.com/open-mrp/api/issues/190)) ([f767802](https://github.com/open-mrp/api/commit/f767802f251a206ccf5c043f35403401412871f8))

## [2.10.0](https://github.com/open-mrp/api/compare/v2.9.2...v2.10.0) (2026-09-24)


### Features

* **redis:** add Redis service and update configurations ([#188](https://github.com/open-mrp/api/issues/188)) ([65bd748](https://github.com/open-mrp/api/commit/65bd748da4ee1e7d05969f8fa98bfda9b1f18c1e))

## [2.9.2](https://github.com/open-mrp/api/compare/v2.9.1...v2.9.2) (2026-09-24)


### Code Refactoring

* **queries:** optimize SQL queries for performance ([#186](https://github.com/open-mrp/api/issues/186)) ([f9e2112](https://github.com/open-mrp/api/commit/f9e211215ce1ab9ef3831a346f56261765a6f2b8))

## [2.9.1](https://github.com/open-mrp/api/compare/v2.9.0...v2.9.1) (2026-09-24)


### Code Refactoring

* **outbox:** enhance enqueuer configuration and commit notification ([#184](https://github.com/open-mrp/api/issues/184)) ([c906e35](https://github.com/open-mrp/api/commit/c906e35a8fb2d994b91e447b30dcecfc96ae6f56))

## [2.9.0](https://github.com/open-mrp/api/compare/v2.8.1...v2.9.0) (2026-09-23)


### Features

* **deploy:** add a manual production rollback workflow ([#177](https://github.com/open-mrp/api/issues/177)) ([56b4c0d](https://github.com/open-mrp/api/commit/56b4c0d30466177e50977968805caec312fc9fc9))


### Bug Fixes

* **auth:** keep password-reset and magic-login links on the first-party domain ([#181](https://github.com/open-mrp/api/issues/181)) ([3644378](https://github.com/open-mrp/api/commit/36443783244dc7481a869111a98f5b95d8e05c99))
* **checkout:** compute customer checkout charge server-side ([#180](https://github.com/open-mrp/api/issues/180)) ([a61fa22](https://github.com/open-mrp/api/commit/a61fa22933a839660c6a6280c88226696857335a))
* **checkout:** derive customer checkout amount and order from stored data ([#182](https://github.com/open-mrp/api/issues/182)) ([a272ef8](https://github.com/open-mrp/api/commit/a272ef87561c8a0352c4e2ab976fdf4f31bed664))
* **sales-orders:** enforce discount reuse check on customer self-create ([#179](https://github.com/open-mrp/api/issues/179)) ([690210a](https://github.com/open-mrp/api/commit/690210a9e42324d2938ded869ceb5b2ad39ad4e1))

## [2.8.1](https://github.com/open-mrp/api/compare/v2.8.0...v2.8.1) (2026-09-23)


### Bug Fixes

* **release:** check out release jobs from qualified tag ref ([#175](https://github.com/open-mrp/api/issues/175)) ([ef73e4a](https://github.com/open-mrp/api/commit/ef73e4a65495cbb3852cd02a4d18463e41a571a1))

## [2.8.0](https://github.com/open-mrp/api/compare/v2.7.4...v2.8.0) (2026-09-23)


### Features

* **identity:** let admins set passwords for any email-less user ([#172](https://github.com/open-mrp/api/issues/172)) ([8adea7d](https://github.com/open-mrp/api/commit/8adea7db9b9fd863b2030be73b29c367ea7314bb))
* **production-runs:** alert the responsible user when someone else adds or deletes batches ([#173](https://github.com/open-mrp/api/issues/173)) ([b1ab60c](https://github.com/open-mrp/api/commit/b1ab60c1c3ea56b6b6321190ef503affe5f4f6a5))

## [2.7.4](https://github.com/open-mrp/api/compare/v2.7.3...v2.7.4) (2026-09-22)


### Bug Fixes

* **schedule:** price run rate per scan unit, not per base unit ([#169](https://github.com/open-mrp/api/issues/169)) ([33a4d4d](https://github.com/open-mrp/api/commit/33a4d4d1d26a8474fa77491fdd0a00a1734da2a0))

## [2.7.3](https://github.com/open-mrp/api/compare/v2.7.2...v2.7.3) (2026-09-21)


### Bug Fixes

* **analytics:** force (account_id, scanned_at) index on OEE batch aggregates ([#167](https://github.com/open-mrp/api/issues/167)) ([3939572](https://github.com/open-mrp/api/commit/3939572a8d62b80e26330a66a60660976f92bc0b))

## [2.7.2](https://github.com/open-mrp/api/compare/v2.7.1...v2.7.2) (2026-09-21)


### Bug Fixes

* **analytics:** resolve issues where performance figures were not properly calculated ([#164](https://github.com/open-mrp/api/issues/164)) ([0780047](https://github.com/open-mrp/api/commit/0780047d79a2389150fbcc444893749d991d48f5))

## [2.7.1](https://github.com/open-mrp/api/compare/v2.7.0...v2.7.1) (2026-09-18)


### Bug Fixes

* **packlist:** enhance pack list generation with account address and update minikube setup instructions ([#161](https://github.com/open-mrp/api/issues/161)) ([f33d8c8](https://github.com/open-mrp/api/commit/f33d8c844b77eb004c3bc5abfc134ff7a1dcafb4))

## [2.7.0](https://github.com/open-mrp/api/compare/v2.6.8...v2.7.0) (2026-09-17)


### Features

* **analytics:** base OEE on shift capacity minus downtime, not scan spans ([#158](https://github.com/open-mrp/api/issues/158)) ([f66e794](https://github.com/open-mrp/api/commit/f66e79442d6c30a04309b7a96d6d9cf731b5c736))
* improve e2e performance and apply default starting filters for some endpoints ([#160](https://github.com/open-mrp/api/issues/160)) ([2ce44d2](https://github.com/open-mrp/api/commit/2ce44d2dfb6eac25b526442521c83404ba04fa4a))


### Bug Fixes

* **scheduling:** count on-hand at every production stage, not just linked batches ([#157](https://github.com/open-mrp/api/issues/157)) ([c2d95fd](https://github.com/open-mrp/api/commit/c2d95fd2da41d70a4620aede62491d4ce62fd807))

## [2.6.8](https://github.com/open-mrp/api/compare/v2.6.7...v2.6.8) (2026-09-16)


### Bug Fixes

* **messaging:** drain consumers on shutdown and requeue lease-held deliveries ([#155](https://github.com/open-mrp/api/issues/155)) ([0bbe97f](https://github.com/open-mrp/api/commit/0bbe97f31ba2916feafe4d01071125490d687c7e))
* **messaging:** record why messages dead-letter and stop feeding unconsumed queues ([#156](https://github.com/open-mrp/api/issues/156)) ([64af311](https://github.com/open-mrp/api/commit/64af311bdca60ce72294e04144858a3ae2f1abf4))
* **picks:** serve the created-at pick list from an index so it stops timing out ([#153](https://github.com/open-mrp/api/issues/153)) ([943fd83](https://github.com/open-mrp/api/commit/943fd837362d9774ed2fe2a4fc0ed86e6bc733cd))

## [2.6.7](https://github.com/open-mrp/api/compare/v2.6.6...v2.6.7) (2026-09-16)


### Bug Fixes

* **invoice:** update SKU handling in invoice queries and related services ([#151](https://github.com/open-mrp/api/issues/151)) ([34f453e](https://github.com/open-mrp/api/commit/34f453e4d1ecb9937fddc4237483a3fc4e075dcb))

## [2.6.6](https://github.com/open-mrp/api/compare/v2.6.5...v2.6.6) (2026-09-16)


### Bug Fixes

* **pricing:** price mixed-unit lines like the dashboard and match its invoice PDF ([#149](https://github.com/open-mrp/api/issues/149)) ([e236493](https://github.com/open-mrp/api/commit/e236493e9f78d40cc7de272534032438e40637a4))

## [2.6.5](https://github.com/open-mrp/api/compare/v2.6.4...v2.6.5) (2026-09-16)


### Bug Fixes

* **migrations:** renumber message_inbox lease migration and guard migration versions ([#148](https://github.com/open-mrp/api/issues/148)) ([b3e780a](https://github.com/open-mrp/api/commit/b3e780a31ec85fe5a9ca5c91f196b7befaf4e049))
* **picks:** index the open-pick ship-by list so it stops scanning closed history ([#146](https://github.com/open-mrp/api/issues/146)) ([627970a](https://github.com/open-mrp/api/commit/627970a6f7081ecd978c6cf196a5ca6ac8533e4c))

## [2.6.4](https://github.com/open-mrp/api/compare/v2.6.3...v2.6.4) (2026-09-04)


### Bug Fixes

* **inventory:** prepare remaining endpoints that touch inventory to move to go ([#141](https://github.com/open-mrp/api/issues/141)) ([e36d1ca](https://github.com/open-mrp/api/commit/e36d1ca5dc1bb54e99cb45290126c2e8da2f35a1))

## [2.6.3](https://github.com/open-mrp/api/compare/v2.6.2...v2.6.3) (2026-09-04)


### Bug Fixes

* **burn-rate:** stop recalc sweep runaway and compute product rates ([#139](https://github.com/open-mrp/api/issues/139)) ([5438fec](https://github.com/open-mrp/api/commit/5438fec72016170534824a4c0db9eb74a2e687b6))

## [2.6.2](https://github.com/open-mrp/api/compare/v2.6.1...v2.6.2) (2026-09-03)


### Bug Fixes

* **picks:** index ship-by list sort; feat(notifications): change count ([#137](https://github.com/open-mrp/api/issues/137)) ([f488ab2](https://github.com/open-mrp/api/commit/f488ab2ff8b4571029231594698538251efcdd96))

## [2.6.1](https://github.com/open-mrp/api/compare/v2.6.0...v2.6.1) (2026-09-03)


### Bug Fixes

* **analytics:** measure OEE performance against scheduled machines' actual run time ([#136](https://github.com/open-mrp/api/issues/136)) ([619d65b](https://github.com/open-mrp/api/commit/619d65b872f1072fa7b490ce40b074c259e73ff8))
* **release:** make PlanetScale prepare step idempotent to an already-deployed release ([#134](https://github.com/open-mrp/api/issues/134)) ([2b2d809](https://github.com/open-mrp/api/commit/2b2d8092cfcdf81ece3ddd086e92289473e9d399))

## [2.6.0](https://github.com/open-mrp/api/compare/v2.5.8...v2.6.0) (2026-09-03)


### Features

* **notifications:** coalesce order-activity alerts per order per day ([#132](https://github.com/open-mrp/api/issues/132)) ([8577317](https://github.com/open-mrp/api/commit/8577317b0379341e5f71d27dcad08600953dee47))


### Bug Fixes

* **burn-rate:** keep item burn rate fresh via write path + periodic sweep ([#130](https://github.com/open-mrp/api/issues/130)) ([1fe0526](https://github.com/open-mrp/api/commit/1fe0526299868e6c5353c47c1ed38efc479cb6ae))


### Performance Improvements

* **picks:** batch line and shipment loads in ListPicks ([#133](https://github.com/open-mrp/api/issues/133)) ([0ef7f79](https://github.com/open-mrp/api/commit/0ef7f79959f12fe418d396c59da8541f8a35dbe1))

## [2.5.8](https://github.com/open-mrp/api/compare/v2.5.7...v2.5.8) (2026-09-02)


### Bug Fixes

* **analytics:** scope OEE performance to scheduled machines and fix baseline selection ([#128](https://github.com/open-mrp/api/issues/128)) ([b7e1909](https://github.com/open-mrp/api/commit/b7e1909561cdec8698a105de081138854cab0b75))

## [2.5.7](https://github.com/open-mrp/api/compare/v2.5.6...v2.5.7) (2026-09-02)


### Bug Fixes

* **analytics:** bucket schedule weeks on the account's week start day ([#125](https://github.com/open-mrp/api/issues/125)) ([043f37f](https://github.com/open-mrp/api/commit/043f37feadcf138ffa2c684a21b1b496c1654eb0))
* **analytics:** derive OEE availability from the published schedule ([#126](https://github.com/open-mrp/api/issues/126)) ([d94653d](https://github.com/open-mrp/api/commit/d94653d4cc13e0aa7ba1fd3f14f25c9e594eeae9))

## [2.5.6](https://github.com/open-mrp/api/compare/v2.5.5...v2.5.6) (2026-09-02)


### Bug Fixes

* pick search match set ([#123](https://github.com/open-mrp/api/issues/123)) ([b242894](https://github.com/open-mrp/api/commit/b2428949dcea33d1409c2b35c47e6181962aa6bd))

## [2.5.5](https://github.com/open-mrp/api/compare/v2.5.4...v2.5.5) (2026-09-02)


### Bug Fixes

* **items:** meet labour time and labour rates in base time units ([#121](https://github.com/open-mrp/api/issues/121)) ([5b19829](https://github.com/open-mrp/api/commit/5b1982970ce8e7c90b969f0f095f01212ea01132))

## [2.5.4](https://github.com/open-mrp/api/compare/v2.5.3...v2.5.4) (2026-09-02)


### Bug Fixes

* item unit cost denominator to match stocking unit ([#119](https://github.com/open-mrp/api/issues/119)) ([e7f2266](https://github.com/open-mrp/api/commit/e7f2266b3c50042bbabbddafa8f77b593e1e65bb))

## [2.5.3](https://github.com/open-mrp/api/compare/v2.5.2...v2.5.3) (2026-09-01)


### Bug Fixes

* **picks:** drive pick search from the match set ([#117](https://github.com/open-mrp/api/issues/117)) ([97a8da3](https://github.com/open-mrp/api/commit/97a8da3de10cee6fc2144e7973038f6d748a6a67))

## [2.5.2](https://github.com/open-mrp/api/compare/v2.5.1...v2.5.2) (2026-09-01)


### Bug Fixes

* **inventory:** drop FOR UPDATE OF, which vtgate rejects ([#115](https://github.com/open-mrp/api/issues/115)) ([ae5a5f6](https://github.com/open-mrp/api/commit/ae5a5f6f3ada347131c5e8c623cb6129715619f7))

## [2.5.1](https://github.com/open-mrp/api/compare/v2.5.0...v2.5.1) (2026-09-01)


### Bug Fixes

* **inventory:** harden inventory allocation process ([70800e1](https://github.com/open-mrp/api/commit/70800e18c4547db9accfd039b153e0fdf30f5a7a))

## [2.5.0](https://github.com/open-mrp/api/compare/v2.4.1...v2.5.0) (2026-09-01)


### Features

* **email:** add new email sending capabilities for invoices, sales orders, and purchase orders; implement document email consumer ([#111](https://github.com/open-mrp/api/issues/111)) ([7ec62fe](https://github.com/open-mrp/api/commit/7ec62fe5cfac49db0b6c8d88568ffb108444d07d))

## [2.4.1](https://github.com/open-mrp/api/compare/v2.4.0...v2.4.1) (2026-09-01)


### Bug Fixes

* **sales-order:** ensure suffixes work with checkout links ([#109](https://github.com/open-mrp/api/issues/109)) ([81cef18](https://github.com/open-mrp/api/commit/81cef186a1916165b5304a38d6bc3572acc115b6))

## [2.4.0](https://github.com/open-mrp/api/compare/v2.3.3...v2.4.0) (2026-08-31)


### Features

* **domain:** add endpoints to set custom merchant email addresses for system emails ([#105](https://github.com/open-mrp/api/issues/105)) ([fd7bd18](https://github.com/open-mrp/api/commit/fd7bd18cb8bf3d59cd63a6004487b72b737d9b15))

## [2.3.3](https://github.com/open-mrp/api/compare/v2.3.2...v2.3.3) (2026-08-31)


### Bug Fixes

* **api-gateway:** hydrate carrier/shipment/order/department/scanning-station/recipient includes via full loaders ([#106](https://github.com/open-mrp/api/issues/106)) ([b7ded11](https://github.com/open-mrp/api/commit/b7ded111e8436b57376ec2bdeff23ea013f3bf23))

## [2.3.2](https://github.com/open-mrp/api/compare/v2.3.1...v2.3.2) (2026-08-31)


### Bug Fixes

* **inventory-change-logs:** hydrate item/responsible_user includes via full loaders ([#103](https://github.com/open-mrp/api/issues/103)) ([61009a4](https://github.com/open-mrp/api/commit/61009a452eef222c7a1ac8d202a5e05364e704de))

## [2.3.1](https://github.com/open-mrp/api/compare/v2.3.0...v2.3.1) (2026-08-31)


### Bug Fixes

* **inventory:** guard the repair command and restamp the Oct-2025 unit mislabel ([#101](https://github.com/open-mrp/api/issues/101)) ([3e99b96](https://github.com/open-mrp/api/commit/3e99b962d2b2cd626620483fa8b4437a4f240f99))

## [2.3.0](https://github.com/open-mrp/api/compare/v2.2.0...v2.3.0) (2026-08-31)


### Features

* **alerts:** add user, account names, and stack trace to 5xx error emails ([#99](https://github.com/open-mrp/api/issues/99)) ([5fe19e3](https://github.com/open-mrp/api/commit/5fe19e34174182e213a16e16579a66dde951dce4))

## [2.2.0](https://github.com/open-mrp/api/compare/v2.1.0...v2.2.0) (2026-08-31)


### Features

* **inventory-change-logs:** add SKU search to list endpoint ([#95](https://github.com/open-mrp/api/issues/95)) ([c88202b](https://github.com/open-mrp/api/commit/c88202bf3560725163d935a0d111fafbd0f13904))
* **seed:** expand sandbox demand history so the schedule plans across weeks ([#96](https://github.com/open-mrp/api/issues/96)) ([7bcdb76](https://github.com/open-mrp/api/commit/7bcdb7633ae7ae41f1cc50dcc81c5692076feb51))


### Bug Fixes

* **inventory:** close receipt double-allocation and correct on-hand reads ([#97](https://github.com/open-mrp/api/issues/97)) ([7044443](https://github.com/open-mrp/api/commit/7044443ade6af4298822b13f689642533d3ab9e4))

## [2.1.0](https://github.com/open-mrp/api/compare/v2.0.7...v2.1.0) (2026-08-31)


### Features

* **migrate:** ensure data version row exists before migrations in production; add more seed data ([#91](https://github.com/open-mrp/api/issues/91)) ([b46ba21](https://github.com/open-mrp/api/commit/b46ba218bf9e7db4e92811195df25aab9f121570))


### Bug Fixes

* **picks:** resolve pick list timeouts on large accounts ([#92](https://github.com/open-mrp/api/issues/92)) ([796cb2f](https://github.com/open-mrp/api/commit/796cb2f9f5b43eb3427cd71cf2ed81044a0c0eb5))

## [2.0.7](https://github.com/open-mrp/api/compare/v2.0.6...v2.0.7) (2026-08-31)


### Bug Fixes

* **product_line:** handle unresolvable unit groups gracefully in product line retrieval ([#89](https://github.com/open-mrp/api/issues/89)) ([98e6814](https://github.com/open-mrp/api/commit/98e6814f0173ee3cb28b422521e6118fe0401053))

## [2.0.6](https://github.com/open-mrp/api/compare/v2.0.5...v2.0.6) (2026-08-28)


### Bug Fixes

* **shared:** add better test coverage and fix a few discovered bugs in shared package ([#86](https://github.com/open-mrp/api/issues/86)) ([9a99f59](https://github.com/open-mrp/api/commit/9a99f59b1eadd1cca5458092cafd10d37a016d24))

## [2.0.5](https://github.com/open-mrp/api/compare/v2.0.4...v2.0.5) (2026-08-28)


### Bug Fixes

* **deploy:** enhance deployment state handling in release script ([4d46e1f](https://github.com/open-mrp/api/commit/4d46e1f5296077235b8a347476e04ab0fce79c53))
* **queries:** enhance query efficiency by adjusting filter application and removing redundant conditions ([#84](https://github.com/open-mrp/api/issues/84)) ([1fe5dc7](https://github.com/open-mrp/api/commit/1fe5dc74f4ac829569edee777fbceaf580ea0ed8))

## [2.0.4](https://github.com/open-mrp/api/compare/v2.0.3...v2.0.4) (2026-08-28)


### Bug Fixes

* **queries:** optimize query performance by refining filter logic and removing unnecessary conditions ([#82](https://github.com/open-mrp/api/issues/82)) ([2af84c6](https://github.com/open-mrp/api/commit/2af84c6ad71108b1ac5f9a684d430bc00163edef))

## [2.0.3](https://github.com/open-mrp/api/compare/v2.0.2...v2.0.3) (2026-08-28)


### Bug Fixes

* **queries:** remove indexes that are no longer in use ([#80](https://github.com/open-mrp/api/issues/80)) ([b9b2909](https://github.com/open-mrp/api/commit/b9b290964eac55e186f31638c0c4297ad11c4075))

## [2.0.2](https://github.com/open-mrp/api/compare/v2.0.1...v2.0.2) (2026-08-28)


### Bug Fixes

* **commitments:** update commitment quoting logic to use order issue date ([#78](https://github.com/open-mrp/api/issues/78)) ([68f3727](https://github.com/open-mrp/api/commit/68f37278a7393ef415dbc615d6e9a8a39548959b))

## [2.0.1](https://github.com/open-mrp/api/compare/v2.0.0...v2.0.1) (2026-08-27)


### Bug Fixes

* **inventory-change-logs:** stop list query scanning the whole table ([#76](https://github.com/open-mrp/api/issues/76)) ([83916c1](https://github.com/open-mrp/api/commit/83916c1c77879fd62bd544fe9999b0c6d58d429c))

## [2.0.0](https://github.com/open-mrp/api/compare/v1.4.5...v2.0.0) (2026-08-27)


### ⚠ BREAKING CHANGES

* commitments sent to their own sub object ([#74](https://github.com/open-mrp/api/issues/74))

### Features

* commitments sent to their own sub object ([#74](https://github.com/open-mrp/api/issues/74)) ([8adbe4d](https://github.com/open-mrp/api/commit/8adbe4d560264239f812e706e84bb6c3c6fbb433))

## [1.4.5](https://github.com/open-mrp/api/compare/v1.4.4...v1.4.5) (2026-08-27)


### Bug Fixes

* **branding:** serve portal logo/favicon from a stable CDN URL ([#70](https://github.com/open-mrp/api/issues/70)) ([a321414](https://github.com/open-mrp/api/commit/a321414c002aa3c3fef3a738ae1aecbed8933bda))

## [1.4.4](https://github.com/open-mrp/api/compare/v1.4.3...v1.4.4) (2026-08-27)


### Bug Fixes

* **core:** add validators, ensure key lookups use primary keys for units ([#71](https://github.com/open-mrp/api/issues/71)) ([876a076](https://github.com/open-mrp/api/commit/876a076b8f08d58050d091de8565e2bece9fd263))

## [1.4.3](https://github.com/open-mrp/api/compare/v1.4.2...v1.4.3) (2026-08-27)


### Bug Fixes

* **core:** improve query performance on frequently ordered products; add fields to picking ([#68](https://github.com/open-mrp/api/issues/68)) ([c52a375](https://github.com/open-mrp/api/commit/c52a375b44ccc82c66ac9d16900f5476a8631b5f))

## [1.4.2](https://github.com/open-mrp/api/compare/v1.4.1...v1.4.2) (2026-08-27)


### Bug Fixes

* **s3:** enable response checksum validation for presigned URLs to prevent 403 errors ([#66](https://github.com/open-mrp/api/issues/66)) ([f41e6d7](https://github.com/open-mrp/api/commit/f41e6d77c24d219b7dc7aa50655d941fdf8b9c58))

## [1.4.1](https://github.com/open-mrp/api/compare/v1.4.0...v1.4.1) (2026-08-26)


### Bug Fixes

* **core:** add replay CLI for stranded inventory_received messages ([#56](https://github.com/open-mrp/api/issues/56)) ([c41afd8](https://github.com/open-mrp/api/commit/c41afd82f643a65bc5396b63bb88cfe30416327a))
* **schedule:** fix issue where manually added lines failed to resolve the SKU properly ([#65](https://github.com/open-mrp/api/issues/65)) ([130b3ac](https://github.com/open-mrp/api/commit/130b3acadaa4b1e3c3975651b9e0b701ae73eb6a))

## [1.4.0](https://github.com/open-mrp/api/compare/v1.3.1...v1.4.0) (2026-08-26)


### Features

* **registration:** include registering user name and email in new-registration alert ([#57](https://github.com/open-mrp/api/issues/57)) ([fb495e0](https://github.com/open-mrp/api/commit/fb495e0858a23ff39b54cfd4d9604e55ae0579d6))
* **scheduling:** make customers make-to-order ([#62](https://github.com/open-mrp/api/issues/62)) ([1755bdb](https://github.com/open-mrp/api/commit/1755bdbf6a9b1a435f9b05779f0b0092ade7635f))


### Bug Fixes

* **scheduling:** price a hand-added campaign off the machine's production step ([#60](https://github.com/open-mrp/api/issues/60)) ([8f9c8dd](https://github.com/open-mrp/api/commit/8f9c8ddead86a51933e3ee6f50cd528efd890a90))


### Performance Improvements

* **db:** drop 10 redundant left-prefix indexes ([#58](https://github.com/open-mrp/api/issues/58)) ([c0ad980](https://github.com/open-mrp/api/commit/c0ad980bdcd677e278351d0536074d7f118d6138))


### Code Refactoring

* **tests:** improve machine status and burn rate tests for reliability ([#61](https://github.com/open-mrp/api/issues/61)) ([c91b36f](https://github.com/open-mrp/api/commit/c91b36f933be4b2a3a5edfa4bd2e3cf4a439059b))

## [1.3.1](https://github.com/open-mrp/api/compare/v1.3.0...v1.3.1) (2026-08-26)


### Bug Fixes

* **core:** defer inventory-received open-issue allocation to the paged consumer ([#54](https://github.com/open-mrp/api/issues/54)) ([f6c6cf2](https://github.com/open-mrp/api/commit/f6c6cf230ae8181ca0606d2997dea6ce58750ee6))

## [1.3.0](https://github.com/open-mrp/api/compare/v1.2.0...v1.3.0) (2026-08-25)


### Features

* **scheduling:** hold a physical greige buffer at the constraint ([#51](https://github.com/open-mrp/api/issues/51)) ([57b6987](https://github.com/open-mrp/api/commit/57b69879000b13a7a33df156727106a60a117ddf))


### Bug Fixes

* **core:** defer batch-scan open-issue allocation to a paged async consumer ([#50](https://github.com/open-mrp/api/issues/50)) ([c8bf1b1](https://github.com/open-mrp/api/commit/c8bf1b1c78dcd2d22107d3bd9994f27e0678e0b1))
* **release:** skip already-shipped migrations on the prod-cut branch ([#53](https://github.com/open-mrp/api/issues/53)) ([9516f54](https://github.com/open-mrp/api/commit/9516f549db441147bc9b93d80bf31fdfb4adef04))

## [1.2.0](https://github.com/open-mrp/api/compare/v1.1.11...v1.2.0) (2026-08-25)


### Features

* **platform:** alert on failed async message processing ([#48](https://github.com/open-mrp/api/issues/48)) ([5142be1](https://github.com/open-mrp/api/commit/5142be1c43da4d361045a6f56f5354c8cb2bbb98))

## [1.1.11](https://github.com/open-mrp/api/compare/v1.1.10...v1.1.11) (2026-08-25)


### Bug Fixes

* **core:** batch physical-inventory levels in batch-scan apply ([#45](https://github.com/open-mrp/api/issues/45)) ([b5be4db](https://github.com/open-mrp/api/commit/b5be4dbdf673ce9a22bad0ff5784b74bf25792ad))
* **scheduling:** measure finishing rates from machine-less scans ([#47](https://github.com/open-mrp/api/issues/47)) ([a9fef05](https://github.com/open-mrp/api/commit/a9fef055c9712a7e87d2ea643ec13e956c112136))

## [1.1.10](https://github.com/open-mrp/api/compare/v1.1.9...v1.1.10) (2026-08-25)


### Bug Fixes

* **db:** make goose_db_version_data creation idempotent ([#43](https://github.com/open-mrp/api/issues/43)) ([7fa4b25](https://github.com/open-mrp/api/commit/7fa4b251ed4958e0a9c321d872a44c550457e702))

## [1.1.9](https://github.com/open-mrp/api/compare/v1.1.8...v1.1.9) (2026-08-25)


### Bug Fixes

* **release:** gate migration deploy on real DB changes; surface pscale role errors ([#38](https://github.com/open-mrp/api/issues/38)) ([5798036](https://github.com/open-mrp/api/commit/579803617b58b132309ed1f6f185b37feb63e5e5))
* **release:** stop Postgres migrate role name collision between steps ([#42](https://github.com/open-mrp/api/issues/42)) ([2859bf3](https://github.com/open-mrp/api/commit/2859bf35eab6692101c5a726be9c1e4370bf3c63))


### Documentation

* **agents:** keep commit messages concise and attribution-free ([#41](https://github.com/open-mrp/api/issues/41)) ([b34b4e7](https://github.com/open-mrp/api/commit/b34b4e79841f5324e2b0e3052bab77bc4fe96aa0))
* **agents:** keep PR descriptions concise and follow the PR template ([#39](https://github.com/open-mrp/api/issues/39)) ([5609018](https://github.com/open-mrp/api/commit/560901856ac81f2343111aa124b2cec557d212ba))

## [1.1.8](https://github.com/open-mrp/api/compare/v1.1.7...v1.1.8) (2026-08-25)


### Bug Fixes

* **core:** recompute item burn rate off the consumption transaction ([#36](https://github.com/open-mrp/api/issues/36)) ([a028c0d](https://github.com/open-mrp/api/commit/a028c0d7bce274aec5ca7d9327b973440c035cf9))

## [1.1.7](https://github.com/open-mrp/api/compare/v1.1.6...v1.1.7) (2026-08-24)


### Bug Fixes

* update deps and fix some tests ([#33](https://github.com/open-mrp/api/issues/33)) ([5b95a9f](https://github.com/open-mrp/api/commit/5b95a9f078830bf2a58c09e1523cc77587743866))


### Documentation

* remove logo from README.md ([b8d510d](https://github.com/open-mrp/api/commit/b8d510de052ca0450e16896cf8c7d18f494ef691))
* update README ([#30](https://github.com/open-mrp/api/issues/30)) ([20f99d1](https://github.com/open-mrp/api/commit/20f99d192125d5e2569caadfebe957fbb7a47af1))

## [1.1.6](https://github.com/open-mrp/api/compare/v1.1.5...v1.1.6) (2026-08-23)


### Bug Fixes

* a few endpoints using type `string` instead of a `constant` ([#27](https://github.com/open-mrp/api/issues/27)) ([fa05ecb](https://github.com/open-mrp/api/commit/fa05ecbc0b8a920f4107e2b19fb14b72e3ee5646))

## [1.1.5](https://github.com/open-mrp/api/compare/v1.1.4...v1.1.5) (2026-08-23)


### Bug Fixes

* keep generated client consistent with org name ([#21](https://github.com/open-mrp/api/issues/21)) ([ea7917d](https://github.com/open-mrp/api/commit/ea7917db345e6926377c39ab94b405cf913f3373))

## [1.1.4](https://github.com/open-mrp/api/compare/v1.1.3...v1.1.4) (2026-08-23)


### Bug Fixes

* allow category property includes in volume discounts ([#19](https://github.com/open-mrp/api/issues/19)) ([98c8cc8](https://github.com/open-mrp/api/commit/98c8cc801a5205a3a5fd999047881797b302201f))

## [1.1.3](https://github.com/open-mrp/api/compare/v1.1.2...v1.1.3) (2026-08-23)


### Bug Fixes

* drop dual-domain support now that only openmrp.ai is served ([#16](https://github.com/open-mrp/api/issues/16)) ([374a350](https://github.com/open-mrp/api/commit/374a3502f764fcd25821c4a55ad0a9ba298ba51c))

## [1.1.2](https://github.com/open-mrp/api/compare/v1.1.1...v1.1.2) (2026-08-23)


### Bug Fixes

* resolve the SDK ref including pre-releases ([#14](https://github.com/open-mrp/api/issues/14)) ([1429ad2](https://github.com/open-mrp/api/commit/1429ad2bbfaa5a4dd974ad9eae089409c42f3849))

## [1.1.1](https://github.com/open-mrp/api/compare/v1.1.0...v1.1.1) (2026-08-22)


### Bug Fixes

* resolve the MCP build context from typescript-sdk's own releases ([#12](https://github.com/open-mrp/api/issues/12)) ([4d25578](https://github.com/open-mrp/api/commit/4d255786b1bfdb2c0693de6b07868b1ea11a984e))

## [1.1.0](https://github.com/open-mrp/api/compare/v1.0.1...v1.1.0) (2026-08-22)


### Features

* add --all-services to force a full release build ([#10](https://github.com/open-mrp/api/issues/10)) ([7efca4f](https://github.com/open-mrp/api/commit/7efca4f0d902bfbba5029a66b79b014754f2368e))

## [1.0.1](https://github.com/open-mrp/api/compare/v1.0.0...v1.0.1) (2026-08-22)


### Bug Fixes

* push images to the augno/ ECR namespace again ([#8](https://github.com/open-mrp/api/issues/8)) ([c6a3668](https://github.com/open-mrp/api/commit/c6a36686b30d70d527b4600827b5398fe0e6ade0))

## [1.0.0](https://github.com/open-mrp/api/compare/v0.50.7...v1.0.0) (2026-08-22)


### ⚠ BREAKING CHANGES

* rename Augno to OpenMRP ([#3](https://github.com/open-mrp/api/issues/3))

### Bug Fixes

* unblock the E2E suite and stop shipment lists 404ing on a concurrent delete ([#6](https://github.com/open-mrp/api/issues/6)) ([44e1c4f](https://github.com/open-mrp/api/commit/44e1c4f83bbcdb04ea1afd15699e2747c31db186))


### Miscellaneous

* rename Augno to OpenMRP ([#3](https://github.com/open-mrp/api/issues/3)) ([c2f082c](https://github.com/open-mrp/api/commit/c2f082cc24f864854a73983e57fc3c89f17c6c84))

## [0.50.7](https://github.com/open-mrp/api/compare/v0.50.6...v0.50.7) (2026-08-21)


### Bug Fixes

* stop baking the generating machine's paths into mocks ([171b738](https://github.com/open-mrp/api/commit/171b738b9cb0d0617d5d8966af4e1703bad5c321))


### Documentation

* start the changelog at the first public release ([85ea94f](https://github.com/open-mrp/api/commit/85ea94fa1240f5f39aef2e40fe552936b3d33903))

## [0.50.6](https://github.com/open-mrp/api/compare/v0.50.5...v0.50.6) (2026-08-21)


### Bug Fixes

* replace status with constant rather than string type in requests ([aec5041](https://github.com/open-mrp/api/commit/aec50417c88d3c6a03119d302bcbac5a7c2fb22b))
* update InputSchema for sales orders endpoint to improve clarity and accuracy ([4ff1622](https://github.com/open-mrp/api/commit/4ff16220009ef5de77d295a3b94d5b68ab0cb4f5))

---

Releases before v0.50.6 predate this repository being opened up. That history was rewritten to
remove production infrastructure and customer data before publication, so the original commits no
longer exist here and entries referring to them would only have pointed at dead links. The full
pre-release changelog is retained in OpenMRP's internal archive.
