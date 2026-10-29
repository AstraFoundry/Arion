---
title: 你的听写去了哪里
description: Arion 在你的电脑上识别语音。这篇文章说明哪些内容留在设备上，哪几件事会用到网络，以及你如何自己验证。
date: 2026-10-27
category: product
tags: [隐私, 离线, 加密]
image: ../../../public/backdrops/flowers-wide-1600.webp
imageAlt: 一张柔和、带运动模糊的花瓣飘落照片
---

语音输入开着的时候，会听到你说的每一句话。所以比任何功能都重要的问题是：音频去了哪里？Arion 的答案很短：留在你的电脑上，由它变成的文字也一样。

## 留在设备上的内容

按下 <kbd>⌥</kbd> <kbd>Space</kbd> 后，Arion 把声音录进内存，用你电脑上的模型识别，再把结果输入到光标所在的位置。音频只为正在进行的这次听写保留，结束即丢弃。草稿和更正也是如此。

<embed-frame src="/blog/where-your-dictation-goes/shortcut.html" title="试一下快捷键" ratio="16/7" caption="在框内按 ⌥ Space 或点击，开始和结束一次模拟听写。不会录制任何声音。"></embed-frame>

你教给 Arion 的词也留在本机。更正一个它输入的词，下次就会自动应用，并且可以撤销：

<dictation-pill state="learned" term="Arion" label="Arion 的听写胶囊提示已学会 Arion 这个词，并带有撤销按钮" caption="更正之后的胶囊。学会的词加密保存在设备上。"></dictation-pill>

已确认和学会的词汇用 AES-256-GCM 加密。密钥在设备上生成，macOS 上保存在登录钥匙串中，Windows 上由 DPAPI 保护。[^keys]

## 会用到网络的地方

只有三件事，而且都不包含你说的话：

| 请求 | 发送什么 | 何时 |
| --- | --- | --- |
| 下载模型 | 请求公开的模型文件，每个文件都用固定的 SHA-256 摘要校验 | 首次使用前一次 |
| 检查更新 | 不含任何内容或设备标识 | 自动进行，可以关闭 |
| 登录与云端润色 | 识别出的文字和已启用的上下文，从不发送音频 | 仅在你登录或连接服务商时 |

模型下载完成后，听写完全不需要网络。Arion 没有云端语音识别，也没有遥测。

<aside>

**同步功能尚未推出。** 它的设计已写入规范：每条记录在上传前先在设备上加密，只有你的设备能读取。

</aside>

## 自己验证

Arion 以 AGPL-3.0 开源，你不必只相信这篇文章。从一份全新的克隆开始构建：

```sh
git clone https://github.com/AstraFoundry/Arion.git
cd Arion
make install
make build desktop
```

然后关掉 Wi-Fi 再听写。如果这里写的任何一点不再成立，那就是 bug；[隐私页面](/privacy/)和规范[^spec]会和代码在同一次变更里一起修改。

[^keys]: 密钥缺失或完整性校验失败都会报错。Arion 绝不会退回到明文保存词汇。

[^spec]: 仓库中的[规范 005：隐私](https://github.com/AstraFoundry/Arion/blob/main/specs/005-privacy.md)。
