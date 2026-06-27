# Checklist

- [x] CommonReq 字段6 `from` 值为 `"1021099l"`（极速版真实渠道号），不再是 `"app"`
- [x] `from` 字段注释说明来源（`TbadkCoreApplication.getFrom()` → `assets/channel`）
- [x] 请求 URL 含 `cmd=302001` 与 `format=protobuf` query 参数
- [x] 当 `stoken != ""` 时，multipart form-data 含 `name="STOKEN"` 独立字段，值=stoken
- [x] Cookie 中 `STOKEN=<stoken>;` 保留（与 multipart STOKEN 双保险）
- [x] `/tmp/tbsign_backend` 执行 `go build ./...` 通过
- [x] `/tmp/tbsign_backend` 执行 `go vet ./...` 通过
- [x] 真实账号 Python 测试请求返回有效 PbPageResponse（含 reply_num 与 total_page），不再"获取回复数失败"
  - 验证结果：tid=9534834391，HTTP 200，errorno=0，reply_num=53，total_page=1，post_count=21，PASS
