<div align="center">

# Arion

基于 [MyGo](https://github.com/egoist/mygo) 原生 Go UI 构建、面向 macOS 与 Windows 的本地优先语音与拼音输入工具。

[English](README.md) · [简体中文](README.zh-CN.md) · [日本語](README.ja.md)

<p align="center">
  <a href="https://x.com/AstraFoundry"><img src="https://img.shields.io/badge/X-%40AstraFoundry-000000?logo=x&logoColor=white" alt="X @AstraFoundry" style="display: inline-block; margin: 2px;" /></a>
  <a href="https://discord.com/invite/72kh465mMm"><img src="https://img.shields.io/badge/Discord-Join_Chat-5865F2?logo=discord&logoColor=white" alt="Discord" style="display: inline-block; margin: 2px;" /></a>
  <a href="https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae"><img src="https://img.shields.io/badge/飞书-加入群聊-3370FF" alt="飞书群" style="display: inline-block; margin: 2px;" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-AGPL--3.0-blue.svg" alt="License: AGPL-3.0" style="display: inline-block; margin: 2px;" /></a>
</p>

<p align="center">
  <a href="go.work"><img src="https://img.shields.io/badge/Go-1.27.1-00ADD8?logo=go&logoColor=white" alt="Go 1.27.1" style="display: inline-block; margin: 2px;" /></a>
  <a href="apps/desktop/go.mod"><img src="https://img.shields.io/badge/MyGo-v0.4.6-4B5563" alt="MyGo v0.4.6" style="display: inline-block; margin: 2px;" /></a>
  <a href="docs/decisions/020-native-qwen-asr.md"><img src="https://img.shields.io/badge/ASR-Qwen3_1.7B-0F766E" alt="Native Qwen3-ASR" style="display: inline-block; margin: 2px;" /></a>
  <a href=".nvmrc"><img src="https://img.shields.io/badge/Node-24-339933?logo=nodedotjs&logoColor=white" alt="Node 24" style="display: inline-block; margin: 2px;" /></a>
  <a href="package.json"><img src="https://img.shields.io/badge/pnpm-10.33.0-F69220?logo=pnpm&logoColor=white" alt="pnpm 10.33.0" style="display: inline-block; margin: 2px;" /></a>
</p>

![Arion 语音听写实际运行效果](docs/readme/demo.gif)

</div>

## 概述

云端语音输入通常需要将原始麦克风音频和窗口上下文上传至远程服务器，而系统自带的基础离线听写又难以兼顾句内中英混说、领域术语自学习与按目标应用自动排版。

[AstraFoundry](https://astrafoundry.si) 构建 **Arion** 的目标是提供一套**开源、本地优先、可自托管**的桌面语音与拼音输入法：在设备本地完成流式语音识别、离线全拼输入、修改习惯学习与上下文智能润色，将原始音频与加密密钥保留在用户本机。

## 核心特性

- **按住说话与原生系统输入法** — 在任意焦点编辑器中按住全局听写快捷键（macOS 默认右 `⌥`，Windows 默认右 `Alt`；可在设置中录制其他单键或修饰键组合，参阅[听写快捷键](docs/project/dictation-shortcuts.md)）即可识别普通话、英语及中英混杂的技术表达（如 *“把这个 PR merge 到 main 之前，先跑一下 CI”*），松开一次性上屏；轻按一下则持续听写直至再次按下（`Esc` 取消）。同时内置原生系统输入法宿主（macOS `ArionInput.app` 与 Windows `ArionTSF.dll`），支持离线简体中文全拼输入与 `Shift` 中英切换。
- **端侧原生语音识别** — 在 Apple silicon 上通过原生 Swift MLX（8-bit）、在 Windows x64 上通过原生 `speech.cpp`（Q8_0，CPU/Vulkan）本地运行 Qwen3-ASR 1.7B（[ADR 020](docs/decisions/020-native-qwen-asr.md)）。模型下载并通过 SHA-256 校验后即可完全离线运行；麦克风音频仅驻留当次会话内存，自动跳过密码框，并内置分档软件输入增益（`−6 dB` 至 `+12 dB`）以支持开放办公区低声口述。
- **改词自学习与草稿隔离** — 说话期间流式草稿仅保留在悬浮胶囊内，松开按键后才一次性提交至编辑器。本地自动应用确定性清理规则与个人词库；在 macOS 原生编辑器中，文字上屏后若手动修改了误识别词，系统会在端侧自动学习并支持一键撤销（[ADR 014](docs/decisions/014-edit-learning.md)）。
- **两阶段上下文智能润色** — 松开听写键后可选执行一次两阶段润色（[ADR 023](docs/decisions/023-final-only-polishing.md)），剔除口语停顿词、合并句内自我更正，结合当前窗口/光标附近的可见术语消歧（[ADR 013](docs/decisions/013-rewrite-context-on-device.md)），并按当前应用自动排版（**匹配当前应用**，[ADR 012](docs/decisions/012-context-scene-selection.md)）。可选的第二个快捷键可边听写边翻译为 12 种目标语言。支持在 macOS Apple silicon 与 Windows x64 上通过 `llama.cpp` 本地运行 Qwen3.5-2B（`Q4_K_M` GGUF，[ADR 018](docs/decisions/018-staged-gguf-rewriting.md)、[ADR 025](docs/decisions/025-windows-local-polishing.md)），或显式接入云端模型（ChatGPT 登录或 OpenAI、Anthropic、Gemini、DeepSeek、OpenRouter API Key），原始音频始终留在本地（[ADR 022](docs/decisions/022-cloud-polishing.md)）。
- **原生 Go 桌面外壳** — 基于 [MyGo](https://github.com/egoist/mygo)（`v0.4.6`）并在 Go 中采用功能切片设计（FSD）构建，无 Electron 或 WebView（[ADR 001](docs/decisions/001-desktop-native-boundaries.md)、[ADR 007](docs/decisions/007-sunbeam-design-system.md)）。常驻菜单栏/系统托盘，悬浮胶囊支持跨全屏空间展示与点击穿透，不抢占编辑器焦点，支持中英文界面实时切换（[i18n](docs/frontend/i18n.md)）。
- **零知识加密同步与签名更新** — 个人词库使用存储在 macOS Keychain 或 Windows DPAPI 中的密钥进行本地 AES-256-GCM 加密（[ADR 005](docs/decisions/005-local-encryption.md)）。仓库内置支持 Docker 自托管且不持有解密密钥的 Go 密文中继服务 `services/sync`（[ADR 015](docs/decisions/015-hosted-relay-account-authorization.md)），以及提供 Ed25519 签名 `stable`/`beta` 通道的更新服务 `services/updates`（[ADR 019](docs/decisions/019-center-update-channels.md)）。

- **可控更新与产品通知** — 自动检查与已验证下载独立设置，下载后的更新等待用户明确选择安全重启。系统通知设置统一控制更新提醒和登录后的账户定向通知，支持应用运行时接收及下次启动补收。参阅[更新生命周期](docs/project/update-lifecycle.md)与[产品通知](docs/project/product-notifications.md)。

## 平台支持

| 平台 | 架构 | 当前状态 |
| --- | --- | --- |
| **macOS** | Apple silicon (`arm64`) | 完整支持菜单栏听写、原生 MLX 语音识别、本地 GGUF 润色与 `ArionInput.app` 输入法宿主（不支持 Intel macOS，[ADR 021](docs/decisions/021-apple-silicon-only-macos.md)） |
| **Windows** | 64 位 (`x64` / `amd64`) | 早期预览版，支持原生 `speech.cpp` 语音识别、本地 GGUF 润色与 `ArionTSF.dll` 输入法宿主 |

详细的里程碑进展与发布验收门禁请参阅[发布状态](docs/operations/release-status.md)、[代理交接文档](docs/project/agent-handoff.md)与[验证报告](docs/quality/validation.md)。

## 快速开始

### 前置条件

- **Go 1.27.1** 且启用 `cgo`（`go.work` 使用 Go 自动工具链选择）。
- **macOS（`arm64`）**：完整的 **Xcode 26.4.1** 应用（Swift 6.3.1）与 **Metal Toolchain** 组件：
  ```sh
  sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
  sudo xcodebuild -license accept && sudo xcodebuild -runFirstLaunch
  xcodebuild -downloadComponent MetalToolchain
  ```
- **Windows（`amd64`）**：原生 C/C++ 工具链、CMake、MinGW-w64 GCC（`CC=x86_64-w64-mingw32-gcc`）与 NSIS。
- **Node 24**（`.nvmrc`）与**锁定版本的 `pnpm@10.33.0`**（`package.json`）。

### 构建与运行

```sh
make install        # 安装锁定版本的 pnpm 工作区依赖 (--frozen-lockfile)
make check          # 文档、JSON、Go 分层边界、工作流检查，以及 core/sync/updates/i18n/评测测试
make desktop-check  # 桌面端测试、键盘检查与 go vet (cgo)
make desktop-dev    # 准备原生运行时并以开发模式启动 Arion Dev
make build desktop  # 在 apps/desktop/build/ 下打包 Arion.app + DMG (macOS) 或 NSIS 安装包 (Windows)
make e2e            # 在真实桌面会话中运行原生窗口生命周期 E2E 测试
make e2e-macos      # 针对打包后的 Arion.app 运行产品场景测试
make site-check     # 宣传站点 (apps/site)：i18n、类型、lint、测试与静态导出
```

首次运行 `make desktop-dev` 时，应用会打开引导界面，请求麦克风与辅助功能权限，并将默认识别模型包下载至 `~/Library/Application Support/Arion/models/`（macOS）或 `%AppData%\Arion\models\`（Windows）。关于 macOS 开发模式构建与 TCC 权限排障说明，请参阅[本地开发指南 (`docs/operations/local-dev.md`)](docs/operations/local-dev.md#macos-development-build-and-tcc-permission-diagnostics)。

## 文档导航

| 领域 | 入口文档 |
| --- | --- |
| **总览与开发准则** | [文档索引 (`docs/README.md`)](docs/README.md) · [产品范围 (`docs/project/README.md`)](docs/project/README.md) · [代理编码指南 (`AGENTS.md`)](AGENTS.md) |
| **架构与规范** | [系统架构 (`docs/project/architecture.md`)](docs/project/architecture.md) · [仓库职责划分 (`docs/project/repository.md`)](docs/project/repository.md) · [产品规范 (`specs/README.md`)](specs/README.md) · [架构决策记录 (`docs/decisions/README.md`)](docs/decisions/README.md) |
| **前端与后端** | [前端 FSD 规范 (`docs/frontend/README.md`)](docs/frontend/README.md) · [界面本地化 (`docs/frontend/i18n.md`)](docs/frontend/i18n.md) · [后端 DDD 规范 (`docs/backend/README.md`)](docs/backend/README.md) |
| **运维与部署** | [本地开发 (`docs/operations/local-dev.md`)](docs/operations/local-dev.md) · [部署与发布门禁 (`docs/operations/deployment.md`)](docs/operations/deployment.md) · [发布状态 (`docs/operations/release-status.md`)](docs/operations/release-status.md) · [Arion 中心服务 (`deploy/center/README.md`)](deploy/center/README.md) · [同步中继运维 (`deploy/sync/README.md`)](deploy/sync/README.md) |
| **质量与评测** | [质量与测试 (`docs/quality/README.md`)](docs/quality/README.md) · [原生 E2E (`docs/quality/e2e.md`)](docs/quality/e2e.md) · [研究证据 (`docs/research/README.md`)](docs/research/README.md) · [评测运行器 (`tools/evaluation/README.md`)](tools/evaluation/README.md) · [基准测试结果 (`benchmarks/README.md`)](benchmarks/README.md) |

## 贡献者

[![AstraFoundry/Arion contributors](https://contrib.rocks/image?repo=AstraFoundry/Arion)](https://github.com/AstraFoundry/Arion/graphs/contributors)

## Star 历史

[![AstraFoundry/Arion Star History Chart](https://api.star-history.com/svg?repos=AstraFoundry/Arion&type=Date)](https://star-history.com/#AstraFoundry/Arion&Date)

## 许可证

Arion 由 [AstraFoundry](https://github.com/AstraFoundry) 开发，并基于 [AGPL-3.0](LICENSE) 许可证开源（[ADR 009](docs/decisions/009-agpl-license.md)）。欢迎在 [X (@AstraFoundry)](https://x.com/AstraFoundry) 关注我们，加入 [Discord](https://discord.com/invite/72kh465mMm) 或[飞书群](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae)交流，并在 [GitHub Issues](https://github.com/AstraFoundry/Arion/issues) 中反馈问题。
