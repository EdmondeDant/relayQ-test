# Leonardo 图片与视频透明中转改造计划

## Summary

将 RelayQ 的 Leonardo 链路从“为每个模型重建一遍上游参数协议”调整为“透明转发 Leonardo 原生 JSON Body，RelayQ 只负责路由、鉴权、账号选择、上传、计费、幂等和任务状态”。

目标形态：

- `/v1/images/generations` 自动识别 OpenAI Images 请求与 Leonardo v2 原生请求。
- `/v1/videos`、`/v1/videos/generations` 自动识别兼容视频请求与 Leonardo v2 原生请求。
- Leonardo 原生请求的 JSON Body 在不需要代上传时按原始字节提交至固定的 `/api/rest/v2/generations`。
- 需要 RelayQ 代上传时，只删除 RelayQ 扩展的 `image.source` 并写入 Leonardo `image.id/type=UPLOADED`，其余字段和值不改写。
- 创建响应继续返回 RelayQ task ID，由现有任务中心负责轮询、Webhook、结算、退款、重启恢复和用户隔离。
- 客户价格仍由服务端决定：本地上游成本 × 7.1；精确价格缺失时，按同媒体类型、同官方能力档位的最高本地价格直接结算，并向客户披露 `pricing_match_type=quality_tier_max`。
- 图片与视频均采用该架构；视频必须完成一个模型的真实创建、轮询、价格探针后才允许灰度开启。

## Current State Analysis

### 当前请求不是透明转发

- [leonardo_media_handler.go](../../backend/internal/handler/leonardo_media_handler.go) 使用固定 DTO 和 `DisallowUnknownFields`，Leonardo 新增字段会被拒绝。
- [leonardo_image_create_orchestrator.go](../../backend/internal/service/leonardo_image_create_orchestrator.go) 重新构造 `prompt/width/height/quantity/guidances`，客户未建模的合法参数会丢失。
- [client.go](../../backend/internal/pkg/leonardo/client.go) 只接受 typed request 并重新 Marshal，不能按原始字节提交。
- 当前 FLUX Schnell `content/style` Registry 和 builder 能正确生成已知协议，但不应再作为“允许原生参数通过”的必要条件；它们只保留给 OpenAI 转换和 RelayQ 代上传校验。

### 必须保留的中转站职责

- API Key、用户、Group 和 Leonardo 平台绑定校验。
- 固定上游 Base URL、替换为 RelayQ 持有的 Leonardo Authorization；不透传客户端 Header、Host、Cookie 或上游 URL。
- 请求大小限制、远程图片 SSRF 防护、MIME/魔数和文件大小校验。
- `Idempotency-Key`、完整请求指纹、提交前资金预留、任务先落库。
- POST 写出后状态不确定时不自动重试、不切换账号，进入 `unknown/manual_review`。
- 轮询、Webhook、重启恢复、输出审核、用户隔离、成功结算和明确失败退款。

### 图片现状

- Verified Registry 目前只有 FLUX Schnell。
- 价格只覆盖 `flux-schnell / 896×896 / quantity=1 / private = $0.003`。
- 图片创建、轮询、任务查询和结算已有闭环。
- multipart、Data URI、远程 URL 的安全读取以及 Leonardo `init-image` 上传能力已经存在，可直接复用。

### 视频现状

- 视频路由存在，但 Leonardo Group 当前被 [openai_videos.go](../../backend/internal/handler/openai_videos.go) 显式拒绝。
- Verified Registry 没有视频模型。
- [leonardo_video_pricing.go](../../backend/internal/service/leonardo_video_pricing.go) 对所有请求 fail closed。
- Leonardo generation response 只解析 `generated_images`；视频输出 Schema 未验证。
- Poller、Webhook、Media Get 和内容代理均显式拒绝 video。
- `video_enabled` 当前只是路由可见开关，不代表视频链路可用。

## API Decisions

### 1. 自动协议识别

`/v1/images/generations` 和 Leonardo Group 下的视频创建入口按以下规则识别：

