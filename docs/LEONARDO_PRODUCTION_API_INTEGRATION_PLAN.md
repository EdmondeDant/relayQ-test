# RelayQ × Leonardo.Ai Production API 完整接入开发计划

## 2026-08-06 协议架构修订

- Leonardo 链路采用双模式，不再要求中转站逐字段复刻全部上游参数：
  - 标准 OpenAI 客户端（包括 ComfyUI、Infinite Canvas 的 OpenAI 节点）发送顶层 `model/prompt/size/n/quality/response_format`，RelayQ 转换为 Leonardo v2、等待异步任务完成并返回 OpenAI Images 响应。
  - 高级客户发送 Leonardo 原生 `model/public/parameters` 时，RelayQ 透明提交 JSON Body；Header、认证和目标 URL仍由 RelayQ控制。
- 两种协议字段混用时 fail closed，避免错误路由和错误计费。
- OpenAI 兼容入口默认 `quality=low`、`size=1024x1024`、`response_format=b64_json`，同步等待上限 900 秒；创建后通过 `X-RelayQ-Task-ID` 提供断线找回。
- OpenAI 客户端未发送 `Idempotency-Key` 时，按用户、API Key、路由、请求 Body 和 5 分钟时间桶生成内部幂等键。
- FLUX Schnell Raw Body 已支持 Content/Style；`image.source` 可使用 Data URI、受控 URL 或 `multipart://字段名`，上传后只定点写入 `id/type=UPLOADED`。
- 标准 OpenAI 协议无法表达 Content/Style 类型与 strength，不做隐式猜测；高级能力由 Leonardo Raw API 或 RelayQ 专用 ComfyUI 节点提供。
- FLUX Schnell 本地价格快照加入 896/1024/1120 三档；其他不超过原生上限的方形尺寸按当前档位最高 `$0.0045` 结算并标记 `quality_tier_max`。2880×2880 在 Pro Upscaler Precise 协议和价格验证前继续拒绝。
- 视频价格快照已具备 low/medium/high 档位最高价计算，但视频模型 UUID、创建参数和完成响应尚待逐次确认的真实付费探针；`video_enabled` 必须保持关闭。

> 文档状态：**Implementation Specification / 开发执行规格**  
> 版本：v1.1
> 日期：2026-08-06
> 项目：RelayQ（本地目录当前实际为 `C:\Users\Administrator\.openclaw\workspace\realyq-test`）  
> 开发负责人：Trae AI  
> 方案与代码审查：小腾  
> 产品决策与最终验收：腾哥

---

## 0. 文档用途与执行规则

本文件不是概念性调研报告，而是本次大型开发的**唯一主计划、接口契约和验收基线**。

Trae AI 应按本文任务编号逐项开发；每一阶段开始前和完成后，都需要同步：

1. 当前任务编号；
2. 准备修改的文件；
3. 真实调用链判断；
4. 实际完成的修改；
5. 执行的测试及原始结果；
6. 尚未解决的风险或文档歧义。

小腾负责：

- 逐阶段审查方案和代码；
- 对照 Leonardo 官方文档和 RelayQ 真实调用链检查；
- 补足遗漏、异常分支、计费和安全边界；
- 必要时提出最小修改建议；
- 以测试和实际 API 行为作为最终结论依据。

### 0.1 当前编码状态

- 当前执行目录：`C:\Users\Administrator\.openclaw\workspace\realyq-leonardo-prod`。
- 当前开发分支：`feat/leonardo-production-api`。
- 当前工作区包含大量尚未提交的 Leonardo 正式实现和测试，任何后续任务开始前必须执行 `git status --short`，不得执行会覆盖现场的 `reset`、`clean` 或覆盖式同步。
- 已有实现主体包括：独立 Leonardo Provider、Production API Client、`generation_jobs` 持久化任务、OpenAI Images 兼容入口、原生 Media API、图片编辑上传、价格预留、Poll/Webhook、unknown/manual review 和成本偏差告警。
- 当前实现仍是图片灰度 MVP，不代表完整 Production API 接入：已验证模型仍仅覆盖 FLUX Schnell；OpenAI/Leonardo Raw 双协议、1024/2048 产品尺寸识别和质量档位最高价基础已接入，但 2880 Upscale、视频、音频和 3D 尚未形成可用闭环。
- Windows 本地验证已确认前端 `3000`、后端 `8080`、PostgreSQL `5432`、Redis `6379` 可启动；后端构建、前端 typecheck 和 Leonardo Client 包测试通过。
- 尚未取得 Leonardo 真实成功付费 E2E 证据；历史创建探针包含 HTTP 500 和 `submission_unknown / side_effect_unknown`，禁止重放历史 unknown 请求。
- 本文第 21 节保留原始阶段任务，第 21.1 节作为当前唯一执行台账；任务状态必须以代码、测试和真实 API 证据更新，不以“已有文件”推定完成。

### 0.2 硬性开发约束

1. 不把 Leonardo 伪装成 OpenAI 平台。
2. 不把 Leonardo Production API 和现有 LeoStudio 网页 Cookie/JWT 逆向服务混为一套账号。
3. 不在 `openai_images.go` 中堆叠大量 Leonardo 特判。
4. 不为每个 Leonardo 模型手写一套完整 Go 请求结构。
5. 不盲目重试创建任务请求。
6. 不允许任务重复生成、重复计费。
7. 不允许 API Key、Webhook Secret、预签名表单敏感字段进入日志或管理 API 响应。
8. 不允许未经 SSRF 防护下载客户提供的 URL。
9. 不允许价格缺失时静默免费。
10. 每个非平凡任务必须有可运行测试。
11. 修改 Ent Schema 后必须规范生成代码和迁移，不手改生成文件冒充完成。
12. 第一阶段不做万能 JSON Schema 表单引擎，不做与当前交付无关的抽象。

---

# 1. 项目目标

将 Leonardo.Ai 官方 Production API 作为 RelayQ 的独立多媒体上游完整接入，使 RelayQ 能够统一提供：

- 图片生成；
- 图片编辑和多参考图；
- 视频生成和编辑；
- 音乐、音效、对白生成；
- 3D 资产生成；
- 图片/视频放大；
- 动态模型同步；
- 异步任务管理；
- Webhook 与轮询补偿；
- 上游成本、客户扣费和毛利记录；
- OpenAI 兼容入口；
- RelayQ 原生 Media API。

## 1.1 最终产品形态

客户侧同时提供两类协议：

### A. OpenAI 兼容门面

用于 Infinite Canvas、ComfyUI 标准 OpenAI 节点及其他已有客户端低成本迁移。客户端不需要理解 Leonardo 的异步协议，RelayQ 负责协议转换、任务创建、轮询和 OpenAI 响应转换：

```http
POST /v1/images/generations
POST /v1/images/edits
POST /v1/videos/generations
GET  /v1/videos/:id
GET  /v1/videos/:id/content
```

### B. RelayQ 原生异步 Media API

用于完整承载 Leonardo 的模型参数和多模态能力：

```http
POST /v1/media/generations
GET  /v1/media/generations/:id
GET  /v1/media/generations/:id/content
GET  /v1/media/generations
```

### C. Leonardo 原生 Raw 模式

高级客户可以向 Leonardo 分组的 `/v1/images/generations` 或 `/v1/media/generations` 发送含 `model/public/parameters` 的 Leonardo v2 JSON。RelayQ 原样转发 JSON Body，仅控制目标 URL、Authorization 和安全 Header，并继续负责计费、幂等和任务中心。

同一路径按请求结构自动识别协议：

- 顶层存在 `parameters`：Leonardo Raw；
- 无 `parameters` 且存在顶层 `prompt`：OpenAI 兼容；
- 两种协议字段混用：返回 400，不猜测优先级。

透明转发不能替代 OpenAI 兼容层。OpenAI 请求中的顶层 `prompt/size/n/quality` 不能直接提交 Leonardo `/v2/generations`。

## 1.2 非目标

本项目第一版不追求：

- 复制 Leonardo 网页端全部 UI；
- 让所有 Leonardo 模型都伪装成 OpenAI 标准模型；
- 第一版就为所有模型生成完全动态的复杂表单；
- 用现有网页 Cookie 账号替代官方 Production API；
- 自动推断不明确的销售价格；
- 为未来可能存在但当前未确认的接口提前搭建复杂抽象。

---

# 2. 官方 API 基线

## 2.1 鉴权与 Base URL

官方 Production API 使用：

```http
Authorization: Bearer <LEONARDO_API_KEY>
```

Production API v2 Base URL：

```text
https://cloud.leonardo.ai/api/rest/v2
```

注意：Leonardo 网页订阅和 Production API 计费相互独立。网页会员、网页 Tokens、Cookie/JWT 不代表具备 Production API 权限。

## 2.2 v2 统一创建任务

```http
POST https://cloud.leonardo.ai/api/rest/v2/generations
```

请求基础结构：

```json
{
  "model": "gpt-image-2",
  "public": false,
  "parameters": {
    "prompt": "A cinematic portrait",
    "quantity": 1,
    "width": 1024,
    "height": 1024,
    "quality": "MEDIUM"
  }
}
```

已确认创建响应至少包含：

```json
{
  "generationId": "uuid",
  "apiCreditCost": 123
}
```

PAYG 迁移期间还可能包含：

```json
{
  "cost": {
    "amount": 0.12,
    "unit": "USD"
  }
}
```

实现必须同时兼容 `cost` 和过渡字段 `apiCreditCost`，但不能把 Credits 和 USD 混算。

## 2.3 v2 模型目录

```http
GET https://cloud.leonardo.ai/api/rest/v2/models
```

`parameters` 是模型参数的 OpenAPI Schema，应作为动态模型能力的官方事实源。

LEO-001 实测 `GET /v2/models` 返回 **69 个模型**；每个模型包含 `id`、`name`、`parameters`，其中 `id` 为 UUID，响应中**没有独立 slug 字段**。该结果不能证明 UUID 与创建请求 `model` 值的映射关系。

模型目录 UUID 不能直接作为创建请求的 `model`；第一阶段仅使用经成功创建验证的映射，并在目录模型记录中分别保存：

```text
provider_model_id
request_model_slug
```

禁止把二者混为一个字段。

## 2.4 任务查询

Leonardo 官方指南确认 v2 创建、v1 查询：

```http
GET /generations/{id}
```

返回仍使用兼容结构：

```json
{
  "generations_by_pk": {
    "id": "...",
    "status": "PENDING | COMPLETE | FAILED",
    "generated_images": []
  }
}
```

LEO-002 已执行一次 `POST /api/rest/v2/generations` 创建探针，但响应为 HTTP 500 且没有 `generationId`，因此未执行状态查询；`GET /api/rest/v1/generations/{id}` 来自官方指南，尚未实测。

```text
POST /api/rest/v2/generations
GET  /api/rest/v1/generations/{id}  # 官方指南路径，尚未实测
```

Client 仍应把创建与查询路径分别封装为单独方法/常量，不把版本差异散落在业务代码中。

## 2.5 图片上传

当前官方文档仍使用：

```http
POST https://cloud.leonardo.ai/api/rest/v1/init-image
```

请求：

```json
{
  "extension": "jpg"
}
```

响应包含：

```text
uploadInitImage.id
uploadInitImage.url
uploadInitImage.fields
uploadInitImage.key
```

上传流程：

1. 请求预签名 URL；
2. 解析 `fields`（官方示例中可能是 JSON 字符串）；
3. 向预签名 URL 发送 multipart 表单；
4. **不带 Leonardo Authorization Header**；
5. `204 No Content` 是正常成功；
6. 第一步返回的 `id` 即后续引用的 uploaded image ID。

## 2.6 Webhook

Webhook 在创建 Production API Key 时配置：

- Callback URL 必须 HTTPS；
- Callback Secret 通过以下请求头传递：

```http
Authorization: Bearer <WEBHOOK_SECRET>
```

官方示例事件：

```json
{
  "type": "image_generation.complete",
  "object": "generation",
  "timestamp": 1699490546932,
  "api_version": "v1",
  "data": {
    "object": {
      "id": "...",
      "status": "COMPLETE",
      "images": [
        {
          "id": "...",
          "url": "https://...",
          "nsfw": false
        }
      ]
    }
  }
}
```

Webhook 示例仍偏 v1。实现时必须允许保存经过脱敏的原始 JSON 以兼容未来媒体类型，但业务解析器不能假设所有回调只有 `images`。

## 2.7 官方限流概念

Leonardo 区分三种限制：

1. HTTP API Rate Limit；
2. Generation Concurrency；
3. Queue/Pending Limit。

RelayQ 不能只用一个 HTTP 并发信号量替代三者。

具体默认值可能依账号方案变化，不在代码中猜测硬编码。应支持后台配置和响应头/错误信息驱动的动态暂停。

## 2.8 内容审核

Leonardo 默认会：

- 在 Prompt 级别阻断部分 NSFW 请求；
- 在生成结果中返回 `nsfw` 标记。

RelayQ 仍需保留自身前置审核和输出结果过滤，不能完全依赖上游。

## 2.9 模型弃用

官方模型和参数会快速变化。已发生的类型包括：

- 模型下线；
- preview alias 删除；
- 参数改名（例如 `mode` → `quality`）；
- 第三方模型提供商驱动的突然下线。

