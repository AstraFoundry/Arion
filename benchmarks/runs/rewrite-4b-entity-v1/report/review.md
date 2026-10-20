# Complete fixed-4B JSON entity-policy review

Complete raw/strict/cleanup inspection by assistant, not independent human ratings. Scene assignment is explicit fixture input.

## baseline / owner-anchor

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh

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

Faithful: True; scene useful: True; strict: accepted

**Cleanup**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

Cleanup exactly equals inspected source. All original facts and intent retained, with original layout.

**Strict delivery**

```text
- 小陈负责设计，设计稿周三交。
- 小刘负责测试，测试周五完成。
```

Owner/task/date facts preserved; correct two-bullet grouping.

## baseline / identifier-wrong-relevant

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh

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

Faithful: True; scene useful: True; strict: accepted

**Cleanup**

```text
给 load cash 加日志，不记录用户内容，先别改缓存策略。
```

Cleanup exactly equals inspected source. Required intended spelling remains unrepaired.

**Strict delivery**

```text
给 loadCache 加日志，不记录用户内容，先别改缓存策略。
```

Correct loadCache repair with no-user-content logging and unchanged cache policy. Diagnostic logging wording remains consistent with the dictated task.

## baseline / control-work-exact

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh

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

Faithful: True; scene useful: True; strict: accepted

**Cleanup**

```text
施雨宁先核对名单，没确认前别提交。
```

Cleanup exactly equals inspected source. All original facts and intent retained, with original layout.

**Strict delivery**

```text
施雨宁先核对名单，没确认前别提交。
```

Correct name and confirmation-before-submit prohibition retained.

## baseline / negative-question

Scene: work → work; template: ocr-alignment-v2-aligned-work-zh

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

Negative question and no-action prohibition both retained despite conflicting OCR.

Faithful: True; scene useful: True; strict: accepted

**Cleanup**

```text
这个方案还不能发布吗？先别操作。
```

Cleanup exactly equals inspected source. All original facts and intent retained, with original layout.

**Strict delivery**

```text
这个方案还不能发布吗？先别操作。
```

Negative question and no-action prohibition both retained despite conflicting OCR.

## baseline / email-anchor

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh

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

Salutation/body paragraphs preserve leave, completed handoff 交接给小赵了, approval request and thanks.

Faithful: True; scene useful: True; strict: accepted

**Cleanup**

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Cleanup exactly equals inspected source. All original facts and intent retained, with original layout.

**Strict delivery**

```text
周经理您好：

我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Salutation/body paragraphs preserve leave, completed handoff 交接给小赵了, approval request and thanks.

## baseline / recipient-wrong-relevant

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh

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

Does not repair 林雨桐 to intended contextual 林语桐; other clauses retained.

Faithful: False; scene useful: False; strict: accepted

**Cleanup**

```text
林雨桐您好，报告明早发您，请先不要转发，谢谢。
```

Cleanup exactly equals inspected source. Required intended spelling remains unrepaired.

**Strict delivery**

```text
林雨桐您好：

报告明早发您，请先不要转发，谢谢。
```

Does not repair 林雨桐 to intended contextual 林语桐; other clauses retained.

## baseline / wrong-ocr

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh

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

Faithful: True; scene useful: True; strict: accepted

**Cleanup**

```text
韩若宁您好，我明早再给您答复。
```

Cleanup exactly equals inspected source. All original facts and intent retained, with original layout.

**Strict delivery**

```text
韩若宁您好：

我明早再给您答复。
```

Correct 韩若宁 preserved against different-pronunciation OCR; sender will reply tomorrow.

## baseline / conditional-fact

Scene: email → email; template: ocr-alignment-v2-aligned-email-zh

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
罗先生您好，  
您确认后我再寄，运费还没确定，请不要提前付款。
```

Recipient confirmation, sender action, undetermined fee and no-payment preserved; salutation has only a line break, not the requested blank-line paragraph separation.

Faithful: True; scene useful: False; strict: accepted

**Cleanup**

```text
罗先生您好，您确认后我再寄，运费还没确定，请不要提前付款。
```

Cleanup exactly equals inspected source. All original facts and intent retained, with original layout.

**Strict delivery**

```text
罗先生您好，  
您确认后我再寄，运费还没确定，请不要提前付款。
```

Recipient confirmation, sender action, undetermined fee and no-payment preserved; salutation has only a line break, not the requested blank-line paragraph separation.

