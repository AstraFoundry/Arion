# Arion

基于 [MyGo](https://github.com/egoist/mygo) 原生 Go UI 构建、面向 macOS 与 Windows 的本地优先语音输入工具。

[English](README.md) · [简体中文](README.zh-CN.md) · [日本語](README.ja.md)

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)
[![Go 1.27.1](https://img.shields.io/badge/Go-1.27.1-00ADD8?logo=go&logoColor=white)](go.work)
[![MyGo v0.4.6](https://img.shields.io/badge/MyGo-v0.4.6-4B5563)](apps/desktop/go.mod)
[![sherpa-onnx 1.13.8](https://img.shields.io/badge/sherpa--onnx-1.13.8-0F766E)](apps/desktop/go.mod)
[![Node 24](https://img.shields.io/badge/Node-24-339933?logo=nodedotjs&logoColor=white)](.nvmrc)
[![pnpm 10.33.0](https://img.shields.io/badge/pnpm-10.33.0-F69220?logo=pnpm&logoColor=white)](package.json)

![Arion 语音输入窗口与悬浮听写胶囊](docs/readme/hero.png)

![Arion 语音听写实际运行效果](docs/readme/demo.gif)

## 概述

Arion 集成了端侧流式语音识别、保守规则纠错、个人词库学习、可选的实时本地改写、本地候选排序契约以及加密记忆同步。用户可以通过全局快捷键或系统输入法（IME）集成进行听写。在完成模型准备后，听写完全离线运行；原始麦克风音频和临时的外部窗口上下文绝不会离开设备。

macOS 与 Windows 是主要桌面目标平台（`P0`），iOS 与 Android 为未来目标平台（`P2`）。

## 功能特性

- **全局快捷键听写与 macOS 输入法宿主** — 在任意应用中按下快捷键（默认 `⌥Space`），说出普通话、英语或中英混杂的技术文本，再次按下即可一次性插入到当前焦点编辑器（按 `Esc` 取消）。同时支持按住说话模式（[DSK-02](specs/001-desktop-input.md)）以及 `ArionInput.app` macOS InputMethodKit 输入法宿主（[ADR 006](docs/decisions/006-macos-input-method-host.md)）。
- **端侧模型自动选择** — 语音识别通过 `sherpa-onnx`（`v1.13.8`）在本地运行。系统会自动下载、校验摘要并加载通过延迟门禁且实测准确率最佳的评测模型包，无需手动选择模型（[ADR 004](docs/decisions/004-automatic-model-selection.md)）。
- **保守纠错与编辑学习** — 应用确定性规则（`arion-rules-v1`）与用户批准的词库，同时将识别草稿与已提交文本严格隔离。在 macOS 上，对已插入文本的后续修改会在端侧自动学习，并支持一键撤销（[ADR 014](docs/decisions/014-edit-learning.md)）。
- **可选的实时本地改写（macOS Apple silicon）** — 在说话过程中通过隔离的 `arion-llm` ONNX Runtime GenAI 工作进程运行 Qwen3 1.7B 实时润色文本，支持 KV 前缀复用、场景感知格式化（**匹配当前应用**，[ADR 012](docs/decisions/012-context-scene-selection.md)），以及端侧光标上下文、屏幕与历史术语提取（[ADR 011](docs/decisions/011-live-local-rewriting.md)、[ADR 013](docs/decisions/013-rewrite-context-on-device.md)）。
- **原生 Go 桌面外壳** — 使用 [MyGo](https://github.com/egoist/mygo)（`v0.4.6`）并在 Go 中采用功能切片设计（Feature-Sliced Design，[ADR 001](docs/decisions/001-desktop-native-boundaries.md)、[ADR 007](docs/decisions/007-sunbeam-design-system.md)），桌面应用不包含 Electron 或 webview。内置菜单栏弹窗、可穿透点击并加入全屏空间的悬浮胶囊、首次运行引导、设置界面、本地诊断 ZIP 导出，以及可实时切换的英文 / 简体中文界面本地化（[i18n](docs/frontend/i18n.md)）。
- **本地加密与可自托管同步中继** — 个人词库使用存储在 macOS 登录钥匙串或 Windows DPAPI 中的密钥进行本地 AES-256-GCM 加密（[ADR 005](docs/decisions/005-local-encryption.md)）。本仓库同时提供 `services/sync`（支持 `noop` 与 `oidc` 账号门禁的可自托管 Go 密文中继服务，[ADR 015](docs/decisions/015-hosted-relay-account-authorization.md)）以及桌面端本地回环 PKCE 登录服务（[账号登录](docs/project/account-signin.md)）。

## 当前状态与限制

Arion 目前是一个可在 macOS（`arm64`）上正常工作的菜单栏听写应用。Windows（`amd64`）代码已可编译并通过无头测试，以及在 Windows Server 2022 CI 上的窗口生命周期、`SendInput` 和长按按键钩子端到端（E2E）测试，但完整的交互式听写尚未在 Windows 主机上完成端到端运行验证。

尚未实现或仍待完成的事项：

- **本地候选排序（`RNK-01`）** — 排序契约与评测框架已存在，但当前识别器仅暴露单一假设结果，因此候选排序尚未启用。
- **桌面记忆同步客户端与设备注册（`MEM-01` / `SYNC-10`）** — `services/sync` 中继服务及桌面端 OIDC 登录视图/服务已实现，但桌面端 SQLite 发件箱（outbox）、设备注册以及原生操作系统凭据存储调用仍待完成。
- **Windows TSF 与实时改写工作进程（`TSF-01` / `RWT-RUN-01`）** — Windows 文本服务框架（TSF）输入法集成、Windows 编辑学习以及 Windows GenAI 改写工作进程尚未实现。
- **按住说话与授权引导的交互式验证** — 按住说话手势逻辑与 Windows 底层钩子已通过 CI E2E 测试，而 macOS 事件监听（event tap）仍需在交互式桌面环境中由用户授予“输入监控”权限进行验证。
- **改写盲测人工评估与通用大语言模型纠错（`LRW-REVIEW-01` / `COR-01`）** — 开发基准测试结果已记录在 [`benchmarks/results`](benchmarks/README.md) 中，但盲测人工评分（`LRW-A06`）仍待完成；独立于实时改写流水线之外的通用 LLM 纠错未被采用。
- **签名公开发布（`REL-01`）** — 应用内更新器已实现（[ADR 010](docs/decisions/010-in-app-updates.md)），但 Developer ID 代码签名、公证（notarization）及生产环境 Bundle ID 仍待配置。

详细的验证证据与任务门禁请参阅[代理交接文档](docs/project/agent-handoff.md)与[验证报告](docs/quality/validation.md)。

## 快速开始

### 前置条件

- **Go 1.27.1** 且启用 `cgo`（`go.work` 使用 Go 自动工具链选择；`packages/core` 与 `services/sync` 也会在 `GOWORK=off` 下独立检查）。
- **macOS（`arm64`）**：Xcode Command Line Tools 与 macOS SDK。
- **Windows（`amd64`）**：MyGo 所需的原生工具链，以及交叉打包 `cgo` 时所需的 MinGW-w64（`CC=x86_64-w64-mingw32-gcc`）和构建安装程序所需的 NSIS。
- **Node 24**（`.nvmrc`）与**锁定版本的 `pnpm@10.33.0`**（`package.json`），用于仓库检查、评测工具及静态宣传站点（`apps/site`）。拒绝使用 `npm` 与 `yarn`。

### 构建与运行

```sh
make install        # 安装锁定版本的 pnpm 工作区依赖 (--frozen-lockfile)
make check          # 文档、JSON、Go 分层边界、core/sync 与评测测试
make desktop-check  # 桌面端测试与 go vet (cgo)
make desktop-dev    # 准备原生运行时并以开发模式启动桌面应用
make desktop-build  # 在 apps/desktop/build/ 下打包 Arion.app 与 DMG
make e2e            # 在真实桌面会话中运行原生窗口生命周期 E2E 测试
make e2e-macos      # 针对打包后的 Arion.app 运行产品场景测试
make site-check     # 宣传站点 (apps/site)：i18n、类型、lint、测试与静态导出
```

首次运行 `make desktop-dev` 时，应用会打开引导界面，请求麦克风与辅助功能（Accessibility）权限，并将自动选择的识别模型包（约 270 MB）下载至 `~/Library/Application Support/Arion/models/`（macOS）或 `%AppData%\Arion\models\`（Windows）。模型准备完成后，听写即可离线工作。运行 `make help` 可查看所有基准测试、评测与诊断命令。

## 文档导航

### 项目与规范

- [代理编码指南 (`AGENTS.md`)](AGENTS.md) — 面向贡献者与编码代理的规则、边界与工作流。
- [文档索引 (`docs/README.md`)](docs/README.md) — 所有文档领域的顶层导航图。
- [产品范围 (`docs/project/README.md`)](docs/project/README.md) — 已确认的产品范围、明确边界与技术栈状态表。
- [系统架构 (`docs/project/architecture.md`)](docs/project/architecture.md) — 数据流、实时本地改写流水线、运行时形态与隐私边界。
- [仓库职责划分 (`docs/project/repository.md`)](docs/project/repository.md) — Go 工作区模块、FSD/DDD 包边界与工作规则。
- [产品规范 (`specs/README.md`)](specs/README.md) — 规范性需求（`001` 至 `009`）与验收场景 ID。
- [代理交接 (`docs/project/agent-handoff.md`)](docs/project/agent-handoff.md) — 已实现的里程碑、待办的 `P0`/`P1`/`P2` 任务及完成门禁。
- [架构决策记录 (`docs/decisions/README.md`)](docs/decisions/README.md) — ADR `001` 至 `016`。

### 工程、运维与评测

- [前端规范 (`docs/frontend/README.md`)](docs/frontend/README.md) — 原生 Go 中的功能切片设计（FSD）与[界面本地化 (`docs/frontend/i18n.md`)](docs/frontend/i18n.md)。
- [后端规范 (`docs/backend/README.md`)](docs/backend/README.md) — 面向 `packages/core` 与 `services/sync` 的领域驱动设计（DDD）规则。
- [本地开发 (`docs/operations/local-dev.md`)](docs/operations/local-dev.md) — 工具链、数据目录、环境变量与故障排查。
- [部署与发布门禁 (`docs/operations/deployment.md`)](docs/operations/deployment.md) 与[同步中继运维手册 (`deploy/sync/README.md`)](deploy/sync/README.md)。
- [质量与测试 (`docs/quality/README.md`)](docs/quality/README.md)、[验证报告 (`docs/quality/validation.md`)](docs/quality/validation.md) 与[原生 E2E (`docs/quality/e2e.md`)](docs/quality/e2e.md)。
- [研究证据 (`docs/research/README.md`)](docs/research/README.md)、[基准测试方法论 (`docs/research/benchmark-methodology.md`)](docs/research/benchmark-methodology.md)、[评测运行器 (`tools/evaluation/README.md`)](tools/evaluation/README.md) 与[基准测试结果 (`benchmarks/README.md`)](benchmarks/README.md)。

## 贡献者

[![AstraFoundry/Arion contributors](https://contrib.rocks/image?repo=AstraFoundry/Arion)](https://github.com/AstraFoundry/Arion/graphs/contributors)

## Star 历史

[![AstraFoundry/Arion Star History Chart](https://api.star-history.com/svg?repos=AstraFoundry/Arion&type=Date)](https://star-history.com/#AstraFoundry/Arion&Date)

## 许可证

Arion 由 [AstraFoundry](https://github.com/AstraFoundry) 开发，并基于 [AGPL-3.0](LICENSE) 许可证开源（[ADR 009](docs/decisions/009-agpl-license.md)）。请在 [GitHub Issues](https://github.com/AstraFoundry/Arion/issues) 中反馈问题。
