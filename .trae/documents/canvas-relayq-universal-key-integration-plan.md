# 无线画布与 RelayQ 全模型中转集成计划

## Summary

下一阶段把无线画布从“绑定单个分组的一把托管 Key”升级为“每个用户一把 RelayQ 托管的 Canvas 万能 Key”。该 Key 对客户界面完全隐藏，但按用户选择允许保留在浏览器会话中，因此了解开发者工具的用户仍可提取；它不是绕过权限的全站超级权限，而是每次请求实时聚合该用户当前有权使用的分组、订阅、模型和端点，并由 RelayQ 服务端选择唯一真实分组完成调度与扣费。

Canvas 不建立独立钱包、价格或充值体系。所有生成请求继续进入 RelayQ 现有网关和计费链路，余额由 RelayQ 扣减；Canvas 顶栏只显示 RelayQ 当前余额，不显示价格、费用估算、API Key、充值或独立账单。Canvas 增加“退出无线画布”按钮，保持登录并返回 RelayQ `/dashboard`。

## Current State Analysis

### 已有能力

* RelayQ 已有登录保护入口 `/canvas`，位于 `frontend/src/router/index.ts`，登录回跳可直接复用。

* `frontend/src/views/user/CanvasView.vue` 调用 `POST /api/v1/canvas/bootstrap`，将结果写入 `sessionStorage.relayq_canvas_bootstrap` 后进入 `/canvas-app/`。

* `backend/internal/handler/canvas_handler.go` 已实现 bootstrap，并返回用户、余额、托管 Key、单个分组、模型和能力。

* `backend/internal/service/api_key_service.go` 已能创建 `client_app=infinite-canvas`、`managed=true` 的托管 Key，用户不能编辑或删除。

* `deploy/infinite-canvas-base.patch` 已让上游 Canvas 读取 bootstrap 配置，并在图片/视频请求上增加 RelayQ 来源头。

* RelayQ 网关已有成熟的 API Key 鉴权、用户余额/订阅校验、真实分组调度、渠道定价、同步扣费和 Leonardo 异步预留/结算能力。

* `GET /v1/usage` 已能按 API Key 返回余额和用量，可复用其余额响应结构，但万能 Key 不应被某个单一分组限制。

### 当前限制

* `api_keys.group_id` 是单值；一个普通 Key 只能绑定一个分组，而每个 `groups.platform` 也只有一个平台，所以创建一个普通数据库“超级组”无法安全覆盖 OpenAI、XAI、Gemini、Anthropic、Leonardo 等平台。

* 当前 Canvas Key 会绑定默认或用户指定的单一分组；切换分组会原地改绑同一 Key，旧标签页权限会漂移。

* bootstrap 模型目录只读取一个分组，并通过模型名字符串猜测协议；固定 `capabilities=true` 与真实可调用能力不一致。

* API Key 鉴权在进入 handler 前就依赖 Key 的单个 group 完成订阅/余额检查和路由上下文设置，不能直接支持跨分组 Key。

* `/v1/models` 是展示目录，不是完整授权结果；真实权限还取决于用户分组权利、有效订阅、账号映射、渠道限制、端点能力和运行配置。

* Canvas 顶栏没有 RelayQ 余额和返回 RelayQ 的操作；Canvas 配置入口仍可能暴露 Base URL/API Key 编辑能力。

* RelayQ 注销不会清理 `relayq_canvas_bootstrap`。

## Assumptions & Decisions

* **用户范围**：所有正常登录的 RelayQ 客户。

* **万能 Key**：每个用户恰好一把长期托管 Canvas Key；Key 本身不绑定真实分组，`group_id=NULL`。

* **“超级组”定义**：实现为服务端虚拟授权集合，不创建跨平台普通 group。集合等于用户在请求时有权绑定的全部活跃分组，订阅到期、撤权、分组停用立即生效。

* **模型与端点范围**：允许万能 Key 使用用户权限交集内的所有 RelayQ 网关模型与端点，而不只限图片/视频；普通非 Canvas Key 行为不变。