必须建立模型 `last_seen_at`、`deprecated` 和同步差异提示，不允许仅靠代码硬编码长期维持。

---

# 3. RelayQ 当前真实基础

## 3.1 图片链路

入口：

```text
backend/internal/handler/openai_images.go
```

主服务：

```text
backend/internal/service/openai_images.go
```

已有能力：

- OpenAI Images generations/edits；
- JSON 和 multipart 解析；
- 图片权限；
- 内容审核；
- 用户和账号并发限制；
- 账号调度；
- failover；
- 模型映射；
- 图片 URL/base64 转换；
- 输出数量和尺寸统计；
- 图片计费。

当前模型识别主要依赖：

```text
gpt-image-*
grok-imagine-image*
```

因此 Leonardo 的 `nano-banana-2`、`seedream-4.5`、`flux-pro-2.0` 等不能仅通过添加 Base URL 接入。

## 3.2 视频链路

入口：

```text
backend/internal/handler/openai_videos.go
backend/internal/service/openai_videos.go
```

当前强绑定 xAI 平台和 xAI 参数。Leonardo 视频应通过 Provider Adapter 接入，不应继续扩大 `XAIVideo` 分支。

## 3.3 账号与调度

账号表已经提供：

- `platform`
- `type`
- `credentials` JSONB
- `extra` JSONB
- proxy
- concurrency/load factor
- priority
- rate multiplier
- status/schedulable
- rate limit reset
- overload/temp unschedulable
- group 关联

Leonardo 第一阶段无需给账号表新增大量专属列；平台特有配置放入 `credentials` 和 `extra`。

## 3.4 现有模型同步

主要位置：

```text
backend/internal/service/upstream_models.go
backend/internal/handler/admin/account_handler.go
backend/internal/server/routes/admin.go
```

现有后台路由包含：

```http
POST /admin/accounts/models/sync-upstream-preview
POST /admin/accounts/:id/models/sync-upstream
```

Leonardo 应接入此主干，而不是再建立一套孤立的账号模型同步 API。

## 3.5 现有计费

主要位置：

```text
backend/internal/service/billing_service.go
backend/internal/service/usage_billing.go
backend/internal/service/model_pricing_resolver.go
backend/internal/service/image_billing_size.go
backend/ent/schema/usage_log.go
```

现有图片 `1K/2K/4K` 计费不能完整覆盖 Leonardo 视频、音频、3D、时长和实际 PAYG 成本。

---

# 4. 总体架构

```text
                               RelayQ Client
                                    │
                  ┌─────────────────┴─────────────────┐
                  │                                   │
          OpenAI-compatible API                RelayQ Media API
        /v1/images/* /v1/videos/*          /v1/media/generations
                  │                                   │
                  └─────────────────┬─────────────────┘
                                    │
                         Canonical Media Request
                                    │
                       Provider Router + Scheduler
                                    │
             ┌──────────────────────┼──────────────────────┐
             │                      │                      │
       OpenAI Adapter          xAI Adapter          Leonardo Adapter
                                                            │
                                  ┌─────────────────────────┼─────────────┐
                                  │                         │             │
                            v2/generations              v2/models    v1/init-image
                                  │
                          generation_jobs DB
                                  │
                   Webhook + Poller + Reconciliation
                                  │
                        Canonical Media Result
                                  │
                      Billing + Usage + Client Output
```

## 4.1 Provider 边界

定义独立平台：

```go
const PlatformLeonardo = "leonardo"
```

推荐保留概念区分：

```text
leonardo          官方 Production API
leostudio_legacy  现有网页 Cookie/JWT 逆向服务（如需显式标识）
```

第一阶段不强制重命名现有 LeoStudio 平台字段，但代码和文档中必须明确两者不是同一认证体系。

## 4.2 最小 Provider 接口

不要一开始设计庞大插件框架。仅抽出当前实际需要的内部边界，例如：

```go
type MediaProvider interface {
    CreateGeneration(ctx context.Context, account *Account, req MediaGenerationRequest) (*ProviderGeneration, error)
    GetGeneration(ctx context.Context, account *Account, upstreamID string) (*ProviderGeneration, error)
}
```

上传作为可选能力，不必强塞进所有 Provider：

```go
type MediaUploadProvider interface {
    UploadImage(ctx context.Context, account *Account, input MediaUploadInput) (*ProviderMediaRef, error)
}
```

如果第一阶段只有 Leonardo 实现，可以先用具体 `LeonardoClient` 和路由 switch；当第二个异步 Provider 真正接入时再稳定接口。禁止为“未来也许会有”提前堆工厂和注册中心。

---

# 5. 账号与配置契约

## 5.1 账号示例

```json
{
  "platform": "leonardo",
  "type": "apikey",
  "credentials": {
    "api_key": "***",
    "base_url": "https://cloud.leonardo.ai/api/rest"
  },
  "extra": {
    "api_version": "v2",
    "query_api_version": "v1",
    "webhook_enabled": true,
    "webhook_secret": "***",
    "supported_models": [
      "gpt-image-2",
      "nano-banana-2"
    ],
    "supported_modalities": [
      "image",
      "video",
      "audio",
      "3d",
      "upscale"
    ],
    "leonardo_limits": {
      "rpm": 60,
      "max_active_jobs": 10,
      "max_pending_jobs": 50
    }
  }
}
```

`rpm/max_active_jobs/max_pending_jobs` 只是配置结构示例，不是官方默认值。UI 和后端允许留空。

## 5.2 Base URL

默认：

```text
https://cloud.leonardo.ai/api/rest
```

Client 自己附加 `/v1` 或 `/v2`，避免账号配置在不同版本间混乱。

必须使用 RelayQ 现有 `validateUpstreamBaseURL` 及 SSRF 防护，不能因为是管理员配置就完全放弃校验。

## 5.3 凭证安全

敏感字段至少包括：

```text
api_key
webhook_secret
webhook_callback_api_key
authorization
```

要求：

- 管理 API 响应不返回明文；
- 编辑时空值表示保留旧 Key；
- 日志不打印；
- 错误信息不拼入请求头；
- Webhook 原始负载中的 `apiKey` 对象必须剥离；
- `credentials_status.has_api_key` 可用于前端显示已配置状态。

## 5.4 账号测试

Leonardo 账号测试第一步只调用：

```http
GET /v2/models
```

成功条件：

- HTTP 2xx；
- JSON 可解析；
- `productionApiAvailableModels` 非空。

默认账号测试禁止直接创建付费媒体。若后台提供“付费生成测试”，必须单独按钮、明确提示预计产生费用，并由管理员主动触发。

---

# 6. 动态模型目录

## 6.1 数据表建议

新增：

```text
provider_model_catalog
```

建议字段：

```text
id                  bigint / internal id
provider            varchar, indexed
provider_model_id   varchar
request_model_slug  varchar
name                varchar
modality             varchar
parameter_schema     jsonb
schema_hash          varchar
active               bool
deprecated           bool
first_seen_at        timestamptz
last_seen_at         timestamptz
synced_at            timestamptz
raw_metadata         jsonb (sanitized)
created_at
updated_at
```

唯一索引：

```text
(provider, provider_model_id)
(provider, request_model_slug) where request_model_slug is not null
```

## 6.2 模态分类

优先从官方 Schema/元数据中获取。若官方响应没有 modality，第一阶段使用明确白名单映射，不用模糊字符串猜测作为最终真相。

允许分类：

```text
image
video
audio
3d
upscale
unknown
```

`unknown` 模型可以同步到后台，但默认不向客户开放。

## 6.3 同步行为

1. 拉取 `/v2/models`；
2. 严格限制响应体大小；
3. 解析模型；
4. Upsert 目录；
5. 更新本次出现模型的 `last_seen_at`；
6. 未出现模型不立刻物理删除；
7. 连续若干次未出现后标记 inactive/deprecated；
8. 输出同步差异：新增、更新、消失、Schema 变化；
9. 账号 `supported_models` 只保存允许调度的模型 slug；
10. 同步失败不清空旧目录。

## 6.4 模型 ID 与 slug

创建 API 使用 slug，例如：

```text
gpt-image-2
nano-banana-2
seedream-4.5
grok-imagine-1.5
```

LEO-001 实测模型目录返回 UUID，而不是可直接用于创建请求的 slug。目录 UUID、显示名和创建请求 `model` 值必须分别保存。

必须显式保存映射。映射来源优先级：

1. 官方响应中的直接 slug 字段；
2. 官方 OpenAPI discriminator；
3. 经成功创建探针验证的静态映射；
4. 未确认则标记 unavailable，不自动转换。

---

# 7. Canonical Media 数据结构

## 7.1 请求

建议内部结构：

```go
type MediaModality string

const (
    MediaModalityImage   MediaModality = "image"
    MediaModalityVideo   MediaModality = "video"
    MediaModalityAudio   MediaModality = "audio"
    MediaModality3D      MediaModality = "3d"
    MediaModalityUpscale MediaModality = "upscale"
)

type MediaGenerationRequest struct {
    Model           string
    Modality        MediaModality
    Prompt          string
    Quantity        int
    Width           int
    Height          int
    DurationSeconds int
    Public          bool
    Inputs          []MediaInput
    Parameters      map[string]any
}

type MediaInput struct {
    Role       string
    URL        string
    Data       []byte
    MIMEType   string
    ProviderID string
    SourceType string
    Strength   string
}
```

公共字段统一，模型特有字段保留在 `Parameters`，不要尝试做一个覆盖全部模型的超级结构。

## 7.2 输出

```go
type ProviderGeneration struct {
    UpstreamID string
    Status     string
    Outputs    []MediaOutput
    Cost       *ProviderCost
    Raw        json.RawMessage
}

type MediaOutput struct {
    ProviderID string
    Type       string
    URL        string
    MIMEType   string
    Width      int
    Height     int
    Duration   float64
    NSFW       bool
}

type ProviderCost struct {
    Amount float64
    Unit   string // USD | CREDIT
}
```

`Raw` 只能保存经过大小限制和敏感字段脱敏后的数据。

---

# 8. 持久化异步任务中心

## 8.1 为什么必须持久化

Redis request→account TTL 不足以支撑：

- 服务重启；
- 长时间视频任务；
- Webhook 丢失；
- 客户多次查询；
- 创建响应不确定；
- 计费幂等；
- 后台审计。

## 8.2 `generation_jobs` Schema

建议新增 Ent Schema：

```text
backend/ent/schema/generation_job.go
```

字段建议：

```text
id                         bigint
public_id                  varchar, unique, indexed
provider                   varchar, indexed
modality                   varchar, indexed
model                      varchar
upstream_model             varchar
user_id                    bigint, indexed
api_key_id                 bigint, indexed
group_id                   bigint, nullable
account_id                 bigint, indexed
upstream_generation_id     varchar, nullable, indexed
status                     varchar, indexed
upstream_status            varchar, nullable
request_hash               varchar
request_payload            jsonb (sanitized)
result_payload             jsonb (sanitized)
error_code                 varchar, nullable
error_message              text, nullable
output_count               int
actual_upstream_cost_amount decimal, nullable
actual_upstream_cost_unit   varchar, nullable
customer_cost              decimal, nullable
billing_status             varchar
billing_reference          varchar, nullable
poll_attempts              int
next_poll_at               timestamptz, nullable, indexed
last_polled_at             timestamptz, nullable
submitted_at               timestamptz, nullable
started_at                 timestamptz, nullable
completed_at               timestamptz, nullable
failed_at                  timestamptz, nullable
created_at
updated_at
```

## 8.3 状态机

RelayQ 标准状态：

```text
created
submitting
queued
running
succeeded
failed
cancelled
unknown
```

允许转换：

```text
created -> submitting
submitting -> queued | running | succeeded | failed | unknown
queued -> running | succeeded | failed | cancelled
running -> succeeded | failed | cancelled
unknown -> queued | running | succeeded | failed
```

终态：

```text
succeeded
failed
cancelled
```

禁止从终态回到运行态，除非管理员执行明确的修复操作并留下审计记录。

## 8.4 `unknown` 状态

以下情况进入 `unknown`：

- POST 已发送，但读取响应前网络断开；
- 上游返回 2xx 但响应体无法解析 generationId；
- `POST /v2/generations` 请求可能已经发送、返回 HTTP 500 且没有 generationId，并且无法证明上游未受理；
- 代理在可能送达请求后超时；
- 无法确认上游是否受理。

**unknown 任务禁止自动重新创建。**

进入 `unknown` 时，`error_code=submission_unknown`、`billing_status=manual_review`；`actual_upstream_cost_amount`、`actual_upstream_cost_unit`、`customer_cost`、`gross_margin` 和 `cost_variance` 均保持 `null`。不得用预估成本、报价或 0 填充实际成本，也不得扣款、结算、退款或释放、消费资金预留，直至取得 Leonardo 明确成本、账单或任务证据并经人工处理。

处理方式：

- 查询上游用户最近任务（若官方支持且能可靠关联）；
- 依赖 Webhook 回填；
- 管理员人工确认；
- 通过 request hash、时间、账号和模型做有限匹配，但不得仅凭模糊匹配自动扣两次费。

## 8.5 请求幂等

RelayQ 应支持客户 `Idempotency-Key`：

- 同一用户/API Key + 同一 key + 同一请求 hash 返回已有任务；
- 同一 key 但请求内容不同返回 409；
- 不将 RelayQ 幂等键假设为 Leonardo 官方幂等键，除非文档明确支持。

