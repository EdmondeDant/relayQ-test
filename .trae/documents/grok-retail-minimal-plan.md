# Grok 零售隔离方案计划

## Summary

这版按你最新确认的“零影响生产优先”设计，第一批只做 Grok 系列零售能力，并把零售链路与现有生产主链彻底分开：

- 不复用现有 `/v1` 网关。
- 不复用现有 `api_keys` 表。
- 不写入现有 `usage_logs`、`usage_billing`、dashboard 统计链。
- 不改现有公开 `/key-usage` 页面。
- 不把入口塞进现有 `SettingsView.vue`。

第一批落地形态改为：

- 独立管理员路由：`/admin/retail-grok`
- 独立买家公开页：
  - `/retail/grok/key-usage`
  - `/retail/grok/docs`
- 独立零售 API 前缀：
  - `/retail/v1/chat/completions`
  - `/retail/v1/images/generations`
  - `/retail/v1/videos/generations`
  - `/retail/v1/videos/edits`
  - `/retail/v1/videos/extensions`
  - `/retail/v1/videos/:request_id`
  - `/retail/v1/usage`
- 独立零售数据表：
  - `retail_grok_keys`
  - `retail_grok_usage_logs`

这意味着第一批代码会比“最少代码方案”多，但可以满足你现在的核心要求：不影响现有生产端。

## Current State Analysis

### 前端现状

- 当前普通公开页通过 `frontend/src/router/index.ts` 的 `requiresAuth: false` 路由暴露。
- 公开页是否出现在菜单，不是由路由决定，而是由：
  - `frontend/src/stores/app.ts`
  - `frontend/src/components/layout/AppSidebar.vue`
  - `frontend/src/components/layout/AppHeader.vue`
  - `custom_menu_items`
  控制。
- 这意味着新增一个 `requiresAuth: false` 的零售页，只要不加进菜单，就不会被普通用户主动看到。
- 现有管理员系统设置页位于：
  - `frontend/src/views/admin/SettingsView.vue`
  - 该文件体积很大、Tab 很多，继续往里塞零售功能会直接扩大对现有管理页的影响。
- 现有公开 Key 查询页位于：
  - `frontend/src/views/KeyUsageView.vue`
  - 路由 `/key-usage`
  - 如果直接复用它，会改变当前公开页语义，不符合“零影响生产优先”。
- 现有登录后接口文档位于：
  - `frontend/src/views/user/APIDocsView.vue`
  - 已包含 Grok 图片/视频示例，可作为零售文档页素材来源。

### 后端现状

- 当前路由装配集中在：
  - `backend/internal/server/router.go`
- 当前公共与业务路由注册方式为：
  - `routes.RegisterCommonRoutes(...)`
  - `routes.RegisterAuthRoutes(...)`
  - `routes.RegisterUserRoutes(...)`
  - `routes.RegisterAdminRoutes(...)`
  - `routes.RegisterGatewayRoutes(...)`
  - `routes.RegisterPaymentRoutes(...)`
- 现有 OpenAI Compatible 主网关注册在：
  - `backend/internal/server/routes/gateway.go`
  - 入口为 `/v1/*`
- 当前 API Key 鉴权中间件位于：
  - `backend/internal/server/middleware/api_key_auth.go`
  - 它只认现有 `api_keys`
  - 并且 `/v1/usage` 只是现有主链上的特例

### 现有统计耦合

- `backend/ent/schema/usage_log.go` 中的 `usage_logs.api_key_id` 是必填，并强依赖现有 `api_keys`。
- `backend/internal/repository/usage_log_repo.go`、`backend/internal/service/usage_billing.go`、`backend/internal/service/openai_gateway_service.go`、`backend/internal/handler/gateway_handler.go` 都直接建立在现有 `usage_logs + api_keys` 上。
- 结论：
  - 只要继续走现有 `/v1` 和现有 `api_keys`，就无法承诺“零影响生产端”。
  - 要实现真正隔离，必须独立表、独立路由、独立查询页。

### 注入与扩展方式

- handler 注入集中在：
  - `backend/internal/handler/wire.go`
  - `backend/internal/handler/handler.go`