* **凭证可见性**：Canvas 和 RelayQ 普通 Key 列表不展示、复制或编辑该 Key；按用户选择，Key 仍注入浏览器会话，不能承诺对开发者工具不可见。Key 不写 URL、不写日志、不持久化到 Canvas 的长期配置存储。

* **路由策略**：先按端点支持的平台过滤，再按请求模型、账号映射、渠道限制、真实能力和用户权限过滤；最后按管理员现有 `group.sort_order` 选择。排序相同且存在多个有效候选时严格返回 `CANVAS_ROUTE_AMBIGUOUS`，不静默随机选择。

* **持续请求**：创建异步任务时记录实际 `group_id`；查询状态、下载内容、编辑或扩展时按资源路由记录恢复原分组，不能重新按当前默认顺序猜测。

* **严格失败**：无定价、余额不足、没有候选、候选不唯一、分组授权失效时，在调用上游前拒绝；不允许零价回退或负余额。

* **计费**：Canvas 不建立独立计费表和钱包；请求最终都携带解析后的真实 group 进入现有 RelayQ 计费链路。usage 和异步任务记录万能 Key ID、真实 group、来源 `infinite-canvas`。

* **余额 UI**：顶栏初次加载、窗口重新获得焦点、每次生成完成/失败后刷新，并以 30 秒轮询兜底；点击余额返回 RelayQ `/usage`。不显示本次费用、价格、充值入口或独立账单。

* **退出**：按钮文案“退出无线画布”，清理 Canvas 会话配置后跳转同源 `/dashboard`，保持 RelayQ 登录态。

* **兼容性**：现有普通 API Key、Playground 特殊链路和所有非 Canvas 请求保持原行为；已有单分组 Canvas Key迁移为 `group_id=NULL` 并继续复用，不重新发 Key。

## Proposed Changes

### 1. 将 Canvas Key 改为无固定分组的虚拟超级组凭证

**文件：**

* `backend/internal/service/api_key_service.go`

* `backend/internal/service/api_key.go`

* `backend/internal/repository/api_key_repo.go`

* `backend/ent/schema/api_key.go`

* 新增 `backend/migrations/158_canvas_universal_routing.sql`

**改动：**

* 将 `GetOrCreateCanvasAPIKey(ctx, userID, groupID)` 改为按 `(user_id, client_app, managed_purpose)` 精确获取或创建，不再接收或写入 `group_id`。

* Repository 新增精确查询，移除当前“只扫描前 100 把 Key”的逻辑。

* 普通用户 Key 列表默认过滤 managed Canvas Key；管理端仍可按 `client_app` 审计，但响应只返回掩码，不返回完整 Key。

* migration 将现有 `client_app='infinite-canvas' AND managed_purpose='canvas_bootstrap'` Key 的 `group_id` 置空，保留 Key 值和累计使用量；唯一索引继续保证每用户一把。

* 保留 managed Key 的不可编辑、不可删除约束。

### 2. 增加 Canvas 跨分组授权与确定性路由服务

**文件：**

* 新增 `backend/internal/service/canvas_routing_service.go`

* 新增 `backend/internal/service/canvas_routing_service_test.go`

* `backend/internal/service/wire.go`

* `backend/cmd/server/wire_gen.go`

* 复用 `backend/internal/service/api_key_service.go` 的 `GetAvailableGroups`

* 复用 `backend/internal/service/channel_service.go`、`gateway_service.go` 的账号、模型与定价判断

**接口：**

```go
type CanvasRouteRequest struct {
    UserID       int64
    APIKeyID     int64
    Method       string
    Endpoint     string
    Model        string
    ResourceID   string
    Modality     string
}

type CanvasRoute struct {
    Group        *Group
    Platform     string
    Model        string
    Protocol     string
}
```

**路由规则：**

