# 修复 PbPage 请求协议对齐 TiebaLite 完整版 Spec

## Why

当前 `weltolkPbPageRequest`（commit `732f49e` stoken 补全后）依然"获取回复数失败"。对比 `zzc10086/TiebaLite`（完整版第三方客户端，用 wire protobuf 库，协议实现权威）后发现实现存在确定的协议错误：CommonReq 的 `from` 字段值用了 `"app"`，而极速版真实渠道号是 `1021099l`（从极速版 APK 的 `assets/channel` 文件读出，经 `TbadkCoreApplication.getFrom()` → `TbConfig.getFrom()` 填入 CommonReq 字段6）。`from` 是百度服务端风控识别客户端合法性的关键字段，错误值会导致服务端判定为非法客户端、返回空数据或错误，这正是"获取回复数失败"的真正根因（stoken 补全虽正确但 from 错误未修复）。

## What Changes

- **修复 `from` 字段值**：CommonReq 字段6 从 `"app"` 改为极速版真实渠道号 `"1021099l"`（依据：极速版 APK `assets/channel` 文件内容，对应 `TbadkCoreApplication.getFrom()`）
- **URL 增加 `cmd` 与 `format` query 参数**：请求 URL 从 `https://tiebac.baidu.com/c/f/pb/page` 改为 `https://tiebac.baidu.com/c/f/pb/page?cmd=302001&format=protobuf`（依据：TiebaLite `OfficialProtobufTiebaApi.kt:79` 的 `@POST("/c/f/pb/page?cmd=302001&format=protobuf")`，`format=protobuf` 是服务端返回 protobuf 格式的开关）
- **multipart form-data 增加 `STOKEN` 独立字段**：除 Cookie 的 `STOKEN` 外，在 multipart body 中额外以 `name="STOKEN"` 表单字段提交 stoken（依据：TiebaLite `ProtobufRequest.kt:39-42` 的 `buildProtobufRequestBody` 在 `needSToken=true` 时将 STOKEN 作为独立 form-data 字段加入，与 Cookie 双保险）

## Impact

- Affected specs: `use-pbpage-protobuf-for-floors`（其 Task 2.2/Task 8.1 实现的 URL 与 CommonReq 字段需以此 spec 为准修正）
- Affected code: `/tmp/tbsign_backend/plugins/s_weltolk_autoreply.go` 的 `weltolkPbPageRequest` 函数（约 636-718 行）

## ADDED Requirements

### Requirement: PbPage 请求协议对齐极速版真实客户端

`weltolkPbPageRequest` 构造的请求 SHALL 在以下三处与极速版真实客户端一致，确保通过服务端风控：

1. CommonReq 的 `from` 字段（字段6）值 SHALL 为极速版渠道号 `"1021099l"`
2. 请求 URL SHALL 包含 `cmd=302001` 与 `format=protobuf` query 参数
3. 当 stoken 非空时，multipart form-data SHALL 包含 `name="STOKEN"` 的独立表单字段（值同 stoken），与 Cookie 中的 `STOKEN=` 并存

#### Scenario: 真实账号请求成功获取回复数
- **WHEN** 自动回帖任务对真实帖子调用 `weltolkGetReplyCount`
- **THEN** `weltolkPbPageRequest` 发出的请求含正确的 `from=1021099l`、URL 含 `cmd=302001&format=protobuf`、multipart 含 `STOKEN` 字段
- **AND** 服务端返回有效 PbPageResponse，`weltolkPbPageParse` 解析出 `replyCount >= 0` 与 `totalPage >= 1`
- **AND** `weltolkGetReplyCount` 返回 `ok=true`，不再输出"获取回复数失败"

#### Scenario: stoken 为空时降级
- **WHEN** 用户账号未绑定 stoken（stoken=""）
- **THEN** multipart 不添加 `STOKEN` 字段，Cookie 也不含 `STOKEN=`
- **AND** 请求仍带 `from=1021099l` 与 `cmd=302001&format=protobuf`，尽力尝试（可能因缺 stoken 失败，但不影响协议正确性）

## MODIFIED Requirements

### Requirement: weltolkPbPageRequest 请求构造

`weltolkPbPageRequest(tid, bduss, stoken, pn, rn)` 构造的 HTTP 请求 SHALL：

- URL: `https://tiebac.baidu.com/c/f/pb/page?cmd=302001&format=protobuf`
- Method: POST
- Header: `Content-Type: multipart/form-data; boundary=...`、`User-Agent: bdtb for Android 12.41.7.1`、`x_bd_data_type: protobuf`、`Accept-Encoding: gzip`、`Connection: keep-alive`、`Cookie: BDUSS=<bduss>; STOKEN=<stoken>;`（stoken 非空时）
- Body (multipart/form-data):
  - 字段 `data`（filename="file"）：PbPageReqIdl{data=DataReq} protobuf 二进制
  - 字段 `STOKEN`（仅 stoken 非空时）：值=stoken
- DataReq 字段：mark=0(2)、back=0(3)、kz=tid(4)、lz=0(5)、r=0(6)、with_floor=1(8)、floor_rn=3(9)、rn(13)、scr_w=1080(14)、scr_h=1920(15)、pn(18)、st_type="tb_frslist"(19)、common=CommonReq(25)
- CommonReq 字段：_client_type=2(1)、_client_version="12.41.7.1"(2)、_client_id="888888888888888"(3)、_phone_imei="000000000000000"(5)、**from="1021099l"(6)**、cuid(7)、_timestamp(8)、model="2201123C"(9)、BDUSS=bduss(10)、stoken(30, 仅非空时)
