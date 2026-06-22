# Checklist
- [ ] `weltolkAutoreplyProcessTask` 函数体内任务级代码缩进已修正为单层 tab，不再有多余缩进。
- [ ] 函数体内任务级跳过/错误分支中的 `continue` 已替换为 `return`。
- [ ] keyword/floor/subpost 内部循环的元素级 `continue` 保持为 `continue`，未被误改。
- [ ] 函数体内所有推进高水位的调用均使用 `highWaterKey` 参数。
- [ ] 原循环末尾已改为 `return`，不再存在 `break`。
- [ ] 引用未定义 `tasks` 的尾部代码块已被删除。
- [ ] 在 `/workspace/5` 执行 `go build ./...` 通过，无编译错误。
