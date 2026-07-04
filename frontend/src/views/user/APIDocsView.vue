<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="bg-gradient-to-r from-indigo-500 via-violet-500 to-fuchsia-500 px-6 py-8 text-white md:px-8">
          <div class="max-w-4xl space-y-4">
            <div class="inline-flex items-center rounded-full border border-white/20 bg-white/10 px-3 py-1 text-xs font-semibold">
              OpenAI Compatible + 异步任务说明
            </div>
            <h2 class="text-3xl font-black tracking-tight md:text-4xl">接口文档</h2>
            <p class="max-w-3xl text-sm leading-7 text-white/90 md:text-base">
              这一页用于说明如何接入本站的视频、图片、音频模型。默认以 `www.relayq.top` 作为 Base URL，
              鉴权统一使用 API Key。模型名以后台分组白名单和你当前账号实际开通情况为准。
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
                <span>左边这个文档投喂给你的 agents，人看不明白没关系，AI 看明白就行。</span>
              </span>
            </div>
            <div class="flex flex-wrap gap-2 text-xs font-semibold text-white/90">
              <span class="rounded-full bg-white/10 px-3 py-1">Base URL: {{ defaultBaseUrl }}</span>
              <span class="rounded-full bg-white/10 px-3 py-1">Authorization: Bearer sk-xxx</span>
              <span class="rounded-full bg-white/10 px-3 py-1">Content-Type: application/json / multipart/form-data</span>
            </div>
          </div>
        </div>

        <div class="grid gap-4 px-6 py-6 md:grid-cols-3 md:px-8">
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
              <li>你只需要把请求发到我们给你的站点地址，不需要自己再去对接上游厂商。</li>
              <li>接口里填写的模型名称，以你后台实际能看到和能选到的模型为准。</li>
              <li>图片和大多数普通请求直接按页面示例发 JSON 就可以，音频上传按表单上传文件即可。</li>
              <li>视频生成通常不是立刻返回成品，而是先提交任务，再按文档里的方式查询结果。</li>
            </ul>
          </div>
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900 md:p-8">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-xl font-black text-slate-950 dark:text-white">2. 图片模型</h3>
            <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">适用于文生图、图片编辑。当前推荐按 OpenAI Images 兼容格式接入。</p>
          </div>
          <span class="rounded-full bg-indigo-50 px-3 py-1 text-xs font-bold text-indigo-700 dark:bg-indigo-500/10 dark:text-indigo-300">POST /v1/images/generations</span>
        </div>

        <div class="mt-5 grid gap-4 xl:grid-cols-2">
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">请求示例</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ imageRequestExample }}</pre>
          </div>
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">返回示例</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ imageResponseExample }}</pre>
          </div>
        </div>

        <div class="mt-4 rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm leading-6 text-amber-900 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-100">
          常见模型示例：`grok-imagine-image`、`grok-imagine-image-quality`。如果分组未开启图片生成权限，接口会直接拒绝请求。
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900 md:p-8">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-xl font-black text-slate-950 dark:text-white">3. 视频模型</h3>
            <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">视频能力建议按异步任务格式接入：先提交任务，再轮询任务状态。</p>
          </div>
          <span class="rounded-full bg-fuchsia-50 px-3 py-1 text-xs font-bold text-fuchsia-700 dark:bg-fuchsia-500/10 dark:text-fuchsia-300">POST /v1/videos/generations</span>
        </div>

        <div class="mt-5 grid gap-4 xl:grid-cols-2">
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">提交任务</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ videoSubmitExample }}</pre>
          </div>
          <div class="rounded-2xl bg-slate-50 p-4 dark:bg-dark-800">
            <div class="text-sm font-bold text-slate-900 dark:text-white">轮询结果</div>
            <pre class="mt-3 overflow-x-auto rounded-xl bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ videoPollExample }}</pre>
          </div>
        </div>

        <div class="mt-4 rounded-2xl border border-sky-200 bg-sky-50 p-4 text-sm leading-6 text-sky-900 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-100">
          推荐模型示例：`grok-imagine-video`。当前正式网关支持 `POST /v1/videos/generations`、`POST /v1/videos/edits`、`POST /v1/videos/extensions`，提交成功后通常先得到 `request_id` 或任务状态，再通过 `GET /v1/videos/{request_id}` 轮询，直到拿到 `video.url`。官方成功态以 `done` 为准。
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900 md:p-8">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-xl font-black text-slate-950 dark:text-white">4. 音频模型</h3>
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

        <div class="mt-4 rounded-2xl border border-emerald-200 bg-emerald-50 p-4 text-sm leading-6 text-emerald-900 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-100">
          音频上传类接口一般需要 `multipart/form-data`；返回音频流时则通常是二进制内容，前端按文件下载或播放器方式处理即可。
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900 md:p-8">
        <h3 class="text-xl font-black text-slate-950 dark:text-white">5. 最小 cURL 清单</h3>
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
    title: '图片模型',
    endpoint: 'POST /v1/images/generations',
    description: '同步返回图片结果，常见返回字段为 data[].url 或 data[].b64_json。'
  },
  {
    title: '视频模型',
    endpoint: 'POST /v1/videos/generations',
    description: '异步任务模式，先提交任务，再用任务 ID 轮询状态和视频地址。'
  },
  {
    title: '音频模型',
    endpoint: '/v1/audio/transcriptions / /v1/audio/speech',
    description: '支持转写和语音合成两类格式，上传音频文件时通常使用 multipart/form-data。'
  }
]

const defaultBaseUrl = 'https://www.relayq.top/v1'
const agentDocDownloadUrl = '/relayq-agent-api-reference.md'

const authExample = `POST /v1/images/generations HTTP/1.1
Host: www.relayq.top
Authorization: Bearer sk-your-api-key
Content-Type: application/json`

const imageRequestExample = `curl ${defaultBaseUrl}/images/generations \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "grok-imagine-image-quality",
    "prompt": "一位美女坐在窗边吃苹果，电影感写实光影",
    "size": "1024x1024"
  }'`

const imageResponseExample = `{
  "created": 1750000000,
  "data": [
    {
      "url": "https://cdn.example.com/generated/abc.png"
    }
  ]
}`

const videoSubmitExample = `curl ${defaultBaseUrl}/videos/generations \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "grok-imagine-video",
    "prompt": "海边日落时，女生回头微笑，镜头缓慢推进",
    "duration": 5
  }'

// 典型返回
{
  "id": "video_req_123",
  "status": "pending"
}`

const videoPollExample = `curl ${defaultBaseUrl}/videos/video_req_123 \\
  -H "Authorization: Bearer sk-your-api-key"

// 典型完成返回
{
  "id": "video_req_123",
  "status": "done",
  "video": {
    "url": "https://cdn.example.com/generated/demo.mp4"
  }
}`

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
    title: '图片生成',
    code: imageRequestExample
  },
  {
    title: '视频提交与轮询',
    code: `${videoSubmitExample}\n\n${videoPollExample}`
  },
  {
    title: '音频转写 / 语音合成',
    code: `${audioTranscriptionExample}\n\n${audioSpeechExample}`
  }
]
</script>
