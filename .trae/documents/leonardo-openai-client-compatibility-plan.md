# Leonardo OpenAI 客户端兼容层计划

## Summary

透明转发不能替代 OpenAI 兼容层。Infinite Canvas、ComfyUI 标准 OpenAI 节点发送的是 OpenAI Images 协议，不能原样提交到 Leonardo `/api/rest/v2/generations`。最终采用双模式入口：

- **OpenAI 兼容模式**：客户端发送标准 OpenAI Images 请求；RelayQ 转换为 Leonardo v2 请求，创建异步任务并在服务端等待完成，最终返回标准 OpenAI `{created,data}`。
- **Leonardo 原生模式**：高级客户发送含 `parameters` 的 Leonardo v2 JSON；RelayQ 原样转发 Body，并返回 RelayQ task ID。
- 两种模式共用 RelayQ API Key 鉴权、账号路由、价格、资金预留、幂等、任务持久化、轮询、Webhook 和 unknown 安全处理。

本计划是已批准的 [leonardo-transparent-image-video-proxy-plan.md](./leonardo-transparent-image-video-proxy-plan.md) 的客户端兼容补充。实施时以本计划修正其中“OpenAI 客户端入口”的行为，不取消原生透明转发。

## Current State Analysis

- 路由已经根据 API Key 所属 Group 将 Leonardo 客户送入 [leonardo_media_handler.go](../../backend/internal/handler/leonardo_media_handler.go)，不依赖 User-Agent 或客户端名称。
- `/v1/images/generations` 已能根据顶层 `parameters` 自动区分 Leonardo Raw 与 OpenAI JSON。
- Raw 分支固定返回 HTTP 202 RelayQ task；这适合原生 API 客户，不适合只接受阻塞 OpenAI JSON 的通用软件。
- OpenAI 分支默认同步等待并返回 OpenAI Images JSON，但当前只允许 `896×896`、`n=1`，默认 `response_format=url`，并强制 `Idempotency-Key`。
- 典型 ComfyUI/Infinite Canvas OpenAI 节点通常：
  - 发送顶层 `model/prompt/size/n/quality/response_format`；
  - 期待一次 HTTP 请求返回最终 `{created,data}`；
  - 不会发送 Leonardo `parameters/guidances`；
  - 不一定允许配置 `Idempotency-Key`；
  - 无法表达 Content、Style、Character 等 Leonardo 专用参考图语义。
- [waitLeonardoOpenAIImages](../../backend/internal/handler/leonardo_media_handler.go) 当前只跟随请求 Context，没有独立 900 秒上限。
- 服务端 HTTP 层没有设置全局 WriteTimeout，技术上允许长响应；但反向代理和客户端仍可能更早断开。
- 已存在 OpenAI `url`/`b64_json` 输出转换能力；`b64_json` 会安全下载 Leonardo 输出后内联。

## Product Decisions

### 客户端协议

- `/v1/images/generations` 继续自动识别：
  - 有 `parameters`：Leonardo 原生 Raw。
  - 无 `parameters` 且有顶层 `prompt`：OpenAI 兼容。
  - 两套字段混用：HTTP 400。
- 普通 OpenAI 客户无需了解 Leonardo 是异步 API；RelayQ 隐藏创建和轮询过程。
- 原生 Raw 请求保持任务响应，不伪装成同步 OpenAI 响应。

### OpenAI 同步行为

- 默认同步；保留 RelayQ 扩展 `async:true` 给支持任务轮询的客户。
- 同步最长等待 **900 秒**。
- 900 秒内成功：HTTP 200 标准 OpenAI Images JSON。
- 明确失败：返回 OpenAI 风格错误。
- 达到 900 秒或客户端断线：上游任务继续运行和正常计费，不自动取消、不重复提交。
- 创建任务后立即设置响应头 `X-RelayQ-Task-ID` 和 `X-Request-ID`；即使连接随后断开，客户可用 task ID 查询 `/v1/media/generations/:id` 找回结果。
- 超时响应使用 HTTP 504，并在 JSON error 及 `X-RelayQ-Task-ID` 中返回可恢复任务 ID。

### 自动幂等

- 客户显式提供 `Idempotency-Key` 时继续严格使用。
- 未提供时，RelayQ 生成内部键：
  - 输入为 user ID、API key ID、规范化路由、完整请求 Body 哈希和 5 分钟时间桶。
  - 同一主体、同一请求在同一 5 分钟窗口内复用同一任务。
  - 不把自动生成键写回或要求客户端保存。
- 自动幂等 TTL 至少覆盖 5 分钟窗口和 900 秒同步等待，任务本身仍由 generation job 永久标识。
- 显式键不受 5 分钟窗口影响，继续使用系统标准幂等 TTL。
- Body、quality、response_format、参考图内容任一变化必须产生不同指纹。

