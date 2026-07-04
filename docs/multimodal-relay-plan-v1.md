# RelayQ 多模态协议改造方案 v1

> 目标：不要靠猜；对齐 OpenAI-compatible / vLLM / new-api 等成熟网关思路，让模型该有的图片、视频、音频输入能力尽量完整发挥，同时避免把不支持的模态静默错映射。

## 参考依据

### vLLM
- OpenAI-compatible Chat Completions 中明确支持 `image_url`、`video_url`、`input_audio`、`audio_url`。
- `image_url` 标准形态为 `{ "type":"image_url", "image_url": {"url":"..."} }`。
- `video_url` 标准形态为 `{ "type":"video_url", "video_url": {"url":"..."} }`。
- 远程媒体需要考虑 SSRF：允许域名、跳转、超时、文件大小限制。
- 视频既支持原生 video URL，也支持客户端抽帧后以 `data:video/jpeg;base64,...` 携带，并通过 `media_io_kwargs` 保留 fps、duration、frame indices 等元信息。

### QuantumNous/new-api
- 路由按 endpoint/format 分：`/v1/chat/completions`、`/v1/responses`、`/v1/images/*`、`/v1/audio/*`、Claude/Gemini format 等。
- 其 issue #4252 暴露出典型坑：Gemini `fileData.fileUri` 视频被错误映射为 OpenAI `image_url`，导致视频理解失败。
- 经验：media type 必须按 MIME/type 严格映射，不能把 video/audio/file 静默塞进 image。

## 总原则

1. **端点语义分离**：聊天理解、图片生成、视频生成、音频转写各走各端点，不因模型名混用。
2. **入口兼容，内部归一，出口标准**：兼容主流客户端写法，内部统一为 canonical media part，出口按上游标准转换。
3. **不做静默错配**：video 不得自动降级为 image_url；audio 不得自动降级为 text；不支持则明确报错或显式 fallback。
4. **能力矩阵先行**：模型/渠道必须声明 text/image/video/audio input 与 generation/edit 能力。
5. **安全边界必须补**：远程媒体 URL、data URI、本地文件路径都要有约束。

## Canonical Content Part 设计

建议新增内部统一结构（后续落地）：

```go
type CanonicalContentPart struct {
    Kind     string // text | image | video | audio | file
    Text     string
    URL      string
    Data     string
    MIMEType string
    Detail   string
    UUID     string
    Metadata map[string]any
}
```

## 入口兼容规则

### Chat Completions
应兼容：

- Text：`text`、`input_text`
- Image：`image_url: {url}`、`image_url: "..."`、`input_image`
- Video：`video_url: {url}`、`video_url: "..."`、`input_video`（兼容别名，出口转 `video_url`）
- Audio：`input_audio`、`audio_url: {url}`、`audio_url: "..."`

### Responses
当前只应稳定桥接明确支持的内容：

- `input_text` ↔ text
- `input_image` ↔ image

对于 video/audio，如果上游 Responses 未确认支持，不得静默丢弃，应报 unsupported 或走显式 fallback。

### Gemini/Claude 转 OpenAI
按 MIME/type 映射：

- `image/*` → image
- `video/*` → video
- `audio/*` → audio
- 其他 → file/unsupported

## 上游适配策略

### Raw OpenAI-compatible Chat
如果上游本身支持 OpenAI-compatible media part，则保持标准形态转发：

- image → `image_url: {url}`
- video → `video_url: {url}`
- audio URL → `audio_url: {url}`
- inline audio → `input_audio`

### Chat → Responses Bridge
当前阶段只桥接 text/image。video/audio 不静默丢弃，直接明确错误，直到确认目标上游支持相应 Responses schema。

### xAI
- `grok-4.3` 等聊天模型走 chat multimodal；图片已实测可用。
- `/v1/videos/*` 只代表视频生成/编辑/续写，不代表视频理解。
- 如果 xAI chat 后续支持 `video_url`，按标准 chat media part 透传。

## 视频理解落地策略

### Phase A：schema 接住 + capability reject
先让 RelayQ 能识别视频输入，不再误映射；上游不支持时明确返回 unsupported。

### Phase B：显式 fallback
实现 `video -> frames + optional ASR -> image/text multimodal`：

1. 下载/接收视频（受安全策略限制）
2. 抽关键帧
3. 可选音频 ASR
4. 以多张 `image_url` + ASR 文本喂给图片理解模型
5. 响应标记 `fallback: video_frames`

### Phase C：native video_url
确认具体上游支持协议后，对该 channel/model 开启 native `video_url`。

