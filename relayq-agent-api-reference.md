# RelayQ Agent API Reference

适用于需要把本项目当前图片、视频、音频接口规则直接投喂给 agents、SDK 生成器、工作流编排器的场景。

本文件只描述当前仓库里的实际接入格式与兼容行为，优先级高于旧文档示例。

## 1. 基础规则

- Base URL: `https://www.relayq.top/v1`
- 鉴权头: `Authorization: Bearer sk-your-api-key`
- Content-Type: `application/json`
- 不要直连 `api.x.ai`、`imgen.x.ai`、`vidgen.x.ai`、`api.openai.com`
- 一律请求 RelayQ 网关，由网关根据模型名转发到对应上游

标准请求头：

```http
POST /v1/images/generations HTTP/1.1
Host: www.relayq.top
Authorization: Bearer sk-your-api-key
Content-Type: application/json
```

## 2. 模型族与字段规则

### 2.1 gpt-image-2-adobe / gpt-image-* 系列

这些模型走 OpenAI 兼容图片接口语义。

支持的主字段：

```json
{
  "model": "gpt-image-2-adobe",
  "prompt": "string",
  "size": "1:1 | 16:9 | 9:16 | 3:2 | 2:3",
  "quality": "low | medium | high | auto",
  "style": "natural | vivid",
  "background": "opaque | transparent | auto",
  "n": 1
}
```

关键规则：

- `gpt-image-2-adobe` 和其他 `gpt-image-*` 模型使用 `size`
- 不使用 `aspect_ratio`
- 可以带 `quality`
- 可以带 `style`
- 可以带 `background`
- 网关不会再强制追加 `response_format`

文生图示例：

```json
{
  "model": "gpt-image-2-adobe",
  "prompt": "一张高端商业产品海报，玻璃香水瓶置于黑金背景，电影级打光，细节清晰",
  "size": "16:9",
  "quality": "high",
  "style": "natural",
  "background": "opaque",
  "n": 1
}
```

图片编辑示例：

```json
{
  "model": "gpt-image-2-adobe",
  "prompt": "保留原始构图，把画面中的中文招牌替换成英文招牌，整体风格不变",
  "images": [
    {
      "image_url": "data:image/png;base64,..."
    }
  ],
  "size": "1:1",
  "quality": "high",
  "style": "natural",
  "background": "opaque"
}
```

### 2.2 grok-imagine-image* 系列

这些模型走 xAI 图片接口语义。

支持的主字段：

```json
{
  "model": "grok-imagine-image",
  "prompt": "string",
  "aspect_ratio": "1:1 | 16:9 | 9:16 | 3:2 | 2:3 | auto",
  "resolution": "1k | 2k",
  "n": 1
}
```

关键规则：

- `grok-imagine-image*` 不用 `size`
- `grok-imagine-image*` 不用 `style`
- `grok-imagine-image*` 不用 `background`
- 画质通过 `resolution` 表达，不是直接传 `quality`
- 前端体验层如果用户选 `quality=high`，会被映射为 `resolution=2k`
- 其他情况默认映射为 `resolution=1k`

文生图示例：

```json
{
  "model": "grok-imagine-image-quality",
  "prompt": "东方玄幻电影海报，两位主角在云海之上对决，人物面部清晰，光效强烈",
  "aspect_ratio": "16:9",
  "resolution": "2k",
  "n": 1
}
```

图片编辑示例：

```json
{
  "model": "grok-imagine-image-quality",
  "prompt": "保留人物脸和服装，把背景改成黄昏沙漠战场，增加风沙与电影感",
  "images": [
    {
      "image_url": "data:image/png;base64,..."
    }
  ],
  "aspect_ratio": "16:9",
  "resolution": "2k"
}
```

### 2.3 非 gpt-image 且非 grok-imagine-image 的兼容图片模型

这类模型仍然通过图片接口调用，但网关/前端会附带：

```json
{
  "response_format": "b64_json"
}
```

这是当前实现里的兼容分支，不适用于 `gpt-image-*` 与 `grok-imagine-image*`。

## 3. 图片接口

### 3.1 文生图

端点：`POST /v1/images/generations`

