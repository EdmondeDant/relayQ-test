# Leonardo Production API Probe Results

## LEO-901 只读鉴权复验（2026-08-05）

### 执行边界

- 使用隔离数据库 `leo300a7_r1_20260803_2024` 中唯一的 Leonardo API Key 账号；
- 仅执行一次 `GET /api/rest/v2/models`，未发送创建、上传或其他付费请求；
- API Key 只在进程内读取并用于 Authorization Header，未写入文档和命令输出；
- 本次不是 LEO-002 或 LEO-002C 的重试，也没有查询或补发历史 unknown 请求。

### 结果

| 项目 | 值 |
|------|-----|
| 时间 | 2026-08-05 Asia/Shanghai |
| 请求 | `GET /api/rest/v2/models` |
| 结果 | 失败 |
| 脱敏错误 | `Invalid response from authorization hook` |
| 错误路径 | `$` |
| 错误代码 | `unexpected` |
| 创建 POST 次数 | 0 |
| 付费副作用 | 无 |

### 判定与处置

- 当前账号不能通过只读鉴权前置门禁；历史上同一项目存在 GET 200 证据，但不能据此推定当前凭证仍有效；
- 在账号归属、Key 状态、Production API 权限、PAYG/余额和 Leonardo 服务状态人工确认前，LEO-901 真实创建保持阻塞；
- 不执行付费图片创建、不切换账号自动重试、不把该错误解释为安全可重试的创建失败；
- 恢复后必须先重新取得一次 `/v2/models` 200，再建立新的探针 ID、预算和停止条件。

> 探针时间：2026-08-01T17:08:50+08:00 (Asia/Shanghai)
> 目标环境：Leonardo Production API v2
> 请求性质：模型目录 GET 为只读；另记录一次 POST 提交（结果见下）
> Key：仅临时环境变量，未持久化

---

## 有效 Key 探针

### 请求

```http
GET https://cloud.leonardo.ai/api/rest/v2/models
Accept: application/json
Authorization: Bearer <LEONARDO_API_KEY>
```

### HTTP 结果

| 项目 | 值 |
|------|-----|
| 状态码 | 200 |
| Content-Type | application/json; charset=utf-8 |
| Body 大小 | 251,726 bytes (~246 KB) |
| X-Request-ID | 未返回 |
| RateLimit 系列头 | 未返回 |
| Retry-After | 未返回 |

### 响应头（可公开部分）

```
Date: Sat, 01 Aug 2026 09:08:50 GMT
Content-Type: application/json; charset=utf-8
X-Content-Type-Options: nosniff
```

### 模型目录

| 项目 | 值 |
|------|-----|
| 顶层字段 | `productionApiAvailableModels`（仅有这一个顶层键） |
| 模型总数 | 69 |
| 单模型字段集 | `id`, `name`, `parameters` |

### ID 与 Name 对比

| 字段 | 类型 | 示例 |
|------|------|------|
| `id` | UUID v4 string | `740848f7-3240-481a-b251-625f92ecb7ec` |
| `name` | 人类可读名 | `Motion 2.0` |
| **独立 request slug** | **不存在** | `/v2/models` 未直接返回用于 API 请求的模型 slug |

> **确认**：Leonardo Production API v2 模型目录中不存在官方文档所述的 `GPT_IMAGE_2` 等 slug 字段。实际 `id` 为 UUID，`name` 为显示名。使用模型的 API 请求格式需在 LEO-002 中通过实际创建调用来确认。

### parameters 结构（JSON Schema）

```json
{
  "type": "object",
  "properties": { "...": {} },
  "additionalProperties": false,
  "description": "...",
  "required": ["..."]
}
```

parameters 为标准 JSON Schema dict，包含 `type`、`properties`、`additionalProperties`、`description`、`required` 字段。

---

## 无效 Key 探针

### 请求

```http
GET https://cloud.leonardo.ai/api/rest/v2/models
Authorization: Bearer 00000000-0000-0000-0000-000000000000
```

### HTTP 结果

| 项目 | 值 |
|------|-----|
| 状态码 | 401 |
| Content-Type | application/json; charset=utf-8 |
| Body 大小 | 91 bytes |

### 错误结构

```json
{
  "error": "Authentication hook unauthorized this request",
  "path": "$",
  "code": "access-denied"
}
```

错误字段：`error`（消息）、`path`（JSON Pointer）、`code`（错误码）。

---

## 与官方文档的差异

