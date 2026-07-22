[OPEN] proxy-test-failure

## 症状
- 账户配置了 IP 代理，代理本身联通。
- 执行“连接测试”时报错：`Chat Completions API (/v1/chat/completions) request failed: Post "https://api.x.ai/v1/chat/completions": read tcp ...: wsarecv: An existing connection was forcibly closed by the remote host.`

## 期望
- 连接测试应通过代理成功访问目标模型接口，或至少返回更准确的失败原因。

## 当前可证伪假设
1. 连接测试请求未实际使用账户配置的代理。
2. 代理协议与测试代码使用的代理拨号方式不匹配。
3. 请求目标、请求头或模型提供商映射错误，导致远端主动断开连接。
4. Go HTTP 客户端的 TLS/HTTP2/连接复用与当前代理出口不兼容。
5. 表单配置未生效，测试时读到了其他层级的默认配置。

## 调试计划
1. 定位前端“连接测试”触发点和后端 API。
2. 定位后端实际发起 xAI 请求与代理装配位置。
3. 加最小化日志，确认测试请求最终使用的 provider、base URL、proxy、transport 配置。
4. 复现一次并对照日志排除假设。

## 已收集证据
- 账号 ID：3，账号名：`grok7.12 5日抛2`
- 页面显示代理：`http://206.53.57.7:12323`
- 复现模型：`grok-4.3`
- 埋点日志文件：`.dbg/trae-debug-log-proxy-test-failure.ndjson`
- 对照验证：
  - 同一代理访问 `https://example.com/` 返回 `200`
  - 同一代理访问 `https://api.x.ai/v1/models` 返回 `Connection was reset`

## 假设验证
| ID | 假设 | 结论 | 证据 |
|----|------|------|------|
| A | 连接测试请求未实际使用账户配置的代理 | 否 | 日志 1 显示 `proxy_url=http://...@206.53.57.7:12323`，测试请求已组装代理 |
| B | 代理协议与测试代码使用的代理拨号方式不匹配 | 部分否 | 日志 2/3 显示代理被识别为 `http`，并按 OpenAI profile 进入 `openai_h2` 路径，不是“未识别代理协议” |
| C | 请求目标、请求头或模型提供商映射错误 | 否 | 日志 1/2 显示目标为 `https://api.x.ai/v1/chat/completions`，模型为 `grok-4.3`，与页面一致 |
| D | 代理出口到 `api.x.ai` 的链路被远端或代理中间节点重置 | 是 | 日志 4/5 为 `wsarecv: An existing connection was forcibly closed by the remote host`；同代理访问 `example.com` 正常、访问 `api.x.ai` 复现 reset |
| E | 表单配置未生效，测试时读到了其他层级默认配置 | 否 | 页面账号 ID、代理和后端埋点一致，没有配置串号迹象 |

## 当前结论
- 这不是“连接测试没走代理”。
- 问题集中在：**该 HTTP 代理到 `api.x.ai` 的链路会被重置**。
- 结合当前实现，测试请求命中的是 `openai_h2` 模式；但更强证据是即使用系统 `curl --http1.1` 通过同一代理访问 `api.x.ai` 也会 reset，说明至少当前这条代理出口对 `api.x.ai` 本身不通或被中间节点/目标侧拒绝。
