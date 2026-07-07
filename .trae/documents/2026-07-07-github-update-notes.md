# 2026-07-07 更新记录：零售 Grok Key、接口文档、客户端安装与生产隔离

## 1. 更新背景

本次更新围绕两个目标展开：

1. 为零散客户、小额试用、闲鱼/二手小店售卖 token 场景增加独立的 Grok 零售 Key 能力。
2. 在不影响现有生产 API Key、现有 `/v1` 主链路、现有每日统计和用量体系的前提下，提供独立管理、独立调用、独立用量查询和独立接口说明。

本次改动坚持“零影响生产优先”：零售功能使用独立路由、独立表、独立鉴权、独立前端页面，不接入现有生产 API Key 表和主用量统计链路。

## 2. 主要功能新增

### 2.1 管理端：Grok 零售 Key 批量生成

新增管理端页面：

- 本地开发地址：`http://localhost:3001/admin/retail-grok`
- 前端文件：`frontend/src/views/admin/RetailGrokView.vue`
- API 文件：`frontend/src/api/admin/retailGrok.ts`

支持能力：

1. 选择 xAI 分组。
2. 设置名称前缀。
3. 设置生成数量。
4. 设置有效期天数。
5. 设置 Token 上限。
6. 设置图片上限。
7. 设置视频上限。
8. 批量生成 `rgk-` 前缀零售 Key。
9. 查看单个零售 Key 用量。
10. 删除零售 Key。
11. 导出全部零售 Key CSV。

管理端后端接口：

- `POST /api/v1/admin/retail-grok/batch-generate`
- `GET /api/v1/admin/retail-grok/keys`
- `GET /api/v1/admin/retail-grok/keys/:id/usage`
- `DELETE /api/v1/admin/retail-grok/keys/:id`

相关后端文件：

- `backend/internal/handler/admin/retail_grok_handler.go`
- `backend/internal/server/routes/admin.go`
- `backend/internal/service/retail_grok_service.go`
- `backend/internal/repository/retail_grok_key_repo.go`
- `backend/internal/repository/retail_grok_usage_log_repo.go`

### 2.2 客户端：零售 Key 用量查询页

新增客户公开查询页：

- 本地开发地址：`http://localhost:3001/retail/grok/key-usage`
- 前端文件：`frontend/src/views/public/RetailGrokKeyUsageView.vue`
- API 文件：`frontend/src/api/retailGrok.ts`

客户输入 `rgk-` Key 后可查看：

1. Key 名称。
2. Key 状态。
3. 到期时间。
4. Token 已用 / 总额。
5. 图片已用 / 总额。
6. 视频已用 / 总额。
7. 最近请求记录。
8. 接口说明入口。

页面文案已更新为：

`如需技术支持或长期合作 请访问主站：www.relayq.top`

客户查询后端接口：

- `GET /retail/v1/usage`
- `GET /api/v1/retail-grok/usage`

其中 `/api/v1/retail-grok/usage` 用于前端开发环境稳定走 `/api` 代理，避免 Vite SPA fallback 返回 HTML。

### 2.3 客户端：零售 Grok 接口说明页

新增客户接口说明页：

- 本地开发地址：`http://localhost:3001/retail/grok/docs`
- 前端文件：`frontend/src/views/public/RetailGrokDocsView.vue`

文档页已补充：

1. 明确兼容 OpenAI API 调用格式。
2. 明确零售 `base_url`：`https://www.reayq.top/retail/v1`。
3. 明确零售 Key 只能调用 `/retail/v1/*`，不能调用普通 `/v1/*`。
4. 明确文本/多模态接口地址。
5. 明确文生图接口地址。
6. 明确图生图/图片编辑接口地址。
7. 明确文生视频/图生视频接口地址。
8. 明确视频结果查询接口地址。
9. 明确支持模型列表。
10. 按 xAI/Grok 官方接口参数补齐图片和视频示例。

