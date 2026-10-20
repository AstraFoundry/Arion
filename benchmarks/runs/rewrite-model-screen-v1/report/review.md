# Complete checkpoint/export screen review

Same native worker, package, templates and budgets; different checkpoint/export bundles. Every proposal and cleanup personally inspected by assistant. Eight exposed probes, not independent review or release latency evidence.

## production-1.7b / owner-anchor

Scene: work → work (fixture assigned); template: ocr-alignment-v2-aligned-work-zh

**Source**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

**Full reference**

```json
{}
```

**Raw output**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

Owner/task/date facts preserved; unchanged inline ordering, no owner grouping.

Faithful: True; scene ready: False; strict: accepted

**Strict delivery**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

Owner/task/date facts preserved; unchanged inline ordering, no owner grouping.

## production-1.7b / identifier-wrong-relevant

Scene: work → work (fixture assigned); template: ocr-alignment-v2-aligned-work-zh

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

**Raw output**

```text
给 loadCache 加日志，不记录用户内容，先别改缓存策略。
```

Correct loadCache repair with no-user-content logging and unchanged cache policy. Diagnostic logging wording remains consistent with the dictated task.

Faithful: True; scene ready: True; strict: accepted

**Strict delivery**

```text
给 loadCache 加日志，不记录用户内容，先别改缓存策略。
```

Correct loadCache repair with no-user-content logging and unchanged cache policy. Diagnostic logging wording remains consistent with the dictated task.

## production-1.7b / control-work-exact

Scene: work → work (fixture assigned); template: ocr-alignment-v2-aligned-work-zh

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

**Raw output**

```text
负责人：施雨宁、施语宁；名单核对。
```

Copies the two-person OCR excerpt, dropping source actor/task instruction and confirmation-before-submit prohibition.

Faithful: False; scene ready: False; strict: negation_dropped

**Strict delivery**

```text
施雨宁先核对名单，没确认前别提交。
```

Cleanup exactly equals inspected source. All original facts and intent retained, with original layout.

## production-1.7b / negative-question

Scene: work → work (fixture assigned); template: ocr-alignment-v2-aligned-work-zh

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

**Raw output**

```text
这个方案还不能发布吗？先别操作。
```

Negative question and no-action restriction retained.

Faithful: True; scene ready: True; strict: accepted

**Strict delivery**

```text
这个方案还不能发布吗？先别操作。
```

Negative question and no-action restriction retained.

## production-1.7b / email-anchor

Scene: email → email (fixture assigned); template: ocr-alignment-v2-aligned-email-zh

**Source**

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

**Full reference**

```json
{}
```

**Raw output**

```text
周经理您好：

我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Leave date/duration, completed handoff, approval request and thanks retained in salutation/body paragraphs.

Faithful: True; scene ready: True; strict: accepted

**Strict delivery**

```text
周经理您好：

我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Leave date/duration, completed handoff, approval request and thanks retained in salutation/body paragraphs.

## production-1.7b / recipient-wrong-relevant

Scene: email → email (fixture assigned); template: ocr-alignment-v2-aligned-email-zh

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

**Raw output**

```text
林雨桐您好：

报告明早发您，请先不要转发，谢谢。
```

Keeps 林雨桐 instead of reference-supported intended 林语桐.

Faithful: False; scene ready: False; strict: accepted

**Strict delivery**

```text
林雨桐您好：

报告明早发您，请先不要转发，谢谢。
```

Keeps 林雨桐 instead of reference-supported intended 林语桐.

## production-1.7b / wrong-ocr

Scene: email → email (fixture assigned); template: ocr-alignment-v2-aligned-email-zh

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

**Raw output**

```text
韩若宁您好：

我明早再给您答复。
```

Correct 韩若宁 preserved against different-pronunciation OCR; sender will reply tomorrow.

Faithful: True; scene ready: True; strict: accepted

**Strict delivery**

```text
韩若宁您好：

我明早再给您答复。
```

Correct 韩若宁 preserved against different-pronunciation OCR; sender will reply tomorrow.

## production-1.7b / conditional-fact

Scene: email → email (fixture assigned); template: ocr-alignment-v2-aligned-email-zh

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

**Raw output**

```text
罗先生您好：

您确认后我再寄，运费还没确定，请不要提前付款。
```

Recipient confirms before sender ships; fee undetermined and no-advance-payment request preserved.

Faithful: True; scene ready: True; strict: accepted

**Strict delivery**

```text
罗先生您好：

您确认后我再寄，运费还没确定，请不要提前付款。
```

Recipient confirms before sender ships; fee undetermined and no-advance-payment request preserved.

## prior-4b-export / owner-anchor

Scene: work → work (fixture assigned); template: ocr-alignment-v2-aligned-work-zh

