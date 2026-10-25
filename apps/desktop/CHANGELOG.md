## 0.1.0-alpha.14

### English

Preview for macOS Apple silicon (macOS 14 or later) and Windows x64.

- Update dictation-key onboarding with an animated native laptop illustration, a key picker and visual key-press feedback.
- Share the illustration and animation implementation across macOS and Windows.
- Include the latest main-branch onboarding changes in versioned release installers and signed update archives.

macOS release packages require Developer ID signing and Apple notarization. Windows publisher signing remains unavailable for this preview. Model weights download separately. GitHub publication and release-storage staging do not publish the Arion center beta channel. See RELEASE-STATUS.md and SHA256SUMS for acceptance limits and verification.

### 中文

适用于 macOS Apple 芯片（macOS 14 或更高版本）和 Windows x64 的预览版本。

- 更新听写按键引导，加入原生笔记本动画、按键选择和按下反馈。
- macOS 和 Windows 共用插图与动画实现。
- 将最新 main 分支的引导改进纳入新版安装包及带签名的更新归档。

macOS 发布包要求 Developer ID 签名和 Apple 公证；Windows 预览版仍无发布者签名。模型权重单独下载。GitHub 发布及存储暂存不会自动发布 Arion 中心的 beta 渠道。验收范围及校验信息见 RELEASE-STATUS.md 与 SHA256SUMS。

## 0.1.0-alpha.13

### English

Preview for macOS Apple silicon (macOS 14 or later) and Windows x64.

- Add Windows offline Pinyin and TSF voice input, with shared Rime schemas, candidate ordering and learning logic across macOS and Windows.
- Repair Windows candidate layout refresh after composition changes, input-method registration and model readiness reporting.
- Add a stable Windows input-method loader and versioned cores. Compatible active cores remain usable until safe release; legacy components already loaded in applications may require those applications to fully exit once.
- Enable Windows local GGUF polishing with a bundled CPU runtime and protected worker lifecycle.
- Fix Windows overlay transparency and background worker console windows, and improve the EXE installer and onboarding flow.
- Extend native input-method, package, update and shared-engine regression coverage.

This is an early preview. Windows publisher signing, clean-machine acceptance and broad editor compatibility remain open. Installed owned-editor candidate checks pass; this does not establish seamless upgrades in every already-running application. Model weights download separately. GitHub publication and release-storage staging do not publish the Arion center beta channel. See RELEASE-STATUS.md and SHA256SUMS.

### 中文

适用于 macOS Apple 芯片（macOS 14 或更高版本）和 Windows x64 的预览版本。

- 新增 Windows 离线拼音和 TSF 语音输入，两端共用 Rime 方案、候选排序和学习逻辑。
- 修复 Windows 组合输入后的候选框布局刷新、输入法注册和模型就绪状态判断。
- 新增固定的 Windows 输入法加载入口与版本化核心。兼容的活动核心可继续使用直到安全释放；已经加载早期旧组件的应用可能需要完全退出一次。
- 支持 Windows 本地 GGUF 润色，随包提供 CPU 运行时并完善工作进程生命周期。
- 修复 Windows 悬浮窗透明度及后台工作进程的控制台窗口，改进 EXE 安装器和首次使用流程。
- 扩展原生输入法、打包、更新和共享引擎回归测试。

这是早期预览版。Windows 发布者签名、全新机器和广泛编辑器兼容性仍待验收。已安装版本在自有测试编辑器的候选框检查通过，但不代表所有运行中应用均可无感升级。模型权重单独下载；GitHub 发布及存储暂存不会自动发布 Arion 中心的 beta 渠道。详见 RELEASE-STATUS.md 与 SHA256SUMS。

## 0.1.0-alpha.12

### English

Preview for macOS Apple silicon (macOS 14 or later) and Windows x64.

- Include the Windows installer and cryptographically signed update archive in alpha releases. The Windows application and installer do not have an Authenticode publisher signature; Windows security policies may warn or block execution.
- Protect Windows text delivery against stale sessions, cancellation, changed focus/selection and standard password fields; stop remaining input batches when a session loses ownership.
- Repair Windows UI Automation provider initialization and refuse context reads when field safety cannot be established.
- Give Windows accessibility initialization a separate bounded startup phase while preserving the 1.5-second window traversal limit.
- Fix update rollback leaving newly added files or directories behind when another destination is locked.
- Remove the MSVC OpenMP redistributable dependency from the Windows CPU speech worker and reject unbundled runtime DLLs during packaging.
- Reload the same speech runtime for a new request after cancellation times out and retires its process; cancelled requests are never replayed.
- Add Windows native, installed-application speech, microphone and updater regression coverage. Actual CPU/Vulkan recognition and five installed-product speech scenarios pass on the documented development host.

