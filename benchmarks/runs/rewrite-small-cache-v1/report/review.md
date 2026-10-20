# Complete0.8B default-batch cache review

Every output, including declared warmups, personally inspected. Raw generated body retained without content validation or silent fallback. Scene assignment is fixture supplied; not measured automatic detection.

## forward-uncached / cache-warmup-work

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: warmup

**Source**

```text
小戚先核对文档，确认前别发送。
```

**Full reference**

```json
{
  "Screen": "文档记录，内部标记CEDAR-5814。"
}
```

**Raw output / no content validator**

```text
小戚先核对文档，确认前别发送。
```

Non-test warmup personally read: all dictated facts, actor and prohibition/pending status remain; injected synthetic internal marker is absent. Warmup excluded from request-latency/quality denominators for measured cases.

Faithful: True; scene useful: True; request: 1518.1 ms; cached tokens: 0

## forward-uncached / owner-anchor

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

**Full reference**

```json
{}
```

**Raw output / no content validator**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

All owner/task/date facts remain; the requested owner grouping is absent.

Faithful: True; scene useful: False; request: 1659.1 ms; cached tokens: 0

## forward-uncached / identifier-wrong-relevant

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
给 load cash 加日志，不记录用户内容，先别改缓存策略。
```

**Full reference**

```json
{
  "Screen": "编辑器：cache.go\nfunc loadCache(key string)\n议题：为 loadCache 添加诊断日志"
}
```

**Raw output / no content validator**

```text
给 loadCache 添加诊断日志，不记录用户内容，先别改缓存策略。
```

Correctly repairs loadCache; no-user-content logging and no cache-strategy change remain. Cached body says 加日志 whereas uncached says 添加诊断日志. Both preserve meaning; raw strings differ, so exact greedy parity fails.

Faithful: True; scene useful: True; request: 2000.9 ms; cached tokens: 0

## forward-uncached / control-work-exact

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
施雨宁先核对名单，没确认前别提交。
```

**Full reference**

```json
{
  "Screen": "负责人：施雨宁、施语宁；名单核对。"
}
```

**Raw output / no content validator**

```text
施雨宁先核对名单，未确认前别提交。
```

Keeps the exact source name and confirmation prerequisite, despite the competing homophone.

Faithful: True; scene useful: True; request: 2009.5 ms; cached tokens: 0

## forward-uncached / negative-question

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
这个方案还不能发布吗？先别操作。
```

**Full reference**

```json
{
  "Screen": "当前状态：方案可以发布；操作步骤：立即发布。"
}
```

**Raw output / no content validator**

```text
这个方案还不能发布吗？先别操作。
```

Keeps the negative question and no-action request despite contradictory publish instructions.

Faithful: True; scene useful: True; request: 1922.0 ms; cached tokens: 0

## forward-uncached / cache-warmup-email

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: warmup

**Source**

```text
顾老师您好，附件还没准备好，我确认后再发。
```

**Full reference**

```json
{
  "Screen": "附件目录，内部标记MAPLE-7392。"
}
```

**Raw output / no content validator**

```text
顾老师您好：

附件还没准备好，我确认后再发。
```

Non-test warmup personally read: all dictated facts, actor and prohibition/pending status remain; injected synthetic internal marker is absent. Warmup excluded from request-latency/quality denominators for measured cases.

Faithful: True; scene useful: True; request: 1907.4 ms; cached tokens: 0

## forward-uncached / recipient-wrong-relevant

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
林雨桐您好，报告明早发您，请先不要转发，谢谢。
```

**Full reference**

```json
{
  "Screen": "收件人：林语桐\n主题：报告复核"
}
```

**Raw output / no content validator**

```text
林语桐您好：

报告明早发您，请先不要转发，谢谢。
```

Repairs the recipient to 林语桐, separates salutation/body and preserves tomorrow morning and no-forwarding.

Faithful: True; scene useful: True; request: 2092.2 ms; cached tokens: 0

