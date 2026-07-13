<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-7xl flex-col gap-5 pb-8">
      <section class="overflow-hidden rounded-3xl border border-primary-200/70 bg-gradient-to-br from-primary-50 via-white to-purple-50 p-5 shadow-sm dark:border-primary-900/50 dark:from-primary-950/30 dark:via-dark-900 dark:to-purple-950/20 lg:p-7">
        <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div class="max-w-3xl">
            <div class="mb-3 inline-flex items-center gap-2 rounded-full border border-primary-200 bg-white/75 px-3 py-1 text-xs font-medium text-primary-700 shadow-sm dark:border-primary-800 dark:bg-dark-800/70 dark:text-primary-300">
              <span class="h-2 w-2 rounded-full bg-emerald-500"></span>
              无需 API Key，直接用账户余额体验
            </div>
            <h1 class="text-3xl font-bold tracking-tight text-gray-950 dark:text-dark-50 lg:text-4xl">在线体验</h1>
            <p class="mt-3 text-sm leading-6 text-gray-600 dark:text-dark-300 lg:text-base">
              先从一张图开始验证效果。精选模型、示例已填、费用可见；生成成功后再看实际扣费和调用文档。
            </p>
          </div>
          <div class="grid gap-3 sm:grid-cols-3 lg:min-w-[420px]">
            <div class="rounded-2xl bg-white/75 p-4 shadow-sm ring-1 ring-gray-200/70 dark:bg-dark-800/75 dark:ring-dark-700">
              <div class="text-xs text-gray-500 dark:text-dark-400">账户余额</div>
              <div class="mt-1 text-2xl font-bold text-gray-950 dark:text-dark-50">{{ formatMoney(balance) }}</div>
            </div>
            <RouterLink class="btn btn-secondary justify-center rounded-2xl py-4" :to="rechargeLink">充值</RouterLink>
            <RouterLink class="btn btn-secondary justify-center rounded-2xl py-4" to="/usage">用量</RouterLink>
          </div>
        </div>
      </section>

      <div class="grid min-h-[680px] gap-5 xl:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)]">
        <section class="card flex flex-col gap-5 p-5 lg:p-6">
          <div class="inline-flex rounded-2xl bg-gray-100 p-1 dark:bg-dark-800">
            <button
              v-for="item in tabs"
              :key="item.value"
              type="button"
              :class="[
                'flex-1 rounded-xl px-4 py-2 text-sm font-medium transition',
                activeTab === item.value
                  ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-700 dark:text-primary-300'
                  : 'text-gray-500 hover:text-gray-900 dark:text-dark-300 dark:hover:text-dark-50'
              ]"
              @click="activeTab = item.value"
            >
              {{ item.label }}
            </button>
          </div>

          <div v-if="activeTab === 'image'" class="flex flex-1 flex-col gap-5">
            <div>
              <label class="input-label">模型</label>
              <Select v-model="selectedImageModel" :options="imageModelOptions" searchable />
              <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">精选图片模型，避免全量模型选择困难。</p>
            </div>

            <div>
              <div class="mb-2 flex items-center justify-between gap-3">
                <label class="input-label mb-0">灵感示例</label>
                <button class="text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-300" type="button" @click="imagePrompt = currentImageModel.examples[0]">
                  恢复示例
                </button>
              </div>
              <div class="flex flex-wrap gap-2">
                <button
                  v-for="example in currentImageModel.examples"
                  :key="example"
                  type="button"
                  class="rounded-full border border-gray-200 bg-white px-3 py-1.5 text-xs text-gray-700 transition hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-200 dark:hover:border-primary-700 dark:hover:text-primary-300"
                  @click="imagePrompt = example"
                >
                  {{ exampleLabel(example) }}
                </button>
              </div>
            </div>

            <div class="flex flex-1 flex-col">
              <label class="input-label">描述你想要的画面</label>
              <textarea
                v-model="imagePrompt"
                class="input min-h-[180px] resize-none text-base leading-7"
                placeholder="例如：一张赛博朋克风格的城市夜景海报，霓虹灯，电影感构图..."
                :disabled="submitting"
              />
            </div>

            <details class="rounded-2xl border border-gray-200 bg-gray-50/70 p-4 dark:border-dark-700 dark:bg-dark-800/50">
              <summary class="cursor-pointer text-sm font-medium text-gray-800 dark:text-dark-100">更多选项</summary>
              <div class="mt-4">
                <label class="input-label">图片尺寸</label>
                <Select v-model="imageSize" :options="imageSizeOptions" />
              </div>
            </details>

            <div class="rounded-2xl border border-primary-100 bg-primary-50/70 p-4 text-sm dark:border-primary-900/50 dark:bg-primary-950/20">
              <div class="flex items-center justify-between gap-3">
                <span class="text-gray-600 dark:text-dark-300">预计费用</span>
                <span class="font-semibold text-gray-950 dark:text-dark-50">约 {{ formatMoney(currentImageModel.price) }}</span>
              </div>
              <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">以生成成功后的实际扣费为准；失败、取消或内容拦截不应扣费。</p>
            </div>

            <div class="grid gap-3 sm:grid-cols-[1fr_auto]">
              <button class="btn btn-primary rounded-2xl py-3 text-base" :disabled="!imagePrompt.trim() || submitting" @click="generateImage">
                {{ submitting && activeMode === 'image' ? '生成中…' : '生成图片' }}
              </button>
              <button v-if="submitting" class="btn btn-secondary rounded-2xl px-6" type="button" @click="stopGeneration">停止</button>
            </div>
          </div>

          <div v-else class="flex flex-1 flex-col gap-5">
            <div>
              <label class="input-label">模型</label>
              <Select v-model="selectedChatModel" :options="chatModelOptions" searchable />
              <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">对话保留轻量入口，主要用于快速确认文本模型响应。</p>
            </div>

            <div class="flex min-h-[280px] flex-1 flex-col gap-3 overflow-y-auto rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/50">
              <div v-if="!chatMessages.length" class="m-auto text-center text-sm text-gray-500 dark:text-dark-400">
                选一个示例或直接输入问题，开始对话。
              </div>
              <div v-for="(message, index) in chatMessages" :key="index" :class="['flex', message.role === 'user' ? 'justify-end' : 'justify-start']">
                <div :class="['max-w-[85%] rounded-2xl px-4 py-3 text-sm leading-6', message.role === 'user' ? 'bg-primary-600 text-white' : 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-dark-50']">
                  <div class="whitespace-pre-wrap break-words">{{ message.content || '…' }}</div>
                </div>
              </div>
            </div>

            <div class="flex flex-wrap gap-2">
              <button
                v-for="example in currentChatModel.examples"
                :key="example"
                type="button"
                class="rounded-full border border-gray-200 px-3 py-1.5 text-xs text-gray-600 hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:text-dark-300 dark:hover:border-primary-700"
                @click="chatInput = example"
              >
                {{ exampleLabel(example) }}
              </button>
            </div>

            <div class="flex gap-3">
              <textarea
                v-model="chatInput"
                class="input min-h-[56px] flex-1 resize-none"
                rows="2"
                placeholder="输入问题，Enter 发送，Shift + Enter 换行"
                :disabled="submitting"
                @keydown.enter.exact.prevent="sendChat"
              />
              <button v-if="submitting" class="btn btn-secondary self-end" type="button" @click="stopGeneration">停止</button>
              <button v-else class="btn btn-primary self-end" :disabled="!chatInput.trim()" @click="sendChat">发送</button>
            </div>
          </div>
        </section>

        <section class="card relative overflow-hidden p-5 lg:p-6">
          <div class="pointer-events-none absolute inset-x-0 top-0 h-36 bg-gradient-to-b from-primary-100/60 to-transparent dark:from-primary-950/20"></div>
          <div class="relative flex h-full flex-col gap-5">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 class="text-xl font-semibold text-gray-950 dark:text-dark-50">结果预览</h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">成功后会显示实扣金额和 request_id，方便对账与排查。</p>
              </div>
              <button v-if="lastResult?.imageUrl" class="btn btn-secondary btn-sm" type="button" @click="downloadImage">下载图片</button>
            </div>

            <div v-if="submitting" class="flex flex-1 items-center justify-center rounded-3xl border border-dashed border-primary-200 bg-primary-50/50 p-8 text-center dark:border-primary-900 dark:bg-primary-950/10">
              <div>
                <div class="mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-4 border-primary-200 border-t-primary-600 dark:border-primary-900 dark:border-t-primary-400"></div>
                <div class="text-lg font-semibold text-gray-900 dark:text-dark-50">正在生成…</div>
                <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">不展示假进度，等网关返回真实结果。</p>
              </div>
            </div>

            <div v-else-if="error" class="flex flex-1 items-center justify-center rounded-3xl border border-red-200 bg-red-50 p-8 text-center dark:border-red-900/60 dark:bg-red-950/20">
              <div class="max-w-md">
                <div class="text-lg font-semibold text-red-700 dark:text-red-300">生成失败</div>
                <p class="mt-2 text-sm leading-6 text-red-600 dark:text-red-200">{{ error }}</p>
                <p v-if="requestId" class="mt-4 text-xs text-red-500 dark:text-red-300">request_id：{{ requestId }}</p>
                <button class="btn btn-secondary mt-5" type="button" @click="retryLast">重试</button>
              </div>
            </div>

            <div v-else-if="lastResult?.imageUrl" class="flex flex-1 flex-col gap-4">
              <a :href="lastResult.imageUrl" target="_blank" rel="noreferrer" class="group flex flex-1 items-center justify-center overflow-hidden rounded-3xl border border-gray-200 bg-gray-100 dark:border-dark-700 dark:bg-dark-800">
                <img :src="lastResult.imageUrl" :alt="lastResult.prompt" class="max-h-[560px] w-full object-contain transition group-hover:scale-[1.01]" />
              </a>
              <div class="grid gap-3 lg:grid-cols-[1fr_auto]">
                <div class="rounded-2xl bg-gray-50 p-4 text-sm text-gray-600 dark:bg-dark-800/70 dark:text-dark-300">
                  <div class="font-medium text-gray-900 dark:text-dark-50">{{ lastResult.modelName }}</div>
                  <p class="mt-2 line-clamp-3">{{ lastResult.revisedPrompt || lastResult.prompt }}</p>
                  <p v-if="requestId" class="mt-3 text-xs text-gray-500 dark:text-dark-400">request_id：{{ requestId }}</p>
                </div>
                <div class="rounded-2xl bg-emerald-50 p-4 text-sm text-emerald-700 dark:bg-emerald-950/20 dark:text-emerald-300 lg:min-w-[190px]">
                  <div class="text-xs opacity-80">实扣回执</div>
                  <div class="mt-1 text-xl font-bold">{{ billingText }}</div>
                  <div v-if="lastBalanceText" class="mt-1 text-xs opacity-80">余额 {{ lastBalanceText }}</div>
                </div>
              </div>
            </div>

            <div v-else class="flex flex-1 items-center justify-center rounded-3xl border border-dashed border-gray-200 bg-gray-50/70 p-8 text-center dark:border-dark-700 dark:bg-dark-800/40">
              <div class="max-w-sm">
                <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-3xl bg-gradient-to-br from-primary-500 to-purple-500 text-2xl text-white shadow-lg">✨</div>
                <div class="text-lg font-semibold text-gray-900 dark:text-dark-50">准备好第一张图了</div>
                <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-dark-400">左侧示例已填好，点一次生成即可看到结果。这里不会出现 API Key 或上游密钥。</p>
              </div>
            </div>

            <div v-if="recentResults.length" class="border-t border-gray-200 pt-4 dark:border-dark-700">
              <div class="mb-3 flex items-center justify-between">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-dark-50">最近提示词</h3>
                <button class="text-xs text-gray-500 hover:text-red-500" type="button" @click="clearRecent">清空</button>
              </div>
              <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                <button
                  v-for="item in recentResults"
                  :key="item.id"
                  type="button"
                  class="rounded-2xl border border-gray-200 bg-white p-3 text-left text-xs transition hover:border-primary-300 dark:border-dark-700 dark:bg-dark-800 dark:hover:border-primary-700"
                  @click="restoreResult(item)"
                >
                  <div class="truncate font-medium text-gray-900 dark:text-dark-50">{{ item.modelName }}</div>
                  <div class="mt-1 line-clamp-2 text-gray-500 dark:text-dark-400">{{ item.prompt }}</div>
                </button>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select from '@/components/common/Select.vue'
