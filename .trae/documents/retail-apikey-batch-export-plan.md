# 零售 API Key 批量导出与客户自助页计划

## Summary

在现有项目基础上，新增一套面向管理员的“零售 API Key 批量导出”能力，并复用现有公网 `/key-usage` 页面作为客户自助页。

第一版范围控制如下：

- 管理端入口放在现有 `系统设置` 页面内，不新增独立左侧导航。
- 一次批量生成一批真实 `api_keys` 记录，并在前端当场展示、支持 CSV 下载。
- 每批 Key 统一绑定到 **1 个现有分组**，模型能力继承该分组。
- 每个 Key 新增三类零售硬额度：
  - `token` 总量上限（输入+输出实际总 token）
  - `image` 总次数上限（所有 `/v1/images/*`）
  - `video` 总次数上限（所有 `/v1/videos/generations|edits|extensions`）
- 额度拦截采用你确认的策略：**结算后封顶**。也就是本次请求先完成并记账，达到或超过上限后，后续请求全部拒绝。
- 客户入口继续使用现有公开页 `/key-usage`，但补成更适合零售客户的页面：展示该 Key 的剩余额度/用量、一个站点入口按钮，以及图片/视频接口的简版 JSON 说明。

## Current State Analysis

### 前端现状

- 管理员系统设置页为单路由多 Tab，入口文件是 `frontend/src/views/admin/SettingsView.vue`。
  - 现有 `settingsTabs` 已集中定义，适合继续加一个业务型 Tab，而不是单独开新管理路由。
- 公开页 `frontend/src/views/KeyUsageView.vue` 已存在，路由 `/key-usage` 已在 `frontend/src/router/index.ts` 注册为 `requiresAuth: false`。
  - 该页面当前已经支持：
    - 粘贴 API Key
    - 调用 `/v1/usage`
    - 展示总额度/余额、时间窗口、明细、`daily_usage`、`model_stats`
- 登录后说明页 `frontend/src/views/user/APIDocsView.vue` 已有图片/视频/音频说明，但它依赖登录态 `AppLayout`，不适合作为零售客户公开页直接使用。
- 站点公共配置已可从前端读取：
  - `site_name`
  - `site_logo`
  - `doc_url`
  - `custom_menu_items`
  - 相关代码在 `frontend/src/stores/app.ts`、`frontend/src/stores/adminSettings.ts`、`backend/internal/handler/dto/settings.go`

### 后端现状

- 当前 API Key 主模型与限额字段集中在：
  - `backend/ent/schema/api_key.go`
  - `backend/internal/service/api_key.go`
  - `backend/internal/service/api_key_service.go`
  - `backend/internal/repository/api_key_repo.go`
- `api_keys` 表目前只支持：
  - 金额配额 `quota / quota_used`
  - 过期时间 `expires_at`
  - 金额型窗口限流 `rate_limit_5h / 1d / 7d`
- 管理员侧目前只有：
  - 查看某用户的 Key：`GET /api/v1/admin/users/:id/api-keys`
  - 修改 Key 分组：`PUT /api/v1/admin/api-keys/:id`
  - 没有批量生成/导出零售 Key 的接口。
- API Key 鉴权和运行前拦截位于 `backend/internal/server/middleware/api_key_auth.go`。
  - 这里已经会做：
    - Key 是否有效
    - 用户状态
    - 分组状态
    - 余额 / 订阅 / 美元配额 / 美元窗口限流
  - `/v1/usage` 已被明确视为“只鉴权，不做计费拦截”的特例，适合继续作为客户自助页后端接口。
- `/v1/usage` 的公开响应位于 `backend/internal/handler/gateway_handler.go`。
  - 当前已经会返回：
    - `mode`
    - `quota`
    - `rate_limits`
    - `remaining`
    - `usage`
    - `daily_usage`
    - `model_stats`
- 计费落点已存在：
  - usage log 创建：`backend/internal/service/openai_gateway_service.go`
  - 结算命令与事务接口：`backend/internal/service/usage_billing.go`
  - 事务仓储：`backend/internal/repository/usage_billing_repo.go`
- 用量日志已具备区分媒体请求所需字段：
  - `total_tokens`
  - `image_count`
  - `inbound_endpoint`
  - `billing_mode`
  - 定义与查询在 `backend/internal/pkg/usagestats/usage_log_types.go`、`backend/internal/repository/usage_log_repo.go`

### 约束与关键现实

- 现有模型权限体系是 **分组驱动**，不是 `api_key` 直接挂模型白名单。
  - 因此第一版按你的决策，批量导出页选择 **1 个现有分组**，不在 Key 上重复造模型配置。
- 你要的 `token` 上限是“输入+输出实际总量”，该值只能在请求完成后得知。
  - 因此第一版采用你确认的策略：**结算后封顶**，允许最后一次请求把额度打穿一点点，但下一次开始严格拒绝。

## Assumptions & Decisions

1. 不新增“每个零售 Key 对应一个独立用户”的模式。
   - 第一版只按 API Key 隔离。
