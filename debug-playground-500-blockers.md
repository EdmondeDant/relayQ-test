[OPEN]

# Debug Session: playground-500-blockers

## Symptoms
- `AI 生图` 与 `图片编辑` 无法开始，因为 Playground 缺少可用 API Key。
- `/api/v1/groups/available`、`/api/v1/keys`、`/api/v1/playground/records`、`/api/v1/auth/me` 等接口返回 `500`。
- 目标是恢复到可以正常文生图、正常图片编辑。

## Falsifiable Hypotheses
1. 某个公共当前用户查询路径 panic，导致多个依赖登录态的接口统一 `500`。
2. 分组或 API Key 的仓储/服务层在处理现有脏数据时触发未处理错误。
3. 某个 DTO/mapper 在序列化 groups/keys/playground 数据时触发运行时错误。
4. 当前环境数据库或配置状态异常，多个接口共享的前置依赖失败。

## Evidence Plan
- 先抓运行日志与目标接口响应，定位第一个共同失败点。
- 只做插桩，不先改业务逻辑。
- 证据确认后做最小修复，再回归验证文生图与图片编辑。

## Progress
- 已确认前后端可编译。
- 已确认上述接口当前在本地运行环境返回 `500`。

## Next
- 为公共入口与可疑 handler/service 增加最小运行时埋点。
- 复现并比对 pre-fix / post-fix 证据。
