# Cold and request-order parity diagnostic

Same aligned work template, same four inputs; the cold, forward and reverse arms have 0 output mismatches. This bounded check does not establish all runtime parity.

## cold / negative-question

**Source**

```text
这个方案还不能发布吗？先别操作。
```

**Context**

```json
{
  "Screen": "当前状态：方案可以发布；操作步骤：立即发布。"
}
```

**Output**

```text
当前状态：方案可以发布；操作步骤：立即发布。
```

Matches the already inspected cold/forward/reverse output: screen/demo content replaces the dictated request; material failure.

## cold / unrelated-fact

**Source**

```text
文档已经更新，等评审通过再合并。
```

**Context**

```json
{
  "Screen": "会议在下周三，负责人是小秦。"
}
```

**Output**

```text
会议在下周三，负责人是小秦。
```

Matches the already inspected cold/forward/reverse output: screen/demo content replaces the dictated request; material failure.

## cold / homophone-wrong-no-context

**Source**

```text
部署前先做回归侧试，不通过就不要发布。
```

**Context**

```json
{}
```

**Output**

```text
DONE. 已上线。
```

Matches the already inspected cold/forward/reverse output: screen/demo content replaces the dictated request; material failure.

## cold / homophone-wrong-relevant

**Source**

```text
部署前先做回归侧试，不通过就不要发布。
```

**Context**

```json
{
  "Screen": "发布讨论：上线前必须完成回归测试，尚未开始。"
}
```

**Output**

```text
部署前先做回归测试，不通过就不要发布。
```

Matches the already inspected cold/forward/reverse output: correct test spelling with preserved release restriction.

## forward / negative-question

**Source**

```text
这个方案还不能发布吗？先别操作。
```

**Context**

```json
{
  "Screen": "当前状态：方案可以发布；操作步骤：立即发布。"
}
```

**Output**

```text
当前状态：方案可以发布；操作步骤：立即发布。
```

Matches the already inspected cold/forward/reverse output: screen/demo content replaces the dictated request; material failure.

## forward / unrelated-fact

**Source**

```text
文档已经更新，等评审通过再合并。
```

**Context**

```json
{
  "Screen": "会议在下周三，负责人是小秦。"
}
```

**Output**

```text
会议在下周三，负责人是小秦。
```

Matches the already inspected cold/forward/reverse output: screen/demo content replaces the dictated request; material failure.

## forward / homophone-wrong-no-context

**Source**

```text
部署前先做回归侧试，不通过就不要发布。
```

**Context**

```json
{}
```

**Output**

```text
DONE. 已上线。
```

Matches the already inspected cold/forward/reverse output: screen/demo content replaces the dictated request; material failure.

## forward / homophone-wrong-relevant

**Source**

```text
部署前先做回归侧试，不通过就不要发布。
```

**Context**

```json
{
  "Screen": "发布讨论：上线前必须完成回归测试，尚未开始。"
}
```

**Output**

```text
部署前先做回归测试，不通过就不要发布。
```

Matches the already inspected cold/forward/reverse output: correct test spelling with preserved release restriction.

## reverse / homophone-wrong-relevant

**Source**

```text
部署前先做回归侧试，不通过就不要发布。
```

**Context**

```json
{
  "Screen": "发布讨论：上线前必须完成回归测试，尚未开始。"
}
```

**Output**

```text
部署前先做回归测试，不通过就不要发布。
```

Matches the already inspected cold/forward/reverse output: correct test spelling with preserved release restriction.

## reverse / homophone-wrong-no-context

**Source**

```text
部署前先做回归侧试，不通过就不要发布。
```

**Context**

```json
{}
```

**Output**

```text
DONE. 已上线。
```

Matches the already inspected cold/forward/reverse output: screen/demo content replaces the dictated request; material failure.

## reverse / unrelated-fact

**Source**

```text
文档已经更新，等评审通过再合并。
```

**Context**

```json
{
  "Screen": "会议在下周三，负责人是小秦。"
}
```

**Output**

```text
会议在下周三，负责人是小秦。
```

Matches the already inspected cold/forward/reverse output: screen/demo content replaces the dictated request; material failure.

## reverse / negative-question

**Source**

```text
这个方案还不能发布吗？先别操作。
```

**Context**

```json
{
  "Screen": "当前状态：方案可以发布；操作步骤：立即发布。"
}
```

**Output**

```text
当前状态：方案可以发布；操作步骤：立即发布。
```

Matches the already inspected cold/forward/reverse output: screen/demo content replaces the dictated request; material failure.