1. 每次请求调用 `GetAvailableGroups(userID)`，实时计算用户权限交集。
2. 根据端点建立平台候选：例如 embeddings 只允许 OpenAI；Media 只允许 Leonardo；视频生成允许 OpenAI/XAI/Leonardo；编辑/扩展遵循现有 gateway 路由限制；Gemini `/v1beta` 只进入 Gemini/Antigravity 兼容分组。
3. 从 body、URL 或异步资源绑定中取得模型；读取 body 后必须恢复，不能影响后续 handler。
4. 对每个候选执行真实可调度账号、账号 `model_mapping`、渠道模型限制、运行配置和端点能力检查。
5. 检查渠道或全局定价可解析；无定价候选直接排除并最终返回 `CANVAS_PRICING_UNAVAILABLE`。
6. 按 `group.sort_order` 升序选择；最优排序并列时返回 409 `CANVAS_ROUTE_AMBIGUOUS`，响应带模型和候选平台名称但不泄露账号信息。
7. 返回真实 Group，由后续鉴权、调度和计费继续使用现有代码。

不修改普通 Group 的单平台语义，也不把不同平台账号硬塞进一个数据库 group。

### 3. 在 API Key 鉴权阶段解析万能 Key 的真实分组

**文件：**

* `backend/internal/server/middleware/api_key_auth.go`

* `backend/internal/server/middleware/api_key_auth_cache_impl.go`

* `backend/internal/server/middleware/middleware.go`

* `backend/internal/server/routes/gateway.go`

* 必要时新增 `backend/internal/server/middleware/canvas_route.go`

**数据流：**

```text
Bearer Canvas Key
→ 识别 managed + client_app=infinite-canvas
→ CanvasRoutingService 实时解析真实 group
→ clone APIKey 并只在请求上下文写入 resolved GroupID/Group
→ 执行该真实 group 的订阅、余额、配额和 endpoint 校验
→ 进入原 Gateway/OpenAI/Leonardo handler
→ 原计费链路按真实 group 扣 RelayQ 余额或订阅额度
```

**边界处理：**

* `/v1/models` 不选单组，而是聚合当前用户所有授权候选并去重，返回结构化的模型、平台、协议和端点。

* `/v1/usage` 对 Canvas Key直接返回 RelayQ 用户钱包余额及聚合用量，不依赖单个订阅分组。

* 普通 Key 继续使用现有固定 Group 路径。

* 禁止客户端用 header/query 指定任意 group；真实 group 只能由服务端路由结果决定。

* Canvas Key 若被复制后直接调用，也仍只能获得该用户当前权限交集，不会变成全站管理员权限。

### 4. 持久化异步资源的真实路由

**文件：**

* `backend/migrations/158_canvas_universal_routing.sql`

* 新增 `backend/ent/schema/canvas_resource_route.go`

* 新增 `backend/internal/service/canvas_resource_route.go`

* 新增 `backend/internal/repository/canvas_resource_route_repo.go`

* `backend/internal/repository/wire.go`

* `backend/internal/service/wire.go`

* `backend/internal/handler/openai_video*.go`、`backend/internal/handler/leonardo_media_handler.go` 中实际创建/返回异步 ID 的入口

**表结构：**

```text
canvas_resource_routes
- id
- api_key_id
- user_id
- resource_id
- group_id
- platform
- model
- endpoint_family
- expires_at
- created_at
UNIQUE(api_key_id, resource_id)
```

**行为：**

* 创建视频、Media 或其他异步资源成功获得公开 ID 后，写入实际 group 路由。

* status/content/edit/extension 请求先按 `(api_key_id, resource_id)` 取回原 group，再验证用户现在仍有该 group 权限；权限已失效则 403，不改路由到其他平台。

* 与现有 `generation_jobs.group_id` 保持一致；Leonardo 已有 generation job 时优先复用，不重复猜测。

* 路由记录保留到资源到期后再清理；清理任务沿用项目现有后台任务模式。

### 5. 重做 Bootstrap 为聚合能力契约

**文件：**

* `backend/internal/handler/canvas_handler.go`

* 新增或扩充 `backend/internal/handler/canvas_handler_test.go`

* `frontend/src/api/canvas.ts`

* `frontend/src/views/user/CanvasView.vue`

**新响应：**

```json
{
  "base_url": "/v1",
  "api_key": "仅用于注入，不在 UI 展示",
  "client_app": "infinite-canvas",
  "user": { "id": 1, "username": "u", "balance": 12.34 },
  "models": [
    {
      "id": "model-id",
      "modality": "image",
      "platform": "openai",
      "protocol": "openai",
      "endpoints": ["/v1/images/generations"]
    }
  ],
  "dashboard_url": "/dashboard",
  "usage_url": "/usage"
}
```

