# Tasks
- [x] Task 1: 修复总开关不生效
  - [x] SubTask 1.1: 在 `weltolkAutoreplyProcessUser` 函数入口处（高水位查询之前）增加用户级开关检查：读取 `_function.GetUserOption(weltolkAutoreplyOpenKey, uid)`，值为 `"0"` 时直接 `return`（不推进高水位）
- [x] Task 2: 修复回复主题模式重复回帖
  - [x] SubTask 2.1: 在 `weltolkAutoreplyProcessTask` 的 `triggerMode != "keyword"` 分支中，将 `latestPid`（最新楼层ID）保存到外层作用域变量，供成功/vcode 分支使用
  - [x] SubTask 2.2: 成功分支中，`!allowReplied` 时 `newLastRepliedPid` 改用保存的 `latestPid`（而非 `weltolkToInt64(quoteID)`）；vcode 分支同理
- [x] Task 3: 修复 ExportAccount/ImportAccount 开关检查
  - [x] SubTask 3.1: `ExportAccount` 中将 `pluginInfo.GetSwitch()` 改为检查 `_function.GetUserOption(weltolkAutoreplyOpenKey, strconv.Itoa(int(uid))) != "0"`
  - [x] SubTask 3.2: `ImportAccount` 中将 `pluginInfo.GetSwitch()` 改为检查 `_function.GetUserOption(weltolkAutoreplyOpenKey, strconv.Itoa(int(uid))) != "0"`
- [x] Task 4: 修复 Reset 不清 success_count
  - [x] SubTask 4.1: `Reset` 函数的 `Updates` map 中增加 `"success_count": 0`
- [x] Task 5: 编译验证
  - [x] SubTask 5.1: 在 `/tmp/tbsign_backend` 执行 `go build ./...` 确认无编译错误

# Task Dependencies
- Task 5 depends on Task 1, 2, 3, 4
