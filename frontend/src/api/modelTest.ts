export interface ChatMessage {
  role: 'system' | 'user' | 'assistant'
  content: string | Array<{
    type: 'text' | 'image_url' | 'audio_url' | 'input_audio'
    text?: string
    image_url?: { url: string }
    audio_url?: { url: string }
    input_audio?: { data: string; format?: string }
  }>
}

export interface PlaygroundBilling {
  amount?: number
  currency?: string
  balance_after?: number
}

export interface PlaygroundImage {
  url: string
  revisedPrompt?: string
}

export interface PlaygroundGenerationResult {
  images: PlaygroundImage[]
  requestId?: string
  billing?: PlaygroundBilling
}

export interface PlaygroundVideoResult {
  requestId: string
  status?: string
  progress?: number
  videoUrl?: string
  billing?: PlaygroundBilling
}

function buildVideoContentUrl(requestId: string): string {
  const value = String(requestId || '').trim()
  return value ? `${GATEWAY_BASE_URL}/v1/videos/${encodeURIComponent(value)}/content` : ''
}

function extractReadyVideoUrl(payload: any): string {
  const candidates = [
    payload?.video?.url,
    payload?.output?.video?.url,
    payload?.output_url,
    payload?.url,
  ]
  for (const item of candidates) {
    const value = String(item || '').trim()
    if (value) return value
  }
  return ''
}

export interface PlaygroundAudioResult {
  requestId?: string
  billing?: PlaygroundBilling
  text?: string
  audioUrl?: string
  dataUrl?: string
  transcript?: string
  audioFormat?: string
  audioBlob?: Blob
}

function base64ToObjectUrl(base64: string, mimeType: string): string | undefined {
  try {
    const binary = atob(base64)
    const bytes = new Uint8Array(binary.length)
    for (let i = 0; i < binary.length; i += 1) bytes[i] = binary.charCodeAt(i)
    const blob = new Blob([bytes], { type: mimeType })
    return URL.createObjectURL(blob)
  } catch {
    return undefined
  }
}

