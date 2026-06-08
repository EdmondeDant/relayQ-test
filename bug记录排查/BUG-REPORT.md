# RelayQ 文生图 Bug 诊断报告

> 日期: 2025-06-08 | 分析人: 小腾 | 目标: sub2api 底层代码修复

---

## 问题现象

客户通过 WorkBuddy 连接 RelayQ 中转站，文本模型（gpt-5.5/5.4）正常工作，但调用 `gpt-image-2` 文生图时：

- 提交请求后**无任何响应**（挂死直到超时）
- WorkBuddy 短暂弹出错误提示："暂时连不上 server"
- RelayQ 网页后台的「模型测试」功能**能正常出图**
- 使用 OpenAI 官方网页 / 其他渠道测试出图正常

---

## 测试过程与证据

### 1. 渠道与上游配置

| 组件 | 配置 |
|---|---|
| 中转站 | sub2api (RelayQ)，Nginx 反代，服务器 IP: 198.44.178.118 |
| 渠道 | Channel 1 "gpt-image-2"，status: active，group: 2 (openai平台组) |
| 上游账号 | Account 1 "生图gpt"，platform: openai，base_url: https://image.codesonline.dev/v1 |
| 上游连通性 | ✅ 可达，0.6s 响应 |
| `codex_image_generation_bridge` | 测试过 true/false 均无影响 |
| `image_output_price` | 已从 null 修复为 0.04（之前 null 可能影响计费） |

### 2. API 测试结果（使用有效 sk-xxx 完整 key）

| 测试场景 | 结果 | 详情 |
|---|---|---|
| `response_format: "b64_json"` | ✅ **HTTP 200** | 40s 返回，1.9MB 图片数据 |
| `response_format: "url"` | ❌ **挂死无响应** | HTTP 000，30s/60s/120s 均超时 |
| 不指定 `response_format`（默认 url） | ❌ **挂死无响应** | 同上 |
| 聊天接口 `/v1/chat/completions` | ⚠️ 503 | 用户 key 可能不在文本渠道组 |

### 3. 前端代码分析

RelayQ 网页的「模型测试」功能（ModelTestView）中，图片生成的调用代码：

```javascript
// assets/ModelTestView-m_qD0VZe.js (关键代码)
async function Me(e) {
  const t = await fetch(`${W}/v1/images/generations`, {
    method: "POST",
    headers: z(e.apiKey),
    body: JSON.stringify({
      model: e.model,
      prompt: e.prompt,
      n: 1,
      response_format: "b64_json"   // ← 前端强制 b64_json！
    }),
    signal: e.signal
  });
  return await J(t), ((await t.json()).data || []).map(o => ({
    url: o.url || (o.b64_json ? `data:image/png;base64,${o.b64_json}` : ""),
    revisedPrompt: o.revised_prompt
  })).filter(o => o.url)
}
```

**这就是为什么网页测试能出图而 API 不能**：前端强制 `response_format: "b64_json"`，返回后自行渲染；而外部客户端默认发送 `url` 格式请求，RelayQ 后端处理 `url` 格式的代码路径存在 bug。

---

## 根因结论

### 核心 Bug

**sub2api 在处理 `/v1/images/generations` 时，`response_format: "url"` 的代码路径存在阻塞问题。**

| 请求链路 | 状态 |
|---|---|
| Client → RelayQ 鉴权 → JSON 解析 | ✅ 正常 |
| RelayQ → 上游 `image.codesonline.dev` 转发 | ⚠️ 可能正常 |
| 上游返回 url 格式响应 → RelayQ 处理 | ❌ **卡在此处** |
| RelayQ 返回客户端 | ❌ 永不返回 |

### 推测的 Bug 位置

当上游返回如下标准 OpenAI Images Response（url 格式）：

```json
{
  "created": 1234567890,
  "data": [
    {
      "url": "https://image.codesonline.dev/...png",
      "revised_prompt": "..."
    }
  ]
}
```

RelayQ 可能在某些步骤挂死：

1. **图片 URL 代理/下载**：RelayQ 尝试下载返回的图片 URL（用于缓存/防盗链/审查），但连接挂死
2. **响应体读取**：上游返回的 URL 内容过大或格式异常，导致 ReadAll/解析循环
3. **计费计算**：基于图片响应的计费逻辑在 url 格式下触发异常路径
4. **HTTP 客户端复用**：连接池/keep-alive 问题导致读取 response body 时阻塞

### 为什么 b64_json 正常？

`b64_json` 格式返回的是纯 JSON（base64 字符串内嵌），RelayQ 直接透传不经过图片 URL 下载步骤，因此绕过了 bug。

---

## 建议修复方向

### 1. 排查上游响应处理代码

搜索 sub2api 代码中处理 images/generations 响应的逻辑，关键词：
- `images/generations`
- `image_generation`
- `imageOutput`
- `response_format`
- `url` vs `b64_json`

### 2. 检查是否有图片 URL 下载逻辑

```go
// 可能的 bug 代码模式：
resp, err := http.Get(imageURL)  // 下载上游返回的图片 URL
body, _ := io.ReadAll(resp.Body)  // 可能在此挂死
```

### 3. 建议修复方式

- **方案 A（根治）**：为 images 响应处理添加超时（context.WithTimeout），并将 url 响应透明转发给客户端（不下载）
- **方案 B（快速止血）**：在转发前强制将 `response_format` 改为 `b64_json`，收到响应后转换回 url 格式（data: URI）
- **方案 C（兼容）**：如果上游返回错误，正确透传错误信息而不是挂死

### 4. 配置层面

- `codex_image_generation_bridge`: 当前为 false，开启后可能有不同的处理路径，可尝试
- `image_output_price`: Channel 1 之前为 null，已修复为 0.04

---

## 附录：数据备份

修改前的 Channel 1 完整配置已备份，可通过以下命令恢复：

```bash
PUT /api/v1/admin/channels/1
x-api-key: admin-cded401ee465fbc241315d94f63c2dbed0a4c5b28dd506f4f0bb694699407cdf
```

## 附录：快速验证方法

```bash
# b64_json 格式（正常）
curl -X POST https://www.relayq.top/v1/images/generations \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-image-2","prompt":"cat","n":1,"response_format":"b64_json"}'

# url 格式（BUG - 挂死）
curl -X POST https://www.relayq.top/v1/images/generations \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-image-2","prompt":"cat","n":1,"response_format":"url"}'
```
