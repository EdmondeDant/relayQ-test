# Debug Session: chat-assistant-doubao-label [OPEN]

## Symptom
- 对话助手页面无论选择什么模型，回复都自称或暗示自己是“豆包”。

## Scope
- 先做证据收集，不改业务逻辑。
- 重点排查前端默认消息、快捷场景提示词、后端系统提示词、模型路由注入文案。

## Initial Hypotheses
1. 前端欢迎语或默认首条消息把“豆包”写死了，用户误以为是模型自述。
2. 对话助手的快捷场景/系统提示词模板里写死了“豆包”。
3. 后端在对话助手接口里追加了固定 persona/system prompt，包含“豆包”。
4. 某个模型映射或渠道层把品牌自我介绍注入到了请求中，与所选模型无关。
5. 前端展示拿错了旧会话消息，导致看起来所有模型都在说“豆包”。

## Evidence Log
- 静态代码证据：
- 全局搜索 `豆包|Doubao|doubao` 未发现对话助手前端文案、快捷场景或后端 `chat` 提示词中写死“豆包”。
- 前端发送逻辑 [PlaygroundView.vue](file:///c:/Users/Administrator/.openclaw/workspace/realyq-test/frontend/src/views/user/PlaygroundView.vue#L785-L803) 只是把 `chatMessages` 原样提交。
- 后端 `chat` 构造逻辑 [playground_job_payload.go](file:///c:/Users/Administrator/.openclaw/workspace/realyq-test/backend/internal/service/playground_job_payload.go#L298-L345) 对 `kind=chat` 不注入任何 system prompt；仅在 `copywriting` 写入固定系统提示词。
- 运行时记录证据：
- 最新对话助手任务 `id=21`，记录显示：
- `model = grok-4.5`
- `request_payload.messages = [{role:user, content:\"你是什么模型？\"}]`
- `metadata.platform = xai`
- `metadata.group_name = grok`
- `result_payload.content = 我是豆包，是由字节跳动公司开发的人工智能助手。`
- 结论：前端和 Playground 后端任务层没有把“豆包”写死；“豆包”文本来自更下游的真实模型响应或账号路由结果。
- 账号配置证据：
- 管理后台账户列表显示当前 `grok` 使用的是 `account id=4`, `platform=xai`, `type=oauth`。
- 该任务记录也标记为 `platform = xai`，说明 Playground 至少在调度层选择的是 xAI 账号，不是直接选到 MiMo/OpenAI 分组。

## Next Step
- 若继续深挖，需要在 OpenAI/xAI 转发链路增加最小埋点，抓取该请求最终命中的上游账号、上游模型和返回体摘要，确认是：
- xAI 上游真实回了“豆包”，还是
- OAuth/转发链路实际被错误路由到了别的兼容上游。