支持模型说明：

- 推理/多模态模型：`grok-4.3`
- 图片模型：`grok-imagine-image`、`grok-imagine-image-quality`
- 视频模型：`grok-imagine-video`

文档示例域名已统一为：

`https://www.reayq.top`

## 3. 零售调用接口

### 3.1 OpenAI 兼容 Base URL

零售客户应配置：

`base_url = https://www.reayq.top/retail/v1`

鉴权：

`Authorization: Bearer YOUR_RETAIL_GROK_KEY`

零售 Key 前缀：

`rgk-`

### 3.2 模型列表接口

新增 OpenAI 兼容模型列表：

`GET /retail/v1/models`

返回格式：

```json
{
  "object": "list",
  "data": [
    { "id": "grok-4.3", "object": "model", "owned_by": "xai", "created": 0 },
    { "id": "grok-imagine-image", "object": "model", "owned_by": "xai", "created": 0 },
    { "id": "grok-imagine-image-quality", "object": "model", "owned_by": "xai", "created": 0 },
    { "id": "grok-imagine-video", "object": "model", "owned_by": "xai", "created": 0 }
  ]
}
```

该接口加入零售非消耗类鉴权放行，不扣额度。有效零售 Key 可用它刷新模型列表。

相关文件：

- `backend/internal/server/routes/retail_grok.go`
- `backend/internal/handler/retail_grok_gateway_handler.go`
- `backend/internal/server/middleware/retail_grok_key_auth.go`

### 3.3 文本 / 多模态识图

接口：

`POST /retail/v1/chat/completions`

支持模型：

- `grok-4.3`

用途：

- 文本推理。
- 图片理解。
- 多模态识图。

计费：

- 记录 input tokens。
- 记录 output tokens。
- 累加 `token_used_total`。

### 3.4 文生图

接口：

`POST /retail/v1/images/generations`

支持模型：

- `grok-imagine-image`
- `grok-imagine-image-quality`

主要参数：

- `prompt`
- `n`
- `aspect_ratio`
- `resolution`
- `response_format`

计费：

- 按 `ImageCount` 累加 `image_used_total`。

### 3.5 图生图 / 图片编辑

接口：

`POST /retail/v1/images/edits`

支持模型：

- `grok-imagine-image`
- `grok-imagine-image-quality`

支持：

- 单图 `image`。
- 多图 `images`。
- 图片 URL。
- base64 data URL。

计费：

- 按 `ImageCount` 累加 `image_used_total`。

### 3.6 文生视频 / 图生视频

接口：

`POST /retail/v1/videos/generations`

其他视频接口：

- `POST /retail/v1/videos/edits`
- `POST /retail/v1/videos/extensions`
- `GET /retail/v1/videos/:request_id`

支持模型：

- `grok-imagine-video`

主要参数：

- `prompt`
- `duration`
- `aspect_ratio`
- `resolution`
- `image`
- `reference_images`

计费：

- 成功提交视频任务后累加 `video_used_total`。
- 视频轮询接口不扣额度。

## 4. 生产隔离设计

### 4.1 数据隔离

新增独立表：

- `retail_grok_keys`
- `retail_grok_usage_logs`

迁移文件：

- `backend/migrations/147_create_retail_grok_keys.sql`
- `backend/migrations/148_create_retail_grok_usage_logs.sql`

明确不写入：

- 现有 `api_keys`
- 现有 `usage_logs`
- 现有 `usage_billing`

### 4.2 鉴权隔离

新增零售专用鉴权：

- `backend/internal/server/middleware/retail_grok_key_auth.go`

零售 Key 使用 `rgk-` 前缀。

普通生产 `/v1/*` 鉴权已新增前缀拦截：

- `rgk-` 调普通 `/v1/chat/completions` 必须失败。
- `rgk-` 只能调 `/retail/v1/*`。

相关文件：

- `backend/internal/server/middleware/api_key_auth.go`