- JSON 顶层存在对象类型 `parameters`，并且不存在 OpenAI/兼容协议的顶层业务字段组合时，识别为 Leonardo v2 原生请求。
- 不存在 `parameters`，但存在兼容协议所需的顶层 `prompt` 时，识别为 OpenAI/兼容请求，继续执行现有转换。
- 同时出现 Leonardo `parameters` 与 OpenAI 顶层 `prompt/size/n/quality` 等冲突字段时返回 HTTP 400，不猜测优先级。
- `/v1/media/generations` 新增对 Leonardo 原生 Body 的透明模式；现有带 `modality` 和顶层 `prompt` 的 RelayQ legacy envelope 暂时兼容并继续转换，避免破坏已有客户。
- 原生请求至少要求 `model`、`public`（可缺省为 false）和对象类型 `parameters`；用于计费和审计的字段从原始 JSON旁路读取，不据此重建 Body。

### 2. 透明边界

- 无代上传：把客户 JSON 原始字节提交给 Leonardo，保持字段顺序、空白和数字表达。
- 仅转发 Body；客户端 Header 不透明转发。
- RelayQ 总是覆盖 Authorization、Accept、Content-Type、Host 和目标 URL。
- 模型必须存在于 Verified Model Registry，账号也必须声明支持该模型。
- Registry 不再枚举所有 Leonardo 参数；只记录模型身份、modality、官方质量/尺寸能力、计价提取规则和需要 RelayQ 代上传时允许修改的 JSON 路径。

### 3. 参考图和代上传

支持三种 `image.source`：

```json
{
  "image": {
    "source": "data:image/png;base64,..."
  }
}
```

```json
{
  "image": {
    "source": "https://public.example/reference.png"
  }
}
```

```json
{
  "image": {
    "source": "multipart://content_1"
  }
}
```

处理规则：

- `source` 是 RelayQ 扩展字段，不发送给 Leonardo。
- Data URI、URL 和 multipart 文件继续复用现有 20 MiB、JPEG/PNG/WebP、MIME/魔数、SSRF 和重定向限制。
- 每个 Verified Model 声明可代上传的精确 JSON 路径；FLUX Schnell 首批只允许 `parameters.guidances.content[*].image` 与 `parameters.guidances.style[*].image`。
- `source` 与现有 `id` 同时出现时返回 400，避免覆盖客户 ID。
- 上传成功后仅把对应对象替换为 `{"id":"<uploaded-id>","type":"UPLOADED"}`；同级非 RelayQ 字段保持不变。
- URL/Data URI 可用于 JSON 请求；multipart 请求使用一个 `request` JSON part 和若干由 `multipart://字段名` 引用的文件 part。
- 上传发生在账号选定之后，因为 uploaded image ID 绑定上游 Leonardo 账号；上传缓存继续按 `accountID + SHA-256` 隔离。
- 已有 Leonardo 图片 ID 可直接原样透传；后续必须增加生成图片归属校验，禁止跨 RelayQ 用户引用本站历史任务产生的 ID。外部 Leonardo ID 无法证明归属时，仅在显式配置允许的账号范围内使用并记录审计。

### 4. 质量档位与计费

- 透明原生请求使用 `X-RelayQ-Quality: low|medium|high` 指定客户计费档位；该 Header 不发送上游。
- Header 缺失时默认 `low`，即最低、最便宜档位。
- OpenAI/兼容请求如自身提供合法 quality，则使用请求值；缺失同样默认 low。
- 档位按模型官方能力映射，不按价格简单三等分：
  - 图片优先映射官方 `quality/mode`；没有官方质量参数的模型按官方尺寸/计算规格映射，仍不能伪造上游 quality 字段。
  - 视频按官方分辨率和模式映射，例如 Standard→low、HD→medium、Full HD/4K→high；具体映射存入 Registry/价格快照。
