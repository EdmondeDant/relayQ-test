<template>
  <AppLayout>
    <div class="docs-page">
      <header class="docs-hero">
        <div>
          <span class="eyebrow">OPENAI-COMPATIBLE MEDIA API</span>
          <h1>RelayQ 图片与视频 API</h1>
          <p>
            使用一套 RelayQ API Key 调用图片生成、参考图编辑和异步视频任务。
            模型名称区分大小写，请直接复制本文列出的名称。
          </p>
        </div>
        <button class="copy-button" type="button" @click="copy(baseUrl, 'base')">
          {{ copied === 'base' ? '已复制' : '复制 Base URL' }}
        </button>
        <code class="base-url">{{ baseUrl }}</code>
      </header>

      <div class="docs-layout">
        <aside class="docs-sidebar">
          <button :class="navClass('overview')" type="button" @click="select('overview')">快速开始</button>

          <p class="nav-title">图片模型</p>
          <button
            v-for="model in imageModels"
            :key="model.id"
            :class="navClass(model.id)"
            type="button"
            @click="select(model.id)"
          >
            <span class="model-dot image-dot" />{{ model.id }}
          </button>

          <p class="nav-title">视频模型</p>
          <button
            v-for="model in videoModels"
            :key="model.id"
            :class="navClass(model.id)"
            type="button"
            @click="select(model.id)"
          >
            <span class="model-dot video-dot" />{{ model.id }}
          </button>

          <p class="nav-title">通用说明</p>
          <button :class="navClass('tasks')" type="button" @click="select('tasks')">任务查询与下载</button>
          <button :class="navClass('errors')" type="button" @click="select('errors')">错误与重试</button>
        </aside>

        <main class="docs-content">
          <section v-if="active === 'overview'" class="doc-section">
            <div class="section-heading">
              <span class="kind-badge overview-badge">快速开始</span>
              <h2>三步完成接入</h2>
              <p>准备一个属于多媒体分组的 RelayQ API Key，然后按下面的请求调用。</p>
            </div>

            <h3>连接信息</h3>
            <div class="info-grid">
              <InfoCard label="Base URL" :value="baseUrl" />
              <InfoCard label="鉴权" value="Authorization: Bearer YOUR_API_KEY" />
              <InfoCard label="请求格式" value="Content-Type: application/json" />
              <InfoCard label="建议超时" value="图片 ≥ 5 分钟；视频 ≥ 15 分钟" />
            </div>

            <h3>接口一览</h3>
            <div class="table-wrap">
              <table class="param-table">
                <thead><tr><th>方法</th><th>路径</th><th>说明</th></tr></thead>
                <tbody>
                  <tr><td><MethodBadge method="GET" /></td><td><code>/v1/models</code></td><td>查询当前 API Key 可用模型</td></tr>
                  <tr><td><MethodBadge method="POST" /></td><td><code>/v1/images/generations</code></td><td>文生图或通过 URL 提供参考图</td></tr>
                  <tr><td><MethodBadge method="POST" /></td><td><code>/v1/images/edits</code></td><td>JSON URL 或 multipart 文件方式编辑图片</td></tr>
                  <tr><td><MethodBadge method="POST" /></td><td><code>/v1/videos/generations</code></td><td>文生视频、图生视频或多素材参考</td></tr>
                  <tr><td><MethodBadge method="GET" /></td><td><code>/v1/videos/{task_id}</code></td><td>查询异步任务状态</td></tr>
                  <tr><td><MethodBadge method="GET" /></td><td><code>/v1/videos/{task_id}/content</code></td><td>下载完成的视频</td></tr>
                </tbody>
              </table>
            </div>

            <h3>第一步：查询模型</h3>
            <CodeBlock id="models" :code="modelsExample" :copied="copied" @copy="copy" />
            <p class="hint">返回结果会按 API Key 所属分组过滤。调用时请使用本文中的精确模型名称。</p>

            <h3>第二步：生成图片</h3>
            <CodeBlock id="quick-image" :code="quickImageExample" :copied="copied" @copy="copy" />

            <h3>第三步：生成视频</h3>
            <CodeBlock id="quick-video" :code="quickVideoExample" :copied="copied" @copy="copy" />
            <div class="notice">
              <strong>视频通常是异步任务。</strong>响应中取得 <code>task_id</code> 或 <code>id</code> 后，使用状态接口轮询；
              不要通过重复提交生成请求来“查询进度”。
            </div>
          </section>

          <section v-else-if="activeImageModel" class="doc-section">
            <div class="section-heading">
              <span class="kind-badge image-badge">图片模型</span>
              <h2>{{ activeImageModel.id }}</h2>
              <p>{{ activeImageModel.summary }}</p>
            </div>

            <div class="cap-grid">
              <InfoCard label="供应商" :value="activeImageModel.vendor" />
              <InfoCard label="推荐场景" :value="activeImageModel.recommendation" />
              <InfoCard label="接口" value="POST /v1/images/generations" />
              <InfoCard label="返回" value="OpenAI data[] 图片结果" />
            </div>

            <h3>请求参数</h3>
            <div class="table-wrap"><table class="param-table">
              <thead><tr><th>参数</th><th>类型</th><th>必填</th><th>说明</th></tr></thead>
              <tbody>
                <tr><td><code>model</code></td><td>string</td><td>是</td><td>固定填写 <code>{{ activeImageModel.id }}</code></td></tr>
                <tr><td><code>prompt</code></td><td>string</td><td>是</td><td>图片描述或编辑指令</td></tr>
                <tr><td><code>n</code></td><td>integer</td><td>否</td><td>生成数量，建议从 1 开始</td></tr>
                <tr><td><code>aspect_ratio</code></td><td>string</td><td>否</td><td><code>1:1</code>、<code>16:9</code>、<code>9:16</code>、<code>4:3</code></td></tr>
                <tr><td><code>size</code></td><td>string</td><td>否</td><td>例如 <code>1024x1024</code>；与 aspect_ratio 二选一</td></tr>
                <tr><td><code>quality</code></td><td>string</td><td>否</td><td>模型支持时可填写 <code>low</code>、<code>medium</code>、<code>high</code></td></tr>
                <tr><td><code>image_url</code></td><td>string</td><td>否</td><td>单张公网参考图 URL</td></tr>
                <tr><td><code>image_urls</code></td><td>string[]</td><td>否</td><td>多张公网参考图 URL，具体上限以模型响应为准</td></tr>
              </tbody>
            </table></div>

            <h3>文生图示例</h3>
            <CodeBlock
              :id="`image-${activeImageModel.id}`"
              :code="imageCurl(activeImageModel)"
              :copied="copied"
              @copy="copy"
            />

            <h3>参考图编辑（JSON URL）</h3>
            <p class="hint">参考图必须能从 RelayQ 服务器公网访问。需要上传本地文件时，请使用下面的 multipart 示例。</p>
            <CodeBlock
              :id="`image-ref-${activeImageModel.id}`"
              :code="imageReferenceCurl(activeImageModel)"
              :copied="copied"
              @copy="copy"
            />

            <h3>本地文件编辑（multipart/form-data）</h3>
            <CodeBlock
              :id="`image-edit-${activeImageModel.id}`"
              :code="imageEditCurl(activeImageModel)"
              :copied="copied"
              @copy="copy"
            />
          </section>

          <section v-else-if="activeVideoModel" class="doc-section">
            <div class="section-heading">
              <span class="kind-badge video-badge">视频模型</span>
              <h2>{{ activeVideoModel.id }}</h2>
              <p>{{ activeVideoModel.summary }}</p>
            </div>

            <div class="cap-grid">
              <InfoCard label="供应商" :value="activeVideoModel.vendor" />
              <InfoCard label="分辨率" :value="activeVideoModel.resolutions.join(' / ')" />
              <InfoCard label="时长" :value="durationLabel(activeVideoModel)" />
              <InfoCard label="原生音频" :value="activeVideoModel.audio ? '支持' : '不支持'" />
              <InfoCard label="图片参考" :value="activeVideoModel.maxImages ? `最多 ${activeVideoModel.maxImages} 张` : '首帧/尾帧'" />
              <InfoCard label="接口" value="POST /v1/videos/generations" />
            </div>

            <div v-if="activeVideoModel.note" class="notice"><strong>素材组合限制：</strong>{{ activeVideoModel.note }}</div>

            <h3>请求参数</h3>
            <div class="table-wrap"><table class="param-table">
              <thead><tr><th>参数</th><th>类型</th><th>必填</th><th>说明</th></tr></thead>
              <tbody>
                <tr><td><code>model</code></td><td>string</td><td>是</td><td>固定填写 <code>{{ activeVideoModel.id }}</code></td></tr>
                <tr><td><code>prompt</code></td><td>string</td><td>是</td><td>视频内容、镜头运动与风格描述</td></tr>
                <tr><td><code>resolution</code></td><td>string</td><td>否</td><td>{{ activeVideoModel.resolutions.join(' / ') }}</td></tr>
                <tr><td><code>duration</code></td><td>integer</td><td>否</td><td>{{ durationLabel(activeVideoModel) }}；也接受别名 <code>seconds</code></td></tr>
                <tr><td><code>aspect_ratio</code></td><td>string</td><td>否</td><td><code>16:9</code>、<code>9:16</code>、<code>1:1</code></td></tr>
                <tr v-if="activeVideoModel.audio"><td><code>audio</code></td><td>boolean</td><td>否</td><td>是否生成原生音频</td></tr>
                <tr><td><code>image_url</code></td><td>string</td><td>否</td><td>首张参考图或图生视频输入</td></tr>
                <tr><td><code>start_image_url</code></td><td>string</td><td>否</td><td>显式指定首帧</td></tr>
                <tr><td><code>end_image_url</code></td><td>string</td><td>否</td><td>显式指定尾帧</td></tr>
                <tr v-if="activeVideoModel.maxImages"><td><code>image_urls</code></td><td>string[]</td><td>否</td><td>多图参考，最多 {{ activeVideoModel.maxImages }} 张</td></tr>
                <tr v-if="activeVideoModel.maxImages"><td><code>image_strengths</code></td><td>string[]</td><td>否</td><td>与图片一一对应：<code>LOW</code> / <code>MID</code> / <code>HIGH</code></td></tr>
                <tr v-if="activeVideoModel.maxVideos"><td><code>video_urls</code></td><td>string[]</td><td>否</td><td>视频参考，最多 {{ activeVideoModel.maxVideos }} 个</td></tr>
                <tr v-if="activeVideoModel.maxAudios"><td><code>audio_urls</code></td><td>string[]</td><td>否</td><td>音频参考，最多 {{ activeVideoModel.maxAudios }} 个</td></tr>
              </tbody>
            </table></div>

            <h3>文生视频示例</h3>
            <CodeBlock
              :id="`video-${activeVideoModel.id}`"
              :code="videoCurl(activeVideoModel)"
              :copied="copied"
              @copy="copy"
            />

            <h3>首帧 / 图生视频示例</h3>
            <CodeBlock
              :id="`video-ref-${activeVideoModel.id}`"
              :code="videoReferenceCurl(activeVideoModel)"
              :copied="copied"
              @copy="copy"
            />
          </section>

          <section v-else-if="active === 'tasks'" class="doc-section">
            <div class="section-heading">
              <span class="kind-badge overview-badge">异步任务</span>
              <h2>查询状态与下载视频</h2>
              <p>创建视频后保存响应中的 <code>task_id</code> 或 <code>id</code>。建议每 3–5 秒查询一次。</p>
            </div>
            <CodeBlock id="task-flow" :code="taskExample" :copied="copied" @copy="copy" />

            <h3>常见状态</h3>
            <div class="table-wrap"><table class="param-table">
              <thead><tr><th>状态</th><th>含义</th><th>客户端动作</th></tr></thead>
              <tbody>
                <tr><td><code>queued</code></td><td>排队等待执行</td><td>等待后继续轮询</td></tr>
                <tr><td><code>processing</code></td><td>正在生成</td><td>继续轮询，不要重复提交</td></tr>
                <tr><td><code>completed</code></td><td>生成完成</td><td>读取结果 URL 或下载 content</td></tr>
                <tr><td><code>failed</code></td><td>生成失败</td><td>记录 error，修正参数后使用新的幂等键重试</td></tr>
                <tr><td><code>cancelled</code></td><td>任务取消</td><td>停止轮询</td></tr>
              </tbody>
            </table></div>

            <h3>Python 轮询示例</h3>
            <CodeBlock id="python-poll" :code="pythonExample" :copied="copied" @copy="copy" />
          </section>

          <section v-else class="doc-section">
            <div class="section-heading">
              <span class="kind-badge error-badge">错误处理</span>
              <h2>错误、幂等与安全重试</h2>
              <p>错误响应使用 OpenAI 风格 <code>error</code> 对象；请同时记录 HTTP 状态码和响应体。</p>
            </div>

            <CodeBlock id="error" :code="errorExample" :copied="copied" @copy="copy" />
            <div class="table-wrap"><table class="param-table">
              <thead><tr><th>HTTP</th><th>常见原因</th><th>处理建议</th></tr></thead>
              <tbody>
                <tr><td><code>400</code></td><td>参数、素材组合或模型名称错误</td><td>按 error.message 修改请求，不要原样重试</td></tr>
                <tr><td><code>401</code></td><td>API Key 缺失或无效</td><td>检查 Bearer Header 和 Key 所属分组</td></tr>
                <tr><td><code>403</code></td><td>分组无模型权限或上游拒绝</td><td>确认 API Key 分组和模型名称</td></tr>
                <tr><td><code>409</code></td><td>幂等键复用但请求体不同</td><td>同一业务请求复用原值；新请求生成新值</td></tr>
                <tr><td><code>429</code></td><td>并发、速率或额度限制</td><td>读取 Retry-After，指数退避</td></tr>
                <tr><td><code>5xx</code></td><td>RelayQ 或上游暂时不可用</td><td>查询既有任务状态；确认未创建后再重试</td></tr>
              </tbody>
            </table></div>

            <div class="notice">
              <strong>视频创建建议携带稳定的 Idempotency-Key。</strong>
              网络超时时，先查询已有 task_id；不要立刻使用不同幂等键重复创建，避免生成重复任务和重复计费。
            </div>
          </section>
        </main>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'

