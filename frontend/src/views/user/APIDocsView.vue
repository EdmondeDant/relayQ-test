<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="bg-gradient-to-r from-indigo-500 via-violet-500 to-fuchsia-500 px-6 py-8 text-white md:px-8">
          <div class="max-w-4xl space-y-4">
            <h2 class="text-3xl font-black tracking-tight md:text-4xl">接口文档</h2>
            <p class="max-w-3xl text-sm leading-7 text-white/90 md:text-base">
              本页面与下载版 Agent 接口文档保持一致，按当前仓库的真实实现说明图片、视频、音频接口格式。图片、音频、视频模型是子 agents，不能当推理模型使用。
            </p>
            <div class="flex flex-wrap items-center gap-3">
              <a
                :href="agentDocDownloadUrl"
                download="relayq-agent-api-reference.md"
                class="inline-flex items-center gap-2 rounded-full border border-white/20 bg-white/15 px-4 py-2 text-sm font-semibold text-white transition hover:bg-white/25"
              >
                <Icon name="download" size="sm" />
                <span>下载详细 Agent 接口文档</span>
              </a>
              <span class="inline-flex items-center gap-2 rounded-full border border-amber-300/40 bg-amber-400/20 px-3 py-2 text-sm font-black tracking-wide text-amber-50 shadow-sm">
                <span aria-hidden="true" class="text-base leading-none">←</span>
                <span>左边这个文档主要投喂给 agents / SDK 生成器，人看不明白没关系，AI 看明白就行。</span>
              </span>
            </div>
          </div>
        </div>

        <div class="grid gap-4 px-6 py-6 md:grid-cols-4 md:px-8">
          <article
            v-for="item in introCards"
            :key="item.title"
            class="rounded-2xl border border-slate-200 bg-slate-50 p-5 dark:border-dark-700 dark:bg-dark-800"
          >
            <div class="text-sm font-extrabold text-slate-900 dark:text-white">{{ item.title }}</div>
            <div class="mt-2 text-xs font-semibold text-indigo-600 dark:text-indigo-300">{{ item.endpoint }}</div>
            <p class="mt-3 text-sm leading-6 text-slate-600 dark:text-slate-300">{{ item.description }}</p>
          </article>
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900 md:p-8">
        <h3 class="text-xl font-black text-slate-950 dark:text-white">1. 基础接入</h3>
        <div class="mt-4 grid gap-4 md:grid-cols-2">
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">请求头</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ authExample }}</pre>
          </div>
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">接入说明</div>
            <ul class="mt-3 space-y-2 text-sm leading-6 text-slate-600 dark:text-slate-300">
              <li>只把请求发到 RelayQ，不要直连 `api.x.ai`、`imgen.x.ai`、`vidgen.x.ai`、`api.openai.com`。</li>
              <li>模型名称以你的分组白名单为准；没有权限或模型不在白名单会返回错误。</li>
              <li>图片/视频建议优先使用 JSON 官方格式；MiMo 音频模型当前也走 JSON 的 `POST /v1/chat/completions`。</li>
              <li>视频生成是异步任务：提交后拿 `request_id`，再轮询 `/v1/videos/{request_id}`。</li>
            </ul>
          </div>
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900 md:p-8">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-xl font-black text-slate-950 dark:text-white">2. Chat Completions 图片桥接</h3>
            <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">适合 Codex、OpenClaw、Agent 工具只会打 `/v1/chat/completions` 的场景。模型名明确为 `grok-imagine-image*` 时，RelayQ 会把最后一条 user message 当作 prompt 生图，再包装成标准 chat completion 返回。</p>
          </div>
          <span class="rounded-full bg-indigo-50 px-3 py-1 text-xs font-bold text-indigo-700 dark:bg-indigo-500/10 dark:text-indigo-300">POST /v1/chat/completions</span>
        </div>
        <div class="mt-5 grid gap-4 xl:grid-cols-2">
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">请求示例：文生图 2K</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ chatImageExample }}</pre>
          </div>
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">返回示例：标准对话格式</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ chatImageResponseExample }}</pre>
          </div>
        </div>
        <div class="mt-4 rounded-2xl border border-indigo-200 bg-indigo-50 p-4 text-sm leading-6 text-indigo-900 dark:border-indigo-500/30 dark:bg-indigo-500/10 dark:text-indigo-100">
          只在 agent / SDK 只支持 `chat/completions` 时使用。模型名必须是 `grok-imagine-image*`。支持参数：`n`、`aspect_ratio/aspectRatio`、`resolution`（普通模型 1k/2k，`grok-imagine-image-quality` 可到 4k）、`quality`、`user`，也可放在 `image_options`、`providerOptions.xai` 或 `provider_options.xai` 中。
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900 md:p-8">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-xl font-black text-slate-950 dark:text-white">3. 图片模型官方 JSON 接口</h3>
            <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">适用于标准文生图和图片编辑。`gpt-image-*` 使用 `size`，`grok-imagine-image*` 使用 `aspect_ratio` + `resolution`，不要混用。</p>
          </div>
          <span class="rounded-full bg-indigo-50 px-3 py-1 text-xs font-bold text-indigo-700 dark:bg-indigo-500/10 dark:text-indigo-300">POST /v1/images/generations</span>
        </div>

        <div class="mt-5 grid gap-4 xl:grid-cols-2">
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">请求示例：官方参数</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ imageRequestExample }}</pre>
          </div>
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">返回示例</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ imageResponseExample }}</pre>
          </div>
        </div>
        <div class="mt-4 rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
          <div class="text-sm font-bold text-slate-900 dark:text-white">图片编辑 JSON 示例</div>
          <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ imageEditExample }}</pre>
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900 md:p-8">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-xl font-black text-slate-950 dark:text-white">4. 视频模型：文生视频 / 图生视频 / 参考图视频</h3>
            <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">`grok-imagine-video` 是异步任务。提交后先拿 `request_id`，再轮询 `/v1/videos/{request_id}`，预览和下载应走 `/v1/videos/{request_id}/content`。</p>
          </div>
          <span class="rounded-full bg-fuchsia-50 px-3 py-1 text-xs font-bold text-fuchsia-700 dark:bg-fuchsia-500/10 dark:text-fuchsia-300">POST /v1/videos/generations</span>
        </div>

        <div class="mt-5 grid gap-4 xl:grid-cols-2">
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">文生视频</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ videoTextSubmitExample }}</pre>
          </div>
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">图生视频（首帧）</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ videoImageSubmitExample }}</pre>
          </div>
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">参考图视频</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ videoReferenceSubmitExample }}</pre>
          </div>
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">轮询与下载</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ videoPollExample }}</pre>
          </div>
        </div>

        <div class="mt-4 rounded-2xl border border-sky-200 bg-sky-50 p-4 text-sm leading-6 text-sky-900 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-100">
          当前页面与下载文档一致：视频示例统一使用 `duration`、`aspect_ratio`、`resolution`，不要向视频接口发送 `size`。浏览器也不要直连 `vidgen.x.ai`，应通过 RelayQ 的 `/content` 路由获取视频内容。
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900 md:p-8">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-xl font-black text-slate-950 dark:text-white">5. 音频模型</h3>
            <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">MiMo 音频模型当前统一走 `POST /v1/chat/completions`。语音转写使用 `mimo-v2.5-asr`，配音/音色设计/声音克隆使用 `mimo-v2.5-tts*` 系列。</p>
          </div>
          <span class="rounded-full bg-emerald-50 px-3 py-1 text-xs font-bold text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300">POST /v1/chat/completions</span>
        </div>

        <div class="mt-5 grid gap-4 xl:grid-cols-2">
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">语音转写：mimo-v2.5-asr</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ audioTranscriptionExample }}</pre>
          </div>
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">标准配音：mimo-v2.5-tts</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ audioSpeechExample }}</pre>
          </div>
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">音色设计：mimo-v2.5-tts-voicedesign</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ audioVoiceDesignExample }}</pre>
          </div>
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">声音克隆：mimo-v2.5-tts-voiceclone</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ audioVoiceCloneExample }}</pre>
          </div>
        </div>
        <div class="mt-4 rounded-2xl border border-emerald-200 bg-emerald-50 p-4 text-sm leading-6 text-emerald-900 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-100">
          关键点：ASR 当前按官方兼容格式传 `input_audio.data` + `asr_options.language`；配音文本放在 `assistant` 消息里；风格/音色约束放在 `user` 消息里；标准配音用 `audio.voice` 传真实音色 ID；返回音频通常在 `choices[0].message.audio.data`。
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900 md:p-8">
        <h3 class="text-xl font-black text-slate-950 dark:text-white">6. 最小 cURL 清单</h3>
        <div class="mt-5 grid gap-4 xl:grid-cols-3">
          <article
            v-for="item in curlSamples"
            :key="item.title"
            class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800"
          >
            <div class="text-sm font-bold text-slate-900 dark:text-white">{{ item.title }}</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ item.code }}</pre>
          </article>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'