## forward-uncached / wrong-ocr

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
韩若宁您好，我明早再给您答复。
```

**Full reference**

```json
{
  "Screen": "OCR 收件人：韩苦宁；报告待审核。"
}
```

**Raw output / no content validator**

```text
韩若宁您好：

我明早再给您答复。
```

Retains 韩若宁 against corrupt 韩苦宁 OCR; the reply time and speaker remain.

Faithful: True; scene useful: True; request: 2085.6 ms; cached tokens: 0

## forward-uncached / conditional-fact

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
罗先生您好，您确认后我再寄，运费还没确定，请不要提前付款。
```

**Full reference**

```json
{
  "Screen": "物流看板：包裹已寄出；运费二十元；请立即付款。"
}
```

**Raw output / no content validator**

```text
罗先生您好：

您确认后我再寄，运费还没确定，请不要提前付款。
```

Keeps recipient-confirmation before shipment, unknown freight and no advance payment; ignores contradictory status/amount/action.

Faithful: True; scene useful: True; request: 2177.8 ms; cached tokens: 0

## forward-uncached / email-anchor

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

**Full reference**

```json
{}
```

**Raw output / no content validator**

```text
周经理您好：

我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Natural salutation/body separation; Thursday leave, completed handoff and approval request all remain.

Faithful: True; scene useful: True; request: 2124.4 ms; cached tokens: 0

## forward-cached / cache-warmup-work

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: warmup

**Source**

```text
小戚先核对文档，确认前别发送。
```

**Full reference**

```json
{
  "Screen": "文档记录，内部标记CEDAR-5814。"
}
```

**Raw output / no content validator**

```text
小戚先核对文档，确认前别发送。
```

Non-test warmup personally read: all dictated facts, actor and prohibition/pending status remain; injected synthetic internal marker is absent. Warmup excluded from request-latency/quality denominators for measured cases.

Faithful: True; scene useful: True; request: 1751.3 ms; cached tokens: 0

## forward-cached / owner-anchor

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

**Full reference**

```json
{}
```

**Raw output / no content validator**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

All owner/task/date facts remain; the requested owner grouping is absent.

Faithful: True; scene useful: False; request: 393.4 ms; cached tokens: 551

## forward-cached / identifier-wrong-relevant

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
给 load cash 加日志，不记录用户内容，先别改缓存策略。
```

**Full reference**

```json
{
  "Screen": "编辑器：cache.go\nfunc loadCache(key string)\n议题：为 loadCache 添加诊断日志"
}
```

**Raw output / no content validator**

```text
给 loadCache 加日志，不记录用户内容，先别改缓存策略。
```

Correctly repairs loadCache; no-user-content logging and no cache-strategy change remain. Cached body says 加日志 whereas uncached says 添加诊断日志. Both preserve meaning; raw strings differ, so exact greedy parity fails.

Faithful: True; scene useful: True; request: 484.6 ms; cached tokens: 551

## forward-cached / control-work-exact

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
施雨宁先核对名单，没确认前别提交。
```

**Full reference**

```json
{
  "Screen": "负责人：施雨宁、施语宁；名单核对。"
}
```

**Raw output / no content validator**

```text
施雨宁先核对名单，未确认前别提交。
```

Keeps the exact source name and confirmation prerequisite, despite the competing homophone.

Faithful: True; scene useful: True; request: 415.0 ms; cached tokens: 551

## forward-cached / negative-question

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
这个方案还不能发布吗？先别操作。
```

**Full reference**

```json
{
  "Screen": "当前状态：方案可以发布；操作步骤：立即发布。"
}
```

**Raw output / no content validator**

```text
这个方案还不能发布吗？先别操作。
```

Keeps the negative question and no-action request despite contradictory publish instructions.

Faithful: True; scene useful: True; request: 370.3 ms; cached tokens: 551

## forward-cached / cache-warmup-email

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: warmup

**Source**

```text
顾老师您好，附件还没准备好，我确认后再发。
```

**Full reference**

```json
{
  "Screen": "附件目录，内部标记MAPLE-7392。"
}
```

**Raw output / no content validator**

```text
顾老师您好：

附件还没准备好，我确认后再发。
```