function pickAudioUrl(candidate: any): string | undefined {
  const value = candidate?.url || candidate?.audio_url || candidate?.path || candidate?.src
  if (typeof value !== 'string' || !value.trim()) return undefined
  const trimmed = value.trim()
  if (/^https?:\/\//i.test(trimmed)) return trimmed
  if (trimmed.startsWith('/')) return `${GATEWAY_BASE_URL}${trimmed}`
  return trimmed
}

function pickAudioBase64(candidate: any): string | undefined {
  const value = candidate?.b64_json || candidate?.data || candidate?.audio_base64 || candidate?.base64
  return typeof value === 'string' && value.trim() ? value.trim() : undefined
}

function resolveAudioMimeType(format: string) {
  const normalized = String(format || '').trim().toLowerCase()
  if (normalized === 'mp3' || normalized === 'mpeg') return 'audio/mpeg'
  if (normalized === 'wav' || normalized === 'wave' || normalized === 'x-wav') return 'audio/wav'
  if (normalized === 'm4a' || normalized === 'mp4') return 'audio/mp4'
  if (normalized === 'ogg') return 'audio/ogg'
  return normalized ? `audio/${normalized}` : 'audio/wav'
}

async function fetchProtectedAudioAsBlob(url: string, auth: PlaygroundAuthContext, signal?: AbortSignal): Promise<Blob | undefined> {
  const value = String(url || '').trim()
  if (!value) return undefined
  const response = await fetch(value, {
    headers: {
      Authorization: `Bearer ${auth.apiKey}`,
    },
    credentials: 'include',
    signal,
    redirect: 'follow',
  })
  await ensureOk(response)
  const contentType = String(response.headers.get('content-type') || '').toLowerCase()
  if (contentType.includes('application/json')) return undefined
  const blob = await response.blob()
  return blob.size > 0 ? blob : undefined
}

function extractAudioFromContentParts(content: any): { url?: string; base64?: string; format?: string; textParts: string[] } {
  const parts = Array.isArray(content) ? content : []
  let url: string | undefined
  let base64: string | undefined
  let format: string | undefined
  const textParts: string[] = []

  for (const part of parts) {
    if (!part || typeof part !== 'object') continue
    const type = String(part.type || '').toLowerCase()
    if (!url) url = pickAudioUrl(part) || pickAudioUrl(part.audio) || pickAudioUrl(part.output_audio)
    if (!base64) base64 = pickAudioBase64(part) || pickAudioBase64(part.audio) || pickAudioBase64(part.output_audio)
    if (!format) {
      const rawFormat = part.format || part.audio?.format || part.output_audio?.format
      if (typeof rawFormat === 'string' && rawFormat.trim()) format = rawFormat.trim()
    }
    if (type === 'text' || typeof part.text === 'string') {
      const value = typeof part.text === 'string' ? part.text : ''
      if (value) textParts.push(value)
    }
  }

  return { url, base64, format, textParts }
}

export interface PlaygroundAuthContext {
  apiKey: string
}

export interface PlaygroundChatOptions {
  auth: PlaygroundAuthContext
  model: string
  messages: ChatMessage[]
  signal?: AbortSignal
  onDelta: (delta: string) => void
  onBilling?: (billing: PlaygroundBilling, requestId?: string) => void
  onDone?: () => void
}

interface ImageGenerationResponse {
  data?: Array<{
    url?: string
    b64_json?: string
    revised_prompt?: string
  }>
  request_id?: string
  billing?: PlaygroundBilling
}

const GATEWAY_BASE_URL = ''

function authHeaders(auth: PlaygroundAuthContext, extra: HeadersInit = {}): HeadersInit {
  return {
    Authorization: `Bearer ${auth.apiKey}`,
    'Content-Type': 'application/json',
    ...extra,
  }
}

async function readErrorMessage(response: Response): Promise<string> {
  try {
    const payload = await response.json()
    const code = payload?.error?.code || payload?.code
    const message = payload?.message || payload?.error?.message || `请求失败：${response.status} ${response.statusText}`
    const detail = payload?.error?.details || payload?.details || payload?.error?.param || payload?.param
    if (response.status === 402 || code === 'INSUFFICIENT_BALANCE') return '余额不足，充值后会回到本页继续生成。'
    if (code === 'CONTENT_POLICY_VIOLATION') return '内容未通过审核，本次不会扣费。'
    if (code === 'MODEL_NOT_ALLOWED') return '该模型暂未开放在线体验，请换一个精选模型。'
    return detail ? `${message}（${typeof detail === 'string' ? detail : JSON.stringify(detail)}）` : message
  } catch {
    return `请求失败：${response.status} ${response.statusText}`
  }
}

async function ensureOk(response: Response): Promise<void> {
  if (!response.ok) {
    throw new Error(await readErrorMessage(response))
  }
}

function parseSsePayload(data: string): { delta: string; billing?: PlaygroundBilling; requestId?: string } {
  try {
    const payload = JSON.parse(data)
    const choice = payload?.choices?.[0]
    return {
      delta: choice?.delta?.content || choice?.message?.content || '',
      billing: payload?.billing,
      requestId: payload?.request_id,
    }
  } catch {
    return { delta: '' }
  }
}

export async function runPlaygroundChat(options: {
  auth: PlaygroundAuthContext
  model: string
  messages: ChatMessage[]
  signal?: AbortSignal
}): Promise<{ content: string; billing?: PlaygroundBilling; requestId?: string }> {
  const response = await fetch(`${GATEWAY_BASE_URL}/v1/chat/completions`, {
    method: 'POST',
    headers: authHeaders(options.auth),
    body: JSON.stringify({ model: options.model, messages: options.messages, stream: false }),
    signal: options.signal,
  })
  await ensureOk(response)
  const payload = await response.json()
  return {
    content: payload?.choices?.[0]?.message?.content || '',
    billing: payload?.billing,
    requestId: payload?.request_id || response.headers.get('x-request-id') || undefined,
  }
}

export async function runPlaygroundAudio(options: {
  auth: PlaygroundAuthContext
  model: string
  messages: ChatMessage[]
  mode?: 'standard' | 'voicedesign' | 'voiceclone' | 'transcribe'
  audio?: { format: string; voice?: string; optimize_text_preview?: boolean }
  /** MiMo ASR 官方字段：auto / zh / en */
  asrOptions?: { language?: string }
  signal?: AbortSignal
}): Promise<PlaygroundAudioResult> {
  const body = options.mode === 'transcribe'
    ? {
        model: options.model,
        messages: options.messages,
        stream: false,
        ...(options.asrOptions ? { asr_options: options.asrOptions } : {}),
      }
    : {
        model: options.model,
        audio: options.audio || { format: 'wav', voice: 'mimo_default' },
        messages: options.messages,
        stream: false,
      }
  const response = await fetch(`${GATEWAY_BASE_URL}/v1/chat/completions`, {
    method: 'POST',
    headers: authHeaders(options.auth),
    body: JSON.stringify(body),
    signal: options.signal,
  })
  await ensureOk(response)
  const payload = await response.json()
  const message = payload?.choices?.[0]?.message
  const content = message?.content
  const extracted = extractAudioFromContentParts(content)
  const text = typeof content === 'string'
    ? content
    : Array.isArray(content)
      ? extracted.textParts.join('')
      : ''
  const audio = payload?.audio || payload?.data?.audio || payload?.output_audio || message?.audio
  const audioUrl = pickAudioUrl(audio) || pickAudioUrl(payload) || extracted.url
  const audioBase64 = pickAudioBase64(audio) || pickAudioBase64(payload) || extracted.base64
  const format = extracted.format || audio?.format || options.audio?.format || 'wav'
  const mimeType = resolveAudioMimeType(format)
  let audioBlob: Blob | undefined
  let objectUrl: string | undefined

  if (audioBase64) {
    objectUrl = base64ToObjectUrl(audioBase64, mimeType)
    try {
      const response = await fetch(`data:${mimeType};base64,${audioBase64}`)
      audioBlob = await response.blob()
    } catch {
      audioBlob = undefined
    }
  } else if (audioUrl) {
    try {
      audioBlob = await fetchProtectedAudioAsBlob(audioUrl, options.auth, options.signal)
      if (audioBlob) objectUrl = URL.createObjectURL(audioBlob)
    } catch {
      audioBlob = undefined
    }
  }

  return {
    requestId: payload?.request_id || response.headers.get('x-request-id') || undefined,
    billing: payload?.billing,
    text,
    transcript: payload?.transcript || payload?.text || text,
    audioUrl: objectUrl || audioUrl,
    dataUrl: audioBase64 ? `data:${mimeType};base64,${audioBase64}` : undefined,
    audioFormat: format,
    audioBlob,
  }
}

function isGrokImagineModel(model: string) {
  return /^grok-imagine-image/i.test(model)
}

function isGptImageModel(model: string) {
  return /^gpt-image-/i.test(model)
}

function isBlobObjectUrl(value?: string) {
  return /^blob:/i.test(String(value || '').trim())
}

async function blobUrlToDataUrl(blobUrl: string): Promise<string> {
  const response = await fetch(blobUrl)
  if (!response.ok) throw new Error('读取本地图片失败。')
  const blob = await response.blob()
  return await new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result || ''))
    reader.onerror = () => reject(new Error('转换图片失败。'))
    reader.readAsDataURL(blob)
  })
}

