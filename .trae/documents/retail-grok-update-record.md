# Retail Grok Key 更新记录

## 更新时间

2026-07-06

## 更新目标

为零散客户试用或小额售卖场景增加独立 Grok 零售 Key 功能，支持管理员批量生成、客户自助查询用量、客户查看接口说明。

## 本次新增能力

1. 管理端新增零售 Grok Key 批量生成页面：`/admin/retail-grok`。
   - 本地开发访问地址：`http://localhost:3001/admin/retail-grok`
2. 客户查询页新增：`/retail/grok/key-usage`。
   - 本地开发访问地址：`http://localhost:3001/retail/grok/key-usage`
3. 客户接口说明页新增：`/retail/grok/docs`。
   - 本地开发访问地址：`http://localhost:3001/retail/grok/docs`。
4. 后端新增零售专用接口：
   - `POST /retail/v1/chat/completions`
   - `POST /retail/v1/images/generations`
   - `POST /retail/v1/images/edits`
   - `POST /retail/v1/videos/generations`
   - `POST /retail/v1/videos/edits`
   - `POST /retail/v1/videos/extensions`
   - `GET /retail/v1/videos/:request_id`
   - `GET /retail/v1/usage`
   - `GET /api/v1/retail-grok/usage`
5. 客户文档页已按 xAI 官方接口参数补充：
   - Grok 4.3 多模态识图
   - 文生图
   - 图生图 / 多图编辑
   - 文生视频
   - 图生视频 / 参考图生成视频

## 页面入口记录

- 管理端批量导出/生成零售 Key：`http://localhost:3001/admin/retail-grok`
- 客户零售 Key 用量查询：`http://localhost:3001/retail/grok/key-usage`
- 客户接口说明页：`http://localhost:3001/retail/grok/docs`

## 生产隔离检查

本次实现遵循“零影响生产优先”：

1. 数据隔离
   - 新增独立表 `retail_grok_keys`。
   - 新增独立表 `retail_grok_usage_logs`。
   - 不写入现有 `api_keys`。
   - 不写入现有 `usage_logs`。
   - 不写入现有 `usage_billing`。

2. 鉴权隔离
   - 新增 `RetailGrokKeyAuthMiddleware`。
   - 零售 Key 使用 `rgk-` 前缀。
   - 不复用现有 `APIKeyAuthMiddleware`。
   - 不改变现有 `/v1` API Key 鉴权逻辑。

3. 路由隔离
   - 零售调用入口使用 `/retail/v1/*`。
   - 客户用量查询使用 `/retail/v1/usage` 或 `/api/v1/retail-grok/usage`。
   - 管理端生成接口位于 `/api/v1/admin/retail-grok/*`，仍受管理员鉴权保护。
   - 未把零售入口挂到普通用户导航或普通用户设置页。

4. 前端隔离
   - 新增独立客户页和文档页。
   - 新增独立管理页。
   - 普通登录用户不会在现有用户侧菜单中看到零售功能。

## 已修复问题

1. 管理端生成接口 404：重启后端后确认路由已加载。
2. 客户查询页无显示：前端改走稳定 `/api/v1/retail-grok/usage` 代理入口。
3. 客户接口说明不完整：补齐多模态、文生图、图生图、文生视频、图生视频示例和官方参数。
4. 查询页说明文案更新为：`如需技术支持或长期合作 请访问主站：www.relayq.top`。
5. 接口说明页示例域名更新为：`https://www.reayq.top`。
6. `/api/v1/retail-grok/usage` 已加入零售用量查询白名单，避免额度耗尽后客户无法查看自己的用量。

## 验证记录

1. 后端直连查询验证通过：
   - `GET http://localhost:3000/api/v1/retail-grok/usage`
   - 使用零售 Key 返回 JSON 用量数据。
2. 前端代理查询验证通过：
   - `GET http://localhost:3001/api/v1/retail-grok/usage`
   - 使用零售 Key 返回 JSON 用量数据。
3. 客户查询页访问验证通过：
   - `http://localhost:3001/retail/grok/key-usage`
4. 客户接口说明页访问验证通过：
   - `http://localhost:3001/retail/grok/docs`
5. 前端类型检查曾通过一次；后续纯文案更新后，IDE 终端执行 `pnpm exec vue-tsc --noEmit` 出现挂起/跳过，未取得新的最终结果。

## 当前注意事项

1. 零售功能依赖新增 migration：
   - `backend/migrations/147_create_retail_grok_keys.sql`
   - `backend/migrations/148_create_retail_grok_usage_logs.sql`
2. 生产部署前需确认 migration 已执行。
3. 当前零售多模态文档使用 `/retail/v1/chat/completions` 示例；xAI 官方最新推荐的 `/v1/responses` 尚未作为零售入口单独暴露。
