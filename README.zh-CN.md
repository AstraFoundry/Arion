# Arion

基于 [MyGo](https://github.com/egoist/mygo) 原生 Go UI 构建、面向 macOS 与 Windows 的本地优先（Local-first）语音与拼音输入工具。

[English](README.md) · [简体中文](README.zh-CN.md) · [日本語](README.ja.md)

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)
[![Go 1.27.1](https://img.shields.io/badge/Go-1.27.1-00ADD8?logo=go&logoColor=white)](go.work)
[![MyGo v0.4.6](https://img.shields.io/badge/MyGo-v0.4.6-4B5563)](apps/desktop/go.mod)
[![Native Qwen3-ASR](https://img.shields.io/badge/ASR-Qwen3_1.7B-0F766E)](docs/decisions/020-native-qwen-asr.md)
[![Node 24](https://img.shields.io/badge/Node-24-339933?logo=nodedotjs&logoColor=white)](.nvmrc)
[![pnpm 10.33.0](https://img.shields.io/badge/pnpm-10.33.0-F69220?logo=pnpm&logoColor=white)](package.json)

![Arion 语音输入窗口与悬浮听写胶囊](docs/readme/hero.png)

![Arion 语音听写实际运行效果](docs/readme/demo.gif)

## 项目背景与要解决的问题

输入法位于操作系统文本输入链路的底层，直接接触即时通讯、文档撰写、搜索与代码指令等高频个人上下文。现有桌面语音输入方案通常面临两类工程局限：

1. **云端语音输入依赖远程服务端识别与文本后处理**，原始音频流、修改记录及窗口上下文需要上传至外部服务器，难以满足数据留在本地的隐私与合规要求。
2. **系统内置的基础离线听写缺乏个人上下文适应能力**，在中英混说（Code-switching）、领域专有术语自学习、口语停顿词清理以及按目标应用格式化上存在明显短板。

[AstraFoundry](https://astrafoundry.si) 构建 Arion 的目标是提供一套**开源、端侧优先、可自托管**的桌面语音与拼音输入实现：在设备本地完成流式语音识别、离线全拼输入、修改习惯学习与可选的上下文文本润色，将原始音频与加密密钥保留在用户本机。

### 解决的具体场景

- **离线语音识别与输入安全边界**
  识别模型在首次下载并通过固定 SHA-256 摘要校验后，听写完全在本机离线运行。麦克风音频仅驻留于当次会话内存，会话结束即释放，不落盘、不外发；遇到密码框或安全属性未知的输入控件时，系统自动跳过上下文读取与词库学习（[DSK-08](specs/001-desktop-input.md)）。
- **近讲与低声拾音软件增益**
  内置按说话距离标定的分档软件输入增益（`−6 dB` 对应 15 cm 近距低声至 `+12 dB` 对应 120 cm，步进 3 dB，[DSK-12](specs/001-desktop-input.md)）。支持在开放办公区低声口述，仅缩放 Arion 内部采集音频，不改动操作系统全局麦克风音量。
- **句内中英混说与草稿隔离**
  无需预先切换语种即可在同一句话中识别中文、英文及技术术语（例如 *“把这个 PR merge 到 main 之前，先跑一下 CI”*）。说话期间流式草稿仅保留在悬浮胶囊内，松开按键或确认结束后才向焦点输入框一次性提交。
- **端侧改词自学习与可选两阶段润色**
  - **改词学习（Edit Learning）**：在暴露 `AXValue` 的 macOS 原生编辑器中，文字上屏后若用户手动修改了误识别词，端侧跟踪器会对比提取词对并加密写入本地词库，支持在悬浮胶囊上一键撤销（[ADR 014](docs/decisions/014-edit-learning.md)）。
  - **两阶段上下文润色**：开启润色后，松开听写键时通过独立的 `arion-llm`（`llama.cpp`）工作进程运行本地语言模型（或显式启用云端文本模型），剔除停顿词、合并句中口头更正，并结合当前窗口/光标附近的可见术语进行专有名词消歧与场景化排版。

## 核心优势

| 维度 | 实现方式 |
| --- | --- |
| **端侧推理与零知识同步** | 语音识别与词库学习在本地执行；个人词库使用系统凭据库（macOS Keychain / Windows DPAPI）托管密钥进行 AES-256-GCM 加密。同步中继（`services/sync`）仅存储密文与路由元数据，支持 Docker 自托管，账号登录不持有解密密钥。 |
| **全局听写与系统 IME 统一路由** | 支持按住全局听写键（macOS 默认右 `⌥`，Windows 默认右 `Alt`）向任意焦点窗口输入，或通过内置的系统输入法宿主（macOS `ArionInput.app` / Windows TSF `ArionTSF.dll`）进行离线全拼与语音合成，支持短按 `Shift` 切换中英文。 |
| **原生 Go 桌面外壳** | 基于 [MyGo](https://github.com/egoist/mygo) 原生 Go UI 构建（不依赖 Electron 或 WebView），常驻菜单栏/系统托盘；悬浮胶囊支持跨 macOS 全屏空间展示与点击穿透，不抢占目标编辑器焦点。 |
| **确定性会话与错误边界** | 识别草稿与已提交文本严格隔离，取消或焦点转移会立即作废后台生成代数（Generation）；模型缺失或权限不足时返回明确错误码与修复指引，不静默回退至云端识别。 |
| **全栈开放源码** | 基于 [AGPL-3.0](LICENSE) 开源，单仓库包含桌面客户端、原生语音与润色工作进程、密文同步中继、更新通道服务及可复现的评测基准。 |

## 技术架构与实现特性

Apple silicon 上的 macOS 与 Windows x64 是主要桌面目标平台（`P0`）；Intel macOS 不再支持（[ADR 021](docs/decisions/021-apple-silicon-only-macos.md)）；iOS 与 Android 为未来目标平台（`P2`）。

- **全局听写键与原生输入法宿主** — 在任意应用中按住听写键（macOS 默认右 `⌥`，Windows 默认右 `Alt`）说话，松开即可一次性插入；轻点一下则持续听写，直到再次按下结束（按 `Esc` 取消，[DSK-02](specs/001-desktop-input.md)）。同时内置支持离线全拼与加密词频学习的 `ArionInput.app` macOS InputMethodKit 宿主（[ADR 006](docs/decisions/006-macos-input-method-host.md)、[ADR 017](docs/decisions/017-offline-pinyin-keyboard.md)）与 Windows x64 TSF 输入法宿主（[Windows 输入法](docs/operations/windows-input-method.md)）。
- **端侧原生语音识别** — Qwen3-ASR 1.7B 在 Apple silicon 上通过原生 Swift MLX 8-bit 运行，在 Windows x64 上通过原生 `speech.cpp` Q8_0（CPU/Vulkan）运行（[ADR 020](docs/decisions/020-native-qwen-asr.md)）。
- **保守规则纠错与编辑学习** — 应用确定性规则（`arion-rules-v4`）与用户批准的加密词库；在 macOS 原生编辑器中通过辅助功能观察上屏后的手动改词并在本机学习（[ADR 014](docs/decisions/014-edit-learning.md)）。
- **可选的本地或云端两阶段润色** — 在松开听写键后执行一次两阶段（术语选择器 + 场景格式化器）润色（[ADR 023](docs/decisions/023-final-only-polishing.md)、[ADR 024](docs/decisions/024-shared-polishing-pipeline.md)），支持场景感知格式化（**匹配当前应用**，[ADR 012](docs/decisions/012-context-scene-selection.md)）以及端侧光标、屏幕与历史术语提取（[ADR 013](docs/decisions/013-rewrite-context-on-device.md)）。本地润色在隔离的 `arion-llm`（`llama.cpp`）工作进程中运行 Qwen3.5-2B（`Q4_K_M` GGUF，[ADR 018](docs/decisions/018-staged-gguf-rewriting.md)、[ADR 025](docs/decisions/025-windows-local-polishing.md)）；可选的云端润色需显式开启并连接服务商，原始音频始终留在本地（[ADR 022](docs/decisions/022-cloud-polishing.md)）。
- **原生 Go 桌面外壳（FSD）** — 使用 [MyGo](https://github.com/egoist/mygo)（`v0.4.6`）并在 Go 中采用功能切片设计（[ADR 001](docs/decisions/001-desktop-native-boundaries.md)、[ADR 007](docs/decisions/007-sunbeam-design-system.md)），支持首次运行引导、本地诊断 ZIP 导出，以及可实时切换的英文 / 简体中文界面（[i18n](docs/frontend/i18n.md)）。
- **本地加密、可自托管中继与中心签名更新** — 个人词库本地 AES-256-GCM 加密（[ADR 005](docs/decisions/005-local-encryption.md)）；仓库内置支持 `noop` 与 `oidc` 门禁的 `services/sync` 密文中继服务（[ADR 015](docs/decisions/015-hosted-relay-account-authorization.md)）、桌面端 PKCE 登录（[账号登录](docs/project/account-signin.md)），以及由 `services/updates` 驱动的 Ed25519 签名更新通道（[ADR 019](docs/decisions/019-center-update-channels.md)、[deploy/center](deploy/center/README.md)）。

## 当前状态与限制

Arion 目前是一个可在 macOS（`arm64`）上正常工作的菜单栏听写应用，并提供 Windows（`amd64`）早期预览版（[发布状态](docs/operations/release-status.md)）。原生语音识别、本地润色与输入法核心原语已在开发机与 CI 上通过验证，但跨干净宿主环境与广泛第三方编辑器的完整验收仍在进行中。

尚未实现或仍待完成的事项：

- **本地候选排序（`RNK-01`）** — 排序契约与评测框架已存在，但当前识别器仅暴露单一假设结果，因此候选排序尚未启用。
- **桌面记忆同步客户端与设备注册（`MEM-01` / `SYNC-10`）** — `services/sync` 中继服务及桌面端 OIDC 登录视图/服务已实现，但桌面端 SQLite 发件箱（outbox）与设备注册仍待完成。
- **输入法与 Windows 宿主广泛验收（`IMK-01` / `TSF-01` / `RWT-RUN-01`）** — macOS `ArionInput.app`、Windows `ArionTSF.dll` 与 Windows 本地 GGUF 润色已通过隔离测试及开发机验证，但干净机器首次安装、完整编辑器兼容性矩阵以及 Windows 编辑学习仍待完成。
- **改写盲测人工评估与通用大语言模型纠错（`LRW-REVIEW-01` / `COR-01`）** — 开发基准测试结果已记录在 [`benchmarks/results`](benchmarks/README.md) 中，但盲测人工评分（`LRW-A06`）仍待完成；独立于润色流水线之外的通用 LLM 纠错未被采用。
- **Windows 发布者签名与正式通道升级验收（`REL-01`）** — macOS `arm64` Developer ID 代码签名、公证（notarization）及输入法描述文件校验已就绪（[部署文档](docs/operations/deployment.md)）；Windows Authenticode 发布者签名以及干净宿主上的旧版到新版通道升级验收仍待完成。

详细的验证证据与任务门禁请参阅[代理交接文档](docs/project/agent-handoff.md)、[发布状态](docs/operations/release-status.md)与[验证报告](docs/quality/validation.md)。

## 快速开始

### 前置条件

- **Go 1.27.1** 且启用 `cgo`（`go.work` 使用 Go 自动工具链选择；`packages/core` 与 `services/sync` 也会在 `GOWORK=off` 下独立检查）。
- **macOS（`arm64`）**：完整的 **Xcode 26.4.1** 应用（Swift 6.3.1）以及 **Metal Toolchain** 组件。仅安装 Command Line Tools 无法通过 `xcodebuild` 编译 `ArionASR`，首次准备请执行：
  ```sh
  sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
  sudo xcodebuild -license accept && sudo xcodebuild -runFirstLaunch
  xcodebuild -downloadComponent MetalToolchain
  ```
- **Windows（`amd64`）**：原生 C/C++ 工具链、CMake、用于构建原生键盘/cgo 的 MinGW-w64 GCC（`CC=x86_64-w64-mingw32-gcc`）以及构建安装程序所需的 NSIS。
- **Node 24**（`.nvmrc`）与**锁定版本的 `pnpm@10.33.0`**（`package.json`），用于仓库检查、评测工具及静态宣传站点（`apps/site`）。拒绝使用 `npm` 与 `yarn`。

### 构建与运行

```sh
make install        # 安装锁定版本的 pnpm 工作区依赖 (--frozen-lockfile)
make check          # 文档、JSON、Go 分层边界、core/sync 与评测测试
make desktop-check  # 桌面端测试、键盘检查与 go vet (cgo)
make desktop-dev    # 准备原生运行时并以开发模式启动 Arion Dev
make build desktop  # 在 apps/desktop/build/ 下打包 Arion.app + DMG (macOS) 或 NSIS 安装包 (Windows)
make e2e            # 在真实桌面会话中运行原生窗口生命周期 E2E 测试
make e2e-macos      # 针对打包后的 Arion.app 运行产品场景测试
make site-check     # 宣传站点 (apps/site)：i18n、类型、lint、测试与静态导出
```

首次运行 `make desktop-dev` 时，应用会打开引导界面，请求麦克风与辅助功能（Accessibility）权限，并将默认识别模型包下载至 `~/Library/Application Support/Arion/models/`（macOS）或 `%AppData%\Arion\models\`（Windows）。模型准备完成后，听写即可离线工作。运行 `make help` 可查看所有基准测试、评测与诊断命令。

### macOS 开发模式构建与 TCC 权限排障（`make desktop-dev`）

> 对应的英文运维文档见 [`docs/operations/local-dev.md` — macOS development build and TCC permission diagnostics](docs/operations/local-dev.md#macos-development-build-and-tcc-permission-diagnostics)。

`make desktop-dev` 在 macOS 上会通过 `xcodebuild` 编译原生 Swift/Metal 语音引擎，并在每次运行时对 `Arion Dev.app`（`si.astrafoundry.arion.dev`）执行 ad hoc 重签名。常见现象、机制原因与处理命令如下：

| # | 现象 / 错误输出 | 机制原因 | 处理命令与步骤 |
| --- | --- | --- | --- |
| **1** | `tool 'xcodebuild' requires Xcode, but active developer directory '/Library/Developer/CommandLineTools' is a command line tools instance` | 原生 Swift MLX 语音引擎（`ArionASR`）依赖完整 Xcode 工具链，仅安装 Command Line Tools 无法编译；新装 Xcode 还需完成首次初始化（否则返回 `firstLaunchExit=69`）。 | 安装 `/Applications/Xcode.app`（26.4.1+），执行：<br>`sudo xcode-select -s /Applications/Xcode.app/Contents/Developer && sudo xcodebuild -license accept && sudo xcodebuild -runFirstLaunch` |
| **2** | `cannot execute tool 'metal' due to missing Metal Toolchain` | Xcode 默认未包含独立的 Metal 着色器编译器组件（约 839 MB）。 | 执行 `xcodebuild -downloadComponent MetalToolchain`。 |
| **3** | 首次执行 `make desktop-dev` 时终端持续 2～5 分钟无输出 | `scripts/stage-asr-runtime.mjs` 调用 `xcodebuild -quiet`，首次冷编译需拉取 11 个 SwiftPM 依赖（`mlx-swift` 等）并编译 C++/Metal 内核，期间不输出逐行日志。 | 等待输出 `Staged native ASR for darwin-arm64.` 与 `✓ Built Arion Dev`，不要中断进程；后续运行会复用 `.workspace/native-asr-migration/build` 增量缓存。 |
| **4** | 系统设置中“辅助功能 / 输入监控”开关已开启，但应用仍报未授权；或按提示“退出并重新打开”后循环失效（日志报 `hold_register PERMISSION_DENIED`） | macOS TCC 将权限绑定在二进制的签名哈希（`CDHash`）上。每次执行 `make desktop-dev` 或触发热重载都会重新 ad hoc 签名，使旧开关绑定的 `CDHash` 失效。 | 1. 重置记录：`tccutil reset Accessibility si.astrafoundry.arion.dev && tccutil reset ListenEvent si.astrafoundry.arion.dev`<br>2. 在 **设置 › 通用** 点击 **授权**，并在系统设置中开启 **Arion Dev**。<br>3. 从右上角菜单栏退出 `Arion Dev`。<br>4. 直接用 `open` 启动终端打印的**已编译缓存包**（不要重新执行 `make desktop-dev`）：<br>`open "~/Library/Caches/Arion/desktop-dev/<checkout-hash>/darwin-arm64/Arion Dev.app"` |
| **5** | 使用 `open` 重开 `Arion Dev.app` 后，已授予的权限仍显示未授权 | 打开了工作区下旧版残留的 `apps/desktop/.mygo/dev/darwin-arm64/Arion Dev.app`，其 `CDHash` 与当前缓存目录中已授权的新包不一致。 | 始终打开启动时终端打印的 `~/Library/Caches/Arion/desktop-dev/<checkout-hash>/darwin-arm64/Arion Dev.app`，并删除残留的 `apps/desktop/.mygo` 目录。 |
| **6** | 执行 `open ".../Arion Dev.app"` 后终端无输出且无窗口弹出 | `open` 成功后默认静默返回 `0`；`Arion Dev` 为菜单栏常驻应用（`LSUIElement`，无 Dock 图标、完成引导后不主动弹出主窗口）。 | 查看屏幕右上角菜单栏图标，或执行 `pgrep -af "Arion Dev"` 确认进程状态。 |
| **7** | 系统设置“麦克风”列表中无 `Arion Dev`、无 `+` 按钮，点击“授权”仅跳转系统设置而不弹窗 | macOS 麦克风列表仅由系统原生授权弹窗写入；一旦此前拒绝过授权，系统不会再次弹窗。 | 执行 `tccutil reset Microphone si.astrafoundry.arion.dev`，用 `open` 重开 `Arion Dev.app`，在 **设置 › 通用**（非“麦克风”设备选择页）点击麦克风行的 **授权** 并允许系统弹窗。 |
| **8** | 屏幕录制引导无拖拽图标，或在文件选择器中找不到 `Arion Dev.app` | 屏幕录制引导条不提供拖拽磁贴，且 `~/Library/Caches` 在访达中默认隐藏；文件选择器顶部下拉菜单显示的是父级路径层级，而非当前目录内容。 | 在“系统设置 › 录屏与系统录音”点击 **`+`** ➔ 按 **`⌘⇧G`** ➔ 粘贴 `~/Library/Caches/Arion/desktop-dev/<checkout-hash>/darwin-arm64` ➔ 在文件列表中选中 `Arion Dev.app`（可按 `⌘2` 切换列表视图）➔ 开启开关后用 `open` 重开同一个包（若仅测试听写，也可关闭“屏幕内容”）。 |
| **9** | 听写后在 Cursor / VS Code 中改词未触发词库学习；或安装系统输入法报错 `ARTIFACT_UNVERIFIED` | 改词学习（[ADR 014](docs/decisions/014-edit-learning.md)）依赖辅助功能读取 `AXValue`，Electron 编辑器与终端不暴露该属性；系统输入法安装预检要求 Developer ID 签名与 `MACOS_IME_PROVISIONING_PROFILE`，ad hoc 开发包会拒绝安装。 | 在原生 Cocoa 编辑器（**备忘录**、**文本编辑**）中修改单个词并等待约 15 秒（或切走焦点）以验证改词学习；测试 `ArionInput.app` 需使用配置了签名凭据的 `make build desktop` 产物。 |

## 文档导航

### 项目与规范

- [代理编码指南 (`AGENTS.md`)](AGENTS.md) — 面向贡献者与编码代理的规则、边界与工作流。
- [文档索引 (`docs/README.md`)](docs/README.md) — 所有文档领域的顶层导航图。
- [产品范围 (`docs/project/README.md`)](docs/project/README.md) — 已确认的产品范围、明确边界与技术栈状态表。
- [系统架构 (`docs/project/architecture.md`)](docs/project/architecture.md) — 数据流、润色流水线、运行时形态与隐私边界。
- [仓库职责划分 (`docs/project/repository.md`)](docs/project/repository.md) — Go 工作区模块、FSD/DDD 包边界与工作规则。
- [产品规范 (`specs/README.md`)](specs/README.md) — 规范性需求（`001` 至 `009`）与验收场景 ID。
- [代理交接 (`docs/project/agent-handoff.md`)](docs/project/agent-handoff.md) — 已实现的里程碑、待办的 `P0`/`P1`/`P2` 任务及完成门禁。
- [架构决策记录 (`docs/decisions/README.md`)](docs/decisions/README.md) — ADR `001` 至 `025`。

### 工程、运维与评测

- [前端规范 (`docs/frontend/README.md`)](docs/frontend/README.md) — 原生 Go 中的功能切片设计（FSD）与[界面本地化 (`docs/frontend/i18n.md`)](docs/frontend/i18n.md)。
- [后端规范 (`docs/backend/README.md`)](docs/backend/README.md) — 面向 `packages/core` 与 `services/sync` 的领域驱动设计（DDD）规则。
- [本地开发 (`docs/operations/local-dev.md`)](docs/operations/local-dev.md) — 工具链、数据目录、环境变量与故障排查。
- [部署与发布门禁 (`docs/operations/deployment.md`)](docs/operations/deployment.md)、[发布状态 (`docs/operations/release-status.md`)](docs/operations/release-status.md)、[Arion 中心服务 (`deploy/center/README.md`)](deploy/center/README.md) 与[同步中继运维手册 (`deploy/sync/README.md`)](deploy/sync/README.md)。
- [质量与测试 (`docs/quality/README.md`)](docs/quality/README.md)、[验证报告 (`docs/quality/validation.md`)](docs/quality/validation.md) 与[原生 E2E (`docs/quality/e2e.md`)](docs/quality/e2e.md)。
- [研究证据 (`docs/research/README.md`)](docs/research/README.md)、[基准测试方法论 (`docs/research/benchmark-methodology.md`)](docs/research/benchmark-methodology.md)、[评测运行器 (`tools/evaluation/README.md`)](tools/evaluation/README.md) 与[基准测试结果 (`benchmarks/README.md`)](benchmarks/README.md)。

## 贡献者

[![AstraFoundry/Arion contributors](https://contrib.rocks/image?repo=AstraFoundry/Arion)](https://github.com/AstraFoundry/Arion/graphs/contributors)

## Star 历史

[![AstraFoundry/Arion Star History Chart](https://api.star-history.com/svg?repos=AstraFoundry/Arion&type=Date)](https://star-history.com/#AstraFoundry/Arion&Date)

## 许可证

Arion 由 [AstraFoundry](https://github.com/AstraFoundry) 开发，并基于 [AGPL-3.0](LICENSE) 许可证开源（[ADR 009](docs/decisions/009-agpl-license.md)）。请在 [GitHub Issues](https://github.com/AstraFoundry/Arion/issues) 中反馈问题。
