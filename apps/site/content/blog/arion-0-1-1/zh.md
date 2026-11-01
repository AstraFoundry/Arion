---
title: Arion 0.1.1：第一个正式版本
description: Arion 0.1.1 今天发布，支持 Apple 芯片 Mac 和 Windows x64。语音听写、润色、翻译、离线拼音键盘和剪贴板历史都在你的电脑上运行，以 AGPL-3.0 协议开源。
date: 2026-11-01
authors: stvlynn
category: announcement
tags: [发布, 语音输入, 拼音输入法, 离线]
image: /blog/arion-0-1-1/cover.webp
imageAlt: 模拟录屏：聊天窗口的输入框已放大，屏幕下方的 Arion 听写胶囊正显示一句英文草稿，右下角提示正按住右 Option 键
draft: true
---

今天我们发布 Arion 0.1.1，这是 Arion 的第一个正式版本。它支持 macOS 14 及以上的 Apple 芯片 Mac，以及 Windows x64，以 AGPL-3.0 协议开源，可以免费下载使用。

Arion 是一个本地优先的语音与拼音输入法。按住一个键说话，中英混杂的句子在你的电脑上完成识别，松开后整句写进光标所在的位置。同一个应用里还有可选的润色与语音翻译、离线全拼键盘，以及加密保存的剪贴板历史。

从第一个 alpha 到 0.1.1，我们发布了 21 个预览版本，和最早的一批用户一起在真实的编辑器里调整快捷键、候选框、润色规则和更新流程。0.1.1 汇总了这些改动，是我们推荐日常使用的第一个版本。下面按使用场景介绍它能做什么，最后一节写明它还没有做到的事。

<embed-frame src="/blog/arion-0-1-1/stage.zh.html#dictation" poster="./posters/dictation.zh.webp" title="演示：在聊天输入框里按住右 Option 听写" ratio="16/10" caption="在聊天输入框里按住右 Option 说话。草稿只出现在胶囊里，松开后整句写进输入框。模拟录屏，未录制任何声音。"></embed-frame>

## 按住说话，松开上屏

在任何可以打字的地方，按住听写键说话，松开即可。macOS 上默认是右 Option，Windows 上是右 Alt。轻按一下会进入免按住模式，一直听到你再按一次；说到一半按 <kbd>Esc</kbd> 会取消这次听写，不写入任何文字。听写键可以在设置里改成任意单键或组合键，录入时直接按下想用的键，和翻译、剪贴板快捷键冲突时会当场提示。

说话过程中，流式识别的草稿只显示在屏幕下方的胶囊里，不会写进你的文档。松开按键后，Arion 只向光标位置插入一次最终结果。如果这段时间里你切换了窗口、移动了焦点或取消了听写，旧的结果会被丢弃，不会被插到别处。当前焦点不是输入框时，结果会留在剪贴板上，方便你粘贴到想要的位置。

语音识别使用 Qwen3-ASR 1.7B，在 Apple 芯片上通过 MLX 以 8-bit 运行，在 Windows 上通过 speech.cpp 以 Q8_0 运行，支持 CPU 和 Vulkan。模型在首次使用前下载一次（Mac 约 2.5 GB，Windows 约 2.2 GB），逐个文件按固定的 SHA-256 摘要校验，之后听写完全离线。同一句话里中英混说，比如「把这个 PR merge 到 main 之前，先跑一下 CI」，不需要事先切换语言。

在开放办公区小声说话时，可以在设置里把软件输入增益从 −6 dB 调到 +12 dB。遇到密码框，Arion 会跳过上下文读取，也不会从中学习。

## 说完再整理

润色默认关闭。打开后，Arion 会在你松开按键后用本地语言模型把这句话整理一遍：去掉「嗯」「那个」这样的语气词，处理说到一半的改口，比如「周三，不对，周四」只保留「周四」，再按当前应用调整格式。