interface ImageModel {
  id: string
  vendor: string
  summary: string
  recommendation: string
  prompt: string
}

interface VideoModel {
  id: string
  vendor: string
  summary: string
  resolutions: string[]
  durations: number[]
  defaultDuration: number
  audio: boolean
  maxImages?: number
  maxVideos?: number
  maxAudios?: number
  note?: string
  prompt: string
}

const baseUrl = 'https://www.relayq.top/v1'
const active = ref('overview')
const copied = ref('')

const imageModels: ImageModel[] = [
  { id: 'GPT Image-2', vendor: 'OpenAI', summary: '细节、文字排版和参考图编辑能力均衡。', recommendation: '产品图、海报文字、精细编辑', prompt: 'A glass perfume bottle, studio lighting, premium product photography' },
  { id: 'Nano Banana', vendor: 'Google', summary: '强调生成速度和低成本的图片模型。', recommendation: '快速草图、批量素材', prompt: 'A minimalist product photograph on a soft beige background' },
  { id: 'Nano Banana 2', vendor: 'Google', summary: '适合复杂指令、多图理解与一致性编辑。', recommendation: '多图参考、复杂构图', prompt: 'A cinematic product composition with dramatic rim lighting' },
  { id: 'Nano Banana Pro', vendor: 'Google', summary: '高质量生成与复杂参考图编辑。', recommendation: '高质量成片、复杂编辑', prompt: 'A vibrant street market at golden hour, editorial photography' },
  { id: 'Seedream 4.5', vendor: 'ByteDance', summary: '中文提示词理解和写实图片表现突出。', recommendation: '中文场景、人物与写实摄影', prompt: '一个穿汉服的女孩站在樱花树下，写实摄影，柔和自然光' }
]

