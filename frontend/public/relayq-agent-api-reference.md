# RelayQ Agent API Reference

## Purpose

This document is designed for machine ingestion by external agents, SDK generators, workflow engines, code assistants, and integration bots.

Primary objectives:

- Generate valid requests against the RelayQ gateway.
- Avoid direct upstream vendor calls.
- Avoid unsupported field assumptions.
- Correctly handle image, video, audio, and chat-bridge media workflows.
- Treat this file as the highest-priority contract for RelayQ media integration.

Human-friendly explanation is not the goal. Strict protocol compliance is the goal.

---

## Canonical Gateway

- Default site host: `https://www.relayq.top`
- Default API base URL: `https://www.relayq.top/v1`
- Authentication scheme: `Authorization: Bearer <RELAYQ_API_KEY>`

Never route requests to upstream vendor hosts such as `api.x.ai`, `api.openai.com`, or any other provider URL when integrating through RelayQ.

All examples in this document assume RelayQ gateway access.

---

## Global Integration Rules

1. Always send the customer's RelayQ API key in the `Authorization` header.
2. Use only model names visible to the customer's current group whitelist.
3. Do not hardcode upstream provider model catalogs as authoritative.
4. Preserve RelayQ response fields instead of dropping unknown fields.
5. Surface RelayQ errors as-is whenever possible.
6. Do not silently transform media workflows across endpoints unless explicitly documented here.
7. For image generation, prefer JSON request bodies.
8. For video generation, treat the workflow as asynchronous unless the actual response proves otherwise.
9. For audio transcription, use `multipart/form-data` when uploading audio files.
10. For TTS, expect binary audio response unless RelayQ returns an error JSON.

---

## Transport Contract

### Auth Header

```http
Authorization: Bearer <RELAYQ_API_KEY>
```

### JSON Header

```http
Content-Type: application/json
```

### Multipart Upload Header

Let the HTTP client set the multipart boundary automatically.

```http
Content-Type: multipart/form-data; boundary=...
```

---

# 1. Chat Completions Image Bridge

Use this when the client/agent can only call `/v1/chat/completions`, but wants image generation.

RelayQ behavior:

1. If `model` is `grok-imagine-image` or `grok-imagine-image-quality`, RelayQ treats the request as image generation.
2. RelayQ extracts the last `user` message text as the image prompt.
3. RelayQ calls the enabled upstream image model.
4. RelayQ wraps the image as a standard OpenAI-compatible chat completion.
5. For `stream: true`, RelayQ emits SSE chat completion chunks and ends with `data: [DONE]`.

## Endpoint

```http
POST /v1/chat/completions
```

## Typical Models

- `grok-imagine-image`
- `grok-imagine-image-quality`

## Request JSON: text-to-image through chat

```json
{
  "model": "grok-imagine-image-quality",
  "messages": [
    {
      "role": "user",
      "content": "仙魔大战，场面宏大，人物面部清晰，东方玄幻电影海报"
    }
  ],
  "aspect_ratio": "16:9",
  "resolution": "2k",
  "quality": "high",
  "n": 1
}
```

## Supported Image Options in Chat Bridge

RelayQ reads image options from top-level fields and from nested options.

Accepted shapes:

```json
{
  "aspect_ratio": "16:9",
  "resolution": "2k",
  "quality": "high",
  "n": 1
}
```

```json
{
  "image_options": {
    "aspect_ratio": "16:9",
    "resolution": "2k",
    "quality": "high"
  }
}
```

```json
{
  "providerOptions": {
    "xai": {
      "aspectRatio": "16:9",
      "resolution": "2k",
      "quality": "high"
    }
  }
}
```

Also accepted:

```json
{
  "provider_options": {
    "xai": {
      "aspectRatio": "16:9",
      "resolution": "2k",
      "quality": "high"
    }
  }
}
```

## Response JSON: non-streaming

```json
{
  "id": "chatcmpl-image-1750000000",
  "object": "chat.completion",
  "created": 1750000000,
  "model": "grok-imagine-image-quality",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "![generated image](data:image/png;base64,...)"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 0,
    "completion_tokens": 0,
    "total_tokens": 0
  }
}
```

## Response SSE: streaming