- 存在精确价格：`customer_cost = exact_upstream_cost × 7.1`，`pricing_match_type=exact`。
- 精确价格不存在：取“同 modality + 同 quality tier”价格池的最高本地上游成本，乘 7.1 后预扣并作为最终结算，`pricing_match_type=quality_tier_max`。
- 如果对应媒体和档位的价格池为空，拒绝请求且不调用上游；不能无上限猜价。
- 创建和任务查询响应增加 `pricing_match_type`、`quality_tier`、`customer_cost_usd`，明确披露最高价结算。
- 完整原始 Body（代上传后使用定点改写后的最终 Body）、quality tier 和最终服务端报价共同进入幂等指纹。

### 5. 付费探针规则

- 每个真实付费 POST 前单独向用户确认，不设自动总预算。
- 每个已写出的未知 POST 绝不自动重放。
- 图片先验证 FLUX Schnell：纯原生透传、content、style 各选择必要的最小探针。
- 视频先从价格最低、时长最短且 Registry/模型目录可确认的单一模型开始。
- 视频真实创建、轮询输出 Schema、成本字段和内容 URL 未验证前，`video_enabled` 保持关闭。

## Proposed Changes

### Phase A：通用 Raw 提交内核

1. 修改 [client.go](../../backend/internal/pkg/leonardo/client.go)：
   - 增加 `CreateGenerationRaw(ctx, []byte)`。
   - typed `CreateGeneration` 只负责 Marshal，随后复用 Raw 内核。
   - Raw 内核继续执行现有 Authorization 注入、固定 `/v2/generations`、响应大小限制、generation ID 多 envelope 解析、成本宽松解析和 `httptrace.WroteRequest` unknown 判定。
2. 修改 [leonardo_generation_adapter.go](../../backend/internal/service/leonardo_generation_adapter.go) 和 [leonardo_generation_service.go](../../backend/internal/service/leonardo_generation_service.go)：
   - 扩展 client 接口以支持 Raw 创建。
   - Generation Service 接收最终上游 Body 和经过脱敏的审计 payload。
   - 原始 Body 仅用于即时提交和哈希，不直接落库；任务中只保存递归脱敏后的 JSON 对象。
3. 为 Raw Client 增加测试：未知字段和数字表达原样保留、Authorization 不透传、写出后超时 unknown、非 2xx 不重试、响应解析兼容。

### Phase B：请求识别和图片透明转发

1. 修改 [leonardo_media_handler.go](../../backend/internal/handler/leonardo_media_handler.go)：
   - 一次性受限读取 Body。
   - 增加协议探测器，只提取 model、parameters、计价字段，不使用 `DisallowUnknownFields` 限制 Leonardo 原生参数。
   - `/v1/images/generations` 自动选择 OpenAI 转换分支或 Leonardo Raw 分支。
   - 冲突协议返回稳定 400 错误码。
   - 支持 JSON 与 multipart `request` part。
2. 修改 [leonardo_media_create_service.go](../../backend/internal/service/leonardo_media_create_service.go)：
   - 输入增加 `Modality`、`Protocol`、`RawBody`、`QualityTier` 和 upload sources。
   - 使用最终上游 Body 哈希生成请求指纹；保留稳定 public ID 和账号白名单。
3. 修改 [leonardo_image_create_orchestrator.go](../../backend/internal/service/leonardo_image_create_orchestrator.go)：
   - Raw 分支不再构造 parameters。
   - OpenAI/legacy 分支继续复用现有构造逻辑。
   - 两个分支复用报价、资金预留、任务持久化和 unknown 处理。
4. 保留 [leonardo_image_reference.go](../../backend/internal/service/leonardo_image_reference.go) 供 OpenAI 转换和代上传路径使用；移除其对纯 Raw 请求的门禁作用。

### Phase C：定点图片上传改写

1. 新增最小 JSON 定点改写 helper，放在现有 Leonardo image input/upload service 附近，不引入新依赖。
2. 复用 [leonardo_image_input.go](../../backend/internal/service/leonardo_image_input.go) 读取 URL、Data URI 和 multipart。
3. 复用 [leonardo_image_upload_service.go](../../backend/internal/service/leonardo_image_upload_service.go) 上传并缓存。
4. 扩展 [models.go](../../backend/internal/pkg/leonardo/models.go)：
   - 为模型登记可上传路径、quality tier 映射和计价字段提取规则。
   - FLUX Schnell 首批登记 content/style 路径。
