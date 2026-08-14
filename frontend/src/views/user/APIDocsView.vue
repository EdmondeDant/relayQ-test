<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-8 px-4 py-8">
      <header class="rounded-2xl bg-gradient-to-br from-primary-600 to-violet-700 p-8 text-white">
        <span class="text-xs font-semibold tracking-[0.2em]">OPENAI-COMPATIBLE V1</span>
        <h1 class="mt-3 text-3xl font-bold">RelayQ 媒体 API</h1>
        <p class="mt-3 max-w-3xl text-white/80">客户侧统一使用 OpenAI-compatible /v1 接口。供应商协议、来源账号与模型映射由 RelayQ 封装，不需要发送 Leonardo 原生 parameters。</p>
        <div class="mt-6 flex items-center gap-3 rounded-lg bg-black/20 p-3"><code class="flex-1">{{ baseUrl }}</code><button class="btn bg-white text-primary-700" @click="copy(baseUrl)">{{ copied ? '已复制' : '复制' }}</button></div>
      </header>

      <nav class="flex flex-wrap gap-2"><a v-for="item in sections" :key="item.id" class="rounded-full border border-gray-200 px-4 py-2 text-sm dark:border-dark-600" :href="`#${item.id}`">{{ item.label }}</a></nav>

      <section id="authentication" class="doc-card"><h2>认证</h2><p>所有请求使用 RelayQ API Key。Base URL 可以包含 /v1，请勿重复拼接。</p><CodeBlock :code="authExample" /></section>
      <section id="models" class="doc-card"><h2>模型列表</h2><p>返回当前 API Key 入口分组已授权且可用的统一媒体商品，不暴露 Provider 或来源 Group。</p><Endpoint method="GET" path="/v1/models" /><CodeBlock :code="modelsExample" /></section>
      <section id="images" class="doc-card"><h2>图片生成与编辑</h2><Endpoint method="POST" path="/v1/images/generations" /><CodeBlock :code="imageExample" /><Endpoint method="POST" path="/v1/images/edits" /><CodeBlock :code="imageEditExample" /></section>
      <section id="videos" class="doc-card"><h2>视频生成、编辑与扩展</h2><p>创建类请求必须携带稳定 Idempotency-Key。重试同一业务请求时复用该值。</p><Endpoint method="POST" path="/v1/videos" /><Endpoint method="POST" path="/v1/videos/generations" /><CodeBlock :code="videoExample" /><Endpoint method="POST" path="/v1/videos/edits" /><Endpoint method="POST" path="/v1/videos/extensions" /></section>
      <section id="tasks" class="doc-card"><h2>任务查询与内容下载</h2><Endpoint method="GET" path="/v1/videos/{task_id}" /><Endpoint method="GET" path="/v1/videos/{task_id}/content" /><CodeBlock :code="taskExample" /></section>
      <section id="errors" class="doc-card"><h2>错误响应</h2><p>统一返回 OpenAI 风格 error 对象。不会将未知价格按零价处理，也不会在付费请求副作用未知时自动重发。</p><CodeBlock :code="errorExample" /><div class="mt-4 grid gap-2 md:grid-cols-2"><code v-for="code in errorCodes" :key="code" class="rounded bg-gray-100 p-2 dark:bg-dark-700">{{ code }}</code></div></section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'

const baseUrl = 'https://www.realyq.top/v1'
const copied = ref(false)
const sections = [{ id: 'authentication', label: '认证' }, { id: 'models', label: '模型' }, { id: 'images', label: '图片' }, { id: 'videos', label: '视频' }, { id: 'tasks', label: '任务' }, { id: 'errors', label: '错误' }]
const errorCodes = ['media_product_not_available', 'media_product_price_unavailable', 'unsupported_media_params', 'no_trusted_media_offer', 'media_offer_exhausted', 'media_submission_unknown']
const authExample = `Authorization: Bearer sk-your-relayq-api-key\nContent-Type: application/json`
const modelsExample = `curl ${baseUrl}/models \\\n+  -H "Authorization: Bearer sk-your-relayq-api-key"`
const imageExample = `curl ${baseUrl}/images/generations \\\n+  -H "Authorization: Bearer sk-your-relayq-api-key" \\\n+  -H "Content-Type: application/json" \\\n+  -d '{"model":"gpt-image-2","prompt":"电影感产品摄影","size":"1024x1024","quality":"low","n":1}'`
const imageEditExample = `curl ${baseUrl}/images/edits \\\n+  -H "Authorization: Bearer sk-your-relayq-api-key" \\\n+  -F "model=gpt-image-2" -F "prompt=把背景改成夜景" \\\n+  -F "image=@input.png" -F "size=1024x1024"`
const videoExample = `curl ${baseUrl}/videos/generations \\\n+  -H "Authorization: Bearer sk-your-relayq-api-key" \\\n+  -H "Idempotency-Key: video-request-001" \\\n+  -H "Content-Type: application/json" \\\n+  -d '{"model":"seedance-1.0-pro-fast","prompt":"海边日落延时摄影","seconds":4,"size":"1280x720"}'`
const taskExample = `curl ${baseUrl}/videos/task_123 \\\n+  -H "Authorization: Bearer sk-your-relayq-api-key"\n\ncurl ${baseUrl}/videos/task_123/content \\\n+  -H "Authorization: Bearer sk-your-relayq-api-key" -o result.mp4`
const errorExample = `{"error":{"message":"No fixed price matches the requested specification","type":"invalid_request_error","code":"media_product_price_unavailable"}}`
async function copy(value: string) { await navigator.clipboard.writeText(value); copied.value = true; window.setTimeout(() => { copied.value = false }, 1500) }
</script>

<script lang="ts">
import { defineComponent, h } from 'vue'
export const Endpoint = defineComponent({ props: { method: String, path: String }, setup: props => () => h('div', { class: 'my-4 flex items-center gap-3 rounded-lg border border-primary-200 bg-primary-50 p-3 dark:border-primary-800 dark:bg-primary-950/30' }, [h('b', { class: 'rounded bg-primary-600 px-2 py-1 text-xs text-white' }, props.method), h('code', props.path)]) })
export const CodeBlock = defineComponent({ props: { code: String }, setup: props => () => h('pre', { class: 'mt-4 overflow-x-auto rounded-xl bg-gray-950 p-4 text-sm text-gray-100' }, h('code', props.code)) })
export default { components: { Endpoint, CodeBlock } }
</script>

<style scoped>
.doc-card { @apply scroll-mt-6 rounded-2xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-800; }
.doc-card h2 { @apply mb-3 text-xl font-bold text-gray-900 dark:text-white; }
.doc-card p { @apply text-sm leading-7 text-gray-600 dark:text-gray-300; }
</style>