### 尺寸与质量

- OpenAI `size` 映射产品档位：
  - `1024x1024` → Small。
  - `2048x2048` → Medium。
  - `2880x2880` → Large。
- 模型原生无法达到目标尺寸时，按主计划调用 Leonardo 官方 Pro Upscaler Precise；最终输出必须严格达到请求尺寸，不能静默降级。
- `quality` 接受 `low|medium|high`；缺失默认 `low`。
- quality 只表示 RelayQ 产品档位；每个模型按官方真实 `quality/mode` 映射。没有官方质量字段的模型不能向上游伪造该字段。
- `n` 仍只开放价格表、模型能力和任务结算都已验证的数量；首批保持 `n=1`。

### 输出格式

- 未传 `response_format` 时默认 `b64_json`，提高 ComfyUI/Infinite Canvas 对临时 URL、跨域和 URL 过期的兼容性。
- `response_format=b64_json` 返回 `data[].b64_json`。
- `response_format=url` 返回经过安全校验的 Leonardo HTTPS URL；后续可增加 RelayQ 稳定代理 URL，但不作为本阶段前置条件。
- 返回结构保持标准 OpenAI Images：

```json
{
  "created": 1785571200,
  "data": [
    {"b64_json": "..."}
  ]
}
```

### 高级参考图能力

- 标准 OpenAI 文生图客户端只保证文生图。
- `/v1/images/edits` 用于标准 OpenAI 单图编辑，但不擅自把 image 固定解释为 Leonardo Content 或 Style。
- Content/Style/Character 等高级能力通过：
  - Leonardo 原生 Raw API；或
  - RelayQ 专用 ComfyUI 节点。
- 专用 ComfyUI 节点明确提供 reference type、strength、图片源、quality、size 和 async/sync 控件，再构造 Leonardo Raw 或 RelayQ 扩展请求。
- Infinite Canvas 若只支持标准 OpenAI 字段，则只获得标准文生图/编辑能力；除非其扩展机制能加载 RelayQ 专用适配器。

## Proposed Changes

### 1. 完善 OpenAI Images 请求 DTO 与转换

修改 [leonardo_media_handler.go](../../backend/internal/handler/leonardo_media_handler.go)：

- `quality` 接受 `low|medium|high`，默认 low。
- `size` 接受 1024、2048、2880 三个产品档位，移除固定 896 限制。
- `response_format` 缺失默认 `b64_json`。
- OpenAI 分支根据 Verified Registry 生成 Leonardo typed 请求；不把 OpenAI Body 原样发给 Leonardo。
- Raw 分支保持原样转发，不受 OpenAI 默认值影响。
- OpenAI 转换后的最终 Leonardo payload 和原 OpenAI 请求共同进入任务审计与幂等指纹。

### 2. 增加自动幂等键

在 Handler 附近增加最小 helper，不修改全局幂等语义：

- 显式 Header 非空：使用现有值和标准 TTL。
- Header 缺失：按 `SHA-256(userID + apiKeyID + route + rawBodySHA256 + floor(now/5m))` 生成 ASCII 内部键。
- 自动键使用独立 scope 后缀，避免与客户显式键碰撞。
- 单元测试使用注入时钟或纯函数时间参数验证时间桶边界。
- 不关闭现有 request fingerprint 冲突检查。

### 3. 增加 900 秒同步等待和任务找回

修改 [waitLeonardoOpenAIImages](../../backend/internal/handler/leonardo_media_handler.go) 及调用处：

- 创建任务成功后、等待前设置 `X-RelayQ-Task-ID`。
- 使用 `context.WithTimeout(requestContext, 900*time.Second)`。
- 每秒轮询现有 Get Service，不新增第二套轮询逻辑。
- 超时返回带 task ID 的 OpenAI error；任务不会取消。
- 请求 Context 取消后停止当前 HTTP 等待，但后台 poll runner 继续推进任务。
- 记录同步等待耗时、超时和客户端取消指标，不记录 prompt 或图片数据。

### 4. 实现产品尺寸映射

扩展 [models.go](../../backend/internal/pkg/leonardo/models.go)、[leonardo_image_pricing.go](../../backend/internal/service/leonardo_image_pricing.go) 和图片创建编排：

- Registry 为每个模型记录 Small/Medium/Large 的原生尺寸或 Upscale 方案。
- 首批 FLUX Schnell：按官方原生尺寸上限决定 Small/Medium；Large 使用已验证的 Pro Upscaler Precise 两阶段任务。
- 两阶段任务必须共享一个 RelayQ public task，保存父 generation ID 和 upscaler generation ID，并按总上游成本 ×7.1 计费。
- 任一阶段明确失败按资金规则退款；任一提交结果 unknown 进入 manual_review。
- 在 Upscaler 协议和价格真实探针完成前，对需要 Upscale 的档位 fail closed，不返回伪尺寸。