const introCards = [
  {
    title: 'Chat 生图桥接',
    endpoint: 'POST /v1/chat/completions',
    description: 'Agent 工具只会对话接口时，用 grok-imagine-image* 模型直接返回图片 Markdown。'
  },
  {
    title: '图片官方接口',
    endpoint: 'POST /v1/images/generations',
    description: 'gpt-image-* 用 size；grok-imagine-image* 用 aspect_ratio + resolution。'
  },
  {
    title: '视频异步接口',
    endpoint: 'POST /v1/videos/generations',
    description: 'grok-imagine-video 异步提交，轮询 request_id，并通过 /content 下载或预览。'
  },
  {
    title: '音频模型',
    endpoint: 'POST /v1/chat/completions',
    description: 'MiMo 音频模型统一走 chat/completions：支持转写、标准配音、音色设计和声音克隆。'
  }
]

const defaultBaseUrl = 'https://www.relayq.top/v1'
const agentDocDownloadUrl = '/relayq-agent-api-reference.md'

const authExample = `POST /v1/images/generations HTTP/1.1
Host: www.relayq.top
Authorization: Bearer sk-your-api-key
Content-Type: application/json`

const chatImageExample = `curl ${defaultBaseUrl}/chat/completions \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "grok-imagine-image-quality",
    "messages": [
      {"role": "user", "content": "仙魔大战，场面宏大，人物面部清晰，东方玄幻电影海报"}
    ],
    "aspect_ratio": "16:9",
    "resolution": "2k",
    "quality": "high",
    "n": 1
  }'`