#### gpt-image-2-adobe 请求体

```json
{
  "model": "gpt-image-2-adobe",
  "prompt": "高端珠宝广告，模特侧脸特写，冷白珠宝反光，杂志封面质感",
  "size": "3:2",
  "quality": "high",
  "style": "natural",
  "background": "opaque",
  "n": 1
}
```

#### grok-imagine-image 请求体

```json
{
  "model": "grok-imagine-image-quality",
  "prompt": "高端珠宝广告，模特侧脸特写，冷白珠宝反光，杂志封面质感",
  "aspect_ratio": "3:2",
  "resolution": "2k",
  "n": 1
}
```

典型响应：

```json
{
  "data": [
    {
      "url": "data:image/png;base64,...",
      "revised_prompt": ""
    }
  ],
  "request_id": "img_123",
  "billing": {
    "amount": 0.12,
    "currency": "USD",
    "balance_after": 19.88
  }
}
```

说明：

- 当前网关可能返回 `url`
- 也可能返回 `b64_json`
- 前端最终会统一消费为可展示图片 URL
- 某些 xAI / 外联远程图片地址会被网关内联成 `data:` URL，避免浏览器直连超时

### 3.2 图片编辑

端点：`POST /v1/images/edits`

当前实际实现使用的是 `images` 数组，不是单个 `image` 字段。

#### gpt-image-2-adobe 编辑请求体

```json
{
  "model": "gpt-image-2-adobe",
  "prompt": "把图片中的海报文案替换成法语，保留原排版、字体层级与视觉风格",
  "images": [
    {
      "image_url": "data:image/png;base64,..."
    }
  ],
  "mask": {
    "image_url": "data:image/png;base64,..."
  },
  "size": "16:9",
  "quality": "high",
  "style": "natural",
  "background": "opaque"
}
```

#### grok-imagine-image 编辑请求体

```json
{
  "model": "grok-imagine-image-quality",
  "prompt": "把图片中的海报文案替换成法语，保留原排版、字体层级与视觉风格",
  "images": [
    {
      "image_url": "data:image/png;base64,..."
    }
  ],
  "mask": {
    "image_url": "data:image/png;base64,..."
  },
  "aspect_ratio": "16:9",
  "resolution": "2k"
}
```

说明：

- `grok-imagine-image*` 的 edits 现在走原生 `/v1/images/edits`
- 不能再错误转成 OpenAI Responses 路径
- 这正是之前 502 的一个根因，现已修正

## 4. 视频接口

### 4.1 提交视频任务

端点：`POST /v1/videos/generations`

当前视频是异步模式。

提交成功后会先拿到：

```json
{
  "request_id": "video_req_123",
  "status": "queued"
}
```

然后轮询：

- `GET /v1/videos/{request_id}` 查看任务状态
- `GET /v1/videos/{request_id}/content` 获取 RelayQ 代理后的视频内容

### 4.2 grok-imagine-video 请求格式

当前前端实际发送格式：

```json
{
  "model": "grok-imagine-video",
  "prompt": "string",
  "duration": 10,
  "aspect_ratio": "16:9",
  "resolution": "720p"
}
```

如果是首帧图生视频：

```json
{
  "model": "grok-imagine-video",
  "prompt": "用这张图作为首帧，人物抬头看向镜头，头发被风吹动，镜头缓慢推进",
  "duration": 10,
  "aspect_ratio": "16:9",
  "resolution": "1080p",
  "image": {
    "url": "data:image/png;base64,..."
  }
}
```

当前项目里已经接入并展示的分辨率选项：

```json
[
  "480p",
  "720p",
  "1080p"
]
```

当前项目里已经接入并展示的时长选项：

```json
[
  5,
  10,
  15,
  20
]
```

说明：

- 视频比图片更慢，这是正常现象
- 当前是轮询模式，不是流式完成通知
- 浏览器不要直连 `vidgen.x.ai`
- 应优先使用 `GET /v1/videos/{request_id}/content`
- 前端当前也已经改为使用这个 content 路径做预览和下载

### 4.3 轮询响应示例

任务处理中：

