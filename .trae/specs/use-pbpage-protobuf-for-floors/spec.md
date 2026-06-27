# 用极速版 PbPage protobuf 接口替换失败的 JSON 楼层获取 Spec

## Why
自动回帖一直输出"获取回复数失败"和"获取楼层内容失败"。现有代码用 `http://c.tieba.baidu.com/c/f/pb/page` 的 **JSON 表单方式** 获取楼层，而贴吧极速版 APK 实际使用同一 URL 的 **protobuf 二进制协议**（`pbPageHttpResponseMessage.decodeInBackGround(byte[])` 用 Wire/protobuf 解码）。JSON 方式已无法正确返回数据，需严格按极速版逻辑改用 protobuf。

## What Changes
- 新增 `weltolkPbPageRequest` 函数：按极速版 `tbclient.PbPage.PbPageReqIdl` 结构手写 protobuf 编码请求体（CommonReq + DataReq），POST 到 `https://tiebac.baidu.com/c/f/pb/page`，Content-Type `multipart/form-data`（data 字段放 protobuf 二进制），Header `x_bd_data_type: protobuf`
- 新增 `weltolkPbPageParse` 函数：按 `PbPageResIdl` 结构手写 protobuf 解码响应，提取 `error.errorno/errmsg`、`page.total_page`、`thread.reply_num`、`post_list[]`（id/floor/author_id/author.name_show+portrait/content[].text/sub_post_list）
- 重写 `weltolkGetReplyCount`：改用 `weltolkPbPageRequest`（pn=1, rn=1），从 protobuf 响应解析 `thread.reply_num` 和 `page.total_page`
- 重写 `weltolkGetLastFloorContent`：改用 `weltolkPbPageRequest`（pn=totalPage, rn=较大值如30），从 protobuf 响应解析 post_list，取**最后一个**元素作为最新楼层（修复 rn=1 取首页第一条的旧 bug）
- 删除 `weltolkCallTiebaJSONAPI`（不再使用）
- `weltolkFloor` 结构体保持不变，解析后填充

## Impact
- Affected code: `/tmp/tbsign_backend/plugins/s_weltolk_autoreply.go`（后端仓库 `zhx57/5.git`）
- 依赖 n0099/tbclient.protobuf 的 proto 字段编号（已逆向确认）
- 不影响前端、API 契约、`autoreplyAddPostResult` 结构体
- 调用点不变（`weltolkGetReplyCount`/`weltolkGetLastFloorContent` 签名不变）

## ADDED Requirements
### Requirement: 极速版 PbPage protobuf 请求
系统 SHALL 用 protobuf 二进制协议请求 `c/f/pb/page`，请求体为 `PbPageReqIdl{data: DataReq{common: CommonReq, kz=tid, pn, rn, ...}}` 编码后的二进制，通过 multipart `data` 字段提交，带 `x_bd_data_type: protobuf` 头。

#### Scenario: 成功获取楼层
- **WHEN** 调用 `weltolkGetLastFloorContent(tid, bduss, 30, totalPage)`
- **THEN** 用 protobuf 请求最后一页，返回 post_list 解析出的 `[]*weltolkFloor`
- **AND** 列表最后一个元素是最新楼层

### Requirement: 回复数与总页数
系统 SHALL 从 PbPage 响应的 `thread.reply_num` 获取回复数，从 `page.total_page` 获取总页数。

#### Scenario: 获取回复数
- **WHEN** 调用 `weltolkGetReplyCount(tid, bduss)`
- **THEN** 返回 `(replyCount, totalPage, true)`

### Requirement: 错误判断
系统 SHALL 从响应 `error.errorno` 判断成功（==0），非0时记录 `errmsg`。

## MODIFIED Requirements
### Requirement: 最新楼层选取
`weltolkGetLastFloorContent` SHALL 请求最后一页（pn=totalPage）并用较大 rn（30），取返回 post_list 的最后一个元素作为最新楼层，而非 rn=1 取第一条。

## REMOVED Requirements
### Requirement: JSON 表单获取楼层
**Reason**: 贴吧该接口要求 protobuf，JSON 方式返回数据异常导致"获取回复数失败/获取楼层内容失败"
**Migration**: 所有调用 `weltolkCallTiebaJSONAPI` 的地方改用 `weltolkPbPageRequest`；`weltolkGetReplyCount`/`weltolkGetLastFloorContent` 内部实现替换，外部签名不变

## 字段编号参考（来自 n0099/tbclient.protobuf，与极速版 APK 一致）
- CommonReq: BDUSS=10, tbs=11, _client_type=1, _client_version=2, cuid=7, _phone_imei=5, model=9, ka=15, _timestamp=8
- DataReq: kz=4, pn=18, rn=13, with_floor=8, floor_rn=9, lz=5, r=6, back=3, mark=2, scr_w=14, scr_h=15, st_type=19
- PbPageReqIdl: data=1
- PbPageResIdl: error=1, data=2
- Error: errorno=1, errmsg=2
- DataRes: page=3, thread=8, post_list=6
- Page: total_page=5, current_page=3
- ThreadInfo: id=1, tid=2, reply_num=4
- Post: id=1, floor=3, content=5, sub_post_list=15, author_id=19, author=23
- SubPostList: sub_post_list=1
- SubPost: id=1, content=4, author_id=8
- PbContent: type=1, text=2
- User: name_show=4, portrait=2, name=1
