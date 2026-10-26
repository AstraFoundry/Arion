# rime-llm-reranker

在 Rime 输入法（鼠须管 Squirrel）中集成本地 LLM，实时对候选词重新排序。基于 oMLX + Apple Silicon 统一内存架构，同步直连，无中间层，无额外进程。

---

## 工作原理

```
用户按键
  ↓
Rime 主线程
  ├─ 传统翻译器生成候选词
  ├─ simplifier 转换简繁
  └─ llm_reranker 过滤器（本项目）
       ├─ 去抖动：拼音太短 / 末尾非音节 → 直接透传
       ├─ 缓存命中 → 直接返回上次结果
       └─ 同步调用 oMLX（luasocket，~1ms）
            ↓
          按 LLM 偏好重排候选词
```

为什么可以同步调用：oMLX 配合 KV Cache，在 Apple Silicon 上响应时间约 1ms（P99 < 5ms），远低于用户选词速度（400–800ms），不会造成感知延迟。

---

## 前置条件

| 依赖 | 说明 |
|------|------|
| macOS + Apple Silicon | M1/M2/M3/M4 均可 |
| [Squirrel 1.x](https://github.com/rime/squirrel/releases) | 需要内置 librime-lua 插件 搭配 [雾凇输入法](https://github.com/iDvel/rime-ice) 效果更好 |
| [oMLX](https://github.com/oliverphilcox/oMLX) | 本地 LLM 推理服务，OpenAI 兼容 API |
| 中文模型 | 推荐 `Qwen3.5-0.8B-MLX-bf16`（流畅）或 `Qwen3.5-2B-OptiQ-4bit`（效果更好但略慢） |

---

## 安装

### 1. 安装 luasocket（推荐，将延迟从 ~20ms 降到 ~1ms）

```bash
brew install lua@5.4 luarocks
luarocks --lua-dir=/opt/homebrew/opt/lua@5.4 install luasocket
```

### 2. 运行安装脚本

```bash
chmod +x install.sh
./install.sh
```

脚本会自动：
- 将 `rime.lua` 复制到 `~/Library/Rime/`
- 将 `luna_pinyin_simp.custom.yaml` 复制到 `~/Library/Rime/`（已存在则跳过）
- 检查 oMLX 运行状态和 luasocket 安装情况

### 3. 修改配置

编辑 `~/Library/Rime/luna_pinyin_simp.custom.yaml`，将模型名改为你在 oMLX 中加载的模型：

```yaml
llm_reranker:
  model: "Qwen3.5-0.8B-MLX-bf16"   # 改成你的模型名
  endpoint: "http://localhost:8000/v1/chat/completions"
```

### 4. 重新部署

点击菜单栏 Squirrel 图标 → **重新部署**。

---

## 使用其他输入方案

本项目默认配置针对「朙月拼音·简化字」（`luna_pinyin_simp`）。如需用于其他方案，复制一份 custom yaml 并重命名：

```bash
# 示例：朙月拼音（繁体）
cp config/luna_pinyin_simp.custom.yaml ~/Library/Rime/luna_pinyin.custom.yaml
```

> **注意**：繁体方案需将 `@before 1` 改为 `@before 0`，因为繁体方案没有 simplifier。

---

## 配置参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `enabled` | `true` | 总开关，`false` 恢复纯 n-gram |
| `endpoint` | `http://localhost:8000/v1/chat/completions` | oMLX API 地址 |
| `model` | `Qwen3.5-0.8B-MLX-bf16` | 模型名称 |
| `max_candidates` | `5` | 参与重排序的候选词数量 |
| `timeout` | `0.5` | HTTP 超时（秒），超时后保留原始顺序 |
| `min_preedit_len` | `3` | 去抖动：拼音长度少于此值时跳过 LLM |
| `temperature` | `0.0` | 模型温度，0 为确定性输出 |
| `debug_log` | `false` | 调试日志，写入 `~/Library/Rime/rime_llm.log` |

---

## 调试

```bash
# 开启日志
# 将 luna_pinyin_simp.custom.yaml 中 debug_log 设为 true，重新部署

# 实时查看日志
tail -f ~/Library/Rime/rime_llm.log
```

日志格式：
```
09:15:32 backend: luasocket         ← 确认使用低延迟模式
09:15:33 OK 1ms → 「测试」          ← LLM 返回结果及延迟
09:15:34 cache hit: ceshi           ← 缓存命中，未调用 LLM
09:15:35 FAIL 500ms curl empty      ← oMLX 未响应，降级到原始排序
```

---

## 模型选择建议

| 模型 | 延迟 | 中文效果 | 推荐场景 |
|------|------|---------|---------|
| `Qwen3.5-0.8B-MLX-bf16` | ~1ms | 好 | 日常使用首选 |
| `Qwen3.5-0.8B-8bit` | ~1ms | 好 | 与 bf16 相当，内存略小 |
| `Qwen3.5-2B-OptiQ-4bit` | ~5ms | 更好 | 对效果要求高且可接受轻微延迟 |

> 量化（4bit/8bit）减少的是内存占用，不减少计算量。更大的模型即使量化后也比更小模型慢，输入法场景建议优先选小模型。

---

## 文件结构

```
rime-llm-reranker/
├── rime.lua                          # 主程序（放到 ~/Library/Rime/）
├── config/
│   └── luna_pinyin_simp.custom.yaml  # 方案配置（放到 ~/Library/Rime/）
├── install.sh                        # 一键安装脚本
└── README.md
```