## 本次第一批代码改造

已落地基础：

- `ChatContentPart` 扩展为 text/image/video/audio/input_audio
- 兼容 `image_url` 字符串、`input_image`、`input_text`
- raw chat path 规范化：`input_image -> image_url`、`input_video -> video_url`、字符串 URL -> `{url}`
- Chat → Responses 桥接对 video/audio 明确报 unsupported，避免静默丢失媒体
- 增加单测覆盖图片兼容、video/audio reject、raw path video/audio normalize

## 第二批代码改造进度

已开始落地：

- 新增 `CanonicalContentPart` / `CanonicalContentKind`，把 text/image/video/audio/file 从 provider schema 中抽出来。
- `ChatContentPart -> CanonicalContentPart` 已接入，避免后续 bridge 逻辑继续直接操作上游字段。
- `Chat -> Responses` bridge 已改为通过 canonical parts 转换，且对 video/audio 明确返回 unsupported，避免静默丢媒体。
- 新增 `CanonicalKindFromMIMEType`，用于后续 Gemini/Claude/FileData 转换时按 `image/*`、`video/*`、`audio/*` 精确分类。
- 账号能力矩阵开始扩展：新增 `chat_image_input`、`chat_video_input`、`chat_audio_input` 能力常量与组合检测辅助函数。

## 第三批代码改造进度

已开始落地：

- 调度器新增多 endpoint capability 入口：`SelectAccountWithSchedulerForCapabilities`。
- `/v1/chat/completions` 已把请求中检测到的 `chat_image_input/chat_video_input/chat_audio_input` 与基础 `chat_completions` 一起传给账号调度。
- 默认调度器的账号兼容性检查已支持 capability set，显式配置了 `openai_capabilities` 的账号会按多模态能力筛选。
- 新增 `RequiredChatMediaCapabilitiesFromBody`，从请求体解析所需媒体能力。
- 新增远程媒体 URL 安全预检基础函数 `IsPotentiallyUnsafeRemoteMediaURL`，先覆盖 scheme、localhost、loopback/private/link-local/unspecified IP；后续真实 fetch 时还需做 DNS rebinding 防护。

## 第四批代码改造进度

已开始落地：

- 内容安全提取中的 Gemini `inlineData/fileData` 已按 MIME 精确筛选，只有 `image/*` 会进入 image moderation；`video/*`/`audio/*` 不再被当图片塞入。
- 新增视频理解 fallback 计划层：`BuildVideoFallbackPlan`。
- fallback 计划默认：最多 8 帧、0.5 FPS、可选 ASR、method=`video_frames`。
- fallback 入口已接入远程媒体 URL 安全预检：拒绝非 http/https、localhost、loopback/private/link-local/unspecified IP。
- 新增文档 `docs/multimodal-video-fallback-design-v1.md`，明确 video_url -> frames + optional ASR -> image/text multimodal 的落地路线。

## 第五批代码改造进度

已开始落地：

- 新增 `VideoFrameExtractor.ExtractVideoFrames(...)`，对已下载到本地的视频文件执行 ffmpeg 抽帧。
- 新增 `VideoFrameFilesToChatImageParts(...)`，将 JPEG 帧转成标准 `image_url` data URI content parts。
- 新增 `DownloadRemoteVideoForFallback(...)`，使用 SSRF-safe HTTP client 下载远程视频，默认 30s 超时、50MB 限制、拒绝重定向、校验 content-type。
- 新增 `BuildVideoFallbackChatImageParts(...)`，串联 `video_url -> safe download -> ffmpeg -> image_url parts` 的 pipeline 骨架。
- 新增 ffprobe 时长限制，默认最大 60 秒。
- 新增 `VideoFrameFilesToFallbackChatParts(...)`，输出完整 chat content：说明文本 + 可选 ASR 文本 + 帧图片。
- 当前本机 ffmpeg 是 RelayQ 临时 shim，不是真 ffmpeg；代码已落地但真实抽帧需服务器/运行环境安装真实 ffmpeg 后验证。

## 下一批任务

1. 前端/API 返回模型能力标签。
2. 查全 Gemini/Claude → OpenAI 的所有媒体转换点，继续按 MIME 精确映射。
3. 视频 fallback Phase 2 继续：真实 ffmpeg/ffprobe 环境验证。
4. 视频 fallback Phase 3：ASR 可选接入。
5. 将 fallback pipeline 挂入 chat 请求转换路径，并按 capability 选择 native/fallback/unsupported。