验证结果：

- `POST /v1/chat/completions` + `rgk-...` 返回 `401`。
- `GET /retail/v1/usage` + `rgk-...` 返回 `200`，前提是 Key 未删除且有效。

### 4.3 路由隔离

零售调用路由独立注册在：

- `backend/internal/server/routes/retail_grok.go`

零售调用入口：

- `/retail/v1/*`

管理端入口：

- `/api/v1/admin/retail-grok/*`

客户查询兼容入口：

- `/api/v1/retail-grok/usage`

普通用户菜单没有挂零售入口，普通登录用户不需要看到零售页面。

### 4.4 计费隔离

零售计费只写：

- `retail_grok_usage_logs`
- `retail_grok_keys.token_used_total`
- `retail_grok_keys.image_used_total`
- `retail_grok_keys.video_used_total`

不进入现有主站统计，不影响现有每日统计、用量主链和账单主链。

## 5. 今日修复的问题

### 5.1 管理端生成接口 404

问题：

- 管理端点击生成零售 Key 时返回 404。

原因：

- 后端旧进程未重启，新路由未加载。

处理：

- 重启后端后确认 `/api/v1/admin/retail-grok/batch-generate` 生效。

### 5.2 客户查询页无显示

问题：

- 客户查询页输入 Key 后没有显示。

原因：

- 前端 dev server 请求 `/retail/v1/usage` 时被 Vite fallback 返回 `index.html`，HTTP 200 但不是 JSON。

处理：

- 新增 `/api/v1/retail-grok/usage` 兼容查询路由。
- 前端查询改走 `/api/v1/retail-grok/usage`。
- 前端增加响应校验，避免 HTML 假成功。

### 5.3 客户接口说明不完整

问题：

- 文档缺少多模态识图、官方图片参数、图生图格式、视频分辨率和时长参数。

处理：

- 补齐 Grok 4.3 多模态识图示例。
- 补齐文生图参数。
- 补齐图生图/多图编辑示例。
- 补齐文生视频和图生视频示例。
- 明确 OpenAI 兼容 `base_url`。

### 5.4 CSV 导出不是全部 Key

问题：

- 原 CSV 只导出当前生成批次。

处理：

- 点击导出时重新请求后端最多 10000 条零售 Key。
- 导出按钮改为“导出全部 CSV”。

### 5.5 删除按钮报错

问题：

- 管理页点击删除可能报错。

原因：

- 原实现物理删除 `retail_grok_keys`，对有用量日志或需要审计的 Key 不合适。

处理：

- 改成软删除：`status = 'deleted'`。
- `GetByID`、`GetByKey`、`List` 均排除 `deleted`。
- 被删除 Key 不再可用于零售接口。
- 历史用量日志保留。

### 5.6 视频调用不扣次数

问题：

- 视频接口成功调用后 `video_used_total` 不增加。

原因：

- 视频成功后传入 `nil` result，原计费构造器在 `result == nil` 时提前返回，导致 `video_count = 0`。

处理：

- 对视频端点先设置 `video_count = 1`，再处理 result。

相关文件：

- `backend/internal/service/retail_grok_gateway_service.go`

### 5.7 文本 token / 图片次数可能不落库

问题：

- 文本和图片调用成功，但用量页看不到增加。

原因：

- 零售计费写库使用 `c.Request.Context()`，而写库发生在响应转发完成之后。如果客户端连接关闭或流式响应结束，context 可能被取消，导致写库失败。
- 原先 `RecordUsage` 错误被静默吞掉。

处理：

- 零售计费写库改为独立 `context.Background()` + 10 秒超时。
- 保留 request id / client request id 作为日志上下文。
- `RecordUsage` 失败时写日志 `retail_grok.record_usage_failed`。

相关文件：

- `backend/internal/handler/retail_grok_gateway_handler.go`

### 5.8 `rgk-` 误调普通 `/v1/*` 必须失败

