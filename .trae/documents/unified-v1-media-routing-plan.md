# RelayQ 统一 V1 媒体接口与跨 Provider 最低成本调度计划

## Summary

RelayQ 客户侧统一只公开 OpenAI-compatible `/v1` 媒体接口；Leonardo Production API v2 仅作为后端供应商适配协议，不再作为客户文档中的请求模式。旧 Leonardo Group 继续兼容现有 `parameters` 原生请求，避免破坏已有调用；新统一媒体商品入口只接受 OpenAI-compatible DTO。

本次同时覆盖图片生成/编辑与视频生成/编辑/扩展/查询/下载。新增独立的统一媒体商品目录，将客户看到的模型商品、固定规格售价、来源 Offer、能力和可信成本分离。商品显式授权给 OpenAI 入口 Group；来源 Offer 继续引用各自的 OpenAI-compatible、xAI 或 Leonardo 来源 Group，不改变 `group.platform`，不把 Leonardo 账号混入 OpenAI Group。

运行时先按操作和请求参数过滤完整支持的 Offer，再按未过期的人工可信成本升序选择。客户始终按商品规格固定售价扣费，不随实际 Provider 改变。无可信可用 Offer 时失败关闭；只有明确证明请求未写出上游时才允许切换到下一 Offer，任何写出后未知状态都禁止重发。

## Current State Analysis

### 公共接口与文档