import { modelTestAPI, type ChatMessage, type PlaygroundBilling } from '@/api/modelTest'
import { useAuthStore } from '@/stores/auth'

type TabValue = 'image' | 'chat'

interface CatalogModel {
  id: string
  name: string
  price: number
  examples: string[]
}

interface RecentResult {
  id: string
  imageUrl?: string
  prompt: string
  revisedPrompt?: string
  modelName: string
  billing?: PlaygroundBilling
  requestId?: string
}

const imageModels: CatalogModel[] = [
  {
    id: 'gpt-image-2',
    name: 'GPT Image 2 · 海报/产品图',
    price: 0.12,
    examples: [
      '为 RelayQ 设计一张高级感科技风海报：深色背景、蓝紫渐变光效、中央是发光的 AI 网关节点，画面干净、有商业发布会质感。',
      '一张奶茶新品产品图：透明杯、芒果芝士口味、夏日阳光、浅黄色背景、真实摄影质感、电商主图构图。',
    ],
  },
  {
    id: 'gpt-image-2-pro',
    name: 'GPT Image 2 Pro · 精细视觉',
    price: 0.18,
    examples: [
      '一个可爱的龙虾 AI 助手 IP 形象，红橙色，小巧聪明，戴着耳机，3D 毛绒玩具质感，白底。',
      '二次元赛博城市夜景，雨后街道、霓虹灯反射、远处高楼、电影感构图、细节丰富。',
    ],
  },
]