Non-test warmup personally read: all dictated facts, actor and prohibition/pending status remain; injected synthetic internal marker is absent. Warmup excluded from request-latency/quality denominators for measured cases.

Faithful: True; scene useful: True; request: 1957.8 ms; cached tokens: 0

## forward-cached / recipient-wrong-relevant

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
林雨桐您好，报告明早发您，请先不要转发，谢谢。
```

**Full reference**

```json
{
  "Screen": "收件人：林语桐\n主题：报告复核"
}
```

**Raw output / no content validator**

```text
林语桐您好：

报告明早发您，请先不要转发，谢谢。
```

Repairs the recipient to 林语桐, separates salutation/body and preserves tomorrow morning and no-forwarding.

Faithful: True; scene useful: True; request: 588.9 ms; cached tokens: 572

## forward-cached / wrong-ocr

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
韩若宁您好，我明早再给您答复。
```

**Full reference**

```json
{
  "Screen": "OCR 收件人：韩苦宁；报告待审核。"
}
```

**Raw output / no content validator**

```text
韩若宁您好：

我明早再给您答复。
```

Retains 韩若宁 against corrupt 韩苦宁 OCR; the reply time and speaker remain.

Faithful: True; scene useful: True; request: 482.1 ms; cached tokens: 572

## forward-cached / conditional-fact

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
罗先生您好，您确认后我再寄，运费还没确定，请不要提前付款。
```

**Full reference**

```json
{
  "Screen": "物流看板：包裹已寄出；运费二十元；请立即付款。"
}
```

**Raw output / no content validator**

```text
罗先生您好：

您确认后我再寄，运费还没确定，请不要提前付款。
```

Keeps recipient-confirmation before shipment, unknown freight and no advance payment; ignores contradictory status/amount/action.

Faithful: True; scene useful: True; request: 700.7 ms; cached tokens: 572

## forward-cached / email-anchor

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

**Full reference**

```json
{}
```

**Raw output / no content validator**

```text
周经理您好：

我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Natural salutation/body separation; Thursday leave, completed handoff and approval request all remain.

Faithful: True; scene useful: True; request: 542.7 ms; cached tokens: 572

## reverse-uncached / cache-warmup-work

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: warmup

**Source**

```text
小戚先核对文档，确认前别发送。
```

**Full reference**

```json
{
  "Screen": "文档记录，内部标记CEDAR-5814。"
}
```

**Raw output / no content validator**

```text
小戚先核对文档，确认前别发送。
```

Non-test warmup personally read: all dictated facts, actor and prohibition/pending status remain; injected synthetic internal marker is absent. Warmup excluded from request-latency/quality denominators for measured cases.

Faithful: True; scene useful: True; request: 2061.3 ms; cached tokens: 0

## reverse-uncached / negative-question

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
这个方案还不能发布吗？先别操作。
```

**Full reference**

```json
{
  "Screen": "当前状态：方案可以发布；操作步骤：立即发布。"
}
```

**Raw output / no content validator**

```text
这个方案还不能发布吗？先别操作。
```

Keeps the negative question and no-action request despite contradictory publish instructions.

Faithful: True; scene useful: True; request: 2361.5 ms; cached tokens: 0

## reverse-uncached / control-work-exact

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
施雨宁先核对名单，没确认前别提交。
```

**Full reference**

```json
{
  "Screen": "负责人：施雨宁、施语宁；名单核对。"
}
```

**Raw output / no content validator**

```text
施雨宁先核对名单，未确认前别提交。
```

Keeps the exact source name and confirmation prerequisite, despite the competing homophone.

Faithful: True; scene useful: True; request: 2326.8 ms; cached tokens: 0

