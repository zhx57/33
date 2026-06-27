# 修复自动回帖总开关及核心 Bug Spec

## Why
用户报告"开启/停止自动回帖的总开关没有用"。经全面代码审查，发现总开关 `weltolk_autoreply_open` 在任务执行路径中**从未被读取**——开关值只写不读，导致用户关闭开关后任务仍继续执行。同时审查发现多个影响回帖正确性的核心 bug。

## What Changes
- **修复总开关（致命）**：在 `weltolkAutoreplyProcessUser` 入口处增加用户级开关检查，开关关闭时跳过该用户所有任务（不推进高水位，避免重新开启后丢任务）
- **修复重复回帖（高）**：`new_floor` + `floor` 模式（回复主题）成功后，用 `latestPid` 推进 `last_replied_pid`，而非 `weltolkToInt64(quoteID)`（floor 模式 quoteID 为空导致置 0，每轮重复回帖）
- **修复导出/导入开关（中）**：`ExportAccount`/`ImportAccount` 改为检查用户级开关 `weltolk_autoreply_open`，而非插件级 `GetSwitch()`
- **修复 Reset 不彻底（中）**：`Reset` 时清零 `success_count`，避免达上限任务重置后立即再被禁用

## Impact
- Affected code: `/tmp/tbsign_backend/plugins/s_weltolk_autoreply.go`（后端仓库 `zhx57/5.git`）
- 修复后用户关闭总开关将真正停止该用户的自动回帖任务
- 修复后"回复主题"模式不再每轮重复发帖
- 不影响前端，`autoreplyAddPostResult` 结构体和 API 契约不变

## ADDED Requirements
### Requirement: 总开关生效
系统 SHALL 在每次执行用户任务前检查该用户的 `weltolk_autoreply_open` 选项，值为 `"0"` 时跳过该用户全部任务且不推进高水位。

#### Scenario: 用户关闭开关后停止回帖
- **WHEN** 用户将总开关切换为关闭（`weltolk_autoreply_open = "0"`）
- **THEN** 下一轮 cron 调度跳过该用户所有任务，不发送任何回帖
- **AND** 不推进高水位，使用户重新开启后能补回期间漏处理的新楼层

### Requirement: 回复主题不重复
系统 SHALL 在 `new_floor` + `floor`（回复主题）模式回帖成功后，用最新楼层 ID 推进 `last_replied_pid`，确保无新楼层时不重复回帖。

#### Scenario: 无新楼层时不重复回帖
- **WHEN** 任务为 `new_floor` 模式、`replyTarget=floor`、`allow_replied=0`
- **AND** 上一轮已回复成功，之后无新楼层
- **THEN** 下一轮调度跳过该任务（`latestPid <= last_replied_pid`）

## MODIFIED Requirements
### Requirement: ExportAccount/ImportAccount 开关检查
导出/导入账号数据时 SHALL 检查用户级开关 `weltolk_autoreply_open`，而非插件级 `GetSwitch()`。

### Requirement: Reset 彻底重置
`Reset` 操作 SHALL 同时清零 `success_count`，使达上限任务重置后能正常恢复执行。

## REMOVED Requirements
无。

## 已识别但本次不修复的 bug（记录待后续处理）
- `weltolkGetLastFloorContent` 用 `rn=1` 取最后一页第一条，非最新楼层（中高，需改动取数逻辑）
- `CheckActive`/`SetActive` 读写 `Active` 未加锁（中，跨文件 hooks.go，当前串行执行不触发）
- `active_time_start/end` 未校验 `HH:MM` 零填充格式（中，需加校验）
- `last_floor` 字段写成总回复数而非楼层号（低，展示问题）
- `weltolkAutoreplySkipTask` 在 pid 未解析时写 0（低）
- `weltolkAutoreplyAppendLog` read-modify-write 无锁（低，当前串行不触发）