### 5. 输出兼容和错误格式

- 成功只返回 OpenAI `created/data`，不混入 RelayQ task 字段；task ID 放响应头。
- 异步扩展仍返回 202 `{created,task_id,status}`。
- 错误保持 `{error:{message,type,param,code}}`。
- 超时、余额不足、模型不支持、尺寸不支持、上游 submission unknown 使用稳定且不同的 error code。
- `submission_unknown` 不能提示客户立即重试；错误中说明可用 task ID 查询。

### 6. RelayQ ComfyUI 专用节点

单独建立最小交付物，不耦合后端核心：

- 一个文生图节点：标准 OpenAI 请求。
- 一个 Leonardo Advanced 节点：Content/Style、strength、source、quality、size。
- 节点默认使用同步 OpenAI 模式；高级节点可选择异步并轮询 task。
- API Key 只保存在 ComfyUI 本地配置/凭据输入，不写 workflow 明文字段或日志。
- 节点实现安排在后端 OpenAI/Raw 双模式稳定之后；不阻塞基础 API 上线。

## Failure Modes

- 客户端只会 OpenAI：始终进入转换分支，不会把 OpenAI JSON误发给 Leonardo。
- 客户 Body 含 `parameters` 又含顶层 OpenAI字段：400，不猜协议。
- 客户缺少 Idempotency-Key：自动键保护网络重试；跨 5 分钟窗口的重试可能成为新任务，因此超时错误必须返回 task ID。
- 900 秒之前代理断开：后台任务继续；客户端通过 task ID找回。
- 客户软件不显示响应头：可使用 RelayQ 请求日志中的 request ID关联任务；管理/API 查询必须限制为原用户/API Key/Group。
- base64 输出过大：受现有 50 MiB下载上限和请求链路限制；超限返回明确错误，任务本身仍保持成功和已结算，不重新生成。
- Large 需要 Upscaler 但该模型方案未验证：提交前拒绝，不先生成再失败。
- 标准 OpenAI客户端无法表达 Style/Content：不做隐式猜测，要求专用节点或原生 API。

## Verification

### 单元测试

- OpenAI 与 Raw 自动识别及混合冲突。
- OpenAI请求不会原样发送上游；上游收到正确 Leonardo `parameters`。
- Raw 请求未知字段保持原样。
- size 三档和 quality 三档映射；缺省 low、缺省 b64_json。
- 自动幂等：同一 5 分钟桶复用、跨桶生成新键、用户/API Key隔离、显式键优先。
- 同步成功、明确失败、900 秒超时、客户端取消和 task header。
- URL 与 b64_json 输出。
- 不支持的 Upscale 请求在付费提交前拒绝。

### 集成测试

- 使用模拟 ComfyUI 请求（无 Idempotency-Key、无 async、无 response_format）验证最终 HTTP 200 和 `data[].b64_json`。
- 使用模拟 Infinite Canvas 请求验证 `url` 和 b64 两种输出。
- 验证同步等待期间任务已经持久化、余额只扣一次。
- 模拟客户端断线后由后台 Poller 完成任务，再通过 task ID查询成功。
- 验证相同请求在 5 分钟内不会重复调用 Leonardo。
- 回归 Leonardo Raw、Media、Images Edits 和非 Leonardo OpenAI/xAI 路由。

### 真实验收

- 使用实际 ComfyUI OpenAI 节点连接 RelayQ Base URL 和 RelayQ API Key，执行一次最低成本 Small/low 文生图。
- 使用 Infinite Canvas 或其等价 OpenAI 客户端执行同一兼容测试。
- 每个真实付费 POST 前继续单独向用户确认。
- 高级 Content/Style 使用原生 Raw API 验证；专用 ComfyUI 节点完成后再做节点级付费验收。

## Assumptions & Decisions

- 通用 OpenAI客户端优先兼容，不要求理解 Leonardo 异步协议。
- 默认同步，最大 900 秒；`async:true` 是 RelayQ 扩展。
- 缺少 Idempotency-Key 时使用 5 分钟自动幂等窗口。
- 缺省 response format 为 b64_json。
- size 采用 Small/Medium/Large 产品映射。
- quality 只接受 low/medium/high，默认 low。
- 断线不取消上游任务；任务必须可找回。
- 标准客户端不隐式获得 Content/Style；高级能力通过原生 API或专用 ComfyUI节点。
- 原生透明转发继续存在，但它不是 OpenAI客户端的执行路径。
- 不自动执行任何真实付费请求；逐次获得用户确认。