## exact / owner-anchor

Scene: work → work; template: 4b-json-entity-priority-v1-work

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

Faithful: True; scene useful: True; strict: accepted

**Cleanup**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

Cleanup exactly equals inspected source. All original facts and intent retained, with original layout.

**Strict delivery**

```text
- 小陈负责设计，设计稿周三交。
- 小刘负责测试，测试周五完成。
```

Owner/task/date facts preserved; correct two-bullet grouping.

## exact / identifier-wrong-relevant

Scene: work → work; template: 4b-json-entity-priority-v1-work

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

Faithful: True; scene useful: True; strict: accepted

**Cleanup**

```text
给 load cash 加日志，不记录用户内容，先别改缓存策略。
```

Cleanup exactly equals inspected source. Required intended spelling remains unrepaired.

**Strict delivery**

```text
给 loadCache 加日志，不记录用户内容，先别改缓存策略。
```

Correct loadCache repair with no-user-content logging and unchanged cache policy. Diagnostic logging wording remains consistent with the dictated task.

## exact / control-work-exact

Scene: work → work; template: 4b-json-entity-priority-v1-work

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

Faithful: True; scene useful: True; strict: accepted

**Cleanup**

```text
施雨宁先核对名单，没确认前别提交。
```

Cleanup exactly equals inspected source. All original facts and intent retained, with original layout.

**Strict delivery**

```text
施雨宁先核对名单，没确认前别提交。
```

Correct name and confirmation-before-submit prohibition retained.

## exact / negative-question

Scene: work → work; template: 4b-json-entity-priority-v1-work

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

Copies permission-to-publish OCR, losing the negative question and no-action prohibition.

Faithful: False; scene useful: False; strict: negation_dropped

**Cleanup**

```text
这个方案还不能发布吗？先别操作。
```

Cleanup exactly equals inspected source. All original facts and intent retained, with original layout.

**Strict delivery**

```text
这个方案还不能发布吗？先别操作。
```

Cleanup exactly equals inspected source. All original facts and intent retained, with original layout.

## exact / email-anchor

Scene: email → email; template: 4b-json-entity-priority-v1-email

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

Salutation/body paragraphs preserve leave, completed handoff 交接给小赵了, approval request and thanks.

Faithful: True; scene useful: True; strict: accepted

**Cleanup**

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Cleanup exactly equals inspected source. All original facts and intent retained, with original layout.

**Strict delivery**

```text
周经理您好：

我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Salutation/body paragraphs preserve leave, completed handoff 交接给小赵了, approval request and thanks.

## exact / recipient-wrong-relevant

Scene: email → email; template: 4b-json-entity-priority-v1-email

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

Does not repair 林雨桐 to intended contextual 林语桐; other clauses retained.

Faithful: False; scene useful: False; strict: accepted

**Cleanup**

```text
林雨桐您好，报告明早发您，请先不要转发，谢谢。
```

Cleanup exactly equals inspected source. Required intended spelling remains unrepaired.

**Strict delivery**

```text
林雨桐您好：

报告明早发您，请先不要转发，谢谢。
```

Does not repair 林雨桐 to intended contextual 林语桐; other clauses retained.

## exact / wrong-ocr

Scene: email → email; template: 4b-json-entity-priority-v1-email

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

Faithful: True; scene useful: True; strict: accepted

**Cleanup**

```text
韩若宁您好，我明早再给您答复。
```

Cleanup exactly equals inspected source. All original facts and intent retained, with original layout.

**Strict delivery**

```text
韩若宁您好：

我明早再给您答复。
```

Correct 韩若宁 preserved against different-pronunciation OCR; sender will reply tomorrow.

## exact / conditional-fact

Scene: email → email; template: 4b-json-entity-priority-v1-email

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

Correct salutation/body paragraphs preserve recipient-before-sender condition, unknown fee and no-payment request.

Faithful: True; scene useful: True; strict: accepted

**Cleanup**

```text
罗先生您好，您确认后我再寄，运费还没确定，请不要提前付款。
```

Cleanup exactly equals inspected source. All original facts and intent retained, with original layout.

**Strict delivery**

```text
罗先生您好：

您确认后我再寄，运费还没确定，请不要提前付款。
```

Correct salutation/body paragraphs preserve recipient-before-sender condition, unknown fee and no-payment request.