```json
{
  "request_id": "video_req_123",
  "status": "processing",
  "progress": 65,
  "billing": {
    "amount": 0.8,
    "currency": "USD"
  }
}
```

任务完成：

```json
{
  "request_id": "video_req_123",
  "status": "done",
  "progress": 100,
  "video": {
    "url": "https://vidgen.x.ai/xai-vidgen-bucket/...mp4"
  },
  "billing": {
    "amount": 0.8,
    "currency": "USD",
    "balance_after": 19.2
  }
}
```

实际消费建议：

- 用 `request_id` 做状态跟踪
- 不要在浏览器里直接拿 `video.url` 播放
- 应改用：

```http
GET /v1/videos/{request_id}/content
Authorization: Bearer sk-your-api-key
```

## 5. Chat Completions 图片桥接

端点：`POST /v1/chat/completions`

仅当 agent / SDK 只支持 chat 接口时使用。

当模型名是 `grok-imagine-image*` 时，可以直接在 chat/completions 里生图。

示例：

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

兼容写法也支持把这些参数放到：

```json
{
  "image_options": {},
  "providerOptions": {
    "xai": {}
  },
  "provider_options": {
    "xai": {}
  }
}
```

典型返回：

```json
{
  "id": "chatcmpl-image-123",
  "object": "chat.completion",
  "model": "grok-imagine-image-quality",
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "![generated image](data:image/png;base64,...)"
      },
      "finish_reason": "stop"
    }
  ]
}
```

## 6. 音频接口

当前音频统一走：`POST /v1/chat/completions`

### 6.1 ASR 转写

只保留官方兼容做法：

```json
{
  "model": "mimo-v2.5-asr",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "input_audio",
          "input_audio": {
            "data": "base64-audio-data",
            "format": "wav"
          }
        }
      ]
    }
  ],
  "asr_options": {
    "language": "zh"
  },
  "stream": false
}
```

说明：

- 当前项目已去掉“时间戳文本 / 字幕草稿”那类额外格式约束
- 结果只保留纯文本

### 6.2 标准配音

```json
{
  "model": "mimo-v2.5-tts",
  "audio": {
    "format": "wav",
    "voice": "mimo_default"
  },
  "messages": [
    {
      "role": "user",
      "content": "请用自然讲述风格朗读，输出语言：中文。"
    },
    {
      "role": "assistant",
      "content": "你好，这是一段语音合成测试。"
    }
  ],
  "stream": false
}
```

## 7. 下载入口与前端按钮

接口文档下载按钮当前就是这个入口：

```html
<a href="/relayq-agent-api-reference.md" download="relayq-agent-api-reference.md">下载详细 Agent 接口文档</a>
```

含义：

- `href` 指向站点根目录的静态文档 `relayq-agent-api-reference.md`
- `download` 文件名固定为 `relayq-agent-api-reference.md`
- 你这次要求补充的内容，已经写进这个文件

如果部署环境会把根目录静态文件映射到站点根路径，这个按钮无需再改 href。

## 8. 给 Agent/SDK 的选择建议

### 图片模型选择

- 如果模型名是 `gpt-image-2-adobe` 或其他 `gpt-image-*`
  - 用 `size`
  - 可加 `quality`
  - 可加 `style`
  - 可加 `background`
- 如果模型名是 `grok-imagine-image*`
  - 用 `aspect_ratio`
  - 用 `resolution`
  - 不要发 `style`
  - 不要发 `background`
  - 不要发 `size`

### 视频模型选择

- 如果模型名是 `grok-imagine-video`
  - 用 `POST /v1/videos/generations`
  - 用 `duration`
  - 用 `aspect_ratio`
  - 用 `resolution`
  - 有首帧图时传 `image.url`
  - 用 `GET /v1/videos/{request_id}` 轮询
  - 用 `GET /v1/videos/{request_id}/content` 下载或预览

### 错误避免

- 不要把 grok 图片编辑发到 Responses 风格接口
- 不要把 xAI 图片/视频返回的远程 URL 直接交给浏览器
- 不要混用 `size` 和 `aspect_ratio`
- 不要把图片模型当推理模型使用
- 不要把视频模型当 chat 模型使用

