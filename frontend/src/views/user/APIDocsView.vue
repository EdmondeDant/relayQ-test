<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="bg-gradient-to-r from-indigo-500 via-violet-500 to-fuchsia-500 px-6 py-8 text-white md:px-8">
          <div class="max-w-4xl space-y-4">
            <div class="inline-flex items-center rounded-full border border-white/20 bg-white/10 px-3 py-1 text-xs font-semibold">
              OpenAI Compatible + Grok Imagine 媒体接口
            </div>
            <h2 class="text-3xl font-black tracking-tight md:text-4xl">接口文档</h2>
            <p class="max-w-3xl text-sm leading-7 text-white/90 md:text-base">
              本页面详细介绍各个模型的接口模式，如果人看不懂，请你让的agent看懂就行了。图片，音频，视频模型是子agents。不能当推理模型使用。
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
            <div class="flex flex-wrap gap-2 text-xs font-semibold text-white/90">
              <span class="rounded-full bg-white/10 px-3 py-1">Base URL: {{ defaultBaseUrl }}</span>
              <span class="rounded-full bg-white/10 px-3 py-1">Authorization: Bearer sk-xxx</span>
              <span class="rounded-full bg-white/10 px-3 py-1">Content-Type: application/json / multipart/form-data</span>
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
              <li>只把请求发到 RelayQ，不要直连 `api.x.ai`、`api.openai.com` 等上游。</li>
              <li>模型名称以你的分组白名单为准；没有权限或模型不在白名单会返回错误。</li>
              <li>图片/视频建议优先使用 JSON 官方格式；音频上传通常使用 multipart/form-data。</li>
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
          支持参数：`n`、`aspect_ratio/aspectRatio`、`resolution`（1k/2k）、`quality`、`user`，也可放在 `image_options`、`providerOptions.xai` 或 `provider_options.xai` 中。
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900 md:p-8">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-xl font-black text-slate-950 dark:text-white">3. 图片模型官方 JSON 接口</h3>
            <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">适用于标准文生图和图片编辑。Grok Imagine 官方不支持 `size`，请优先使用 `aspect_ratio` + `resolution`。</p>
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
            <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">Grok Imagine 视频是异步任务。RelayQ 支持官方 `/v1/videos/generations`，也兼容 OpenAI/Sora 风格 `/v1/videos`。</p>
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
          参数：`duration` 1-15 秒；参考图模式最大 10 秒；`aspect_ratio` 支持 1:1、16:9、9:16、4:3、3:4、3:2、2:3；`resolution` 支持 480p/720p，部分模型/模式支持 1080p。`size` 不是官方字段，RelayQ 会尽量转换成 `aspect_ratio + resolution`。
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900 md:p-8">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-xl font-black text-slate-950 dark:text-white">5. 音频模型</h3>
            <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">音频通常分两类：音频转文字与文本转语音。建议按 OpenAI Audio 兼容格式接入。</p>
          </div>
          <span class="rounded-full bg-emerald-50 px-3 py-1 text-xs font-bold text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300">/v1/audio/transcriptions / /v1/audio/speech</span>
        </div>

        <div class="mt-5 grid gap-4 xl:grid-cols-2">
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">转写示例</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ audioTranscriptionExample }}</pre>
          </div>
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">语音合成示例</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ audioSpeechExample }}</pre>
          </div>
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
    description: '同步返回图片结果，支持 n、aspect_ratio、resolution、quality、response_format。'
  },
  {
    title: '视频异步接口',
    endpoint: 'POST /v1/videos/generations',
    description: '支持文生视频、首帧图生视频、参考图视频，提交后轮询 request_id。'
  },
  {
    title: '音频模型',
    endpoint: '/v1/audio/transcriptions / /v1/audio/speech',
    description: '支持转写和语音合成，上传音频文件时通常使用 multipart/form-data。'
  }
]

const defaultBaseUrl = 'https://www.relayq.top/v1'
const agentDocDownloadUrl = '/relayq-agent-api-reference.md'

const authExample = `POST /v1/chat/completions HTTP/1.1
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

const imageRequestExample = `curl ${defaultBaseUrl}/images/generations \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "grok-imagine-image-quality",
    "prompt": "一张现代写实电影感人物照片，面部清晰",
    "n": 1,
    "aspect_ratio": "3:4",
    "resolution": "2k",
    "quality": "high",
    "response_format": "url"
  }'`

const imageResponseExample = `{
  "data": [
    {
      "url": "https://.../generated.png",
      "b64_json": null,
      "mime_type": "image/png",
      "revised_prompt": ""
    }
  ],
  "usage": {
    "cost_in_usd_ticks": 123456789
  }
}`

const imageEditExample = `curl ${defaultBaseUrl}/images/edits \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "grok-imagine-image-quality",
    "prompt": "把这张图改成铅笔素描，保留人物五官",
    "image": {
      "url": "data:image/png;base64,...",
      "type": "image_url"
    },
    "resolution": "2k",
    "response_format": "url"
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

const videoImageSubmitExample = `curl ${defaultBaseUrl}/videos/generations \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "grok-imagine-video",
    "prompt": "使用图片作为首帧，两人开始战斗，剑气和黑火碰撞",
    "duration": 10,
    "aspect_ratio": "16:9",
    "resolution": "720p",
    "image": {
      "url": "data:image/png;base64,..."
    }
  }'`

const videoReferenceSubmitExample = `curl ${defaultBaseUrl}/videos/generations \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "grok-imagine-video",
    "prompt": "让参考图中的人物出现在宏大战场中，保持服装和脸部特征",
    "duration": 10,
    "aspect_ratio": "16:9",
    "resolution": "720p",
    "reference_images": [
      {"url": "https://example.com/character.png"},
      {"url": "data:image/jpeg;base64,..."}
    ]
  }'`

const videoPollExample = `# 提交后返回
{
  "request_id": "21f7f4af-0fb0-9a85-aa10-69b0caa3b901"
}

# 轮询
curl ${defaultBaseUrl}/videos/21f7f4af-0fb0-9a85-aa10-69b0caa3b901 \\
  -H "Authorization: Bearer sk-your-api-key"

# 完成
{
  "status": "done",
  "video": {
    "url": "https://vidgen.x.ai/.../video.mp4",
    "duration": 10
  },
  "model": "grok-imagine-video",
  "progress": 100
}

# RelayQ 兼容下载入口
curl -L ${defaultBaseUrl}/videos/21f7f4af-0fb0-9a85-aa10-69b0caa3b901/content \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -o result.mp4`

const audioTranscriptionExample = `curl ${defaultBaseUrl}/audio/transcriptions \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -F "file=@demo.mp3" \\
  -F "model=whisper-1" \\
  -F "language=zh"

// 典型返回
{
  "text": "这是转写后的文本内容"
}`

const audioSpeechExample = `curl ${defaultBaseUrl}/audio/speech \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o-mini-tts",
    "voice": "alloy",
    "input": "你好，这是一段语音合成测试。",
    "format": "mp3"
  }'

// 返回通常为音频二进制流`

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
    code: `${videoTextSubmitExample}\n\n${videoImageSubmitExample}\n\n${videoPollExample}`
  },
  {
    title: '音频转写 / 语音合成',
    code: `${audioTranscriptionExample}\n\n${audioSpeechExample}`
  }
]
</script>
