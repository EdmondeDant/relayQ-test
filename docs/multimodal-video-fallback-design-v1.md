# RelayQ 视频理解 fallback 设计 v1

## 目标

当上游模型/渠道没有确认支持原生 `video_url` chat 输入时，不再把视频错误映射成图片，而是走显式 fallback：

`video_url -> 抽帧 -> 图片多模态 + 可选 ASR 文本 -> chat model`

## 输入

标准 OpenAI-compatible chat：

```json
{
  "type": "video_url",
  "video_url": {"url":"https://example.com/video.mp4"}
}
```

或兼容形态：

```json
{"type":"input_video", "video_url":"https://example.com/video.mp4"}
```

## 阶段设计

### Phase 1：只做计划与安全校验
已落地：`BuildVideoFallbackPlan`。

- 检查 URL 非空
- `http/https` 限制
- 拒绝 localhost / loopback / private / link-local / unspecified IP
- 输出 fallback method：`video_frames`
- 默认抽帧：最多 8 帧，0.5 FPS

注意：当前安全校验不做 DNS 解析。真实下载时仍必须重复做解析后 IP 检查，防 DNS rebinding。

### Phase 2：下载与抽帧
建议使用 ffmpeg：

```bash
ffmpeg -i input.mp4 -vf fps=0.5,scale='min(768,iw)':-2 -frames:v 8 frame_%03d.jpg
```

代码进度：

- 已新增 `VideoFrameExtractor.ExtractVideoFrames(...)`，负责对已下载到本地的视频执行 ffmpeg 抽帧。
- 已新增 `VideoFrameFilesToChatImageParts(...)`，将抽出的 JPEG 帧转换为标准 OpenAI-compatible `image_url` data URI parts。
- 已新增 `DownloadRemoteVideoForFallback(...)`，使用 SSRF-safe HTTP client 下载远程视频，禁止重定向，限制最大字节数，校验 content-type。
- 已新增 `BuildVideoFallbackChatImageParts(...)`，串联 `video_url -> safe download -> ffmpeg frames -> image_url parts` 的骨架。
- 已新增 ffprobe 时长探测：默认最大 60 秒，超过则拒绝抽帧。
- 已新增 `VideoFrameFilesToFallbackChatParts(...)`，输出完整 chat content：说明文本 + 可选 ASR 文本 + 帧图片。
- 当前本机 ffmpeg 是 shim，真实抽帧仍需真实 ffmpeg 环境验证。

要求：

- 下载超时：已在 downloader 默认 30s
- 最大文件大小：已默认 50MB
- 最大时长：已通过 ffprobe 默认限制 60s
- content-type 白名单：已支持 `video/*` 与 `application/octet-stream`
- 禁止重定向：已默认拒绝 3xx
- DNS rebinding 防护：下载 client 使用 safe dial；后续仍需端到端压测

### Phase 3：ASR 可选
如果视频含音频：

- 提取音轨
- 调用已有音频转写能力或 DashScope/Whisper 类能力
- 将转写文本作为 text part 附加

### Phase 4：拼接为多模态 chat
输出给图片理解模型：

```json
[
  {"type":"text", "text":"下面是一个视频抽帧，请结合帧顺序总结视频内容。"},
  {"type":"image_url", "image_url":{"url":"data:image/jpeg;base64,..."}},
  {"type":"image_url", "image_url":{"url":"data:image/jpeg;base64,..."}}
]
```

如果有 ASR：

```json
{"type":"text", "text":"音频转写：..."}
```

## 响应标记

fallback 响应应在内部日志/metadata 标记：

```json
{
  "fallback_method": "video_frames",
  "native_video": false,
  "frames_used": 8,
  "asr_used": true
}
```

代码进度：`VideoFrameFilesToFallbackChatParts(...)` 已生成上述结构，并支持可选 ASR 文本。

## 原则

- native video 支持明确开启时，优先原生 `video_url`。
- 否则 fallback 必须显式，不能静默伪装。
- 不允许 video -> image_url 单图错误映射。