/** playground size / 比例 → xAI aspect_ratio */
function toGrokAspectRatio(size?: string): string {
  const s = (size || '1:1').trim().toLowerCase()
  if (['1:1', '16:9', '9:16', '3:2', '2:3', 'auto'].includes(s)) return s
  if (s === '1024x1024' || s === 'square') return '1:1'
  if (s === '1536x1024' || s === '1792x1024') return '16:9'
  if (s === '1024x1536' || s === '1024x1792') return '9:16'
  return '1:1'
}

export async function generatePlaygroundImage(options: {
  auth: PlaygroundAuthContext
  model: string
  prompt: string
  size?: string
  quality?: string
  style?: string
  background?: string
  signal?: AbortSignal
}): Promise<PlaygroundGenerationResult> {
  // 按模型族组装请求：
  // - grok-imagine-image: aspect_ratio + resolution（xAI 文档）
  // - gpt-image-*: size + quality/style/background（外联/OpenAI 兼容）
  // - 其他: 附带 response_format=b64_json
  const body: Record<string, unknown> = {
    model: options.model,
    prompt: options.prompt,
    n: 1,
  }

  if (isGrokImagineModel(options.model)) {
    body.aspect_ratio = toGrokAspectRatio(options.size)
    // xAI 当前只接受 1k/2k；quality 模型的高画质由模型本身体现，high 统一走 2k
    body.resolution = options.quality === 'high' ? '2k' : '1k'
  } else {
    if (options.size) body.size = options.size
    if (options.quality) body.quality = options.quality
    if (options.style) body.style = options.style
    if (options.background) body.background = options.background
    if (!isGptImageModel(options.model)) body.response_format = 'b64_json'
  }

  const response = await fetch(`${GATEWAY_BASE_URL}/v1/images/generations`, {
    method: 'POST',
    headers: authHeaders(options.auth, { 'Idempotency-Key': createIdempotencyKey() }),
    body: JSON.stringify(body),
    signal: options.signal,
  })
  await ensureOk(response)
  const payload = (await response.json()) as ImageGenerationResponse
  return {
    requestId: payload.request_id || response.headers.get('x-request-id') || response.headers.get('x-relayq-request-id') || undefined,
    billing: payload.billing,
    images: (payload.data || [])
      .map((item) => ({
        url: item.url || (item.b64_json ? `data:image/png;base64,${item.b64_json}` : ''),
        revisedPrompt: item.revised_prompt,
      }))
      .filter((item) => item.url),
  }
}

