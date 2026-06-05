<template>
  <AppLayout>
    <div class="flex h-[calc(100vh-7rem)] flex-col gap-4">
      <div class="card p-4">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-end">
          <div class="min-w-0 flex-1">
            <label class="input-label">API Key</label>
            <Select
              v-model="selectedKeyId"
              :options="apiKeyOptions"
              :disabled="loadingKeys || sending"
              placeholder="请选择 API Key"
              @change="onApiKeyChange"
            />
          </div>
          <div class="min-w-0 flex-1">
            <label class="input-label">模型</label>
            <Select
              v-model="selectedModel"
              :options="modelOptions"
              :disabled="!selectedApiKey || loadingModels || sending"
              :placeholder="loadingModels ? '正在加载模型...' : '请选择模型'"
              searchable
              @change="onModelChange"
            />
          </div>
          <button class="btn btn-secondary" :disabled="!selectedModel || sending" @click="clearCurrentHistory">
            清空记录
          </button>
        </div>
        <p class="mt-3 text-xs text-gray-500 dark:text-dark-400">
          聊天记录保存在当前浏览器本地，切换 API Key 或模型会自动切换对应历史。
        </p>
      </div>

      <div class="card flex min-h-0 flex-1 flex-col overflow-hidden">
        <div ref="messagesContainer" class="flex-1 space-y-4 overflow-y-auto p-4">
          <div v-if="!messages.length" class="flex h-full items-center justify-center text-center text-gray-500 dark:text-dark-400">
            <div>
              <div class="mb-2 text-lg font-semibold text-gray-700 dark:text-dark-200">开始测试模型</div>
              <div class="text-sm">选择 API Key 和模型后，在下方输入消息即可开始流式对话。</div>
            </div>
          </div>

          <div
            v-for="(message, index) in messages"
            :key="index"
            :class="['flex', message.role === 'user' ? 'justify-end' : 'justify-start']"
          >
            <div
              :class="[
                'max-w-[85%] rounded-2xl px-4 py-3 text-sm leading-6 shadow-sm',
                message.role === 'user'
                  ? 'bg-primary-600 text-white'
                  : 'bg-gray-100 text-gray-900 dark:bg-dark-700 dark:text-dark-50'
              ]"
            >
              <div class="whitespace-pre-wrap break-words">{{ message.content }}</div>
              <div v-if="message.images?.length" class="mt-3 grid gap-3 sm:grid-cols-2">
                <a
                  v-for="(image, imageIndex) in message.images"
                  :key="imageIndex"
                  :href="image.url"
                  target="_blank"
                  rel="noreferrer"
                  class="block overflow-hidden rounded-xl border border-white/20 bg-black/5 dark:border-dark-600"
                >
                  <img :src="image.url" :alt="image.revisedPrompt || message.content || 'generated image'" class="h-auto w-full" />
                </a>
              </div>
            </div>
          </div>

          <div v-if="sending" class="flex justify-start">
            <div class="rounded-2xl bg-gray-100 px-4 py-3 text-sm text-gray-500 dark:bg-dark-700 dark:text-dark-300">
              正在生成...
            </div>
          </div>
        </div>

        <div v-if="error" class="border-t border-red-100 bg-red-50 px-4 py-2 text-sm text-red-600 dark:border-red-900/40 dark:bg-red-900/20 dark:text-red-300">
          {{ error }}
        </div>

        <div class="border-t border-gray-200 p-4 dark:border-dark-700">
          <div v-if="attachedFiles.length" class="mb-3 space-y-2">
            <div
              v-for="(file, index) in attachedFiles"
              :key="`${file.fileName}-${index}`"
              class="flex items-center justify-between rounded-lg bg-gray-50 px-3 py-2 text-xs text-gray-700 dark:bg-dark-800 dark:text-dark-200"
            >
              <span class="truncate">{{ file.fileName }}（{{ file.text.length }} 字）</span>
              <button class="text-red-500 hover:text-red-600" :disabled="sending" @click="removeAttachedFile(index)">移除</button>
            </div>
          </div>
          <div class="mb-3 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
            <input
              ref="fileInputRef"
              class="hidden"
              type="file"
              multiple
              accept=".txt,.md,.csv,.json,.log,.xml,.html,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,image/*"
              @change="handleFilesSelected"
            />
            <button class="btn btn-secondary btn-sm" :disabled="extractingFiles || sending" @click="fileInputRef?.click()">
              {{ extractingFiles ? '正在提取文件文字...' : '上传文件提取文字' }}
            </button>
            <span>支持图片 OCR、PDF、Word、Excel、PPT、TXT 等文件。</span>
          </div>
          <div class="flex gap-3">
            <textarea
              v-model="input"
              class="input min-h-[52px] flex-1 resize-none"
              rows="2"
              placeholder="输入消息，Enter 发送，Shift + Enter 换行"
              :disabled="!canChat && !sending"
              @keydown.enter.exact.prevent="sendMessage"
            />
            <button v-if="sending" class="btn btn-secondary self-end" @click="stopGeneration">停止</button>
            <button v-else class="btn btn-primary self-end" :disabled="!canChat || extractingFiles || (!input.trim() && !attachedFiles.length)" @click="sendMessage">
              发送
            </button>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select from '@/components/common/Select.vue'