2. 不做“批次历史表”。
   - 一次批量生成后，前端立即展示本次生成结果并支持 CSV 下载。
   - 后续管理依然通过已有 API Key 数据本身完成。
3. 每批 Key 绑定 **1 个现有分组**。
   - Key 的可用模型完全继承该分组现有的 `models_list_config` / 白名单能力。
4. 媒体次数口径：
   - 图片：所有 `/v1/images/*`
   - 视频：所有 `/v1/videos/generations|edits|extensions`
   - 视频轮询 `GET /v1/videos/{request_id}` 不计入视频额度
5. 额度口径：
   - `token_limit_total` / `token_used_total` 按 usage log 的实际 `total_tokens`
   - `image_limit_total` / `image_used_total` 按 usage log 的实际 `image_count`
   - `video_limit_total` / `video_used_total` 按成功落账的媒体请求数累计
6. 拦截时机：
   - token / image / video 都采用“结算后封顶”。
7. 客户公开页沿用 `/key-usage`，不再额外开新公开路由。
8. “网站入口”按钮默认跳转到站点首页 `/home`，不复用 `doc_url`。

## Proposed Changes

### 1. 扩展 API Key 数据模型，承载零售额度

#### 文件

- `backend/migrations/147_add_retail_api_key_limits.sql`
- `backend/ent/schema/api_key.go`
- `backend/internal/service/api_key.go`
- `backend/internal/handler/dto/types.go`
- `backend/internal/handler/dto/mappers.go`
- `frontend/src/types/index.ts`

#### 变更内容

- 给 `api_keys` 新增零售字段：
  - `retail_enabled` `boolean`
  - `token_limit_total` `bigint`
  - `token_used_total` `bigint`
  - `image_limit_total` `integer`
  - `image_used_total` `integer`
  - `video_limit_total` `integer`
  - `video_used_total` `integer`
- 这些字段默认值全部为 `0`。
  - `0` 代表该维度不限额。
- 在 service / DTO / 前端类型里补齐对应字段，使：
  - 管理端可创建与读取
  - `/v1/usage` 可返回
  - 用户页 / 公开页可展示

#### 原因

- 这是当前最小可行改法。
- 不需要新建零售客户表，也不需要引入新的 Key 绑定关系。
- 能直接复用现有 `api_keys` 的鉴权、状态、分组、过期时间和公开查询链路。

### 2. 新增管理员批量生成功能

#### 文件

- `backend/internal/handler/admin/retail_api_key_handler.go`
- `backend/internal/server/routes/admin.go`
- `backend/internal/service/retail_api_key_service.go`
- `backend/internal/service/api_key_service.go`
- `backend/internal/repository/api_key_repo.go`
- `frontend/src/api/admin/retailApiKeys.ts`
- `frontend/src/views/admin/SettingsView.vue`
- `frontend/src/components/admin/retail/RetailApiKeyBatchPanel.vue`
- `frontend/src/i18n/locales/zh.ts`
- `frontend/src/i18n/locales/en.ts`

#### 变更内容

- 后端新增管理员接口，例如：
  - `POST /api/v1/admin/retail-api-keys/batch-generate`
- 请求体包含：
  - `group_id`
  - `count`
  - `name_prefix`
  - `expires_in_days`
  - `token_limit_total`
  - `image_limit_total`
  - `video_limit_total`
- 执行逻辑：
  - 使用当前管理员用户 ID 作为 `api_keys.user_id`
  - 批量调用现有 API Key 创建逻辑生成真实 Key
  - 每个 Key 统一写入：
    - 分组
    - 到期时间
    - 零售额度字段
    - `retail_enabled=true`
  - 返回一次性明文结果列表，供前端下载 CSV
- 前端在 `SettingsView.vue` 中新增一个 Tab（建议 key：`retailKeys`），并把复杂表单拆到独立组件 `RetailApiKeyBatchPanel.vue`。
- 页面展示：
  - 选择现有分组
  - 批量数量
  - 名称前缀
  - 有效期
  - token / image / video 上限
  - 生成结果表格
  - CSV 下载按钮

#### 原因

- 入口放在现有 `系统设置` 里，符合你的产品预期。
- 把生成面板拆出去，避免 `SettingsView.vue` 继续膨胀。
- 复用现有 API Key 创建链路，风险低于另造一套 Key 生成逻辑。

### 3. 在鉴权阶段加入零售额度前置拒绝

#### 文件

- `backend/internal/server/middleware/api_key_auth.go`
- `backend/internal/service/api_key.go`

#### 变更内容

- 在现有美元配额 / 过期 / 订阅检查之后，新增零售额度拒绝逻辑：
  - 当 `retail_enabled=true` 且：
    - `token_limit_total > 0 && token_used_total >= token_limit_total`
    - 或 `image_limit_total > 0 && image_used_total >= image_limit_total`
    - 或 `video_limit_total > 0 && video_used_total >= video_limit_total`
  - 则直接拒绝后续业务请求。
