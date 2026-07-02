# RelayQ xAI OAuth 集成设计文档

> 创建日期：2026-07-02  
> 版本：v1.0  
> 作者：小腾（基于 d68267d6 提交）

---

## 一、背景与目标

### 1.1 为什么需要 xAI OAuth？

- Grok 模型（`grok-4.3`、`grok-imagine-video-1.5-preview` 等）需要通过 xAI 官方 OAuth 账号体系访问
- 现有 OpenAI OAuth 体系（`chatgpt_account_id` + session）无法直接复用
- 需要支持：账号添加、重授权、踢下线、负载均衡、session 刷新

### 1.2 设计目标

1. **统一账号模型**：`Account` 表通过 `platform` 字段区分 `openai` / `xai`
2. **OAuth 流程一致性**：前端授权流程、回调处理、错误提示尽量与 OpenAI OAuth 保持一致
3. **Session 同步机制**：xAI OAuth 账号的 access_token / refresh_token 需要与上游 session 保持同步
4. **异常处理矩阵**：401/403/429 等错误码的处理策略明确

---

## 二、核心数据结构

### 2.1 Account 表新增字段（逻辑层面）

```go
type Account struct {
    ID           int64
    Platform     string            // "openai" | "xai" | ...
    Type         string            // "api_key" | "oauth"
    Credentials  map[string]any    // xai: { "access_token", "refresh_token", "xai_org_id" }
    // ... 其他字段
}
```

### 2.2 xAI OAuth Credentials 结构

```json
{
  "access_token": "xai-...",
  "refresh_token": "xai-refresh-...",
  "xai_org_id": "org_xxx",
  "expires_at": 1752000000,
  "scope": "grok chat completions"
}
```

---

## 三、OAuth 授权流程

### 3.1 前端流程（`useXAIOAuth.ts`）

```
用户点击「添加 xAI OAuth 账号」
  → 打开 /oauth/xai/authorize（后端生成 state + PKCE）
  → 重定向到 xAI 官方授权页
  → 用户授权后回调 /oauth/xai/callback?code=...&state=...
  → 前端调用 create-xai-oauth-account 接口
  → 后端换取 access_token / refresh_token 并落库
```

### 3.2 后端关键接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/admin/xai/oauth/authorize` | 生成 state，返回授权 URL |
| POST | `/api/v1/admin/xai/oauth/callback` | 换 token、创建账号 |
| POST | `/api/v1/admin/accounts/{id}/reauth` | 重授权（刷新 token） |
| POST | `/api/v1/admin/accounts/{id}/logout` | 踢下线（使 session 失效） |

---

## 四、Session 同步与踢下线

### 4.1 同步时机

- **创建账号时**：首次换 token 成功即建立 session
- **每次请求前**：检查 `expires_at`，若剩余 < 5 分钟则触发刷新
- **收到 401 时**：立即尝试 refresh_token，若失败则标记账号不可用并通知管理员

### 4.2 踢下线机制

- 管理员在「账号详情」页点击「强制下线」
- 后端调用 xAI `/oauth/revoke` 接口
- 本地清除 `access_token` / `refresh_token`，账号状态改为 `disabled`

---

## 五、负载均衡与平台判断

### 5.1 平台判断逻辑

```go
func (a *Account) IsXAI() bool {
    return a.Platform == PlatformXAI || strings.HasPrefix(a.Name, "xai-")
}
```

### 5.2 路由分发

- `openai_gateway_xai_oauth.go` 负责把 xAI OAuth 账号的请求转发到 `https://api.x.ai/v1/...`
- 普通 OpenAI 账号继续走原有路径

---

## 六、异常处理矩阵

| 错误码 | 场景 | 处理策略 | 是否告警 |
|--------|------|----------|----------|
| 401 | token 过期 | 自动 refresh，失败则禁用账号 | ✅ |
| 403 | 账号被封 / 权限不足 | 立即禁用账号 + 通知管理员 | ✅ |
| 429 | 速率限制 | 退避重试，记录到 `ratelimit_service` | ❌ |
| 5xx | 上游故障 | 标记账号临时不可用，30 分钟后重试 | ✅ |

---

## 七、测试覆盖

### 7.1 已有测试

- `xai_oauth_service_test.go`（待补充）
- `openai_gateway_xai_oauth_test.go`（待补充）

### 7.2 建议补充的测试场景

1. refresh_token 过期后自动禁用账号
2. 并发请求时只刷新一次 token
3. 踢下线后该账号的后续请求全部 401

---

## 八、运维 checklist

- [ ] xAI OAuth 账号的 `refresh_token` 必须持久化（已做）
- [ ] 监控面板增加「xAI 账号健康度」指标
- [ ] 每周巡检被禁用的 xAI 账号，确认是否需要人工介入
- [ ] 备份脚本已包含 `xai_oauth` 相关表（确认中）

---

## 九、已知风险与后续优化

| 风险 | 缓解措施 | 状态 |
|------|----------|------|
| xAI 上游接口变更 | 抽象 `XAIClient` 接口，版本化 | 进行中 |
| OAuth state 泄露 | state 使用 Redis 短时缓存 + 签名 | 已实现 |
| 并发刷新 token 导致 race | 使用 `sync.Once` + Redis 分布式锁 | 待实现 |

---

## 十、参考资料

- xAI OAuth 官方文档（内部）
- OpenAI OAuth 集成历史文档（`docs/OAUTH_INTEGRATION.md` 待创建）
- 提交 `d68267d6` 的完整 diff

---

**文档状态**：v1.0（首次发布）  
**下次更新触发条件**：xAI OAuth 流程有重大变更或发现生产事故

---

*文档由小腾生成，基于实际代码 + 部署经验整理，供后续维护者参考。*