export async function editPlaygroundImage(options: {
  auth: PlaygroundAuthContext
  model: string
  prompt: string
  image?: string
  images?: string[]
  mask?: string
  size?: string
  quality?: string
  style?: string
  background?: string
  signal?: AbortSignal
}): Promise<PlaygroundGenerationResult> {
  // 与 generate 一致：按模型族组装 edits 请求
  // - grok-imagine-image: aspect_ratio + resolution
  // - gpt-image-*: size + quality/style/background
  // - 其他: 附带 response_format=b64_json
  const resolvedImages = await Promise.all(
    (options.images || (options.image ? [options.image] : [])).map(async (url) => isBlobObjectUrl(url) ? blobUrlToDataUrl(url) : url)
  )
  const resolvedMask = options.mask ? (isBlobObjectUrl(options.mask) ? await blobUrlToDataUrl(options.mask) : options.mask) : undefined

  const body: Record<string, unknown> = {
    model: options.model,
    prompt: options.prompt,
    images: resolvedImages.map((url) => ({ image_url: url })),
  }
  if (resolvedMask) body.mask = { image_url: resolvedMask }

  if (isGrokImagineModel(options.model)) {
    body.aspect_ratio = toGrokAspectRatio(options.size)
    // xAI 当前只接受 1k/2k；quality 模型 high 统一走 2k
    body.resolution = options.quality === 'high' ? '2k' : '1k'
  } else {
    body.size = options.size || '1:1'
    if (options.quality) body.quality = options.quality
    if (options.style) body.style = options.style
    if (options.background) body.background = options.background
    if (!isGptImageModel(options.model)) body.response_format = 'b64_json'
  }

  const response = await fetch(`${GATEWAY_BASE_URL}/v1/images/edits`, {
    method: 'POST',
    headers: authHeaders(options.auth, { 'Idempotency-Key': createIdempotencyKey() }),
    body: JSON.stringify(body),
    signal: options.signal,
  })
  await ensureOk(response)
  const payload = (await response.json()) as ImageGenerationResponse
  return {
    requestId: payload.request_id || response.headers.get('x-request-id') || response.headers.get('x-relayq-request-id') || undefined,
    billing: payload.billing,
    images: (payload.data || [])
      .map((item) => ({
        url: item.url || (item.b64_json ? `data:image/png;base64,${item.b64_json}` : ''),
        revisedPrompt: item.revised_prompt,
      }))
      .filter((item) => item.url),
  }
}

