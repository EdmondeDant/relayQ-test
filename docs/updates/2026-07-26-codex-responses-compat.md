# 2026-07-26 Codex Responses 多轮兼容修复

## 问题

新版 Codex 会在多轮历史中回放本地生成的 `item_*` ID。OpenAI Responses/Codex OAuth 上游要求：

- `message.id` 使用 `msg*` 前缀；
- function/tool call 输入项的 `id` 使用 `fc*` 前缀；
- `store=false` 时不可重新引用未持久化的 `rs_*` reasoning ID。

RelayQ 原先会把这些上游 400 参数错误包装成 502，导致客户端反复重试同一份坏上下文。

## 修复

- 在 Responses 入站清洗层删除不符合 item 类型前缀约束的 ID，不伪造新 ID；
- 保留合法 `msg*`/`fc*` ID及 `function_call_output.call_id` 配对；
- reasoning item 只删除不可重放的 `rs_* id`，保留 `encrypted_content`，缺失 `summary` 时补空数组；
- 对明确的 OpenAI `400/404/409/422 invalid_request_error` 透传原状态和错误体，不再伪装成 502；
- namespace 清洗继续保留。

## 验证范围

- Codex 多轮 message ID；
- function call / function output 配对；
- stateless encrypted reasoning；
- 普通与 passthrough Responses 错误透传；
- `gpt-5.6-sol` 流式生产请求。

参考上游：Wei-Shaw/sub2api #3981、commits `4d4ba64b`、`73de2ea7`、`c5d9d579`。