const chatModels: CatalogModel[] = [
  {
    id: 'deepseek-v4-flash',
    name: 'DeepSeek V4 Flash · 快速问答',
    price: 0.01,
    examples: ['用三句话介绍 RelayQ 的优势。', '帮我把这段提示词优化得更适合生成产品海报。'],
  },
  {
    id: 'gpt-5.4',
    name: 'GPT-5.4 · 创作润色',
    price: 0.03,
    examples: ['给我 5 个适合在线体验页的示例 prompt。', '把这句话改得更像面向小白用户的产品文案。'],
  },
]

const tabs: Array<{ value: TabValue; label: string }> = [
  { value: 'image', label: '图片' },
  { value: 'chat', label: '对话' },
]

const RECENT_KEY = 'relayq-playground:recent-results'

const authStore = useAuthStore()
const activeTab = ref<TabValue>('image')
const selectedImageModel = ref(imageModels[0].id)
const selectedChatModel = ref(chatModels[0].id)
const imagePrompt = ref(imageModels[0].examples[0])
const imageSize = ref('1024x1024')
const chatInput = ref(chatModels[0].examples[0])
const chatMessages = ref<ChatMessage[]>([])
const submitting = ref(false)
const activeMode = ref<TabValue | null>(null)
const error = ref('')
const requestId = ref('')
const lastResult = ref<RecentResult | null>(null)
const recentResults = ref<RecentResult[]>(loadRecentResults())
let abortController: AbortController | null = null

