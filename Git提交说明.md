# 2026-07-23 Git 提交说明

## 建议提交标题

`fix: 修正 Playground 对话助手 key/model 串台并补齐性能优化`

## 提交说明

- 修复 Playground 视频任务成功后未进入 usage / billing 记账链路的问题。
- 为 `grok-imagine-video` 等视频任务补齐成功后的 usage log 与统一扣费逻辑，并使用 `playground-video:<request_id>` 作为幂等 request_id。
- 优化创作记录加载逻辑，移除列表页全量媒体预取与批量 hydrate，降低首屏等待时间。
- 修正 Playground 对话助手的模型筛选逻辑，不再误伤 Anthropic / Claude 可用文本模型。
- 优化 `no available accounts` 的前端提示文案，改为面向用户的可理解错误说明。
- 修正 Playground 对话助手与其它工具共用 `selectedKeyId` 导致的 key/model 串台问题，拆分为独立的对话助手状态。

## 可直接使用的提交正文

```text
fix: 修正 Playground 对话助手 key/model 串台并补齐性能优化

- 修复 Playground 视频任务成功后未进入 usage/billing 记账链路的问题
- 为 grok-imagine-video 等视频任务补充成功后的 usage log 与统一扣费逻辑
- 使用 playground-video:<request_id> 作为视频扣费幂等 request_id，避免重复记账
- 优化创作记录加载逻辑，移除列表页全量媒体预取与批量 hydrate
- 将媒体获取收敛为按需加载，减少记录页首屏等待与额外请求
- 修正对话助手模型筛选逻辑，恢复 Claude 等可用文本模型的正常选择
- 优化 no available accounts 的前端报错提示，改为可理解的用户提示
- 拆分对话助手独立 key/model 状态，避免与其它 Playground 工具共享选择状态
```

## 本轮 Grok/XAI 建议提交标题

`fix: 修复 Grok/XAI 账号配置、模型同步与测试链路`

## 本轮提交说明

- 修复 Grok 添加/编辑账号弹窗错误复用 Anthropic Base URL、API Key placeholder 与提示文案的问题。
- 为 `xai` 平台补齐 API Key 模式默认值与文案，支持正确使用第三方中转站或官方 xAI API。
- 修复上游模型同步链路对 `xai` 的错误平台判断，改为按 OpenAI-compatible 方式处理。
- 修复“同步上游支持模型”错误提示被前端吞掉的问题，支持展示后端返回的真实安全错误摘要。
- 修复账号测试弹窗模型来源逻辑，使测试模型严格以本地白名单同步结果为准。
- 修复 xAI/Grok API Key 账号测试连接误读 `access_token` 的后端问题，改为正确读取 `api_key`。
- 补充 Grok/XAI 相关回归测试，并完成前端类型检查与后端编译验证。

## 本轮可直接使用的提交正文

```text
fix: 修复 Grok/XAI 账号配置、模型同步与测试链路

- 修复 Grok 添加/编辑账号弹窗错误复用 Anthropic 默认 Base URL 与文案的问题
- 为 xai 平台补齐 API Key 模式的 Base URL、placeholder 与提示文案
- 修复 xai 上游模型同步被误判 unsupported 的后端分支逻辑
- 统一透传同步上游模型失败的真实错误摘要，便于定位中转站兼容性问题
- 调整账号测试弹窗模型来源，只展示本地白名单同步进来的模型
- 修复 Grok API Key 测试连接误读 access_token 的后端逻辑，改为正确读取 api_key
- 补充 xai/grok 相关回归测试，并完成前端 typecheck 与后端编译验证
```
