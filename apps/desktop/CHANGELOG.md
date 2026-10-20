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