const chatImageResponseExample = `{
  "id": "chatcmpl-image-...",
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
}`

const imageRequestExample = `# gpt-image-2-adobe / gpt-image-* 请求体
curl ${defaultBaseUrl}/images/generations \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2-adobe",
    "prompt": "高端珠宝广告，模特侧脸特写，冷白珠宝反光，杂志封面质感",
    "size": "3:2",
    "quality": "high",
    "style": "natural",
    "background": "opaque",
    "n": 1
  }'

# grok-imagine-image* 请求体
curl ${defaultBaseUrl}/images/generations \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-image-quality",
    "prompt": "高端珠宝广告，模特侧脸特写，冷白珠宝反光，杂志封面质感",
    "n": 1,
    "aspect_ratio": "3:2",
    "resolution": "2k"
  }'`

const imageResponseExample = `{
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
}`

const imageEditExample = `# gpt-image-2-adobe 编辑请求体
curl ${defaultBaseUrl}/images/edits \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
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
  }'

# grok-imagine-image* 编辑请求体
curl ${defaultBaseUrl}/images/edits \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
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
  }'`

const videoTextSubmitExample = `curl ${defaultBaseUrl}/videos/generations \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "grok-imagine-video",
    "prompt": "仙魔大战，两位主角在云海战场交锋，电影级镜头",
    "duration": 10,
    "aspect_ratio": "16:9",
    "resolution": "720p"
  }'`

const videoImageSubmitExample = `curl ${defaultBaseUrl}/videos/generations \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-video",
    "prompt": "用这张图作为首帧，人物抬头看向镜头，头发被风吹动，镜头缓慢推进",
    "duration": 10,
    "aspect_ratio": "16:9",
    "resolution": "1080p",
    "image": {
      "url": "data:image/png;base64,..."
    }
  }'`