```text
data: {"id":"chatcmpl-image-...","object":"chat.completion.chunk","choices":[{"delta":{"role":"assistant","content":"![generated image](data:image/png;base64,...)"}}]}

data: {"id":"chatcmpl-image-...","object":"chat.completion.chunk","choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

---

# 2. Image Generation Official JSON Interface

Use this when the client can call dedicated image APIs.

## Endpoint

```http
POST /v1/images/generations
```

## Typical Models

- `grok-imagine-image`
- `grok-imagine-image-quality`

## Request JSON: text-to-image

```json
{
  "model": "grok-imagine-image-quality",
  "prompt": "一张现代写实电影感人物照片，面部清晰",
  "n": 1,
  "aspect_ratio": "3:4",
  "resolution": "2k",
  "quality": "high",
  "response_format": "url"
}
```

## Important Image Parameter Notes

Prefer Grok/xAI-style fields:

- `aspect_ratio`: aspect ratio control.
- `resolution`: image resolution tier, commonly `1k` or `2k`.
- `quality`: quality hint if the enabled model accepts it.
- `n`: number of images.
- `response_format`: `url` or `b64_json`.

Do not rely on OpenAI-only `size` for Grok image models. If a client sends extra compatible fields, RelayQ preserves/forwards supported fields when possible.

## Response Handling

Agents must parse in this order:

1. `data[].url`
2. `data[].b64_json`
3. any provider-specific preserved fields

## Typical Response

```json
{
  "created": 1750000000,
  "data": [
    {
      "url": "https://cdn.example.com/generated/abc.png",
      "b64_json": null,
      "mime_type": "image/png",
      "revised_prompt": ""
    }
  ],
  "usage": {
    "cost_in_usd_ticks": 123456789
  }
}
```

---

# 3. Image Edit

Use only when the customer's enabled account/model supports image edit.

## Endpoint

```http
POST /v1/images/edits
```

## JSON Request Shape

RelayQ supports JSON image URL/data URL inputs for compatible image edit flows.

```json
{
  "model": "grok-imagine-image-quality",
  "prompt": "把这张图改成铅笔素描，保留人物五官和构图",
  "images": [
    {
      "image_url": "data:image/png;base64,..."
    }
  ],
  "resolution": "2k",
  "quality": "high",
  "response_format": "url"
}
```

## Multipart Request Shape

When a client has local files, it may use multipart form data:

```bash
curl https://www.relayq.top/v1/images/edits \
  -H "Authorization: Bearer ***" \
  -F "model=grok-imagine-image-quality" \
  -F "prompt=把图片改成电影感海报" \
  -F "image=@input.png" \
  -F "resolution=2k" \
  -F "quality=high"
