# Changelog

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