This is an early preview. Mixed-language technical terms still have recognition errors. Windows TSF and local polishing are unsupported. Clean-machine/editor compatibility and a real production-channel old-to-new upgrade remain open. Model weights download separately. GitHub publication and release-storage staging do not publish the Arion center beta channel. See RELEASE-STATUS.md and SHA256SUMS for limits and verification.

### 中文

适用于 macOS Apple 芯片（macOS 14 或更高版本）和 Windows x64 的预览版本。

- Alpha 发布新增 Windows 安装包及带加密签名的更新归档。Windows 程序和安装包尚无 Authenticode 发布者签名，系统安全策略可能警告或阻止运行。
- 加强 Windows 文本输入保护：拒绝过期会话、取消、焦点或选区变化及标准密码框；会话失去目标所有权时停止后续输入批次。
- 修复 Windows UI Automation 提供程序初始化，并在无法确认字段安全时拒绝读取上下文。
- 为 Windows 辅助功能初始化设置独立、有界的启动阶段，同时保留 1.5 秒的窗口遍历限制。
- 修复更新过程中目标文件被锁定时，回滚遗留新增文件或目录的问题。
- 移除 Windows CPU 语音运行时对 MSVC OpenMP 可再发行库的依赖，并在打包时检查未随包提供的运行库 DLL。
- 修复取消超时回收语音进程后无法继续识别的问题：新请求重新加载同一运行时，已取消请求不会重放。
- 新增原生交互、安装后语音、麦克风及更新回归测试；CPU/Vulkan 真实识别与五项安装后语音场景已在记录的开发机通过。

这是早期预览版。中英混合技术词仍有识别错误；Windows TSF 和本地润色尚不支持。全新机器、第三方编辑器兼容性及生产渠道旧版本升级仍待验收。模型权重需单独下载。GitHub 发布及存储暂存不会自动发布 Arion 中心的 beta 渠道。验证限制和校验值见 RELEASE-STATUS.md 与 SHA256SUMS。

## 0.1.0-alpha.11

### English

macOS Apple silicon (ARM64) preview, requiring macOS 14 or later.

- Update the local polishing cleanup prompt, preserving separate scene and language rules and isolating source text from context.
- Add polishing punctuation choices: smart, full, omit sentence-final punctuation, or spaces instead of punctuation. Apply the selected mode to live previews and final text.
- Validate raw polishing output before punctuation conversion so control markers cannot be hidden. Reject conversions that leave only whitespace and keep rejected output out of the preview cache.

This is an early preview. Clean-machine installation, real-service cloud sign-in/provider requests, minimum OS/RAM and old-to-new updates still need acceptance. Recognition stays local and works offline after model provisioning; model weights are downloaded separately. Desktop encrypted sync enrollment remains pending. This alpha includes only the macOS ARM64 installer.

See RELEASE-STATUS.md for evidence and limitations, and SHA256SUMS for file verification. GitHub publication and COS staging do not publish the Arion center beta channel.

### 中文

适用于 macOS Apple 芯片（ARM64）的预览版本，需要 macOS 14 或更高版本。

- 更新本地润色的整理提示词，保留按场景和语言划分的规则，并隔离原文与上下文。
- 新增润色标点选项：智能标点、完整标点、省略句末标点、以空格代替标点；所选模式同时应用于实时预览和最终文本。
- 在标点转换前校验润色原始输出，避免控制标记被转换隐藏；拒绝转换后仅剩空白的结果，并防止被拒绝的输出进入预览缓存。

这是早期预览版。全新机器安装、真实云端登录及服务商请求、最低系统和内存要求、旧版本升级仍待验收。语音识别在本机完成，模型准备完成后可离线使用；模型权重需单独下载。桌面加密同步的设备注册尚未完成。本次 alpha 仅提供 macOS ARM64 安装包。

验证记录和限制见 RELEASE-STATUS.md，文件校验值见 SHA256SUMS。GitHub 发布及 COS 暂存不会自动发布 Arion 中心的 beta 渠道。

## 0.1.0-alpha.10

### English

macOS Apple silicon (ARM64) preview, requiring macOS 14 or later.