* 后端只注册公共 `/v1` 媒体路由，并没有 RelayQ 公共 `/v2`：[gateway.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/internal/server/routes/gateway.go#L38-L118)。

* 当前图片、视频入口先读取 API Key Group 的 `platform`，再选择 OpenAI 或 Leonardo Handler：[gateway.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/internal/server/routes/gateway.go#L156-L309)。

* 文档中的“Leonardo Production API v2”实际是同一个 `/v1` 路径下的 `parameters` 原生请求格式，不是 RelayQ `/v2`：[APIDocsView.vue](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/frontend/src/views/user/APIDocsView.vue#L109-L129)。

* Leonardo 后端创建才调用上游 `POST /v2/generations`；查询仍调用上游 v1：[client.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/internal/pkg/leonardo/client.go#L188-L224)。

* Canvas bootstrap 已下发 `/v1`，因此客户契约统一为 v1 与现状一致：[canvas\_handler.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/internal/handler/canvas_handler.go#L68-L86)。

### 分组、调度与价格

* `group.platform` 当前是账号查询、路由、模型列表和渠道价格的硬边界；OpenAI Group 不会调度 Leonardo 账号。

* 通用调度按平台、模型映射、健康、额度、并发、优先级和 LRU 选择账号，不比较上游价格：[gateway\_service.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/internal/service/gateway_service.go#L2446-L2501)。

* Leonardo 创建链路要求来源 Group 中恰好一个有效账号，多账号会返回歧义错误：[leonardo\_media\_create\_service.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/internal/service/leonardo_media_create_service.go#L191-L211)。

* ChannelModelPricing 是客户计费表，不是账号或供应商成本 Offer；按 platform 查询，不能直接用于跨 Provider 最低上游成本调度：[channel.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/internal/service/channel.go#L77-L111)。

* Leonardo 已有动态估价、Reserve、异步任务、Settle/Release 和 `submission_unknown` 安全状态；这些能力不能被普通 OpenAI 转发链路绕过：[leonardo\_image\_create\_orchestrator.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/internal/service/leonardo_image_create_orchestrator.go#L148-L206)。

### 任务、账务与兼容风险

* `generation_jobs` 已包含 Provider、模态、客户模型、上游模型、账号、请求哈希、结果、成本快照和账务状态，适合扩展为统一媒体执行记录：[generation\_job.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/ent/schema/generation_job.go#L44-L163)。

* 现有资金预留表命名及实现绑定 Leonardo，但统一商品需要 Provider-neutral 的一次预留/一次结算边界：[152\_leonardo\_image\_funds\_reservations.sql](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/migrations/152_leonardo_image_funds_reservations.sql#L1-L20)。

* `usage_logs` 已保存 requested/upstream model、Group、Channel 和客户成本，但缺商品、Offer、来源 Group、Provider 和可信成本快照：[usage\_log.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/ent/schema/usage_log.go#L31-L108)。

* 视频状态和 content 当前依赖 Group Platform 或 xAI 的短期 sticky account；统一入口必须改为通过持久化 job 找回首次选中的 Offer、Provider 和账号，查询时绝不能重新选路：[openai\_videos.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/internal/service/openai_videos.go#L700-L831)。

* 当前无版本 `/images/*`、`/videos/*` 别名继续存在；本次保留为隐藏兼容入口，但文档只展示 `/v1`。

## Proposed Changes

### 1. 冻结公共契约和统一 DTO

修改 [APIDocsView.vue](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/frontend/src/views/user/APIDocsView.vue)：

* 只展示以下公共接口：

  * `POST /v1/images/generations`

  * `POST /v1/images/edits`

  * `POST /v1/videos` 与兼容别名 `POST /v1/videos/generations`

  * `POST /v1/videos/edits`

  * `POST /v1/videos/extensions`

  * `GET /v1/videos/{task_id}`

  * `GET /v1/videos/{task_id}/content`

* 删除客户文档中的“Leonardo 原生模式”“Production API v2 parameters”目录、请求示例及原生 `/media/generations` 查询说明。

* 明确说明 Provider 由 RelayQ 自动选择、模型名是统一商品名、售价固定、参数不会静默降级。

* 保留 OpenAI-compatible 图片和视频 JSON/multipart 示例，并确保 ComfyUI、画布类客户端只需配置 RelayQ `/v1` Base URL。

修改 [APIDocsView.spec.ts](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/frontend/src/views/user/__tests__/APIDocsView.spec.ts)：

* 断言页面只出现 RelayQ `/v1` 公共契约，不出现 Leonardo 原生 `parameters` 示例或把上游 v2 描述成客户接口。

* 校验图片生成/编辑、视频生成/编辑/扩展/查询/content 的路径和可解析示例。

后端保留现有 `/v1/media/generations*` 和 `parameters` 自动识别，但仅允许旧 `platform=leonardo` Group 使用；统一商品入口检测到 `parameters` 时返回 OpenAI 格式 `unsupported_media_params`，不把原生字段带入跨 Provider 调度。

### 2. 新增统一媒体商品目录

新增下一号迁移 `backend/migrations/159_unified_media_catalog.sql`，不修改历史 migration。建立以下表：

#### `media_products`

* `id BIGSERIAL PRIMARY KEY`

* `public_model VARCHAR(200) NOT NULL`

* `modality VARCHAR(16) NOT NULL CHECK (image/video)`

* `enabled BOOLEAN NOT NULL DEFAULT FALSE`

* `description TEXT`

* `created_at/updated_at TIMESTAMPTZ`

* 唯一约束 `(public_model, modality)`

商品代表客户模型身份，同名图片和视频仍由 modality 区分。

#### `media_product_group_bindings`

* `product_id`、`group_id` 联合主键并建立外键。

* 只允许绑定 `platform=openai` 的入口 Group。

* 绑定后该 Group 的 API Key 才能通过统一商品入口调用；订阅、额度、RPM、并发和客户账务仍归入口 Group。

#### `media_product_prices`

* `id`、`product_id`、`operation`、`spec_key`。

* `unit_price_usd DECIMAL(20,10) > 0`、`currency='USD'`、`version`、`enabled`。

* `spec_key` 使用服务端规范化后的稳定键：

  * 图片：操作、尺寸、质量、数量计价单位。

  * 视频：操作、时长、分辨率/尺寸、音频等会影响售价的字段。

* 唯一约束 `(product_id, operation, spec_key, version)`。

* 请求规格没有启用的精确售价时，在发起上游请求前返回 `media_product_price_unavailable`。

#### `media_offers`

* `id`、`product_id`、`provider`、`source_group_id`、`upstream_model`、`enabled`、`priority`。

* `operations JSONB`：明确支持 generations/edits/extensions/status/content。

* `capabilities JSONB`：保存该 Offer 可接受的尺寸、质量、数量、时长、分辨率、参考图数量、首尾帧、音频及字段互斥规则。

* 人工可信成本快照字段：`cost_rules JSONB`、`cost_source`、`cost_version`、`verified_at`、`expires_at`。

* 唯一约束 `(product_id, provider, source_group_id, upstream_model)`。

* 来源 Group 必须存在、启用且 platform 与 Provider 相符；账号仍只绑定来源 Group，不跨平台迁移。

* `expires_at <= now()`、成本规则不覆盖本次规格、成本非正数或来源为空时，该 Offer 不参与调度。

新增 Ent schema：

* `backend/ent/schema/media_product.go`

* `backend/ent/schema/media_product_price.go`

* `backend/ent/schema/media_offer.go`

* 对 Group 增加商品绑定 edge，不改变 `group.platform`。

执行项目现有 Ent 生成流程更新生成代码，禁止手工修改 Ent 生成产物。

新增领域与仓储文件：

* `backend/internal/service/media_product.go`：商品、价格、Offer、能力和成本规则类型。

* `backend/internal/repository/media_product_repo.go`：目录 CRUD、按入口 Group+模型+模态加载商品及 Offer。

* `backend/internal/service/media_catalog_service.go`：保存时校验来源 Group、Provider、操作、能力、价格和成本快照；删除被任务引用的商品/Offer 时改为禁用而非物理删除。

### 3. 新增 Provider-neutral 媒体调度器

新增 `backend/internal/service/media_offer_selector.go`，输入规范化后的 `CanonicalMediaRequest`，只负责选择，不发送请求、不扣费：

1. 按入口 Group、public model、modality 加载已启用商品。
2. 校验商品已显式绑定入口 Group。
3. 生成稳定 `spec_key` 并解析唯一启用客户售价。
4. 展开已启用 Offer。
5. 按 operation 和全部显式请求参数做能力过滤；任一字段不支持即跳过，不删除、不改写、不降级参数。
6. 过滤成本过期、成本维度不完整、成本非正数、来源 Group/账号池不可用的 Offer。
7. 对本次数量和规格计算 `trusted_cost`，按 `trusted_cost ASC, priority ASC, offer_id ASC` 稳定排序。
8. 无候选返回 `no_trusted_media_offer`，且不得预留或扣费。

“同名即等价”仅表示多个 Offer 可挂在同一个 `public_model` 商品下；实际每次请求仍必须通过完整能力过滤。

新增可审计的 skip reason：`unsupported_operation`、`unsupported_param`、`untrusted_cost`、`expired_cost`、`no_capacity`、`source_group_unavailable`。

### 4. 建立统一媒体 Orchestrator 与 Provider Adapter

新增：

* `backend/internal/service/media_orchestrator.go`

* `backend/internal/service/media_provider_adapter.go`

* `backend/internal/service/media_openai_adapter.go`

* `backend/internal/service/media_leonardo_adapter.go`

接口保持最小且统一：

* `Submit(ctx, job, canonicalRequest, offer) -> SubmissionOutcome`

* `Poll(ctx, job) -> CanonicalMediaResult`

* `Content(ctx, job, index) -> media stream`

`SubmissionOutcome` 必须明确区分：

* `not_written`：确认请求未写出，可尝试下一 Offer。

* `submitted`：已获得可信上游 ID，固定绑定当前 Offer。

* `side_effect_unknown`：可能已写出但响应不可信，任务进入 `unknown/manual_review`，禁止自动重发。

统一执行顺序：

1. 解析和校验公开 OpenAI DTO、内容审核、入口 Group 权限和并发限制。
2. 选择 Offer 并冻结客户售价、可信成本、价格/成本版本。
3. 以稳定 request fingerprint 和 `Idempotency-Key` 创建或复用统一 generation job。
4. 预留一次客户固定售价。
5. 调用选中 Adapter；只有 `not_written` 才按排序尝试下一个 Offer，且所有尝试共用同一客户 job 和预留。
6. 提交后 job 固定 Provider、Offer、来源 Group、账号和上游 ID；状态查询、Webhook、轮询和 content 永远按 job 执行，不重新选价或选路。
7. 成功且有有效输出时一次结算；明确失败且无输出时一次释放；未知副作用保持预留并进入人工复核。
8. 同一 job 只生成一条客户 UsageLog；多次上游尝试作为独立 attempt/audit 记录，不重复扣客户款。

OpenAI Adapter 复用现有图片和视频转发、能力判断、代理、错误分类及账号 scheduler；增加按 `source_group_id` 调度的入口，但仍执行平台、模型映射、健康、额度、限流和并发过滤。

Leonardo Adapter 复用现有 Registry、参数转换、参考图上传、v2 创建、v1 查询、结果解析、SSRF/content 校验和 unknown 安全语义。移除统一商品路径中的“多个 Leonardo 账号即歧义”限制，改为在来源 Group 内按关系优先级、账号优先级、健康、负载和 LRU 选择；旧 Leonardo Group 原生路径保持现有行为，避免隐式迁移。

### 5. 扩展统一任务、尝试记录和资金预留

新增迁移 `backend/migrations/160_unified_media_execution.sql`：

#### 扩展 `generation_jobs`

* `product_id BIGINT NULL`

* `offer_id BIGINT NULL`

* `source_group_id BIGINT NULL`

* `operation VARCHAR(32) NULL`

* `customer_price_version VARCHAR(64) NULL`

* 保留现有 `group_id` 表示入口 Group，不能改写为来源 Group。

* 现有 `provider/account_id/upstream_model` 保存最终提交成功或副作用未知的实际执行来源。

修改 [generation\_job.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/ent/schema/generation_job.go)、`backend/internal/service/generation_job.go` 和 `backend/internal/repository/generation_job_repo.go` 传播上述字段。

#### 新增 `media_job_attempts`

* 保存 `job_id`、`offer_id`、`provider`、`source_group_id`、`account_id`、`upstream_model`、`trusted_cost_snapshot`、`submission_state`、错误码和时间。

* 每次 Offer 尝试一条，支持证明为何发生安全切换；`side_effect_unknown` 后禁止创建后续 attempt。

#### 新增 Provider-neutral `media_funds_reservations`

* 字段包括 reference、user\_id、public\_id、product\_id、固定客户金额、price version、status、released/settled 时间。

* 唯一约束 `(user_id, public_id)` 和 reference，Reserve/Settle/Release 使用数据库事务及 CAS，保证并发幂等。

* 新统一商品使用此表；旧 Leonardo Group 继续使用原 `leonardo_image_funds_reservations`，不在本次做危险的历史账务表迁移。

新增 `backend/internal/repository/media_funds_repo.go` 和 `backend/internal/service/media_funds.go`，统一图片与视频、OpenAI-compatible 与 Leonardo 的客户资金状态机。

### 6. 统一 UsageLog 与成本审计

在同一迁移中为 `usage_logs` 增加：

* `media_product_id`

* `media_offer_id`

* `upstream_platform`

* `source_group_id`

* `trusted_cost_amount/unit/source/version`

* `customer_price_version`

修改 [usage\_log.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/ent/schema/usage_log.go)、`backend/internal/service/usage_log.go` 和 `backend/internal/repository/usage_log_repo.go`。

账务不变量：

* `customer_cost` 只来自商品固定规格售价快照。

* `trusted_cost` 只用于 Offer 排序和毛利预估，绝不作为客户扣费金额。

* 实际上游账单写入 `actual_upstream_cost`，可与可信成本比较，但不追溯改变已结算客户价。

* API Key 配额、订阅和余额归入口 Group；来源 Group 仅提供供应执行。

* `usage_billing_dedup` 或统一 reservation reference 以公开 job/request ID 为作用域，跨 Provider failover 仍只能结算一次。

### 7. 接入公共 V1 路由

修改 [gateway.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/internal/server/routes/gateway.go)：

* 对 OpenAI 入口 Group，先判断请求模型是否绑定已启用统一媒体商品。

* 命中商品时交给统一 Media Handler/Orchestrator；未命中时保持旧 OpenAI 路径，保证显式迁移。

* 旧 Leonardo Group 继续进入 Leonardo Handler，并兼容原生 `parameters`。

* `/v1/images/generations` 和 `/v1/images/edits` 接入统一商品。

* `/v1/videos`、`/v1/videos/generations`、`/v1/videos/edits`、`/v1/videos/extensions` 接入统一商品。

* `/v1/videos/{task_id}` 与 `/content` 首先按统一 job ownership 查询；若不是统一 job，再走旧平台兼容路径。

* 无 `/v1` 别名调用同一 Handler，继续兼容但不写入文档。

* `/v1/media/generations*` 继续只对旧 Leonardo Group 开放，不作为统一商品契约。

新增 `backend/internal/handler/media_handler.go` 负责 OpenAI 错误格式和 Canonical DTO 解析。将当前 Leonardo Handler 中可复用的 OpenAI 图片/视频转换和等待逻辑下沉到 service，禁止 Handler 相互调用或伪造 Gin Group Platform。

### 8. `/v1/models` 只展示统一媒体商品

修改当前模型列表合成逻辑（入口在 [gateway.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/internal/server/routes/gateway.go#L70-L71) 对应的 Models service）：

* 对启用统一媒体商品的 OpenAI 入口 Group，媒体模型部分只返回该 Group 已绑定、已启用、至少有一个能力和成本快照有效且账号池可用的 `public_model`。

* 同名商品只出现一次，不暴露 Provider、Offer、来源 Group、账号或上游模型。

* 非媒体聊天/嵌入模型维持当前列表；“仅统一商品”约束只替换该入口 Group 的媒体模型集合，不误删文本模型。

* 旧 Group 未绑定统一商品时模型列表行为完全不变。

### 9. 管理端独立媒体商品目录

新增后端管理 API：

* `GET/POST /api/v1/admin/media-products`

* `GET/PUT/DELETE /api/v1/admin/media-products/:id`

* 商品价格、Group 绑定和 Offer 在商品详情中整体读取，使用事务保存。

* 删除改为禁用；已有 generation job/usage 引用时禁止物理删除。

新增：

* `backend/internal/handler/admin/media_product_handler.go`

* 在 `backend/internal/server/routes/admin.go` 注册路由。

* 在 service wiring 中注入 catalog、selector、orchestrator 和 adapters。

新增前端：

* `frontend/src/views/admin/MediaProductsView.vue`

* `frontend/src/api/admin/mediaProducts.ts`

* `frontend/src/components/admin/media-product/ProductPriceEditor.vue`

* `frontend/src/components/admin/media-product/OfferEditor.vue`

* 在管理端路由和侧栏加入“媒体商品”。

* 更新 `frontend/src/i18n/locales/zh.ts` 与 `en.ts`。

UI 必须支持：

* public model、modality、启用状态。

* 绑定一个或多个 OpenAI 入口 Group。

* 按 operation + 规范规格维护客户固定售价。

* 为 Offer 选择 Provider、来源 Group、upstream model、操作、能力矩阵、优先级。

* 人工录入可信成本规则、来源、版本、验证时间和到期时间。

* 保存前显示阻断错误：来源 Group 平台不符、无固定售价、成本不完整/过期、无操作、能力与价格/成本规格无法对应。

* 管理端不得引导把 Leonardo 账号直接绑定到入口 OpenAI Group。

### 10. Canvas 与客户端兼容

修改 [canvas\_routing\_service.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-leonardo-prod/backend/internal/service/canvas_routing_service.go) 和 Canvas bootstrap/catalog：

* 目录 endpoint 统一发布真实 `/v1/images/*`、`/v1/videos/*`，移除当前不存在的 `/v1/media/videos/generations`。

* Canvas 只消费公开 OpenAI-compatible DTO，不发送 Leonardo `parameters`。

* Base URL builder 明确识别 `/v1`，删除对公共 `/v2` 的假设，避免 `/v2/v1`。

* 视频创建显式携带稳定 `Idempotency-Key`；创建、fallback alias、轮询不能生成新键。

* 创建响应兼容 `id/request_id/job_id/task_id` 及 `data.*`，但统一商品优先返回稳定 RelayQ `task_id/id`。

对应修改 `.tmp-infinite-canvas/web/src/services/api/video.ts`、`.tmp-infinite-canvas/web/src/stores/use-config-store.ts` 及其测试；若无限画布源码实际由其他目录构建，执行前先以当前启动脚本确认权威源码位置，仅修改权威目录。

### 11. 错误、观测和运营保护

统一 OpenAI 格式错误码：

* `media_product_not_available`

* `media_product_price_unavailable`

* `unsupported_media_params`

* `no_trusted_media_offer`

* `media_offer_exhausted`

* `media_submission_unknown`

增加结构化指标/日志：

* Offer skip reason 计数。

* 选中 Provider/Offer 分布。

* fail closed 与 safe failover 次数。

* 客户收入、可信成本、实际成本、毛利及成本偏差。

* `submission_unknown/manual_review` 告警。

日志禁止记录 API Key、上游密钥、完整 Data URI 或私有媒体内容。

## Implementation Order

用户选择本期图片和视频一起交付且商品启用后直接切流，因此不设置长期 shadow 阶段；仍按以下依赖顺序开发，并在最后一次性启用目标商品：

1. **契约与数据层**：完成公共 v1 文档调整、目录表、执行审计表、Ent/schema/repository 和管理 API。
2. **纯选择逻辑**：完成规格规范化、固定售价解析、能力过滤、人工可信成本校验与确定性最低成本排序，先用纯单元测试锁定。
3. **统一账务与任务层**：完成 Provider-neutral reservation、job、attempt、usage 快照和幂等不变量。
4. **图片链路**：接入 generations 和 edits，验证 OpenAI-compatible 与 Leonardo Offer 的参数转换、同步/异步返回、参考图及安全 failover。
5. **视频链路**：接入 generations、edits、extensions、status、content；不支持编辑/扩展的 Leonardo Offer 由 operation filter 自动跳过。
6. **管理端与模型列表**：完成商品、价格、Offer、入口 Group 授权配置和 `/v1/models` 去 Provider 化。
7. **Canvas/ComfyUI 兼容**：统一 Base URL、路径、任务 ID、幂等和响应形状。
8. **联合回归后显式切流**：先创建禁用商品并完成真实探针；验收通过后逐商品绑定入口 Group 并启用。旧 Group 和未绑定模型保持原行为。

## Assumptions & Decisions

* 客户侧不存在需要迁移的 RelayQ `/v2` 公共接口；本次是移除文档中的原生 v2 语义，不是删除公共 v2 路由。

* 客户只看到 OpenAI-compatible `/v1` 和统一商品名；上游 Leonardo v2 完全封装在 Adapter 内。

* Leonardo 原生 `parameters` 仅旧 Leonardo Group 兼容，统一商品入口不接受。

* 本期仅统一媒体模型，不改变聊天、Responses、Embeddings、音频等平台调度。

* 图片范围包含 generations 和 edits；视频范围包含 generations、edits、extensions、status 和 content。

* 同名模型可归入同一商品，但 Offer 必须完整支持本次操作和参数；禁止静默降级。

* 客户售价按商品操作和规格固定；Provider 变化不改变客户费用。

* 上游成本由管理员按 Offer+规格录入，必须带来源、版本、验证时间和到期时间；不从客户 Channel 售价反推。

* 无可信成本候选失败关闭；未知价不按零价处理，也不回退旧调度。

* 只有确定请求未写出时才可换 Offer；写出后响应未知进入人工复核，禁止自动重发。

* 商品显式绑定 OpenAI 入口 Group；旧 Group、旧 API Key 和未绑定模型默认不切流。

* 用户选择不做产品级 shadow 阶段；上线保护依靠默认禁用、真实探针和逐商品显式启用。

* “直接放进 OpenAI 接口分组”解释为客户协议和商品展示统一为 OpenAI-compatible，而不是把不同 Provider 账号混入同一 platform Group。

## Verification

### 数据与管理端

* migration 在全新库和现有 Windows 数据库均可幂等执行；历史 Group、Channel、Leonardo job 和账务记录不变。

* 商品、价格、Offer、Group 绑定 CRUD round-trip 正确；禁用/删除不会破坏历史 job 和 UsageLog。

* 非 OpenAI 入口 Group 绑定、来源 Group/Provider 不一致、过期成本、空能力、缺少售价均在保存时拒绝。

* 管理端创建、编辑、回填、禁用、错误提示和中英文显示通过前端测试及浏览器人工路径。

### 选择与能力

* 两个 Offer 都完整支持时选择可信成本最低者；同价按 priority 和 ID 稳定选择。

* 最低价 Offer 缺少当前 operation、尺寸、质量、参考图、时长、分辨率或音频能力时跳过并选择下一 Offer。

* 成本过期、规则不覆盖、来源不可信、账号池不可用时不形成候选。

* 所有候选无效时返回 `no_trusted_media_offer`，没有上游 POST、预留或扣费。

### 图片端到端

* 用同一统一商品分别强制选择 OpenAI-compatible 和 Leonardo Offer，验证 `/v1/images/generations` 均返回标准 OpenAI `data[]`。

* `/v1/images/edits` 验证单图、多图、模型允许的最大参考图数量及不支持 mask 的明确错误。

* ComfyUI 风格请求不携带 async/response\_format/Idempotency-Key 时仍得到兼容响应；服务端生成稳定幂等键。

* Leonardo 内部真实请求使用 Production v2 Schema，外部请求和文档不出现原生 `parameters`。

* 内容安全、输出数量、HTTPS URL/base64、进度、终态账务和 usage 均通过。

### 视频端到端

* generations、edits、extensions 分别验证 operation filter；不支持某操作的 Offer 不被选择。

* 创建返回稳定 RelayQ task ID；status/content 通过 job 找回首次 Provider、Offer、来源 Group 和账号，不重新调度。

* Leonardo 视频验证真实 v2 创建、v1 轮询、`motionMP4URL` canonical 转换、MP4 MIME/magic、SSRF 和 no-redirect。

* 首帧/参考图确实上传并进入上游请求；不只验证前端预览。

* Canvas 使用 `/v1`、稳定幂等键并轮询到终态；不存在 `/v2/v1` 或 `/v1/media/videos/generations` 请求。

### 幂等与账务

* 相同 Idempotency-Key 与相同 fingerprint 只创建一个客户 job、一次预留、一次结算和一条 UsageLog。

* 相同 Idempotency-Key 不同 fingerprint 返回冲突，不复用旧任务。

* 首个 Offer `not_written` 时可切换到第二 Offer，客户仍只扣一次固定售价。

* 首个 Offer `side_effect_unknown` 时不创建第二次付费 POST，job 为 unknown/manual\_review，预留保持待处理。

* 明确失败且无输出释放预留；成功结算；部分成功按固定商品规则结算且保留实际输出数。

* UsageLog/GenerationJob 同时可审计入口 Group、商品、Offer、来源 Group、Provider、账号、requested/upstream model、客户售价、可信成本和实际成本。

### 兼容与回归

* 未绑定统一商品的 OpenAI Group 继续走原图片/视频路径。

* 旧 Leonardo Group 的 OpenAI-compatible 和原生 `parameters` 请求继续可用，现有任务查询/content 不受影响。

* `/v1/models` 对统一入口只展示一次商品名且不暴露 Provider；文本模型列表保持原状。

* 无版本 `/images/*`、`/videos/*` 别名继续工作，但文档只展示 `/v1`。

* 后端执行相关 service/handler/route/repository 单元与集成测试，前端执行类型检查和组件测试。

* Windows 使用当前源码重新构建后端并设置正确 `DATA_DIR`；使用 `curl.exe --noproxy "*"` 做 HTTP 回归。

* 上线前为目标图片和视频商品各做至少一次最低规格真实探针，并核对请求参数、上游任务、媒体终态、客户余额、reservation、GenerationJob、attempt、UsageLog、可信成本与实际上游账单。

## Release Gates

* 文档仍把 Leonardo v2/parameters 当客户主接口。

* 任何路径通过修改 `group.platform` 或混挂 Leonardo 账号实现统一调度。

* 能力不完整仍参与最低成本比较，或自动删除客户参数。

* 成本缺失/过期仍放行，或客户售价随 Provider 改变。

* 写出后未知状态自动切换 Provider 重发。

* 视频查询/content 根据当前最低价重新选路，而非绑定创建 job。

* 同一请求出现多次客户扣款、多个 UsageLog 结算或 reservation 无法闭环。

* `/v1/models` 暴露来源 Provider 或同一商品重复出现。

* 未完成目标商品自身的真实图片、视频和 Windows 画布验收。