import { keysAPI, modelTestAPI } from '@/api'
import { extractTextFromFiles, type ExtractedFileText } from '@/utils/fileTextExtractor'
import type { ApiKey } from '@/types'
import type { ChatMessage, GeneratedImage } from '@/api/modelTest'

type ModelTestMessage = ChatMessage & { images?: GeneratedImage[] }

const apiKeys = ref<ApiKey[]>([])
const models = ref<string[]>([])
const selectedKeyId = ref<number | null>(null)
const selectedModel = ref<string | null>(null)
const messages = ref<ModelTestMessage[]>([])
const input = ref('')
const error = ref('')
const loadingKeys = ref(false)
const loadingModels = ref(false)
const sending = ref(false)
const extractingFiles = ref(false)
const attachedFiles = ref<ExtractedFileText[]>([])
const fileInputRef = ref<HTMLInputElement | null>(null)
const messagesContainer = ref<HTMLElement | null>(null)
let modelsAbortController: AbortController | null = null
let chatAbortController: AbortController | null = null
let imageTimeoutId: ReturnType<typeof setTimeout> | null = null

function isAbortError(err: unknown): boolean {
  return err instanceof DOMException && err.name === 'AbortError'
}

const selectedApiKey = computed(() => apiKeys.value.find((key) => key.id === selectedKeyId.value) || null)
const canChat = computed(() => !!selectedApiKey.value && !!selectedModel.value && !sending.value)

const apiKeyOptions = computed(() =>
  apiKeys.value
    .filter((key) => key.status === 'active')
    .map((key) => ({
      value: key.id,
      label: key.group?.name ? `${key.name}（${key.group.name}）` : key.name,
    }))
)

const modelOptions = computed(() => models.value.map((model) => ({ value: model, label: model })))
const isImageModel = computed(() => selectedModel.value ? /(^|[-_/])(image|img)([-_/]|$)/i.test(selectedModel.value) : false)

function historyKey(): string | null {
  if (!selectedKeyId.value || !selectedModel.value) return null
  return `model-test:${selectedKeyId.value}:${selectedModel.value}`
}

function loadHistory(): void {
  const key = historyKey()
  if (!key) {
    messages.value = []
    return
  }
  try {
    const raw = localStorage.getItem(key)
    messages.value = raw ? JSON.parse(raw) : []
  } catch {
    messages.value = []
  }
}

function saveHistory(): void {
  const key = historyKey()
  if (!key) return
  localStorage.setItem(key, JSON.stringify(messages.value.slice(-100)))
}

