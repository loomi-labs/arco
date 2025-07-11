# Changelog

## [0.14.0](https://github.com/loomi-labs/arco/compare/v0.13.1...v0.14.0) (2025-07-11)


### Features

* add arco-cloud integration (beta) ([#159](https://github.com/loomi-labs/arco/issues/159)) ([c5ae6d3](https://github.com/loomi-labs/arco/commit/c5ae6d332fb923ef75254b1965a1eddb866ca38a))
* add auth service for cloud login ([#157](https://github.com/loomi-labs/arco/issues/157)) ([ed75de1](https://github.com/loomi-labs/arco/commit/ed75de1a33b606d46fac8432c25ec8c33963216d))
* add Borg 1.4.1 support ([#168](https://github.com/loomi-labs/arco/issues/168)) ([4ec2b35](https://github.com/loomi-labs/arco/commit/4ec2b350973cc80382436c3f8052cd8cacee335b))


### Bug Fixes

* borg locks ([#163](https://github.com/loomi-labs/arco/issues/163)) ([5ca25f4](https://github.com/loomi-labs/arco/commit/5ca25f46e111a9f0635a3ae33646ddc43dddff6a))
* enable macOS universal build support in task system ([#162](https://github.com/loomi-labs/arco/issues/162)) ([64a3df0](https://github.com/loomi-labs/arco/commit/64a3df000cec9d0068b7819a9f2390b700db29ca))
* cancel operations ([#167](https://github.com/loomi-labs/arco/issues/167)) ([3e9278b](https://github.com/loomi-labs/arco/commit/3e9278bd1f6bf640b2e46f6e8e9fd88f7e0baa3a))

## [0.13.1](https://github.com/loomi-labs/arco/compare/v0.13.0...v0.13.1) (2025-05-29)


### Bug Fixes

* Show last backup only for current backup profile ([#153](https://github.com/loomi-labs/arco/issues/153)) ([7489f0d](https://github.com/loomi-labs/arco/commit/7489f0db3bb093567d8e27320c9f4072094c4f1b))

## [0.13.0](https://github.com/loomi-labs/arco/compare/v0.12.2...v0.13.0) (2025-03-13)


### Features

* add footer ([#145](https://github.com/loomi-labs/arco/issues/145)) ([2c7bf04](https://github.com/loomi-labs/arco/commit/2c7bf04681a5aaf6ed21c899b058555173dece55))
* improve UI ([b934661](https://github.com/loomi-labs/arco/commit/b934661a8f523b1649a4c701417a4e5c58b43a54))
* make data/schedule sections collapsible ([e0fba4a](https://github.com/loomi-labs/arco/commit/e0fba4a89ad8c564a5bc8a57dad4e1f49f3708b5))
* upgrade to wails v3 ([#146](https://github.com/loomi-labs/arco/issues/146)) ([5d5f459](https://github.com/loomi-labs/arco/commit/5d5f459b55ea4db6ef2ac10a4ed5ada969c542f4))


### Bug Fixes

* layout glitch of pruning/deletion ([496f779](https://github.com/loomi-labs/arco/commit/496f779cbe61b3f04c4173509df401ede5cd68d4))
* logging of create/prune command ([f8e1def](https://github.com/loomi-labs/arco/commit/f8e1defea57fd41b0a86c694c71fe46c38e64023))
* logging of create/prune command ([4444957](https://github.com/loomi-labs/arco/commit/444495768ce43a1bcf003fc8c3b85a06596c6852))
* show correct time for archives ([5e96eb3](https://github.com/loomi-labs/arco/commit/5e96eb341e85406f72e541004b548542326af632))

## [0.12.2](https://github.com/loomi-labs/arco/compare/v0.12.1...v0.12.2) (2025-01-16)


### Bug Fixes

* memory usage ([#137](https://github.com/loomi-labs/arco/issues/137)) ([eff770b](https://github.com/loomi-labs/arco/commit/eff770b31ef1d410e424b62746e293b8ede2e2f5))

## [0.12.1](https://github.com/loomi-labs/arco/compare/v0.12.0...v0.12.1) (2025-01-01)


### Bug Fixes

* eliminate race conditions in state.go ([74d7a93](https://github.com/loomi-labs/arco/commit/74d7a93d526d1712704943fc9635a3a07f686647))
* linux CI/CD build ([#133](https://github.com/loomi-labs/arco/issues/133)) ([80879e0](https://github.com/loomi-labs/arco/commit/80879e029383fe1d86e9e40b6c6a77a049fe4fb6))
* logging for migrations ([ce076a3](https://github.com/loomi-labs/arco/commit/ce076a3996e38da37fc69436edeefde7ecccb8bc))

## [0.12.0](https://github.com/loomi-labs/arco/compare/v0.11.5...v0.12.0) (2024-12-11)


### Features

* add goose migrations ([#130](https://github.com/loomi-labs/arco/issues/130)) ([dd6208d](https://github.com/loomi-labs/arco/commit/dd6208d6788c2086087689638224a295376ec98a))
* build with webkit2gtk 4.1 ([#132](https://github.com/loomi-labs/arco/issues/132)) ([009a267](https://github.com/loomi-labs/arco/commit/009a26762bc289190b220fccbeff62a7eff7f0fb))

## [0.11.5](https://github.com/loomi-labs/arco/compare/v0.11.4...v0.11.5) (2024-12-02)


### Bug Fixes

* add default settings ([#127](https://github.com/loomi-labs/arco/issues/127)) ([6b9a209](https://github.com/loomi-labs/arco/commit/6b9a2097ee5b4193cdc33457b3e3640b963a2656))

## [0.11.4](https://github.com/loomi-labs/arco/compare/v0.11.3...v0.11.4) (2024-12-02)


### Bug Fixes

* logger path ([9476dad](https://github.com/loomi-labs/arco/commit/9476dadb4e6b1b7caeb05b99e217cf6eaf529b68))

## [0.11.3](https://github.com/loomi-labs/arco/compare/v0.11.2...v0.11.3) (2024-12-02)


### Bug Fixes

* macos build ([#120](https://github.com/loomi-labs/arco/issues/120)) ([0139a93](https://github.com/loomi-labs/arco/commit/0139a93def8872dc00816f365dd184e9c44303e4))

## [0.11.2](https://github.com/loomi-labs/arco/compare/v0.11.1...v0.11.2) (2024-12-02)


### Bug Fixes

* CI/CD ([3ad1ddc](https://github.com/loomi-labs/arco/commit/3ad1ddc5fe75901acfb9c75fd660082508f3877c))
* CI/CD ([0586d4a](https://github.com/loomi-labs/arco/commit/0586d4a0bdceff76fca06ed57c4853b998548a06))

## [0.11.1](https://github.com/loomi-labs/arco/compare/v0.11.0...v0.11.1) (2024-12-02)


### Bug Fixes

* parsing error when having a warning in `info` ([06fc3b4](https://github.com/loomi-labs/arco/commit/06fc3b402f5458ff30183244ce04071c2ab57944))

## [0.11.0](https://github.com/loomi-labs/arco/compare/v0.10.0...v0.11.0) (2024-12-02)


### Features

* simplify dark/light-mode settings ([de97019](https://github.com/loomi-labs/arco/commit/de97019886ab4a78015b2c575e6daad5a94bd87c))
* update icons ([5d28a80](https://github.com/loomi-labs/arco/commit/5d28a80f11eb19581ee45a2d8f76e75052e9fa0c))
* update welcome popup ([7f9e434](https://github.com/loomi-labs/arco/commit/7f9e4342f6e2454a5a711bb1a1b1d5ba9d1b4b6c))


### Bug Fixes

* schedule equality check ([27f0c28](https://github.com/loomi-labs/arco/commit/27f0c28b75a15f08fca4abb6a6f95fd24d904850))
* timezone/scheduler bug ([181c320](https://github.com/loomi-labs/arco/commit/181c320f1b2d159ef62e69ac0eb8c8c8d050192c))

## [0.10.0](https://github.com/loomi-labs/arco/compare/v0.9.0...v0.10.0) (2024-11-27)


### Features

* make binary location independent ([249f6be](https://github.com/loomi-labs/arco/commit/249f6be1fed0040121112cf1d5105c45f0504f98))


### Bug Fixes

* borg binary permissions ([cae7325](https://github.com/loomi-labs/arco/commit/cae7325f4f773fc597666a62642afc48dab6916d))
* password validation ([12b778b](https://github.com/loomi-labs/arco/commit/12b778b82a738183cac2e48b9ed76bb258f74aba))

## [0.9.0](https://github.com/loomi-labs/arco/compare/v0.8.0...v0.9.0) (2024-11-27)


### Features

* add auto-update flag ([#106](https://github.com/loomi-labs/arco/issues/106)) ([83bf624](https://github.com/loomi-labs/arco/commit/83bf624130aa86e9d8e1f48d4acfd9298b2c15cc))

## [0.8.0](https://github.com/loomi-labs/arco/compare/v0.7.0...v0.8.0) (2024-11-25)


### Features

* add better version handling (2024-11-22)

## [0.7.0](https://github.com/loomi-labs/arco/compare/v0.6.0...v0.7.0) (2024-11-22)


### Features

* add systray ([#96](https://github.com/loomi-labs/arco/issues/96)) ([9f50aef](https://github.com/loomi-labs/arco/commit/9f50aef29e63864bab53cfe567f6de3a8c743a84))

## [0.6.0](https://github.com/loomi-labs/arco/compare/v0.5.0...v0.6.0) (2024-11-21)


### Features

* add delete repository ([#91](https://github.com/loomi-labs/arco/issues/91)) ([52112ae](https://github.com/loomi-labs/arco/commit/52112aef417e087c7af46ebf6877ec17196c15e2))
* add improved repository connector ([#93](https://github.com/loomi-labs/arco/issues/93)) ([8a9a2ed](https://github.com/loomi-labs/arco/commit/8a9a2ed691f35d0d3ef54ea6ebc294837a790fdd))
* add linux installer script ([#94](https://github.com/loomi-labs/arco/issues/94)) ([716d447](https://github.com/loomi-labs/arco/commit/716d4476321d916bcc22a809c4f8ac542dcc6e3b))

## [0.5.0](https://github.com/loomi-labs/arco/compare/v0.4.0...v0.5.0) (2024-11-15)


### Features

* add archive rename ([#90](https://github.com/loomi-labs/arco/issues/90)) ([0da554e](https://github.com/loomi-labs/arco/commit/0da554e5653f97ccedb5b44139f119a68f1144ff))
* add duration to archives ([#89](https://github.com/loomi-labs/arco/issues/89)) ([4cf150a](https://github.com/loomi-labs/arco/commit/4cf150adca6bc511a36909a08aecba36fca85d1b))
* add startup page ([#87](https://github.com/loomi-labs/arco/issues/87)) ([19de333](https://github.com/loomi-labs/arco/commit/19de33379c87942afd07a3b2f40f7e885256eb1f))

## [0.4.0](https://github.com/loomi-labs/arco/compare/v0.3.0...v0.4.0) (2024-11-13)


### Features

* add welcome message ([#84](https://github.com/loomi-labs/arco/issues/84)) ([035fa80](https://github.com/loomi-labs/arco/commit/035fa80d8964225f685cdf0ace7d4a411c6dd71c))

## [0.3.0](https://github.com/loomi-labs/arco/compare/v0.2.1...v0.3.0) (2024-11-07)

### Features

* improve styling

## [0.2.1](https://github.com/loomi-labs/arco/compare/v0.2.0...v0.2.1) (2024-11-05)


### Bug Fixes

* CI/CD pipeline releasing ([#78](https://github.com/loomi-labs/arco/issues/78)) ([63d6bf5](https://github.com/loomi-labs/arco/commit/63d6bf592b8d2d1fb8623c2eec52491531e5033e))
* CI/CD release version ([#76](https://github.com/loomi-labs/arco/issues/76)) ([377600c](https://github.com/loomi-labs/arco/commit/377600c934a0f6756fbc65f5c8759a4413af9446))

## [0.2.0](https://github.com/loomi-labs/arco/compare/v0.1.0...v0.2.0) (2024-11-04)


### Features

* add auto update ([#71](https://github.com/loomi-labs/arco/issues/71)) ([2befe16](https://github.com/loomi-labs/arco/commit/2befe165eafba3c3a099df69aa9e66654f670a2f))

## 0.1.0 (2024-11-04)


### Features

* firs release