OpenAI 客户端兼容规则：

- 客户显式提供 `Idempotency-Key` 时严格使用该值和标准 TTL；
- 客户未提供时，按 `user ID + API key ID + 规范化路由 + 完整请求 Body SHA-256 + 5 分钟时间桶` 生成内部键；
- 自动键使用独立 scope，不能与显式键碰撞；
- 同一主体、同一请求在同一 5 分钟窗口内复用任务，跨窗口视为新任务；
- 自动键 TTL 必须覆盖 5 分钟窗口和 900 秒同步等待；
- Body、quality、response format 或参考图内容变化必须形成不同请求指纹。

可复用现有 `idempotency_record` 模型或将 key 记录在任务表，选择前先审查现有实现。

---

# 9. Leonardo Client 设计

建议目录：

```text
backend/internal/pkg/leonardo/
```

或按项目既有习惯放 `backend/internal/service/leonardo_*.go`。二选一即可，不要重复两层封装。

## 9.1 Client 方法

第一阶段必须包含：

```go
ListModels(ctx, auth) ([]Model, error)
CreateGeneration(ctx, auth, req) (*CreateGenerationResponse, error)
CreateGenerationRaw(ctx, auth, body []byte) (*CreateGenerationResponse, error)
GetGeneration(ctx, auth, id) (*Generation, error)
CreateInitImageUpload(ctx, auth, extension) (*InitImageUpload, error)
UploadInitImage(ctx, presigned, file) error
```

## 9.2 HTTP 要求

- 使用 RelayQ 现有 HTTP Upstream/Proxy 基础设施；
- 支持账号代理；
- 连接、响应头、总请求超时分开考虑；
- 限制错误体和成功体大小；
- 保留安全的 request-id；
- 解析 `Retry-After`；
- 不记录 Authorization；
- 采集上游错误时先限制响应体大小，再对 Key、Authorization、Cookie、签名、预签名字段、内部账号及代理信息脱敏；日志、任务审计和错误响应只能保存脱敏结果；
- 错误中输出脱敏后的上游 reason；
- `POST /v2/generations` 请求体已开始发送或是否发送无法确认后，任何层均不得自动重试或切换账号，包括 Leonardo Client、HTTP Transport、通用 retry middleware、反向代理、任务队列、Worker、超时恢复和故障转移逻辑；HTTP 429、500、502、503、504、超时、连接中断及 2xx 响应解析失败均适用；
- GET 模型/状态允许受控重试。
- `CreateGenerationRaw` 必须按原始字节发送合法 JSON；不得先反序列化为 map 再 Marshal，除非请求包含 RelayQ `image.source` 且必须做定点上传替换；
- Raw Body 只用于即时提交和哈希，任务审计只保存递归脱敏后的 JSON，不得持久化 Data URI、预签名 URL、Cookie 或认证字段。

## 9.3 上游错误类型

建议：

```go
type LeonardoError struct {
    StatusCode       int
    Code             string
    Message          string
    Path             string
    RequestID        string
    RetryAfter       time.Duration
    RetryableRead    bool
    SubmissionStatus string
}
```

Leonardo 响应 `error` → `Message`、`code` → `Code`、`path` → `Path`；`RequestID` 仅取允许采集的响应头 request-id，未返回或未采集时保持空值，不得伪造。`RetryableRead` 只控制 GET 等读取请求，不得使创建 POST 可重试；`SubmissionStatus` 标记创建副作用是否明确。

需要区分：

- 客户参数错误；
- 内容审核；
- Key 失效；
- 余额不足；
- Rate Limit；
- Queue Full；
- 临时服务错误；
- 提交状态未知。

---

# 10. 图片上传与 URL 安全

## 10.1 支持输入

- multipart 文件；
- data URI；
- 公网 HTTPS/HTTP URL；
- 已有 Leonardo uploaded image ID；
- 已有 Leonardo generated image ID；
- RelayQ 历史任务媒体引用。

## 10.2 URL 下载安全

必须复用或加强 RelayQ 现有媒体 SSRF 安全逻辑：

- 仅允许 http/https；
- 拒绝 localhost；
- 拒绝 loopback/private/link-local/unspecified；
- DNS 解析后检查 IP；
- 每次重定向重新检查目标；
- 限制重定向次数；
- 禁止访问云元数据地址；
- 限制响应体大小；
- 限制下载总时长；
- 校验 Content-Type 和文件魔数；
- 不信任文件名扩展；
- 临时文件可靠清理。

## 10.3 上传行为

- 扩展名仅允许官方支持类型；
- `fields` 兼容字符串 JSON 或对象；
- 预签名上传禁止带 Leonardo Bearer；
- `204` 视为成功；
- 其他 2xx 是否成功通过真实 API 验证后决定；
- 失败时不创建 generation；
- 上传 ID 和源文件 hash 可短期缓存，避免同任务重复上传。

---

# 11. OpenAI 图片兼容桥

## 11.1 输入示例

```json
{
  "model": "nano-banana-2",
  "prompt": "一只穿西装的龙虾",
  "n": 1,
  "size": "1024x1024",
  "response_format": "url",
  "provider_options": {
    "leonardo": {
      "prompt_enhance": "OFF",
      "style_ids": [
        "111dc692-d470-4eec-b791-3475abac4c46"
      ]
    }
  }
}
```

典型 ComfyUI/Infinite Canvas OpenAI 节点请求：

```json
{
  "model": "flux-schnell",
  "prompt": "一只穿西装的龙虾",
  "n": 1,
  "size": "1024x1024",
  "quality": "low",
  "response_format": "b64_json"
}
```

## 11.2 基础映射

```text
model           -> Leonardo model slug
prompt          -> parameters.prompt
n               -> parameters.quantity
size            -> width + height
quality         -> parameters.quality（按模型 Schema 转换大小写/枚举）
provider_options.leonardo -> 模型特有参数白名单透传
```

禁止把未知 OpenAI 字段全部透传给 Leonardo。

OpenAI 分支必须构造 Leonardo typed request；只有检测为 Leonardo Raw 的请求才允许透明 Body 提交。

## 11.3 模型识别

当前 `isOpenAIImageGenerationModel` 仅识别 OpenAI/xAI 前缀。不得改成简单的 `strings.Contains(model, "image")` 模糊判断。

正确流程：

1. 查询 RelayQ 模型目录；
2. 确认模型 modality=image；
3. 确认当前分组有支持该模型的账号；
4. 进入 Provider Router。

为启动期间目录不可用可以保留少量经过验证的内置模型元数据，但不能成为长期唯一来源。

## 11.4 同步响应策略

OpenAI 图片接口默认同步，最长等待 **900 秒**。

- 在窗口内完成：返回标准 OpenAI Images 结果；
- 达到 900 秒：返回 OpenAI 风格 504 和 `generation_timeout`，不能伪造失败后重试；
- 客户端断线或超时后，上游任务继续运行并正常计费，不自动取消、不重复提交；
- 创建任务后立即设置 `X-RelayQ-Task-ID` 和 `X-Request-ID`，客户可通过 `/v1/media/generations/:id` 找回结果；
- 请求 Context 取消只停止当前 HTTP 等待，后台 Poller 继续推进任务；
- 显式 `async:true` 返回 202 `{created,task_id,status}`，仅作为 RelayQ 扩展。

同步成功响应不得混入 RelayQ task 字段，task ID 只放响应头。错误保持 `{error:{message,type,param,code}}`，submission unknown 必须明确提示不得立即重试。

## 11.5 输出转换

Leonardo 成功结果转换为：

```json
{
  "created": 1785571200,
  "data": [
    {
      "url": "https://..."
    }
  ]
}
```

`response_format=b64_json` 时：

- 如果上游只给 URL，RelayQ 服务端受控下载并转 base64；
- 遵守体积和超时限制；
- 下载失败不能算生成失败并再次提交；
- 任务仍保持 succeeded，但客户端响应可以报告结果下载失败。

未传 `response_format` 时默认 `b64_json`，避免通用客户端受 Leonardo 临时 URL、跨域和 URL 过期影响。显式 `url` 时返回经过安全校验的 Leonardo HTTPS URL。

## 11.6 图片编辑

`/v1/images/edits`：

1. 解析 multipart/JSON；
2. 上传输入图；
3. 根据模型 Schema 组成 `guidances.image_reference`；
4. 区分 `UPLOADED` 和 `GENERATED`；
5. mask/inpainting 只有模型明确支持时开放；
6. 不支持 mask 时返回明确 400，而非忽略 mask。

标准 OpenAI `/v1/images/edits` 不能表达 Leonardo Content、Style、Character 类型和各自 strength，不得把单张 `image` 擅自解释为 Content 或 Style。高级参考图能力只通过 Leonardo Raw API 或 RelayQ 专用 ComfyUI 节点提供。

## 11.7 OpenAI 客户端默认值

| 字段 | 默认值 | 规则 |
|---|---|---|
| `n` | `1` | 首批只开放单图 |
| `size` | `1024x1024` | 允许 `1024x1024`、`2048x2048`、`2880x2880` 产品档位 |
| `quality` | `low` | 只接受 `low/medium/high`，按模型官方能力映射 |
| `response_format` | `b64_json` | 可显式选择 `url` |
| `async` | `false` | `true` 为 RelayQ 扩展任务模式 |

- 模型没有官方 `quality/mode` 时不得伪造上游字段；不支持的档位返回 `quality_not_supported`；
- 需要 Upscaler 才能交付的尺寸，在 Upscaler 协议和价格验证前必须提交前拒绝；
- 最终交付尺寸必须与请求严格一致，不得静默降级。

## 11.8 双模式协议探测

```text
POST /v1/images/generations
  ├─ parameters object 存在 → Leonardo Raw → HTTP 202 RelayQ task
  ├─ 顶层 prompt 存在      → OpenAI Adapter → 默认同步 OpenAI response
  └─ 两套字段混用          → HTTP 400 protocol_conflict
```

- 不通过 User-Agent、ComfyUI 名称或模型名称猜客户端；
- Raw 请求至少要求 `model` 和对象类型 `parameters`；
- Raw 请求未知字段进入完整 Body 指纹并原样提交；
- OpenAI 未知字段继续严格拒绝或按明确白名单转换；
- 两种模式共用账号路由、价格、预留、任务状态、Poll/Webhook 和 unknown 保护。

## 11.9 ComfyUI 与 Infinite Canvas 能力边界

- 标准 OpenAI 节点保证文生图和能够无损映射的标准 edits；
- 标准节点不保证 Content/Style/Character 等 Leonardo 专用 Guidance；
- RelayQ 专用 ComfyUI 节点分为文生图节点和 Leonardo Advanced 节点；
- Advanced 节点提供 reference type、strength、source、quality、size 和 async/sync 控件；
- API Key 只保存在 ComfyUI 本地凭据配置，不进入 workflow 明文字段或日志；
- 专用节点在后端 OpenAI/Raw 双模式稳定后交付，不阻塞基础 API 灰度。

## 11.10 客户端兼容验收

- 模拟无 `Idempotency-Key`、无 `async`、无 `response_format` 的 ComfyUI 请求，最终返回 HTTP 200 和 `data[].b64_json`；
- 模拟 Infinite Canvas 请求，分别验证 URL 和 base64 输出；
- 验证同一请求 5 分钟内只调用一次 Leonardo；
- 模拟客户端断线，后台任务完成后可通过 `X-RelayQ-Task-ID` 查询；
- 回归 Leonardo Raw、Media、Images Edits 以及非 Leonardo OpenAI/xAI 路由；
- 实际客户端付费验收每个 POST 单独审批，禁止自动执行。

---

# 12. 原生 Media API 契约

## 12.1 创建任务

```http
POST /v1/media/generations
```

请求：

```json
{
  "model": "grok-imagine-1.5",
  "modality": "video",
  "prompt": "A cinematic city at dusk",
  "public": false,
  "input": [
    {
      "role": "start_frame",
      "url": "https://example.com/frame.png"
    }
  ],
  "parameters": {
    "duration": 6,
    "width": 720,
    "height": 1280,
    "motion_has_audio": true
  }
}
```

响应：

```json
{
  "id": "gen_rq_xxx",
  "object": "media.generation",
  "provider": "leonardo",
  "model": "grok-imagine-1.5",
  "modality": "video",
  "status": "queued",
  "created_at": 1785571200
}
```

## 12.2 查询任务

```http
GET /v1/media/generations/:id
```

响应：

```json
{
  "id": "gen_rq_xxx",
  "object": "media.generation",
  "provider": "leonardo",
  "model": "grok-imagine-1.5",
  "modality": "video",
  "status": "succeeded",
  "outputs": [
    {
      "type": "video",
      "url": "https://www.relayq.top/v1/media/generations/gen_rq_xxx/content",
      "mime_type": "video/mp4",
      "width": 720,
      "height": 1280,
      "duration": 6
    }
  ],
  "created_at": 1785571200,
  "completed_at": 1785571260
}
```

普通客户响应不暴露：

- account_id；
- upstream generation id（除非有调试权限）；
- 上游真实 URL（如需隐藏拓扑）；
- upstream cost；
- 原始 Webhook；
- Key 信息。

## 12.3 内容端点