const imageModelOptions = computed(() => imageModels.map((model) => ({ value: model.id, label: model.name })))
const chatModelOptions = computed(() => chatModels.map((model) => ({ value: model.id, label: model.name })))
const imageSizeOptions = [
  { value: '1024x1024', label: '方图 1024×1024' },
  { value: '1024x1536', label: '竖图 1024×1536' },
  { value: '1536x1024', label: '横图 1536×1024' },
]

const currentImageModel = computed(() => imageModels.find((model) => model.id === selectedImageModel.value) || imageModels[0])
const currentChatModel = computed(() => chatModels.find((model) => model.id === selectedChatModel.value) || chatModels[0])
const balance = computed(() => authStore.user?.balance ?? 0)
const rechargeLink = computed(() => `/purchase?return_url=${encodeURIComponent('/model-test')}`)
const billingText = computed(() => {
  const amount = lastResult.value?.billing?.amount
  return typeof amount === 'number' ? formatMoney(amount) : '以账单为准'
})
const lastBalanceText = computed(() => {
  const amount = lastResult.value?.billing?.balance_after
  return typeof amount === 'number' ? formatMoney(amount) : ''
})
const legacyAuth = computed(() => ({ apiKey: localStorage.getItem('auth_token') || '' }))

watch(selectedImageModel, () => {
  imagePrompt.value = currentImageModel.value.examples[0]
})

async function generateImage(): Promise<void> {
  if (!imagePrompt.value.trim() || submitting.value) return
  const prompt = imagePrompt.value.trim()
  const model = currentImageModel.value
  const balanceBefore = balance.value
  startSubmit('image')
  try {
    const result = await modelTestAPI.generatePlaygroundImage({
      auth: legacyAuth.value,
      model: model.id,
      prompt,
      size: imageSize.value,
      signal: abortController?.signal,
    })
    const firstImage = result.images[0]
    if (!firstImage) throw new Error('生成成功但没有返回图片，请复制 request_id 联系客服排查。')
    requestId.value = result.requestId || ''
    const billing = await resolveBillingReceipt(result.billing, balanceBefore)
    const item: RecentResult = {
      id: createId(),
      imageUrl: firstImage.url,
      prompt,
      revisedPrompt: firstImage.revisedPrompt,
      modelName: model.name,
      billing,
      requestId: result.requestId,
    }
    lastResult.value = item
    saveRecent(item)
  } catch (err) {
    if (!isAbortError(err)) error.value = err instanceof Error ? err.message : '图片生成失败，本次不应扣费。'
  } finally {
    endSubmit()
  }
}