const videoModels: VideoModel[] = [
  { id: 'seedance-2.0', vendor: 'ByteDance', summary: '支持文生视频、首尾帧以及图片、视频、音频参考。', resolutions: ['480P', '720P', '1080P'], durations: range(4, 15), defaultDuration: 8, audio: true, maxImages: 4, maxVideos: 3, maxAudios: 1, prompt: 'A cat walking on a sunny beach, cinematic tracking shot' },
  { id: 'seedance-2.0-fast', vendor: 'ByteDance', summary: 'Seedance 2.0 的快速版本，适合批量生产。', resolutions: ['480P', '720P'], durations: range(4, 15), defaultDuration: 8, audio: true, maxImages: 4, maxVideos: 3, maxAudios: 1, prompt: 'A futuristic train crossing a neon city at night' },
  { id: 'seedance-2.0-mini', vendor: 'ByteDance', summary: '轻量快速版本，支持多模态参考与原生音频。', resolutions: ['480P', '720P'], durations: range(4, 15), defaultDuration: 8, audio: true, maxImages: 4, maxVideos: 3, maxAudios: 1, prompt: 'Ocean waves at sunset, slow cinematic camera movement' },
  { id: 'kling-3.0', vendor: 'Kling', summary: '动态表现强，支持首尾帧和多图参考。', resolutions: ['720P', '1080P'], durations: range(3, 15), defaultDuration: 9, audio: true, maxImages: 7, note: '首尾帧与 image_urls 多图参考不要同时使用。', prompt: 'A dragon flying over ancient mountains, cinematic' },
  { id: 'kling-video-o-3', vendor: 'Kling', summary: '适合复杂多模态输入，支持图片与视频参考。', resolutions: ['720P', '1080P'], durations: range(3, 15), defaultDuration: 9, audio: true, maxImages: 7, maxVideos: 1, note: '首尾帧不能与多图或视频参考同时使用；使用视频参考时建议不超过 10 秒。', prompt: 'An astronaut floating above Earth, realistic cinematic lighting' },
  { id: 'wan-2.7', vendor: 'Alibaba', summary: '支持文生视频、首尾帧和多图参考，不生成原生音频。', resolutions: ['720P', '1080P'], durations: range(2, 10), defaultDuration: 6, audio: false, maxImages: 5, prompt: 'A koi fish swimming in a clear pond, gentle camera movement' }
]

