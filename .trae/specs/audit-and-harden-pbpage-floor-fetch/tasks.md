# Tasks

- [x] Task 1: 为 PbPage 链路补全诊断日志
  - [x] SubTask 1.1: 在 `weltolkPbPageRequest` 返回前输出 slog.Debug（成功，含 tid/pn/rn/HTTP状态/响应长度）与 slog.Warn（err 或空响应）
  - [x] SubTask 1.2: 在 `weltolkPbPageParse` 输出 slog.Debug（errorno/errmsg/replyCount/totalPage/floors数），errorno!=0 时升级 Warn
  - [x] SubTask 1.3: 在 `weltolkGetReplyCount` 失败分支输出 slog.Warn 含 tid/err/errorno，成功分支 Debug 含 replyCount/totalPage
  - [x] SubTask 1.4: 在 `weltolkGetLastFloorContent` 失败/回退分支输出 slog.Warn，成功 Debug 含 floor_count/latest_floor/latest_pid

- [x] Task 2: 加固 `weltolkGetLastFloorContent` 越界页兜底
  - [x] SubTask 2.1: 请求 `(pn, rn)` 后若 floors 为空且 `pn > 1`，回退请求 `(1, rn)`
  - [x] SubTask 2.2: 回退仍空时输出 Warn 并返回 nil
  - [x] SubTask 2.3: 成功时正常反转返回

- [x] Task 3: 新增独立诊断 API `POST /plugins/weltolk_autoreply/diagnose`
  - [x] SubTask 3.1: 在插件路由注册处新增 `POST /diagnose` handler（参照现有 switch/list handler 写法）
  - [x] SubTask 3.2: handler 取当前登录 uid，查 `TcBaiduid` 取 pid 最小绑定，`GetCookie(pid, true)` 得 bduss/stoken
  - [x] SubTask 3.3: 依次调用 weltolkPbPageRequest → weltolkPbPageParse → weltolkGetReplyCount → weltolkGetLastFloorContent，收集每步状态
  - [x] SubTask 3.4: 返回 JSON `{code:200, data:{steps:[{step,ok,...}], summary}}`，未绑定账号返回 code=400

- [x] Task 4: 新增单元测试 `plugins/s_weltolk_autoreply_test.go`
  - [x] SubTask 4.1: `TestPbPageParseSynthetic` —— 构造含 error/page/thread/post_list 的 protobuf 字节，验证 weltolkPbPageParse 解析出正确的 errorno/totalPage/replyCount/floors
  - [x] SubTask 4.2: `TestPbPageRealRequest` —— 真实网络调用（`-short` 跳过），验证 weltolkGetReplyCount 与 weltolkGetLastFloorContent 返回 ok=true 且 floors 非空

- [x] Task 5: 编译与静态检查
  - [x] SubTask 5.1: `go build ./...` 通过
  - [x] SubTask 5.2: `go vet ./...` 通过

- [x] Task 6: 真实账号端到端验证
  - [x] SubTask 6.1: 用提供的 BDUSS+stoken 跑 `TestPbPageRealRequest`（或 Python 等价脚本调用 diagnose 逻辑），确认 replyCount/totalPage/floors 全部正确
  - [x] SubTask 6.2: 确认日志输出格式可读，失败场景（构造 errorno!=0）能输出 Warn

# Task Dependencies
- Task 1, 2 在同一文件可串行实现
- Task 3 依赖 Task 1（复用日志）但可独立实现 handler
- Task 4 依赖 Task 1, 2, 3（测试覆盖新逻辑）
- Task 5 依赖 Task 1-4
- Task 6 依赖 Task 5
