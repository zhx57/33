# 重构 weltolkAutoreplyProcessTask 循环体 Spec

## Why
将原 `Action()` 中复制到 `weltolkAutoreplyProcessTask` 的代码适配为单任务处理函数：去掉多余缩进、修正控制流与水位 key，并清理因复制遗留的循环/task 级代码，使其通过编译并保持行为一致。

## What Changes
- 修正 `/workspace/5/plugins/s_weltolk_autoreply.go` 中 `weltolkAutoreplyProcessTask` 函数体内多余的一层缩进。
- 将任务级跳过逻辑中的 `continue` 替换为 `return`；保留 `keyword` / `floor` / `subpost` 内部循环中用于跳到下一个元素的 `continue`。
- 将函数体内所有推进高水位的 `_function.SetOption(weltolkAutoreplyHighWaterKey, int(taskID)+1)` 替换为 `_function.SetOption(highWaterKey, int(taskID)+1)`。
- 删除原循环末尾的 `break`，改为直接 `return`。
- 删除复制遗留的、引用未定义 `tasks` 变量的尾部代码块。
- 修改后在 `/workspace/5` 执行 `go build ./...` 确保无编译错误。

## Impact
- Affected specs: weltolk-autoreply 插件单任务执行流程。
- Affected code: `/workspace/5/plugins/s_weltolk_autoreply.go` 中的 `weltolkAutoreplyProcessTask` 函数。

## MODIFIED Requirements
### Requirement: 单任务处理函数控制流
- The function `weltolkAutoreplyProcessTask` SHALL process exactly one task and exit via `return` instead of loop-level control statements.
- **WHEN** a task-level skip/error condition is encountered inside the function, **THEN** it SHALL call `_function.SetOption(highWaterKey, int(taskID)+1)` and `return`.
- **WHEN** the task execution branch completes (success/vcode/failure), **THEN** it SHALL advance the high-water mark using `highWaterKey` and `return`.

## REMOVED Requirements
### Requirement: 函数内循环/task-level 尾部逻辑
- **Reason**: The function now receives a single task; the caller already handles iteration over `tasks` and resets the high-water mark when no enabled tasks exist.
- **Migration**: Remove the leftover loop wrapper and the trailing `len(tasks)==0` block from this function.
