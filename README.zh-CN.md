# Arion

基于 [MyGo](https://github.com/egoist/mygo) 原生 Go UI 构建、面向 macOS 与 Windows 的本地优先语音输入工具。

[English](README.md) · [简体中文](README.zh-CN.md) · [日本語](README.ja.md)

[![X @AstraFoundry](https://img.shields.io/badge/X-%40AstraFoundry-000000?logo=x&logoColor=white)](https://x.com/AstraFoundry)
[![Discord](https://img.shields.io/badge/Discord-Join_Chat-5865F2?logo=discord&logoColor=white)](https://discord.com/invite/72kh465mMm)
[![飞书群](https://img.shields.io/badge/飞书-加入群聊-3370FF)](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)
[![Go 1.27.1](https://img.shields.io/badge/Go-1.27.1-00ADD8?logo=go&logoColor=white)](go.work)
[![MyGo v0.4.6](https://img.shields.io/badge/MyGo-v0.4.6-4B5563)](apps/desktop/go.mod)
[![Native Qwen3-ASR](https://img.shields.io/badge/ASR-Qwen3_1.7B-0F766E)](docs/decisions/020-native-qwen-asr.md)
[![Node 24](https://img.shields.io/badge/Node-24-339933?logo=nodedotjs&logoColor=white)](.nvmrc)
[![pnpm 10.33.0](https://img.shields.io/badge/pnpm-10.33.0-F69220?logo=pnpm&logoColor=white)](package.json)

![Arion 语音输入窗口与悬浮听写胶囊](docs/readme/hero.png)

![Arion 语音听写实际运行效果](docs/readme/demo.gif)

## 概述

Arion 是一款本地优先的桌面语音输入工具，集成了端侧流式语音识别、确定性规则纠错、个人词库学习、上下文感知改写与加密记忆同步。

- **默认离线运行** — 完成首次模型准备后，语音识别与本地改写完全在本地设备运行。
- **严格的隐私边界** — 原始麦克风音频不会离开设备；只有显式选择云端润色后，才会发送已开启的上下文（[005 隐私规范](specs/005-privacy.md)）。
- **聚焦核心桌面平台** — Apple silicon 架构的 macOS（`arm64`）与 Windows（`x64`）是主要 `P0` 目标平台；不支持 Intel macOS（[ADR 021](docs/decisions/021-apple-silicon-only-macos.md)），iOS 与 Android 为未来 `P2` 目标平台。

## 核心特性

### 语音输入与系统集成

- **全局听写快捷键** — 在任意应用中按住听写键（macOS 默认右 `⌥`，Windows 默认右 `Alt`），说出普通话、英语或中英混杂的技术文本，松开即可一次性插入到当前焦点编辑器；轻按一下则会持续监听直至再次按下（按 `Esc` 取消，[DSK-02](specs/001-desktop-input.md)）。
- **macOS 输入法宿主（`ArionInput.app`）** — 作为系统输入源直接集成，支持语音合成上屏以及离线简体中文全拼输入（[ADR 006](docs/decisions/006-macos-input-method-host.md)、[ADR 017](docs/decisions/017-offline-pinyin-keyboard.md)）。
- **原生 Go 桌面外壳** — 基于 [MyGo](https://github.com/egoist/mygo)（`v0.4.6`）与 Sunbeam 设计系统，在 Go 中采用功能切片设计（Feature-Sliced Design）构建，不包含 Electron 或 webview（[ADR 001](docs/decisions/001-desktop-native-boundaries.md)、[ADR 007](docs/decisions/007-sunbeam-design-system.md)）。内置菜单栏弹窗、可穿透点击并加入全屏空间的悬浮胶囊、首次运行引导、设置界面、本地诊断 ZIP 导出，以及可实时切换的英文 / 简体中文界面本地化（[i18n](docs/frontend/i18n.md)）。

### 语音识别、纠错与智能润色

- **原生端侧语音识别（ASR）** — 在 Apple silicon 上通过原生 Swift MLX（8-bit）运行 Qwen3-ASR 1.7B，在 Windows x64 上通过原生 `speech.cpp`（Q8_0）运行，并支持带摘要校验的模型自动下载与准备（[ADR 004](docs/decisions/004-automatic-model-selection.md)、[ADR 020](docs/decisions/020-native-qwen-asr.md)）。
- **保守纠错与编辑学习** — 应用确定性规则（`arion-rules-v3`）与用户批准的词库，同时将识别草稿与已提交文本严格隔离。在 macOS 上，对已插入文本的后续修改会在端侧自动学习，并支持一键撤销（[ADR 014](docs/decisions/014-edit-learning.md)）。
- **分阶段本地改写（macOS Apple silicon）** — 通过本地两阶段 GGUF 流水线（基于 `llama.cpp` 的 `Qwen3.5-2B`）润色语音文本，支持场景感知格式化（**匹配当前应用**）以及端侧光标、屏幕与近期听写上下文提取（[ADR 011](docs/decisions/011-live-local-rewriting.md)、[ADR 012](docs/decisions/012-context-scene-selection.md)、[ADR 013](docs/decisions/013-rewrite-context-on-device.md)、[ADR 018](docs/decisions/018-staged-gguf-rewriting.md)）。
- **可选的云端润色** — 支持通过 [Republic](https://github.com/AstraFoundry/republic) 显式开启云端终稿润色，可使用 ChatGPT 登录或接入单个服务商 API Key（OpenAI、Anthropic、Google Gemini、DeepSeek、OpenRouter）。与本机共用分阶段润色流程及上下文开关，已开启的上下文与召回词条用于纠错，音频始终留在本机（[ADR 022](docs/decisions/022-cloud-polishing.md)、[云端润色说明](docs/project/cloud-polishing.md)）。

### 安全加密、同步与更新

- **本地加密与可自托管同步中继** — 个人词库使用存储在 macOS 登录钥匙串或 Windows DPAPI 中的密钥进行本地 AES-256-GCM 加密（[ADR 005](docs/decisions/005-local-encryption.md)）。仓库同时提供支持 `noop` 与 `oidc` 账号门禁的可自托管 Go 密文中继服务 `services/sync`（[ADR 015](docs/decisions/015-hosted-relay-account-authorization.md)）及桌面端本地回环 PKCE 登录服务（[账号登录](docs/project/account-signin.md)）。
- **中心服务签名更新** — 应用内更新器通过 Arion 中心服务（`api.arion.astrafoundry.si`、`services/updates`）检查 `stable` 与 `beta` 渠道，并在安装前验证 Ed25519 软件包签名（[ADR 010](docs/decisions/010-in-app-updates.md)、[ADR 019](docs/decisions/019-center-update-channels.md)、[deploy/center](deploy/center/README.md)）。

## 当前状态与待完成门禁

Arion 目前是一个可在 macOS（`arm64`）上正常工作的菜单栏听写应用。Windows（`amd64`）代码已可编译并通过无头测试，以及在 Windows Server 2022 CI 上的窗口生命周期、`SendInput` 和长按按键钩子端到端（E2E）测试，但完整的交互式听写尚未在 Windows 主机上完成端到端运行验证。

尚未启用或仍待完成的事项：

- **本地候选排序（`RNK-01`）** — 排序契约与评测框架已存在，但当前识别器仅暴露单一假设结果，因此候选排序尚未启用。
- **桌面记忆同步客户端与设备注册（`MEM-01` / `SYNC-10`）** — `services/sync` 中继服务及桌面端 OIDC 登录流程已实现，但桌面端 SQLite 发件箱（outbox）、设备注册以及端到端密钥同步仍待完成。
- **Windows TSF 与本地改写工作进程（`TSF-01` / `RWT-RUN-01`）** — Windows 文本服务框架（TSF）输入法集成、Windows 编辑学习以及 Windows 本地改写运行时尚未实现。
- **交互式系统权限与输入法验收** — 按住说话手势逻辑与 Windows 底层钩子已通过 CI E2E 测试，而 macOS 事件监听（输入监控权限）及全新环境下的 IMK 注册仍需在交互式桌面环境中验证。
- **改写盲测人工评估（`LRW-REVIEW-01` / `COR-01`）** — 开发基准测试结果已记录在 [`benchmarks/results`](benchmarks/README.md) 中，但盲测人工评分（`LRW-A06`）仍待完成；独立于改写流水线之外的通用 LLM 纠错未被采用。
- **签名公开发布（`REL-01`）** — 应用内更新器与发布暂存流水线已实现，生产环境 Developer ID 代码签名、公证（notarization）及正式 Bundle ID 仍待配置。

详细的验证证据与任务门禁请参阅[代理交接文档](docs/project/agent-handoff.md)与[验证报告](docs/quality/validation.md)。

## 快速开始

### 前置条件

- **Go 1.27.1** 且启用 `cgo`（`go.work` 使用 Go 自动工具链选择；`packages/core` 与 `services/sync` 也会在 `GOWORK=off` 下独立检查）。
- **macOS（`arm64`）** — Xcode Command Line Tools 与 macOS SDK。
- **Windows（`amd64`）** — MyGo 所需的原生工具链，以及交叉打包 `cgo` 时所需的 MinGW-w64（`CC=x86_64-w64-mingw32-gcc`）和构建安装程序所需的 NSIS。
- **Node 24**（`.nvmrc`）与**锁定版本的 `pnpm@10.33.0`**（`package.json`）— 用于仓库检查、评测工具及静态宣传站点（`apps/site`）。拒绝使用 `npm` 与 `yarn`。

### 常用命令

```sh
make install        # 安装锁定版本的 pnpm 工作区依赖 (--frozen-lockfile)
make check          # 文档、JSON、Go 分层边界、core/sync 与评测测试
make desktop-check  # 桌面端测试与 go vet (cgo)
make desktop-dev    # 准备原生运行时并以开发模式启动桌面应用
make build desktop  # 在 apps/desktop/build/ 下打包 Arion.app 与 DMG
make e2e            # 在真实桌面会话中运行原生窗口生命周期 E2E 测试
make e2e-macos      # 针对打包后的 Arion.app 运行产品场景测试
make site-check     # 宣传站点 (apps/site)：i18n、类型、lint、测试与静态导出
```

首次运行 `make desktop-dev` 时，应用会打开引导界面，请求麦克风与辅助功能（Accessibility）权限，并将默认识别模型包下载至 `~/Library/Application Support/Arion/models/`（macOS）或 `%AppData%\Arion\models\`（Windows）。模型准备完成后，听写即可离线工作。运行 `make help` 可查看所有基准测试、评测与诊断命令。

## 文档导航

| 领域 | 入口文档 |
| --- | --- |
| **开发准则与总览** | [代理编码指南 (`AGENTS.md`)](AGENTS.md) · [文档索引 (`docs/README.md`)](docs/README.md) |
| **产品与架构** | [产品范围 (`docs/project/README.md`)](docs/project/README.md) · [系统架构 (`docs/project/architecture.md`)](docs/project/architecture.md) · [仓库职责划分 (`docs/project/repository.md`)](docs/project/repository.md) |
| **规范与决策** | [产品规范 (`specs/README.md`)](specs/README.md) · [架构决策记录 (`docs/decisions/README.md`)](docs/decisions/README.md) · [代理交接 (`docs/project/agent-handoff.md`)](docs/project/agent-handoff.md) |
| **前端与后端** | [前端 FSD 规范 (`docs/frontend/README.md`)](docs/frontend/README.md) · [界面本地化 (`docs/frontend/i18n.md`)](docs/frontend/i18n.md) · [后端 DDD 规范 (`docs/backend/README.md`)](docs/backend/README.md) |
| **运维与部署** | [本地开发 (`docs/operations/local-dev.md`)](docs/operations/local-dev.md) · [部署与发布门禁 (`docs/operations/deployment.md`)](docs/operations/deployment.md) · [同步中继运维手册 (`deploy/sync/README.md`)](deploy/sync/README.md) · [中心服务部署 (`deploy/center/README.md`)](deploy/center/README.md) |
| **质量与评测** | [质量与测试 (`docs/quality/README.md`)](docs/quality/README.md) · [验证报告 (`docs/quality/validation.md`)](docs/quality/validation.md) · [原生 E2E (`docs/quality/e2e.md`)](docs/quality/e2e.md) · [研究证据 (`docs/research/README.md`)](docs/research/README.md) · [基准测试方法论 (`docs/research/benchmark-methodology.md`)](docs/research/benchmark-methodology.md) · [评测运行器 (`tools/evaluation/README.md`)](tools/evaluation/README.md) · [基准测试结果 (`benchmarks/README.md`)](benchmarks/README.md) |

## 贡献者

[![AstraFoundry/Arion contributors](https://contrib.rocks/image?repo=AstraFoundry/Arion)](https://github.com/AstraFoundry/Arion/graphs/contributors)

## Star 历史

[![AstraFoundry/Arion Star History Chart](https://api.star-history.com/svg?repos=AstraFoundry/Arion&type=Date)](https://star-history.com/#AstraFoundry/Arion&Date)

## 许可证

Arion 由 [AstraFoundry](https://github.com/AstraFoundry) 开发，并基于 [AGPL-3.0](LICENSE) 许可证开源（[ADR 009](docs/decisions/009-agpl-license.md)）。欢迎在 [X (@AstraFoundry)](https://x.com/AstraFoundry) 关注我们，加入 [Discord](https://discord.com/invite/72kh465mMm) 或[飞书群](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae)交流，并在 [GitHub Issues](https://github.com/AstraFoundry/Arion/issues) 中反馈问题。