- 新增一组 `retail` handler/service 后，可以通过现有 Wire 方式正常接入。
- 因此“独立零售模块”在这个仓库里是自然扩展方式，不需要硬改主网关逻辑。

## Assumptions & Decisions

1. 第一批只支持 Grok 系列零售。
   - 文本推理
   - 图片
   - 视频

2. 第一批以“隔离优先”高于“代码最少”。
   - 允许新增表、服务、handler、路由、页面。
   - 不允许把零售流量接到现有主网关统计链。

3. 管理员入口采用独立管理路由。
   - 不放进 `SettingsView.vue`
   - 使用独立页面 `/admin/retail-grok`

4. 买家页保留，但独立成未挂导航公开页。
   - `/retail/grok/key-usage`
   - `/retail/grok/docs`

5. 零售调用入口独立前缀。
   - 使用 `/retail/v1/*`
   - 不复用现有 `/v1/*`

6. 零售 Key 采用独立表。
   - 不扩展现有 `api_keys`

7. 零售用量采用独立日志表。
   - 不写入现有 `usage_logs`

8. 第一批继续采用“结算后封顶”。
   - token / image / video 实际用量在请求完成后落账
   - 达到上限后拒绝后续请求

9. 第一批模型权限仍以“后台固定 Grok 范围”实现，不引入复杂可配置模型白名单系统。

## Proposed Changes

### 1. 新增独立零售数据模型

#### 文件

- `backend/migrations/147_create_retail_grok_keys.sql`
- `backend/migrations/148_create_retail_grok_usage_logs.sql`
- `backend/ent/schema/retail_grok_key.go`
- `backend/ent/schema/retail_grok_usage_log.go`
- `backend/internal/service/retail_grok_key.go`
- `backend/internal/service/retail_grok_usage_log.go`
- `backend/internal/repository/retail_grok_key_repo.go`
- `backend/internal/repository/retail_grok_usage_log_repo.go`

#### 变更内容

- 新建 `retail_grok_keys` 表，字段至少包括：
  - `id`
  - `key`
  - `name`
  - `status`
  - `expires_at`
  - `token_limit_total`
  - `token_used_total`
  - `image_limit_total`
  - `image_used_total`
  - `video_limit_total`
  - `video_used_total`
  - `created_by_admin_id`
  - `created_at`
  - `updated_at`
- 新建 `retail_grok_usage_logs` 表，字段至少包括：
  - `id`
  - `retail_grok_key_id`
  - `request_id`
  - `inbound_endpoint`
  - `model`
  - `input_tokens`
  - `output_tokens`
  - `total_tokens`
  - `image_count`
  - `video_count`
  - `status`
  - `error_message`
  - `created_at`
- 两张表都不与现有 `api_keys` / `usage_logs` 建立依赖。

#### 原因

- 这是实现“零影响生产端”的基础。
- 只要还依赖现有 `api_keys` 或 `usage_logs`，生产统计与行为就会被波及。

### 2. 新增独立零售鉴权与上下文

#### 文件

- `backend/internal/server/middleware/retail_grok_key_auth.go`
- `backend/internal/server/middleware/retail_grok_key_auth_test.go`
- 如需要上下文 key：`backend/internal/server/middleware/auth_subject.go` 或新增独立 context key 文件

#### 变更内容

- 新增只服务于 `/retail/v1/*` 的零售 Key 鉴权中间件。
- 只认零售 Key 表，不读取现有 `api_keys`。
- 校验项：
  - Key 是否存在
  - 是否启用
  - 是否过期
  - 是否达到 token/image/video 上限
- `/retail/v1/usage` 允许额度耗尽的 Key 继续查询自身状态。
- 中间件只把零售 Key 放进独立上下文，不污染现有 `/v1` 使用的上下文语义。

#### 原因

- 避免修改现有 `api_key_auth.go`
- 避免零售逻辑误伤现有生产网关

### 3. 新增独立零售服务，转发到现有上游能力，但不走现有主计费链

#### 文件

