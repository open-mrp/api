# Changelog

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