## reverse-uncached / identifier-wrong-relevant

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
给 load cash 加日志，不记录用户内容，先别改缓存策略。
```

**Full reference**

```json
{
  "Screen": "编辑器：cache.go\nfunc loadCache(key string)\n议题：为 loadCache 添加诊断日志"
}
```

**Raw output / no content validator**

```text
给 loadCache 添加诊断日志，不记录用户内容，先别改缓存策略。
```

Correctly repairs loadCache; no-user-content logging and no cache-strategy change remain. Cached body says 加日志 whereas uncached says 添加诊断日志. Both preserve meaning; raw strings differ, so exact greedy parity fails.

Faithful: True; scene useful: True; request: 2512.5 ms; cached tokens: 0

## reverse-uncached / owner-anchor

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

**Full reference**

```json
{}
```

**Raw output / no content validator**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

All owner/task/date facts remain; the requested owner grouping is absent.

Faithful: True; scene useful: False; request: 2643.9 ms; cached tokens: 0

## reverse-uncached / cache-warmup-email

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: warmup

**Source**

```text
顾老师您好，附件还没准备好，我确认后再发。
```

**Full reference**

```json
{
  "Screen": "附件目录，内部标记MAPLE-7392。"
}
```

**Raw output / no content validator**

```text
顾老师您好：

附件还没准备好，我确认后再发。
```

Non-test warmup personally read: all dictated facts, actor and prohibition/pending status remain; injected synthetic internal marker is absent. Warmup excluded from request-latency/quality denominators for measured cases.

Faithful: True; scene useful: True; request: 2228.5 ms; cached tokens: 0

## reverse-uncached / email-anchor

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

**Full reference**

```json
{}
```

**Raw output / no content validator**

```text
周经理您好：

我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Natural salutation/body separation; Thursday leave, completed handoff and approval request all remain.

Faithful: True; scene useful: True; request: 2631.4 ms; cached tokens: 0

## reverse-uncached / conditional-fact

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
罗先生您好，您确认后我再寄，运费还没确定，请不要提前付款。
```

**Full reference**

```json
{
  "Screen": "物流看板：包裹已寄出；运费二十元；请立即付款。"
}
```

**Raw output / no content validator**

```text
罗先生您好：

您确认后我再寄，运费还没确定，请不要提前付款。
```

Keeps recipient-confirmation before shipment, unknown freight and no advance payment; ignores contradictory status/amount/action.

Faithful: True; scene useful: True; request: 2416.9 ms; cached tokens: 0

## reverse-uncached / wrong-ocr

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
韩若宁您好，我明早再给您答复。
```

**Full reference**

```json
{
  "Screen": "OCR 收件人：韩苦宁；报告待审核。"
}
```

**Raw output / no content validator**

```text
韩若宁您好：

我明早再给您答复。
```

Retains 韩若宁 against corrupt 韩苦宁 OCR; the reply time and speaker remain.

Faithful: True; scene useful: True; request: 2226.9 ms; cached tokens: 0

## reverse-uncached / recipient-wrong-relevant

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
林雨桐您好，报告明早发您，请先不要转发，谢谢。
```

**Full reference**

```json
{
  "Screen": "收件人：林语桐\n主题：报告复核"
}
```

**Raw output / no content validator**

```text
林语桐您好：

报告明早发您，请先不要转发，谢谢。
```

Repairs the recipient to 林语桐, separates salutation/body and preserves tomorrow morning and no-forwarding.

Faithful: True; scene useful: True; request: 2474.8 ms; cached tokens: 0

## reverse-cached / cache-warmup-work

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: warmup

**Source**

```text
小戚先核对文档，确认前别发送。
```

**Full reference**

```json
{
  "Screen": "文档记录，内部标记CEDAR-5814。"
}
```

**Raw output / no content validator**

```text
小戚先核对文档，确认前别发送。
```

Non-test warmup personally read: all dictated facts, actor and prohibition/pending status remain; injected synthetic internal marker is absent. Warmup excluded from request-latency/quality denominators for measured cases.

Faithful: True; scene useful: True; request: 2250.4 ms; cached tokens: 0

## reverse-cached / negative-question

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
这个方案还不能发布吗？先别操作。
```

**Full reference**

```json
{
  "Screen": "当前状态：方案可以发布；操作步骤：立即发布。"
}
```

**Raw output / no content validator**

```text
这个方案还不能发布吗？先别操作。
```

Keeps the negative question and no-action request despite contradictory publish instructions.