```http
GET /v1/media/generations/:id/content
```

规则：

- 校验任务归属和 API Key 权限；
- succeeded 才允许下载；
- 支持 Range（视频体验需要）；
- 可选择 302 到安全 CDN URL，或由 RelayQ 代理；
- 如果隐藏上游拓扑，使用代理/CDN 缓存；
- 不把 Leonardo Bearer 发送给 CDN；
- 多输出任务支持 `?index=0` 或独立 output id。

## 12.4 列表

```http
GET /v1/media/generations?status=succeeded&modality=image&limit=20&after=...
```

必须按当前认证主体过滤，管理员 API 另设。

---

# 13. Webhook、轮询和恢复

## 13.1 Webhook 路由

建议内部路由：

```http
POST /internal/webhooks/leonardo/:account_id/:route_token
```

安全要求：

1. HTTPS；
2. 不可猜测 route token；
3. 校验 `Authorization: Bearer`；
4. 常量时间比较；
5. Body 大小限制；
6. 可选官方 IP allowlist，但不能代替 Secret；
7. 事件幂等；
8. 先快速入库，再异步处理；
9. 不记录完整敏感负载。

## 13.2 Webhook 幂等

建议新增 webhook event 表或在任务事件记录中唯一约束：

```text
provider + upstream_generation_id + event_type + timestamp/hash
```

重复 Webhook：

- 返回 2xx；
- 不重复结算；
- 不重复写 outputs；
- 不把终态改回运行态。

## 13.3 轮询器

扫描条件：

```text
status in (queued, running, unknown)
next_poll_at <= now
```

退避建议：

- 图片初始 2–5 秒；
- 视频初始 5–15 秒；
- 逐渐增加，设最大间隔；
- 加随机抖动；
- 尊重 Retry-After；
- 超过任务最大时长不直接删除，标记超时/unknown 并保留审计。

## 13.4 服务重启恢复

启动后：

- 不把全部任务同时立即轮询；
- 分批扫描；
- 重建账号 active/pending 计数；
- 对终态但 billing pending 的任务执行幂等结算；
- 对 queued/running 继续查询；
- 对 submitting/unknown 进入专门 reconciliation。

## 13.5 Webhook 与 Poll 竞争

数据库更新必须带状态条件或乐观锁：

```text
UPDATE ... WHERE status NOT IN (succeeded, failed, cancelled)
```

输出和计费必须幂等，Webhook 和 Poll 同时到达也只能成功一次。

---

# 14. 调度与限流

## 14.1 三层限制

每个 Leonardo 账号分别管理：

```text
HTTP RPM
Active Generation Jobs
Pending/Queued Jobs
```

## 14.2 账号选择

候选条件：

- platform=leonardo；
- status active；
- schedulable；
- 未过期；
- 不在 rate limit/overload/temp disabled；
- supported_models 包含模型；
- supported_modalities 包含模态；
- active/pending 未达到配置上限；
- 分组允许该账号。

在符合条件的候选中复用现有 priority/load factor/account multiplier 调度逻辑。

## 14.3 创建后占用

- 创建前可占本地 pending reservation；
- 得到 generationId 后确认占用；
- 任务终态释放；
- 创建明确失败释放；
- unknown 不立即释放，避免过量提交；
- Redis 计数失真时由数据库任务重建。

## 14.4 错误副作用

```text
401/403 invalid key  -> account error / unschedulable
402/余额不足         -> account quota exhausted / pause
429 rate limit       -> rate_limit_reset_at
queue full           -> short temp unschedulable
5xx overload         -> overload_until
content moderation   -> 不惩罚账号
客户参数 400         -> 不切账号
```

---

# 15. 重试与 Failover 矩阵

## 15.1 可以安全重试

- GET `/v2/models`；
- GET generation status；
- Webhook 后补拉详情；
- 429 且有 Retry-After 的查询；
- 查询类 502/503；
- 请求体尚未开始发送且可证明未到达上游的连接建立失败；
- 预签名上传在能确认上传未成功时重试。

## 15.2 不可盲目重试

HTTP 失败只描述响应结果，不代表创建没有产生副作用；`POST /v2/generations` 请求体已开始发送或是否发送无法确认时，不得自动重试。

- `POST /v2/generations` 请求体发送后超时；
- `POST /v2/generations` 请求可能已经发送、返回 HTTP 500 且没有 generationId，并且无法证明上游未受理；
- 已收到 2xx 但解析失败；
- 已拿到 generationId；
- 客户断开连接；
- 代理连接中途断开但不能确认上游未受理。

这些情况进入 `unknown` 或继续跟踪原 generationId，禁止自动切账号重复生成。仅当创建请求可能已经发送、HTTP 500 响应没有 generationId 且无法证明上游未受理时，才按 `submission_unknown` 处理；能够证明请求体发送前失败的按未提交处理。

## 15.3 允许切账号

只有在能确认任务**未被上游受理**时：

- 请求体尚未开始发送且能可靠证明请求未到达上游的连接建立失败；
- 明确 401/403；
- 明确余额不足；
- 明确 queue full/rate limit 且响应说明未创建任务；

即便切账号，也要记录一次 failover 事件和原账号错误。

---

# 16. 计费、价格快照与资金安全

## 16.1 已有 Leonardo 定价计算器资产

腾哥已经实现一套独立静态计算器：

```text
tools/leonardo-pricing-calculator/
├── index.html
├── pricing-data.js
└── README.md
```

数据说明：

- 采集时间：`2026-08-01`；
- 来源：Leonardo 已登录 Pricing Calculator 页面；
- 图片模型：32 个；
- 视频模型：31 个；
- 合计：63 个；
- 价格单位：USD / generation；
- 图片维度：模型、质量、尺寸档位；
- 视频维度：模型、时长、分辨率；
- 部分模型使用最短/最长时长端点做线性按秒估算；
- 页面支持 USD/CNY 展示换算，默认汇率 `1 USD = 7.19 CNY`。

本次审查已对 `pricing-data.js` 做结构完整性检查：

```text
images = 32
videos = 31
total  = 63
配置数组长度问题 = 0
非法/负数价格 = 0
```

该计算器应纳入 RelayQ 的 Leonardo 定价事实源之一，用于：

1. 客户提交请求前的成本估算；
2. 后台设置销售价时的上游成本参考；
3. 余额预检和额度预留；
4. 模型毛利测算；
5. 当 Leonardo 创建响应没有返回实际 `cost` 时的保守计费基线；
6. 检查上游实际扣费与价格快照的偏差。

但服务端**不得在运行时直接加载或执行前端 `pricing-data.js`**。该文件是静态网页数据源，不是后端计费 API。Trae AI 应把同一份价格数据转换为版本化、可校验的服务端价格目录，或者建立单一 JSON/Go 数据源，再由脚本生成网页 JS，避免前后端两份价格逐渐漂移。

## 16.2 服务端价格目录设计

推荐建立规范化价格快照，例如：

```text
backend/internal/pricing/leonardo/pricing-2026-08-01.json
```

也可存入数据库价格表，但必须保留不可变版本号。建议结构：

```json
{
  "provider": "leonardo",
  "version": "2026-08-01",
  "currency": "USD",
  "source": "leonardo_authenticated_pricing_calculator",
  "captured_at": "2026-08-01T00:00:00+08:00",
  "models": [
    {
      "display_name": "GPT Image 2",
      "model_slug": "gpt-image-2",
      "modality": "image",
      "pricing_mode": "matrix",
      "dimensions": {
        "quality": "Medium",
        "size": "Small 1376×768"
      },
      "cost_usd_per_generation": 0.0987
    }
  ]
}
```

必须显式区分：

```text
display_name       计算器展示名，例如 GPT Image 2
model_slug         API 请求 slug，例如 gpt-image-2
provider_model_id  Leonardo 模型目录内部 ID（若有）
```

禁止只靠展示名参与生产计费。展示名与 API slug 的映射必须由真实 `/v2/models` 和经过验证的映射表确认。

### 图片价格键

推荐规范化键：

```text
provider + model_slug + modality=image + quality + width + height
```

如果某模型比例不影响价格，可以把比例作为验证维度而不是价格键；但必须先验证计算器当前注释“同一质量/尺寸档位下比例同价”对该模型成立。

### 视频价格键

离散时长模型：

```text
provider + model_slug + modality=video + duration_seconds + resolution + audio_flag（若影响价格）
```

连续滑块模型：

```text
min_duration
max_duration
min_cost_usd
max_cost_usd
interpolation = linear
resolution
```

线性公式与当前计算器一致：

```text
cost = minCost + (duration - minDuration)
                 × (maxCost - minCost)
                 / (maxDuration - minDuration)
```

要求：

- 时长必须在 `[minDuration, maxDuration]` 内；
- 金额使用 decimal，不使用二进制 float 做最终扣费；
- 中间计算保留足够精度；
- 最终客户价格按统一货币精度规则舍入；
- `null` quality/size/duration/resolution 是“该维度不适用/使用默认档位”，不能误当字符串或数字 0；
- `Motion 2.0` 等默认时长模型在真实 API 规格未确认前只能做默认配置报价，不能接受任意时长后仍收同价。

## 16.3 价格解析接口

建议服务端提供内部 Resolver：

```go
type LeonardoPriceQuery struct {
    ModelSlug      string
    Modality       string
    Quantity       int
    Quality        string
    Width          int
    Height         int
    Duration       int
    Resolution     string
    AspectRatio    string
    Audio          *bool
}

type LeonardoPriceEstimate struct {
    UnitCostUSD       decimal.Decimal
    Quantity          int
    EstimatedCostUSD  decimal.Decimal
    PricingVersion    string
    PricingSource     string
    MatchType         string // exact | interpolated | default
}
```

核心方法：

```go
EstimateLeonardoCost(query LeonardoPriceQuery) (LeonardoPriceEstimate, error)
```

行为要求：

- 精确配置优先；
- 连续时长仅在标记 `linear` 时插值；
- 未找到模型或规格时返回 `pricing_not_found`；
- 禁止自动选择“最接近但更便宜”的规格；
- 如允许安全回退，应选择不低于目标规格的保守价格并明确 `match_type`；
- quantity 必须大于 0；
- 估价结果必须携带快照版本。

## 16.4 预估成本、实际成本与销售价

每个任务至少记录：

```text
estimated_upstream_cost_amount
estimated_upstream_cost_unit
pricing_snapshot_version
pricing_match_type
actual_upstream_cost_amount
actual_upstream_cost_unit
customer_cost
gross_margin
cost_variance
```

语义：

- `estimated_upstream_cost`：创建前由本地 Leonardo 价格快照计算；
- `actual_upstream_cost`：Leonardo 创建响应 `cost` 或经官方账单确认的真实成本；
- `customer_cost`：RelayQ 最终向客户收取的金额；
- `cost_variance`：实际成本与快照估价的差异。

本次失败创建探针的 `estimated_upstream_cost` 约为 USD 0.003，仅是提交前估价，不得复制到实际成本字段或作为上游已扣费证据。提交结果为 `unknown` 时，实际成本字段必须为 `null`，不得以预估成本、客户报价或 0 代替，`cost_variance` 不计算，计费状态为 `manual_review`；只有获得 Leonardo 明确成本或官方账单证据后才能回填。

客户价格优先级：

1. 后台配置的模型/规格固定销售价；
2. 本地价格快照估价 × 加价倍率；
3. Leonardo 返回的实际成本 × 加价倍率（用于最终校正，必须遵守客户报价规则）；
4. 明确配置的保底价；
5. 无法定价则创建前拒绝。

禁止价格未知时按 0 扣费。

**不能在任务完成后无上限追扣客户。** 如果实际成本高于提交前报价，产品必须选择并固定一种规则：

- 推荐：按提交前锁定销售价向客户收费，差异计入毛利；
- 或：提交前展示估价区间并预留上限，最终按实际值结算；
- 不推荐：任务完成后直接按未知实际成本追扣。

## 16.5 汇率与客户币种

静态计算器中的 `7.19` 仅是默认展示汇率，不应硬编码为生产财务汇率。

生产规则：

- 上游成本原始保存 USD；
- 客户账户若仍以 USD 计费，不需要先转 CNY；
- 如产品使用人民币展示/结算，使用 RelayQ 统一汇率配置；
- 每次报价记录 `fx_rate`、`fx_source`、`fx_timestamp`；
- 修改当前汇率不能重算历史账单；
- 页面计算器可以继续允许手动修改汇率，仅作为测算工具。

## 16.6 不同计价维度

需要支持：

- 图片：模型、分辨率、数量、质量；
- 视频：模型、时长、分辨率、是否音频、数量；
- 音频：模型、时长/分钟、数量；
- 3D：模型、质量、mesh mode；
- upscale：模型、输入/输出规格。

当前计算器只覆盖图片和视频。音频、3D、Upscale 在没有独立价格快照或官方实际成本前，不得沿用图片/视频价格。

第一阶段图片可以继续复用现有图片计费流程，但 Leonardo 模型必须先通过专用 Price Resolver 获取配置成本，不能假定所有 2K 图片成本一致。

## 16.7 价格快照更新与漂移检测

Leonardo 可能随时调价或上下架模型。更新流程：

