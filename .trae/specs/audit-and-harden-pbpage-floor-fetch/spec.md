# 彻查并加固 PbPage 楼层获取链路 Spec

## Why

用户反馈"依然获取回复数、楼层失败"。经彻查：协议层已正确（Python + Go 原生 HTTP 双重验证 tid=9534834391 返回 HTTP 200 / errorno=0 / 53 回复 / 21 楼层），后端 `go build ./...` 通过。但当前 `weltolkPbPageRequest → weltolkPbPageParse → weltolkGetReplyCount/weltolkGetLastFloorContent → weltolkAutoreplyProcessTask` 调用链存在三类隐患，使任何一环异常都被笼统报告为"获取回复数失败/获取楼层内容失败"，无法定位真因（可能是部署未更新、特定帖子越界页、Transport 差异、stoken 缺失等）。本 spec 通过**补全诊断日志 + 加固边界处理 + 提供独立测试入口**彻底消除盲区，让失败时可立即看到根因。

## What Changes

- **补全诊断日志**：在 `weltolkPbPageRequest`、`weltolkPbPageParse`、`weltolkGetReplyCount`、`weltolkGetLastFloorContent` 关键节点输出 slog.Debug/Warn 日志，含 tid、pn、rn、HTTP 状态、响应长度、errorno、errmsg、replyCount、totalPage、floors 数量。失败时升级为 slog.Warn 含完整上下文，使"获取回复数失败"日志可直接看到是 HTTP 错误、errorno 非零、还是解析为空
- **加固 `weltolkGetLastFloorContent` 越界页处理**：当 `pn > totalPage` 时（百度对越界页返回 cur_page=pn 但数据可能为空或回退首页），改为请求 `pn=totalPage` 兜底；若仍为空再退 `pn=1`，避免因 totalPage 计算偏差导致获取楼层失败
- **加固 `weltolkGetReplyCount` 的 totalPage 兜底**：当 `result.totalPage < 1` 时已设为 1（现有逻辑保留），同时当 `replyCount` 为 0 但 floors 非空时，用 floors 数量兜底避免误判
- **新增独立诊断 API**：在插件路由注册 `POST /plugins/weltolk_autoreply/diagnose`，接受 `tid`，用当前登录用户绑定的百度账号实际调用完整链路（PbPageRequest → Parse → GetReplyCount → GetLastFloorContent），返回每一步的状态与原始数据摘要，供前端/运维一键诊断"为什么失败"，无需 SSH 看日志
- **新增单元测试**：`plugins/s_weltolk_autoreply_test.go` 含 `TestPbPageRealRequest`（真实网络，`-short` 跳过）与 `TestPbPageParseSynthetic`（构造的 protobuf 字节，验证解析字段号与边界），确保解析逻辑可回归

## Impact

- Affected specs: `use-pbpage-protobuf-for-floors`、`fix-pbpage-protocol-align-tiebalite`（本 spec 在其基础上补可观测性与健壮性，不改变协议）
- Affected code:
  - `/tmp/tbsign_backend/plugins/s_weltolk_autoreply.go`（weltolkPbPageRequest/Parse/GetReplyCount/GetLastFloorContent 加日志与边界，新增 diagnose handler 与路由）
  - `/tmp/tbsign_backend/plugins/s_weltolk_autoreply_test.go`（新增测试文件）

## ADDED Requirements

### Requirement: PbPage 链路可观测性

每个 PbPage 相关函数 SHALL 在关键节点输出结构化日志，使失败时无需复现即可定位根因。

#### Scenario: 请求失败时日志含完整上下文
- **WHEN** `weltolkPbPageRequest` 返回 err，或 `weltolkPbPageParse` 得到 errorno!=0，或 floors 为空
- **THEN** slog.Warn 输出含 tid、pn、rn、HTTP 状态码、响应字节数、errorno、errmsg、replyCount、totalPage、floors 数量
- **AND** 调用方 `weltolkAutoreplyProcessTask` 的"获取回复数失败/获取楼层内容失败"日志能关联到上述 Warn

#### Scenario: 成功时 Debug 日志可追踪
- **WHEN** 链路成功执行
- **THEN** slog.Debug 输出每步关键指标（replyCount、totalPage、floors 数量、最新楼层 floor 号与 pid）

### Requirement: 越界页兜底

`weltolkGetLastFloorContent` SHALL 在请求页越界或返回空时自动回退，避免因 totalPage 偏差导致获取楼层失败。

#### Scenario: pn 超过 totalPage
- **WHEN** 调用方传入 `pn=totalPage` 但百度返回 floors 为空（因 totalPage 计算偏差）
- **THEN** 自动回退请求 `pn=1`（首页必有数据）
- **AND** 若首页仍空，返回 nil 并输出 Warn 日志说明"首页也返回空，疑似风控或帖子已删"

### Requirement: 独立诊断 API

系统 SHALL 提供 `POST /plugins/weltolk_autoreply/diagnose` 端点，一键诊断指定 tid 的完整获取链路。

#### Scenario: 诊断成功
- **WHEN** 已登录用户 POST `/plugins/weltolk_autoreply/diagnose` body `tid=9534834391`
- **THEN** 用该用户绑定的百度账号（取 pid 最小的绑定）调用完整链路
- **AND** 返回 JSON 含每步状态：`{step:"pbpage_request", ok:true, http_status:200, resp_len:23612}`、`{step:"parse", errorno:0, reply_count:53, total_page:1, floors:21}`、`{step:"get_reply_count", ok:true, reply_count:53, total_page:1}`、`{step:"get_last_floor", ok:true, floor_count:21, latest_floor:30, latest_pid:153630281595}`
- **OR** 失败时返回对应 step 的 `ok:false` 与 `error` 详情

#### Scenario: 用户未绑定百度账号
- **WHEN** 用户未绑定任何百度账号
- **THEN** 返回 code=400，message="未绑定百度账号"

## MODIFIED Requirements

### Requirement: weltolkGetLastFloorContent 楼层获取

`weltolkGetLastFloorContent(tid, bduss, stoken, limit, totalPage)` SHALL：

1. 计算 `pn := totalPage`（若 <1 则 1），`rn := max(30, limit)`
2. 请求 `(pn, rn)`，解析得 floors
3. 若 floors 为空且 `pn > 1`，回退请求 `(1, rn)` 并解析
4. 若仍为空，输出 slog.Warn 含 tid/pn/rn/totalPage，返回 nil
5. 否则反转 floors（降序，`[0]` 为最新），返回

每步输出 slog.Debug（成功）或 slog.Warn（回退/失败）日志。

### Requirement: weltolkGetReplyCount 回复数获取

`weltolkGetReplyCount(tid, bduss, stoken)` SHALL：

1. 请求 `(pn=1, rn=30)`，解析
2. 若 err 或 respData 空，输出 Warn，返回 `ok=false`
3. 若 errorno!=0，输出 Warn 含 errorno/errmsg，返回 `ok=false`
4. 若 totalPage<1，置 1
5. 输出 Debug 含 replyCount/totalPage/floors 数量
6. 返回 `(replyCount, totalPage, true)`