function scrollToBottom(): void {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

async function loadApiKeys(): Promise<void> {
  loadingKeys.value = true
  error.value = ''
  try {
    const response = await keysAPI.list(1, 100, { status: 'active' })
    apiKeys.value = response.items
    if (!selectedKeyId.value && apiKeys.value.length > 0) {
      selectedKeyId.value = apiKeys.value[0].id
      await loadModels()
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载 API Key 失败'
  } finally {
    loadingKeys.value = false
  }
}

async function loadModels(): Promise<void> {
  modelsAbortController?.abort()
  models.value = []
  selectedModel.value = null
  loadHistory()

  if (!selectedApiKey.value) return

  modelsAbortController = new AbortController()
  loadingModels.value = true
  error.value = ''
  try {
    models.value = await modelTestAPI.listGatewayModels(selectedApiKey.value, modelsAbortController.signal)
    selectedModel.value = models.value[0] || null
    loadHistory()
  } catch (err) {
    if (!isAbortError(err)) {
      error.value = err instanceof Error ? err.message : '加载模型失败'
    }
  } finally {
    loadingModels.value = false
  }
}

function onApiKeyChange(): void {
  stopGeneration()
  loadModels()
}

function onModelChange(): void {
  stopGeneration()
  loadHistory()
}

async function handleFilesSelected(event: Event): Promise<void> {
  const inputElement = event.target as HTMLInputElement
  const files = Array.from(inputElement.files || [])
  inputElement.value = ''
  if (!files.length) return

  extractingFiles.value = true
  error.value = ''
  try {
    const extracted = await extractTextFromFiles(files)
    attachedFiles.value.push(...extracted)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '文件文字提取失败'
  } finally {
    extractingFiles.value = false
  }
}

function removeAttachedFile(index: number): void {
  attachedFiles.value.splice(index, 1)
}

async function sendMessage(): Promise<void> {
  if (!canChat.value || !selectedApiKey.value || !selectedModel.value || (!input.value.trim() && !attachedFiles.value.length)) return

  const fileText = attachedFiles.value
    .map((file) => `【文件：${file.fileName}】\n${file.text}`)
    .join('\n\n')
  const text = input.value.trim()
  const content = fileText ? `${text ? `${text}\n\n` : ''}以下是我上传文件中提取出的文字内容：\n\n${fileText}` : text
  input.value = ''
  attachedFiles.value = []
  error.value = ''
  messages.value.push({ role: 'user', content })
  const assistantIndex = messages.value.push({ role: 'assistant', content: '' }) - 1
  saveHistory()
  scrollToBottom()

  chatAbortController = new AbortController()
  sending.value = true
  try {
    if (isImageModel.value) {
      imageTimeoutId = setTimeout(() => {
        chatAbortController?.abort()
        error.value = '图片生成超时：上游图片接口长时间未返回，请检查上游账号、代理或模型可用性。'
      }, 120_000)
      const images = await modelTestAPI.generateGatewayImage({
        apiKey: selectedApiKey.value.key,
        model: selectedModel.value,
        prompt: content,
        signal: chatAbortController.signal,
      })
      if (imageTimeoutId) {
        clearTimeout(imageTimeoutId)
        imageTimeoutId = null
      }
      messages.value[assistantIndex].content = images.length ? '图片生成完成' : '未返回图片'
      messages.value[assistantIndex].images = images
      saveHistory()
      scrollToBottom()
      return
    }

    await modelTestAPI.streamGatewayChat({
      apiKey: selectedApiKey.value.key,
      model: selectedModel.value,
      messages: messages.value.slice(0, assistantIndex),
      signal: chatAbortController.signal,
      onDelta(delta) {
        messages.value[assistantIndex].content += delta
        saveHistory()
        scrollToBottom()
      },
      onDone() {
        saveHistory()
      },
    })
  } catch (err) {
    if (!isAbortError(err)) {
      error.value = err instanceof Error ? err.message : '模型请求失败'
      if (!messages.value[assistantIndex].content) {
        messages.value.splice(assistantIndex, 1)
      }
      saveHistory()
    }
  } finally {
    sending.value = false
    chatAbortController = null
  }
}

function stopGeneration(): void {
  chatAbortController?.abort()
  chatAbortController = null
  if (imageTimeoutId) {
    clearTimeout(imageTimeoutId)
    imageTimeoutId = null
  }
  sending.value = false
}

function clearCurrentHistory(): void {
  const key = historyKey()
  if (key) localStorage.removeItem(key)
  messages.value = []
}

watch(messages, saveHistory, { deep: true })

onMounted(() => {
  loadApiKeys()
})

onBeforeUnmount(() => {
  modelsAbortController?.abort()
  chatAbortController?.abort()
})
</script>
