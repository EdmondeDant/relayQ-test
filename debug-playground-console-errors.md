[OPEN]

# Debug Session: playground-console-errors

## Symptoms
- 浏览器控制台持续出现 `http://127.0.0.1:7777/event` 的 CORS 报错。
- `GET /api/v1/playground/assets/14/content` 反复返回 `401` 或 `404`。
- `GET /api/v1/auth/me?timezone=Asia/Shanghai` 返回 `500`，前端打印 `Auto-refresh user failed`。
- 当前目标是判断这些报错的严重性、修掉真实问题，并验证 Playground 页面恢复稳定。

## Falsifiable Hypotheses
1. `/api/v1/auth/me` 的 `500` 发生在身份汇总查询阶段，而不是用户主表查询阶段。
2. `assets/14/content` 报错来自旧创作记录中的失效资产，数据库记录与磁盘文件或权限状态不一致。
3. `7777/event` 的 CORS 报错来自前端残留调试请求，不影响图片/视频主业务接口。
4. 前端恢复媒体时缺少失败缓存或降级处理，导致同一失效资产被重复请求并持续刷错误。

## Evidence Plan
- 在 `/api/v1/auth/me` 现有埋点基础上补充分支结果，确认失败阶段与具体错误。
- 在资产读取链路补充最小埋点，记录 `asset_id`、`storage_key`、打开文件结果与权限状态。
- 在前端媒体恢复链路补充一次性失败观测点，确认是否重复请求同一失效 URL。
- 复现后对照运行日志，明确哪些属于调试噪音，哪些属于真实业务故障。

## Status
- 已确认前端代码中存在直接请求 `http://127.0.0.1:7777/event` 的调试逻辑。
- 已确认前端恢复创作记录时会主动 `fetch` 受保护媒体 URL。
- 下一步先补充最小埋点并复现，再做针对性修复。