**Source**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

**Full reference**

```json
{}
```

**Raw output**

```text
- 小陈负责设计，设计稿周三交。  
- 小刘负责测试，测试周五完成。
```

Owner/task/date facts preserved; correct two-bullet grouping.

Faithful: True; scene ready: True; strict: accepted

**Strict delivery**

```text
- 小陈负责设计，设计稿周三交。  
- 小刘负责测试，测试周五完成。
```

Owner/task/date facts preserved; correct two-bullet grouping.

## prior-4b-export / identifier-wrong-relevant

Scene: work → work (fixture assigned); template: ocr-alignment-v2-aligned-work-zh

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

**Raw output**

```text
为 loadCache 添加诊断日志，不记录用户内容，先别改缓存策略。
```

Correct loadCache repair with no-user-content logging and unchanged cache policy. Diagnostic logging wording remains consistent with the dictated task.

Faithful: True; scene ready: True; strict: accepted

**Strict delivery**

```text
为 loadCache 添加诊断日志，不记录用户内容，先别改缓存策略。
```

Correct loadCache repair with no-user-content logging and unchanged cache policy. Diagnostic logging wording remains consistent with the dictated task.

## prior-4b-export / control-work-exact

Scene: work → work (fixture assigned); template: ocr-alignment-v2-aligned-work-zh

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

**Raw output**

```text
施雨宁先核对名单，没确认前别提交。
```

Correct name and confirmation-before-submit prohibition retained.

Faithful: True; scene ready: True; strict: accepted

**Strict delivery**

```text
施雨宁先核对名单，没确认前别提交。
```

Correct name and confirmation-before-submit prohibition retained.

## prior-4b-export / negative-question

Scene: work → work (fixture assigned); template: ocr-alignment-v2-aligned-work-zh

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

**Raw output**

```text
当前状态：方案可以发布；操作步骤：立即发布。
```

Copies permission-to-publish OCR, losing the negative question and no-action restriction. Strict rejects it.

Faithful: False; scene ready: False; strict: negation_dropped

**Strict delivery**

```text
这个方案还不能发布吗？先别操作。
```

Cleanup exactly equals inspected source. All original facts and intent retained, with original layout.

## prior-4b-export / email-anchor

Scene: email → email (fixture assigned); template: ocr-alignment-v2-aligned-email-zh

**Source**

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

**Full reference**

```json
{}
```

**Raw output**

```text
周经理您好，我周四请假一天，交接给小赵，麻烦批准，谢谢。
```

Drops 了 from completed handoff and produces 交接给小赵 without completed status; salutation/body also stay inline. Strict accepts this uncertain change of action status.

Faithful: False; scene ready: False; strict: accepted

**Strict delivery**

```text
周经理您好，我周四请假一天，交接给小赵，麻烦批准，谢谢。
```

Drops 了 from completed handoff and produces 交接给小赵 without completed status; salutation/body also stay inline. Strict accepts this uncertain change of action status.

## prior-4b-export / recipient-wrong-relevant

Scene: email → email (fixture assigned); template: ocr-alignment-v2-aligned-email-zh

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

**Raw output**

```text
林语桐您好：

报告明早发您，请先不要转发，谢谢。
```

Correct 林语桐 repair; tomorrow-morning delivery and no-forward restriction preserved.

Faithful: True; scene ready: True; strict: accepted

**Strict delivery**

```text
林语桐您好：

报告明早发您，请先不要转发，谢谢。
```

Correct 林语桐 repair; tomorrow-morning delivery and no-forward restriction preserved.

## prior-4b-export / wrong-ocr

Scene: email → email (fixture assigned); template: ocr-alignment-v2-aligned-email-zh

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

**Raw output**

```text
韩若宁您好：

我明早再给您答复。
```

Correct 韩若宁 preserved against different-pronunciation OCR; sender will reply tomorrow.

Faithful: True; scene ready: True; strict: accepted

**Strict delivery**

```text
韩若宁您好：

我明早再给您答复。
```

Correct 韩若宁 preserved against different-pronunciation OCR; sender will reply tomorrow.

## prior-4b-export / conditional-fact

Scene: email → email (fixture assigned); template: ocr-alignment-v2-aligned-email-zh

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

**Raw output**

```text
罗先生您好：

您确认后我再寄，运费还没确定，请不要提前付款。
```

Recipient confirms before sender ships; fee undetermined and no-advance-payment request preserved.

Faithful: True; scene ready: True; strict: accepted

**Strict delivery**

```text
罗先生您好：

您确认后我再寄，运费还没确定，请不要提前付款。
```

Recipient confirms before sender ships; fee undetermined and no-advance-payment request preserved.