<embed-frame src="/blog/arion-0-1-1/stage.zh.html#polish" poster="./posters/polish.zh.webp" title="演示：在邮件里口述并润色" ratio="16/10" caption="在邮件正文里口述一段带语气词和改口的话。松开后，本地模型去掉被划掉的部分，并按邮件的格式写入问候和结尾。模拟录屏。"></embed-frame>

本地润色模型是 Qwen3.5-2B（Q4_K_M GGUF，约 1.3 GB），通过 llama.cpp 在 macOS 和 Windows 上运行。「按应用调整格式」开启时，Arion 会根据焦点所在的输入框类型、网页域名和应用标识，为每次听写选择一种场景：聊天、邮件、工作消息、给 AI 的提示词、搜索或列表。场景只决定版式，不改变内容；判断时只读取这些元数据，不读取输入框里的文字。[^scene]

标点有四种处理方式：智能标点、完整标点、省略句末标点，以及中文界面下的「以空格代替标点」。你也可以允许润色参考光标附近和当前窗口里的文字，用来纠正同音的人名和术语。这些内容通过系统辅助功能接口读取，不截屏，并且只在本机使用。[^context]

润色只在松开按键之后运行一次。我们曾在说话过程中实时显示润色预览，但实测中这些预览总会被最终结果覆盖，从未改变过最后插入的文字，所以去掉了这一步，省下了说话时的算力。[^final]

如果你更想用云端模型，可以登录 ChatGPT，或填入 OpenAI、Anthropic、Google Gemini、DeepSeek、OpenRouter 的 API Key。只有在你主动选择云端引擎、连接服务商并选定模型之后，Arion 才会发送请求，内容仅限识别后的文字、当前场景和召回的词库条目。音频、光标和窗口上下文、听写历史都留在本机。每个服务商的 Key 分开保存在系统凭据库里，切换服务商不会互相覆盖。

## 换一个键，直接翻译

语音翻译有自己的快捷键：macOS 上默认是右 Command，Windows 上是右 Ctrl。按住它用中文说话，Arion 会先识别原文、纠正同音词和专有名词，再写出目标语言的句子。

<embed-frame src="/blog/arion-0-1-1/stage.zh.html#translate" poster="./posters/translate.zh.webp" title="演示：按住右 Command 用中文说话，写入英文回复" ratio="16/10" caption="同事用英文提问。按住右 Command 用中文回答，胶囊里显示中文草稿，松开后输入框里是英文译文。模拟录屏。"></embed-frame>

目标语言可以选 12 种：English、简体中文、繁體中文、日本語、한국어、Español、Français、Deutsch、Português、Italiano、Русский 和 العربية。翻译和润色共用同一套本地或云端模型，所以即使关闭了润色，按下翻译键时也会调用它。如果模型缺失、超时或输出没有通过校验，这次翻译会直接失败并在胶囊里提示，不会把没翻译的原文写进对话框；原文只保留在这次会话的内存里。你在译文上做的修改不会被当作纠错学习，以免目标语言的写法混进你的中文词库。

## 越用越准

词库按「正确的写法」组织：每个词条下面列出它常被识别成的错误形式；安装了 Arion 输入法时，还可以直接编辑这个词的拼音读音。只填写正确写法的词条会作为热词交给本地语音识别；符合条件的已学纠错也会同步到语音识别和拼音键盘的词库里。新添加的词条排在最前面。

在 macOS 的原生编辑器里，如果你手动改掉了 Arion 刚插入的某个词，它会记住这次修改，下次自动用你的写法，并在胶囊里给出撤销按钮。

<dictation-pill state="learned" term="Arion" label="Arion 的听写胶囊提示已记住 Arion 这个词，并带有撤销按钮" caption="改掉错字之后，胶囊提示已记住这个词，可以一键撤销。"></dictation-pill>

