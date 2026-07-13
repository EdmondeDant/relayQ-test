## Summary

把在线创作工作台从“浏览器带登录 JWT + `X-RelayQ-Source=playground` + 前后端固定 allowlist/模型名”的免 API Key 体验模式，改为“用户必须选择自己生成的 API Key，系统基于该 Key 当前绑定的分组与可见渠道动态给出模型列表与价格，并按渠道定价正常扣费”的统一模式。此次改造覆盖整个工作台，不只图片工具，包括 AI 生图、图片编辑、批量主图、批量克隆、水印去除、对话助手、商品文案、图片翻译、AI 视频。

目标是彻底消除当前因为管理员改分组、改模型映射、升级模型名导致前端按钮指向失效模型的问题，同时去掉专门的“playground 免 API Key 特权链路”，让工作台与用户真实 API 使用环境保持一致。

## Current State Analysis

### 1. 当前前端模型来源是硬编码，和真实分组配置脱节

- [PlaygroundView.vue](file:///c:/Users/Administrator/.openclaw/workspace/RelayQ-test/frontend/src/views/user/PlaygroundView.vue#L220-L273) 里直接写死了：
  - `imageModels = [{ id: 'gpt-image-2' ... }, { id: 'gpt-image-2-pro' ... }]`
  - `chatModels = [{ id: 'deepseek-v4-flash' ... }, { id: 'gpt-5.4' ... }]`
- 下拉 `button` 当前展示内容直接取 `selectedImageModel` 与上述硬编码数组。
- 批量主图、批量克隆、水印去除、图片编辑、文案、翻译、视频都共用这些硬编码或硬编码常量，因此只要管理员改了分组可见模型、channel restrict models、上游模型命名，工作台就会出现 `MODEL_NOT_ALLOWED` 或上游 `model not found`。

### 2. 当前后端还维护了一条 playground 专用 allowlist

- [playground_guard.go](file:///c:/Users/Administrator/.openclaw/workspace/RelayQ-test/backend/internal/handler/playground_guard.go#L15-L21) 里存在固定的 `playgroundAllowedModels`：
  - `gpt-image-2`
  - `gpt-image-2-pro`
  - `deepseek-v4-flash`
  - `gpt-5.4`
  - `grok-imagine-video`
- 这意味着即使前端修成动态模型，只要还走 `X-RelayQ-Source=playground`，后端一样会挡掉新模型。
- 这个 guard 本质是为“浏览器不用 API Key 也能调网关”做的 MVP 保护，不适合继续存在于“用户自带 API Key”模式。

### 3. 仓库已经有用户侧所需的数据接口，不需要凭空再造模型管理系统

- API Key 列表：[/keys](file:///c:/Users/Administrator/.openclaw/workspace/RelayQ-test/frontend/src/api/keys.ts#L17-L45)
- 用户可用分组：[/groups/available](file:///c:/Users/Administrator/.openclaw/workspace/RelayQ-test/frontend/src/api/groups.ts#L17-L26)
- 用户可用渠道聚合：[/channels/available](file:///c:/Users/Administrator/.openclaw/workspace/RelayQ-test/frontend/src/api/channels.ts#L74-L82)
- `channels/available` 已经返回：
  - 用户可访问的 groups
  - 每个平台支持的模型 `supported_models`
  - 定价 `pricing`
  - 图片分辨率价格 `image_pricing`
  - 分组是否允许生图 `allow_image_generation`
- 后端 [available_channel_handler.go](file:///c:/Users/Administrator/.openclaw/workspace/RelayQ-test/backend/internal/handler/available_channel_handler.go#L51-L179) 已经做了用户可见性过滤与平台隔离，不需要额外新发明“工作台专用模型列表”接口。

### 4. 当前工作台请求链路依赖浏览器 JWT，而不是用户自己的 API Key

- [modelTest.ts](file:///c:/Users/Administrator/.openclaw/workspace/RelayQ-test/frontend/src/api/modelTest.ts) 当前通过 `Authorization: Bearer <登录 JWT>` + `X-RelayQ-Source: playground` 调 `/v1/...`。
- 这条链路的计费、可见模型和权限实际上绕开了“用户到底选了哪个 API Key、这个 Key 绑定哪个 group、该 group 当前有哪些模型/定价”的真实业务语义。
- 用户现在希望移除这条免 API Key 模式，连管理员也不保留。

### 5. 当前“云端任务/作品”与本次改造弱耦合

- 第二期的 `playground_tasks` / `playground_assets` 是任务记录层。
- 它们不决定模型选择与计费来源。
- 本次改造不需要推翻任务中心与作品库，只需让新请求链路继续写入既有任务/作品记录。

## Assumptions & Decisions

### 已确认产品决策

- 完全移除“免 API Key 直计费”模式。
- 工作台所有工具统一切到“用户自己的 API Key”模式，不只图片工具。
- 模型列表不再硬编码，而是从用户当前可见渠道/分组动态计算。
- 计费按用户所选 API Key 所在分组与渠道定价执行，不再走 playground 特殊定价语义。

### 关键实现决策

- 不新增“第三套工作台专用模型配置”。
- 优先复用现有 `/keys`、`/channels/available`、`/groups/available`。
- 前端工作台新增“API Key 选择器”，选中后根据 key 的 `group_id` 反推出所属 group/platform，再从 `channels/available` 中筛出模型列表。
- 如果某个 API Key 未绑定 group，前端不允许用于工作台提交，要求用户先去 API Key 页绑定分组；不在工作台里偷偷选默认组。
- 工作台前端显示的价格仅作为“当前 group/channel 的前端提示”，最终结算以后端 usage 记录为准。
- 现有 `PlaygroundGuard` 与 `X-RelayQ-Source=playground` 链路整体下线，不保留兼容开关。
- 工作台请求改为真正使用“所选 API Key 的明文 key 值”调用 `/v1/...`；前提是后端 `/keys` 详情接口能返回创建时的一次性原始 key 或另有安全可用的“工作台专用短期 token”接口。仓库当前只明确存在 `GET /keys/:id`，但尚未从已读文件确认它是否会回传原始 key 文本，因此这一点需要在实施前先核实具体返回结构；如果不返回原始 key，则必须新增一个“获取该 key 的短期工作台访问票据”接口，不能在前端伪造。
- 因为用户要求的是“客户选择自己生成的 API Key”，工作台请求鉴权最终必须体现为该 API Key 所属权限，而不是继续只靠登录 JWT。

### 安全边界

- 若后端允许再次查看完整 API Key 明文，需要评估是否与现有产品安全策略一致；如果现有策略是“创建后只展示一次”，则更稳妥的是新增短期 session token / delegated token 接口，而不是把原始 key 再次暴露给浏览器。
- 计划默认优先做最小安全改动：
  1. 如果当前 `GET /keys/:id` 已返回可直接用于调用的 key 值，则复用现有接口；
  2. 如果不返回，则新增受登录态保护的“工作台会话票据”接口，由后端把用户选中的 key 解析成一次短时 bearer，用于工作台发 `/v1`。
- 执行阶段必须先验证这一事实，再决定采用 1 还是 2。

## Proposed Changes

### A. 前端工作台：增加 API Key 选择，移除硬编码模型

#### 1. 文件：`frontend/src/views/user/PlaygroundView.vue`

**改什么**

- 删除 `imageModels` / `chatModels` 等硬编码模型数组。
- 新增以下页面状态：
  - `availableKeys`
  - `selectedKeyId`
  - `selectedKeyDetail`
  - `availableChannels`
  - `resolvedGroup`
  - `resolvedPlatform`
  - `resolvedModelsByCapability`
  - `modelSelectionLoading`
  - `keyModeError`
- 顶部或参数区新增“API Key”选择控件，优先放在每个工具的第一个必选项，而不是藏在高级设置里。
- `button` 当前的模型下拉不再固定显示 `GPT Image 2`，而是显示“当前 API Key 所属分组下的可用图片模型”。
- 当用户切换 API Key 时：
  1. 拉取或读取该 key 详情；
  2. 检查是否绑定 `group_id`；
  3. 从 `channels/available` 中找到该 group 所属 platform；
  4. 基于工具类型筛选出可用模型：图片、聊天、视频；
  5. 自动回填第一个可用模型；若当前选择已失效则重置。
- 所有提交函数统一增加“未选 API Key / API Key 无分组 / 当前工具无可用模型”的前置校验。
- 现有 `onMounted(() => appStore.setSidebarCollapsed(true))` 保留不变。

**为什么**

- 这是这次需求的主 UI 入口。
- 模型可见性、计费来源、失败兜底都要从所选 key 出发。

**怎么做**

- 不拆大而全 store，继续沿用当前单文件页面状态，但把“模型解析逻辑”抽成小函数，避免几十处内联判断。
- 按能力筛模型时用最小规则：
  - 图片工具：优先使用 `allow_image_generation=true` 的 group 对应平台模型；再根据 `image_pricing` 或模型名判断图片能力。
  - 对话/文案/翻译：取当前平台可用文本/多模态模型。
  - 视频：只保留明确支持视频的平台模型；没有就禁用视频工具并显示原因。
- 不在首版计划里做跨平台智能推断库，只采用仓库已有字段 + 简单稳定规则。

#### 2. 文件：`frontend/src/api/modelTest.ts`

**改什么**

- 移除统一加 `X-RelayQ-Source: playground` 的逻辑。
- 请求鉴权改为“所选 API Key”模式。
- 所有导出方法签名新增“鉴权上下文”参数，例如：
  - `apiKeyValue` 或 `sessionToken`
  - `apiKeyId`
  - 可选 `groupId` / `platform` 仅供记录，不参与后端鉴权
- 保留图片请求的 `Idempotency-Key`。
- `editPlaygroundImage`、`generatePlaygroundImage`、`streamPlaygroundChat`、`runPlaygroundChat`、`createPlaygroundVideo`、`getPlaygroundVideo` 全部统一走新鉴权来源。

**为什么**

- 这是从 JWT playground 特权模式切换到用户真实 API Key 模式的核心节点。

**怎么做**

- 仅改请求头构造，不改每个 API 方法的核心解析逻辑。
- 如果执行阶段确认不能把原始 key 发给前端，则这里接收的是后端颁发的短期 token。

#### 3. 文件：`frontend/src/api/keys.ts`

**改什么**

- 保留已有 CRUD。
- 如当前 `ApiKey` 类型未包含工作台所需字段，则补齐前端读取字段：
  - `id`
  - `name`
  - `group_id`
  - `status`
  - 以及执行阶段核实后需要的 `key` / `masked_key` / session token 相关字段。
- 如需要短期票据接口，则在这里新增 `createWorkbenchSession(id)` 一类方法。

**为什么**

- 工作台现在依赖 API Key 作为第一入口。

#### 4. 文件：`frontend/src/api/channels.ts`

**改什么**

- 复用 `getAvailable()`。
- 可能只需补类型注释或帮助函数，不需要改 HTTP 接口。

**为什么**

- 该接口已经是动态模型与价格的真实来源。

#### 5. 文件：`frontend/src/types/...`（以现有 API Key 类型定义真实位置为准）

**改什么**

- 补齐 API Key 详情、工作台会话票据、按能力过滤模型所需的类型。
- 不额外创建复杂领域模型，保持和后端返回结构一一对应。

### B. 前端工作台：统一按所选 key 的 group/channel 动态显示价格与能力

#### 6. 文件：`frontend/src/views/user/PlaygroundView.vue`

**改什么**

- `imageModels[0].price` 这种硬编码价格删除。
- 图片工具预计价格改为：
  - 优先展示 `group.allow_image_generation` 对应的 `ImagePrice1K / 2K / 4K`；
  - 若模型自带 `image_pricing`，按选中尺寸映射展示；
  - 没有明确价格时显示“以渠道结算为准”。
- 对话/文案/翻译工具展示所选模型的 `pricing.billing_mode` 和可得价格。
- 视频若没有价格或模型能力信息，按钮禁用并展示“当前 API Key 分组未提供可用视频模型”。

**为什么**

- 用户的核心诉求之一就是按渠道定价扣费，而不是 playground 写死的价格提示。

### C. 后端：下线 playground 专用 guard，恢复统一 API Key 网关语义

#### 7. 文件：`backend/internal/handler/playground_guard.go`

**改什么**

- 删除或废弃 `playgroundAllowedModels` 固定 allowlist。
- 删除对 `X-RelayQ-Source=playground` 的特殊拦截。
- 如果该 middleware 仅用于工作台专用契约，直接从路由挂载处移除。
- 若还有 multipart/幂等性校验是普通网关也应保留的，则把它们迁移到更通用的位置，而不是继续放在 playground guard 里。

**为什么**

- 这个 guard 是当前模型找不到问题的后端根因之一。

#### 8. 文件：`backend/internal/server/routes/...`（以实际挂载 PlaygroundGuard 的路由文件为准）

**改什么**

- 去掉工作台专用中间件挂载。
- 确保 `/v1/...` 请求在用户 API Key 模式下继续走正常的网关鉴权、分组路由、价格结算、usage 统计。

**为什么**

- 只有完全走回正常 API Key 语义，工作台才不会和真实生产链路分叉。

### D. 后端：如有必要，补一个“工作台短期票据”接口

#### 9. 文件：`backend/internal/handler/...`、`backend/internal/server/routes/user.go`、`backend/internal/service/...`

**触发条件**

- 只有在执行阶段确认 `GET /keys/:id` 不会返回原始 key，且产品又不允许重新暴露完整 key 明文时，才新增。

**改什么**

- 新增受登录态保护的用户接口，例如：
  - `POST /api/v1/keys/:id/workbench-session`
- 请求体可为空，返回：
  - `token`
  - `expires_at`
  - `group_id`
  - `key_id`
- 后端在服务端根据该 key 生成短期 bearer，把它映射到真正的 API Key 权限。

**为什么**

- 这是“不把长期 API Key 明文回传浏览器”和“仍然让工作台按选中 key 实际鉴权”之间的最小折中。

**怎么做**

- 只做工作台最小票据，不扩展成通用 OAuth/STS 系统。
- 票据有效期短，例如 10–30 分钟；过期由前端静默重新申请。

### E. 后端：任务与作品记录继续保留，但补充所选 key 上下文

#### 10. 文件：`backend/internal/service/playground.go`、`backend/internal/repository/playground_repo.go`、对应 handler

**改什么**

- 当前任务/作品记录创建时，补充所选 `api_key_id`、`group_id`、`platform` 到 `request_payload` / `metadata`。
- 不新增数据库列，优先写入现有 JSON 字段，够用即可。

**为什么**

- 后续排查“某个 API Key 选了什么模型、为什么这个分组没视频模型、为什么价格不一致”会更直接。

### F. 前端路由与交互：增加无 key / 无 group / 无模型的明确空状态

#### 11. 文件：`frontend/src/views/user/PlaygroundView.vue`

**改什么**

- 工作台首页与各工具页新增 3 类阻断空状态：
  - 没有 API Key：引导去 `/keys` 创建。
  - API Key 未绑定分组：引导去编辑该 key 绑定 group。
  - 当前分组无对应能力模型：显示当前 group/platform 不支持该工具。
- 对视频工具单独显示能力缺失提示，不让用户点提交后才失败。

**为什么**

- 这是从“所有人都能点试玩”切到“真实 key 权限驱动”后必须补的 UX。

## Implementation Order

1. 核实 API Key 详情接口是否能安全提供工作台调用所需凭证。
2. 若不能，先补后端短期工作台票据接口与前端接入。
3. 改前端 `modelTest.ts` 请求头来源，移除 `X-RelayQ-Source=playground`。
4. 改 `PlaygroundView.vue`：接 API Key 列表 + channels/available，删硬编码模型与固定价格。
5. 为全工具接入“按所选 key 动态筛模型/价格/能力”。
6. 移除后端 `PlaygroundGuard` 与相关路由挂载。
7. 补任务/作品 metadata。
8. 回归测试图片、聊天、翻译、视频、批量图工具。

## Edge Cases / Failure Modes

- API Key 被禁用：工作台不允许选择或选择后立即报错。
- API Key 没有 group：工具区整体禁用。
- group 存在但 `channels/available` 因 feature flag 关闭返回空数组：工作台显示“当前站点未开启可用渠道查询，无法动态列模型”，不要悄悄退回硬编码模型。
- 管理员修改 group、channel restrict models 或模型名：用户刷新工作台后自动拿到新模型列表；如果当前已选模型失效，自动切到第一个可用模型并提示。
- 用户所选 key 的 group 不允许图片生成：图片工具、批量图工具、水印去除禁用；聊天工具如有文本模型则仍可用。
- 用户所选 key 的平台没有视频模型：仅视频工具禁用。
- 票据过期：前端收到 401/403 后自动刷新一次票据并重试一次，失败再提示重新选择 API Key。
- 渠道价格缺失：前端显示“以最终结算为准”，不乱算。
- 任务/作品列表中的旧 playground 记录没有 key 上下文：按旧数据兼容展示，不回填历史。

## Verification Steps

### 前端验证

- `pnpm run typecheck`
- `pnpm run build`
- 若已有 API 单测框架，补并运行：
  - `keys` / `channels` 联动解析测试
  - `modelTest.ts` 请求头不再包含 `X-RelayQ-Source`
  - 票据或 API Key 注入逻辑测试
  - “模型失效后自动回退到第一个可用模型”测试

### 后端验证

- 运行与 API Key、available channels、gateway、playground guard 相关测试。
- 新增或调整测试覆盖：
  - 工作台不再依赖 `PlaygroundGuard`
  - 普通 API Key 能正常调用图片、聊天、视频端点
  - 若新增 workbench session：
    - 只能为当前用户自己的 key 申请
    - 禁用 key / 无 group key / 不属于本人 key 会失败
    - 过期票据不可用

### 手工验收

- 用户没有 API Key 时，工作台显示创建引导。
- 用户有多个 API Key 时，切换不同 key，模型下拉内容随 group/platform 实时变化。
- 管理员改某 group 的模型映射后，刷新工作台不再出现硬编码旧模型。
- AI 生图、图片编辑、批量主图、批量克隆、水印去除使用同一个 key 成功提交并按正常 usage 计费。
- 对话助手、商品文案、图片翻译、AI 视频同样走所选 key。
- 任务中心与作品库仍可查看新请求生成的数据。