async function sendChat(): Promise<void> {
  const text = chatInput.value.trim()
  if (!text || submitting.value) return
  const balanceBefore = balance.value
  chatInput.value = ''
  error.value = ''
  requestId.value = ''
  chatMessages.value.push({ role: 'user', content: text })
  const assistantIndex = chatMessages.value.push({ role: 'assistant', content: '' }) - 1
  startSubmit('chat')
  try {
    await modelTestAPI.streamPlaygroundChat({
      auth: legacyAuth.value,
      model: selectedChatModel.value,
      messages: chatMessages.value.slice(0, assistantIndex),
      signal: abortController?.signal,
      onDelta(delta) {
        chatMessages.value[assistantIndex].content += delta
      },
      onBilling(billing, id) {
        requestId.value = id || requestId.value
        lastResult.value = {
          id: createId(),
          prompt: text,
          modelName: currentChatModel.value.name,
          billing,
          requestId: id,
        }
      },
      async onDone() {
        const billing = await resolveBillingReceipt(lastResult.value?.billing, balanceBefore)
        if (lastResult.value?.prompt === text) {
          lastResult.value = { ...lastResult.value, billing }
        }
      },
    })
  } catch (err) {
    if (!isAbortError(err)) {
      error.value = err instanceof Error ? err.message : '对话请求失败，本次不应扣费。'
      if (!chatMessages.value[assistantIndex].content) chatMessages.value.splice(assistantIndex, 1)
    }
  } finally {
    endSubmit()
  }
}

function startSubmit(mode: TabValue): void {
  abortController?.abort()
  abortController = new AbortController()
  activeMode.value = mode
  submitting.value = true
  error.value = ''
  requestId.value = ''
}

function endSubmit(): void {
  submitting.value = false
  activeMode.value = null
  abortController = null
}

function stopGeneration(): void {
  abortController?.abort()
  endSubmit()
}

async function resolveBillingReceipt(billing: PlaygroundBilling | undefined, balanceBefore: number): Promise<PlaygroundBilling | undefined> {
  try {
    const latestUser = await authStore.refreshUser()
    const balanceAfter = latestUser.balance ?? balance.value
    const amount = billing?.amount ?? Math.max(0, Number((balanceBefore - balanceAfter).toFixed(6)))
    return {
      ...billing,
      amount: amount > 0 ? amount : billing?.amount,
      balance_after: billing?.balance_after ?? balanceAfter,
    }
  } catch {
    return billing
  }
}

function retryLast(): void {
  if (activeTab.value === 'image') generateImage()
  else sendChat()
}

function downloadImage(): void {
  if (!lastResult.value?.imageUrl) return
  const link = document.createElement('a')
  link.href = lastResult.value.imageUrl
  link.download = 'relayq-playground-image.png'
  link.target = '_blank'
  link.rel = 'noreferrer'
  link.click()
}

function restoreResult(item: RecentResult): void {
  imagePrompt.value = item.prompt
  if (item.imageUrl) {
    lastResult.value = item
    requestId.value = item.requestId || ''
  }
}

function saveRecent(item: RecentResult): void {
  recentResults.value = [item, ...recentResults.value.filter((result) => result.id !== item.id)].slice(0, 6)
  localStorage.setItem(RECENT_KEY, JSON.stringify(recentResults.value.map(({ imageUrl: _imageUrl, ...rest }) => rest)))
}

function loadRecentResults(): RecentResult[] {
  try {
    return JSON.parse(localStorage.getItem(RECENT_KEY) || '[]')
  } catch {
    return []
  }
}

function clearRecent(): void {
  recentResults.value = []
  localStorage.removeItem(RECENT_KEY)
}

function formatMoney(value: number): string {
  return `¥${Number(value || 0).toFixed(2)}`
}

function exampleLabel(example: string): string {
  return example.split(/[：:，,。]/)[0].slice(0, 14) || '示例'
}

function createId(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) return crypto.randomUUID()
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function isAbortError(err: unknown): boolean {
  return err instanceof DOMException && err.name === 'AbortError'
}

onBeforeUnmount(() => {
  abortController?.abort()
})
</script>
