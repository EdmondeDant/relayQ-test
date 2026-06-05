import type { ApiKey } from '@/types'

export interface GatewayModel {
  id: string
  object?: string
  created?: number
  owned_by?: string
}

interface GatewayModelsResponse {
  object: string
  data: GatewayModel[]
}

export interface ChatMessage {
  role: 'system' | 'user' | 'assistant'
  content: string
}

export interface StreamChatOptions {
  apiKey: string
  model: string
  messages: ChatMessage[]
  signal?: AbortSignal
  onDelta: (delta: string) => void
  onDone?: () => void
}

interface ImageGenerationResponse {
  data?: Array<{
    url?: string
    b64_json?: string
    revised_prompt?: string
  }>
}

export interface GeneratedImage {
  url: string
  revisedPrompt?: string
}

const GATEWAY_BASE_URL = ''

function authHeaders(apiKey: string): HeadersInit {
  return {
    Authorization: `Bearer ${apiKey}`,
    'Content-Type': 'application/json',
  }
}

async function readErrorMessage(response: Response): Promise<string> {
  try {
    const payload = await response.json()
    return payload?.message || payload?.error?.message || `请求失败：${response.status} ${response.statusText}`
  } catch {
    return `请求失败：${response.status} ${response.statusText}`
  }
}

async function ensureOk(response: Response): Promise<void> {
  if (!response.ok) {
    throw new Error(await readErrorMessage(response))
  }
}

export async function listGatewayModels(apiKey: ApiKey, signal?: AbortSignal): Promise<string[]> {
  const response = await fetch(`${GATEWAY_BASE_URL}/v1/models`, {
    method: 'GET',
    headers: authHeaders(apiKey.key),
    signal,
  })
  await ensureOk(response)
  const data = (await response.json()) as GatewayModelsResponse
  return (data.data || []).map((model) => model.id).filter(Boolean).sort()
}

function parseDelta(data: string): string {
  try {
    const payload = JSON.parse(data)
    const choice = payload?.choices?.[0]
    return choice?.delta?.content || choice?.message?.content || ''
  } catch {
    return ''
  }
}

export async function streamGatewayChat(options: StreamChatOptions): Promise<void> {
  const response = await fetch(`${GATEWAY_BASE_URL}/v1/chat/completions`, {
    method: 'POST',
    headers: authHeaders(options.apiKey),
    body: JSON.stringify({
      model: options.model,
      messages: options.messages,
      stream: true,
    }),
    signal: options.signal,
  })
  await ensureOk(response)

  if (!response.body) {
    throw new Error('当前浏览器不支持流式响应')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder('utf-8')
  let buffer = ''

  while (true) {
    const { value, done } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() || ''

    for (const line of lines) {
      const trimmed = line.trim()
      if (!trimmed.startsWith('data:')) continue

      const data = trimmed.slice(5).trim()
      if (!data) continue
      if (data === '[DONE]') {
        options.onDone?.()
        return
      }

      const delta = parseDelta(data)
      if (delta) {
        options.onDelta(delta)
      }
    }
  }

  options.onDone?.()
}

export async function generateGatewayImage(options: {
  apiKey: string
  model: string
  prompt: string
  signal?: AbortSignal
}): Promise<GeneratedImage[]> {
  const response = await fetch(`${GATEWAY_BASE_URL}/v1/images/generations`, {
    method: 'POST',
    headers: authHeaders(options.apiKey),
    body: JSON.stringify({
      model: options.model,
      prompt: options.prompt,
      n: 1,
      response_format: 'b64_json',
    }),
    signal: options.signal,
  })
  await ensureOk(response)
  const payload = (await response.json()) as ImageGenerationResponse
  return (payload.data || [])
    .map((item) => ({
      url: item.url || (item.b64_json ? `data:image/png;base64,${item.b64_json}` : ''),
      revisedPrompt: item.revised_prompt,
    }))
    .filter((item) => item.url)
}

export const modelTestAPI = {
  listGatewayModels,
  streamGatewayChat,
  generateGatewayImage,
}
