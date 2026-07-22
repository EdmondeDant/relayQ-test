# 2026-07-22 Git 提交说明

## 建议提交标题

`fix: 补齐 Playground 扣费并优化创作记录与对话助手体验`

## 提交说明

- 修复 Playground 视频任务成功后未进入 usage / billing 记账链路的问题。
- 为 `grok-imagine-video` 等视频任务补齐成功后的 usage log 与统一扣费逻辑，并使用 `playground-video:<request_id>` 作为幂等 request_id。
- 优化创作记录加载逻辑，移除列表页全量媒体预取与批量 hydrate，降低首屏等待时间。
- 修正 Playground 对话助手的模型筛选逻辑，不再误伤 Anthropic / Claude 可用文本模型。
- 优化 `no available accounts` 的前端提示文案，改为面向用户的可理解错误说明。

## 可直接使用的提交正文

```text
fix: 补齐 Playground 扣费并优化创作记录与对话助手体验

- 修复 Playground 视频任务成功后未进入 usage/billing 记账链路的问题
- 为 grok-imagine-video 等视频任务补充成功后的 usage log 与统一扣费逻辑
- 使用 playground-video:<request_id> 作为视频扣费幂等 request_id，避免重复记账
- 优化创作记录加载逻辑，移除列表页全量媒体预取与批量 hydrate
- 将媒体获取收敛为按需加载，减少记录页首屏等待与额外请求
- 修正对话助手模型筛选逻辑，恢复 Claude 等可用文本模型的正常选择
- 优化 no available accounts 的前端报错提示，改为可理解的用户提示
```