export async function createPlaygroundVideo(options: {
  auth: PlaygroundAuthContext
  model: string
  prompt: string
  image?: string
  duration: number
  aspectRatio: string
  resolution?: string
  signal?: AbortSignal
}): Promise<PlaygroundVideoResult> {
  const response = await fetch(`${GATEWAY_BASE_URL}/v1/videos/generations`, {
    method: 'POST',
    headers: authHeaders(options.auth, { 'Idempotency-Key': createIdempotencyKey() }),
    body: JSON.stringify({
      model: options.model,
      prompt: options.prompt,
      duration: options.duration,
      aspect_ratio: options.aspectRatio,
      resolution: options.resolution || '720p',
      ...(options.image ? { image: { url: options.image } } : {}),
    }),
    signal: options.signal,
  })
  await ensureOk(response)
  const payload = await response.json()
  const requestId = payload.request_id || payload.id || ''
  // 生成接口通常只返回任务 id；只有上游真返回可直链时才给 videoUrl。
  // 不要默认塞 /v1/videos/{id}/content：浏览器 <video> 和创作记录落库都带不上 API Key，会 401。
  return {
    requestId,
    status: payload.status,
    progress: payload.progress,
    videoUrl: extractReadyVideoUrl(payload),
    billing: payload.billing,
  }
}

export async function getPlaygroundVideo(auth: PlaygroundAuthContext, requestId: string, signal?: AbortSignal): Promise<PlaygroundVideoResult> {
  const response = await fetch(`${GATEWAY_BASE_URL}/v1/videos/${encodeURIComponent(requestId)}`, {
    headers: authHeaders(auth),
    signal,
  })
  await ensureOk(response)
  const payload = await response.json()
  const normalizedRequestId = requestId || payload.request_id || payload.id || ''
  const readyUrl = extractReadyVideoUrl(payload)
  const status = String(payload.status || '').toLowerCase()
  const isReady = Boolean(readyUrl) || ['completed', 'succeeded', 'ready', 'done'].includes(status)

  let videoUrl = readyUrl
  // 状态已完成但没有直链时：用鉴权拉取 /content（会 302 到 CDN 或直接返回视频流），再转成 blob URL 供 <video> 播放。
  if (!videoUrl && isReady && normalizedRequestId) {
    const contentResp = await fetch(buildVideoContentUrl(normalizedRequestId), {
      headers: {
        Authorization: `Bearer ${auth.apiKey}`,
      },
      signal,
      redirect: 'follow',
    })
    if (contentResp.ok) {
      const contentType = String(contentResp.headers.get('content-type') || '').toLowerCase()
      if (contentType.includes('application/json')) {
        // 兼容 content 接口仍返回 JSON 的情况
        try {
          const contentPayload = await contentResp.json()
          videoUrl = extractReadyVideoUrl(contentPayload)
        } catch {
          videoUrl = ''
        }
      } else {
        const blob = await contentResp.blob()
        if (blob.size > 0) {
          videoUrl = URL.createObjectURL(blob)
        }
      }
    }
  }

  return {
    requestId: normalizedRequestId,
    status: payload.status,
    progress: payload.progress,
    videoUrl,
    billing: payload.billing,
  }
}

export async function streamPlaygroundChat(options: PlaygroundChatOptions): Promise<void> {
  const response = await fetch(`${GATEWAY_BASE_URL}/v1/chat/completions`, {
    method: 'POST',
    headers: authHeaders(options.auth),
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

      const parsed = parseSsePayload(data)
      if (parsed.delta) options.onDelta(parsed.delta)
      if (parsed.billing) options.onBilling?.(parsed.billing, parsed.requestId)
    }
  }

  options.onDone?.()
}

function createIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export const modelTestAPI = {
  generatePlaygroundImage,
  editPlaygroundImage,
  streamPlaygroundChat,
  runPlaygroundChat,
  runPlaygroundAudio,
  createPlaygroundVideo,
  getPlaygroundVideo,
}