1. 登录官方 Pricing Calculator 重新采集；
2. 生成新版本价格快照，不覆盖旧版本；
3. 运行结构校验和单元测试；
4. 输出 diff：新增模型、下线模型、配置变化、涨跌幅；
5. 对异常涨幅要求管理员确认；
6. 更新 `pricing-data.js` 或由统一数据源重新生成；
7. 灰度启用新版本；
8. 历史任务仍引用旧版本。

当实际 `cost` 可用时，任务完成后计算：

```text
variance_rate = (actual - estimated) / estimated
```

超过配置阈值时：

- 记录告警；
- 标记价格快照可能过期；
- 可暂停该模型新任务，避免持续负毛利；
- 不自动改写价格快照。

## 16.8 计费状态

```text
unpriced
estimated
reserved
submitted
settled
refunded
manual_review
```

## 16.9 推荐流程

1. 规范化模型 slug 与请求规格；
2. 使用版本化价格快照计算上游预估成本；
3. 根据销售价规则计算并锁定客户报价；
4. 检查余额并预留客户额度；
5. 创建上游任务；
6. 记录 Leonardo 返回的 `cost/apiCreditCost`，保留单位；
7. 任务完成后按已锁定规则结算；
8. 比较估价与实际成本，记录 variance；
9. 明确失败按上游实际收费决定退款；
10. unknown 进入 manual review，不自动全额退款并重复提交。

若第一版暂不实现完整 reservation，最低要求：

- 创建前必须由价格快照得到有效价格；
- 创建前余额检查；
- 创建成功后幂等扣费；
- `billing_reference` 唯一；
- Webhook/Poll 不能重复扣；
- 失败退款幂等；
- unknown 不重复扣也不自动重新创建；
- 任务记录使用的价格快照版本。

## 16.10 用量日志

`usage_logs` 增加或通过 Extra 记录：

```text
generation_job_id
modality
estimated_upstream_cost
actual_upstream_cost
upstream_cost_unit
pricing_snapshot_version
pricing_match_type
cost_variance
billing_status
```

是否增加正式列由现有查询和报表需求决定；频繁筛选/聚合字段不应长期埋在 JSON。

---

# 17. 错误标准化

| Leonardo 情况 | RelayQ HTTP | RelayQ code | 是否切账号 |
|---|---:|---|---|
| API Key 无效 | 401/502（对客户隐藏上游认证细节） | `upstream_authentication_error` | 是 |
| 模型不支持 | 400 | `model_not_supported` | 否 |
| 参数/尺寸错误 | 400 | `invalid_request_error` | 否 |
| 内容审核拒绝 | 400/403 | `content_policy_violation` | 否 |
| 余额不足 | 503/402 | `upstream_quota_exhausted` | 是 |
| API Rate Limit | 429 | `rate_limit_error` | 受控 |
| Queue 满 | 429/503 | `provider_queue_full` | 受控 |
| 服务错误 | 502 | `upstream_error` | 仅确认未受理时 |
| 查询超时 | 504 | `generation_timeout` | 否 |
| 创建请求可能已发送且响应无 generationId、无法证明未受理（含 HTTP 500） | 202 | `submission_unknown` | 否 |

对客户返回的信息不能暴露：

- Leonardo Key；
- 内部账号 ID/名称；
- 上游代理；
- 内部 Base URL 拓扑；
- 原始 Webhook Secret。

---

# 18. 内容安全与隐私

## 18.1 三层审核

```text
RelayQ 前置 Prompt/输入图审核
        ↓
Leonardo 上游审核
        ↓
输出 nsfw 标记与 RelayQ 输出审核
```

## 18.2 输出 NSFW

若结果 `nsfw=true`：

- 不直接返回媒体；
- 任务标记 `failed` 或专门 policy 状态（若扩展状态需全局统一）；
- 记录上游是否收费；
- 不把敏感媒体 URL写普通日志；
- 管理后台只显示必要审核元数据。

## 18.3 `public`

所有 RelayQ 默认请求必须：

```json
"public": false
```

只有明确产品功能、管理员设置和客户显式授权同时成立时才允许 public=true。

## 18.4 日志

默认不记录完整：

- Prompt（可记录 hash/长度）；
- 上传原图；
- 输出敏感 URL；
- 原始 Webhook；
- Credentials；
- Presigned fields。

排障日志需带：

```text
relayq_job_id
provider
model
account_id（仅内部）
upstream_generation_id（仅内部）
status_code
safe_error_code
latency
```

---

# 19. 前端后台改造

## 19.1 账号编辑

重点文件（以实际代码为准）：

```text
frontend/src/components/account/EditAccountModal.vue
frontend/src/components/account/AccountTestModal.vue
frontend/src/components/account/ModelWhitelistSelector.vue
frontend/src/api/admin/accounts.ts
frontend/src/types/*
```

需要：

- 平台 `leonardo`；
- 类型仅 `apikey`；
- API Key 输入和已配置状态；
- Base URL；
- proxy；
- concurrency/load factor；
- active/pending/rpm 可选配置；
- 模型同步；
- 模型白名单；
- 支持模态；
- Webhook 配置状态；
- 连接测试。

## 19.2 模型管理

模型显示：

- Display name；
- slug；
- modality；
- active/deprecated；
- last synced；
- Schema changed 提示；
- 是否向当前分组开放。

## 19.3 Playground

第一阶段只实现常用字段：

- prompt；
- model；
- width/height 或比例；
- quantity；
- quality；
- 参考图；
- prompt enhance；
- style ids。

高级参数提供受控 JSON 编辑框：

```json
{
  "provider_options": {
    "leonardo": {}
  }
}
```

禁止第一阶段实现万能复杂 Schema UI。

## 19.4 异步任务 UI

需显示：

- RelayQ task id；
- 状态；
- 模型/模态；
- 创建/完成时间；
- 输出预览；
- 安全错误；
- 管理员可见账号和上游任务 ID；
- 上游成本/客户价格（仅管理员）；
- 重新查询，不提供盲目“重新生成”按钮。

---

# 20. 后端文件落点建议

以下是建议，不要求机械照搬；Trae AI 修改前需根据真实依赖注入和调用链确认。

## 20.1 平台与账号

```text
backend/internal/domain/constants.go
backend/internal/service/domain_constants.go
backend/internal/service/account.go
backend/internal/service/account_service.go
backend/internal/service/account_test_service.go
backend/internal/service/account_credentials_redact.go
```

## 20.2 Leonardo Client

```text
backend/internal/pkg/leonardo/client.go
backend/internal/pkg/leonardo/types.go
backend/internal/pkg/leonardo/models.go
backend/internal/pkg/leonardo/generations.go
backend/internal/pkg/leonardo/upload.go
backend/internal/pkg/leonardo/errors.go
```

保持文件数量克制；如果一个 `client.go + types.go + client_test.go` 已足够，优先少文件。

## 20.3 模型同步

```text
backend/internal/service/upstream_models.go
backend/internal/handler/admin/account_handler.go
backend/internal/server/routes/admin.go
backend/ent/schema/provider_model_catalog.go
backend/internal/repository/provider_model_catalog_repo.go
```

## 20.4 任务中心

```text
backend/ent/schema/generation_job.go
backend/internal/repository/generation_job_repo.go
backend/internal/service/media_generation.go
backend/internal/service/media_reconciler.go
backend/internal/handler/media_generations.go
backend/internal/server/routes/gateway.go
```

## 20.5 兼容桥

```text
backend/internal/handler/openai_images.go
backend/internal/service/openai_images.go
backend/internal/handler/openai_videos.go
backend/internal/service/openai_videos.go
```

## 20.6 计费和日志

```text
backend/internal/service/billing_service.go
backend/internal/service/usage_billing.go
backend/internal/service/model_pricing_resolver.go
backend/ent/schema/usage_log.go
```

## 20.7 Webhook

```text
backend/internal/handler/leonardo_webhook.go
backend/internal/service/leonardo_webhook.go
backend/internal/server/routes/*
```

---

# 21. 分阶段实施任务

## Phase 0：真实协议探针

### LEO-000 基线隔离

- 检查 `git status`；
- 确认主分支现场改动归属；
- 选择 Trae AI 专用分支/worktree；
- 不覆盖小腾或其他开发中的文件；
- 记录基线 commit。

验收：

- 独立分支；
- 工作区修改归属清楚；
- 可随时 diff/revert。

### LEO-001 Production Key 连接验证

已使用 Production Key 脱敏实测 `GET /v2/models`：有效 Key 可访问目录，无效 Key 的状态码、响应头和脱敏错误结构已记录；返回 69 个模型，每个模型包含 `id`、`name`、`parameters`，`id` 为 UUID 且没有独立 slug 字段。本任务未执行创建请求或生成状态查询，不能证明 UUID 与 request slug 的映射关系。

```http
GET /v2/models
```

记录：

- 状态码；
- 响应头；
- 响应结构；
- 模型 ID/slug 字段；
- 限流头；
- 错误结构。

Key 不得写入代码、Markdown、日志或测试 fixture。

### LEO-002 图片创建与查询

历史探针已发送且仅发送一次 `POST /v2/generations`：请求使用 `flux-schnell`，响应为 HTTP 500，响应体无 `generationId`、`cost` 和 `apiCreditCost`；受理、任务创建及实际扣费均未知，归类为 `submission_unknown` / `side_effect_unknown`。因无 `generationId`，状态 GET 实际执行次数为 0，查询 URL、状态枚举、结果字段、NSFW 字段和 CDN 行为均未验证。本次请求不得重试；如需再次发送付费创建请求，必须创建新的、独立审批的探针任务，不得视为本次请求的重试、续跑或重放。

### LEO-003 上传与图生图

验证：

- `/v1/init-image`；
- fields 类型；
- 204；
- uploaded ID；
- `guidances.image_reference`；
- UPLOADED/GENERATED 区别。

### LEO-004 Webhook

验证：

- Header；
- 事件类型；
- v2 模型回调结构；
- 图片/视频字段；
- 重试行为；
- 非 2xx 重发间隔（若可观察）。

### LEO-005 错误探针

安全地验证：

- 无效 Key；
- 无效模型；
- 无效尺寸；
- 内容审核；
- 余额不足（若当前账号可模拟）；
- Rate Limit 不主动压测生产账号，仅从文档/响应头确认。

Phase 0 输出：

```text
docs/leonardo-api-probe-results.md
```

该文件必须脱敏。

---

## Phase 1：Provider 与模型同步

### LEO-100 平台常量与账号方法

- `PlatformLeonardo`；
- `IsLeonardo()`；
- Production API Key 获取；
- Base URL 默认；
- 支持 apikey；
- 账号验证；
- 凭证脱敏。

### LEO-101 Leonardo Client

实现：

- models；
- create generation；
- get generation；
- init upload；
- presigned upload；
- 错误解析；
- proxy 和 timeout。

全部使用 `httptest.Server` Mock 测试。

### LEO-102 上游模型同步

接入现有 sync preview/sync route：

- Leonardo request builder；
- 特殊响应解析；
- 模型目录 upsert；
- 支持模型白名单；
- 同步失败保留旧模型。

### LEO-103 账号测试

默认只测试 `/v2/models`，不产生费用。

### LEO-104 定价快照导入与服务端 Price Resolver

以现有工具为输入资产：

```text
tools/leonardo-pricing-calculator/pricing-data.js
```

完成：

- 将 32 个图片模型、31 个视频模型转换为版本化服务端价格目录；
- 建立 display name → request slug 显式映射；
- 实现图片矩阵价格精确匹配；
- 实现视频离散时长价格和受控线性插值；
- 使用 decimal 计算；
- 返回 snapshot version/source/match type；
- 价格缺失明确拒绝；
- 提供结构校验、边界和代表模型单元测试；
- 设计单一数据源或生成脚本，避免网页 JS 与服务端价格双份手工维护。

### LEO-105 价格漂移与快照更新工具

完成：

- 新旧快照 diff；
- 模型新增/下线；
- 配置新增/删除；
- 价格涨跌幅；
- 异常变化确认门禁；
- 保留历史版本；
- 实际 cost 与估价 variance 告警。

Phase 1 验收：

- 能在后台创建 Leonardo 账号；
- API Key 不回显；
- 连接测试成功；
- 模型同步成功；
- 63 个现有图片/视频模型价格数据通过结构校验；
- 代表性图片矩阵、离散视频和滑块视频估价与静态计算器一致；
- 现有 OpenAI/xAI/Gemini 测试不回归。

---

## Phase 2：持久化任务中心

### LEO-200 Ent Schema 与迁移

新增：

- `generation_jobs`；
- 必要索引；
- 外键/软删除策略；
- 迁移回滚说明。

### LEO-201 Repository

实现：

- create；
- find by public id；
- bind upstream id；
- conditional status transition；
- due jobs scan；
- idempotent billing mark；
- append/update outputs。

### LEO-202 Service 状态机

- 创建本地任务；
- 调度账号；
- 提交 Leonardo；
- 处理明确失败/unknown；
- 状态更新；
- 终态释放。

### LEO-203 Poller/Reconciler

- 分批扫描；
- 指数退避；
- 重启恢复；
- Webhook/Poll 竞争安全；
- graceful shutdown。

Phase 2 验收：

- 服务重启后任务不丢；
- 任务状态可恢复；
- 同一终态只落一次；
- unknown 不重复创建。

---

## Phase 3：图片生成 MVP

### LEO-300 原生 Media 图片接口

模型首批建议：

