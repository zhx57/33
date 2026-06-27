# Checklist

- [x] `weltolkPbPageRequest` 成功输出 slog.Debug（tid/pn/rn/HTTP/响应长度），失败输出 slog.Warn
- [x] `weltolkPbPageParse` 输出 Debug（errorno/errmsg/replyCount/totalPage/floors数），errorno!=0 升级 Warn
- [x] `weltolkGetReplyCount` 失败输出 Warn（tid/err/errorno），成功输出 Debug（replyCount/totalPage）
- [x] `weltolkGetLastFloorContent` 失败/回退输出 Warn，成功输出 Debug（floor_count/latest_floor/latest_pid）
- [x] `weltolkGetLastFloorContent` 在 floors 空且 pn>1 时回退请求 pn=1
- [x] 回退仍空时返回 nil 并输出 Warn
- [x] 新增 `POST /plugins/weltolk_autoreply/diagnose` 路由
- [x] diagnose handler 取 uid 绑定的最小 pid 账号，未绑定返回 code=400
- [x] diagnose 返回 JSON 含每步 step/ok 与关键指标
- [x] 新增 `TestPbPageParseSynthetic` 验证解析字段号与边界
- [x] 新增 `TestPbPageRealRequest`（-short 跳过）验证真实链路
- [x] `go build ./...` 通过
- [x] `go vet ./...` 通过
- [x] 真实账号端到端验证 replyCount/totalPage/floors 全部正确
- [x] 构造 errorno!=0 场景验证 Warn 日志输出