问题：

- 零售 Key 不能误进入普通生产接口。

处理：

- 普通 API Key 鉴权遇到 `rgk-` 前缀直接返回 401。
- 错误信息提示必须使用 `/retail/v1`。

相关文件：

- `backend/internal/server/middleware/api_key_auth.go`

### 5.9 零售模型列表刷不出来

问题：

- OpenAI SDK / 客户端配置 `base_url = https://www.reayq.top/retail/v1` 后通常会请求 `/models`。
- 之前没有 `GET /retail/v1/models`。

处理：

- 新增 `GET /retail/v1/models`。
- 返回 OpenAI 兼容模型列表。
- 加入零售非消耗类鉴权放行。

### 5.10 ccswitch 拉起协议供应商名称含中文

问题：

- `ccswitch://v1/import` 的 `name` 和 `notes` 使用中文供应商名时，ccswitch 可能解析失败。

处理：

- 供应商标识固定为英文 `Relayq`。

相关文件：

- `frontend/src/utils/toolConfigExport.ts`
- `frontend/src/utils/ccswitchImport.ts`

### 5.11 Codex Windows 下载链接改为官方入口

问题：

- 页面上的 Windows Codex EXE 下载链接走站内 `/downloads/...`。

处理：

- 改为 OpenAI 官方 Windows 安装入口：

`https://get.microsoft.com/installer/download/9PLM9XGG6VKS?cid=website_cta_psi`

相关文件：

- `frontend/src/views/user/StarterInstallView.vue`

### 5.12 用户侧 API 文档文案调整

修改用户侧 API 文档顶部文案：

`本页面详细介绍各个模型的接口模式，如果人看不懂，请你让的agent看懂就行了。图片，音频，视频模型是子agents。不能当推理模型使用。`

相关文件：

- `frontend/src/views/user/APIDocsView.vue`

## 6. 关键文件清单

### 后端新增/修改

- `backend/internal/server/routes/retail_grok.go`
- `backend/internal/server/routes/admin.go`
- `backend/internal/server/middleware/retail_grok_key_auth.go`
- `backend/internal/server/middleware/api_key_auth.go`
- `backend/internal/handler/retail_grok_gateway_handler.go`
- `backend/internal/handler/admin/retail_grok_handler.go`
- `backend/internal/service/retail_grok_types.go`
- `backend/internal/service/retail_grok_service.go`
- `backend/internal/service/retail_grok_gateway_service.go`
- `backend/internal/repository/retail_grok_key_repo.go`
- `backend/internal/repository/retail_grok_usage_log_repo.go`
- `backend/migrations/147_create_retail_grok_keys.sql`
- `backend/migrations/148_create_retail_grok_usage_logs.sql`

### 前端新增/修改

- `frontend/src/views/admin/RetailGrokView.vue`
- `frontend/src/views/public/RetailGrokKeyUsageView.vue`
- `frontend/src/views/public/RetailGrokDocsView.vue`
- `frontend/src/api/admin/retailGrok.ts`
- `frontend/src/api/retailGrok.ts`
- `frontend/src/router/index.ts`
- `frontend/vite.config.ts`
- `frontend/src/utils/toolConfigExport.ts`
- `frontend/src/utils/ccswitchImport.ts`
- `frontend/src/views/user/StarterInstallView.vue`
- `frontend/src/views/user/APIDocsView.vue`

### 文档

- `.trae/documents/retail-grok-update-record.md`
- `.trae/documents/2026-07-07-github-update-notes.md`

## 7. 验证记录

已执行过的验证包括：

1. 后端健康检查：

`GET http://localhost:3000/health` 返回 `200`。

2. 前端首页访问：

`GET http://localhost:3001/` 返回 `200`。

3. 客户查询页访问：

`http://localhost:3001/retail/grok/key-usage`

4. 客户接口说明页访问：

`http://localhost:3001/retail/grok/docs`