- `gpt-image-2`；
- `nano-banana-2`；
- `seedream-4.5`。

以真实 `/v2/models` 可用性为准。

### LEO-301 OpenAI Images generations 桥

- 模型目录识别；
- 参数映射；
- Provider 路由；
- 异步等待；
- URL/b64 输出；
- 错误标准化。

### LEO-302 定价、成本校验与客户结算

- 接入 `LEO-104` Leonardo Price Resolver；
- 创建前按模型、质量、尺寸/时长、分辨率和数量估算上游 USD 成本；
- 锁定价格快照版本和客户报价；
- 记录上游 `cost/apiCreditCost` 及其单位；
- 区分 estimated upstream cost、actual upstream cost 和 customer cost；
- 幂等扣费；
- usage log；
- 计算 cost variance 和 gross margin；
- 实际成本明显偏离快照时告警，不静默改价；
- 汇率仅使用 RelayQ 统一配置，不能把计算器默认 `7.19` 硬编码进生产计费。

### LEO-303 前端账号和图片测试

- Leonardo 表单；
- 模型同步；
- 付费图片测试明确提示；
- 图片预览。

Phase 3 验收：

- 三个模型至少各成功一次；
- OpenAI 兼容调用成功；
- 原生任务查询成功；
- 不重复扣费；
- 输出 NSFW 被阻止；
- 其他图片渠道不回归。

---

## Phase 4：图片编辑与参考图

### LEO-400 安全上传链路

- multipart；
- data URI；
- URL；
- SSRF；
- presigned upload；
- hash cache。

### LEO-401 多参考图

- 模型最大数量；
- strength；
- GENERATED/UPLOADED；
- 0x0 参考图比例行为（仅模型支持时）。

### LEO-402 `/v1/images/edits`

- 兼容 OpenAI multipart；
- mask 显式能力；
- 不支持参数明确报错。

Phase 4 验收：

- 单图编辑；
- 多图参考；
- URL SSRF 拒绝；
- 超大文件拒绝；
- Presigned 上传不携带 Authorization。

---

## Phase 5：Webhook 生产化

### LEO-500 Webhook 安全入口

- route token；
- Bearer Secret；
- 常量时间比较；
- body limit；
- 幂等事件。

### LEO-501 Webhook 状态更新

- 图片；
- 视频；
- 通用 raw fallback；
- 输出脱敏；
- 计费幂等。

### LEO-502 轮询兜底

- Webhook 丢失后补偿；
- Webhook 与 Poll race；
- 重启恢复。

Phase 5 验收：

- 重复回调不重复扣费；
- 错 Secret 拒绝；
- 回调丢失仍能靠 Poll 完成；
- 敏感 apiKey 对象不入日志/结果。

---

## Phase 6：视频

### LEO-600 通用视频 Provider 路由

改造现有视频入口，不再 Handler 强制 xAI。

### LEO-601 首批视频模型

建议：

- `grok-imagine-1.5`；
- `veo-3.1-generate-001`；
- 一个 Wan/Kling/Seedance 模型。

以 Production API 实际可用模型为准。

### LEO-602 视频参数

- duration；
- width/height；
- start frame；
- end frame；
- motion audio；
- quantity；
- 模型 Schema 校验。

### LEO-603 内容代理

- Range；
- Content-Type；
- Content-Length；
- 上游 URL 隐藏策略；
- 下载权限。

### LEO-604 视频计费

- 时长；
- 分辨率；
- 音频；
- 实际 cost；
- 销售价。

Phase 6 验收：

- 文生视频；
- 图生视频；
- 竖屏；
- 状态查询；
- Range 播放；
- 计费一次；
- xAI 视频不回归。

---

## Phase 7：音频、3D、Upscale

### LEO-700 Audio

- Music v1；
- Sound Effects v2；
- Dialogue v3；
- 播放/下载；
- 时长和数量计费。

### LEO-710 3D

- Rodin v2；
- GLB；
- image references；
- mesh mode/quality；
- 文件下载。

### LEO-720 Upscale

- Aurora Precise；
- Aurora Creative；
- 输入图；
- 输出规格；
- 计费。

---

## Phase 8：运营与生产治理

### LEO-800 成本与毛利报表

- 按模型；
- 按账号；
- 按用户；
- 按模态；
- 上游成本/客户收入/毛利。

### LEO-801 账号监控

- 无效 Key；
- 余额不足；
- 队列满；
- active/pending；
- 成功率；
- P50/P95 时延。

### LEO-802 模型弃用

- 自动同步；
- Schema diff；
- missing model；
- deprecated UI；
- 管理员提醒；
- 禁止新任务但保留历史查询。

---

## Phase 9：现状核对与生产收口

### 21.1 执行台账规则

本节是后续逐项开发的唯一执行清单。状态只允许使用：

- `已完成`：代码、自动化测试和要求的真实行为证据全部满足；
- `部分完成`：已有代码主体，但仍缺测试、真实协议或产品闭环；
- `阻塞`：继续开放会造成错误计费、重复生成、安全问题或明确不可用；
- `未开始`：尚无可验收实现。

每项任务完成时必须同步：状态、修改文件、测试命令、原始结果、真实 API 证据、剩余风险和最后更新日期。真实付费请求必须使用新任务和独立幂等键；任何 `submission_unknown` 历史请求都不得重放。

### 21.2 当前能力基线

| 能力 | 状态 | 当前事实 | 剩余工作 |
|---|---|---|---|
| 独立 Provider 与账号 | 部分完成 | Leonardo 已独立于 OpenAI 和 LeoStudio，支持 API Key/Base URL | 补生产密钥轮换和多账号容量策略验收 |
| Production API Client | 部分完成 | 已封装 `/v2/models`、`/v2/generations`、`/v1/generations/{id}`、`/v1/init-image` | 补真实成功响应、上传、错误和限流证据 |
| 异步任务中心 | 部分完成 | 已有 generation jobs、状态机、Poll、Webhook、unknown/manual review | 补重启恢复、并发竞争和故障注入验收 |
| OpenAI Images 门面 | 部分完成 | 已实现 OpenAI/Leonardo Raw 自动识别；OpenAI 默认同步、900 秒上限、默认 low/b64_json，并支持缺失幂等键时自动生成 | 补实际 ComfyUI/Infinite Canvas E2E、Upscale 和稳定错误码验收 |
| 原生 Media API | 部分完成 | 已有创建、按 ID 查询和 content | 缺列表、分页、管理查询和多模态创建 |
| 图片价格计算器 | 阻塞 | FLUX Schnell 已有 896/1024/1120 精确价与当前档位最高价基础，完整模型和 Upscale 价格仍缺失 | 导入完整服务端价格快照和版本化 Resolver |
| 视频价格计算器 | 阻塞 | 已固化 low/medium/high 档位最高价池，但未验证模型、时长、分辨率和音频精确价 | 完成首个视频模型真实价格与任务闭环 |
| 图片生成 | 部分完成 | 单账号单模型代码闭环存在 | 缺真实成功付费 E2E |
| 图片编辑/参考图 | 部分完成 | 已有安全上传、缓存和 multipart Handler | 缺真实 init-image 与生成验收 |
| 多账号调度 | 阻塞 | 两个以上可用账号时创建返回 503 | 实现容量感知选择、并发槽位和 RPM 门禁 |
| 视频 | 阻塞 | 路由可挂载，但 Handler、模型和定价不具备可用闭环 | 保持 Flag 关闭，完成一个模型 E2E 后再开放 |
| 音频、3D、Upscale | 未开始 | 仅有 fail-closed 定价桩或规格占位 | 按模态逐个完成模型、价格、任务和内容闭环 |
| 管理与可观测性 | 部分完成 | 有 manual review API、worker heartbeat 和成本偏差基础 | 补任务 UI、完整指标、报表和告警 |
| 生产发布 | 阻塞 | 本地构建可启动 | 缺完整 CI、迁移演练、回滚和真实 E2E 证据 |

### LEO-900 计划与代码基线对账

**状态：已完成**

- 以当前 `realyq-leonardo-prod` 工作区为事实源更新本文；
- 明确 OpenAI 兼容入口与 Leonardo 上游异步协议的边界；
- 明确当前已实现主体、阻断项和安全开放边界；
- 后续不得把“代码存在”直接标记为“生产完成”。

验收：本文 0.1、21.1、21.2 与当前分支、路由、Client、价格 Resolver 和任务状态一致。

### LEO-901 Production API 真实协议补探针

**状态：阻塞；第一优先任务**

当前进展（2026-08-05）：已使用隔离数据库中的唯一 Leonardo 账号执行一次只读 `GET /v2/models`，上游返回 `Invalid response from authorization hook`（`path=$`、`code=unexpected`）。本次创建 POST 为 0，未产生付费副作用。当前账号未通过只读鉴权前置门禁，在人工确认 Key、Production API 权限、PAYG/余额和上游服务状态前禁止执行付费创建。

- 使用新幂等任务完成一次最小图片创建，确认 `generationId`；
- 使用真实任务 ID 验证 `/v1/generations/{id}` 的状态、输出、NSFW 和成本字段；
- 验证 `/v1/init-image`、预签名字段、S3 204 和参考图参数；
- 配置测试 Webhook，保存脱敏 payload，验证鉴权、重复投递和 Poll 竞争；
- 记录 400、401、402/余额不足、429、500、超时和断连的脱敏响应；
- 对账 Leonardo 响应成本、控制台账单和 RelayQ 任务成本，Credits 与 USD 不得混算；
- 历史 LEO-002/002C unknown 任务只允许人工核对，不得重放。

验收：生成、查询、输出、成本、计费和退款形成一条可审计证据链，所有原始材料脱敏保存。

### LEO-902 Verified Model Registry 扩充

**状态：部分完成**

当前进展（2026-08-06）：Registry 已登记 FLUX Schnell、GPT Image 2、Nano Banana 2、Nano Banana 2 Lite。三个新增模型的 UUID 来自同日只读 `GET /api/rest/v2/models`：GPT Image 2=`135b2740-a20b-48c8-8f86-6f68199e06c5`、Nano Banana 2=`7418e71f-4133-4e1b-9895-bee19f48f2ce`、Nano Banana 2 Lite=`21278dfe-ac26-4292-82e0-8e588373a30c`。Registry 已记录数量上限、质量能力、`image_reference` 最大 6 张、允许来源类型和 strength 差异；返回值使用深拷贝。

- 当前创建白名单包含 `flux-schnell`、`gpt-image-2`、`nano-banana-2`、`nano-banana-2-lite`；新增三模型已完成目录验证，真实付费创建仍需逐模型确认后执行；
- 为每个开放模型分别记录 display name、provider model UUID、request model slug、modality；
- 首批图片目标：GPT Image 2、Nano Banana 2、Nano Banana 2 Lite 和当前 FLUX Schnell；
- 首批视频目标：只选择一个价格和创建协议均已验证的模型；
- 未通过真实创建验证的模型不得加入可创建白名单；
- 模型展示名、目录 UUID 和创建 slug 不得互相代替。
- Registry 必须逐模型记录官方尺寸约束、输出数量、`quality` 或 `mode`、Guidance 类型、参考图数量、strength 枚举和允许的图片来源类型；
- 不得把一种通用参考图结构强行用于全部模型：FLUX Dev/Schnell 使用 `content/style`，Phoenix 使用 `image_to_image/content/character/style`，Kontext、FLUX.2、GPT、Nano Banana、Seedream 使用 `image_reference`；
- GPT Image 2 的图片参考不得发送 `strength`；Ideogram、P-Image-Ideogram、Krea 当前官方指南未声明图片参考能力，保持关闭。

验收：每个 registry 项都有模型目录证据、真实创建证据和单元测试。

### LEO-903 Provider Model Catalog 持久化

**状态：未开始**

- 新增 Provider Model Catalog Schema、Repository 和规范迁移；
- 保存 provider、provider model ID、request slug、display name、modality、parameter schema、schema hash、last seen、missing 和 deprecated；
- `/v2/models` 同步使用 upsert，单次同步失败保留上次可用目录；
- 发现 Schema 变化、模型消失或重新出现时记录可审计事件；
- 动态目录只表示“上游存在”，是否可创建仍由 Verified Model Registry 和价格证据共同决定。

验收：同步幂等、失败保留旧值、missing/deprecated 转换和 Schema diff 测试通过。

### LEO-904 图片与视频价格快照导入

**状态：部分完成**

当前进展（2026-08-06）：已将 GPT Image 2 的 3×3 价格矩阵、Nano Banana 2 的 Small/Medium 价格和 Nano Banana 2 Lite 的单价导入服务端 decimal Resolver；数量按单价相乘，客户报价继续固定为本地成本 ×7.1。Nano Banana 2 的 2880 产品尺寸及 Lite 的 2048/2880 因非原生规格且尚未接入 Upscale，保持 fail closed。

