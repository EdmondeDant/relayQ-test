# Leonardo Production API Probe Results

> 探针时间：2026-08-01T17:08:50+08:00 (Asia/Shanghai)
> 目标环境：Leonardo Production API v2
> 请求性质：只读模型目录，不产生生成费用
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