- `/v1/usage` 继续保留 skip-billing 例外。
  - 也就是 Key 即使额度耗尽，也仍然能打开客户自助页查看剩余额度为 0 的状态。

#### 原因

- 需要让额度耗尽后的 Key 立即停止继续消费。
- 同时不能影响客户查看自己已售 Key 的剩余额度和说明页。

### 4. 在结算事务中累计 token / image / video 实际用量

#### 文件

- `backend/internal/service/usage_billing.go`
- `backend/internal/repository/usage_billing_repo.go`
- `backend/internal/service/openai_gateway_service.go`

#### 变更内容

- 在现有 usage billing 事务里，把零售累计数一起更新，保证和实际落账一致。
- 累计规则：
  - `token_used_total += usageLog.TotalTokens()`
  - `image_used_total += usageLog.ImageCount`
  - `video_used_total += 1`，仅当本次 usage 对应的 `inbound_endpoint` 命中：
    - `/v1/videos/generations`
    - `/v1/videos/edits`
    - `/v1/videos/extensions`
- 该累计只在请求成功并进入实际结算路径时发生。
- 由于你已确认“结算后封顶”，这里不做额度预占。

#### 原因

- token / image / video 的真实值在请求结束后最可靠。
- 放在现有 billing 事务里，能够最大程度减少“已记 usage log 但没记零售额度”或相反的状态分裂。

### 5. 扩展 `/v1/usage` 响应，让客户页能展示零售额度

#### 文件

- `backend/internal/handler/gateway_handler.go`
- `frontend/src/views/KeyUsageView.vue`

#### 变更内容

- 后端在现有 `/v1/usage` 返回结构中增加零售额度块，例如：
  - `retail_limits.token.limit/used/remaining`
  - `retail_limits.image.limit/used/remaining`
  - `retail_limits.video.limit/used/remaining`
- 前端 `KeyUsageView.vue`：
  - 保留现有查询流程、环形卡片、明细卡片
  - 新增零售额度展示区，优先展示你这次新增的 token/image/video 剩余额度
  - 当某维度为不限额时，明确显示“无限制”而不是隐藏
  - 页面下方补一个简版客户说明区：
    - 站点入口按钮（跳 `/home`）
    - 图片接口简版 JSON 示例
    - 视频接口简版 JSON 示例

#### 原因

- 你要的客户页，本质上就是在现有公开 `KeyUsageView` 上做零售化补全。
- 这样最省路线，也能直接利用已存在的 `/v1/usage`、`daily_usage`、`model_stats`。

### 6. 补齐前端和后端测试

#### 文件

- `backend/internal/handler/admin/*_test.go`
- `backend/internal/service/*_test.go`
- `backend/internal/server/middleware/api_key_auth_test.go`
- `frontend/src/views/__tests__/KeyUsageView.spec.ts`
- `frontend/src/views/admin/__tests__/SettingsView.spec.ts`
- 如需要，新增 `frontend/src/components/admin/retail/__tests__/RetailApiKeyBatchPanel.spec.ts`

#### 变更内容

- 后端测试覆盖：
  - 批量生成管理员接口参数校验
  - 批量生成后字段写入正确
  - 零售额度达到上限后拦截
  - `/v1/usage` 在 Key 已耗尽时仍可查询
  - 结算后 token/image/video 累计正确
- 前端测试覆盖：
  - 系统设置新 Tab 可见
  - 生成成功后表格与 CSV 数据正确
  - `KeyUsageView` 正确显示零售额度
  - 客户说明区与站点入口显示正确

#### 原因

- 这次改动会触及鉴权、结算和公开页，回归面比普通表单页大，必须有聚焦测试。

## Verification Steps

1. 运行数据库迁移，确认 `api_keys` 新字段存在。
2. 后端单测至少覆盖：
   - 批量生成接口
   - 额度拦截
   - 结算累加
   - `/v1/usage` 返回零售额度
3. 前端单测至少覆盖：
   - `SettingsView` 新 Tab
   - 批量生成表单
   - `KeyUsageView` 零售额度与简版说明
4. 手工验收路径：
   - 管理员打开 `系统设置 -> 零售 API Key`
   - 选择现有分组、输入数量和额度，生成一批 Key
   - 下载 CSV，确认包含：
     - API Key
     - 分组
     - 到期时间
     - token/image/video 上限
     - 客户查询页地址 `/key-usage`
   - 拿其中一个 Key 打开 `/key-usage`
   - 确认页面能看到：
     - 剩余 token/image/video
     - 当前用量
     - 网站入口按钮
     - 图片/视频 JSON 简版说明
   - 用该 Key 连续请求，直到某项额度耗尽
   - 确认：
     - 最后一次请求结算成功
     - 后续请求被拒绝
     - `/key-usage` 仍可查看，且剩余额度为 0 或负值被钳制为 0

## Out of Scope

- 零售批次历史管理
- 每个零售 Key 对应独立用户
- 专属短链接 / 免输 Key 自动登录页
- 图文音视频更细粒度的不同售价套餐
- 按请求预估的严格预占额度机制