- `backend/internal/service/retail_grok_gateway_service.go`
- `backend/internal/handler/retail_grok_gateway_handler.go`
- `backend/internal/server/routes/retail_grok.go`
- `backend/internal/handler/wire.go`
- `backend/internal/handler/handler.go`
- `backend/internal/server/router.go`

#### 变更内容

- 新增独立零售路由注册函数，例如：
  - `RegisterRetailGrokRoutes(...)`
- 在 `backend/internal/server/router.go` 的 `registerRoutes(...)` 中额外挂载：
  - `/retail/v1/chat/completions`
  - `/retail/v1/images/generations`
  - `/retail/v1/videos/generations`
  - `/retail/v1/videos/edits`
  - `/retail/v1/videos/extensions`
  - `/retail/v1/videos/:request_id`
  - `/retail/v1/usage`
- 这些路由只使用零售中间件与零售 handler。
- 零售 handler 不调用现有 `applyUsageBilling` 和 `writeUsageLogBestEffort`。
- 零售服务只负责：
  - 校验零售请求
  - 转发到现有 xAI / OpenAI-compatible 上游能力
  - 在零售 usage log 里记录结果
  - 回写零售额度累计值

#### 关键实现策略

- 尽量复用已有上游请求构造与响应解析逻辑，而不是重新发明整套 Grok 协议处理。
- 但复用方式应控制在服务内部 helper 级别，不把请求送进现有 `/v1` 主网关 handler，以免触发主链记账与统计。
- 如果现有上游交互逻辑无法安全抽用，则第二优先是复制最小必要的 xAI/Grok 请求处理代码到独立零售服务。

#### 原因

- “零影响生产端”比复用优先级更高。
- 独立 handler/service 比在主网关里塞分支更安全。

### 4. 新增管理员独立页面与批量发 Key 能力

#### 文件

- `frontend/src/router/index.ts`
- `frontend/src/views/admin/RetailGrokView.vue`
- `frontend/src/components/admin/retail/RetailGrokBatchPanel.vue`
- `frontend/src/api/admin/retailGrok.ts`
- `backend/internal/handler/admin/retail_grok_handler.go`
- `backend/internal/server/routes/admin.go`
- `backend/internal/service/retail_grok_admin_service.go`
- `frontend/src/i18n/locales/zh.ts`
- `frontend/src/i18n/locales/en.ts`

#### 变更内容

- 新增管理员路由：
  - `/admin/retail-grok`
- 不修改现有 `SettingsView.vue`
- 管理员页面提供：
  - 批量生成数量
  - 名称前缀
  - 有效期
  - token 上限
  - image 上限
  - video 上限
  - 生成结果表格
  - CSV 下载
- 后端新增管理员接口，例如：
  - `POST /api/v1/admin/retail-grok/batch-generate`
  - `GET /api/v1/admin/retail-grok/keys`
  - `GET /api/v1/admin/retail-grok/keys/:id/usage`

#### 原因

- 避免改现有大体量的 `SettingsView.vue`
- 管理功能仍然只对管理员可见
- 与现有生产用户路径隔离更干净

### 5. 新增买家公开页与简版文档页

#### 文件

- `frontend/src/router/index.ts`
- `frontend/src/views/public/RetailGrokKeyUsageView.vue`
- `frontend/src/views/public/RetailGrokDocsView.vue`
- `frontend/src/api/retailGrok.ts`
- `frontend/src/i18n/locales/zh.ts`
- `frontend/src/i18n/locales/en.ts`

#### 变更内容

- 新增未挂导航公开路由：
  - `/retail/grok/key-usage`
  - `/retail/grok/docs`
- `RetailGrokKeyUsageView.vue` 只查询零售接口：
  - `GET /retail/v1/usage`
- 页面展示：
  - Key 状态
  - token/image/video 已用与剩余
  - 到期时间
  - 文档入口
- `RetailGrokDocsView.vue` 只保留第一批 Grok 范围说明：
  - 文本推理请求示例
  - 图片请求示例
  - 视频提交和轮询示例
- 这些页面不加入任何普通菜单和 `custom_menu_items`。

#### 原因