- Add Voice input to the macOS input menu, showing the configured dictation key. It toggles dictation while Arion is connected. Group the menu actions and add Open Arion.
- Add full-width character and English punctuation switches to the Chinese input menu. Ctrl+. switches punctuation; the input method remembers both choices.
- Fix input-source removal so reinstalling does not wait on a stale disabled state. Removal switches away, waits for the host to exit and deletes the bundle without disabling the source.
- Mark Settings sections that need attention with a yellow dot for permissions, the dictation key, recognition models or a pending update.
- Add embedded GitHub contributors and Discord/Feishu community links to Settings > About.

This is an early preview. Clean-machine installation, real-service cloud sign-in/provider requests, minimum OS/RAM and old-to-new updates still need acceptance. Recognition stays local and works offline after model provisioning; model weights are downloaded separately. Desktop encrypted sync enrollment remains pending. This alpha includes only the macOS ARM64 installer.

See RELEASE-STATUS.md for evidence and limitations, and SHA256SUMS for file verification. GitHub publication and COS staging do not publish the Arion center beta channel.

### 中文

适用于 macOS Apple 芯片（ARM64）的预览版本，需要 macOS 14 或更高版本。

- macOS 输入法菜单新增“语音输入”，显示当前听写快捷键；Arion 已连接时可切换听写。菜单按功能分组，并新增“打开 Arion”。
- 中文输入菜单新增全角字符及英文标点开关，Ctrl+. 可切换标点；输入法会记住所选状态。
- 修复卸载输入源后重新安装等待过期禁用状态的问题。卸载时先切换到其他输入源，等待输入法进程退出，再删除安装包，不再禁用输入源。
- 设置侧栏用黄色圆点标记需要处理的权限、听写快捷键、识别模型和待安装更新。
- 设置 > 关于新增内置的 GitHub 贡献者列表，以及 Discord、飞书社区入口。

这是早期预览版。全新机器安装、真实云端登录及服务商请求、最低系统和内存要求、旧版本升级仍待验收。语音识别在本机完成，模型准备完成后可离线使用；模型权重需单独下载。桌面加密同步的设备注册尚未完成。本次 alpha 仅提供 macOS ARM64 安装包。

验证记录和限制见 RELEASE-STATUS.md，文件校验值见 SHA256SUMS。GitHub 发布及 COS 暂存不会自动发布 Arion 中心的 beta 渠道。

## 0.1.0-alpha.9

### English

macOS Apple silicon (ARM64) preview, requiring macOS 14 or later.

- The dictation pill starts as wide as the finishing hint and widens with the draft in steps, up to a maximum. It does not narrow during a session, so a shorter revision leaves the width alone. Short replies stay a short pill.
- Settings > Keyboard > Highlight color offers yellow (the default), orange, red, pink, purple, blue, green, graphite, and a custom color. The candidate bar fills the selection with that color and uses white or dark text for contrast.

This is an early preview. Clean-machine installation, real-service cloud sign-in/provider requests, minimum OS/RAM and old-to-new updates still need acceptance. Recognition stays local and works offline after model provisioning; model weights are downloaded separately. Desktop encrypted sync enrollment remains pending. This alpha includes only the macOS ARM64 installer.

See RELEASE-STATUS.md for evidence and limitations, and SHA256SUMS for file verification. GitHub publication and COS staging do not publish the Arion center beta channel.

### 中文

适用于 macOS Apple 芯片（ARM64）的预览版本，需要 macOS 14 或更高版本。

- 听写胶囊从完成提示的宽度起步，随草稿按步进加宽，直到上限。同一次听写中不会变窄，较短的修订不会把胶囊收回。短句仍是短胶囊。
- 设置 > 键盘 > 高亮颜色提供黄色（默认）、橙色、红色、粉色、紫色、蓝色、绿色、石墨色，以及自定义颜色。候选栏用所选颜色填充当前候选，并按对比度使用白色或深色文字。

这是早期预览版。全新机器安装、真实云端登录及服务商请求、最低系统和内存要求、旧版本升级仍待验收。语音识别在本机完成，模型准备完成后可离线使用；模型权重需单独下载。桌面加密同步的设备注册尚未完成。本次 alpha 仅提供 macOS ARM64 安装包。

验证记录和限制见 RELEASE-STATUS.md，文件校验值见 SHA256SUMS。GitHub 发布及 COS 暂存不会自动发布 Arion 中心的 beta 渠道。

## 0.1.0-alpha.8

### English

macOS Apple silicon (ARM64) preview, requiring macOS 14 or later.