词库、学到的纠错和剪贴板历史在写入磁盘前都用 AES-256-GCM 加密，密钥在本机生成，macOS 上存放在钥匙串中，Windows 上由 DPAPI 保护。密钥缺失或校验失败时，Arion 会直接报错，不会退回明文保存。[^privacy]

## 同一个输入法里的拼音键盘

Arion 自带一个离线的全拼输入法：macOS 上是 IMK 输入法，Windows 上是 TSF 输入法，两端共用同一套 Rime 方案、候选排序和学习逻辑。轻按 <kbd>Shift</kbd> 切换中英文，光标旁的标识会翻转，显示当前模式。macOS 的输入法菜单里还可以切换全角字符和英文标点，<kbd>Ctrl</kbd> <kbd>.</kbd> 切换标点。

<embed-frame src="/blog/arion-0-1-1/stage.zh.html#keyboard" poster="./posters/keyboard.zh.webp" title="演示：拼音候选按前文重排，再轻按 Shift 切换英文" ratio="16/10" caption="在「我们在婚礼上许下」之后输入 shi yan。候选框先按原顺序出现，本地排序参考前文把「誓言」提到第一位；轻按 Shift 后，光标旁的标识从「中」翻成「英」。模拟录屏。"></embed-frame>

候选排序会参考你刚才写了什么。最基础的一层只记录最近 120 秒内、同一个应用里确认上屏的前后词搭配，一次确认最多把候选往前提两位，避免一次误选就占据首位。在 50 对同音词的测试中，从第二次出现开始，只靠词频学习时首选命中率是 32.2%，加上前后词搭配后是 88.7%。[^imk]

在此之上，你可以在设置里打开本地神经排序。它分两路运行：一个约 4.5 MB 的小模型在每次输入变化时立即重排，预算 100 ms；停顿 150 ms 后，本地润色模型再参考光标前后和当前窗口的文字做一次判断，预算 900 ms。候选框总是先按原顺序立刻弹出，两路模型都只能调整已有候选的顺序，不会生成新的词。macOS 上这些上下文通过辅助功能接口读取，遇到密码框和拒绝名单里的应用时不读取；Windows 上目前只参考你最近确认上屏的文字。[^ranking]

键盘还支持中文模式下的英文单词和前缀补全、完整拼音词的 Emoji 候选，以及八种候选高亮色和自定义颜色。macOS 的输入法菜单里有「语音输入」，可以直接开始听写。

## 剪贴板历史

剪贴板历史默认关闭，打开后会记录你复制的文字、链接、图片和文件引用。按 <kbd>⌘</kbd> <kbd>⇧</kbd> <kbd>V</kbd>（Windows 上是 <kbd>Ctrl</kbd> <kbd>Shift</kbd> <kbd>V</kbd>）会在光标旁打开一个紧凑的选择器，可以搜索、置顶，鼠标停留在图片或长文本上会显示预览。按回车会把选中的内容粘贴到原来的输入框；找不到输入框时，则复制到剪贴板。

<embed-frame src="/blog/arion-0-1-1/stage.zh.html#clipboard" poster="./posters/clipboard.zh.webp" title="演示：用剪贴板历史搜索并粘贴一条命令" ratio="16/10" caption="在光标处打开剪贴板历史，鼠标停在截图上显示预览，搜索 make 后按回车，把命令粘贴回文档。模拟录屏。"></embed-frame>

文件只保存引用，不复制文件内容；文件夹不会被记录。历史默认保留 200 条、总共 100 MB，超出时从最早的未置顶记录开始清理，不按时间过期。整个历史用独立的密钥加密保存。听写过程中 Arion 为了插入文字而临时使用的剪贴板，不会进入历史。[^clipboard]

## 更新

Arion 的更新包都带有 Ed25519 签名，安装前在本机校验。默认只接收稳定版；在「设置 › 关于」里打开「允许测试版更新」后，才会收到预发布版本。自动检查和自动下载是两个独立开关，自动下载默认关闭。下载好的更新会等你主动点重启，并避开正在进行的输入。有新版本时，提醒只在听写结束后的空闲时刻出现，可以稍后处理或关闭，系统通知遵循你在操作系统里的设置。

