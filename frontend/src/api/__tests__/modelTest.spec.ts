import { beforeEach, describe, expect, it, vi } from 'vitest'
import { modelTestAPI } from '@/api/modelTest'

describe('model test api', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
    vi.stubGlobal('crypto', { randomUUID: () => 'request-key' })
    vi.stubGlobal('URL', {
      createObjectURL: vi.fn(() => 'blob:https://example.com/test-audio'),
      revokeObjectURL: vi.fn(),
    })
  })

  it('sends image generations with codesonline gpt-image contract', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response(JSON.stringify({
      data: [{ url: 'https://example.com/result.png' }],
      request_id: 'gen-request',
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    const result = await modelTestAPI.generatePlaygroundImage({
      auth: { apiKey: 'rk-user-selected-key' },
      model: 'gpt-image-2',
      prompt: 'sunset cityscape',
      size: '16:9',
      quality: 'high',
      style: 'natural',
      background: 'opaque',
    })

    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect(JSON.parse(String(init?.body))).toEqual({
      model: 'gpt-image-2',
      prompt: 'sunset cityscape',
      n: 1,
      size: '16:9',
      quality: 'high',
      style: 'natural',
      background: 'opaque',
    })
    expect(result).toMatchObject({ requestId: 'gen-request', images: [{ url: 'https://example.com/result.png' }] })
  })

  it('sends grok-imagine-image with aspect_ratio and resolution', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response(JSON.stringify({
      data: [{ url: 'https://example.com/grok.png' }],
      request_id: 'grok-request',
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    await modelTestAPI.generatePlaygroundImage({
      auth: { apiKey: 'rk-user-selected-key' },
      model: 'grok-imagine-image',
      prompt: 'a red apple',
      size: '16:9',
      quality: 'high',
      style: 'natural',
      background: 'opaque',
    })

    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect(JSON.parse(String(init?.body))).toEqual({
      model: 'grok-imagine-image',
      prompt: 'a red apple',
      n: 1,
      aspect_ratio: '16:9',
      resolution: '2k',
    })
  })

  it('sends gpt-image edits with size/quality/style/background', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response(JSON.stringify({ data: [{ url: 'https://example.com/result.png' }], request_id: 'image-request' }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    const result = await modelTestAPI.editPlaygroundImage({
      auth: { apiKey: 'rk-user-selected-key' },
      model: 'gpt-image-2',
      prompt: 'change background',
      images: ['data:image/png;base64,abc', 'data:image/png;base64,reference'],
      mask: 'data:image/png;base64,mask',
      size: '16:9',
      quality: 'high',
      style: 'natural',
      background: 'opaque',
    })

    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect(init?.headers).toMatchObject({ Authorization: 'Bearer rk-user-selected-key', 'Idempotency-Key': 'request-key' })
    expect(JSON.parse(String(init?.body))).toEqual({
      model: 'gpt-image-2',
      prompt: 'change background',
      images: [{ image_url: 'data:image/png;base64,abc' }, { image_url: 'data:image/png;base64,reference' }],
      mask: { image_url: 'data:image/png;base64,mask' },
      size: '16:9',
      quality: 'high',
      style: 'natural',
      background: 'opaque',
    })
    expect(result.requestId).toBe('image-request')
  })

  it('sends grok-imagine-image edits with aspect_ratio and resolution', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response(JSON.stringify({ data: [{ url: 'https://example.com/edit.png' }], request_id: 'grok-edit' }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    await modelTestAPI.editPlaygroundImage({
      auth: { apiKey: 'rk-user-selected-key' },
      model: 'grok-imagine-image',
      prompt: 'remove watermark',
      image: 'data:image/png;base64,abc',
      size: '9:16',
      quality: 'high',
      style: 'natural',
      background: 'opaque',
    })

    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect(JSON.parse(String(init?.body))).toEqual({
      model: 'grok-imagine-image',
      prompt: 'remove watermark',
      images: [{ image_url: 'data:image/png;base64,abc' }],
      aspect_ratio: '9:16',
      resolution: '2k',
    })
  })

  it('submits and queries video tasks', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(new Response(JSON.stringify({ request_id: 'video-request', status: 'queued' }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ status: 'done', progress: 100, video: { url: 'https://example.com/video.mp4' } }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    const auth = { apiKey: 'rk-user-selected-key' }
    const created = await modelTestAPI.createPlaygroundVideo({ auth, model: 'grok-imagine-video', prompt: 'city at dusk', duration: 15, aspectRatio: '16:9', resolution: '1080p' })
    const completed = await modelTestAPI.getPlaygroundVideo(auth, created.requestId)

    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect(JSON.parse(String(init?.body))).toMatchObject({
      model: 'grok-imagine-video',
      prompt: 'city at dusk',
      duration: 15,
      aspect_ratio: '16:9',
      resolution: '1080p',
    })

    expect(created).toMatchObject({ requestId: 'video-request', status: 'queued', videoUrl: '' })
    expect(completed).toMatchObject({ requestId: 'video-request', status: 'done', progress: 100, videoUrl: 'https://example.com/video.mp4' })
  })

  it('submits audio generation with audio modality contract', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response(JSON.stringify({ audio: { url: 'https://example.com/audio.mp3' }, request_id: 'audio-request' }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    const result = await modelTestAPI.runPlaygroundAudio({
      auth: { apiKey: 'rk-user-selected-key' },
      model: 'mimo-v2.5-tts',
      mode: 'standard',
      messages: [{ role: 'user', content: [{ type: 'text', text: 'hello' }] }],
    })

    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect(JSON.parse(String(init?.body))).toMatchObject({
      model: 'mimo-v2.5-tts',
      audio: { voice: 'mimo_default', format: 'wav' },
      stream: false,
    })
    expect(result).toMatchObject({ requestId: 'audio-request', audioUrl: 'https://example.com/audio.mp3' })
  })

  it('extracts audio result from content parts when message.audio is absent', async () => {
    const objectUrlSpy = vi.mocked(URL.createObjectURL)
    vi.mocked(fetch).mockResolvedValue(new Response(JSON.stringify({
      choices: [{ message: { content: [{ type: 'output_audio', audio_url: 'https://example.com/tts.wav', format: 'wav' }, { type: 'text', text: 'done' }] } }],
      request_id: 'audio-content-request',
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    const result = await modelTestAPI.runPlaygroundAudio({
      auth: { apiKey: 'rk-user-selected-key' },
      model: 'mimo-v2.5-tts',
      mode: 'standard',
      messages: [{ role: 'user', content: [{ type: 'text', text: 'hello' }] }],
    })

    expect(result).toMatchObject({
      requestId: 'audio-content-request',
      audioUrl: 'https://example.com/tts.wav',
      text: 'done',
      transcript: 'done',
    })
    expect(objectUrlSpy).not.toHaveBeenCalled()
  })

  it('creates a blob url when upstream only returns audio base64', async () => {
    const objectUrlSpy = vi.mocked(URL.createObjectURL)
    objectUrlSpy.mockReturnValue('blob:https://example.com/generated-audio')
    vi.mocked(fetch).mockResolvedValue(new Response(JSON.stringify({
      choices: [{ message: { content: '', audio: { data: 'UklGRg==', id: 'audio-id' } } }],
      request_id: 'audio-base64-request',
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    const result = await modelTestAPI.runPlaygroundAudio({
      auth: { apiKey: 'rk-user-selected-key' },
      model: 'mimo-v2.5-tts',
      mode: 'standard',
      messages: [{ role: 'user', content: [{ type: 'text', text: 'hello' }] }],
    })

    expect(result).toMatchObject({
      requestId: 'audio-base64-request',
      audioUrl: 'blob:https://example.com/generated-audio',
      audioFormat: 'wav',
    })
    expect(result.dataUrl?.startsWith('data:audio/wav;base64,')).toBe(true)
    expect(objectUrlSpy).toHaveBeenCalledTimes(1)
  })

  it('submits audio transcription with MiMo input_audio + asr_options', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response(JSON.stringify({
      choices: [{ message: { content: '这是转写结果' } }],
      request_id: 'asr-request',
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    const dataUrl = 'data:audio/wav;base64,UklGRiQAAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQAAAAA='
    const result = await modelTestAPI.runPlaygroundAudio({
      auth: { apiKey: 'rk-user-selected-key' },
      model: 'mimo-v2.5-asr',
      mode: 'transcribe',
      asrOptions: { language: 'zh' },
      messages: [{
        role: 'user',
        content: [{ type: 'input_audio', input_audio: { data: dataUrl } }],
      }],
    })

    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect(JSON.parse(String(init?.body))).toMatchObject({
      model: 'mimo-v2.5-asr',
      stream: false,
      asr_options: { language: 'zh' },
      messages: [{
        role: 'user',
        content: [{ type: 'input_audio', input_audio: { data: dataUrl } }],
      }],
    })
    expect(result).toMatchObject({ requestId: 'asr-request', transcript: '这是转写结果' })
  })
})