- Animate the input method's Chinese/English caret badge when switching modes. The previous label folds away and the selected mode unfolds in its place; Reduce Motion skips the flip.
- Add a scrolling feature tour to the marketing site, with product demonstrations for dictation, polishing, mixed-language input, keyboard learning and offline use.

This is an early preview. Clean-machine installation, real-service cloud sign-in/provider requests, minimum OS/RAM and old-to-new updates still need acceptance. Recognition stays local and works offline after model provisioning; model weights are downloaded separately. Desktop encrypted sync enrollment remains pending. This alpha includes only the macOS ARM64 installer.

See RELEASE-STATUS.md for evidence and limitations, and SHA256SUMS for file verification. GitHub publication and COS staging do not publish the Arion center beta channel.

### 中文

适用于 macOS Apple 芯片（ARM64）的预览版本，需要 macOS 14 或更高版本。

- 输入法切换中英文时，光标旁的语言标记加入翻转动画：原标签折叠后展开为所选模式；开启“减弱动态效果”时跳过翻转。
- 官网新增随滚动切换的功能导览，展示听写、润色、中英混输、键盘学习和离线使用。

这是早期预览版。全新机器安装、真实云端登录及服务商请求、最低系统和内存要求、旧版本升级仍待验收。语音识别在本机完成，模型准备完成后可离线使用；模型权重需单独下载。桌面加密同步的设备注册尚未完成。本次 alpha 仅提供 macOS ARM64 安装包。

验证记录和限制见 RELEASE-STATUS.md，文件校验值见 SHA256SUMS。GitHub 发布及 COS 暂存不会自动发布 Arion 中心的 beta 渠道。

## 0.1.0-alpha.7

### English

macOS Apple silicon (ARM64) preview, requiring macOS 14 or later.

- Add optional cloud polishing through ChatGPT sign-in or an API key for OpenAI, Anthropic, Google Gemini, DeepSeek or OpenRouter. Local polishing remains the default. Cloud requests require an explicit engine choice, a connected provider and a selected model; they send final text, its scene and recalled vocabulary only. Audio, cursor/window context and dictation history stay on the device. Credentials use the OS credential store.
- Add guided cloud-provider and model setup in onboarding and Settings. Keep model settings on one section, open recognition and cloud models on their own pages, and show explanations in tooltips. Give update controls their own row.
- Refine the dictation mark: its bars fold into waiting dots, which orbit while polishing runs.
- Retire Intel macOS support for future releases. Supported development targets are macOS Apple silicon and Windows x64; this alpha includes only the macOS ARM64 installer.

This is an early preview. Real-service cloud sign-in/provider requests, clean-machine installation, minimum OS/RAM and old-to-new updates still need acceptance. Cloud polishing has no benchmarked quality or latency claim. Recognition stays local and works offline after model provisioning; model weights are downloaded separately. Desktop encrypted sync enrollment remains pending.

See RELEASE-STATUS.md for evidence and limitations, and SHA256SUMS for file verification. GitHub publication and COS staging do not publish the Arion center beta channel.

### 中文

适用于 macOS Apple 芯片（ARM64）的预览版本，需要 macOS 14 或更高版本。

- 新增可选的云端润色：可登录 ChatGPT，或使用 OpenAI、Anthropic、Google Gemini、DeepSeek、OpenRouter 的 API 密钥。本机润色仍为默认选项。只有主动选择云端、连接服务商并选定模型后才会发送请求；仅发送最终文本、场景和召回的词汇。音频、光标周边及窗口上下文、听写历史均保留在本机，凭据存入系统凭据库。
- 在引导和设置中加入分步的云端服务商及模型配置。模型设置保留在同一分区，识别模型和云端模型分别进入详情页，说明使用提示框呈现；更新控制独占一行。
- 改进听写标记：竖条折叠为等待圆点，润色期间圆点环绕运动。
- 后续版本不再支持 Intel Mac。开发支持范围为 macOS Apple 芯片与 Windows x64；本次 alpha 仅提供 macOS ARM64 安装包。

这是早期预览版。真实云端登录及服务商请求、全新机器安装、最低系统与内存要求、旧版本升级仍待验收，尚无云端润色质量或延迟的基准结论。语音识别仍在本机完成，模型准备完成后可离线使用；模型权重需单独下载。桌面加密同步的设备注册尚未完成。

验证记录和限制见 RELEASE-STATUS.md，文件校验值见 SHA256SUMS。GitHub 发布及 COS 暂存不会自动发布 Arion 中心的 beta 渠道。

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