* 删除 `group_id` 请求和单组选择逻辑。

* 模型目录调用 CanvasRoutingService 聚合真实授权结果，不再按模型名字符串猜协议，也不再固定所有 capabilities 为 true。

* 返回 `Cache-Control: no-store`，日志和错误对象不得输出 `api_key`。

* `CanvasView` 仍只把 Key 写入当前标签页 `sessionStorage`；跳转不携带 query 参数。

### 6. Canvas 顶栏只显示余额和退出按钮

**文件：**

* `deploy/infinite-canvas-base.patch`

* 上游目标组件：`web/src/components/layout/app-top-nav.tsx`

* 上游目标组件：`web/src/components/layout/user-status-actions.tsx`

* 上游目标组件：`web/src/components/canvas/canvas-top-bar.tsx`

* 上游目标初始化：`web/src/components/layout/client-root-init.tsx`

* 上游图片/视频服务：`web/src/services/api/image.ts`、`video.ts`

* 上游 i18n 资源目录 `web/src/i18n/locales/*`

**顶部普通页面：**

* RelayQ 托管模式下隐藏“配置”入口及任何 API Key/Base URL 编辑入口。

* 右侧增加余额按钮，展示 RelayQ 返回的实时余额；点击跳转同源 `/usage`。

* 增加“退出无线画布”按钮，清理 `relayq_canvas_bootstrap` 和 Canvas 内存配置后跳转 `/dashboard`。

**画布项目全屏页面：**

* 当前 `AppTopNav` 会在 `/canvas/:id` 隐藏，因此同样在 `canvas-top-bar.tsx` 放置紧凑余额与退出按钮，保证任何画布状态都可返回 RelayQ。

**余额刷新：**

* 新增最小 RelayQ integration helper，从 session bootstrap 读取托管模式、调用 `GET /v1/usage`。

* 初始化刷新一次；页面可见且聚焦时刷新；图片/视频生成 Promise 完成或失败后触发刷新；30 秒定时刷新兜底。

* 刷新失败只显示“余额暂不可用”，不影响画布编辑；401/403 清理 Canvas 会话并跳转 `/login?redirect=/canvas`。

**计费 UI：**

* RelayQ 托管模式不展示价格、费用预估、充值、账单或 API Key；非 RelayQ 独立部署模式保持上游原行为。

### 7. 收紧退出和凭证生命周期

**文件：**

* `frontend/src/stores/auth.ts`

* `frontend/src/views/user/CanvasView.vue`

* `deploy/infinite-canvas-base.patch`

**改动：**

* RelayQ `clearAuth()` 统一删除 `sessionStorage.relayq_canvas_bootstrap`，覆盖主动注销、401 和 token 刷新失败。

* 进入 `/canvas` 时始终重新 bootstrap，从服务端获取同一托管 Key及最新授权目录。

* Canvas 配置 store 在 RelayQ 托管模式下不把 Key写入长期 localStorage/IndexedDB；只保留当前标签页会话。

* 退出无线画布只离开 Canvas，不调用 RelayQ logout，不清除 RelayQ 登录 JWT。

### 8. 严格计费和审计门禁

**文件：**

* `backend/internal/service/model_pricing_resolver.go`

* `backend/internal/service/openai_gateway_service.go`

* `backend/internal/service/gateway_service.go`

* `backend/internal/repository/usage_billing_repo.go`

* `backend/migrations/158_canvas_universal_routing.sql`

**改动：**

* 仅对 `client_app=infinite-canvas` 启用严格定价 fail-close，避免改变已有普通 API 兼容行为；未找到价格时调用上游前返回 `CANVAS_PRICING_UNAVAILABLE`。

* Canvas 钱包扣费增加原子余额下限，确保并发请求不能把余额扣成负数；订阅分组继续使用其现有限额。

* usage、billing dedup/ledger（若现有结构可扩展）和 generation job 记录 `request_source=infinite-canvas` 与实际 `group_id`。

* 保留现有 request ID 幂等机制；路由重试不得换 group 后重复扣费。

## API And Error Contract

### Bootstrap

