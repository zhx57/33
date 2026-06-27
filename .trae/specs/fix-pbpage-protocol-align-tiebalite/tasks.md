# Tasks

- [x] Task 1: 修复 CommonReq 的 `from` 字段值
  - [x] SubTask 1.1: 在 `/tmp/tbsign_backend/plugins/s_weltolk_autoreply.go` 的 `weltolkPbPageRequest` 函数中，将 CommonReq 字段6 的值从 `"app"` 改为 `"1021099l"`（极速版 APK `assets/channel` 真实渠道号）
  - [x] SubTask 1.2: 更新该行注释，说明 `from` 为极速版渠道号，来源 `TbadkCoreApplication.getFrom()` → `assets/channel`

- [x] Task 2: 请求 URL 增加 `cmd` 与 `format` query 参数
  - [x] SubTask 2.1: 将 `targetURL` 从 `https://tiebac.baidu.com/c/f/pb/page` 改为 `https://tiebac.baidu.com/c/f/pb/page?cmd=302001&format=protobuf`（对齐 TiebaLite `OfficialProtobufTiebaApi.kt:79`）

- [x] Task 3: multipart form-data 增加 `STOKEN` 独立字段
  - [x] SubTask 3.1: 在构造 multipart body 时，当 `stoken != ""` 时，在 `data` 字段之后追加一个 `name="STOKEN"` 的表单字段，值为 stoken（对齐 TiebaLite `ProtobufRequest.kt:39-42`）
  - [x] SubTask 3.2: 保持原有 Cookie 中 `STOKEN=<stoken>;` 不变（双保险）

- [x] Task 4: 编译与静态检查
  - [x] SubTask 4.1: 在 `/tmp/tbsign_backend` 执行 `go build ./...` 通过
  - [x] SubTask 4.2: 在 `/tmp/tbsign_backend` 执行 `go vet ./...` 通过

- [x] Task 5: 真实账号端到端验证
  - [x] SubTask 5.1: 用真实 BDUSS+stoken 编写 Python 测试脚本（复用 `/tmp/test_pbpage3.py` 模式），带 `from=1021099l`、`cmd=302001&format=protobuf`、multipart `STOKEN` 字段，请求一个公开帖子
  - [x] SubTask 5.2: 确认服务端返回有效 PbPageResponse（errorno=0 或无 error 字段，data 含 thread.reply_num 与 page.total_page），不再返回空数据或风控错误

# Task Dependencies
- Task 1, 2, 3 互相独立，可并行实现（但都在同一文件同一函数，实际串行编辑）
- Task 4 depends on Task 1, 2, 3
- Task 5 depends on Task 4