- 将本地 Leonardo 定价计算器数据转换为服务端只读、版本化价格快照；
- 图片覆盖计划要求的 32 个模型，视频覆盖 31 个模型；
- 保存 snapshot version、source、currency、captured at、model slug 和精确匹配维度；
- 图片 Resolver 支持模型、质量、尺寸、数量、公开属性及模型特有参数；
- 所有客户图片价格统一为“本地精确成本快照 × 7.1”，参考图、Upscale、质量档位等多阶段请求必须先将各阶段成本求和，再乘 7.1；
- 对外尺寸档位固定为 `small=1024×1024`、`medium=2048×2048`、`large=2880×2880`，价格快照必须区分原生生成与生成后 Upscale；
- 对外质量名称统一使用 `low/medium/high`，但只允许映射到模型官方真实存在的 `quality` 或 `mode`；无官方质量参数的模型只开放 `default`，不得制造三个无差别的虚假档位；
- 视频 Resolver 支持模型、时长、分辨率、数量、音频及离散/滑块规则；
- 所有金额使用 decimal，禁止 float；
- 缺模型、缺组合、缺币种或歧义映射时 fail closed，禁止静默免费和猜价。

验收：32 图片、31 视频、总计 63 个模型结构校验和代表性价格回归全部通过。

### LEO-905 价格快照 Diff 与漂移门禁

**状态：未开始**

- 实现新旧快照模型、组合、币种和价格差异报告；
- 异常涨跌、模型数骤变、slug 变化必须人工确认后才能替换当前版本；
- 历史快照不可覆盖，generation job 必须固定引用创建时版本；
- 实际上游成本与估价偏差超过阈值时产生指标和告警；
- 新价格只影响新任务，历史任务、退款和毛利不得重算。

验收：新增、删除、改价、币种变化、异常门禁和历史版本回放测试通过。

### LEO-906 多账号容量调度

**状态：阻塞**

- 删除“多个有效账号直接 503”的临时限制；
- 按模型白名单、账号健康、RPM、active jobs、pending jobs 和并发容量选择账号；
- 槽位占用和释放必须原子化，提交 unknown 不能错误释放并转投另一账号；
- 429、余额不足、无效 Key、队列满和熔断状态必须影响调度；
- 不允许因故障转移重复创建或重复计费。

验收：多账号选择、并发争抢、容量耗尽、恢复和 unknown 不重投测试通过。

### LEO-907 原生 Media 列表与管理查询

**状态：部分完成**

- 补 `GET /v1/media/generations`；
- 支持分页、状态、模态、模型和时间过滤；
- 客户查询必须限制任务归属，管理员查询可按用户和账号过滤；
- 响应不得包含 API Key、Webhook Secret、预签名字段和原始敏感错误；
- content 接口继续使用安全代理或受控跳转，并支持适用的 Range 请求。

验收：归属隔离、分页稳定性、过滤、敏感字段和 content 测试通过。

### LEO-908 图片生成真实 E2E 与收费闭环

**状态：进行中**

当前进展（2026-08-06）：`/v1/images/generations` 已按请求结构区分 OpenAI 兼容和 Leonardo Raw；OpenAI 客户端默认同步等待，最长 900 秒，缺省 `quality=low`、`size=1024x1024`、`response_format=b64_json`，未提供幂等键时使用 5 分钟自动幂等窗口。创建后通过 `X-RelayQ-Task-ID` 提供断线找回。尚缺真实 ComfyUI/Infinite Canvas 付费 E2E、Large Upscale 和全价格快照，因此仍不可标记完成。

- 先限定单账号、`flux-schnell`、896×896、单图、private；
- 当前固定上游成本按本地价格快照 `$0.003`，客户扣费固定为 `$0.003 × 7.1 = $0.0213`；
- OpenAI Images 与原生 Media API 必须由服务端统一计算 `$0.0213`，不得接受客户端自定义最终扣费；
- 验证 OpenAI Images 门面和原生 Media API 都进入同一任务中心；
- 使用真实 ComfyUI OpenAI 节点和 Infinite Canvas 或等价客户端验证标准同步响应；
- 验证 900 秒超时、客户端断线、自动幂等和 task ID 找回，不得因客户端重试重复生成；
- 验证预估、资金预留、上游创建、Poll/Webhook、终态结算、Usage 和毛利；
- 验证失败退款、NSFW、unknown/manual review 和重复请求；
- 验证 queued、running、unknown、terminal billing-pending 状态重启恢复；
- 通过后才允许内部测试组小流量 Flag 灰度。

验收：数据库、API 响应、上游账单和客户余额四方一致，不重复生成、不重复扣费。

### LEO-909 图片编辑与多参考图真实 E2E

**状态：部分完成**

当前进展（2026-08-06）：Leonardo Raw JSON 已支持 FLUX Schnell 的 `parameters.guidances.content/style`；无代上传时 Body 原样提交，`image.source` 支持 Data URI、受控 URL 和 `multipart://字段名`，上传后只定点写入 `id/type=UPLOADED`。Content/Style 语义进入幂等指纹并禁止和 legacy generic guidance 混用。OpenAI Images Edits 不猜测 Content/Style；高级能力由 Raw API 或专用 ComfyUI 节点提供。历史图片归属校验、参考图精确价格和真实付费 E2E 尚未完成，因此暂不对生产客户开放。

- 实测 init-image 和预签名上传；
- 验证本地 multipart、受控远程 URL、文件类型、20 MB 上限和 SSRF 防护；
- 验证上传 hash cache 不跨账号泄漏、不缓存失败结果；
- 验证单参考图和多参考图参数映射；
- 图生图、内容参考、风格参考、角色参考和通用图片参考必须按官方逐模型协议一比一实现，不提供会丢失语义的跨模型通用降级；
- FLUX Dev/Schnell：`content` 最多 1 张，strength 为 LOW/MID/HIGH；`style` 最多 1 张，strength 为 LOW/MID/HIGH/ULTRA/MAX；
- Phoenix：分别支持 `image_to_image`、`content`、`character` 和最多 4 张 `style`；
- Kontext Pro/Max、FLUX.2、GPT、Nano Banana、Seedream：按各自官方 `image_reference` 数量、strength 和图片来源类型实现；
- 原生 Media API 暴露有类型的 Guidance 契约；OpenAI Images Edits 仅映射能够无损表达的图生图能力，无法无损表达的官方功能只能通过原生接口使用；
- RelayQ 专用 ComfyUI Advanced 节点提供 reference type、strength 和 source，标准 OpenAI 节点不隐式获得高级 Guidance；
- 参考图片统一先走 init-image 预签名上传；支持引用历史 Leonardo 生成图时，严格校验任务归属，禁止跨用户引用；
- 临时文件、预签名字段和下载内容不得进入日志。

验收：真实编辑成功，恶意 URL、大文件、伪造 MIME、重复上传和超时测试通过。

### LEO-910 Leonardo 视频端到端闭环

**状态：阻塞**

- `video_enabled` 在本任务完成前必须保持关闭；
- 选择一个 Registry、价格和真实创建均已验证的视频模型；
- 完成创建、状态、内容、Range、超时、失败、NSFW 和计费；
- OpenAI Videos 门面与原生 Media API 复用任务中心；
- 视频编辑和续写若官方协议尚未验证，继续列为非目标，不复用 xAI Handler 假装支持。

验收：至少一个视频模型真实 E2E 通过，Handler 不再固定拒绝 Leonardo 请求。

### LEO-911 音频端到端闭环

**状态：未开始**

- 在 Verified Model Registry 和价格证据齐备后实现 Music、SFX、Dialogue；
- 支持对应参数、状态、播放/下载、内容类型、时长和数量计费；
- `audio_enabled` 在验收完成前保持关闭。

验收：每个对外开放音频类型至少一个真实模型 E2E 通过。

### LEO-912 3D 与 Upscale 闭环

**状态：未开始**

- 3D 完成 Rodin v2、参考图、mesh 参数、GLB 内容和计费；
- Upscale 完成 Aurora Precise/Creative、输入图、输出规格和计费；
- `large=2880×2880` 在模型不能原生生成时，使用官方 Pro Upscaler Precise 组成两阶段任务；优先采用 `1440×1440 × 2`，不得向上游伪报模型原生支持 2880；
- 两阶段任务必须共享一个客户任务视图，分别保存生成和 Upscale 的上游任务 ID、成本、状态与失败原因；
- 任一阶段失败都必须按统一账务状态机结算或退款，不允许只完成第一阶段却按 Large 全额结算；
- 大文件下载、Content-Type、Range 和超时策略必须单独验收。

验收：3D 与 Upscale 各至少一个真实模型 E2E 通过。

### 21.4 图片产品统一规格与官方能力映射

本节是所有 Leonardo 图片模型的强制产品契约。统一产品档位只负责客户体验，实际请求必须服从每个模型的官方协议和能力边界。

#### 21.4.1 尺寸档位

| 客户档位 | 最终交付尺寸 | 实现规则 |
|---|---:|---|
| Small | 1024×1024 | 模型原生支持时直接生成；否则使用官方合法近邻尺寸并通过已验证的官方 Upscale 链路交付严格尺寸 |
| Medium | 2048×2048 | 模型原生支持时直接生成；否则追加官方 Pro Upscaler Precise |
| Large | 2880×2880 | 模型不能原生支持时必须追加官方 Pro Upscaler Precise，默认 `1440×1440 × 2` |

- API 响应必须说明 `requested_size_tier`、`delivered_width/height`、`native_generation` 和是否执行 Upscale；
- 不得把模型最大尺寸当作 2880 返回，不得静默返回非目标尺寸；
- 固定正方形输出需要裁切或补边时必须由明确策略控制，默认禁止破坏输入图主体的隐式裁切；
- 生成与 Upscale 的本地成本相加后乘 7.1，才是客户最终扣费。

#### 21.4.2 质量档位

对外标准名称为 `low`、`medium`、`high`，内部按官方字段映射：

| 官方模型族 | Low | Medium | High |
|---|---|---|---|
| GPT Image-1.5 / GPT Image 2 | `quality=LOW` | `quality=MEDIUM` | `quality=HIGH` |
| Ideogram 3.0 | `quality=TURBO` | `quality=BALANCED` | `quality=QUALITY` |
| P-Image-Ideogram | `quality=LOW` | `quality=MEDIUM` | `quality=HIGH` |
| Phoenix | `mode=FAST` | `mode=QUALITY` | `mode=ULTRA` |
| Lucid Origin / Lucid Realism | `mode=FAST` | 不支持 | `mode=ULTRA` |
| 无官方 `quality/mode` 的模型 | 不支持 | `default` | 不支持 |

- 对不支持的档位返回明确的 `quality_not_supported`，不得把三个档位映射成相同请求；
- API 模型目录必须返回每个模型实际支持的质量档位，前端据此禁用不可选项；
- 官方字段有变化时必须先通过 Registry、价格快照和真实探针再开放。

#### 21.4.3 Guidance 一比一复刻原则

- 对外保留官方语义：`image_to_image`、`content`、`style`、`character`、`image_reference` 不得混为同一字段；
- 每个模型严格执行官方最大参考图数量、strength 枚举、默认值、顺序字段和 `image.type` 白名单；
- `style_ids` 表示 Leonardo 预设风格，不等同于上传风格参考图；两者必须使用不同请求字段；
- OpenAI 兼容接口只承载可无损映射的能力，完整官方能力由 `/v1/media/generations` 原生契约提供；
- 未在官方模型指南声明的能力保持 fail closed；不得因为底层 DTO 能拼装任意 JSON 就对外开放；
- 每种已开放 Guidance 至少保留一个脱敏真实 Production API 成功证据和一个精确 payload 合同测试。

官方能力依据：

- FLUX Schnell：<https://docs.leonardo.ai/docs/flux-schnell>
- Phoenix：<https://docs.leonardo.ai/docs/phoenix>
- GPT Image 2：<https://docs.leonardo.ai/docs/gpt-image-2>
- FLUX.2 Pro：<https://docs.leonardo.ai/docs/flux-2-pro>
- Pro Upscaler Precise：<https://docs.leonardo.ai/docs/pro-upscaler-precise>
- 图片预签名上传：<https://docs.leonardo.ai/docs/how-to-upload-an-image-using-a-presigned-url>

### LEO-913 管理后台任务与人工复核 UI

**状态：部分完成**

- 展示任务状态、模态、模型、账号、用户、上游任务 ID、安全错误、估价、实际成本、收入和毛利；
- 提供 unknown/manual review 查询、绑定上游 ID、确认失败和退款操作；
- 高风险操作要求二次确认并写审计日志；
- UI 和管理 API 不得暴露任何密钥、Secret 或预签名字段。

验收：管理员可以不直接操作数据库完成 unknown 任务处置，并有完整审计记录。

### LEO-914 业务指标、告警与成本毛利报表

**状态：部分完成**

- 增加 submitted、queued、running、completed、failed、unknown、duration、queue age、poll、webhook、active 和 pending 指标；
- 按模型、账号、用户和模态统计上游成本、客户收入和毛利；
- 告警覆盖 invalid key、余额不足、队列满、unknown 增长、worker heartbeat、价格偏差和任务积压；
- 指标标签不得包含 prompt、用户输入或高基数字段。

验收：仪表盘和告警演练能定位提交、队列、回调、结算和价格异常。

### LEO-915 模型弃用与同步告警

**状态：未开始**

- 对 Schema diff、missing、deprecated 和 slug 映射失效产生管理告警；
- missing/deprecated 模型禁止新任务，但保留历史任务查询和内容访问；
- 已有任务不能因模型目录变化丢失解析能力；
- 管理 UI 明确区分“上游存在”“已验证可创建”“有价格可销售”。

验收：模型消失、恢复、Schema 变化和禁用新任务测试通过。