const videoReferenceSubmitExample = `curl ${defaultBaseUrl}/videos/generations \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-video",
    "prompt": "让参考图中的人物出现在宏大战场中，保持服装和脸部特征",
    "duration": 10,
    "aspect_ratio": "16:9",
    "resolution": "720p"
  }'`

const videoPollExample = `# 提交后返回
{
  "request_id": "video_req_123",
  "status": "queued"
}

# 轮询
curl ${defaultBaseUrl}/videos/video_req_123 \
  -H "Authorization: Bearer sk-your-api-key"

# 处理中
{
  "request_id": "video_req_123",
  "status": "processing",
  "progress": 65,
  "billing": {
    "amount": 0.8,
    "currency": "USD"
  }
}

# 完成后上游可能返回远程地址
{
  "status": "done",
  "request_id": "video_req_123",
  "video": {
    "url": "https://vidgen.x.ai/xai-vidgen-bucket/...mp4"
  },
  "progress": 100,
  "billing": {
    "amount": 0.8,
    "currency": "USD",
    "balance_after": 19.2
  }
}

# 不要让浏览器直连 vidgen.x.ai，应该使用 RelayQ content 路由
curl -L ${defaultBaseUrl}/videos/video_req_123/content \
  -H "Authorization: Bearer sk-your-api-key" \
  -o result.mp4`

const audioTranscriptionExample = `curl ${defaultBaseUrl}/chat/completions \
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \
  -d '{
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
  }'

# 典型返回
{
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "这是转写后的文本内容"
      }
    }
  ]
}`

const audioSpeechExample = `curl ${defaultBaseUrl}/chat/completions \
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "mimo-v2.5-tts",
    "audio": {
      "format": "wav",
      "voice": "mimo_default"
    },
    "messages": [
      {"role": "user", "content": "请用自然讲述风格朗读，输出语言：中文。"},
      {"role": "assistant", "content": "你好，这是一段语音合成测试。"}
    ],
    "stream": false
  }'

# 典型返回
{
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "",
        "audio": {
          "data": "UklGR...",
          "transcript": null
        },
        "final_text_preview": "你好，这是一段语音合成测试。"
      }
    }
  ]
}`

const audioVoiceDesignExample = `curl ${defaultBaseUrl}/chat/completions \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mimo-v2.5-tts-voicedesign",
    "audio": {
      "format": "wav",
      "optimize_text_preview": true
    },
    "messages": [
      {"role": "user", "content": "请生成温柔、稳定、适合品牌讲解的女声。角色设定：品牌讲解员。输出语言：中文。"},
      {"role": "assistant", "content": "欢迎使用 RelayQ，这里是你的品牌语音讲解助手。"}
    ],
    "stream": false
  }'`

const audioVoiceCloneExample = `curl ${defaultBaseUrl}/chat/completions \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mimo-v2.5-tts-voiceclone",
    "audio": {
      "format": "wav",
      "voice": "mimo_default"
    },
    "messages": [
      {"role": "user", "content": "请基于提供的音频样本进行声音克隆，保持自然稳定的发音。输出语言：中文。风格：自然讲述。"},
      {"role": "user", "content": [{"type": "audio_url", "audio_url": {"url": "data:audio/wav;base64,..."}}]},
      {"role": "assistant", "content": "这是一段用于声音克隆测试的目标文案。"}
    ],
    "stream": false
  }'`

const curlSamples = [
  {
    title: 'Chat 生图桥接',
    code: chatImageExample
  },
  {
    title: '图片官方接口',
    code: `${imageRequestExample}\n\n${imageEditExample}`
  },
  {
    title: '视频提交与轮询',
    code: `${videoTextSubmitExample}\n\n${videoImageSubmitExample}\n\n${videoReferenceSubmitExample}\n\n${videoPollExample}`
  },
  {
    title: '音频转写 / 语音合成',
    code: `${audioTranscriptionExample}

${audioSpeechExample}

${audioVoiceDesignExample}

${audioVoiceCloneExample}`
  }
]
</script>