const activeImageModel = computed(() => imageModels.find(model => model.id === active.value))
const activeVideoModel = computed(() => videoModels.find(model => model.id === active.value))

const modelsExample = `curl "${baseUrl}/models" \\\n  -H "Authorization: Bearer YOUR_API_KEY"`

const quickImageExample = `curl -X POST "${baseUrl}/images/generations" \\\n  -H "Authorization: Bearer YOUR_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{
    "model": "Nano Banana 2",
    "prompt": "a red apple on a white table, studio lighting",
    "n": 1,
    "aspect_ratio": "1:1"
  }'`

const quickVideoExample = `curl -X POST "${baseUrl}/videos/generations" \\\n  -H "Authorization: Bearer YOUR_API_KEY" \\\n  -H "Idempotency-Key: video-20260821-001" \\\n  -H "Content-Type: application/json" \\\n  -d '{
    "model": "seedance-2.0-mini",
    "prompt": "a cat walking on a sunny beach, cinematic",
    "resolution": "720P",
    "duration": 8,
    "aspect_ratio": "16:9",
    "audio": true
  }'`

const taskExample = `# 查询状态
curl "${baseUrl}/videos/TASK_ID" \\\n  -H "Authorization: Bearer YOUR_API_KEY"

# 完成后下载 MP4
curl "${baseUrl}/videos/TASK_ID/content" \\\n  -H "Authorization: Bearer YOUR_API_KEY" \\\n  -o result.mp4`

