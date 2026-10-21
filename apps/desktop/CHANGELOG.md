## 0.1.0-alpha.6

### English

macOS Apple silicon (ARM64) preview, requiring macOS 14 or later.

- Fix standalone Shift language switching when an editor delivers duplicate modifier events. Preserve the short-tap deadline and chord cancellation; the fix uses shared IMK handling and was accepted locally in Codex and Cursor.
- Validate Input Monitoring consent and actual listener readiness for Right Option dictation. Recover after consent changes, retire native listener threads safely, and reject stale queued key events. See the keyboard shortcut incident report linked from RELEASE-STATUS.md for stale macOS permission-entry recovery.
- Improve input-source setup and activation, and include native Qwen3-ASR workers with accessibility context plus rewriting improvements that preserve recognized names.
- Move update checks and download discovery to the Arion center. Prereleases require Allow beta updates and publication to the center beta channel; GitHub publication alone does not offer this version through the center updater.

This is an early preview. Clean-machine installation, microphone/editor compatibility, minimum OS/RAM and old-to-new update acceptance remain open. Local acceptance on the development Mac does not establish every editor. Desktop encrypted sync enrollment is pending. Model weights are downloaded separately; inference runs locally after provisioning. No Intel or Windows installers are included.

See RELEASE-STATUS.md for evidence and limitations, and SHA256SUMS for file verification.

### 中文

适用于 macOS Apple 芯片（ARM64）的预览版本，需要 macOS 14 或更高版本。

- 修复编辑器重复发送修饰键事件时，轻按 Shift 无法切换中英文的问题。保留轻按时限和组合键取消规则；修复作用于通用 IMK 路径，已在本机 Codex、Cursor 中验收。
- 右 Option 录音快捷键会检查输入监控权限及监听器实际状态；支持权限变更后恢复、安全结束原生监听线程，并丢弃旧监听器排队的按键事件。macOS 旧权限条目的恢复方式见 RELEASE-STATUS.md 链接的故障报告。
- 改进输入法安装和激活，包含原生 Qwen3-ASR 语音引擎、辅助功能上下文，以及保留已识别名称的润色改进。
- 更新检查与下载发现迁移到 Arion 中心。测试版需开启“允许测试版更新”，并发布到中心 beta 渠道；仅发布 GitHub Release 不会让中心更新器提供此版本。

这是早期预览版。全新机器安装、麦克风与编辑器兼容性、最低系统和内存要求，以及旧版本升级流程仍待验收。开发机器上的本地验收不代表所有编辑器均已验证。桌面加密同步的设备注册尚未完成。模型权重需单独下载，准备完成后在本机推理。本版不提供 Intel 或 Windows 安装包。

验证记录和限制见 RELEASE-STATUS.md，文件校验值见 SHA256SUMS。

## 0.1.0-alpha.5

### English

macOS Apple silicon (ARM64) preview, requiring macOS 14 or later.

- Add an independent **Allow beta updates** option in Settings → About. It defaults to off and is saved across restarts. Enabling it includes published prereleases for this platform; drafts remain excluded.
- Show an empty update channel as a normal state, distinguish full alpha versions, and discard pending results when the update channel changes. Downloads still use the built-in Ed25519 verification and MyGo installer.
- Include the committed improvements to spoken quantity formatting, technical spelling and caption alignment.

This is an early preview. System input-method activation, clean-machine microphone/editor compatibility, minimum-memory and old-to-new update acceptance remain open. Desktop encrypted sync enrollment is pending. Model weights are downloaded separately; inference runs locally after provisioning. This release includes no Intel or Windows installers.

See RELEASE-STATUS.md for evidence and SHA256SUMS for file verification. Older alpha builds without this switch still use the stable channel; install this version manually to enable beta updates.

### 中文

适用于 macOS Apple 芯片（ARM64）的预览版本，需要 macOS 14 或更高版本。

- 在“设置 → 关于”中新增独立的“允许测试版更新”选项，默认关闭，重启后保留。开启后可接收适用于当前平台的已发布测试版，始终排除草稿。
- 当前渠道暂无版本时显示正常状态；准确比较完整 alpha 版本号，切换渠道时清除旧结果。下载仍使用内置 Ed25519 签名验证及 MyGo 安装机制。
- 包含已提交的口语数字规范化、技术术语拼写和字幕对齐改进。

这是早期预览版。系统输入法激活、全新机器上的麦克风与编辑器兼容性、最低内存要求及旧版本升级流程仍待验收。桌面加密同步的设备注册尚未完成。模型权重需单独下载，准备完成后在本机推理。本版不提供 Intel 或 Windows 安装包。

验证记录见 RELEASE-STATUS.md，文件校验值见 SHA256SUMS。此前没有此开关的 alpha 版本仍使用稳定渠道，请手动安装本版后再开启测试版更新。