5. 测试只修改目标 `image` 对象、删除 source、保留所有未知字段；覆盖非法 scheme、私网 URL、超限、MIME 欺骗、source+id 冲突、缺失 multipart part 和多引用数量限制。

### Phase D：版本化图片/视频价格目录

1. 将 [pricing-data.js](../../tools/leonardo-pricing-calculator/pricing-data.js) 的静态数据转换为 Go 可校验的版本化快照；运行时不解析 JavaScript。
2. 重构 [leonardo_image_pricing.go](../../backend/internal/service/leonardo_image_pricing.go) 和 [leonardo_video_pricing.go](../../backend/internal/service/leonardo_video_pricing.go)：
   - 支持精确匹配与 `quality_tier_max`。
   - 图片键覆盖模型、官方 quality/mode、尺寸/档位、数量和会改变价格的参考图/Upscale 参数。
   - 视频键覆盖模型、时长、分辨率、数量、音频和模式。
   - 价格快照结构启动时校验：金额为正、档位合法、模型可解析、离散组合无重复、slider 端点完整。
3. 保持客户倍率固定 7.1；客户端不得传报价。
4. 现有资金表无需为“最高价即最终结算”增加部分退款操作；reservation amount 就是最终 customer cost。
5. 在 create/get response 和 usage log 中写入 quality tier、match type 和客户成本。

### Phase E：视频 Raw 创建与任务闭环

1. 扩充 [models.go](../../backend/internal/pkg/leonardo/models.go)：真实探针确认后只加入首个视频模型的 UUID、请求 model、modality、质量/分辨率映射和输入图路径。
2. 修改 [gateway.go](../../backend/internal/server/routes/gateway.go) 与 [openai_videos.go](../../backend/internal/handler/openai_videos.go)：
   - Leonardo Group 视频请求改由 Leonardo Media Handler 处理，不再进入 xAI 拒绝分支。
   - xAI 路径保持不变。
   - `/v1/videos` 与 `/v1/videos/generations` 自动识别兼容请求和 Leonardo Raw 请求。
3. 把通用创建状态机从图片编排器复用到 video；保持最小改动，不复制资金和 unknown 逻辑。
4. 通过真实只读模型目录和单次付费探针确认视频创建参数及轮询响应后，扩展：
   - [types.go](../../backend/internal/pkg/leonardo/types.go) 的视频输出结构。
   - [client.go](../../backend/internal/pkg/leonardo/client.go) 的轮询解析。
   - [leonardo_generation_poller.go](../../backend/internal/service/leonardo_generation_poller.go) 的 video 完成/失败规则。
   - [leonardo_generation_poll_orchestrator.go](../../backend/internal/service/leonardo_generation_poll_orchestrator.go) 的 video 结算和 usage log。
   - [leonardo_media_get_service.go](../../backend/internal/service/leonardo_media_get_service.go) 的视频结果、MIME、大小限制和 Range 内容代理。
5. Webhook 使用同一 modality-aware result parser；未验证视频事件不得改变任务或资金状态。

### Phase F：配置、迁移和文档

1. 如现有字段不足，在 `generation_jobs` 增加 quality tier、protocol 和定价披露字段；价格版本、source、match type 尽量复用现有列。
2. 更新 [config.go](../../backend/internal/config/config.go)、示例 YAML/ENV 和依赖注入文件：
   - 明确 provider/media/video 开关组合。
   - Poll runner 在图片或视频任一启用时启动。
   - video 默认关闭。
3. 更新 [LEONARDO_PRODUCTION_API_INTEGRATION_PLAN.md](../../docs/LEONARDO_PRODUCTION_API_INTEGRATION_PLAN.md) 和探针证据：记录从逐字段重建改为 Raw Body 的架构决策、接口示例、价格回退和灰度门禁。

## Security and Failure Modes