const pythonExample = `import time
import requests

BASE_URL = "${baseUrl}"
API_KEY = "YOUR_API_KEY"
TASK_ID = "TASK_ID"
headers = {"Authorization": f"Bearer {API_KEY}"}

while True:
    response = requests.get(f"{BASE_URL}/videos/{TASK_ID}", headers=headers, timeout=30)
    response.raise_for_status()
    task = response.json()
    status = task.get("status", "")
    print(status, task.get("progress"))

    if status in {"completed", "succeeded"}:
        video = requests.get(f"{BASE_URL}/videos/{TASK_ID}/content", headers=headers, timeout=300)
        video.raise_for_status()
        open("result.mp4", "wb").write(video.content)
        break
    if status in {"failed", "cancelled", "canceled"}:
        raise RuntimeError(task.get("error") or task)
    time.sleep(task.get("poll_after", 5) or 5)`

const errorExample = `{
  "error": {
    "message": "Unsupported video parameters for the selected model",
    "type": "invalid_request_error",
    "code": "invalid_request"
  }
}`

function range(start: number, end: number): number[] {
  return Array.from({ length: end - start + 1 }, (_, index) => start + index)
}

function select(id: string) {
  active.value = id
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function navClass(id: string) {
  return ['nav-item', { active: active.value === id }]
}

function durationLabel(model: VideoModel) {
  return `${model.durations[0]}–${model.durations[model.durations.length - 1]} 秒（默认 ${model.defaultDuration} 秒）`
}

function imageCurl(model: ImageModel) {
  return `curl -X POST "${baseUrl}/images/generations" \\\n  -H "Authorization: Bearer YOUR_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{
    "model": "${model.id}",
    "prompt": "${model.prompt}",
    "n": 1,
    "aspect_ratio": "1:1"
  }'`
}

function imageReferenceCurl(model: ImageModel) {
  return `curl -X POST "${baseUrl}/images/edits" \\\n  -H "Authorization: Bearer YOUR_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{
    "model": "${model.id}",
    "prompt": "Preserve the subject and change the background to a sunset beach",
    "images": [
      {"image_url": "https://example.com/input.png"}
    ],
    "aspect_ratio": "1:1",
    "n": 1
  }'`
}

function imageEditCurl(model: ImageModel) {
  return `curl -X POST "${baseUrl}/images/edits" \\\n  -H "Authorization: Bearer YOUR_API_KEY" \\\n  -F "model=${model.id}" \\\n  -F "prompt=Preserve the subject and change the background to a sunset beach" \\\n  -F "image=@input.png" \\\n  -F "size=1024x1024"`
}

function videoCurl(model: VideoModel) {
  return `curl -X POST "${baseUrl}/videos/generations" \\\n  -H "Authorization: Bearer YOUR_API_KEY" \\\n  -H "Idempotency-Key: ${model.id}-request-001" \\\n  -H "Content-Type: application/json" \\\n  -d '{
    "model": "${model.id}",
    "prompt": "${model.prompt}",
    "resolution": "${model.resolutions.includes('720P') ? '720P' : model.resolutions[0]}",
    "duration": ${model.defaultDuration},
    "aspect_ratio": "16:9"${model.audio ? ',\n    "audio": true' : ''}
  }'`
}

function videoReferenceCurl(model: VideoModel) {
  return `curl -X POST "${baseUrl}/videos/generations" \\\n  -H "Authorization: Bearer YOUR_API_KEY" \\\n  -H "Idempotency-Key: ${model.id}-image-request-001" \\\n  -H "Content-Type: application/json" \\\n  -d '{
    "model": "${model.id}",
    "prompt": "The camera slowly moves forward while the subject looks at the horizon",
    "resolution": "${model.resolutions.includes('720P') ? '720P' : model.resolutions[0]}",
    "duration": ${model.defaultDuration},
    "aspect_ratio": "16:9",
    "image_url": "https://example.com/start-frame.jpg"${model.audio ? ',\n    "audio": true' : ''}
  }'`
}

async function copy(value: string, id: string) {
  await navigator.clipboard.writeText(value)
  copied.value = id
  window.setTimeout(() => {
    if (copied.value === id) copied.value = ''
  }, 1500)
}

const MethodBadge = defineComponent({
  props: { method: { type: String, required: true } },
  setup: props => () => h('span', { class: ['method-badge', props.method.toLowerCase()] }, props.method)
})

const InfoCard = defineComponent({
  props: { label: { type: String, required: true }, value: { type: String, required: true } },
  setup: props => () => h('div', { class: 'info-card' }, [
    h('span', { class: 'info-label' }, props.label),
    h('strong', { class: 'info-value' }, props.value)
  ])
})

const CodeBlock = defineComponent({
  props: {
    id: { type: String, required: true },
    code: { type: String, required: true },
    copied: { type: String, required: true }
  },
  emits: ['copy'],
  setup: (props, { emit }) => () => h('div', { class: 'code-shell' }, [
    h('button', { class: 'code-copy', type: 'button', onClick: () => emit('copy', props.code, props.id) }, props.copied === props.id ? '已复制' : '复制'),
    h('pre', { class: 'code-block' }, h('code', props.code))
  ])
})
</script>

<style scoped>
.docs-page { @apply mx-auto max-w-7xl space-y-6 px-4 py-8; }
.docs-hero { @apply relative overflow-hidden rounded-2xl bg-gradient-to-br from-slate-950 via-primary-950 to-violet-950 p-7 text-white shadow-xl md:p-9; }
.docs-hero::after { content: ''; @apply pointer-events-none absolute -right-20 -top-24 h-64 w-64 rounded-full bg-primary-400/20 blur-3xl; }
.docs-hero h1 { @apply mt-3 text-3xl font-black tracking-tight md:text-4xl; }
.docs-hero p { @apply mt-3 max-w-3xl text-sm leading-7 text-white/75 md:text-base; }
.eyebrow { @apply text-xs font-bold tracking-[0.22em] text-primary-200; }
.base-url { @apply mt-6 block w-fit rounded-lg border border-white/15 bg-black/20 px-4 py-2 text-sm text-primary-100; }
.copy-button { @apply absolute right-6 top-6 z-10 rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-xs font-semibold text-white transition hover:bg-white/20; }
.docs-layout { @apply grid items-start gap-6 lg:grid-cols-[250px_minmax(0,1fr)]; }
.docs-sidebar { @apply flex gap-2 overflow-x-auto rounded-xl border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-800 lg:sticky lg:top-20 lg:max-h-[calc(100vh-6rem)] lg:flex-col lg:overflow-y-auto; }
.nav-title { @apply hidden px-2 pb-1 pt-3 text-[11px] font-bold uppercase tracking-wider text-gray-400 lg:block; }
.nav-item { @apply flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-left text-sm text-gray-700 transition hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-dark-700; }
.nav-item.active { @apply bg-primary-50 font-semibold text-primary-700 ring-1 ring-primary-200 dark:bg-primary-950/40 dark:text-primary-300 dark:ring-primary-800; }
.model-dot { @apply h-2 w-2 shrink-0 rounded-full; }
.image-dot { @apply bg-emerald-500; }
.video-dot { @apply bg-amber-500; }
.docs-content { @apply min-w-0; }
.doc-section { @apply rounded-2xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-800 md:p-8; }
.section-heading { @apply mb-7 border-b border-gray-100 pb-6 dark:border-dark-700; }
.section-heading h2 { @apply mt-2 text-2xl font-black text-gray-950 dark:text-white md:text-3xl; }
.section-heading p { @apply mt-2 max-w-3xl text-sm leading-7 text-gray-600 dark:text-gray-300; }
.doc-section h3 { @apply mb-3 mt-8 text-base font-bold text-gray-900 dark:text-white; }
.kind-badge { @apply inline-flex rounded-full px-2.5 py-1 text-xs font-bold; }
.overview-badge { @apply bg-primary-100 text-primary-700 dark:bg-primary-950 dark:text-primary-300; }
.image-badge { @apply bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300; }
.video-badge { @apply bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-300; }
.error-badge { @apply bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300; }
.info-grid, .cap-grid { @apply grid gap-3 sm:grid-cols-2 xl:grid-cols-3; }
.table-wrap { @apply overflow-x-auto rounded-xl border border-gray-200 dark:border-dark-700; }
.param-table { @apply w-full min-w-[680px] border-collapse text-left text-sm; }
.param-table th { @apply bg-gray-50 px-4 py-3 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:bg-dark-900 dark:text-gray-400; }
.param-table td { @apply border-t border-gray-100 px-4 py-3 align-top text-gray-600 dark:border-dark-700 dark:text-gray-300; }
.param-table code, .notice code, .section-heading code, .hint code { @apply rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-900 dark:bg-dark-700 dark:text-gray-100; }
.method-badge { @apply inline-flex min-w-12 justify-center rounded px-2 py-1 text-[11px] font-black text-white; }
.method-badge.get { @apply bg-sky-600; }
.method-badge.post { @apply bg-emerald-600; }
.info-card { @apply rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900/60; }
.info-label { @apply mb-1 block text-xs font-semibold text-gray-400; }
.info-value { @apply break-words text-sm text-gray-900 dark:text-white; }
.code-shell { @apply relative; }
.code-block { @apply max-h-[560px] overflow-auto rounded-xl border border-gray-800 bg-gray-950 p-4 pr-20 text-xs leading-6 text-gray-100 md:text-sm; }
.code-copy { @apply absolute right-3 top-3 z-10 rounded-md border border-white/15 bg-white/10 px-2.5 py-1.5 text-xs font-semibold text-white hover:bg-white/20; }
.hint { @apply mt-3 text-sm leading-6 text-gray-500 dark:text-gray-400; }
.notice { @apply mt-6 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm leading-6 text-amber-900 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200; }
</style>