| 项目 | 官方文档声称 | 本次真实响应确认 | 状态 |
|------|-------------|-----------------|------|
| 模型 ID 格式 | `GPT_IMAGE_2` 等常量 | UUID v4 | **已确认差异** |
| 独立 slug 字段 | 未明确声明 | 不存在 | 确认 |
| `productionApiAvailableModels` | 文档提及 | 确认为唯一顶层键 | 一致 |
| 模型数量 | 未声明 | 69 | — |
| parameters | JSON Schema | JSON Schema | 一致 |
| 限流头 | 未声明 | 当前未返回 | — |

---

## 文件产出

- 临时原始文件：`%TEMP%\relayq-leonardo-probe\`（已删除）
- 本文档：`docs/leonardo-api-probe-results.md`

---

## LEO-002：FLUX Schnell 单次付费创建探针

> 执行时间：未保留

### 脱敏请求

```http
POST https://cloud.leonardo.ai/api/rest/v2/generations
Authorization: Bearer <LEONARDO_API_KEY>
Content-Type: application/json

{
  "model": "flux-schnell",
  "public": false,
  "parameters": {
    "prompt": "A simple red apple on a plain white background",
    "quantity": 1,
    "width": 896,
    "height": 896
  }
}
```

请求体依据获批请求和执行汇报记录；Prompt 来自执行前已批准且执行者确认使用的请求参数，Authorization 已脱敏。

### 请求计数

计数范围仅限本次创建提交及其提交后核验，不包含本文前述两次模型目录探针。

| 请求类型 | 次数 |
|----------|-----:|
| 创建 POST | 1 |
| POST 自动重发 | 0 |
| 提交后任务状态 GET | 0 |
| 本节实际发送请求合计 | 1 |

### 状态查询

未执行状态查询。

原因：

- 创建响应中不存在 `generationId`；
- 无法构造合法的 `GET /api/rest/v1/generations/{generationId}`；
- 不猜测 generation ID；
- 不通过第二次 POST 获取新的 generation ID。

因此：

```text
GET 次数：0
最终任务状态：无法查询
```

### HTTP 与副作用分类

| 分类 | 结果 |
|------|------|
| HTTP 结果 | 500 |
| 响应 Body 格式 | 合法 JSON |
| JSON 顶层字段 | 仅 `error`、`path`、`code` |
| 服务端是否受理 | 未知 |
| 是否创建任务 | 未知 |
| 提交状态 | `submission_unknown` |
| 副作用分类 | `side_effect_unknown` |

HTTP 500 只描述已观测到的 HTTP 响应，不能证明请求未被受理、任务未被创建或服务端没有产生计费副作用。

### 响应字段存在性

| 字段或数据 | 存在性 |
|------------|--------|
| `error` | 存在；值未保留 |
| `path` | 存在；值未保留 |
| `code` | 存在；值未保留 |
| `generationId` | 不存在 |
| `cost` | 不存在 |
| `apiCreditCost` | 不存在 |
| 其他 JSON 顶层字段 | 不存在 |
| `X-Request-ID` 或其他响应头关联 ID | 未确认；完整响应头未保留 |

字段存在性依据已知执行日志；`error`、`path`、`code` 的具体值未保留，不补写或推断。

### 成本与预算

| 项目 | 结果 |
|------|------|
| 提交前预估成本 | USD 0.003 |
| 本次探针预算上限 | USD 0.01 |
| 响应 `cost` | 不存在 |
| 响应 `apiCreditCost` | 不存在 |
| 账号实际扣费 | 未知 |
| 最终实际成本及单位 | 未知 |

预估成本不等于实际扣费；现有证据不足以确认是否产生费用。

### 采集缺陷

- 执行时间未保留。
- HTTP 500 JSON 的 `error`、`path`、`code` 值未保留。
- POST 响应 `Content-Type` 未保留、未确认；合法 JSON 仅描述 Body 可解析，不能据此推断响应头。
- POST 响应 Body 大小未保留。
- POST 响应体 SHA-256 未保留。
- POST 响应 `Retry-After` 未保留、未确认。
- `X-Request-ID` 及其他可公开响应头未保留、未确认。
- 创建响应中不存在 `generationId`，无法构造合法状态查询，故未执行任务状态 GET。
- 未采集控制台任务记录、账单或账户余额变化，无法确认副作用和实际成本。

### 安全处置

- 该不确定提交不得再次 POST，避免重复任务与重复计费。
- 仅可通过 Leonardo 控制台、账单和既有任务记录进行人工核对；本次 LEO-002 调查过程中不发起任何新的创建请求。
- 任何未来付费探针必须作为新的独立操作重新获得人工批准，不能视为本次请求的重试或重放。
- API Key 保持脱敏且不写入文档；不补造未保留的错误值、时间或成本值。

---

## LEO-002A：鉴权状态只读调查

> 请求性质：只读，仅调用模型目录
> GET 次数：1
> POST 次数：0

### 请求

```http
GET https://cloud.leonardo.ai/api/rest/v2/models
Authorization: Bearer <LEONARDO_API_KEY>
```

### 结果

| 项目 | 结果 |
|------|------|
| 模型目录是否获得 | 否 |
| 匹配模型数 | 0（因目录未获得，不表示模型不存在） |
| `error` | `Invalid response from authorization hook` |
| `path` | `$` |
| `code` | `unexpected` |
| 当前凭证状态 | 未通过 Production API 鉴权 |
| 生成任务副作用 | 无；本次未发送创建请求 |

`matches=0` 是因为未获得模型目录，不能据此认定 FLUX Schnell 或任何其他模型不存在。

### 与 LEO-001 的差异

LEO-001 使用当时有效的 Production API Key 调用同一端点返回 HTTP 200；LEO-002A 当前凭证返回鉴权错误。现有证据无法确认两次请求是否使用同一 Key，也无法区分凭证格式、注入、撤销、轮换或上游鉴权异常，不作原因推断。

### 离线布尔核验

以下结果仅来自离线格式核验，未输出或持久化凭证明文：

| 检查项 | 结果 |
|--------|------|
| `configured` | `true` |
| `nonempty` | `true` |
| `contains_whitespace` | `false` |
| `contains_crlf` | `false` |
| `contains_bearer_prefix` | `false` |
| `uuid_format` | `false` |

经审查未再次发送 GET；LEO-002A 的 GET 总次数仍为 1，POST 总次数仍为 0。

### 处置

- 停止使用当前凭证继续发送请求。
- 不重复 GET，不尝试其他 Leonardo 端点。
- 当前凭证禁止继续用于任何探针。
- 当前凭证状态未澄清前，停止所有付费探针以及任何可能产生费用或副作用的请求。
- 仅允许进行不输出凭证明文的本地来源与格式核验；任何只读复验均需重新明确批准。
- 只读复验成功也不自动批准付费 POST；任何未来付费探针仍需独立审批。

---

## LEO-002B：新凭证模型目录只读复验

> 执行时间：未保留
> 请求性质：只读模型目录
> 凭证来源：User 级 `LEONARDO_API_KEY`，未记录明文或凭证指纹

### 请求与计数

```http
GET https://cloud.leonardo.ai/api/rest/v2/models
Accept: application/json
Authorization: Bearer <LEONARDO_API_KEY>
```

| 项目 | 次数 |
|------|-----:|
| GET | 1 |
| POST | 0 |
| 自动重试 | 0 |
| 重定向跟随 | 0 |
| 其他 Leonardo 请求 | 0 |

### HTTP 与响应摘要

| 项目 | 结果 |
|------|------|
| HTTP 状态码 | 200 |
| `Content-Type` | `application/json; charset=utf-8` |
| `Content-Length` | 未返回 |
| Body 大小 | 251,726 bytes |
| Body 读取 | 完整 |
| 响应体大小上限 | 未触发，`body_limit_exceeded=false` |
| Body SHA-256 | `b8bc257929c8ce6a8fbe051d354e67cd7de84da965559c467ecebc8829f5d22c`，基于完整原始响应字节 |
| `X-Request-ID` | `8623718c-3dfe-4de4-be8b-fc35ddb02821` |
| `Retry-After` | 未返回 |

Request ID 属于允许采集的响应关联标识，可完整记录。Body 大小与 LEO-001 相同，但不据此宣称两次响应内容完全相同。

### JSON 与目录摘要

| 项目 | 结果 |
|------|------|
| JSON 解析 | 成功 |
| 顶层 JSON 类型 | object |
| 顶层字段 | 仅 `productionApiAvailableModels` |
| 模型总数 | 69 |
| FLUX Schnell 匹配规则 | `name` 去首尾空白后大小写不敏感精确匹配 |
| FLUX Schnell 精确匹配数 | 1 |
| 匹配项 `id` | `1dd50843-d653-4516-a8e3-f0238ee453ff` |
| 匹配项 UUID 格式核验 | `true` |
| 匹配项 `name` | `FLUX Schnell` |
| 匹配项顶层字段 | `id`, `name`, `parameters` |
| `parameters` | object |
| Schema `type` | `object` |
| Schema `required` | 仅 `prompt` |
| Schema `properties` | `seed`, `width`, `height`, `prompt`, `quantity`, `guidances`, `style_ids`, `prompt_enhance` |
| Schema `additionalProperties` | `false` |
| Schema `description` | 存在 |
| 独立 request slug 字段 | 不存在 |

### 证据边界与处置

- 新 Key 仅在本次访问 `/v2/models` 时鉴权成功；不代表账号全部权限、余额、创建接口、查询接口、Webhook、上传或计费能力可用。
- 本次目录响应未提供 request slug；目录中的 UUID、名称和 Schema 不能证明创建请求应使用 `flux-schnell`，也不能证明 UUID 可直接作为 request model。
- 本节是独立只读复验证据，不覆盖或改写 LEO-001、LEO-002、LEO-002A 的历史结果。
- 本次成功不批准任何付费 POST，也不批准其他 Leonardo 请求。
- LEO-002 仍为永久 `submission_unknown` / `side_effect_unknown`，不得重试、重放或补发。

---

## LEO-002C：独立最低成本付费创建探针

> 执行时间：未保留
> 探针性质：新的独立付费创建探针，不是 LEO-002 的重试、续跑或重放
> 执行环境：本地命令进程；API Key 仅来自环境变量，未记录明文或凭证指纹

### 已批准的脱敏请求

```http
POST https://cloud.leonardo.ai/api/rest/v2/generations
Authorization: Bearer <LEONARDO_API_KEY>
Content-Type: application/json
```

```json
{
  "model": "flux-schnell",
  "public": false,
  "parameters": {
    "prompt": "A single blue ceramic cup centered on a plain white background, API integration probe",
    "quantity": 1,
    "width": 896,
    "height": 896
  }
}
```

以上是调用方拟提交的请求参数。进程异常退出，现有证据无法确认完整请求是否到达上游。

### 请求计数

| 项目 | 结果 |
|------|------|
| 创建命令启动次数 | 1 |
| 可能发送的创建 POST | 0 或 1，无法确定 |
| 补发或重放 POST | 0 |
| 自动重试 | 0 |
| 状态 GET | 0 |
| 换 Key | 0 |
| 换账号 | 0 |
| 故障转移 | 0 |
| 其他 Leonardo 请求 | 0 |

命令启动一次不等于上游确定收到一次 POST。

### 本地执行结果

| 项目 | 结果 |
|------|------|
| 本地进程退出码 | `local_process_exit_code=5999` |
| stdout | 空 |
| stderr | 空 |
| HTTP 状态 | 未取得 |
| 响应头 | 未取得 |
| 响应体 | 未取得 |
| `Content-Type` | 未取得 |
| Body 大小与 SHA-256 | 未取得 |
| `generationId` | 未取得 |
| Request ID | 未取得 |
| `Retry-After` | 未取得 |

`5999` 仅是本地进程退出码，不是 HTTP 状态码、Leonardo 错误码或上游响应。

### 提交与副作用分类

| 项目 | 结果 |
|------|------|
| HTTP 请求是否开始发送 | 未知 |
| 上游是否收到请求 | 未知 |
| 上游是否受理 | 未知 |
| 是否创建任务 | 未知 |
| 是否完成生成 | 未知 |
| 是否产生扣费 | 未知 |
| `submission_status` | `submission_unknown` |
| `side_effect_status` | `side_effect_unknown` |
| `billing_status` | `manual_review` |

本地命令进程异常退出，且 stdout 和 stderr 均为空；没有证据证明请求在发送前失败，也没有上游响应或关联 ID，因此无法排除请求已经到达并产生副作用。

### 成本与计费

| 字段 | 结果 |
|------|------|
| `estimated_upstream_cost` | USD 0.003 |
| `probe_budget_limit` | USD 0.01 |
| `actual_upstream_cost_amount` | `null` |
| `actual_upstream_cost_unit` | `null` |
| `customer_cost` | `null` |
| `gross_margin` | `null` |
| `cost_variance` | `null` |
| `billing_status` | `manual_review` |
| `billing_reference` | `null` |

USD 0.003 来自本地定价快照，只是提交前估价；预算上限不是实际成本。在取得 Leonardo 账单、余额变化或任务记录等明确证据前，不以估价、预算或 0 填充实际成本，不扣款、不结算、不退款，也不推导毛利。

### 官方 Chrome 调查证据

本节记录通过 Chrome 调查 Leonardo 官方页面取得的协议证据，不代表 LEO-002C 获得了任何 HTTP 响应：

| 官方证据 | URL | 已确认摘要 |
|----------|-----|------------|
| FLUX Schnell 指南 | <https://docs.leonardo.ai/docs/flux-schnell> | 创建请求的 `model` 使用 `flux-schnell`；不能使用目录 UUID 或显示名代替 |
| v2 创建接口 | <https://docs.leonardo.ai/reference/creategeneration-1> | 创建端点为 `POST /api/rest/v2/generations` |
| v1 单任务查询 | <https://docs.leonardo.ai/v1.0/reference/getgenerationbyid> | 仅取得合法 `generationId` 后，查询 `GET /api/rest/v1/generations/{generationId}` |
| API FAQ | <https://docs.leonardo.ai/docs/api-faq> | 已公开状态包括 `PENDING`、`COMPLETE`、`FAILED` |
| NSFW 与输出说明 | <https://docs.leonardo.ai/docs/guide-to-handling-not-safe-for-work-image-generation-nsfw> | 成功结果包含生成图片、URL 与 NSFW 信息 |
| PAYG 指南 | <https://docs.leonardo.ai/docs/payg-guide> | API 按美元计费，实际扣费取决于模型和设置 |
| 官方价格计算器说明 | <https://docs.leonardo.ai/docs/plan-with-the-pricing-calculator> | 可用于生成前成本规划，不替代实际账单证据 |

官方参数证据确认 `width` 和 `height` 范围为 32–2048 且必须是 8 的倍数，因此 896×896 合法；`quantity=1` 也在允许范围内。公开官方文档未直接给出本配置的实时美元价格，USD 0.003 仍只属于本地快照估价。

### RelayQ 三字段显示名契约

| RelayQ 字段 | FLUX Schnell 值 | 职责 |
|-------------|-----------------|------|
| `display_name` | `FLUX Schnell` | UI、客户模型目录、订单和账单使用的人类可读显示名 |
| `provider_model_id` | `1dd50843-d653-4516-a8e3-f0238ee453ff` | 关联 `/v2/models` 目录记录的 UUID |
| `request_model_slug` | `flux-schnell` | 发送 `/v2/generations` 时使用的协议值 |

三者必须独立保存和使用：不得把 UUID 显示给客户作为模型名称，不得把 request slug 当作唯一显示名，不得把 UUID 直接传给 v2 创建接口，也不得根据 UUID 或显示名自动猜测 slug。

### 永久处置

- LEO-002C 永久禁止重新 POST、补发、重放或以新任务补偿。
- 不换 Key、账号、网络或工具再次执行本次请求，不进行故障转移。
- 不构造、猜测或搜索 `generationId`；状态 GET 保持 0。
- 仅允许通过 Leonardo 控制台、账单、余额变化或既有任务记录进行人工核对，且人工核对不得触发新生成请求。
- 不把空 stdout 或 stderr 解释成未发送请求、未创建任务或未产生副作用。
- 不补造 HTTP 状态、错误体、执行时间、Request ID 或 `generationId`。
- 本节不覆盖 LEO-001、LEO-002、LEO-002A、LEO-002B 的历史证据。

---

## LEO-NEXT-001：下一批模型目录只读证据

> 执行时间：2026-08-08T19:55:57+08:00
> 请求性质：只读模型目录
> 凭证来源：`LEONARDO_API_KEY` 环境变量，未记录明文、长度、哈希或指纹

### 请求与响应

| 项目 | 结果 |
|------|------|
| 请求 | `GET /api/rest/v2/models` |
| HTTP 状态码 | 200 |
| `Content-Type` | `application/json; charset=utf-8` |
| Body 大小 | 263,126 bytes |
| Body SHA-256 | `df0605ea6c1f76c092dece101546b4d9f1e14ff5596a6e7306581d79cfd37493` |
| `X-Request-ID` | `08dd043c-0737-476f-98cd-4324f6909796` |
| JSON 解析 | 成功 |
| 模型总数 | 73 |
| GET / POST / 自动重试 / 重定向跟随 | 1 / 0 / 0 / 0 |

### 图片候选

| 候选 | Provider UUID | Schema SHA-256 | Required | Properties 摘要 |
|------|---------------|----------------|----------|-----------------|
| Anime | `e71a1c2f-4f80-4800-934f-2c68979d8cc8` | `6a3dff1c054231bd848a3d96435c0c267a7fe55bbc6223fdbd53f4a88674377d` | `prompt` | `contrast,guidances,height,mode,negative_prompt,platform_elements,prompt,prompt_enhance,quantity,seed,style_ids,tiling,transparency,user_elements,width` |
| Cinematic Kino | `aa77f04e-3eec-4034-9c07-d0f619684628` | `3f6769205f4be19f86561c1f6bcde8a3e1b187d20582537f8ab20f34651b69fa` | `prompt` | 同上 |
| Concept Art | `dd29ac47-ea88-4720-8678-b8633245c09c` | `d7e9c1a19dc856e301321e5b2c2691982ca745ca4732283f2e41af19407524be` | `prompt` | 同上 |
| Graphic Design | `9d4ace10-25dd-42fd-a6be-a301a7ac614f` | `c178f259ef7da5bbe85293635dbdaa030221157f990f5bb4316b491280dbda83` | `prompt` | 同上 |

候补 Illustrative Albedo、Leonardo Lightning、Lifelike Vision、Portrait Perfect、Stock Photography 均各精确匹配 1 项，Schema 均为 `type=object`、`additionalProperties=false`、`required=[prompt]`。

### 视频候选

| 候选 | Provider UUID | Schema SHA-256 | Required | Properties |
|------|---------------|----------------|----------|------------|
| Seedance 1.0 Pro Fast | `b959ecc2-a7f0-4618-9877-1bc45fc27570` | `bc599100d3d285453e8a608bc25e034da7cf1ada253f8e6a68ee010b1a3a7c00` | `prompt,quantity` | `duration,guidances,height,mode,prompt,prompt_enhance,quantity,seed,width` |
| Motion 2.0 Fast | `0a7a3eb2-3905-480b-a89a-2f3ffff545e7` | `36db3637b6def74e2276869a88dcf69f1dc5585720412c3225588486c3ec789c` | `prompt` | `controls,elements,frame_interpolation,guidances,height,mode,negative_prompt,prompt,prompt_enhance,quantity,seed,start_frame,style_ids,width` |
| Seedance 1.0 Pro | `728c9eac-b17d-47fe-b382-b9a28687fa85` | `5e1bb50b7c773fa474d00801a936412bcd11b6effdb9069bd629df51d95417fe` | `prompt,quantity` | `duration,guidances,height,mode,prompt,prompt_enhance,quantity,seed,width` |
| Wan 2.7 | `52884d8c-e2b9-4bb1-8ed5-927d390fe53a` | `f41877cd8c7a64aa53a7f14c26b864808bbad2bbbf6731b70a8d71b36e9e6cc4` | `prompt` | `duration,guidances,height,omni_edit,prompt,quantity,resolution,width` |

候补 Hailuo 2.3 Fast、Hailuo 2.3、Kling 2.5 Turbo Standard 均各精确匹配 1 项，Schema 均为 `type=object` 且 `additionalProperties=false`。

### Gate 结论

- 8 个主候选及 8 个候补均在当前账号目录中各精确匹配 1 项；目录鉴权、UUID 和顶层 Schema 结构通过。
- 本次响应仍未提供独立 request slug，不能把显示名或 Provider UUID 当作 `/v2/generations` 的 `model` 值。
- 当前只完成 Gate 1 的目录存在、账号可见和静态 Schema 合法部分；最低价配置、完整枚举/范围、输入能力和 request slug 尚未获得创建成功证据。
- 所有候选继续保持 `candidate/blocked`，不得加入 verified registry，不得打开模型白名单或 `video_enabled`。
- 本次没有创建、上传、状态查询、Webhook 或其他 Leonardo 请求，不产生已知付费副作用。

### I-01 Anime 创建前协议核验

- Leonardo 官方常用模型值文档确认目录 UUID `e71a1c2f-4f80-4800-934f-2c68979d8cc8` 对应 `Leonardo Anime XL`。
- 当前官方 v2 文档索引没有 Anime 专属生成指南；v2 创建接口的公开参考页当前只展示 `motion_2.0` 枚举，不能据此证明 Anime 的 v2 request slug。
- 官方建议通过模型专属生成指南或 Web App 的 `Get API Code` 取得准确实现；现有证据没有取得 Anime 的导出代码。
- 第三方资料出现 `leonardo-anime-xl`，但不属于 Leonardo Production API 官方协议证据，不能批准付费请求。
- 因 request slug、`mode` 枚举和最低价参数仍无官方证据，本轮没有执行 Anime 创建 POST；POST、状态 GET 和付费副作用均为 0。
- 用户决定将 Anime 移出本批接入范围；它不会进入后续探针、verified registry、模型白名单或生产启用流程。

### 其余候选官方 request model 核验

Leonardo 官方 v2 Create a Generation OpenAPI 已确认：

| 显示名 | request model |
|--------|---------------|
| Cinematic Kino | `kino-xl` |
| Concept Art | `concept-art` |
| Graphic Design | `graphic-design` |
| Illustrative Albedo | `illustrative-albedo` |
| Seedance 1.0 Pro Fast | `seedance-1.0-pro-fast` |
| Motion 2.0 Fast | `motion_2.0-fast` |
| Seedance 1.0 Pro | `seedance-1.0-pro` |
| Wan 2.7 | `wan-2.7` |

图片模型共同支持 `FAST/QUALITY`、`quantity=1..8` 和经过 Schema 约束的宽高；视频官方最低档分别为 Seedance 4 秒 864×480、Motion 2.0 Fast 480p 832×480、Wan 2.7 2 秒 1280×720。以上仅完成创建前协议核验，未替代每个模型的一次真实成功探针。

### I-01 至 I-04 最低成本真实探针

四个图片模型各发送一次独立创建 POST，均不重试；仅凭各自返回的 `generationId` 查询原任务。

| 模型 | 配置 | 创建 | 上游成本 | 终态 | 输出 |
|------|------|------|----------|------|------|
| `kino-xl` | FAST / 896×896 / 1 | HTTP 200 | 0.0045 DOLLARS | COMPLETE | 1 个 HTTPS 图片，`nsfw=false` |
| `concept-art` | FAST / 888×888 / 1 | HTTP 200 | 0.0045 DOLLARS | COMPLETE | 1 个 HTTPS 图片，`nsfw=false` |
| `graphic-design` | FAST / 888×888 / 1 | HTTP 200 | 0.0045 DOLLARS | COMPLETE | 1 个 HTTPS 图片，`nsfw=false` |
| `illustrative-albedo` | FAST / 888×888 / 1 | HTTP 200 | 0.0045 DOLLARS | COMPLETE | 1 个 HTTPS 图片，`nsfw=false` |

请求计数：创建 POST 共 4 次，每个模型 1 次；创建自动重试 0；没有补发、重放、换 Key 或故障转移。四个创建响应均返回 `generate.generationId` 和 `generate.cost`，成本与当前最低价快照一致。

四个图片模型已通过 request model、最低价参数、真实创建、终态图片、HTTPS URL、NSFW 和实际成本门禁，可以进入 verified registry。生产模型白名单仍默认关闭，由管理员逐模型灰度启用。

### V-01 Seedance 1.0 Pro Fast 最低成本真实探针

> 执行时间：2026-08-09 Asia/Shanghai
> 创建请求：严格 1 次 POST，自动重试 0，重定向跟随 0

| 项目 | 结果 |
|------|------|
| 模型 | `seedance-1.0-pro-fast` |
| 配置 | 4 秒 / 864×480 / quantity 1 / 无音频 |
| RelayQ 上游估价 | USD 0.0449 |
| RelayQ 客户预留 | USD 0.31879（当前媒体客户倍率 7.1） |
| 创建结果 | 成功，返回可信 UUID generation ID |
| 终态 | `succeeded`，output_count 1 |
| 输出 | HTTPS MP4，`video/mp4`，2,958,782 bytes，`ftyp` magic-byte 通过 |
| 资金预留 | `settled`，金额 USD 0.31879 |
| 用户余额 | USD 1.00000000 → USD 0.68121000，差额 USD 0.31879 |
| Usage log | 唯一 1 条，`billing_mode=video`，`image_count=0`，费用 USD 0.31879 |
| 创建响应成本 | `apiCreditCost=0`，RelayQ 记录 `0 API_CREDIT` |

只轮询创建响应返回的 generation ID，没有猜测或查询其他任务。RelayQ 的 reservation、generation job、usage log 和余额差额完全一致。

Leonardo 当前登录会话显示 Free 套餐和 150 Fast Tokens，但 Subscription and Billing 页面不提供逐笔账单或按 generation 查询用量；generation 页面也不能展示该私有 API 任务。因此 Leonardo 控制台无法独立确认本次实际扣费。可验证的上游实际成本字段仅为创建响应的 `apiCreditCost=0`，它与 USD 0.0449 的预估口径不一致，不能将 USD 0.0449 宣称为已由实际账单确认。上线前应将此成本差异作为账务风险保留，并在可取得逐笔账单或余额前后快照时补证。

### V-02 至 V-04 最低成本真实探针

> 执行时间：2026-08-09 Asia/Shanghai
> 创建请求：每个模型严格 1 次 POST，自动重试 0，重定向跟随 0

| 模型 | 配置 | RelayQ 上游估价 | RelayQ 客户预留 | 终态 | MP4 校验 |
|------|------|------------------|------------------|------|----------|
| `motion_2.0-fast` | 固定时长 / 832×480 / quantity 1 | USD 0.1047 | USD 0.74337 | succeeded | 3,430,500 bytes，`ftyp` 通过 |
| `seedance-1.0-pro` | 4 秒 / 864×480 / quantity 1 | USD 0.1346 | USD 0.95566 | succeeded | 3,634,206 bytes，`ftyp` 通过 |
| `wan-2.7` | 2 秒 / 1280×720 / quantity 1 | USD 0.1645 | USD 1.16795 | succeeded | 946,264 bytes，`ftyp` 通过 |

三个任务均取得可信 generation ID，仅轮询各自返回的 ID。终态均为 `succeeded`、`output_count=1`、HTTPS MP4、`video/mp4`、`nsfw=false`。三个 reservation 均为 `settled`，usage log 各唯一 1 条，`billing_mode=video`、`image_count=0`，用户余额 USD 97.78105550 → USD 94.91407550，差额 USD 2.86698，与三笔客户预留之和一致。

三个创建响应的可验证实际成本仍为 `0 API_CREDIT`，RelayQ 保存的 `actual_upstream_cost_amount=0`。Leonardo 控制台不提供逐 generation 账单，因此 USD 0.1047、USD 0.1346、USD 0.1645 仅能声明为认证价格计算器的精确估价，不能声明为逐笔实际账单成本。

### Seedance 2.0 与 Kling 系列目录复核

> 执行时间：2026-08-14 Asia/Shanghai

- 已联网对照 Seedance 和 Kling 原厂能力、Leonardo 官方模型指南，并对当前账号执行只读 `GET /api/rest/v2/models`，没有发送创建 POST。
- Leonardo 官方文档已公开 Seedance 2.0、Seedance 2.0 Fast、Seedance 2.0 Mini，以及 Kling 2.1 Pro、2.5 Turbo、2.6、3.0、O1 和 O3。
- 当前账号 Production 目录在目标系列中只返回 `Kling Video O3 Omni`；Seedance 2.0 全系列和其他 Kling 型号未返回，因此无法取得当前账号可见 UUID 与 Schema，继续保持 blocked，不加入 verified registry。

| 模型 | Provider UUID | request model | Schema 摘要 | 状态 |
|------|---------------|---------------|-------------|------|
| Kling Video O3 Omni | `0d5109cf-d256-4720-86d3-d8e5ff5a3ce2` | `kling-video-o-3` | `duration=3..15`；720p/1080p/2160p；`motion_has_audio`；start/end/image/video reference | verified |
| Seedance 2.0 / Fast / Mini | 当前目录未返回 | 官方文档分别为 `seedance-2.0`、`seedance-2.0-fast`、`seedance-2.0-mini` | 无当前账号 Production Schema | blocked |
| 其他 Kling 型号 | 当前目录未返回 | 官方文档可查 | 无当前账号 Production Schema | blocked |

- Kling O3 官方尺寸为 720p、1080p、2160p 下的 16:9、1:1、9:16；时长 3–15 秒。视频参考模式最长 10 秒，当前 RelayQ 首批只开放文本和首帧图片，不开放尚未端到端验证的尾帧、多图片和视频编辑。
- 当前认证价格目录给出 Kling Video O3 Omni 3 秒最低价 USD 1.0046、15 秒 USD 5.0232。RelayQ 使用线性插值并按模型、时长和已验证尺寸 fail closed；客户价仍为本地成本 × 7.1。

### Kling O3 最低成本真实探针

> 执行时间：2026-08-15 Asia/Shanghai
> 创建请求：直连 Leonardo 与 RelayQ 各严格 1 次有效 POST，自动重试 0；RelayQ 首次预检因账号模型映射缺失在上游选择前返回 503，没有预留资金、generation job 或上游副作用

- 直连 Leonardo：generation ID `1f197fd2-30f6-6e90-a026-b7ffcf2cba29`，3 秒、1280×720、quantity 1、无音频；17 次 `PENDING` 后 `COMPLETE`，单个输出、`nsfw=false`、HTTPS `video/mp4`、`ftyp` 通过。
- RelayQ：request ID `gen_rq_2d737819c97bc019ad55ab86767da8f9`，状态 `succeeded`，`output_count=1`，content 代理返回 HTTP 200、`video/mp4`、`ftyp` 通过。
- 账务：认证价格估价 USD 1.0046，客户预留与结算 USD 7.13266；reservation=`settled`，usage log 唯一 1 条且总额 USD 7.13266，余额 USD 71.92084550 → USD 64.78818550，差额完全一致。
- 上游实际成本：创建响应与 generation job 均为 `0 API_CREDIT`。USD 1.0046 只能声明为认证价格计算器的精确估价，不能声明为 Leonardo 逐笔实际账单成本。
- 验收发现账号 `model_mapping` 仍停留在接入前的 12 个模型，导致服务在付费 POST 前返回 `LEONARDO_MEDIA_NO_AVAILABLE_ACCOUNT`；同步 `kling-video-o-3` 后恢复为 13 个已验证模型。
- Windows 环境依赖本机 HTTPS 代理。媒体 content 原先使用仅直连的 SSRF 安全客户端，导致 CDN 下载 502；现改为在配置 `HTTPS_PROXY` 时复用“原始目标 IP 预校验 + 显式代理”的安全客户端，仍禁止危险目标和重定向，不放宽 SSRF 策略。