## 你的数据留在哪里

音频只在听写进行时保存在内存里，结束即释放，不会写成临时文件。识别草稿和润色结果不写入诊断日志。

Arion 没有云端语音识别。模型下载完成后，日常听写和拼音输入不需要联网。会联网的只有这几类请求：下载模型、检查更新、你主动选择的云端润色，以及登录账户后的产品通知。登录并明确同意后，Arion 还会上报按天汇总的用量计数，内容只有日期、听写次数、麦克风时长和字数，不包含录音、文字、场景或应用信息。完整的数据流向见[《你的听写去了哪里》](/blog/where-your-dictation-goes/)。

## 开始使用

在[下载页](/#download)选择你的平台。macOS 版需要 macOS 14 或更高版本和 Apple 芯片；Windows 版是 x64 安装包。首次启动后，引导流程会请求所需的系统权限，带你设置听写键、下载识别模型，并可选地打开润色和语音翻译。

已经安装 alpha 版本的用户，可以在「设置 › 关于」里检查更新；如果没有收到 0.1.1，请从下载页手动安装一次。

想从源码构建，可以 clone 仓库后运行：

```sh
make install
make build desktop
```

## 还没做到的事

我们在每一份发布说明里都写明了验收范围，这一版也一样：

- **多设备同步还没有推出。** 加密边界已经写进规范：每条词库记录在上传前先在本机加密，中继服务只能看到密文。桌面端的设备注册尚未完成。
- **纠错学习目前只支持 macOS 原生编辑器。** Windows 上可以手动维护词库。
- **Windows 键盘的上下文排序只参考最近上屏的文字**，还不读取光标和窗口内容。
- **英语和简体中文以外的 10 个翻译目标语言没有单独的评测记录**，它们与英语共用规则。云端润色的质量和延迟也还没有基准测试。
- **Intel Mac 不受支持**（[ADR 021](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/021-apple-silicon-only-macos.md)）。

如果遇到问题，欢迎在 [GitHub Issues](https://github.com/AstraFoundry/Arion/issues) 反馈，或者加入 [Discord](https://discord.com/invite/72kh465mMm) 和[飞书群](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae)。我们如何决定哪些改动可以上线，写在[《我们为什么像做实验一样做语音输入法》](/blog/building-like-an-experiment/)里。

感谢每一位在 alpha 阶段提交问题、录制复现、提出建议的用户和贡献者。

[^scene]: [规范 009：上下文润色](https://github.com/AstraFoundry/Arion/blob/main/specs/009-contextual-rewrite.md#scene-selection)中的场景选择（`SCN-01`、`SCN-02`）。

[^context]: [ADR 013：在本机读取润色上下文](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/013-rewrite-context-on-device.md)。

[^final]: [ADR 023：只在说完后润色](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/023-final-only-polishing.md)。

[^privacy]: [规范 005：隐私与信任边界](https://github.com/AstraFoundry/Arion/blob/main/specs/005-privacy.md)与 [ADR 005：本地加密](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/005-local-encryption.md)。

[^imk]: 测试方法与结果见[《我们为什么像做实验一样做语音输入法》](/blog/building-like-an-experiment/)中「同一次运行里同时测质量与延迟」一节。

[^ranking]: [ADR 036：两阶段本地键盘排序](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/036-two-stage-keyboard-ranking.md)与 [ADR 037](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/037-keyboard-context-and-developer-diagnostics.md)。文中的 100 ms 和 900 ms 是配置的预算上限，对应的实际延迟还没有测量。

[^clipboard]: [规范 010：剪贴板历史](https://github.com/AstraFoundry/Arion/blob/main/specs/010-clipboard.md)与 [ADR 034](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/034-clipboard-history.md)。