### LEO-916 完整自动化回归与慢测治理

**状态：阻塞**

- 拆分并定位当前 service/handler/routes 联合测试中的慢测或阻塞；
- 完成 Leonardo Client、状态机、计费、上传、Webhook、路由和迁移测试；
- 执行前端 typecheck、lint、Vitest 和 build；
- 执行 OpenAI、xAI、Gemini 现有链路回归；
- 增加模拟 ComfyUI/Infinite Canvas 的无幂等键同步请求、默认 base64、900 秒超时和断线找回回归；
- 测试不得依赖真实付费 API，真实探针单独审批和记录。

验收：候选提交 CI 全绿，无无限等待测试，失败能定位到具体包和用例。

### LEO-917 生产迁移、发布与回滚门禁

**状态：阻塞；最终任务**

- 在生产备份副本执行 153–156 及后续迁移，记录耗时、锁表和索引；
- 验证备份、回滚、配置模板和密钥注入；
- 所有 Leonardo Feature Flag 默认关闭，按图片、视频、音频分开灰度；
- 发布前核对历史 unknown 任务、worker、告警、价格版本和账号容量；
- 先内部组、再小流量、再逐级放量，任何重复生成、重复扣费或 unknown 激增立即回滚；
- 只有第 26 节 DoD 全部有证据后才能宣称完整接入完成。

验收：迁移演练、回滚演练、监控演练和真实 E2E 签字记录齐全。

### 21.3 强制执行顺序

```text
LEO-900 已完成基线对账
  ↓
LEO-901 真实协议补探针
  ↓
LEO-902 Verified Model Registry
  ↓
LEO-903 Provider Model Catalog
  ↓
LEO-904 完整图片/视频价格快照
  ↓
LEO-905 价格 Diff 与漂移门禁
  ↓
LEO-906 多账号容量调度
  ↓
LEO-907 Media 列表与管理查询
  ↓
LEO-908 图片真实 E2E
  ↓
LEO-909 图片编辑真实 E2E
  ↓
LEO-910 视频 E2E
  ↓
LEO-911 音频 → LEO-912 3D/Upscale
  ↓
LEO-913 管理 UI → LEO-914 指标报表 → LEO-915 模型弃用
  ↓
LEO-916 完整回归
  ↓
LEO-917 生产发布门禁
```

除非前置任务的验收证据已经写回本文，不得提前打开对应模态 Feature Flag，也不得并行铺开后续产品页面。

---

# 22. 测试矩阵

## 22.1 Client 单元测试

- Authorization 正确；
- Base URL 拼接；
- `/v2/models` 解析；
- model ID/slug；
- create response cost；
- status response；
- 错误体；
- Retry-After；
- Body size limit；
- Presigned fields string/object；
- 上传无 Authorization；
- 204 success。

## 22.2 状态机测试

- 正常 queued→running→succeeded；
- 直接 succeeded；
- failed；
- submit timeout→unknown；
- 终态不可回退；
- Webhook/Poll 同时完成；
- 重复 Webhook；
- 重启恢复；
- active counter 释放。

## 22.3 定价与计费测试

### 价格快照结构

- 图片模型数 32；
- 视频模型数 31；
- 合计 63；
- 图片 `configs == qualities × sizes`；
- 离散视频 `configs == durations × resolutions`；
- 滑块视频配置数与 resolution 数一致；
- 所有成本为非负 decimal；
- snapshot version/source/currency 必填；
- display name 与 model slug 不混用。

### 代表性价格回归

至少覆盖：

- GPT Image 2：Low/Medium/High × Small/Medium/Large；
- Nano Banana 2：三个尺寸档；
- Lucid Origin：Fast/Ultra；
- Phoenix：Fast/Quality/Ultra；
- Veo 3.1：离散时长 × 分辨率；
- Seedance 1.0 Pro：时长 × 分辨率；
- Grok Imagine 1.5：连续时长线性插值；
- Seedance 2.0：不同 resolution 的滑块插值；
- Motion 2.0：默认时长语义；
- quantity > 1；
- 不支持规格；
- 时长越界；
- 未知模型；
- null 维度。

### 结算测试

- 估价；
- 价格快照版本锁定；
- 余额不足；
- reservation；
- settle；
- refund；
- unknown；
- 重复结算；
- cost unit 不同；
- estimated vs actual variance；
- account/group multiplier；
- 价格缺失拒绝；
- 修改新快照不影响历史任务；
- 汇率变更不重算历史账单。

## 22.4 图片桥测试

- gpt-image-2；
- nano-banana-2；
- seedream；
- n/quantity；
- size；
- quality enum；
- provider options；
- URL/b64；
- edits；
- mask unsupported；
- 多参考图上限。

## 22.5 安全测试

- localhost；
- 127.0.0.1；
- 私网；
- IPv6 loopback；
- DNS rebinding；
- redirect to private；
- 云元数据；
- 超大文件；
- MIME 欺骗；
- Webhook 错 Secret；
- 日志不含 Key。

## 22.6 回归测试

至少覆盖现有：

- OpenAI Images；
- xAI Images；
- xAI Videos；
- 账号测试；
- 模型同步；
- 调度；
- 图片计费；
- 管理后台账号编辑。

## 22.7 构建门禁

每阶段按涉及范围执行：

```text
gofmt
go test 定向包
go test ./internal/service/...（条件允许）
go test ./internal/handler/...（条件允许）
前端 typecheck
前端定向 vitest
后端 build
前端 build
```

不得只说“理论上可用”。如全量测试受现有仓库问题阻塞，必须列出：

- 失败命令；
- 与本次修改是否相关；
- 定向测试是否通过；
- 已知 blocker。

---

# 23. 数据库迁移与部署

## 23.1 迁移原则

- 新表/字段向后兼容；
- 不删除现有字段；
- 不改写历史 Usage；
- 新索引评估锁表时间；
- 生产迁移前备份；
- 生成的 Ent 代码与 Schema 一致；
- 回滚时保留任务表数据，优先停用功能而非删表。

## 23.2 Feature Flags

建议：

```text
leonardo_provider_enabled
leonardo_media_api_enabled
leonardo_webhook_enabled
leonardo_video_enabled
```

第一阶段默认关闭，通过后台/配置逐步开启。

## 23.3 上线顺序

1. 部署 DB Schema，但功能关闭；
2. 部署 Provider/模型同步；
3. 添加一个测试 Leonardo 账号；
4. 同步模型；
5. 管理员测试；
6. 开启内部测试组；
7. 开启图片小流量；
8. 观察成功率、重复任务、计费；
9. 再开放视频；
10. 最后开放全模态。

## 23.4 RelayQ 生产部署约束

遵守既有 RelayQ 生产规则：

- 部署前标准备份；
- 不删除 Postgres volume；
- 不删除配置 volume；
- 后端变更优先本地构建 Linux 二进制并替换；
- 部署前检查未提交修改；
- 健康检查、日志和 API 验活；
- 新功能默认 Flag off。

---

# 24. 可观测性

## 24.1 Metrics

建议：

```text
media_generation_submitted_total{provider,model,modality}
media_generation_completed_total{provider,model,status}
media_generation_duration_seconds
media_generation_queue_age_seconds
media_generation_unknown_total
media_webhook_received_total{type,result}
media_poll_total{result}
media_upstream_cost_total{provider,model,unit}
media_customer_cost_total{model}
media_active_jobs{account}
media_pending_jobs{account}
```

## 24.2 告警

- invalid key 连续出现；
- upstream quota exhausted；
- unknown 任务增长；
- queued 超过阈值；
- Webhook 连续缺失；
- 重复扣费保护触发；
- 模型同步失败；
- 模型从目录消失；
- 负毛利或价格缺失。

---

# 25. 代码审查协议（Trae AI ↔ 小腾）

每个任务编号完成后，Trae AI 按以下模板联系小腾：

```markdown
## 审查请求：LEO-XXX

### 目标
...

### 修改文件
- path: symbol / reason

### 真实调用链
入口 -> service -> repository -> upstream

### 核心设计决定
...

### 风险与未确认点
...

### 测试
- command
- result

### Diff 范围
新增 X 行 / 删除 Y 行
```

小腾审查重点：

1. 是否修改了正确的共享入口而不是打补丁；
2. 是否破坏现有渠道；
3. 是否有重复提交/重复计费风险；
4. 是否泄露密钥；
5. 是否尊重异步状态；
6. 是否有 SSRF/大文件风险；
7. 是否错误硬编码模型参数；
8. 是否有最小可运行测试；
9. 是否存在无必要抽象；
10. 文档和真实行为是否一致。

同一个文件若双方都可能修改，必须先说明范围，避免并发覆盖。

---

# 26. Definition of Done

本项目不能以“能出一张图”定义完成。最终完成标准：

- [ ] Leonardo 为独立平台；
- [ ] Production API Key 安全存储和脱敏；
- [ ] `/v2/models` 动态同步；
- [ ] 模型 ID 与 slug 正确；
- [ ] 持久化 generation jobs；
- [ ] 服务重启不丢任务；
- [ ] 创建不确定时不重复生成；
- [ ] Webhook 与 Poll 双保险；
- [ ] Webhook 幂等和安全；
- [ ] OpenAI Images 兼容；
- [ ] 原生 Media API；
- [ ] 图片编辑/参考图安全上传；
- [ ] 至少一个视频模型端到端；
- [ ] Leonardo 静态定价计算器已纳入服务端版本化价格目录；
- [ ] 32 个图片模型、31 个视频模型价格结构校验通过；
- [ ] display name 与 API model slug 有明确映射；
- [ ] 创建前可以给出带快照版本的成本估价；
- [ ] 上游实际成本和快照估价可比较并告警；
- [ ] 历史任务不会因新价格或新汇率被重算；
- [ ] 上游成本和客户扣费可审计；
- [ ] 不重复扣费；
- [ ] NSFW 输出被正确处理；
- [ ] 模型下线可被发现；
- [ ] 管理后台可测试和管理；
- [ ] 现有 OpenAI/xAI/Gemini 链路无回归；
- [ ] 生产部署有备份、Flag、回滚和监控。

---

# 27. 当前开工顺序

原始阶段任务继续作为需求来源，实际执行严格以第 21.3 节的 Phase 9 收口顺序为准。当前起点为 `LEO-901`，不要并行铺开大量代码：

```text
LEO-901 真实协议补探针
→ LEO-902~905 模型和价格事实源
→ LEO-906~909 图片生产闭环
→ LEO-910~912 其他模态
→ LEO-913~915 运营治理
→ LEO-916~917 回归与发布
```

不要先改前端大页面；先把官方真实协议、任务一致性和计费安全钉死。

---

# 28. 官方参考资料

- 文档总索引：<https://docs.leonardo.ai/llms.txt>
- Quick Start：<https://docs.leonardo.ai/docs/getting-started>
- v2 Create Generation：<https://docs.leonardo.ai/reference/creategeneration-1>
- v2 Models：<https://docs.leonardo.ai/reference/getmodels>
- Webhook：<https://docs.leonardo.ai/docs/guide-to-the-webhook-callback-feature>
- Concurrency / Queue / Rate Limit：<https://docs.leonardo.ai/docs/guide-to-concurrency-queue-and-rate-limit>
- API Errors：<https://docs.leonardo.ai/docs/api-error-messages>
- PAYG：<https://docs.leonardo.ai/docs/payg-guide>
- Pricing FAQ：<https://docs.leonardo.ai/docs/pricing-and-plans-faq>
- NSFW：<https://docs.leonardo.ai/docs/guide-to-handling-not-safe-for-work-image-generation-nsfw>
- Deprecations：<https://docs.leonardo.ai/docs/deprecations-changes>
- GPT Image 2：<https://docs.leonardo.ai/docs/gpt-image-2>
- Nano Banana 2：<https://docs.leonardo.ai/docs/nano-banana-2>
- Grok Imagine 1.5：<https://docs.leonardo.ai/docs/grok-imagine-15>
- Image Upload Recipe：<https://docs.leonardo.ai/recipes/uploading-an-image>
- Python SDK：<https://github.com/Leonardo-Interactive/leonardo-python-sdk>
- TypeScript SDK：<https://github.com/Leonardo-Interactive/leonardo-ts-sdk>
- 本地 Leonardo 定价计算器：`tools/leonardo-pricing-calculator/README.md`
- 定价快照数据：`tools/leonardo-pricing-calculator/pricing-data.js`
- 本地定价计算器页面：`tools/leonardo-pricing-calculator/index.html`

---

# 29. 最终架构结论

本次开发应被视为 RelayQ 的**异步多媒体任务基础设施升级**，而不只是增加一个图片渠道。

正确路径：

```text
Leonardo 独立 Provider
+ 动态模型目录
+ 持久化异步任务
+ Webhook/Poll 双保险
+ OpenAI 兼容门面
+ RelayQ 原生 Media API
+ 实际成本与客户计费分离
```

错误路径：

```text
把 Leonardo 当 OpenAI Base URL
+ 在 openai_images.go 里不断加模型名判断
+ 用 Redis TTL 保存长任务
+ 完成后才临时猜价格
```

第一种路径可以继续复用到 Fal、Replicate、Runway、Kling、MiniMax、Veo、Firefly 等异步媒体平台；第二种路径会在视频、音频和 3D 接入时被迫推翻重写。
