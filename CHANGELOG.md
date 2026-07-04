# Changelog

## [0.39.0](https://github.com/Augno/api/compare/v0.38.1...v0.39.0) (2026-07-04)


### Features

* add DeleteEmailDomain functionality to EmailBridgeService ([#404](https://github.com/Augno/api/issues/404)) ([444f917](https://github.com/Augno/api/commit/444f9171dfcee60c91986f90c21df052e99f9c9f))


### Bug Fixes

* implement SwitchAccountDefaultAddressToRelation method to manage default address transitions for non-active accounts ([6a712b0](https://github.com/Augno/api/commit/6a712b04792389914e39f6da2b6fec1e85b38af6))

## [0.38.1](https://github.com/Augno/api/compare/v0.38.0...v0.38.1) (2026-07-03)


### Bug Fixes

* add network policy rule for notification-service to prevent dial timeouts ([d6bde12](https://github.com/Augno/api/commit/d6bde12af2fd7a47afd597f25f091a7a4d6e99aa))

## [0.38.0](https://github.com/Augno/api/compare/v0.37.0...v0.38.0) (2026-07-03)


### Features

* uk on machine account_id, name ([#400](https://github.com/Augno/api/issues/400)) ([9d7cd77](https://github.com/Augno/api/commit/9d7cd7796aac7638047699309bd5bdba717fda56))


### Bug Fixes

* implement topology spread constraints for improved service resilience across Kubernetes deployments ([#401](https://github.com/Augno/api/issues/401)) ([cf85137](https://github.com/Augno/api/commit/cf851371c424a066cd2df2a44d21950e6f744355))

## [0.37.0](https://github.com/Augno/api/compare/v0.36.0...v0.37.0) (2026-07-03)


### Features

* enhance email handling by attributing external senders in conversation service ([62cf4bd](https://github.com/Augno/api/commit/62cf4bd18d346135d10ae47c1f19d0884752e0b1))

## [0.36.0](https://github.com/Augno/api/compare/v0.35.0...v0.36.0) (2026-07-03)


### Features

* nonnull account id on machine ([#397](https://github.com/Augno/api/issues/397)) ([f4cf46d](https://github.com/Augno/api/commit/f4cf46ddf7f0f37ceef0b3a5543aefb335b5e8ef))

## [0.35.0](https://github.com/Augno/api/compare/v0.34.0...v0.35.0) (2026-07-03)


### Features

* UK for depatment, product_line, scanning_station, production_step; nullable machine account_id ([#396](https://github.com/Augno/api/issues/396)) ([3b3d06b](https://github.com/Augno/api/commit/3b3d06b2f957164185d40382f898ba0bedf699fa))


### Bug Fixes

* notify participants of conversation updates when read cursor advances ([fffa05f](https://github.com/Augno/api/commit/fffa05f085782f56d1c377b5929c095a1f2840c9))

## [0.34.0](https://github.com/Augno/api/compare/v0.33.0...v0.34.0) (2026-07-03)


### Features

* add group ID support for email inboxes to manage messaging groups ([fbff4bc](https://github.com/Augno/api/commit/fbff4bc2ce0db212bcb3ec76e7b4573a4e01a68c))

## [0.33.0](https://github.com/Augno/api/compare/v0.32.2...v0.33.0) (2026-07-03)


### Features

* add support for per-inbox forwarding addresses in email service ([7de11cc](https://github.com/Augno/api/commit/7de11ccd53d5af52e721e9e0aa97817973a0e430))

## [0.32.2](https://github.com/Augno/api/compare/v0.32.1...v0.32.2) (2026-07-02)


### Bug Fixes

* websocket issues in production preventing display of messages instantly for sender ([badc2ab](https://github.com/Augno/api/commit/badc2ab7877c9b6a0fbe3e81391205439c7defae))

## [0.32.1](https://github.com/Augno/api/compare/v0.32.0...v0.32.1) (2026-07-02)


### Bug Fixes

* agent action approval getting stuck ([cc280e0](https://github.com/Augno/api/commit/cc280e0aa1fd822759ea8c761da90af929e63b64))

## [0.32.0](https://github.com/Augno/api/compare/v0.31.3...v0.32.0) (2026-07-02)


### Features

* implement idle watchdog and timeout for LLM streaming connections ([ad824b6](https://github.com/Augno/api/commit/ad824b61d22f5ee343d6e996a40a7afc4d434d2f))


### Bug Fixes

* tighten enqueuer polling intervals and add outbox notifier to runner service ([5461f3b](https://github.com/Augno/api/commit/5461f3b6469b2a76f5320dff6b3326a8941f0352))

## [0.31.3](https://github.com/Augno/api/compare/v0.31.2...v0.31.3) (2026-07-02)


### Bug Fixes

* update resource requests and limits for multiple services in Kubernetes configuration ([3082fe3](https://github.com/Augno/api/commit/3082fe316b2cfcac378c4b56b972055391635bc2))

## [0.31.2](https://github.com/Augno/api/compare/v0.31.1...v0.31.2) (2026-07-02)


### Bug Fixes

* update resource requests and limits for agent service in Kubernetes configuration ([9c06c7d](https://github.com/Augno/api/commit/9c06c7dc73675e349511bee8182936a54fce4403))

## [0.31.1](https://github.com/Augno/api/compare/v0.31.0...v0.31.1) (2026-07-02)


### Bug Fixes

* grant CI infra role sqs permissions for inbound-email queues ([#386](https://github.com/Augno/api/issues/386)) ([eeaab31](https://github.com/Augno/api/commit/eeaab319a7d096bbc32125b4509dd36493ba1a99))

## [0.31.0](https://github.com/Augno/api/compare/v0.30.1...v0.31.0) (2026-07-02)


### Features

* chat and agents ([#384](https://github.com/Augno/api/issues/384)) ([3ed0615](https://github.com/Augno/api/commit/3ed0615c774a4176e24d2cdd7b97dd3ca65714e8))

## [0.30.1](https://github.com/Augno/api/compare/v0.30.0...v0.30.1) (2026-06-23)


### Bug Fixes

* seed sales order number allocation from max order number ([#382](https://github.com/Augno/api/issues/382)) ([9c9f2c7](https://github.com/Augno/api/commit/9c9f2c7f00f07962b887549f939c28a5c12b2857))

## [0.30.0](https://github.com/Augno/api/compare/v0.29.0...v0.30.0) (2026-06-19)


### Features

* unit group UK for account_id, name ([#380](https://github.com/Augno/api/issues/380)) ([b034a3b](https://github.com/Augno/api/commit/b034a3bc7e4bf72ec42b036fdbe496a428e70a9d))

## [0.29.0](https://github.com/Augno/api/compare/v0.28.3...v0.29.0) (2026-06-19)


### Features

* make sales order list and create endpoints public ([#378](https://github.com/Augno/api/issues/378)) ([ea9d1d1](https://github.com/Augno/api/commit/ea9d1d1dfeac3012a0e441084c86b85438e79aa7))

## [0.28.3](https://github.com/Augno/api/compare/v0.28.2...v0.28.3) (2026-06-18)


### Bug Fixes

* update Stripe credentials decryption to include account ID as additional authenticated data ([5993f7e](https://github.com/Augno/api/commit/5993f7eab50db57d47b89f399b59d83708c24e5e))

## [0.28.2](https://github.com/Augno/api/compare/v0.28.1...v0.28.2) (2026-06-18)


### Bug Fixes

* hotfix for sales order creation bug preventing rates from landing on carriers ([fa46aee](https://github.com/Augno/api/commit/fa46aee1fd86c04a70b295c6e560f17c6a0030a8))

## [0.28.1](https://github.com/Augno/api/compare/v0.28.0...v0.28.1) (2026-06-18)


### Bug Fixes

* so product line item included without request ([#372](https://github.com/Augno/api/issues/372)) ([8ac8076](https://github.com/Augno/api/commit/8ac8076c9a929c369c0d3fb6cd9be677f2c26130))

## [0.28.0](https://github.com/Augno/api/compare/v0.27.0...v0.28.0) (2026-06-18)


### Features

* adding contacts to sales orders ([#366](https://github.com/Augno/api/issues/366)) ([5091a4c](https://github.com/Augno/api/commit/5091a4cd9832fd8dd40b0afa5e3e6f5388158923))


### Bug Fixes

* **ci:** grant id-token: write to manual SDK generate job ([#369](https://github.com/Augno/api/issues/369)) ([2a5d1d8](https://github.com/Augno/api/commit/2a5d1d897f994a040c45a21153c1c15b4aa45f6b))

## [0.27.0](https://github.com/Augno/api/compare/v0.26.3...v0.27.0) (2026-06-18)


### Features

* sales order creation ([#362](https://github.com/Augno/api/issues/362)) ([1f52475](https://github.com/Augno/api/commit/1f52475d047d1660937e37a34c96a7fcb1d72846))

## [0.26.3](https://github.com/Augno/api/compare/v0.26.2...v0.26.3) (2026-06-17)


### Bug Fixes

* adding some indexes ([82a2838](https://github.com/Augno/api/commit/82a28386e452741b90a942bb2451a71849818a89))
* **sales-orders:** derive payment_status from invoice paid-in-full, matching legacy ([#361](https://github.com/Augno/api/issues/361)) ([8949fee](https://github.com/Augno/api/commit/8949feed3f5caadfc9ffcaa0e3dc567212a28c50))
* **sdk:** generate internal-sdk release workflow via stlc ([#359](https://github.com/Augno/api/issues/359)) ([6ed20c5](https://github.com/Augno/api/commit/6ed20c5713bc93b379eead94e9e81c98e8042256))

## [0.26.2](https://github.com/Augno/api/compare/v0.26.1...v0.26.2) (2026-06-16)


### Bug Fixes

* faster queries in sales orders ([a096a94](https://github.com/Augno/api/commit/a096a949a3e737a95cf778aec351642aaa775f85))

## [0.26.1](https://github.com/Augno/api/compare/v0.26.0...v0.26.1) (2026-06-16)


### Bug Fixes

* issue with querying sales orders not pulling in all filters ([c7949e5](https://github.com/Augno/api/commit/c7949e5524321f3ce0d63c2004ff02a5f0a342e3))

## [0.26.0](https://github.com/Augno/api/compare/v0.25.4...v0.26.0) (2026-06-16)


### Features

* improved query speed and created_by field added to sales orders ([#355](https://github.com/Augno/api/issues/355)) ([cef9fe4](https://github.com/Augno/api/commit/cef9fe45805d9e44d712466bc6ed3870e696104c))

## [0.25.4](https://github.com/Augno/api/compare/v0.25.3...v0.25.4) (2026-06-15)


### Bug Fixes

* internal sdk release please ([#353](https://github.com/Augno/api/issues/353)) ([a354e9e](https://github.com/Augno/api/commit/a354e9e999d218e46275eac9a46a19e294fc1ba0))

## [0.25.3](https://github.com/Augno/api/compare/v0.25.2...v0.25.3) (2026-06-15)


### Bug Fixes

* issue with s3 storage ([#351](https://github.com/Augno/api/issues/351)) ([f59b693](https://github.com/Augno/api/commit/f59b6930b00c221d5d9e025321b26f520e77d5d1))

## [0.25.2](https://github.com/Augno/api/compare/v0.25.1...v0.25.2) (2026-06-15)


### Bug Fixes

* **ci:** avoid OIDC in manual SDK generation ([#344](https://github.com/Augno/api/issues/344)) ([dd139f7](https://github.com/Augno/api/commit/dd139f759ff3add62fcffbdd5ed04557caed2554))
* **ci:** avoid persisting sdk checkout token ([#342](https://github.com/Augno/api/issues/342)) ([e7969f9](https://github.com/Augno/api/commit/e7969f9584d9e4103b1572993341467f8512129e))
* **core-service:** enforce address RBAC for no-role internal users ([#345](https://github.com/Augno/api/issues/345)) ([503eabb](https://github.com/Augno/api/commit/503eabb24fe71766d8a40192b1c07b36c284c4de))
* harden manual MCP deployment ([#343](https://github.com/Augno/api/issues/343)) ([b5e4c62](https://github.com/Augno/api/commit/b5e4c62aa5493fb482ee446316afa54a288b3852))
* harden manual release resume ([#341](https://github.com/Augno/api/issues/341)) ([0d2a3c2](https://github.com/Augno/api/commit/0d2a3c22deb333de78af721ba54a905c650ddff9))
* harden release terraform gate ([#349](https://github.com/Augno/api/issues/349)) ([075a7f4](https://github.com/Augno/api/commit/075a7f4fec384ac70da8b90267e399fb8e2d1e04))
* permission issue will be fixed in the future ([ee6e7e0](https://github.com/Augno/api/commit/ee6e7e0ad658586e326fe7a2eac10a49b9b67bef))
* preserve intentional audit update events ([#347](https://github.com/Augno/api/issues/347)) ([9d56f07](https://github.com/Augno/api/commit/9d56f07f89012529dc45b47cc963919752bfe737))
* preserve target account on async audit events ([#346](https://github.com/Augno/api/issues/346)) ([68c762e](https://github.com/Augno/api/commit/68c762ef7af0f17fb9dc3f8b7d0592e7ea4c7af6))
* prevent rotating revoked api keys ([#348](https://github.com/Augno/api/issues/348)) ([bd8a7a1](https://github.com/Augno/api/commit/bd8a7a1d2cda5ed0ca5491f9b0e97d38125b5919))
* use read-only role for terraform-ci PR plans ([#340](https://github.com/Augno/api/issues/340)) ([4e614d3](https://github.com/Augno/api/commit/4e614d3b21e138e28229a8adbe8558c97223fee5))

## [0.25.1](https://github.com/Augno/api/compare/v0.25.0...v0.25.1) (2026-06-15)


### Bug Fixes

* minor changes in docs ([a47fdca](https://github.com/Augno/api/commit/a47fdca1dd56e5fb3cd52ef48f47dc0bed1c244e))

## [0.25.0](https://github.com/Augno/api/compare/v0.24.0...v0.25.0) (2026-06-15)


### Features

* add filters for target and acting accounts in request logs and audit events ([#334](https://github.com/Augno/api/issues/334)) ([6efc522](https://github.com/Augno/api/commit/6efc522f7cc1005888cc65d56677809ae0da9e31))

## [0.24.0](https://github.com/Augno/api/compare/v0.23.4...v0.24.0) (2026-06-12)


### Features

* add target account filtering to audit events ([#333](https://github.com/Augno/api/issues/333)) ([4b4f5d8](https://github.com/Augno/api/commit/4b4f5d8c8f70f6355cdf6f2cdbce724831f57461))
* **ci:** add manual Deploy MCP Server workflow ([#330](https://github.com/Augno/api/issues/330)) ([4164fa4](https://github.com/Augno/api/commit/4164fa47c4898770e8801196a524aee19612d3f8))


### Bug Fixes

* **ci:** unblock the manual SDK-generation workflow ([#328](https://github.com/Augno/api/issues/328)) ([c8fa935](https://github.com/Augno/api/commit/c8fa935b0bacf2a47ee3a7a184ffea2da90a19b0))
* **ci:** use SDK_WRITE_TOKEN for typescript-sdk checkout in MCP build ([#331](https://github.com/Augno/api/issues/331)) ([7d4bd69](https://github.com/Augno/api/commit/7d4bd690fe400758a0dcb4bbc05c5eb8492ad4aa))
* **mcp:** pin runAsUser/runAsGroup 1001 for mcp-server pods ([#332](https://github.com/Augno/api/issues/332)) ([35c91a0](https://github.com/Augno/api/commit/35c91a0137e967a81551084882d70cea1dbe0897))

## [0.23.4](https://github.com/Augno/api/compare/v0.23.3...v0.23.4) (2026-06-11)


### Bug Fixes

* update comments and documentation for clarity across various endpoints ([c810003](https://github.com/Augno/api/commit/c810003a679217d0052f4953291015d1feaacd72))

## [0.23.3](https://github.com/Augno/api/compare/v0.23.2...v0.23.3) (2026-06-11)


### Bug Fixes

* issues with deployment ([067d108](https://github.com/Augno/api/commit/067d108439fda73e064e36acd3d4ebea58cbdfd0))

## [0.23.2](https://github.com/Augno/api/compare/v0.23.1...v0.23.2) (2026-06-11)


### Bug Fixes

* deployment issues ([0125740](https://github.com/Augno/api/commit/01257403fc55c3d19baade448f4a8f9a984f6804))
* minor change of docs ([b8d243a](https://github.com/Augno/api/commit/b8d243a5ae446a6f2fcd6523e8540563ec837b5f))

## [0.23.1](https://github.com/Augno/api/compare/v0.23.0...v0.23.1) (2026-06-11)


### Bug Fixes

* update descriptions and comments across various resources for clarity and completeness ([d5c814a](https://github.com/Augno/api/commit/d5c814a859c1000c971202f9ab04f2a73433e814))

## [0.23.0](https://github.com/Augno/api/compare/v0.22.0...v0.23.0) (2026-06-11)


### Features

* add ACM certificate for mcp.augno.com ([#319](https://github.com/Augno/api/issues/319)) ([05d92d7](https://github.com/Augno/api/commit/05d92d73612be6cccc956b62dba4bc7ebd823f56))

## [0.22.0](https://github.com/Augno/api/compare/v0.21.5...v0.22.0) (2026-06-11)


### Features

* add per-job least-privilege IAM roles for GitHub Actions (phase A) ([#317](https://github.com/Augno/api/issues/317)) ([18f4c4d](https://github.com/Augno/api/commit/18f4c4ded8e1a39884bfbde550c2e4ac80d4d8ff))

## [0.21.5](https://github.com/Augno/api/compare/v0.21.4...v0.21.5) (2026-06-11)


### Bug Fixes

* allow rabbitmq pod eviction so node drains don't deadlock ([#316](https://github.com/Augno/api/issues/316)) ([7d8f33e](https://github.com/Augno/api/commit/7d8f33eff3cd26565b7faf8c814937620e5a68cc))
* sync Augno-Version default header with spec version on generation ([#320](https://github.com/Augno/api/issues/320)) ([4b8f820](https://github.com/Augno/api/commit/4b8f8201575e2c683ef4a9fed14c383b00581654))

## [0.21.4](https://github.com/Augno/api/compare/v0.21.3...v0.21.4) (2026-06-10)


### Bug Fixes

* harden mcp server ([e9f8fa5](https://github.com/Augno/api/commit/e9f8fa51a0a155df061f576cc996c7052c8bde7f))

## [0.21.3](https://github.com/Augno/api/compare/v0.21.2...v0.21.3) (2026-06-10)


### Bug Fixes

* minor docs updates; mcp server; e2e test flaky fixes ([35b4762](https://github.com/Augno/api/commit/35b476297b9784a4e8f5a0fe062a5ce7ef4e0e00))

## [0.21.2](https://github.com/Augno/api/compare/v0.21.1...v0.21.2) (2026-06-10)


### Bug Fixes

* db in sync with prod ([19efb9c](https://github.com/Augno/api/commit/19efb9cd5c5bb4fcfb72082134a93dc3909fe121))
* faster polling for messages for prod ([bce3ac5](https://github.com/Augno/api/commit/bce3ac5e18756d0401445059093d805baf188a1c))

## [0.21.1](https://github.com/Augno/api/compare/v0.21.0...v0.21.1) (2026-06-10)


### Bug Fixes

* flaky e2e tests and release issues on sdks ([8bb9a60](https://github.com/Augno/api/commit/8bb9a608410ebfb14fb62192e0cda0a7976a706c))
* make SDK codegen robust to config drift + set up uv for Python ([5e655f3](https://github.com/Augno/api/commit/5e655f303a5d71894b27b609eaed234a17520071))

## [0.21.0](https://github.com/Augno/api/compare/v0.20.2...v0.21.0) (2026-06-10)


### Features

* user sub-object on account user rather than using inline fields ([#310](https://github.com/Augno/api/issues/310)) ([3e4ac35](https://github.com/Augno/api/commit/3e4ac35cfcae3335002f8608e9a924dee0f7bac9))


### Bug Fixes

* generate release-please workflow for python/go SDKs natively ([#311](https://github.com/Augno/api/issues/311)) ([ed6c468](https://github.com/Augno/api/commit/ed6c4680ce23c8d72f3e294a7cf3e911f34ad7d1))
* promote LocationTypeCode to a named schema for Go SDK ([#308](https://github.com/Augno/api/issues/308)) ([e620ba3](https://github.com/Augno/api/commit/e620ba35f53919cefa1b4047f2334b5854a73c3d))

## [0.20.2](https://github.com/Augno/api/compare/v0.20.1...v0.20.2) (2026-06-10)


### Bug Fixes

* version bumps in the api should flow through to the sdks (patch, minor, major) ([c45bdb8](https://github.com/Augno/api/commit/c45bdb881390370678373c4de097ba4a10940a28))

## [0.20.1](https://github.com/Augno/api/compare/v0.20.0...v0.20.1) (2026-06-10)


### Bug Fixes

* enforce docs/ patterns and conventions across the codebase ([#305](https://github.com/Augno/api/issues/305)) ([ad952d9](https://github.com/Augno/api/commit/ad952d97a14fb1cf92e39a7eaa073f54b305963c))

## [0.20.0](https://github.com/Augno/api/compare/v0.19.5...v0.20.0) (2026-06-09)


### Features

* adding new revoke_at param for api key rotation ([#303](https://github.com/Augno/api/issues/303)) ([1f3b492](https://github.com/Augno/api/commit/1f3b492991d5dd31d791c40ecdcda0cb284daabe))

## [0.19.5](https://github.com/Augno/api/compare/v0.19.4...v0.19.5) (2026-06-08)


### Bug Fixes

* includes fields not working as expected ([#301](https://github.com/Augno/api/issues/301)) ([ebdaafe](https://github.com/Augno/api/commit/ebdaafe11645d8ae4ab750ae370139677d21decb))

## [0.19.4](https://github.com/Augno/api/compare/v0.19.3...v0.19.4) (2026-06-04)


### Bug Fixes

* enhance OpenAPI and Stainless SDK generation in Makefile and workflows ([44c6b97](https://github.com/Augno/api/commit/44c6b97d7a13abb0fd49daba3980dcef14083770))

## [0.19.3](https://github.com/Augno/api/compare/v0.19.2...v0.19.3) (2026-06-03)


### Bug Fixes

* add Cache-Control and Pragma to CORS headers for enhanced request handling ([03768f8](https://github.com/Augno/api/commit/03768f8b22a450bb35c98eb14feab8b98bcefc8c))
* add User-Agent to CORS headers for improved request handling ([2d64d59](https://github.com/Augno/api/commit/2d64d59e99e70cb22c2fbfe9aed4042300dc3a0a))

## [0.19.2](https://github.com/Augno/api/compare/v0.19.1...v0.19.2) (2026-06-02)


### Bug Fixes

* update Tiltfile paths for service builds and enhance request log queries for account-user resolution ([30e78bd](https://github.com/Augno/api/commit/30e78bde34db86dcbf6d305e6070185598228229))

## [0.19.1](https://github.com/Augno/api/compare/v0.19.0...v0.19.1) (2026-06-02)


### Bug Fixes

* implement account-agnostic user authentication checks in tenancy service ([14a0904](https://github.com/Augno/api/commit/14a0904a017a6eb7ae16b4c38bd1d70cf04d5e12))

## [0.19.0](https://github.com/Augno/api/compare/v0.18.10...v0.19.0) (2026-06-02)


### Features

* enhance tenancy service with user authentication check and improve request log queries ([9435b06](https://github.com/Augno/api/commit/9435b061b297dc75cacc6ad618ab0921320f5186))


### Bug Fixes

* add new indexing keys to database migration scripts for improved query performance ([5d5fd51](https://github.com/Augno/api/commit/5d5fd515e142cf2fdea08a3c299eeb16d2ca3330))


### Code Refactoring

* remove unnecessary keys from database migration scripts for improved clarity and performance ([2747293](https://github.com/Augno/api/commit/274729363a8e2d8d9b1a691a22f5e20f7b0504d7))

## [0.18.10](https://github.com/Augno/api/compare/v0.18.9...v0.18.10) (2026-06-02)


### Code Refactoring

* update resource includes across multiple services to use resourcekit.FilterIncludes for improved performance and consistency ([#293](https://github.com/Augno/api/issues/293)) ([6c886e9](https://github.com/Augno/api/commit/6c886e9b5940559916ad338ff8af9cfb97f7773b))

## [0.18.9](https://github.com/Augno/api/compare/v0.18.8...v0.18.9) (2026-06-01)


### Bug Fixes

* specify npm registry in stainless.yml configuration ([5ebc7bb](https://github.com/Augno/api/commit/5ebc7bb07eb0c8e8f527e2fb926add9077f5848d))

## [0.18.8](https://github.com/Augno/api/compare/v0.18.7...v0.18.8) (2026-06-01)


### Bug Fixes

* burn rate based on inventory changes; improved sdk gen paths for objects based on domain root; improved openapi examples for next_page_url ([#290](https://github.com/Augno/api/issues/290)) ([b2c6ae3](https://github.com/Augno/api/commit/b2c6ae376d81ab221b68c0ada8d15e0abeff1048))

## [0.18.7](https://github.com/Augno/api/compare/v0.18.6...v0.18.7) (2026-06-01)


### Bug Fixes

* add clarification on AccountGroup type field usage ([5f0b9d0](https://github.com/Augno/api/commit/5f0b9d0285a84522b108c962a5c7c42cebb68e7a))

## [0.18.6](https://github.com/Augno/api/compare/v0.18.5...v0.18.6) (2026-06-01)


### Bug Fixes

* unique name error on SDK build for a couple of endpoints ([a281cb6](https://github.com/Augno/api/commit/a281cb6c1a64511a80cdefcaa58fda46dcfc4709))

## [0.18.5](https://github.com/Augno/api/compare/v0.18.4...v0.18.5) (2026-06-01)


### Bug Fixes

* sub-resource include performance ([#286](https://github.com/Augno/api/issues/286)) ([e6e3d70](https://github.com/Augno/api/commit/e6e3d7070e654303c6a8a16698b31df34c3152cf))

## [0.18.4](https://github.com/Augno/api/compare/v0.18.3...v0.18.4) (2026-05-21)


### Bug Fixes

* auth types in sdks ([#285](https://github.com/Augno/api/issues/285)) ([0527c50](https://github.com/Augno/api/commit/0527c50a5323615b9736b8a894025abb6781a8b9))
* don't notify dashboard about release ([e5d0cb4](https://github.com/Augno/api/commit/e5d0cb4b63214637269d512a7ff6224661bc228f))

## [0.18.3](https://github.com/Augno/api/compare/v0.18.2...v0.18.3) (2026-05-21)


### Bug Fixes

* working on improving release workflow to deploy only what changed. ([#282](https://github.com/Augno/api/issues/282)) ([b8406ee](https://github.com/Augno/api/commit/b8406eed14a3ea3c9740613d6ce29eca2973b8ad))

## [0.18.2](https://github.com/Augno/api/compare/v0.18.1...v0.18.2) (2026-05-21)


### Bug Fixes

* simplify sdk release process ([#280](https://github.com/Augno/api/issues/280)) ([3766aed](https://github.com/Augno/api/commit/3766aed82ba7bf60798b0d8085fba95fe3feb8bf))

## [0.18.1](https://github.com/Augno/api/compare/v0.18.0...v0.18.1) (2026-05-21)


### Bug Fixes

* issue with terraform skip causing entire release to skip ([#278](https://github.com/Augno/api/issues/278)) ([60bd5d2](https://github.com/Augno/api/commit/60bd5d2f9d346b9c48915b55b382d6cd78560d58))

## [0.18.0](https://github.com/Augno/api/compare/v0.17.15...v0.18.0) (2026-05-21)


### Features

* enhance CI workflow to detect Terraform changes before applying ([#276](https://github.com/Augno/api/issues/276)) ([1578cba](https://github.com/Augno/api/commit/1578cbac480d5f836ee0cfa03dd0c2947cd2337f))

## [0.17.15](https://github.com/Augno/api/compare/v0.17.14...v0.17.15) (2026-05-21)


### Bug Fixes

* cd workflow to create release in typescript sdk ([#274](https://github.com/Augno/api/issues/274)) ([7b99f8b](https://github.com/Augno/api/commit/7b99f8bf01cb103e3aa6dc5a3007c8b6fff1049e))

## [0.17.14](https://github.com/Augno/api/compare/v0.17.13...v0.17.14) (2026-05-21)


### Bug Fixes

* mark some endpoints as public ([#273](https://github.com/Augno/api/issues/273)) ([a9725a2](https://github.com/Augno/api/commit/a9725a24def83a46dbc303f76ba58bfd39943917))
* remove old release notification for internal sdk ([#271](https://github.com/Augno/api/issues/271)) ([22e86f0](https://github.com/Augno/api/commit/22e86f022ff1a45bbeab082e08cf27f0d53cf2ba))

## [0.17.13](https://github.com/Augno/api/compare/v0.17.12...v0.17.13) (2026-05-20)


### Bug Fixes

* better test coverage on includes fields; CD to parallelize SDK release and consumer notification ([#267](https://github.com/Augno/api/issues/267)) ([145947b](https://github.com/Augno/api/commit/145947b0b4e3aa0a3a23fc99ee0f349c95650199))
* more efficient CD by checking for changes in openapi spec before release ([#269](https://github.com/Augno/api/issues/269)) ([131b694](https://github.com/Augno/api/commit/131b69455b32c6d6b12068a7b27b4b7fcab96ad9))
* release ([#270](https://github.com/Augno/api/issues/270)) ([ca4d049](https://github.com/Augno/api/commit/ca4d0496aad882791ef3fd072d7fc8ed2b9faf6c))

## [0.17.12](https://github.com/Augno/api/compare/v0.17.11...v0.17.12) (2026-05-20)


### Bug Fixes

* include failures on non-GET requests ([#265](https://github.com/Augno/api/issues/265)) ([9ab23b9](https://github.com/Augno/api/commit/9ab23b9f3df62dfc7532798184e9ace018b94ac2))

## [0.17.11](https://github.com/Augno/api/compare/v0.17.10...v0.17.11) (2026-05-20)


### Bug Fixes

* new CI/CD process to gen sdks ([#257](https://github.com/Augno/api/issues/257)) ([095b080](https://github.com/Augno/api/commit/095b080c5738d4b2a7d4690280498156f79857a7))

## [0.17.10](https://github.com/Augno/api/compare/v0.17.9...v0.17.10) (2026-05-19)


### Bug Fixes

* examples in docs ([#255](https://github.com/Augno/api/issues/255)) ([4eb5b01](https://github.com/Augno/api/commit/4eb5b01065f6de44ee4969c8245d698cff30edca))

## [0.17.9](https://github.com/Augno/api/compare/v0.17.8...v0.17.9) (2026-05-19)


### Bug Fixes

* issue with listing parts ([#253](https://github.com/Augno/api/issues/253)) ([e765359](https://github.com/Augno/api/commit/e76535954307ad17eb97d7a84bd28faf79722081))

## [0.17.8](https://github.com/Augno/api/compare/v0.17.7...v0.17.8) (2026-05-19)


### Bug Fixes

* list items filter 'initial_only' broken ([#251](https://github.com/Augno/api/issues/251)) ([85787c0](https://github.com/Augno/api/commit/85787c057424231d99f17b443626d511700d4be1))

## [0.17.7](https://github.com/Augno/api/compare/v0.17.6...v0.17.7) (2026-05-19)


### Bug Fixes

* patch nullable fields ([#249](https://github.com/Augno/api/issues/249)) ([a05d111](https://github.com/Augno/api/commit/a05d111a3ab802bf0e3eed361edcf1d2d93285ee))

## [0.17.6](https://github.com/Augno/api/compare/v0.17.5...v0.17.6) (2026-05-18)


### Bug Fixes

* improved documentation ([#248](https://github.com/Augno/api/issues/248)) ([38beac2](https://github.com/Augno/api/commit/38beac277ed552aa049252a4799a50af5018f683))


### Documentation

* update api endpoints to use docstrings rather than explicit description fields ([#246](https://github.com/Augno/api/issues/246)) ([f279947](https://github.com/Augno/api/commit/f279947a864b99c0c3fb1cc60c6e8c79fa0ec4e5))

## [0.17.5](https://github.com/Augno/api/compare/v0.17.4...v0.17.5) (2026-05-15)


### Bug Fixes

* ci process ([#244](https://github.com/Augno/api/issues/244)) ([ed5feac](https://github.com/Augno/api/commit/ed5feac98b7d75e59adb1ef2dd0ccc4b3489dc66))

## [0.17.4](https://github.com/Augno/api/compare/v0.17.3...v0.17.4) (2026-05-14)


### Bug Fixes

* _parent_child_production_steps column used in initial subassembly check ([#241](https://github.com/Augno/api/issues/241)) ([05540d6](https://github.com/Augno/api/commit/05540d67a9b91efcdaba06c385c4d5b60dd134ce))

## [0.17.3](https://github.com/Augno/api/compare/v0.17.2...v0.17.3) (2026-05-14)


### Bug Fixes

* issues with bulk exports ([#239](https://github.com/Augno/api/issues/239)) ([55bee64](https://github.com/Augno/api/commit/55bee642d2e973bea3392e21e1c1177ff06665c0))

## [0.17.2](https://github.com/Augno/api/compare/v0.17.1...v0.17.2) (2026-05-14)


### Bug Fixes

* queries for bulk updates ([#237](https://github.com/Augno/api/issues/237)) ([ec3e1dd](https://github.com/Augno/api/commit/ec3e1dd6141591bc53ea285889100f19060811b1))

## [0.17.1](https://github.com/Augno/api/compare/v0.17.0...v0.17.1) (2026-05-13)


### Bug Fixes

* issue with performance of exports ([#235](https://github.com/Augno/api/issues/235)) ([75bf6e4](https://github.com/Augno/api/commit/75bf6e44c2f8bf2550e56d48c92e8c6a6102dc20))

## [0.17.0](https://github.com/Augno/api/compare/v0.16.0...v0.17.0) (2026-05-13)


### Features

* export items in bulk ([#233](https://github.com/Augno/api/issues/233)) ([8bb7843](https://github.com/Augno/api/commit/8bb78436715eee0899d1950f652c43491bd4b096))

## [0.16.0](https://github.com/Augno/api/compare/v0.15.6...v0.16.0) (2026-05-13)


### Features

* adding items endpoints to public api ([#232](https://github.com/Augno/api/issues/232)) ([6d81ba6](https://github.com/Augno/api/commit/6d81ba698ce14a61770006743b3ece94796f925a))


### Bug Fixes

* adding new col to tx for funds settlement date ([#229](https://github.com/Augno/api/issues/229)) ([59094b9](https://github.com/Augno/api/commit/59094b9146cab0e7cebde3967a085c6d993b9556))

## [0.15.6](https://github.com/Augno/api/compare/v0.15.5...v0.15.6) (2026-04-29)


### Bug Fixes

* updates to our items endpoints ([#226](https://github.com/Augno/api/issues/226)) ([4100ad6](https://github.com/Augno/api/commit/4100ad66cf5ed0043543a8342ce1ac84c3ccee73))

## [0.15.5](https://github.com/Augno/api/compare/v0.15.4...v0.15.5) (2026-04-29)


### Bug Fixes

* reject empty strings on patch requests ([#224](https://github.com/Augno/api/issues/224)) ([05aa9eb](https://github.com/Augno/api/commit/05aa9eb26c991e93a933dfb59d26c8207f09a6ea))

## [0.15.4](https://github.com/Augno/api/compare/v0.15.3...v0.15.4) (2026-04-29)


### Bug Fixes

* error with request logs and s3 objects ([#222](https://github.com/Augno/api/issues/222)) ([d088a71](https://github.com/Augno/api/commit/d088a7196c98a34c3fbd2b5646a3d2b9ce0330df))

## [0.15.3](https://github.com/Augno/api/compare/v0.15.2...v0.15.3) (2026-04-28)


### Bug Fixes

* update actor identifier references in request and audit log messages ([#220](https://github.com/Augno/api/issues/220)) ([a09add3](https://github.com/Augno/api/commit/a09add3c78cccf98cf8a3e958f91cc40a963b899))

## [0.15.2](https://github.com/Augno/api/compare/v0.15.1...v0.15.2) (2026-04-28)


### Bug Fixes

* audit events and request logs filters ([#217](https://github.com/Augno/api/issues/217)) ([a141e33](https://github.com/Augno/api/commit/a141e338a4bf31cf9cf10ddb4a6d74d508941e83))

## [0.15.1](https://github.com/Augno/api/compare/v0.15.0...v0.15.1) (2026-04-28)


### Bug Fixes

* multi-filters for request logs and audit logs ([#215](https://github.com/Augno/api/issues/215)) ([469cd46](https://github.com/Augno/api/commit/469cd4662e92349a64826e30e1a7b5c264700013))

## [0.15.0](https://github.com/Augno/api/compare/v0.14.4...v0.15.0) (2026-04-27)


### Features

* addresses can be filtered by drop ship status ([#213](https://github.com/Augno/api/issues/213)) ([cac21c3](https://github.com/Augno/api/commit/cac21c32413543484a85dc70b21762bfa94615e6))

## [0.14.4](https://github.com/Augno/api/compare/v0.14.3...v0.14.4) (2026-04-21)


### Bug Fixes

* business logic of products and sales orders ([#211](https://github.com/Augno/api/issues/211)) ([f05cf87](https://github.com/Augno/api/commit/f05cf8774b02fb85e35a1413b53720ec51a97c0a))

## [0.14.3](https://github.com/Augno/api/compare/v0.14.2...v0.14.3) (2026-04-21)


### Bug Fixes

* e2e tests that were failing ([#209](https://github.com/Augno/api/issues/209)) ([6d26d3b](https://github.com/Augno/api/commit/6d26d3b8c35d404209c800c97c097fd24e7c3e95))

## [0.14.2](https://github.com/Augno/api/compare/v0.14.1...v0.14.2) (2026-04-20)


### Bug Fixes

* bug in our carrier queries failing to include service levels on request ([#207](https://github.com/Augno/api/issues/207)) ([6e6ad1b](https://github.com/Augno/api/commit/6e6ad1b3669edfdcbfd48588f0152348318dbdbe))

## [0.14.1](https://github.com/Augno/api/compare/v0.14.0...v0.14.1) (2026-04-20)


### Bug Fixes

* duplicate task management in pods ([#205](https://github.com/Augno/api/issues/205)) ([4e33ed8](https://github.com/Augno/api/commit/4e33ed86e0eda139df500f265f83eb162efe4bb2))

## [0.14.0](https://github.com/Augno/api/compare/v0.13.4...v0.14.0) (2026-04-17)


### Features

* publish carrier/service level endpoints ([#203](https://github.com/Augno/api/issues/203)) ([99251a6](https://github.com/Augno/api/commit/99251a669031b9facabb97c53a83bf62b00a352e))

## [0.13.4](https://github.com/Augno/api/compare/v0.13.3...v0.13.4) (2026-04-16)


### Bug Fixes

* make some more endpoints skipped in the request logs ([#201](https://github.com/Augno/api/issues/201)) ([678f165](https://github.com/Augno/api/commit/678f16569658512e056ce5da7763aa206b14f604))

## [0.13.3](https://github.com/Augno/api/compare/v0.13.2...v0.13.3) (2026-04-16)


### Bug Fixes

* improve error handling on edge cases where a user doesn't have a password ([#199](https://github.com/Augno/api/issues/199)) ([33d30a5](https://github.com/Augno/api/commit/33d30a587ca694b3636f039a4c91e343db0c26d1))

## [0.13.2](https://github.com/Augno/api/compare/v0.13.1...v0.13.2) (2026-04-16)


### Bug Fixes

* add ignored routes for request logs ([#197](https://github.com/Augno/api/issues/197)) ([7c464c3](https://github.com/Augno/api/commit/7c464c30c79e081947c8b1cdfb866e9cccfadd5e))

## [0.13.1](https://github.com/Augno/api/compare/v0.13.0...v0.13.1) (2026-04-16)


### Bug Fixes

* bug in our openapi spec generation ([#194](https://github.com/Augno/api/issues/194)) ([513c363](https://github.com/Augno/api/commit/513c363c7701a8fe316e43c8845a6fc3917ade60))

## [0.13.0](https://github.com/Augno/api/compare/v0.12.1...v0.13.0) (2026-04-16)


### Features

* adding account user endpoints to the public release ([#192](https://github.com/Augno/api/issues/192)) ([d64c99b](https://github.com/Augno/api/commit/d64c99b504510cce05280ec3d3e7b7d1c89befdc))

## [0.12.1](https://github.com/Augno/api/compare/v0.12.0...v0.12.1) (2026-04-13)


### Bug Fixes

* bug in generation of openapi spec due to name collision ([#190](https://github.com/Augno/api/issues/190)) ([85cde4b](https://github.com/Augno/api/commit/85cde4b1940b635c92d6e05075040b524e9b0e0d))

## [0.12.0](https://github.com/Augno/api/compare/v0.11.2...v0.12.0) (2026-04-13)


### Features

* make account status endpoints publicly available ([#188](https://github.com/Augno/api/issues/188)) ([6d1c44b](https://github.com/Augno/api/commit/6d1c44b3157c6e00a3724f327e155d4e3b8e0d51))

## [0.11.2](https://github.com/Augno/api/compare/v0.11.1...v0.11.2) (2026-04-13)


### Bug Fixes

* various styling issues with a variety of the public endpoints ([#186](https://github.com/Augno/api/issues/186)) ([7cdbc10](https://github.com/Augno/api/commit/7cdbc102551e54ea45d4df8dd0461d3f935defd2))

## [0.11.1](https://github.com/Augno/api/compare/v0.11.0...v0.11.1) (2026-04-12)


### Bug Fixes

* cleanup public endpoints to better reflect our best practices and keep consistent conventions ([#183](https://github.com/Augno/api/issues/183)) ([c51492d](https://github.com/Augno/api/commit/c51492d51d5d6a15ddd342c7592f4ad13967982c))
* minor deviations from code conventions ([#185](https://github.com/Augno/api/issues/185)) ([a9b2899](https://github.com/Augno/api/commit/a9b2899c16c64a2b8bda67abc90b6f8c1b93bf86))

## [0.11.0](https://github.com/Augno/api/compare/v0.10.8...v0.11.0) (2026-04-09)


### Features

* add credit limit to customer object ([#181](https://github.com/Augno/api/issues/181)) ([eb748bc](https://github.com/Augno/api/commit/eb748bcaebcdf595303dddf9c4c2dfc77baa3064))

## [0.10.8](https://github.com/Augno/api/compare/v0.10.7...v0.10.8) (2026-04-09)


### Bug Fixes

* update Makefile targets and enhance E2E database setup ([#179](https://github.com/Augno/api/issues/179)) ([758eb19](https://github.com/Augno/api/commit/758eb19947d8947b2c0bdf54fc892e3642002567))

## [0.10.7](https://github.com/Augno/api/compare/v0.10.6...v0.10.7) (2026-04-08)


### Bug Fixes

* e2e test workflow ([#177](https://github.com/Augno/api/issues/177)) ([37ee21e](https://github.com/Augno/api/commit/37ee21eed1a5d89799f2f2d5679bd1a6c572b314))

## [0.10.6](https://github.com/Augno/api/compare/v0.10.5...v0.10.6) (2026-04-08)


### Bug Fixes

* complex joins even without includes ([#175](https://github.com/Augno/api/issues/175)) ([b1be209](https://github.com/Augno/api/commit/b1be2091774f40609c61764d74f4f11ed0aba6e1))

## [0.10.5](https://github.com/Augno/api/compare/v0.10.4...v0.10.5) (2026-04-08)


### Bug Fixes

* release issue for openapi specs ([#173](https://github.com/Augno/api/issues/173)) ([0a226af](https://github.com/Augno/api/commit/0a226af7c7c03e54b1531ac242e76cadf62e5cd5))

## [0.10.4](https://github.com/Augno/api/compare/v0.10.3...v0.10.4) (2026-04-08)


### Bug Fixes

* longer timeout values ([#171](https://github.com/Augno/api/issues/171)) ([3b3abea](https://github.com/Augno/api/commit/3b3abea2ea32e113fb126d1bc4a59a49634364ad))

## [0.10.3](https://github.com/Augno/api/compare/v0.10.2...v0.10.3) (2026-04-08)


### Bug Fixes

* ci issues ([#168](https://github.com/Augno/api/issues/168)) ([725bdd9](https://github.com/Augno/api/commit/725bdd96491c610fefb067a70f99085f41ee3818))
* update configuration and improve idempotency handling ([#170](https://github.com/Augno/api/issues/170)) ([a04ac67](https://github.com/Augno/api/commit/a04ac677a88e0155fe69b4aca91aee7ffe8046ac))

## [0.10.2](https://github.com/Augno/api/compare/v0.10.1...v0.10.2) (2026-04-08)


### Bug Fixes

* customer creation issues ([#166](https://github.com/Augno/api/issues/166)) ([ec4d6a0](https://github.com/Augno/api/commit/ec4d6a06956894e174363dd0b6b8c4cc03b0f521))

## [0.10.1](https://github.com/Augno/api/compare/v0.10.0...v0.10.1) (2026-04-07)


### Bug Fixes

* broken audit logs ([#164](https://github.com/Augno/api/issues/164)) ([a5dd64d](https://github.com/Augno/api/commit/a5dd64d56bf59defbcf4db6f14265aee05f95702))

## [0.10.0](https://github.com/Augno/api/compare/v0.9.10...v0.10.0) (2026-04-07)


### Features

* **auth:** add magic login functionality and account slug support ([#162](https://github.com/Augno/api/issues/162)) ([635b9d5](https://github.com/Augno/api/commit/635b9d5ba127cffd6ec0950014a3453851528e7b))

## [0.9.10](https://github.com/Augno/api/compare/v0.9.9...v0.9.10) (2026-04-07)


### Bug Fixes

* reset password issue ([#160](https://github.com/Augno/api/issues/160)) ([e6b7327](https://github.com/Augno/api/commit/e6b732713d77719d304f4eb52212b1ad32e192ea))

## [0.9.9](https://github.com/Augno/api/compare/v0.9.8...v0.9.9) (2026-04-07)


### Bug Fixes

* update parent account object type and add new fields in customer presenter ([#158](https://github.com/Augno/api/issues/158)) ([45d7d44](https://github.com/Augno/api/commit/45d7d448657be81108283ba8c7478b87a24b939a))

## [0.9.8](https://github.com/Augno/api/compare/v0.9.7...v0.9.8) (2026-04-07)


### Bug Fixes

* prettier formatting for curls ([#156](https://github.com/Augno/api/issues/156)) ([f6d8df7](https://github.com/Augno/api/commit/f6d8df7baf8341e89d6120c246454215931485ad))

## [0.9.7](https://github.com/Augno/api/compare/v0.9.6...v0.9.7) (2026-04-07)


### Bug Fixes

* auth flow bug ([#154](https://github.com/Augno/api/issues/154)) ([6071c6e](https://github.com/Augno/api/commit/6071c6e26de45d6553ff481bff66f543dcd50898))

## [0.9.6](https://github.com/Augno/api/compare/v0.9.5...v0.9.6) (2026-04-05)


### Bug Fixes

* auth bugs ([#152](https://github.com/Augno/api/issues/152)) ([b53e5b6](https://github.com/Augno/api/commit/b53e5b6dbab68f844b34062801547f44911e3381))

## [0.9.5](https://github.com/Augno/api/compare/v0.9.4...v0.9.5) (2026-04-05)


### Bug Fixes

* bug in auth flow ([#150](https://github.com/Augno/api/issues/150)) ([eacab29](https://github.com/Augno/api/commit/eacab29e5bf1cf41e312d460701d8731b7b6378c))

## [0.9.4](https://github.com/Augno/api/compare/v0.9.3...v0.9.4) (2026-04-05)


### Bug Fixes

* more permission tweaks for product lines and unit groups ([#148](https://github.com/Augno/api/issues/148)) ([e36268a](https://github.com/Augno/api/commit/e36268a92bae79ea2dd6706ffe4cec44c739b9a7))

## [0.9.3](https://github.com/Augno/api/compare/v0.9.2...v0.9.3) (2026-04-05)


### Bug Fixes

* enhance access checks in unit group service methods ([#146](https://github.com/Augno/api/issues/146)) ([e713af7](https://github.com/Augno/api/commit/e713af7304a6aa15783683d7bd2ffe561561431f))

## [0.9.2](https://github.com/Augno/api/compare/v0.9.1...v0.9.2) (2026-04-05)


### Bug Fixes

* update unit group service to enhance identity checks ([#144](https://github.com/Augno/api/issues/144)) ([5ab6eba](https://github.com/Augno/api/commit/5ab6eba81849e0da5190a9c349834962ea33ec9b))

## [0.9.1](https://github.com/Augno/api/compare/v0.9.0...v0.9.1) (2026-04-05)


### Bug Fixes

* unit group reads ([#142](https://github.com/Augno/api/issues/142)) ([e693532](https://github.com/Augno/api/commit/e693532a1987cf383e5e2de9d7967e0f570ac666))

## [0.9.0](https://github.com/Augno/api/compare/v0.8.12...v0.9.0) (2026-04-05)


### Features

* add is_publicly_visible field to account plans and update queries ([#140](https://github.com/Augno/api/issues/140)) ([47d7d66](https://github.com/Augno/api/commit/47d7d66131ce559daa4583747769da10f17c79ad))

## [0.8.12](https://github.com/Augno/api/compare/v0.8.11...v0.8.12) (2026-04-05)


### Bug Fixes

* deploy issues resolved - removing admin ([#138](https://github.com/Augno/api/issues/138)) ([223bee8](https://github.com/Augno/api/commit/223bee8cf9422ae362bcdeb492350c9e9bbe5786))

## [0.8.11](https://github.com/Augno/api/compare/v0.8.10...v0.8.11) (2026-04-05)


### Bug Fixes

* tf ([#136](https://github.com/Augno/api/issues/136)) ([7a45e51](https://github.com/Augno/api/commit/7a45e51e8358c8196bd998448447465735ad710d))

## [0.8.10](https://github.com/Augno/api/compare/v0.8.9...v0.8.10) (2026-04-05)


### Bug Fixes

* working on terraform issues ([#134](https://github.com/Augno/api/issues/134)) ([53e0b56](https://github.com/Augno/api/commit/53e0b568f871ad134c098697e7c8e899664312c7))

## [0.8.9](https://github.com/Augno/api/compare/v0.8.8...v0.8.9) (2026-04-05)


### Bug Fixes

* terraform rollout update ([#132](https://github.com/Augno/api/issues/132)) ([e4d2f88](https://github.com/Augno/api/commit/e4d2f88c5646aa0204f2690628b283a192c4a634))

## [0.8.8](https://github.com/Augno/api/compare/v0.8.7...v0.8.8) (2026-04-05)


### Bug Fixes

* avoid ip address bottleneck on deploy ([#130](https://github.com/Augno/api/issues/130)) ([5d27ca7](https://github.com/Augno/api/commit/5d27ca716f23e6ef13fa92e0c29a4a865a649ec2))

## [0.8.7](https://github.com/Augno/api/compare/v0.8.6...v0.8.7) (2026-04-05)


### Bug Fixes

* format ([#128](https://github.com/Augno/api/issues/128)) ([1f1a9cb](https://github.com/Augno/api/commit/1f1a9cbfee442313dda87f294b4debebce87f027))

## [0.8.6](https://github.com/Augno/api/compare/v0.8.5...v0.8.6) (2026-04-05)


### Bug Fixes

* ses terraform issue ([#126](https://github.com/Augno/api/issues/126)) ([97a0233](https://github.com/Augno/api/commit/97a0233ebac07ace0d59174857f1c4ef31c86750))

## [0.8.5](https://github.com/Augno/api/compare/v0.8.4...v0.8.5) (2026-04-05)


### Bug Fixes

* race issue on terraform ([#124](https://github.com/Augno/api/issues/124)) ([99b67ae](https://github.com/Augno/api/commit/99b67ae76e48677b5ac497956d9792364184d91b))

## [0.8.4](https://github.com/Augno/api/compare/v0.8.3...v0.8.4) (2026-04-05)


### Bug Fixes

* IAM issue on terraform ([#122](https://github.com/Augno/api/issues/122)) ([ce8e4d4](https://github.com/Augno/api/commit/ce8e4d4e071b30b78b1a60e222958be5016a8773))

## [0.8.3](https://github.com/Augno/api/compare/v0.8.2...v0.8.3) (2026-04-05)


### Bug Fixes

* add private link to planetscale for pg ([#120](https://github.com/Augno/api/issues/120)) ([01e8de9](https://github.com/Augno/api/commit/01e8de973d2be18ec94bb56e07e238241c31a5d3))

## [0.8.2](https://github.com/Augno/api/compare/v0.8.1...v0.8.2) (2026-04-05)


### Bug Fixes

* region for ses resources ([#118](https://github.com/Augno/api/issues/118)) ([b10f579](https://github.com/Augno/api/commit/b10f579d7000381c716cc5a116d76f8fcb5f1f32))

## [0.8.1](https://github.com/Augno/api/compare/v0.8.0...v0.8.1) (2026-04-05)


### Bug Fixes

* terraform issues ([#116](https://github.com/Augno/api/issues/116)) ([86de757](https://github.com/Augno/api/commit/86de757dff2647681b1c8b46b9b8cce94c543a7d))

## [0.8.0](https://github.com/Augno/api/compare/v0.7.0...v0.8.0) (2026-04-05)


### Features

* adding new unit endpoints for CRUD operations ([#100](https://github.com/Augno/api/issues/100)) ([dae52fb](https://github.com/Augno/api/commit/dae52fb1fae3d348721b16c732d7c2b7aeeeb76c))
* agents ([#113](https://github.com/Augno/api/issues/113)) ([4d3d90c](https://github.com/Augno/api/commit/4d3d90cebbbf11241283249a29a02ec1dbce983a))
* api key endpoints ([#76](https://github.com/Augno/api/issues/76)) ([87bb039](https://github.com/Augno/api/commit/87bb039b22cc4d74982681aa2cde8a266d612e50))
* api keys, sandboxes, units, request logs ([#83](https://github.com/Augno/api/issues/83)) ([740aefb](https://github.com/Augno/api/commit/740aefb306b6087123eee9ebd224e276e54f0053))
* idempotent endpoints ([#60](https://github.com/Augno/api/issues/60)) ([4d7eedc](https://github.com/Augno/api/commit/4d7eedc53820aa60bee42e22f46edfcc59dd713f))
* include values ([#107](https://github.com/Augno/api/issues/107)) ([7428518](https://github.com/Augno/api/commit/7428518682165a40111c459f4b3f717bb00ad60c))


### Bug Fixes

* add admin email notification on registration and plan changes ([#88](https://github.com/Augno/api/issues/88)) ([d4a73eb](https://github.com/Augno/api/commit/d4a73ebcbe3b5194881c299932ae387afcf324e1))
* add alerts to registration limits getting hit ([#90](https://github.com/Augno/api/issues/90)) ([699f13b](https://github.com/Augno/api/commit/699f13b85cddd97027d5e9304b7fb684b6d6c27c))
* add metrics server to eks ([#59](https://github.com/Augno/api/issues/59)) ([ece326a](https://github.com/Augno/api/commit/ece326a2ecced55619fbe55be214d0aec395b0eb))
* add target account ID to request logs ([#51](https://github.com/Augno/api/issues/51)) ([23e125e](https://github.com/Augno/api/commit/23e125e77743c55d9503d2987cb237998db76f20))
* adding env scope to release workflow ([#103](https://github.com/Augno/api/issues/103)) ([90e5168](https://github.com/Augno/api/commit/90e5168338bfc061da03f46833af5735bab7760c))
* adding openapi spec gen to release process ([#105](https://github.com/Augno/api/issues/105)) ([c6c599a](https://github.com/Augno/api/commit/c6c599a2883b6ef1cd316b551b79332594f9f0c1))
* build issues ([#77](https://github.com/Augno/api/issues/77)) ([91bee52](https://github.com/Augno/api/commit/91bee522bfb560c3bc29653868bc18ca6b9d784c))
* cd ([#45](https://github.com/Augno/api/issues/45)) ([903b3d0](https://github.com/Augno/api/commit/903b3d0d1dc57336a9b8fc42a2b001fbf3db0029))
* CD tagging of releases ([#64](https://github.com/Augno/api/issues/64)) ([55a78c5](https://github.com/Augno/api/commit/55a78c52d680d5f0b2c9b294ba0bd55afab6e647))
* combine release process with cd ([#39](https://github.com/Augno/api/issues/39)) ([38e0e1f](https://github.com/Augno/api/commit/38e0e1f42e12c761efaaa09cee60d5a051f13e39))
* cookies misconfiguration ([#53](https://github.com/Augno/api/issues/53)) ([bb66380](https://github.com/Augno/api/commit/bb6638072666f2681445c2fb62292f6b24c0d329))
* correct password reset URL path in email link ([#111](https://github.com/Augno/api/issues/111)) ([5b30f7a](https://github.com/Augno/api/commit/5b30f7a0fc8c6319ca6334b67849b366b303f0a1))
* cors ([#47](https://github.com/Augno/api/issues/47)) ([2bde99b](https://github.com/Augno/api/commit/2bde99bfc4ad1e2e169f5afde4fb6897125a772f))
* deploy logic gates ([#41](https://github.com/Augno/api/issues/41)) ([32733b9](https://github.com/Augno/api/commit/32733b940615c06724ce96fa13c0460e7076a76b))
* enforce TLS and improve healthz performance ([#31](https://github.com/Augno/api/issues/31)) ([be01b2d](https://github.com/Augno/api/commit/be01b2dd178c70e0aa774e767585a6a17ed220dc))
* force delete terraform ([#66](https://github.com/Augno/api/issues/66)) ([ea6aba1](https://github.com/Augno/api/commit/ea6aba19a531d2d6be92777fbc879926f824df86))
* improve ci/cd perf ([#34](https://github.com/Augno/api/issues/34)) ([19c6a02](https://github.com/Augno/api/commit/19c6a020ebc1af57f8d9f9e6406aa87613e83c79))
* issue preventing cd trigger on release ([#35](https://github.com/Augno/api/issues/35)) ([608845d](https://github.com/Augno/api/commit/608845dea4d404af4e433992c866b2d5bc28f57f))
* logic fix for api key parse, proper error handling ([#74](https://github.com/Augno/api/issues/74)) ([99c704b](https://github.com/Augno/api/commit/99c704b0421e6f860987ea310f8c2fb7b9dddf75))
* no emails on dev 500s ([#98](https://github.com/Augno/api/issues/98)) ([6ab719e](https://github.com/Augno/api/commit/6ab719ee975af2ad95b148e3691f4039e6439935))
* openapi spec cd ([#49](https://github.com/Augno/api/issues/49)) ([40114c4](https://github.com/Augno/api/commit/40114c48c065a6fd04d2e2342588115808c1f6c9))
* openapi spec gen on CD ([#43](https://github.com/Augno/api/issues/43)) ([f87c4af](https://github.com/Augno/api/commit/f87c4af9a72b34ed7452f53017c75b1bfbf0374d))
* registration limit checks ([#94](https://github.com/Augno/api/issues/94)) ([22b7a73](https://github.com/Augno/api/commit/22b7a733d6997ed619604a7bea1cb124e7666394))
* registration limits not accounting for period of plan ([#96](https://github.com/Augno/api/issues/96)) ([ff80a6a](https://github.com/Augno/api/commit/ff80a6a201fb26192973c8b62ffbdac898b3312e))
* registration process bug ([#92](https://github.com/Augno/api/issues/92)) ([73e32a4](https://github.com/Augno/api/commit/73e32a45de4629840babeaa9fc7fa34d00bccb72))
* remove account_id from sandbox envelope ([#109](https://github.com/Augno/api/issues/109)) ([dab2a66](https://github.com/Augno/api/commit/dab2a66436ba30fa1c8f2f97573d203e120d63d1))
* remove options from telemetry ([#55](https://github.com/Augno/api/issues/55)) ([dad7369](https://github.com/Augno/api/commit/dad73691731098e764a6d19a59dc5f7fb6378a47))
* remove stainless push for now ([#101](https://github.com/Augno/api/issues/101)) ([f19b6b9](https://github.com/Augno/api/commit/f19b6b973bcd64b6e12a79aeb807be944b53e8df))
* remove terraform force delete ([#68](https://github.com/Augno/api/issues/68)) ([ca88e90](https://github.com/Augno/api/commit/ca88e908b7de86752e6b3a561dc8d2c69269aca8))
* remove unused config ([#70](https://github.com/Augno/api/issues/70)) ([d61202b](https://github.com/Augno/api/commit/d61202bf1b8028d20e0a154bdb693413a5e56c48))
* request log query memory issues ([#84](https://github.com/Augno/api/issues/84)) ([41c2ea9](https://github.com/Augno/api/commit/41c2ea912dc6e01fd30b6d5f634084441eda0a5c))
* seeded account and sandbox data ([#86](https://github.com/Augno/api/issues/86)) ([b27a9cf](https://github.com/Augno/api/commit/b27a9cffe9324f6d6279c5b872b72d1b6d6864df))
* subdomain allowed on cookies ([#79](https://github.com/Augno/api/issues/79)) ([5474bfd](https://github.com/Augno/api/commit/5474bfda4d219329c8147f6305299f26787d567b))
* tagging ([#72](https://github.com/Augno/api/issues/72)) ([b4c2349](https://github.com/Augno/api/commit/b4c2349554eb2893e0cc6bb1b5f5f8f81f825e59))
* tests ([#81](https://github.com/Augno/api/issues/81)) ([8337e0a](https://github.com/Augno/api/commit/8337e0a1292c94038d610e7e53aaa448f06a7750))
* throw errors if assumptions fail in spec generation ([#57](https://github.com/Augno/api/issues/57)) ([eb75a6e](https://github.com/Augno/api/commit/eb75a6ee5c3d5870aeda09ee70181bdfc7b9ff59))
* trigger cd on new tag creation ([#37](https://github.com/Augno/api/issues/37)) ([bf88b38](https://github.com/Augno/api/commit/bf88b38c89189806ef034d6fc36cdafc5f71b8b4))
* unique rabbitmq credentials enforced ([#33](https://github.com/Augno/api/issues/33)) ([5b75f44](https://github.com/Augno/api/commit/5b75f447b23e18ba9291aa275f41060ec66652fa))

## [0.7.0](https://github.com/Augno/api/compare/v0.6.2...v0.7.0) (2026-04-05)


### Features

* agents ([#113](https://github.com/Augno/api/issues/113)) ([4d3d90c](https://github.com/Augno/api/commit/4d3d90cebbbf11241283249a29a02ec1dbce983a))

## [0.6.2](https://github.com/Augno/api/compare/v0.6.1...v0.6.2) (2026-03-31)


### Bug Fixes

* correct password reset URL path in email link ([#111](https://github.com/Augno/api/issues/111)) ([5b30f7a](https://github.com/Augno/api/commit/5b30f7a0fc8c6319ca6334b67849b366b303f0a1))

## [0.6.1](https://github.com/Augno/api/compare/v0.6.0...v0.6.1) (2026-03-04)


### Bug Fixes

* remove account_id from sandbox envelope ([#109](https://github.com/Augno/api/issues/109)) ([dab2a66](https://github.com/Augno/api/commit/dab2a66436ba30fa1c8f2f97573d203e120d63d1))

## [0.6.0](https://github.com/Augno/api/compare/v0.5.2...v0.6.0) (2026-03-04)


### Features

* include values ([#107](https://github.com/Augno/api/issues/107)) ([7428518](https://github.com/Augno/api/commit/7428518682165a40111c459f4b3f717bb00ad60c))


### Bug Fixes

* adding openapi spec gen to release process ([#105](https://github.com/Augno/api/issues/105)) ([c6c599a](https://github.com/Augno/api/commit/c6c599a2883b6ef1cd316b551b79332594f9f0c1))

## [0.5.2](https://github.com/Augno/api/compare/v0.5.1...v0.5.2) (2026-03-03)


### Bug Fixes

* adding env scope to release workflow ([#103](https://github.com/Augno/api/issues/103)) ([90e5168](https://github.com/Augno/api/commit/90e5168338bfc061da03f46833af5735bab7760c))

## [0.5.1](https://github.com/Augno/api/compare/v0.5.0...v0.5.1) (2026-03-03)


### Bug Fixes

* remove stainless push for now ([#101](https://github.com/Augno/api/issues/101)) ([f19b6b9](https://github.com/Augno/api/commit/f19b6b973bcd64b6e12a79aeb807be944b53e8df))

## [0.5.0](https://github.com/Augno/api/compare/v0.4.7...v0.5.0) (2026-03-03)


### Features

* adding new unit endpoints for CRUD operations ([#100](https://github.com/Augno/api/issues/100)) ([dae52fb](https://github.com/Augno/api/commit/dae52fb1fae3d348721b16c732d7c2b7aeeeb76c))


### Bug Fixes

* no emails on dev 500s ([#98](https://github.com/Augno/api/issues/98)) ([6ab719e](https://github.com/Augno/api/commit/6ab719ee975af2ad95b148e3691f4039e6439935))

## [0.4.7](https://github.com/Augno/api/compare/v0.4.6...v0.4.7) (2026-03-02)


### Bug Fixes

* registration limits not accounting for period of plan ([#96](https://github.com/Augno/api/issues/96)) ([ff80a6a](https://github.com/Augno/api/commit/ff80a6a201fb26192973c8b62ffbdac898b3312e))

## [0.4.6](https://github.com/Augno/api/compare/v0.4.5...v0.4.6) (2026-03-02)


### Bug Fixes

* registration limit checks ([#94](https://github.com/Augno/api/issues/94)) ([22b7a73](https://github.com/Augno/api/commit/22b7a733d6997ed619604a7bea1cb124e7666394))

## [0.4.5](https://github.com/Augno/api/compare/v0.4.4...v0.4.5) (2026-03-02)


### Bug Fixes

* registration process bug ([#92](https://github.com/Augno/api/issues/92)) ([73e32a4](https://github.com/Augno/api/commit/73e32a45de4629840babeaa9fc7fa34d00bccb72))

## [0.4.4](https://github.com/Augno/api/compare/v0.4.3...v0.4.4) (2026-02-27)


### Bug Fixes

* add alerts to registration limits getting hit ([#90](https://github.com/Augno/api/issues/90)) ([699f13b](https://github.com/Augno/api/commit/699f13b85cddd97027d5e9304b7fb684b6d6c27c))

## [0.4.3](https://github.com/Augno/api/compare/v0.4.2...v0.4.3) (2026-02-27)


### Bug Fixes

* add admin email notification on registration and plan changes ([#88](https://github.com/Augno/api/issues/88)) ([d4a73eb](https://github.com/Augno/api/commit/d4a73ebcbe3b5194881c299932ae387afcf324e1))

## [0.4.2](https://github.com/Augno/api/compare/v0.4.1...v0.4.2) (2026-02-27)


### Bug Fixes

* seeded account and sandbox data ([#86](https://github.com/Augno/api/issues/86)) ([b27a9cf](https://github.com/Augno/api/commit/b27a9cffe9324f6d6279c5b872b72d1b6d6864df))

## [0.4.1](https://github.com/Augno/api/compare/v0.4.0...v0.4.1) (2026-02-27)


### Bug Fixes

* request log query memory issues ([#84](https://github.com/Augno/api/issues/84)) ([41c2ea9](https://github.com/Augno/api/commit/41c2ea912dc6e01fd30b6d5f634084441eda0a5c))

## [0.4.0](https://github.com/Augno/api/compare/v0.3.2...v0.4.0) (2026-02-27)


### Features

* api keys, sandboxes, units, request logs ([#83](https://github.com/Augno/api/issues/83)) ([740aefb](https://github.com/Augno/api/commit/740aefb306b6087123eee9ebd224e276e54f0053))


### Bug Fixes

* tests ([#81](https://github.com/Augno/api/issues/81)) ([8337e0a](https://github.com/Augno/api/commit/8337e0a1292c94038d610e7e53aaa448f06a7750))

## [0.3.2](https://github.com/Augno/api/compare/v0.3.1...v0.3.2) (2026-02-23)


### Bug Fixes

* subdomain allowed on cookies ([#79](https://github.com/Augno/api/issues/79)) ([5474bfd](https://github.com/Augno/api/commit/5474bfda4d219329c8147f6305299f26787d567b))

## [0.3.1](https://github.com/Augno/api/compare/v0.3.0...v0.3.1) (2026-02-13)


### Bug Fixes

* build issues ([#77](https://github.com/Augno/api/issues/77)) ([91bee52](https://github.com/Augno/api/commit/91bee522bfb560c3bc29653868bc18ca6b9d784c))

## [0.3.0](https://github.com/Augno/api/compare/v0.2.4...v0.3.0) (2026-02-13)


### Features

* api key endpoints ([#76](https://github.com/Augno/api/issues/76)) ([87bb039](https://github.com/Augno/api/commit/87bb039b22cc4d74982681aa2cde8a266d612e50))


### Bug Fixes

* logic fix for api key parse, proper error handling ([#74](https://github.com/Augno/api/issues/74)) ([99c704b](https://github.com/Augno/api/commit/99c704b0421e6f860987ea310f8c2fb7b9dddf75))

## [0.2.4](https://github.com/Augno/api/compare/v0.2.3...v0.2.4) (2026-02-12)


### Bug Fixes

* tagging ([#72](https://github.com/Augno/api/issues/72)) ([b4c2349](https://github.com/Augno/api/commit/b4c2349554eb2893e0cc6bb1b5f5f8f81f825e59))

## [0.2.3](https://github.com/Augno/api/compare/v0.2.2...v0.2.3) (2026-02-11)


### Bug Fixes

* remove unused config ([#70](https://github.com/Augno/api/issues/70)) ([d61202b](https://github.com/Augno/api/commit/d61202bf1b8028d20e0a154bdb693413a5e56c48))

## [0.2.2](https://github.com/Augno/api/compare/v0.2.1...v0.2.2) (2026-02-11)


### Bug Fixes

* remove terraform force delete ([#68](https://github.com/Augno/api/issues/68)) ([ca88e90](https://github.com/Augno/api/commit/ca88e908b7de86752e6b3a561dc8d2c69269aca8))

## [0.2.1](https://github.com/Augno/api/compare/v0.2.0...v0.2.1) (2026-02-11)


### Bug Fixes

* force delete terraform ([#66](https://github.com/Augno/api/issues/66)) ([ea6aba1](https://github.com/Augno/api/commit/ea6aba19a531d2d6be92777fbc879926f824df86))

## [0.2.0](https://github.com/Augno/api/compare/v0.1.13...v0.2.0) (2026-02-11)


### Features

* idempotent endpoints ([#60](https://github.com/Augno/api/issues/60)) ([4d7eedc](https://github.com/Augno/api/commit/4d7eedc53820aa60bee42e22f46edfcc59dd713f))


### Bug Fixes

* CD tagging of releases ([#64](https://github.com/Augno/api/issues/64)) ([55a78c5](https://github.com/Augno/api/commit/55a78c52d680d5f0b2c9b294ba0bd55afab6e647))

## [1.0.forge-preview.1]() (2026-02-11)

### Features

* idempotent endpoints (#60) ([4d7eedc](https://github.com/Augno/api/commit/4d7eedc53820aa60bee42e22f46edfcc59dd713f))

### Bug Fixes

* add metrics server to eks (#59) ([ece326a](https://github.com/Augno/api/commit/ece326a2ecced55619fbe55be214d0aec395b0eb))
* throw errors if assumptions fail in spec generation (#57) ([eb75a6e](https://github.com/Augno/api/commit/eb75a6ee5c3d5870aeda09ee70181bdfc7b9ff59))
* remove options from telemetry (#55) ([dad7369](https://github.com/Augno/api/commit/dad73691731098e764a6d19a59dc5f7fb6378a47))
* cookies misconfiguration (#53) ([bb66380](https://github.com/Augno/api/commit/bb6638072666f2681445c2fb62292f6b24c0d329))
* add target account ID to request logs (#51) ([23e125e](https://github.com/Augno/api/commit/23e125e77743c55d9503d2987cb237998db76f20))
* openapi spec cd (#49) ([40114c4](https://github.com/Augno/api/commit/40114c48c065a6fd04d2e2342588115808c1f6c9))
* cors (#47) ([2bde99b](https://github.com/Augno/api/commit/2bde99bfc4ad1e2e169f5afde4fb6897125a772f))
* cd (#45) ([903b3d0](https://github.com/Augno/api/commit/903b3d0d1dc57336a9b8fc42a2b001fbf3db0029))
* openapi spec gen on CD (#43) ([f87c4af](https://github.com/Augno/api/commit/f87c4af9a72b34ed7452f53017c75b1bfbf0374d))
* deploy logic gates (#41) ([32733b9](https://github.com/Augno/api/commit/32733b940615c06724ce96fa13c0460e7076a76b))
* combine release process with cd (#39) ([38e0e1f](https://github.com/Augno/api/commit/38e0e1f42e12c761efaaa09cee60d5a051f13e39))
* trigger cd on new tag creation (#37) ([bf88b38](https://github.com/Augno/api/commit/bf88b38c89189806ef034d6fc36cdafc5f71b8b4))
* issue preventing cd trigger on release (#35) ([608845d](https://github.com/Augno/api/commit/608845dea4d404af4e433992c866b2d5bc28f57f))
* improve ci/cd perf (#34) ([19c6a02](https://github.com/Augno/api/commit/19c6a020ebc1af57f8d9f9e6406aa87613e83c79))
* unique rabbitmq credentials enforced (#33) ([5b75f44](https://github.com/Augno/api/commit/5b75f447b23e18ba9291aa275f41060ec66652fa))
* enforce TLS and improve healthz performance (#31) ([be01b2d](https://github.com/Augno/api/commit/be01b2dd178c70e0aa774e767585a6a17ed220dc))

## [1.0.forge] - Migration Release

This release migrates from SemVer (`v0.1.x`) to the new codename-based versioning scheme.
All functionality from `v0.1.13` is included.

---

## Legacy Releases (SemVer)

### [0.1.13](https://github.com/Augno/api/compare/v0.1.12...v0.1.13) (2025-12-23)


### Bug Fixes

* add metrics server to eks ([#59](https://github.com/Augno/api/issues/59)) ([ece326a](https://github.com/Augno/api/commit/ece326a2ecced55619fbe55be214d0aec395b0eb))
* throw errors if assumptions fail in spec generation ([#57](https://github.com/Augno/api/issues/57)) ([eb75a6e](https://github.com/Augno/api/commit/eb75a6ee5c3d5870aeda09ee70181bdfc7b9ff59))

## [0.1.12](https://github.com/Augno/api/compare/v0.1.11...v0.1.12) (2025-12-22)


### Bug Fixes

* remove options from telemetry ([#55](https://github.com/Augno/api/issues/55)) ([dad7369](https://github.com/Augno/api/commit/dad73691731098e764a6d19a59dc5f7fb6378a47))

## [0.1.11](https://github.com/Augno/api/compare/v0.1.10...v0.1.11) (2025-12-22)


### Bug Fixes

* cookies misconfiguration ([#53](https://github.com/Augno/api/issues/53)) ([bb66380](https://github.com/Augno/api/commit/bb6638072666f2681445c2fb62292f6b24c0d329))

## [0.1.10](https://github.com/Augno/api/compare/v0.1.9...v0.1.10) (2025-12-22)


### Bug Fixes

* add target account ID to request logs ([#51](https://github.com/Augno/api/issues/51)) ([23e125e](https://github.com/Augno/api/commit/23e125e77743c55d9503d2987cb237998db76f20))

## [0.1.9](https://github.com/Augno/api/compare/v0.1.8...v0.1.9) (2025-12-22)


### Bug Fixes

* openapi spec cd ([#49](https://github.com/Augno/api/issues/49)) ([40114c4](https://github.com/Augno/api/commit/40114c48c065a6fd04d2e2342588115808c1f6c9))

## [0.1.8](https://github.com/Augno/api/compare/v0.1.7...v0.1.8) (2025-12-20)


### Bug Fixes

* cors ([#47](https://github.com/Augno/api/issues/47)) ([2bde99b](https://github.com/Augno/api/commit/2bde99bfc4ad1e2e169f5afde4fb6897125a772f))

## [0.1.7](https://github.com/Augno/api/compare/v0.1.6...v0.1.7) (2025-12-20)


### Bug Fixes

* cd ([#45](https://github.com/Augno/api/issues/45)) ([903b3d0](https://github.com/Augno/api/commit/903b3d0d1dc57336a9b8fc42a2b001fbf3db0029))

## [0.1.6](https://github.com/Augno/api/compare/v0.1.5...v0.1.6) (2025-12-20)


### Bug Fixes

* openapi spec gen on CD ([#43](https://github.com/Augno/api/issues/43)) ([f87c4af](https://github.com/Augno/api/commit/f87c4af9a72b34ed7452f53017c75b1bfbf0374d))

## [0.1.5](https://github.com/Augno/api/compare/v0.1.4...v0.1.5) (2025-12-19)


### Bug Fixes

* deploy logic gates ([#41](https://github.com/Augno/api/issues/41)) ([32733b9](https://github.com/Augno/api/commit/32733b940615c06724ce96fa13c0460e7076a76b))

## [0.1.4](https://github.com/Augno/api/compare/v0.1.3...v0.1.4) (2025-12-19)


### Bug Fixes

* combine release process with cd ([#39](https://github.com/Augno/api/issues/39)) ([38e0e1f](https://github.com/Augno/api/commit/38e0e1f42e12c761efaaa09cee60d5a051f13e39))

## [0.1.3](https://github.com/Augno/api/compare/v0.1.2...v0.1.3) (2025-12-19)


### Bug Fixes

* trigger cd on new tag creation ([#37](https://github.com/Augno/api/issues/37)) ([bf88b38](https://github.com/Augno/api/commit/bf88b38c89189806ef034d6fc36cdafc5f71b8b4))

## [0.1.2](https://github.com/Augno/api/compare/v0.1.1...v0.1.2) (2025-12-19)


### Bug Fixes

* issue preventing cd trigger on release ([#35](https://github.com/Augno/api/issues/35)) ([608845d](https://github.com/Augno/api/commit/608845dea4d404af4e433992c866b2d5bc28f57f))

## [0.1.1](https://github.com/Augno/api/compare/v0.1.0...v0.1.1) (2025-12-19)


### Bug Fixes

* enforce TLS and improve healthz performance ([#31](https://github.com/Augno/api/issues/31)) ([be01b2d](https://github.com/Augno/api/commit/be01b2dd178c70e0aa774e767585a6a17ed220dc))
* improve ci/cd perf ([#34](https://github.com/Augno/api/issues/34)) ([19c6a02](https://github.com/Augno/api/commit/19c6a020ebc1af57f8d9f9e6406aa87613e83c79))
* unique rabbitmq credentials enforced ([#33](https://github.com/Augno/api/issues/33)) ([5b75f44](https://github.com/Augno/api/commit/5b75f447b23e18ba9291aa275f41060ec66652fa))

All notable changes to this project will be documented in this file.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
