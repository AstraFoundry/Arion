# Actual product worker output review

All 17 completed text outputs were read by the implementing agent. This is a single-agent integration review, not blind independent quality acceptance. Repeated/exposed integration observations are not independent held-out cases.

## smoke-results.json / 1

Scene: work; operation: rewrite

Source:

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

Actual returned text:

```text
- 小陈负责设计，周三交。
- 小刘负责测试，周五完成。
```

Responsibility and deadlines are retained in two bullets. The design-draft object is implicit in the shortened wording “周三交”; useful structure, but less explicit than the requested ideal example. Accepted without further case-specific prompt tuning.

## smoke-results.json / 2

Scene: email; operation: rewrite

Source:

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Actual returned text:

```text
周经理您好：

我周四请假一天，已经交接给小赵。麻烦批准，谢谢。
```

Greeting, leave date/duration, completed handover, approval request and thanks are retained; useful email paragraph layout.

## smoke-results.json / 3

Scene: work; operation: rewrite

Source:

```text
给 load cash 加日志，不记录用户内容，先别改缓存策略。
```

Actual returned text:

```text
给 loadCache 加日志，不记录用户内容，先别改缓存策略。
```

load cash is corrected to loadCache using the matching function context. No-log-content and no-cache-policy-change restrictions remain.

## smoke-results.json / 4

Scene: work; operation: rewrite

Source:

```text
给 read cash 加日志，先别提交。
```

Actual returned text:

```text
给 readCash 加日志，先别提交。
```

read cash is canonicalized to readCash, preserving source letters instead of choosing unsupported readCache. The no-submit instruction remains.

## smoke-results.json / 5

Scene: work; operation: rewrite

Source:

```text
请核对服务器的配植清单，暂时不要重启。
```

Actual returned text:

```text
请核对服务器的配植清单，暂时不要重启。
```

Irrelevant nursery OCR is not used to replace the server term with 培植. The original uncertain spelling and no-restart restriction remain.

## smoke-results.json / 6

Scene: prompt; operation: rewrite

Source:

```text
给 fetchState 加校验，不改返回字段，也别动数据库。
```

Actual returned text:

```text
- 给 fetchState 加校验。
- 不改返回字段。
- 也别动数据库。
```

The coding request is split into three bullets; function spelling, response-field restriction and database restriction remain.

## smoke-results.json / 7

Scene: chat; operation: rewrite

Source:

```text
If it rains, I might join online. Do not confirm for me.
```

Actual returned text:

```text
If it rains, I might join online. Do not confirm for me.
```

English condition, uncertainty and no-confirmation request are retained without translation; unchanged output is appropriate.

## native-observations.jsonl / 0

Scene: work; operation: rewrite

Source:

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

Actual returned text:

```text
- 小陈负责设计，周三交。
- 小刘负责测试，周五完成。
```

Responsibility and deadlines are retained in two bullets. The design-draft object is implicit in the shortened wording “周三交”; useful structure, but less explicit than the requested ideal example. Accepted without further case-specific prompt tuning.

## native-observations.jsonl / 1

Scene: email; operation: rewrite

Source:

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Actual returned text:

```text
周经理您好：

我周四请假一天，已经交接给小赵。麻烦批准，谢谢。
```

Greeting, leave date/duration, completed handover, approval request and thanks are retained; useful email paragraph layout.

## native-observations.jsonl / 2

Scene: prompt; operation: rewrite

Source:

```text
给 fetchState 加校验，不改返回字段，也别动数据库。
```

Actual returned text:

```text
- 给 fetchState 加校验。
- 不改返回字段。
- 也别动数据库。
```

The coding request is split into three bullets; function spelling, response-field restriction and database restriction remain.

## native-observations.jsonl / 3

Scene: work; operation: new-session

Source:

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

Actual returned text:

```text
- 小陈负责设计，周三交。
- 小刘负责测试，周五完成。
```

Responsibility and deadlines are retained in two bullets. The design-draft object is implicit in the shortened wording “周三交”; useful structure, but less explicit than the requested ideal example. Accepted without further case-specific prompt tuning. Byte-identical to the earlier same-source output after a native slot erase.

## native-observations.jsonl / 4

Scene: work; operation: preparation

Source:

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

Actual returned text:

```text

```

No final text is expected: generation-free preparation.

## native-observations.jsonl / 5

Scene: work; operation: cancelled

Source:

```text
请整理工作内容，第一小陈负责设计，第二小刘负责测试，第三今天先核对，不要发出去。
```

Actual returned text:

```text

```

Cancelled request returns no publishable text; the next request is independently tested.

## native-observations.jsonl / 6

Scene: email; operation: after-cancellation

Source:

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Actual returned text:

```text
周经理您好：

我周四请假一天，已经交接给小赵。麻烦批准，谢谢。
```

Greeting, leave date/duration, completed handover, approval request and thanks are retained; useful email paragraph layout. Completed successfully after the preceding cancelled request.

## final-native-observations.jsonl / 0

Scene: work; operation: rewrite

Source:

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

Actual returned text:

```text
- 小陈负责设计，周三交。
- 小刘负责测试，周五完成。
```

Responsibilities and deadlines are retained in two bullets. “周三交” leaves the design-draft object implicit; useful structure, but less explicit than the ideal example. No case-specific prompt tuning was introduced.

## final-native-observations.jsonl / 1

Scene: email; operation: rewrite

Source:

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Actual returned text:

```text
周经理您好：

我周四请假一天，已经交接给小赵。麻烦批准，谢谢。
```

Greeting, leave date/duration, handover, approval request and thanks are retained in useful email layout.

## final-native-observations.jsonl / 2

Scene: prompt; operation: rewrite

Source:

```text
给 fetchState 加校验，不改返回字段，也别动数据库。
```

Actual returned text:

```text
- 给 fetchState 加校验。
- 不改返回字段。
- 也别动数据库。
```

Function spelling and all three coding requirements/restrictions survive in separate bullets.

## final-native-observations.jsonl / 3

Scene: work; operation: new-session

Source:

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

Actual returned text:

```text
- 小陈负责设计，周三交。
- 小刘负责测试，周五完成。
```

Responsibilities and deadlines are retained in two bullets. “周三交” leaves the design-draft object implicit; useful structure, but less explicit than the ideal example. No case-specific prompt tuning was introduced. Same output bytes after native slot erasure.

## final-native-observations.jsonl / 4

Scene: work; operation: preparation

Source:

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

Actual returned text:

```text

```

No publishable final text is expected for this preparation/cancellation observation.

## final-native-observations.jsonl / 5

Scene: work; operation: cancelled

Source:

```text
请整理工作内容，第一小陈负责设计，第二小刘负责测试，第三今天先核对，不要发出去。
```

Actual returned text:

```text

```

No publishable final text is expected for this preparation/cancellation observation.

## final-native-observations.jsonl / 6

Scene: email; operation: after-cancellation

Source:

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Actual returned text:

```text
周经理您好：

我周四请假一天，已经交接给小赵。麻烦批准，谢谢。
```

Greeting, leave date/duration, handover, approval request and thanks are retained in useful email layout. The final packaged worker remains usable after cancellation.

