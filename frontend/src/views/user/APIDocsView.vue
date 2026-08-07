<template>
  <AppLayout>
    <div class="docs-shell">
      <aside class="docs-sidebar">
        <a class="brand" href="#top">
          <span class="brand-mark">R</span>
          <span><strong>RelayQ Images</strong><small>API DOCUMENTATION</small></span>
        </a>

        <nav class="docs-nav" aria-label="接口文档目录">
          <div class="nav-group">
            <span>开始使用</span>
            <a v-for="item in gettingStarted" :key="item.id" :href="`#${item.id}`">{{ item.label }}</a>
          </div>
          <div class="nav-group">
            <span>接口模式</span>
            <a href="#openai-mode">OpenAI 兼容模式</a>
            <a href="#raw-mode">Leonardo 原生模式</a>
          </div>
          <div class="nav-group">
            <span>图片模型</span>
            <div v-for="item in models" :key="item.slug" class="model-nav-item">
              <a :href="`#${item.slug}`"><i :class="item.tone">{{ item.initials }}</i>{{ item.name }}</a>
              <a class="protocol-nav-link" :href="`#${item.slug}-openai`">OpenAI 格式</a>
              <a class="protocol-nav-link" :href="`#${item.slug}-raw`">Leonardo 原生格式</a>
            </div>
          </div>
          <div class="nav-group">
            <span>任务与结果</span>
            <a href="#async-task">异步任务</a>
            <a href="#errors">错误响应</a>
          </div>
        </nav>
      </aside>

      <main class="docs-main">
        <section id="top" class="hero-section">
          <div class="hero-copy">
            <span class="eyebrow">RELAYQ · LEONARDO PRODUCTION API</span>
            <h1>图片生成 API</h1>
            <p>通过 RelayQ 中转站接入 Leonardo 图片模型。兼容 OpenAI Images API，也支持 Leonardo Production API 原生参数。</p>
            <div class="base-url">
              <span>Base URL</span>
              <code>{{ baseUrl }}</code>
              <button type="button" @click="copy(baseUrl)">{{ copied === baseUrl ? '已复制' : '复制' }}</button>
            </div>
          </div>
          <div class="hero-badges">
            <span>4 个已验证模型</span>
            <span>Bearer API Key</span>
            <span>同步 / 异步</span>
          </div>
        </section>

        <section id="quick-start" class="doc-section">
          <header><span>01</span><div><h2>快速开始</h2><p>所有请求都发送到 www.realyq.top，并使用 RelayQ API Key 鉴权。</p></div></header>
          <div class="two-columns">
            <article class="info-card">
              <h3>鉴权</h3>
              <p>在每个请求中发送 Bearer Token。不要把 API Key 暴露在浏览器前端或公开仓库中。</p>
              <CodeBlock title="HTTP Headers" :code="authHeaders" @copy="copy" />
            </article>
            <article class="info-card">
              <h3>选择接口模式</h3>
              <ul class="check-list">
                <li><b>OpenAI 兼容：</b>适合 OpenAI SDK、ComfyUI 插件和通用客户端。</li>
                <li><b>Leonardo Raw：</b>适合需要参考图、风格参考图和原生 Guidance 的高级客户。</li>
                <li><b>不要混用：</b>Raw 请求包含顶层 <code>parameters</code>；OpenAI 请求不包含。</li>
              </ul>
            </article>
          </div>
        </section>

        <section id="capability-matrix" class="doc-section">
          <header><span>02</span><div><h2>功能对照</h2><p>OpenAI 模式侧重兼容性，Leonardo 原生模式提供完整的参考图控制能力。</p></div></header>
          <div class="matrix-wrap">
            <table class="capability-table">
              <thead><tr><th>能力</th><th>OpenAI 兼容模式</th><th>Leonardo 原生模式</th></tr></thead>
              <tbody>
                <tr v-for="row in capabilityRows" :key="row.name"><td><b>{{ row.name }}</b><small>{{ row.note }}</small></td><td><StatusDot :value="row.openai" /></td><td><StatusDot :value="row.raw" /></td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <section id="openai-mode" class="doc-section protocol-section">
          <header><span>03</span><div><h2>OpenAI 兼容模式</h2><p>使用标准 Images 风格请求。RelayQ 负责模型映射、上传、异步轮询和计费。</p></div></header>
          <div class="endpoint-line"><b>POST</b><code>/v1/images/generations</code><span>文生图 · 默认同步返回</span></div>
          <div class="two-columns">
            <article class="info-card"><h3>文生图</h3><CodeBlock title="cURL" :code="openAITextExample" @copy="copy" /></article>
            <article class="info-card"><h3>响应</h3><CodeBlock title="200 · application/json" :code="openAIResponse" @copy="copy" /></article>
          </div>
          <div class="endpoint-line"><b>POST</b><code>/v1/images/edits</code><span>单图编辑 · multipart/form-data · 返回异步任务</span></div>
          <CodeBlock title="图片编辑 cURL" :code="openAIEditExample" @copy="copy" />
          <div class="notice warning"><b>OpenAI 模式限制</b><span>仅支持一张编辑图，不支持 mask。FLUX Schnell 不支持此编辑端点。精细的内容参考、风格参考和多参考图请使用 Leonardo 原生模式。</span></div>
        </section>

        <section id="raw-mode" class="doc-section protocol-section">
          <header><span>04</span><div><h2>Leonardo 原生模式</h2><p>请求结构与 Leonardo Production API v2 对齐，但地址仍使用 RelayQ。顶层存在 parameters 时自动进入 Raw 模式。</p></div></header>
          <div class="endpoint-line raw"><b>POST</b><code>/v1/images/generations</code><span>原生 JSON · 异步返回任务</span></div>
          <div class="two-columns">
            <article class="info-card"><h3>原生文生图</h3><CodeBlock title="cURL" :code="rawTextExample" @copy="copy" /></article>
            <article class="info-card"><h3>参考图片来源</h3><ul class="check-list"><li>直接使用已上传图片的 <code>id</code> + <code>type</code>。</li><li>使用 <code>image.source</code> 传 Data URI。</li><li>使用受控 HTTPS URL，RelayQ 会安全下载并上传。</li><li>原生字段会保留并提交 Leonardo v2。</li></ul></article>
          </div>
          <div class="notice success"><b>原生模式适合</b><span>图生图、多参考图、主体一致性、内容参考、风格参考，以及需要直接使用 Leonardo Guidance 参数的客户端。</span></div>
        </section>

        <section v-for="(item, index) in models" :id="item.slug" :key="item.slug" class="doc-section model-section">
          <header>
            <span>{{ String(index + 5).padStart(2, '0') }}</span>
            <div class="model-heading"><i :class="item.tone">{{ item.initials }}</i><div><h2>{{ item.name }}</h2><p><code>{{ item.slug }}</code> · {{ item.summary }}</p></div></div>
          </header>

          <div class="model-meta">
            <div><small>OpenAI 文生图</small><b>支持</b></div>
            <div><small>OpenAI 图片编辑</small><b :class="{ muted: !item.openAIEdit }">{{ item.openAIEdit ? '支持' : '不支持' }}</b></div>
            <div><small>Leonardo Raw</small><b>支持</b></div>
            <div><small>最大数量</small><b>{{ item.maxQuantity }} 张</b></div>
          </div>

          <div class="feature-grid">
            <article v-for="feature in item.features" :key="feature.title" class="feature-card"><span>{{ feature.icon }}</span><div><h3>{{ feature.title }}</h3><p>{{ feature.description }}</p></div></article>
          </div>

          <article :id="`${item.slug}-openai`" class="protocol-manual openai-manual">
            <div class="protocol-title">
              <span class="protocol-number">A</span>
              <div><h3>{{ item.name }} · OpenAI 兼容模式</h3><p>{{ item.openAIDescription }}</p></div>
              <span class="protocol-tag">适合新手 / OpenAI SDK</span>
            </div>
            <div class="endpoint-line"><b>POST</b><code>/v1/images/generations</code><span>application/json · 默认同步返回图片</span></div>

            <div class="beginner-steps">
              <div v-for="(step, stepIndex) in item.openAISteps" :key="step"><span>{{ stepIndex + 1 }}</span><p>{{ step }}</p></div>
            </div>

            <div class="manual-grid">
              <div>
                <h4>文生图请求参数</h4>
                <div class="parameter-table">
                  <div class="parameter-head"><span>字段</span><span>是否必填</span><span>说明</span></div>
                  <div v-for="parameter in item.openAIParameters" :key="parameter.name"><code>{{ parameter.name }}</code><b>{{ parameter.required }}</b><span>{{ parameter.description }}</span></div>
                </div>
              </div>
              <div>
                <h4>这个模式可以做什么</h4>
                <ul class="check-list capability-list"><li v-for="capability in item.openAICapabilities" :key="capability">{{ capability }}</li></ul>
                <div v-if="item.openAIEdit" class="mini-endpoint"><b>图片编辑</b><code>POST /v1/images/edits</code><span>multipart/form-data</span></div>
                <div v-else class="notice warning compact"><b>图片编辑不可用</b><span>该模型不能调用 OpenAI 图片编辑端点。需要参考图时请使用下面的 Leonardo 原生模式。</span></div>
              </div>
            </div>

            <div class="two-columns examples">
              <article class="info-card"><h3>完整文生图示例</h3><p>复制后只需要替换 API Key 和 prompt。</p><CodeBlock title="OpenAI JSON · cURL" :code="item.openAIExample" @copy="copy" /></article>
              <article class="info-card"><h3>同步成功响应</h3><p>response_format=url 时图片地址位于 data[0].url。</p><CodeBlock title="HTTP 200" :code="openAIResponse" @copy="copy" /></article>
            </div>
            <div v-if="item.openAIEdit" class="edit-example">
              <h4>{{ item.name }} 图片编辑示例</h4>
              <p>把本地图片作为 multipart 文件上传。重复 image 字段可上传多张，具体上限以本模型说明为准；不支持 mask，提交后返回异步任务。</p>
              <CodeBlock title="OpenAI Images Edits · cURL" :code="item.openAIEditExample" @copy="copy" />
            </div>
            <div class="notice warning"><b>本模型限制</b><span>{{ item.openAILimits }}</span></div>
          </article>

          <article :id="`${item.slug}-raw`" class="protocol-manual raw-manual">
            <div class="protocol-title">
              <span class="protocol-number">B</span>
              <div><h3>{{ item.name }} · Leonardo 原生模式</h3><p>{{ item.rawDescription }}</p></div>
              <span class="protocol-tag raw">高级功能 / 完整参考图</span>
            </div>
            <div class="endpoint-line raw"><b>POST</b><code>/v1/images/generations</code><span>application/json · 返回异步任务</span></div>

            <div class="beginner-steps raw-steps">
              <div v-for="(step, stepIndex) in item.rawSteps" :key="step"><span>{{ stepIndex + 1 }}</span><p>{{ step }}</p></div>
            </div>

            <div class="manual-grid">
              <div>
                <h4>原生请求参数</h4>
                <div class="parameter-table">
                  <div class="parameter-head"><span>字段</span><span>是否必填</span><span>说明</span></div>
                  <div v-for="parameter in item.rawParameters" :key="parameter.name"><code>{{ parameter.name }}</code><b>{{ parameter.required }}</b><span>{{ parameter.description }}</span></div>
                </div>
              </div>
              <div>
                <h4>这个模式可以做什么</h4>
                <ul class="check-list capability-list"><li v-for="capability in item.rawCapabilities" :key="capability">{{ capability }}</li></ul>
                <div class="image-source-help"><b>参考图怎么传？</b><span>新手最简单：在 image.source 中填写图片 Data URI 或可公开访问的 HTTPS 图片地址。RelayQ 会安全上传并替换为 Leonardo 图片 ID。</span></div>
              </div>
            </div>

            <div class="two-columns examples">
              <article class="info-card"><h3>{{ item.rawExampleTitle }}</h3><p>{{ item.rawExampleHelp }}</p><CodeBlock title="Leonardo Raw JSON" :code="item.rawExample" @copy="copy" /></article>
              <article class="info-card"><h3>异步提交响应</h3><p>保存 id，随后使用任务查询接口轮询状态。</p><CodeBlock title="HTTP 202" :code="rawTaskResponse" @copy="copy" /></article>
            </div>
            <div class="notice success"><b>原生模式要点</b><span>{{ item.rawLimits }}</span></div>
          </article>
        </section>

        <section id="async-task" class="doc-section">
          <header><span>09</span><div><h2>异步任务与结果</h2><p>Raw 模式、OpenAI 图片编辑或显式 async=true 会返回任务对象。</p></div></header>
          <div class="two-columns">
            <article class="info-card"><h3>查询任务</h3><CodeBlock title="GET" :code="taskPollExample" @copy="copy" /></article>
            <article class="info-card"><h3>读取图片</h3><CodeBlock title="GET" :code="taskContentExample" @copy="copy" /></article>
          </div>
        </section>

        <section id="errors" class="doc-section">
          <header><span>10</span><div><h2>错误响应</h2><p>请求失败时请记录 HTTP 状态码、error.code 和 request_id。</p></div></header>
          <CodeBlock title="错误示例" :code="errorExample" @copy="copy" />
        </section>
      </main>

      <aside class="docs-toc">
        <span>本页内容</span>
        <a href="#quick-start">快速开始</a><a href="#capability-matrix">功能对照</a><a href="#openai-mode">OpenAI 模式</a><a href="#raw-mode">Raw 模式</a><a href="#async-task">异步任务</a>
        <div class="toc-note"><b>API Host</b><code>www.realyq.top</code><small>所有示例均为 RelayQ 中转地址</small></div>
      </aside>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { defineComponent, h, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'

type SupportLevel = 'yes' | 'partial' | 'no'

const baseUrl = 'https://www.realyq.top/v1'
const copied = ref('')

const CodeBlock = defineComponent({
  props: { title: { type: String, required: true }, code: { type: String, required: true } },
  emits: ['copy'],
  setup(props, { emit }) {
    return () => h('div', { class: 'code-block' }, [
      h('div', { class: 'code-head' }, [h('span', props.title), h('button', { type: 'button', onClick: () => emit('copy', props.code) }, '复制')]),
      h('pre', [h('code', props.code)]),
    ])
  },
})

const StatusDot = defineComponent({
  props: { value: { type: String as () => SupportLevel, required: true } },
  setup(props) {
    const labels = { yes: '支持', partial: '部分支持', no: '不支持' }
    return () => h('span', { class: ['status-dot', props.value] }, labels[props.value])
  },
})

const gettingStarted = [{ id: 'quick-start', label: '快速开始' }, { id: 'capability-matrix', label: '功能对照' }]

const capabilityRows: Array<{ name: string; note: string; openai: SupportLevel; raw: SupportLevel }> = [
  { name: '文生图', note: 'Prompt 生成图片', openai: 'yes', raw: 'yes' },
  { name: '图生图 / 图片编辑', note: '上传图片后编辑', openai: 'partial', raw: 'yes' },
  { name: '多参考图', note: '最多 6 张，具体取决于模型', openai: 'no', raw: 'yes' },
  { name: '内容参考', note: '保持构图与主体内容', openai: 'no', raw: 'yes' },
  { name: '风格参考', note: '提取并复用视觉风格', openai: 'no', raw: 'yes' },
  { name: '同步返回', note: '直接返回 URL 或 Base64', openai: 'yes', raw: 'no' },
  { name: '原生参数透传', note: 'Leonardo v2 parameters', openai: 'no', raw: 'yes' },
]

const authHeaders = `Authorization: Bearer sk-your-relayq-api-key
Content-Type: application/json`

const openAITextExample = `curl ${baseUrl}/images/generations \\
  -H "Authorization: Bearer sk-your-relayq-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-image-2",
    "prompt": "电影感产品摄影，柔和轮廓光",
    "size": "1024x1024",
    "quality": "low",
    "n": 1,
    "response_format": "url"
  }'`

const openAIResponse = `{
  "created": 1786089600,
  "data": [
    { "url": "https://cdn.example/result.png" }
  ]
}`

const openAIEditExample = `curl ${baseUrl}/images/edits \\
  -H "Authorization: Bearer sk-your-relayq-api-key" \\
  -F "model=gpt-image-2" \\
  -F "prompt=保留主体，把背景改成夜晚城市" \\
  -F "image=@input.png" \\
  -F "size=1024x1024" \\
  -F "quality=low" \\
  -F "n=1"`

const gptImage2EditExample = `curl ${baseUrl}/images/edits \\
  -H "Authorization: Bearer sk-your-relayq-api-key" \\
  -F "model=gpt-image-2" \\
  -F "prompt=结合这些参考图，生成统一风格的商品广告" \\
  -F "image=@C:/images/product.png" \\
  -F "image=@C:/images/style.png" \\
  -F "image=@C:/images/composition.png" \\
  -F "size=1024x1024" \\
  -F "quality=medium" \\
  -F "n=1"`

const rawTextExample = `curl ${baseUrl}/images/generations \\
  -H "Authorization: Bearer sk-your-relayq-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "flux-schnell",
    "public": false,
    "parameters": {
      "prompt": "极简红色苹果，白色背景",
      "quantity": 1,
      "width": 1024,
      "height": 1024
    }
  }'`

const modelExample = (model: string, quality = 'low', size = '1024x1024') => `curl ${baseUrl}/images/generations \\
  -H "Authorization: Bearer sk-your-relayq-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${model}",
    "prompt": "高级产品海报，干净背景，专业灯光",
    "size": "${size}",
    "quality": "${quality}",
    "n": 1,
    "response_format": "url"
  }'`

const fluxRaw = `{
  "model": "flux-schnell",
  "public": false,
  "parameters": {
    "prompt": "保持主体构图，使用参考图的视觉风格",
    "quantity": 1,
    "width": 1024,
    "height": 1024,
    "guidances": {
      "content": [{
        "image": { "source": "data:image/png;base64,..." },
        "strength": "HIGH"
      }],
      "style": [{
        "image": { "id": "generated-image-id", "type": "GENERATED" },
        "strength": "ULTRA"
      }]
    }
  }
}`

const referenceRaw = (model: string, withStrength: boolean) => `{
  "model": "${model}",
  "public": false,
  "parameters": {
    "prompt": "保持人物特征，改成水彩电影海报",
    "quantity": 1,
    "width": ${model.includes('nano') ? 0 : 1024},
    "height": ${model.includes('nano') ? 0 : 1024},
    "guidances": {
      "image_reference": [{
        "image": { "source": "https://example.com/reference.png" }${withStrength ? ',\n        "strength": "MID"' : ''}
      }]
    }
  }
}`

const editExample = (model: string, quality = 'low') => `curl ${baseUrl}/images/edits \\
  -H "Authorization: Bearer sk-your-relayq-api-key" \\
  -F "model=${model}" \\
  -F "prompt=保留商品主体，把背景改成暖色摄影棚" \\
  -F "image=@C:/images/input.png" \\
  -F "size=1024x1024" \\
  -F "quality=${quality}" \\
  -F "n=1"`

const commonOpenAIParameters = (sizeDescription: string, qualityDescription: string, quantityDescription: string) => [
  { name: 'model', required: '是', description: '必须填写本章节给出的精确模型 slug。' },
  { name: 'prompt', required: '是', description: '生成要求，不能为空，最多 4000 个字符。' },
  { name: 'size', required: '否', description: `${sizeDescription}；默认 1024x1024。` },
  { name: 'quality', required: '否', description: `${qualityDescription}；默认 low。` },
  { name: 'n', required: '否', description: `${quantityDescription}；默认 1。` },
  { name: 'response_format', required: '否', description: 'url 返回图片地址；b64_json 返回 Base64。默认 b64_json。' },
  { name: 'async', required: '否', description: 'false 同步等待；true 立即返回异步任务。默认 false。' },
]

const commonRawParameters = (sizeDescription: string, quantityDescription: string) => [
  { name: 'model', required: '是', description: '顶层模型 slug。它决定参考图协议和计价规则。' },
  { name: 'public', required: '否', description: '建议固定为 false，避免将生成结果设为公开。' },
  { name: 'parameters.prompt', required: '是', description: '原生生成提示词，不能为空。' },
  { name: 'parameters.quantity', required: '是', description: quantityDescription },
  { name: 'parameters.width', required: '是', description: sizeDescription },
  { name: 'parameters.height', required: '是', description: sizeDescription },
  { name: 'parameters.guidances', required: '参考图时', description: '按本章节示例放置 content、style 或 image_reference。' },
]

const rawTaskResponse = `{
  "id": "gen_123",
  "status": "queued",
  "model": "gpt-image-2",
  "billing_status": "submitted"
}`

const models = [
  {
    name: 'FLUX Schnell', slug: 'flux-schnell', initials: 'FS', tone: 'purple', summary: '快速文生图，支持独立的内容参考与风格参考。', openAIEdit: false, maxQuantity: 1,
    sizes: '1024²、2048²', qualities: 'low / medium / high', references: 'Content 1 张 + Style 1 张',
    features: [
      { icon: 'T', title: '文生图', description: 'OpenAI 与 Raw 模式均支持。' },
      { icon: 'C', title: '内容参考', description: 'Raw content guidance，控制主体与构图。' },
      { icon: 'S', title: '风格参考', description: 'Raw style guidance，控制视觉风格。' },
    ],
    openAIDescription: '最简单的使用方式，只发送文字生成图片。此模型的 OpenAI 模式不接收参考图片。',
    openAISteps: ['准备 RelayQ API Key。', '将 model 固定为 flux-schnell。', '填写 prompt，按需选择 1024 或 2048 尺寸。', '发送请求并从 data[0].url 或 data[0].b64_json 读取图片。'],
    openAIParameters: commonOpenAIParameters('允许 1024x1024、2048x2048', '请求可填写 low、medium、high', '此模型只允许 n=1'),
    openAICapabilities: ['根据文字描述生成图片。', '同步返回 URL 或 Base64。', '设置 async=true 后改为异步任务。'],
    openAILimits: 'OpenAI 格式只支持文生图；n 必须为 1；不支持 /v1/images/edits，也不能在 OpenAI JSON 中传内容参考或风格参考。',
    openAIExample: modelExample('flux-schnell'), openAIEditExample: '',
    rawDescription: '使用 Leonardo v2 parameters，可分别控制内容参考和风格参考，也可以只做原生文生图。',
    rawSteps: ['准备一张内容图或风格图，可使用 Data URI、HTTPS URL，或已有 Leonardo 图片 ID。', '内容参考放入 parameters.guidances.content；风格参考放入 parameters.guidances.style。', '每种参考最多一张，并设置 strength。', '提交后保存任务 id，再轮询任务状态。'],
    rawParameters: [...commonRawParameters('必须为正方形，当前最大 2048×2048。', '必须为 1。'), { name: 'guidances.content', required: '否', description: '内容参考，最多 1 张，strength 支持 LOW / MID / HIGH。' }, { name: 'guidances.style', required: '否', description: '风格参考，最多 1 张，strength 支持 LOW / MID / HIGH / ULTRA / MAX。' }],
    rawCapabilities: ['原生文生图。', '内容参考：保持主体、轮廓和构图。', '风格参考：复用色彩、材质和视觉语言。', '同时使用一张内容图和一张风格图。'],
    rawExampleTitle: '内容参考 + 风格参考', rawExampleHelp: 'image.source 可换成完整 Data URI 或公开 HTTPS 图片地址。', rawExample: fluxRaw,
    rawLimits: 'content 与 style 各最多 1 张。content 不接受 ULTRA/MAX；style 可以使用 ULTRA/MAX。原生请求固定异步返回。',
  },
  {
    name: 'GPT Image 2', slug: 'gpt-image-2', initials: 'GI', tone: 'green', summary: '高质量图片生成与编辑，支持多张图片参考。', openAIEdit: true, maxQuantity: 8,
    sizes: '1024²、2048²、2880²', qualities: 'low / medium / high', references: 'Image Reference 最多 6 张',
    features: [
      { icon: 'T', title: '文生图', description: '三档质量和三档尺寸。' },
      { icon: 'E', title: '图片编辑', description: 'OpenAI multipart 单图编辑。' },
      { icon: 'R', title: '多参考图', description: 'Raw image_reference 最多 6 张，不传 strength。' },
    ],
    openAIDescription: '既可以直接文生图，也可以通过 OpenAI Images Edits 上传一张本地图片进行编辑。',
    openAISteps: ['文生图使用 /v1/images/generations；参考图生成使用 /v1/images/edits。', 'model 固定为 gpt-image-2，并重复 image 字段上传 1 到 6 张图片。', '从 low、medium、high 中选择质量，并选择输出尺寸。', '文生图默认同步返回；参考图请求返回异步任务。'],
    openAIParameters: commonOpenAIParameters('允许 1024x1024、2048x2048、2880x2880', '允许 low、medium、high', '允许 1 到 8'),
    openAICapabilities: ['文字生成图片。', '通过 multipart 重复 image 字段上传最多 6 张参考图。', '同步返回 URL 或 Base64。', '一次生成最多 8 张。'],
    openAILimits: 'OpenAI 图片编辑支持 1 到 6 个 image 文件，不支持 mask。GPT Image 2 的参考图不使用 strength；需要直接使用 Leonardo 图片 ID 时使用原生模式。',
    openAIExample: modelExample('gpt-image-2', 'medium'), openAIEditExample: gptImage2EditExample,
    rawDescription: '通过 image_reference 使用最多 6 张参考图，适合角色一致性、商品一致性和多图融合。',
    rawSteps: ['准备 1 到 6 张参考图片。', '把每张图片放入 parameters.guidances.image_reference 数组。', 'GPT Image 2 的 image_reference 不要传 strength。', '提交后保存任务 id，并使用任务查询接口等待完成。'],
    rawParameters: [...commonRawParameters('支持 1024、2048、2880 的正方形尺寸。', '允许 1 到 8。'), { name: 'parameters.quality', required: '否', description: '使用大写 LOW / MEDIUM / HIGH。' }, { name: 'guidances.image_reference', required: '参考图时', description: '最多 6 张；每项只放 image，不要放 strength。' }],
    rawCapabilities: ['原生文生图。', '单参考图编辑。', '最多 6 张参考图联合生成。', '使用 UPLOADED 或 GENERATED 图片 ID。'],
    rawExampleTitle: '多参考图生成', rawExampleHelp: '继续向 image_reference 数组追加对象即可增加参考图，最多 6 个。', rawExample: referenceRaw('gpt-image-2', false),
    rawLimits: 'image_reference 项目禁止传 strength。若传入 strength，请求会被拒绝。原生模式提交成功后返回 HTTP 202。',
  },
  {
    name: 'Nano Banana 2', slug: 'nano-banana-2', initials: 'NB', tone: 'yellow', summary: '灵活的图像生成和参考图编辑，支持匹配输入尺寸。', openAIEdit: true, maxQuantity: 8,
    sizes: '1K、2K（上游支持 4K）', qualities: '原生质量由分辨率决定', references: 'Image Reference 最多 6 张',
    features: [
      { icon: 'T', title: '文生图', description: 'OpenAI 与 Raw 模式均支持。' },
      { icon: 'E', title: '图生图', description: 'OpenAI multipart 单图编辑。' },
      { icon: 'R', title: '多参考图', description: 'Raw 支持 LOW / MID / HIGH 强度。' },
    ],
    openAIDescription: '适合通用文生图和单图编辑。Nano Banana 2 没有 low、medium、high 原生质量档，清晰度由输出分辨率决定。',
    openAISteps: ['model 固定为 nano-banana-2。', '使用 size 选择输出清晰度：当前 RelayQ 开放 1K 和 2K。', 'quality 填写 low 只是 OpenAI 兼容占位值，不代表生成低质量图片。', '文生图默认同步返回，编辑请求返回异步任务。'],
    openAIParameters: commonOpenAIParameters('允许 1024x1024（1K）、2048x2048（2K）；尺寸越大，细节越丰富且费用越高', '兼容字段填写 low；它不会传给 Leonardo，也不会降低图片质量', '允许 1 到 8'),
    openAICapabilities: ['文字生成 1K 或 2K 图片。', '上传一张本地图片进行编辑。', '同步返回 URL 或 Base64。', '一次生成最多 8 张。'],
    openAILimits: 'Nano Banana 2 官方通过尺寸区分 1K、2K、4K，不使用质量档。当前 RelayQ OpenAI 接口开放 1K 和 2K；quality=low 仅用于接口兼容和本地计价。多参考图和参考强度必须使用 Leonardo 原生模式。',
    openAIExample: modelExample('nano-banana-2', 'low', '2048x2048'), openAIEditExample: editExample('nano-banana-2'),
    rawDescription: '使用 image_reference 传入最多 6 张参考图，并为每张图片设置 LOW、MID 或 HIGH 强度。',
    rawSteps: ['准备 1 到 6 张参考图片。', '把图片放入 parameters.guidances.image_reference。', '为每张参考图设置 strength；不填时 RelayQ 使用 MID。', 'width 和 height 同时设为 0 时，输出尺寸匹配参考图。'],
    rawParameters: [...commonRawParameters('可填写 1024/2048；或 width=0 且 height=0 匹配参考图尺寸。', '允许 1 到 8。'), { name: 'guidances.image_reference', required: '参考图时', description: '最多 6 张，每项支持 LOW / MID / HIGH strength。' }],
    rawCapabilities: ['原生文生图，清晰度由 width 和 height 决定。', '官方支持 1K、2K、4K 多种宽高比。', '最多 6 张参考图并逐张设置参考强度。', '自动匹配参考图尺寸。'],
    rawExampleTitle: '参考图 + 匹配输入尺寸', rawExampleHelp: '示例中的 width=0、height=0 表示根据参考图片确定输出尺寸。', rawExample: referenceRaw('nano-banana-2', true),
    rawLimits: '该模型没有原生 quality 参数。图片清晰度取决于 width 和 height，不能通过 quality=high 提升。当前 RelayQ 已计价开放 1K、2K；参考图 strength 仅允许 LOW、MID、HIGH，省略时默认为 MID。',
  },
  {
    name: 'Nano Banana 2 Lite', slug: 'nano-banana-2-lite', initials: 'NL', tone: 'orange', summary: '低成本快速生成与参考图编辑。', openAIEdit: true, maxQuantity: 8,
    sizes: '约 1K 原生尺寸', qualities: '固定原生质量档', references: 'Image Reference 最多 6 张',
    features: [
      { icon: 'T', title: '快速文生图', description: '约 1K 原生分辨率，不是低质量模式。' },
      { icon: 'E', title: '图生图', description: 'OpenAI multipart 单图编辑。' },
      { icon: 'R', title: '多参考图', description: 'Raw 支持最多 6 张参考图。' },
    ],
    openAIDescription: '低成本快速模式，适合固定 1024×1024 的文生图和单图编辑。',
    openAISteps: ['model 固定为 nano-banana-2-lite。', 'OpenAI 兼容入口的 size 填写 1024x1024。', 'quality 填写 low 只是兼容占位值，不代表低质量模式。', '文生图调用 generations；单图编辑调用 edits。'],
    openAIParameters: commonOpenAIParameters('当前兼容入口填写 1024x1024', '兼容字段填写 low；Leonardo 原生模型没有质量档参数', '允许 1 到 8'),
    openAICapabilities: ['低成本快速文生图。', '上传一张本地图片进行编辑。', '同步返回 URL 或 Base64。', '一次生成最多 8 张。'],
    openAILimits: 'Lite 是固定原生质量、约 1K 输出的快速模型，不是 low 低质量模型。quality=low 只是 RelayQ 的 OpenAI 兼容占位值；当前不能通过 medium/high 提升清晰度。图片编辑不支持 mask。',
    openAIExample: modelExample('nano-banana-2-lite'), openAIEditExample: editExample('nano-banana-2-lite'),
    rawDescription: '原生模式保留低成本特性，同时开放最多 6 张参考图和参考强度控制。',
    rawSteps: ['准备参考图片并放入 image_reference 数组。', '每张图可设置 LOW、MID 或 HIGH strength。', 'width=0 且 height=0 可匹配参考图片尺寸。', '提交后通过任务 id 查询结果。'],
    rawParameters: [...commonRawParameters('使用 1024×1024；参考图模式也允许 width=0 且 height=0。', '允许 1 到 8。'), { name: 'guidances.image_reference', required: '参考图时', description: '最多 6 张，每项可设置 LOW / MID / HIGH strength。' }],
    rawCapabilities: ['原生文生图。', '最多 6 张参考图。', '参考强度控制。', '匹配参考图片尺寸。'],
    rawExampleTitle: '多参考图生成', rawExampleHelp: '可继续增加 image_reference 数组项，最多 6 张图片。', rawExample: referenceRaw('nano-banana-2-lite', true),
    rawLimits: '该模型没有原生 quality 参数，采用固定原生质量档。默认尺寸为 1024×1024，也支持官方列出的约 1K 横竖尺寸；参考图 strength 仅允许 LOW、MID、HIGH，不填时默认 MID。',
  },
]

const taskPollExample = `curl ${baseUrl}/media/generations/gen_123 \\
  -H "Authorization: Bearer sk-your-relayq-api-key"`
const taskContentExample = `curl -L "${baseUrl}/media/generations/gen_123/content?index=0" \\
  -H "Authorization: Bearer sk-your-relayq-api-key" \\
  -o result.png`
const errorExample = `{
  "error": {
    "type": "invalid_request_error",
    "code": "invalid_request",
    "message": "Unsupported model, size, or quality combination"
  },
  "request_id": "req_123"
}`

async function copy(value: string) {
  await navigator.clipboard.writeText(value)
  copied.value = value
  window.setTimeout(() => { if (copied.value === value) copied.value = '' }, 1500)
}
</script>

<style scoped>
.docs-shell { --ink: #172033; --muted: #687386; --line: #e4e8ef; --soft: #f6f8fb; display: grid; grid-template-columns: 245px minmax(0, 1fr) 180px; max-width: 1540px; min-height: calc(100vh - 9rem); margin: -1rem auto 0; color: var(--ink); }
.dark .docs-shell { --ink: #f4f6fb; --muted: #9aa4b5; --line: #303642; --soft: #20242c; }
.docs-sidebar { position: sticky; top: 5rem; height: calc(100vh - 6rem); overflow-y: auto; border-right: 1px solid var(--line); padding: 20px 20px 30px 4px; }
.brand { display: flex; align-items: center; gap: 11px; color: var(--ink); }.brand-mark { display: grid; width: 38px; height: 38px; place-items: center; border-radius: 11px; background: #6c5ce7; color: white; font-weight: 900; }.brand strong,.brand small { display: block; }.brand strong { font-size: 15px; }.brand small { margin-top: 2px; color: var(--muted); font-size: 9px; letter-spacing: .12em; }
.docs-nav { margin-top: 28px; }.nav-group { margin-bottom: 22px; }.nav-group > span { display: block; margin-bottom: 7px; color: #98a1b1; font-size: 11px; font-weight: 800; letter-spacing: .1em; text-transform: uppercase; }.nav-group a { display: flex; align-items: center; gap: 8px; margin: 2px 0; border-radius: 8px; padding: 7px 9px; color: var(--muted); font-size: 13px; }.nav-group a:hover { background: var(--soft); color: #6c5ce7; }.nav-group i,.model-heading i { display: grid; place-items: center; border-radius: 7px; color: white; font-style: normal; font-weight: 900; }.nav-group i { width: 23px; height: 23px; font-size: 9px; }
.model-nav-item { margin-bottom: 6px; }.nav-group .protocol-nav-link { margin: 0 0 0 40px; padding: 3px 7px; color: #9aa3b3; font-size: 11px; }
.purple { background: linear-gradient(135deg,#7c3aed,#a855f7); }.green { background: linear-gradient(135deg,#059669,#34d399); }.yellow { background: linear-gradient(135deg,#d97706,#facc15); }.orange { background: linear-gradient(135deg,#ea580c,#fb923c); }
.docs-main { min-width: 0; padding: 28px 42px 80px; }.hero-section { display: flex; align-items: flex-end; justify-content: space-between; gap: 24px; border-bottom: 1px solid var(--line); padding: 18px 0 36px; }.eyebrow { color: #6c5ce7; font-size: 10px; font-weight: 900; letter-spacing: .13em; }.hero-copy h1 { margin: 12px 0 10px; font-size: clamp(32px,4vw,52px); font-weight: 900; letter-spacing: -.04em; }.hero-copy > p { max-width: 720px; color: var(--muted); line-height: 1.8; }.base-url { display: flex; max-width: 650px; align-items: center; gap: 10px; margin-top: 22px; border: 1px solid var(--line); border-radius: 11px; background: var(--soft); padding: 9px 11px; }.base-url span { color: var(--muted); font-size: 10px; font-weight: 800; text-transform: uppercase; }.base-url code { min-width: 0; flex: 1; color: #6555da; font-size: 12px; }.base-url button,.code-head button { color: #6c5ce7; font-size: 10px; font-weight: 800; }.hero-badges { display: flex; max-width: 180px; align-items: flex-end; flex-direction: column; gap: 7px; }.hero-badges span { border: 1px solid var(--line); border-radius: 999px; background: var(--soft); padding: 6px 10px; color: var(--muted); font-size: 9px; font-weight: 700; }
.doc-section { scroll-margin-top: 90px; border-bottom: 1px solid var(--line); padding: 42px 0; }.doc-section > header { display: flex; align-items: flex-start; gap: 15px; margin-bottom: 22px; }.doc-section > header > span { padding-top: 5px; color: #a8afbc; font-size: 11px; font-weight: 900; }.doc-section h2 { margin: 0; font-size: 25px; font-weight: 850; letter-spacing: -.02em; }.doc-section header p { margin: 7px 0 0; color: var(--muted); font-size: 14px; line-height: 1.7; }.two-columns { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 16px; }.info-card { min-width: 0; border: 1px solid var(--line); border-radius: 14px; background: var(--soft); padding: 17px; }.info-card h3 { margin: 0 0 10px; font-size: 14px; }.info-card > p { color: var(--muted); font-size: 13px; line-height: 1.7; }.check-list { margin: 0; padding: 0; list-style: none; }.check-list li { position: relative; margin: 9px 0; padding-left: 16px; color: var(--muted); font-size: 13px; line-height: 1.7; }.check-list li::before { position: absolute; top: 8px; left: 0; width: 5px; height: 5px; border-radius: 50%; background: #6c5ce7; content: ''; }.check-list code { color: #6555da; }
.code-block { overflow: hidden; border: 1px solid #293142; border-radius: 11px; background: #111827; }.code-head { display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid #2b3444; padding: 8px 11px; color: #94a3b8; font-size: 11px; font-weight: 700; }.code-head button { color: #c4b5fd; }.code-block pre { overflow-x: auto; margin: 0; padding: 14px; color: #dbe4f0; font: 11px/1.7 ui-monospace,SFMono-Regular,Consolas,monospace; white-space: pre; }
.matrix-wrap { overflow-x: auto; border: 1px solid var(--line); border-radius: 14px; }.capability-table { width: 100%; border-collapse: collapse; font-size: 12px; }.capability-table th,.capability-table td { border-bottom: 1px solid var(--line); padding: 13px 15px; text-align: left; }.capability-table th { background: var(--soft); color: var(--muted); font-size: 10px; text-transform: uppercase; }.capability-table td small { display: block; margin-top: 3px; color: var(--muted); }.status-dot { display: inline-flex; align-items: center; gap: 6px; font-weight: 700; }.status-dot::before { width: 7px; height: 7px; border-radius: 50%; content: ''; }.status-dot.yes { color: #059669; }.status-dot.yes::before { background: #10b981; }.status-dot.partial { color: #d97706; }.status-dot.partial::before { background: #f59e0b; }.status-dot.no { color: #94a3b8; }.status-dot.no::before { background: #cbd5e1; }
.endpoint-line { display: flex; align-items: center; gap: 11px; margin: 15px 0; border: 1px solid #d8d4fb; border-radius: 10px; background: #f5f3ff; padding: 10px 12px; }.dark .endpoint-line { border-color: #47406b; background: #28243b; }.endpoint-line b { border-radius: 5px; background: #6c5ce7; padding: 4px 7px; color: white; font-size: 9px; }.endpoint-line code { font-size: 12px; font-weight: 700; }.endpoint-line span { margin-left: auto; color: var(--muted); font-size: 10px; }.endpoint-line.raw b { background: #0891b2; }.notice { display: flex; gap: 12px; margin-top: 16px; border-radius: 11px; padding: 13px 15px; font-size: 11px; line-height: 1.7; }.notice b { flex: 0 0 auto; }.notice.warning { border: 1px solid #fde68a; background: #fffbeb; color: #92400e; }.notice.success { border: 1px solid #a7f3d0; background: #ecfdf5; color: #065f46; }.dark .notice.warning { border-color: #71551e; background: #362b17; color: #fde68a; }.dark .notice.success { border-color: #245d4b; background: #17352d; color: #a7f3d0; }
.model-heading { display: flex; align-items: center; gap: 12px; }.model-heading i { width: 42px; height: 42px; font-size: 12px; }.model-heading code { color: #6c5ce7; }.model-meta { display: grid; grid-template-columns: repeat(4,1fr); overflow: hidden; border: 1px solid var(--line); border-radius: 13px; }.model-meta div { border-right: 1px solid var(--line); padding: 13px; }.model-meta div:last-child { border: 0; }.model-meta small,.model-meta b { display: block; }.model-meta small { color: var(--muted); font-size: 9px; }.model-meta b { margin-top: 4px; color: #059669; font-size: 11px; }.model-meta b.muted { color: #94a3b8; }.feature-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 10px; margin-top: 13px; }.feature-card { display: flex; gap: 10px; border: 1px solid var(--line); border-radius: 12px; padding: 13px; }.feature-card > span { display: grid; flex: 0 0 27px; height: 27px; place-items: center; border-radius: 7px; background: #ede9fe; color: #6d5bd0; font-size: 10px; font-weight: 900; }.feature-card h3 { margin: 1px 0 4px; font-size: 11px; }.feature-card p { margin: 0; color: var(--muted); font-size: 10px; line-height: 1.55; }.examples { margin-top: 16px; }.parameter-strip { display: grid; grid-template-columns: repeat(3,1fr); margin-top: 13px; border-radius: 10px; background: var(--soft); }.parameter-strip div { padding: 11px 13px; }.parameter-strip span,.parameter-strip b { display: block; }.parameter-strip span { color: var(--muted); font-size: 9px; }.parameter-strip b { margin-top: 3px; font-size: 10px; }
.protocol-manual { scroll-margin-top: 90px; margin-top: 24px; border: 1px solid var(--line); border-radius: 16px; padding: 20px; }.openai-manual { border-top: 3px solid #6c5ce7; }.raw-manual { border-top: 3px solid #0891b2; }.protocol-title { display: flex; align-items: flex-start; gap: 11px; }.protocol-number { display: grid; flex: 0 0 30px; height: 30px; place-items: center; border-radius: 8px; background: #ede9fe; color: #6c5ce7; font-size: 12px; font-weight: 900; }.raw-manual .protocol-number { background: #cffafe; color: #0e7490; }.protocol-title h3 { margin: 1px 0 4px; font-size: 18px; }.protocol-title p { margin: 0; color: var(--muted); font-size: 13px; line-height: 1.65; }.protocol-tag { margin-left: auto; border-radius: 999px; background: #ede9fe; padding: 5px 9px; color: #6c5ce7; font-size: 10px; font-weight: 800; }.protocol-tag.raw { background: #cffafe; color: #0e7490; }.beginner-steps { display: grid; grid-template-columns: repeat(4,1fr); gap: 8px; margin: 14px 0; }.beginner-steps div { display: flex; align-items: flex-start; gap: 8px; border-radius: 10px; background: var(--soft); padding: 10px; }.beginner-steps span { display: grid; flex: 0 0 22px; height: 22px; place-items: center; border-radius: 50%; background: #6c5ce7; color: white; font-size: 10px; font-weight: 900; }.raw-steps span { background: #0891b2; }.beginner-steps p { margin: 1px 0 0; color: var(--muted); font-size: 12px; line-height: 1.6; }.manual-grid { display: grid; grid-template-columns: 1.2fr .8fr; gap: 16px; margin-top: 17px; }.manual-grid h4,.edit-example h4 { margin: 0 0 10px; font-size: 14px; }.parameter-table { overflow: hidden; border: 1px solid var(--line); border-radius: 10px; }.parameter-table > div { display: grid; grid-template-columns: 130px 70px minmax(0,1fr); border-bottom: 1px solid var(--line); }.parameter-table > div:last-child { border: 0; }.parameter-table span,.parameter-table b,.parameter-table code { padding: 8px 9px; font-size: 11px; line-height: 1.55; }.parameter-table code { color: #6555da; font-weight: 700; }.parameter-table b { color: #059669; }.parameter-table span { color: var(--muted); }.parameter-table .parameter-head { background: var(--soft); }.parameter-head span { color: var(--ink); font-weight: 800; }.capability-list { border-radius: 10px; background: var(--soft); padding: 5px 12px; }.mini-endpoint { display: flex; align-items: center; gap: 8px; margin-top: 10px; border: 1px solid var(--line); border-radius: 9px; padding: 9px; font-size: 11px; }.mini-endpoint b { color: #6c5ce7; }.mini-endpoint span { margin-left: auto; color: var(--muted); }.notice.compact { margin-top: 10px; padding: 9px 11px; }.edit-example { margin-top: 16px; border-radius: 12px; background: var(--soft); padding: 15px; }.edit-example > p { color: var(--muted); font-size: 12px; line-height: 1.6; }.image-source-help { margin-top: 10px; border: 1px solid #a5f3fc; border-radius: 10px; background: #ecfeff; padding: 11px; color: #155e75; font-size: 11px; line-height: 1.65; }.image-source-help b,.image-source-help span { display: block; }.image-source-help span { margin-top: 4px; }.dark .image-source-help { border-color: #155e75; background: #15343b; color: #a5f3fc; }
.docs-toc { position: sticky; top: 5rem; height: fit-content; border-left: 1px solid var(--line); padding: 28px 4px 20px 18px; }.docs-toc > span { color: #98a1b1; font-size: 9px; font-weight: 900; letter-spacing: .1em; text-transform: uppercase; }.docs-toc > a { display: block; margin-top: 10px; color: var(--muted); font-size: 10px; }.docs-toc > a:hover { color: #6c5ce7; }.toc-note { margin-top: 24px; border-radius: 10px; background: var(--soft); padding: 11px; }.toc-note b,.toc-note code,.toc-note small { display: block; }.toc-note b { font-size: 9px; }.toc-note code { margin-top: 5px; color: #6c5ce7; font-size: 9px; }.toc-note small { margin-top: 6px; color: var(--muted); font-size: 8px; line-height: 1.5; }
@media (max-width: 1180px) { .docs-shell { grid-template-columns: 210px minmax(0,1fr); }.docs-toc { display: none; }.docs-main { padding-right: 10px; }.beginner-steps { grid-template-columns: repeat(2,1fr); } }
@media (max-width: 820px) { .docs-shell { display: block; margin-top: 0; }.docs-sidebar { position: static; width: 100%; height: auto; border: 0; border-bottom: 1px solid var(--line); padding: 10px 0 14px; }.docs-nav { display: none; }.docs-main { padding: 22px 0 60px; }.hero-section { align-items: flex-start; flex-direction: column; }.hero-badges { max-width: none; align-items: flex-start; flex-direction: row; flex-wrap: wrap; }.two-columns,.feature-grid,.manual-grid,.beginner-steps { grid-template-columns: 1fr; }.model-meta { grid-template-columns: repeat(2,1fr); }.model-meta div:nth-child(2) { border-right: 0; }.parameter-strip { grid-template-columns: 1fr; }.endpoint-line { align-items: flex-start; flex-wrap: wrap; }.endpoint-line span { width: 100%; margin-left: 0; }.protocol-title { flex-wrap: wrap; }.protocol-tag { margin-left: 41px; }.protocol-manual { padding: 14px; }.parameter-table > div { grid-template-columns: 100px 55px minmax(0,1fr); } }
</style>
