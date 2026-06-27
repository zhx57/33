# Checklist
- [x] 总开关：`weltolkAutoreplyProcessUser` 入口处检查 `weltolk_autoreply_open`，值为 `"0"` 时 return 且不推进高水位
- [x] 重复回帖：`new_floor` + `floor` 模式成功后 `last_replied_pid` 用 `latestPid` 推进，不再置 0
- [x] 重复回帖：vcode 分支的 `last_replied_pid` 同样用 `latestPid` 推进
- [x] ExportAccount：检查用户级开关而非插件级 `GetSwitch()`
- [x] ImportAccount：检查用户级开关而非插件级 `GetSwitch()`
- [x] Reset：`Updates` 中包含 `"success_count": 0`
- [x] `/tmp/tbsign_backend` 执行 `go build ./...` 通过，无编译错误