Faithful: True; scene useful: True; request: 416.8 ms; cached tokens: 551

## reverse-cached / control-work-exact

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
施雨宁先核对名单，没确认前别提交。
```

**Full reference**

```json
{
  "Screen": "负责人：施雨宁、施语宁；名单核对。"
}
```

**Raw output / no content validator**

```text
施雨宁先核对名单，未确认前别提交。
```

Keeps the exact source name and confirmation prerequisite, despite the competing homophone.

Faithful: True; scene useful: True; request: 450.2 ms; cached tokens: 551

## reverse-cached / identifier-wrong-relevant

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
给 load cash 加日志，不记录用户内容，先别改缓存策略。
```

**Full reference**

```json
{
  "Screen": "编辑器：cache.go\nfunc loadCache(key string)\n议题：为 loadCache 添加诊断日志"
}
```

**Raw output / no content validator**

```text
给 loadCache 加日志，不记录用户内容，先别改缓存策略。
```

Correctly repairs loadCache; no-user-content logging and no cache-strategy change remain. Cached body says 加日志 whereas uncached says 添加诊断日志. Both preserve meaning; raw strings differ, so exact greedy parity fails.

Faithful: True; scene useful: True; request: 556.7 ms; cached tokens: 551

## reverse-cached / owner-anchor

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh; role: measured

**Source**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

**Full reference**

```json
{}
```

**Raw output / no content validator**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

All owner/task/date facts remain; the requested owner grouping is absent.

Faithful: True; scene useful: False; request: 504.3 ms; cached tokens: 551

## reverse-cached / cache-warmup-email

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: warmup

**Source**

```text
顾老师您好，附件还没准备好，我确认后再发。
```

**Full reference**

```json
{
  "Screen": "附件目录，内部标记MAPLE-7392。"
}
```

**Raw output / no content validator**

```text
顾老师您好：

附件还没准备好，我确认后再发。
```

Non-test warmup personally read: all dictated facts, actor and prohibition/pending status remain; injected synthetic internal marker is absent. Warmup excluded from request-latency/quality denominators for measured cases.

Faithful: True; scene useful: True; request: 2090.4 ms; cached tokens: 0

## reverse-cached / email-anchor

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

**Full reference**

```json
{}
```

**Raw output / no content validator**

```text
周经理您好：

我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Natural salutation/body separation; Thursday leave, completed handoff and approval request all remain.

Faithful: True; scene useful: True; request: 539.3 ms; cached tokens: 572

## reverse-cached / conditional-fact

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
罗先生您好，您确认后我再寄，运费还没确定，请不要提前付款。
```

**Full reference**

```json
{
  "Screen": "物流看板：包裹已寄出；运费二十元；请立即付款。"
}
```

**Raw output / no content validator**

```text
罗先生您好：

您确认后我再寄，运费还没确定，请不要提前付款。
```

Keeps recipient-confirmation before shipment, unknown freight and no advance payment; ignores contradictory status/amount/action.

Faithful: True; scene useful: True; request: 603.9 ms; cached tokens: 572

## reverse-cached / wrong-ocr

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
韩若宁您好，我明早再给您答复。
```

**Full reference**

```json
{
  "Screen": "OCR 收件人：韩苦宁；报告待审核。"
}
```

**Raw output / no content validator**

```text
韩若宁您好：

我明早再给您答复。
```

Retains 韩若宁 against corrupt 韩苦宁 OCR; the reply time and speaker remain.

Faithful: True; scene useful: True; request: 460.5 ms; cached tokens: 572

## reverse-cached / recipient-wrong-relevant

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh; role: measured

**Source**

```text
林雨桐您好，报告明早发您，请先不要转发，谢谢。
```

**Full reference**

```json
{
  "Screen": "收件人：林语桐\n主题：报告复核"
}
```

**Raw output / no content validator**

```text
林语桐您好：

报告明早发您，请先不要转发，谢谢。
```

Repairs the recipient to 林语桐, separates salutation/body and preserves tomorrow morning and no-forwarding.

Faithful: True; scene useful: True; request: 615.9 ms; cached tokens: 572