```

---

# 4. Video Generation

Video generation is asynchronous. Always submit a job, extract `request_id` or `id`, then poll.

## Official Endpoint

```http
POST /v1/videos/generations
```

## OpenAI/Sora-Compatible Alias

RelayQ also accepts:

```http
POST /v1/videos
```

This alias is intended for tools that hardcode OpenAI/Sora style video endpoints. RelayQ maps compatible fields to Grok Imagine video.

## Typical Model

- `grok-imagine-video`

OpenAI/Sora-style aliases accepted by RelayQ compatibility layer:

- `sora-2` -> `grok-imagine-video`
- `sora-2-pro` -> `grok-imagine-video`

---

## 4.1 Text-to-Video

```json
{
  "model": "grok-imagine-video",
  "prompt": "仙魔大战，两位主角在云海战场交锋，电影级镜头",
  "duration": 10,
  "aspect_ratio": "16:9",
  "resolution": "720p"
}
```

## 4.2 Image-to-Video / First-Frame Video

Use this when one image should be the first frame.

```json
{
  "model": "grok-imagine-video",
  "prompt": "使用图片作为首帧，两人开始战斗，剑气和黑火碰撞",
  "duration": 10,
  "aspect_ratio": "16:9",
  "resolution": "720p",
  "image": {
    "url": "data:image/png;base64,..."
  }
}
```

RelayQ also accepts OpenAI/Sora-style input reference:

```json
{
  "model": "sora-2",
  "prompt": "the person lifts a coffee cup, takes a sip, and puts it down",
  "seconds": 10,
  "size": "1280x720",
  "input_reference": {
    "image_url": "data:image/jpeg;base64,..."
  }
}
```

RelayQ compatibility behavior:

- `seconds` -> `duration`
- `sora-2` / `sora-2-pro` -> `grok-imagine-video`
- `size` -> best-effort `aspect_ratio` + `resolution`
- `input_reference.image_url` -> official first-frame `image.url`

## 4.3 Reference-to-Video

Use this when one or more images are references, not necessarily the first frame.

```json
{
  "model": "grok-imagine-video",
  "prompt": "让参考图中的人物出现在宏大战场中，保持服装和脸部特征",
  "duration": 10,
  "aspect_ratio": "16:9",
  "resolution": "720p",
  "reference_images": [
    {
      "url": "https://example.com/character.png"
    },
    {
      "url": "data:image/jpeg;base64,..."
    }
  ]
}
```

Provider option mode variant:

```json
{
  "model": "sora-2",
  "prompt": "use the input image as visual reference",
  "duration": 10,
  "input_reference": {
    "image_url": "data:image/png;base64,..."
  },
  "providerOptions": {
    "xai": {
      "mode": "reference-to-video",
      "aspectRatio": "16:9",
      "resolution": "720p"
    }
  }
}
```

## 4.4 Video Edits

Endpoint:

```http
POST /v1/videos/edits
```

Example JSON:

```json
{
  "model": "grok-imagine-video",
  "prompt": "让视频变成雨夜电影风格，保留主要人物动作",
  "video": {
    "url": "https://example.com/input.mp4"
  },
  "duration": 10,
  "aspect_ratio": "16:9",
  "resolution": "720p"
}
```

## 4.5 Video Extensions

Endpoint:

```http
POST /v1/videos/extensions
```

Example JSON:

```json
{
  "model": "grok-imagine-video",
  "prompt": "继续这个镜头，角色向前走进宫殿",
  "video": {
    "url": "https://example.com/input.mp4"
  },
  "duration": 5,
  "aspect_ratio": "16:9",
  "resolution": "720p"
}
```

## 4.6 Video Parameters

Recommended fields:

- `model`: `grok-imagine-video`
- `prompt`: required text description
- `duration`: seconds, commonly 5/10/15 depending on model/mode
- `aspect_ratio`: common values: `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3`
- `resolution`: common values: `480p`, `720p`; some modes/models may accept `1080p`
- `image.url`: first-frame image for image-to-video
- `reference_images[].url`: reference images for reference-to-video
- `video.url`: video input for edit/extension if supported

Compatibility aliases accepted by RelayQ:

- `aspectRatio` -> `aspect_ratio`
- `seconds` -> `duration`
- `size` -> best-effort `aspect_ratio` + `resolution`
- `input_reference.image_url` -> `image.url` by default
- `images[]` -> `reference_images[]`

---

# 5. Video Polling and Content Download

## Polling Endpoint

```http
GET /v1/videos/{request_id}
```

## Polling Algorithm for Agents

1. Submit job.
2. Extract request id from `request_id`, else `id`.
3. Poll every 5-15 seconds. Do not tight-loop.
4. Stop on terminal status.
5. On success, read `video.url` first.
6. If a client needs binary content through RelayQ, call `/content`.

## Possible Non-Final Statuses

- `pending`
- `queued`
- `processing`
- `in_progress`

## Success-Like Terminal Statuses

- `done`
- `completed`
- `succeeded`
- `success`

## Failure-Like Terminal Statuses

- `failed`
- `canceled`
- `cancelled`
- `expired`
- `error`

## Poll Response Example

```json
{
  "request_id": "21f7f4af-0fb0-9a85-aa10-69b0caa3b901",
  "status": "done",
  "model": "grok-imagine-video",
  "video": {
    "url": "https://vidgen.x.ai/.../video.mp4",
    "duration": 10
  },
  "progress": 100
}
```

## Content Download Endpoint

```http
GET /v1/videos/{request_id}/content
```

RelayQ behavior:

- Resolve original video request/account when possible.
- Query status.
- If `video.url` is available, redirect or stream content.

Example:

```bash
curl -L https://www.relayq.top/v1/videos/21f7f4af-0fb0-9a85-aa10-69b0caa3b901/content \
  -H "Authorization: Bearer ***" \
  -o result.mp4
```

---

# 6. Audio Transcription

Use this for speech-to-text.

## Endpoint

```http
POST /v1/audio/transcriptions
```

## Content Type

```http
multipart/form-data
```

## Required Fields

- `file`: audio file
- `model`: transcription model enabled for the customer

## Common Optional Fields

- `language`: language hint, for example `zh` or `en`
- `prompt`: optional transcription context
- `temperature`: model-dependent
- `response_format`: if supported by enabled model

## cURL Example

```bash
curl https://www.relayq.top/v1/audio/transcriptions \
  -H "Authorization: Bearer ***" \
  -F "file=@demo.mp3" \
  -F "model=whisper-1" \
  -F "language=zh" \
  -F "prompt=这是一段中文会议录音"