- 非 JSON、多个 JSON 文档、Body 超限、协议字段冲突、未知模型、账号不支持模型：提交前 400/拒绝。
- 未知参数不再由 RelayQ 拒绝，但必须进入完整 Body 指纹；Leonardo 上游 4xx 仍按“POST 可能产生副作用”规则处理，不自动重发。
- `image.source` 只允许 Registry 声明路径，防止递归修改任意客户字段。
- 客户端 Authorization、Cookie、Host、Forwarded 和代理 Header 永不转发。
- 原始 Body、Data URI、预签名 URL 和上传字段不得明文写日志或任务表。
- 代上传任何一步失败：不提交 generation；若资金已预留则全额释放。
- 上游 POST 写出后响应丢失：保留预留，任务 unknown/manual_review。
- 视频输出 Schema 未识别：任务进入 manual_review，不按成功自动结算，也不自动退款。
- 质量档位价格池为空：请求提交前拒绝；不使用跨 modality 最高价。
- `quality_tier_max` 是用户选定档位的最高价最终结算，不在任务完成后退差额；响应必须披露。

## Verification

### 单元测试

- 协议探测：OpenAI、Leonardo Raw、legacy Media、混合冲突、非法 parameters。
- Raw Body：字节级保留未知字段、字段顺序、数字格式和嵌套对象。
- Header：只使用 RelayQ 上游认证，不泄露客户端 Header。
- 幂等：同 key 同 Body replay；同 key 不同未知字段、quality header 或 source 内容冲突。
- 上传：URL/Data URI/multipart、缓存隔离、定点替换、失败释放资金。
- 定价：exact、quality tier max、默认 low、空价格池拒绝、倍率 7.1、披露字段。
- 提交安全：request-not-written 可退款；after-write unknown 不重试、不退款。
- 图片 Poll/Webhook 和旧 OpenAI Images 回归。
- 视频在 feature flag 关闭和未验证模型时继续 fail closed。

### 离线集成测试

- 扩展 [gateway_leonardo_integration_test.go](../../backend/internal/server/routes/gateway_leonardo_integration_test.go)：
  - 断言 mock upstream 收到完全相同的 Raw 图片 Body。
  - 断言代上传只修改 source 对应 image。
  - 断言任务、余额、reservation、轮询和结算一致。
  - 增加视频 mock 创建/轮询/内容 Range/结算闭环。
- 路由回归确保 xAI Videos、非 Leonardo Images 和现有别名不受影响。

### 构建与静态检查

- `gofmt`。
- Leonardo pkg/service/handler/routes 定向测试。
- repository 资金测试和 integration tag 测试。
- `go test ./...` 或记录与本改造无关的既有失败。
- `go build ./cmd/server`。
- `git diff --check` 和编辑文件诊断。

### 真实探针与灰度验收

1. 只读 `/v2/models`，确认 API Key、首批图片/视频模型 UUID、request model 和 parameter schema。
2. 每个付费 POST 前向用户单独确认。
3. FLUX Schnell 原生文生图探针。
4. FLUX Schnell content 单参考图探针。
5. FLUX Schnell style 单参考图探针。
6. 一个最低成本、最短时长视频模型的创建与轮询探针。
7. 对比本地价格快照、上游返回成本和客户价格 ×7.1，脱敏记录证据。
8. 图片通过后灰度开启 Raw 图片；视频必须创建、轮询、输出、内容代理和计费全部通过后才开启 `video_enabled`。

## Assumptions & Decisions

- “原封不动”限定为 JSON Body；Header、认证和目标 URL由 RelayQ控制。
- 创建响应保留 RelayQ task protocol，不原样返回 Leonardo response。
- Images/Videos 使用自动协议识别；混合协议明确拒绝。
- 原生和兼容入口均只允许 Verified Registry 模型。
- 参考图支持已有 Leonardo ID，也支持 RelayQ 代上传 URL、Data URI 和 multipart。
- 代上传允许最小定点改写，不承诺字节级完全不变。
- 质量 Header 缺失默认 low。
- 未命中精确价格时按官方能力档位最高价最终结算并明确披露，不退差额。
- 图片和视频均在本计划范围；视频必须探针后灰度。
- 所有真实付费请求逐次征得用户确认。
- 不提交代码、不自动开启生产 Feature Flag，除非用户后续明确要求。