* `POST /api/v1/canvas/bootstrap`

* 鉴权：RelayQ 用户 JWT

* 请求体：空对象

* 成功：返回聚合模型、余额和托管配置

### Balance

* `GET /v1/usage`

* 鉴权：Canvas 托管 Key

* Canvas Key 返回钱包余额与聚合用量；不返回价格或充值能力

### 路由错误

* `CANVAS_ROUTE_NOT_FOUND` / 404：用户当前没有可调用该模型和端点的分组

* `CANVAS_ROUTE_AMBIGUOUS` / 409：最优候选并列，需管理员调整 group sort order

* `CANVAS_PRICING_UNAVAILABLE` / 422：模型缺少可结算价格

* `CANVAS_GROUP_ACCESS_REVOKED` / 403：异步资源原分组权限已失效

* `INSUFFICIENT_BALANCE` / 403：余额不足且未调用上游

* `API_KEY_DISABLED` / 401：Canvas 托管 Key已停用

## Verification

### 后端单元测试

* 每用户重复 bootstrap 始终复用同一 Canvas Key，且 Key `group_id=NULL`。

* 用户拥有 OpenAI、XAI、Leonardo、Gemini 多个分组时，聚合模型目录正确去重并保留平台/协议元数据。

* 端点 + 模型只命中一个候选时选择正确真实 group。

* 同排序候选冲突时返回 409，不调用上游。

* 未定价模型严格拒绝，不产生 usage、不扣费、不调用上游。

* 订阅到期、exclusive group 撤权、group 停用后下一次请求立即拒绝或重路由到仍有权的唯一候选。

* 普通 API Key仍按原固定 group 行为运行。

### 后端集成测试

* 使用一把 Canvas Key分别调用 OpenAI 图片、XAI 图片/视频、Leonardo Media、Gemini 和普通文本端点，验证都通过真实 group 计费。

* 异步创建后改变 group sort order，status/content 仍回到创建时 group。

* 创建后撤销原 group 权限，status/content 返回 403，不能跨组接管资源。

* 并发请求余额不足时不会产生负余额；实际扣款次数与成功上游请求数一致。

* usage log、generation job、billing 幂等记录包含 Canvas 来源、万能 Key ID 和实际 group ID。

* 100+ 普通 Key 用户仍能精确找到 Canvas Key，不会重复创建。

### 前端与上游补丁测试

* 未登录访问 `/canvas` → 登录 → 回跳 → bootstrap → Canvas。

* RelayQ 模式中看不到 API Key、Base URL、价格、充值和独立账单入口。

* 普通顶栏和全屏画布顶栏都显示余额和“退出无线画布”。

* 生成结束后余额刷新；窗口聚焦和 30 秒轮询可恢复余额；余额请求失败不影响画布编辑。

* 点击余额进入 `/usage`；点击退出清理 Canvas 会话并进入 `/dashboard`，RelayQ 登录态仍有效。

* RelayQ 主站注销后，旧 Canvas 标签页下一次请求被拒绝并回登录页。

* 上游锁定 commit 的 patch 可应用，Bun/Vite Windows 构建通过。

### 回归与运行验证

* `go test` 覆盖新增 service、middleware、handler、repository 和迁移相关测试。

* `go vet` 通过。

* RelayQ 前端 `pnpm typecheck`、定向 Vitest 和生产构建通过。

* Infinite Canvas `bun run build` 通过。

* Windows 原生启动后，用浏览器人工走完：登录 → 进入 Canvas → 跨平台模型生成 → 余额减少 → 退出到 dashboard。

* Docker/Compose 配置解析仍通过，但 Docker 不是 Windows 开发验收的前置条件。

## Out Of Scope

* 不实现 Canvas 自有钱包、充值、价格表、账单或订阅。

* 不改变普通 API Key 为跨分组 Key。

* 不创建真正跨平台的普通数据库 group，不破坏现有 `group.platform` 单值约束。

* 不承诺长期 Key 对浏览器开发者工具不可见；若未来要达到该安全目标，再升级为 HttpOnly 会话或服务端代理/短期令牌。

* 本阶段不重构全站所有未定价模型策略，仅对 Canvas 严格拒绝。