```

## Response Example

```json
{
  "text": "这是转写后的文本内容。"
}
```

## Agent Parsing Rule

Prefer `text` as canonical output. Preserve additional fields if present.

---

# 7. Text To Speech

Use this for text-to-audio.

## Endpoint

```http
POST /v1/audio/speech
```

## Content Type

```http
application/json
```

## Request JSON

```json
{
  "model": "gpt-4o-mini-tts",
  "voice": "alloy",
  "input": "你好，这是一段语音合成测试。",
  "format": "mp3"
}
```

## Common Fields

- `model`: TTS model enabled for customer
- `voice`: voice id/name supported by model
- `input`: text to synthesize
- `format`: output format, e.g. `mp3`, `wav`, `opus` if supported
- `speed`: optional, if model supports it

## Response Handling

The response is usually binary audio bytes.

Agent rules:

1. Do not force JSON parsing if `Content-Type` is audio.
2. Save output using extension inferred from `Content-Type` or requested `format`.
3. If RelayQ returns JSON with an `error` object, surface it as failure.

## cURL Example

```bash
curl https://www.relayq.top/v1/audio/speech \
  -H "Authorization: Bearer ***" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini-tts",
    "voice": "alloy",
    "input": "你好，这是一段语音合成测试。",
    "format": "mp3"
  }' \
  -o speech.mp3
```

---

# 8. Minimal cURL Pack

## Chat Image Bridge

```bash
curl https://www.relayq.top/v1/chat/completions \
  -H "Authorization: Bearer ***" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-image-quality",
    "messages": [
      {"role": "user", "content": "仙魔大战，场面宏大，人物面部清晰"}
    ],
    "aspect_ratio": "16:9",
    "resolution": "2k",
    "quality": "high"
  }'
```

## Image Generation

```bash
curl https://www.relayq.top/v1/images/generations \
  -H "Authorization: Bearer ***" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-image-quality",
    "prompt": "现代写实人物摄影，面部清晰",
    "aspect_ratio": "3:4",
    "resolution": "2k",
    "quality": "high"
  }'
```

## Text-to-Video

```bash
curl https://www.relayq.top/v1/videos/generations \
  -H "Authorization: Bearer ***" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-video",
    "prompt": "宏大的仙魔大战，两位主角交锋",
    "duration": 10,
    "aspect_ratio": "16:9",
    "resolution": "720p"
  }'
```

## Image-to-Video

```bash
curl https://www.relayq.top/v1/videos/generations \
  -H "Authorization: Bearer ***" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-video",
    "prompt": "让图片中的两位主角开始战斗",
    "duration": 10,
    "aspect_ratio": "16:9",
    "resolution": "720p",
    "image": {"url": "data:image/png;base64,..."}
  }'
```

## Audio Transcription

```bash
curl https://www.relayq.top/v1/audio/transcriptions \
  -H "Authorization: Bearer ***" \
  -F "file=@demo.mp3" \
  -F "model=whisper-1" \
  -F "language=zh"
```

## Text To Speech

```bash
curl https://www.relayq.top/v1/audio/speech \
  -H "Authorization: Bearer ***" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini-tts",
    "voice": "alloy",
    "input": "你好，这是一段语音合成测试。",
    "format": "mp3"
  }' \
  -o speech.mp3
```

---

# 9. Error Handling Contract

Agents integrating with RelayQ must follow these rules:

1. Preserve HTTP status codes from RelayQ.
2. Preserve original error body whenever possible.
3. Do not rewrite permission errors into generic upstream timeouts.
4. Distinguish between:
   - invalid model
   - missing group permission
   - malformed request body
   - unsupported endpoint
   - upstream temporary failure
   - async video job failure
5. When video submission succeeds but status is pending/processing, do not treat it as failure.
6. Do not poll faster than every 5 seconds unless RelayQ explicitly returns a shorter retry hint.

## Typical Error Shape

```json
{
  "error": {
    "type": "invalid_request_error",
    "message": "model is required"
  }
}
```

---

# 10. Recommended Agent Decision Tree

1. User asks for an image and the agent can call `/v1/images/generations`:
   - Use `/v1/images/generations`.
2. User asks for an image but the agent only has chat completion support:
   - Use `/v1/chat/completions` with `model=grok-imagine-image-quality`.
3. User asks for text-to-video:
   - Use `/v1/videos/generations` with prompt only.
4. User asks to animate an image:
   - Use `/v1/videos/generations` with `image.url`.
5. User asks to use reference character/style images:
   - Use `/v1/videos/generations` with `reference_images[].url`.
6. User asks to transcribe audio:
   - Use `/v1/audio/transcriptions` multipart.
7. User asks for speech/TTS:
   - Use `/v1/audio/speech`, save binary response.

---

# 11. Security and Privacy Notes for Agents

- Never expose RelayQ API keys in logs or visible messages.
- Avoid sending private local files unless the user explicitly requests upload/processing.
- For large media, prefer data URLs only when the client/tool supports large request bodies; otherwise upload/host the asset and send a URL.
- Do not use upstream vendor URLs directly. RelayQ is the customer-facing gateway.