5. 管理端页面访问：

`http://localhost:3001/admin/retail-grok`

6. 零售查询接口：

`GET http://localhost:3000/retail/v1/usage`

7. 前端代理查询接口：

`GET http://localhost:3001/api/v1/retail-grok/usage`

8. 普通 `/v1` 禁止 `rgk-`：

`POST http://localhost:3000/v1/chat/completions` + `rgk-...` 返回 `401`。

9. 零售 `/retail/v1` 允许 `rgk-`：

`GET http://localhost:3000/retail/v1/usage` + 有效 `rgk-...` 返回 `200`。

10. 零售模型列表：

`GET http://localhost:3000/retail/v1/models` 已新增。测试时旧 Key 已软删除，因此返回 `retail grok key not found`，说明路由已进入零售鉴权链路；使用 active Key 应返回模型列表。

11. 前端类型检查：

多次执行：

`pnpm exec vue-tsc --noEmit`

通过。

12. 后端局部测试：

执行过：

`go test ./internal/handler ./internal/service`

`go test ./internal/server/middleware ./internal/handler ./internal/service`

`go test ./internal/repository ./internal/handler ./internal/service`

`go test ./internal/handler ./internal/server/middleware`

均通过。

## 8. 部署注意事项

1. 部署前必须执行新增 migration：

- `backend/migrations/147_create_retail_grok_keys.sql`
- `backend/migrations/148_create_retail_grok_usage_logs.sql`

2. 部署后需要重启后端，确保以下路由生效：

- `/retail/v1/models`
- `/retail/v1/chat/completions`
- `/retail/v1/images/generations`
- `/retail/v1/images/edits`
- `/retail/v1/videos/generations`
- `/retail/v1/usage`
- `/api/v1/retail-grok/usage`
- `/api/v1/admin/retail-grok/*`

3. 前端构建后确认以下页面可访问：

- `/admin/retail-grok`
- `/retail/grok/key-usage`
- `/retail/grok/docs`

4. 确认反向代理允许 `/retail/v1/*` 直达后端。

5. 确认生产域名使用：

`https://www.reayq.top/retail/v1`

6. 确认普通 `/v1/*` 使用 `rgk-` 返回 401，避免零售 Key 误入主生产计费链路。

7. 建议上线后用新生成的 active 零售 Key 测试：

- `GET /retail/v1/models`
- `POST /retail/v1/chat/completions`
- `POST /retail/v1/images/generations`
- `GET /retail/v1/usage`

8. 如果零售用量没有增加，优先查后端日志关键字：

`retail_grok.record_usage_failed`

## 9. GitHub 提交建议

建议提交标题：

`feat: add isolated retail Grok key flow`

建议提交说明：

```text
feat: add isolated retail Grok key flow

- add isolated retail Grok key tables, repositories, services, middleware, and routes
- add admin batch generation, usage lookup, soft delete, and full CSV export
- add public retail key usage and API docs pages
- add OpenAI-compatible /retail/v1 models/chat/images/videos endpoints
- keep retail usage isolated from existing api_keys, usage_logs, and usage_billing
- reject rgk- keys on regular /v1 endpoints
- fix retail usage accounting context and video counting
- update ccswitch provider naming and Codex official download links
```

## 10. 已知注意点

1. 当前 `grok-4.3` 多模态文档使用 `/retail/v1/chat/completions` 示例；xAI 官方最新也有 `/v1/responses` 形态，但本次零售网关尚未单独暴露 `/retail/v1/responses`。
2. 零售删除是软删除，不会物理清理历史记录。
3. CSV 导出默认最多拉取 10000 条零售 Key。
4. 模型实际是否可用仍取决于后台 xAI 分组白名单和账号实际开通情况。
5. 当前工作区还存在一些非本次零售功能相关改动或未跟踪文件，提交前建议用 `git status --short` 再核对一次，避免误提交无关文件。
