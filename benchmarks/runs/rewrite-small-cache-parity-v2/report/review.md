# Complete0.8B token-step cache review

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

Non-test warmup personally read: preserves dictated actor, status and prohibition. Synthetic internal marker is absent; excluded from measured latency/quality denominators.

Faithful: True; scene useful: True; request: 4452.1 ms; cached tokens: 0

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

Faithful: True; scene useful: False; request: 4384.4 ms; cached tokens: 0

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

Repairs loadCache and retains no-content-logging/no-cache-strategy-change. All token-step cached/uncached and forward/reverse versions use 添加诊断日志; raw equality verified separately.

Faithful: True; scene useful: True; request: 4826.0 ms; cached tokens: 0

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

Faithful: True; scene useful: True; request: 4708.8 ms; cached tokens: 0

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

Faithful: True; scene useful: True; request: 4704.2 ms; cached tokens: 0

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

Non-test warmup personally read: preserves dictated actor, status and prohibition. Synthetic internal marker is absent; excluded from measured latency/quality denominators.

Faithful: True; scene useful: True; request: 4827.4 ms; cached tokens: 0

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

Faithful: True; scene useful: True; request: 4961.8 ms; cached tokens: 0

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

Faithful: True; scene useful: True; request: 5087.8 ms; cached tokens: 0

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

Faithful: True; scene useful: True; request: 5047.7 ms; cached tokens: 0

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

Faithful: True; scene useful: True; request: 4968.4 ms; cached tokens: 0

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

Non-test warmup personally read: preserves dictated actor, status and prohibition. Synthetic internal marker is absent; excluded from measured latency/quality denominators.

Faithful: True; scene useful: True; request: 4902.8 ms; cached tokens: 0

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

Faithful: True; scene useful: False; request: 623.2 ms; cached tokens: 551

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
给 loadCache 添加诊断日志，不记录用户内容，先别改缓存策略。
```

Repairs loadCache and retains no-content-logging/no-cache-strategy-change. All token-step cached/uncached and forward/reverse versions use 添加诊断日志; raw equality verified separately.

Faithful: True; scene useful: True; request: 921.5 ms; cached tokens: 551

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

Faithful: True; scene useful: True; request: 716.5 ms; cached tokens: 551

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

Faithful: True; scene useful: True; request: 653.8 ms; cached tokens: 551

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

Non-test warmup personally read: preserves dictated actor, status and prohibition. Synthetic internal marker is absent; excluded from measured latency/quality denominators.

Faithful: True; scene useful: True; request: 5176.1 ms; cached tokens: 0

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

Faithful: True; scene useful: True; request: 791.4 ms; cached tokens: 572

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

Faithful: True; scene useful: True; request: 714.3 ms; cached tokens: 572

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

Faithful: True; scene useful: True; request: 840.2 ms; cached tokens: 572

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

Faithful: True; scene useful: True; request: 682.2 ms; cached tokens: 572

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

Non-test warmup personally read: preserves dictated actor, status and prohibition. Synthetic internal marker is absent; excluded from measured latency/quality denominators.

Faithful: True; scene useful: True; request: 5147.7 ms; cached tokens: 0

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

Faithful: True; scene useful: True; request: 5917.1 ms; cached tokens: 0

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

Faithful: True; scene useful: True; request: 6279.2 ms; cached tokens: 0

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

Repairs loadCache and retains no-content-logging/no-cache-strategy-change. All token-step cached/uncached and forward/reverse versions use 添加诊断日志; raw equality verified separately.

Faithful: True; scene useful: True; request: 7345.8 ms; cached tokens: 0

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

Faithful: True; scene useful: False; request: 7398.9 ms; cached tokens: 0

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

Non-test warmup personally read: preserves dictated actor, status and prohibition. Synthetic internal marker is absent; excluded from measured latency/quality denominators.

Faithful: True; scene useful: True; request: 8423.6 ms; cached tokens: 0

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

Faithful: True; scene useful: True; request: 8825.1 ms; cached tokens: 0

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

Faithful: True; scene useful: True; request: 9628.5 ms; cached tokens: 0

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

Faithful: True; scene useful: True; request: 9845.3 ms; cached tokens: 0

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

Faithful: True; scene useful: True; request: 9672.7 ms; cached tokens: 0

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

Non-test warmup personally read: preserves dictated actor, status and prohibition. Synthetic internal marker is absent; excluded from measured latency/quality denominators.

Faithful: True; scene useful: True; request: 9042.3 ms; cached tokens: 0

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

Faithful: True; scene useful: True; request: 1166.8 ms; cached tokens: 551

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

Faithful: True; scene useful: True; request: 1334.6 ms; cached tokens: 551

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
给 loadCache 添加诊断日志，不记录用户内容，先别改缓存策略。
```

Repairs loadCache and retains no-content-logging/no-cache-strategy-change. All token-step cached/uncached and forward/reverse versions use 添加诊断日志; raw equality verified separately.

Faithful: True; scene useful: True; request: 1613.6 ms; cached tokens: 551

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

Faithful: True; scene useful: False; request: 1234.7 ms; cached tokens: 551

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

Non-test warmup personally read: preserves dictated actor, status and prohibition. Synthetic internal marker is absent; excluded from measured latency/quality denominators.

Faithful: True; scene useful: True; request: 9498.1 ms; cached tokens: 0

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

Faithful: True; scene useful: True; request: 1146.6 ms; cached tokens: 572

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

Faithful: True; scene useful: True; request: 1523.2 ms; cached tokens: 572

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

Faithful: True; scene useful: True; request: 1284.5 ms; cached tokens: 572

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

Faithful: True; scene useful: True; request: 1404.5 ms; cached tokens: 572