- 满足你最早要的“客户拿到 Key 后可单独查看用量和说明页”
- 同时不影响现有 `/key-usage` 的公开使用者

### 6. 定义第一批零售能力边界

#### 文件

- `backend/internal/service/retail_grok_gateway_service.go`
- `frontend/src/views/public/RetailGrokDocsView.vue`

#### 变更内容

- 第一批只支持这些入口：
  - `POST /retail/v1/chat/completions`
  - `POST /retail/v1/images/generations`
  - `POST /retail/v1/videos/generations`
  - `POST /retail/v1/videos/edits`
  - `POST /retail/v1/videos/extensions`
  - `GET /retail/v1/videos/:request_id`
  - `GET /retail/v1/usage`
- 第一批只允许这些模型范围：
  - Grok 文本模型
  - `grok-imagine-image`
  - `grok-imagine-image-quality`
  - `grok-imagine-video`
- 不支持：
  - 现有普通用户 `/v1/*`
  - 其他平台模型
  - 现有 dashboard / usage 页面中的零售视图混入

#### 原因

- 需求范围明确，隔离实现更容易收敛

### 7. 独立零售用量累计规则

#### 文件

- `backend/internal/service/retail_grok_gateway_service.go`
- `backend/internal/repository/retail_grok_key_repo.go`
- `backend/internal/repository/retail_grok_usage_log_repo.go`

#### 变更内容

- 文本推理：
  - `token_used_total += input_tokens + output_tokens`
- 图片：
  - `image_used_total += image_count`
- 视频：
  - 对以下提交接口每次成功计 `+1`
    - `/retail/v1/videos/generations`
    - `/retail/v1/videos/edits`
    - `/retail/v1/videos/extensions`
  - `GET /retail/v1/videos/:request_id` 不计额度
- 全部采用“结算后封顶”。
- 出错请求只记零售 usage log 错误状态，不增加成功额度累计。

#### 原因

- 与你前面的口径一致
- 实现简单且可预测

### 8. 测试与验收

#### 文件

- `backend/internal/server/middleware/retail_grok_key_auth_test.go`
- `backend/internal/handler/admin/retail_grok_handler_test.go`
- `backend/internal/handler/retail_grok_gateway_handler_test.go`
- `backend/internal/service/retail_grok_gateway_service_test.go`
- `frontend/src/views/public/__tests__/RetailGrokKeyUsageView.spec.ts`
- `frontend/src/views/public/__tests__/RetailGrokDocsView.spec.ts`
- `frontend/src/views/admin/__tests__/RetailGrokView.spec.ts`

#### 测试点

- 零售 Key 创建与批量生成正确
- 零售 Key 过期/停用/超额时被正确拒绝
- `/retail/v1/usage` 在额度耗尽后仍可查询
- 零售调用不会写入现有 `usage_logs`
- 零售调用不会影响现有 `api_keys`
- 零售公开页不出现在普通导航中
- 零售文档页仅展示 Grok 第一批能力

## Verification Steps

1. 执行数据库迁移，确认新增：
   - `retail_grok_keys`
   - `retail_grok_usage_logs`
2. 运行后端测试，确认零售模块通过。
3. 运行前端测试，确认零售页面与路由通过。
4. 管理员访问 `/admin/retail-grok`：
   - 生成一批 Key
   - 下载 CSV
5. 买家访问 `/retail/grok/key-usage`：
   - 使用零售 Key 查询额度
6. 使用零售 Key 调用 `/retail/v1/chat/completions`、`/retail/v1/images/generations`、`/retail/v1/videos/generations`：
   - 确认额度正确累计
7. 检查现有生产链：
   - `usage_logs` 无零售记录
   - 现有 `/v1/*` 行为无变化
   - 现有 `/key-usage` 页面无变化
   - 现有 `SettingsView.vue` 无变化

## Risk Notes

- 这版的主要成本在于新增一条独立零售链，而不是复用现有主网关。
- 如果后续你又把优先级切回“最少代码优先”，可以再退回：
  - 复用 `api_keys`
  - 复用 `/v1`
  - 复用 `usage_logs`
  但那样就不能再承诺“零影响生产端”。
