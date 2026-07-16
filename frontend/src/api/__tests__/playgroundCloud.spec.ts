import { beforeEach, describe, expect, it, vi } from 'vitest'
import { apiClient } from '@/api/client'
import { playgroundCloudAPI } from '@/api/playgroundCloud'

vi.mock('@/api/client', () => ({
  apiClient: {
    post: vi.fn(),
    get: vi.fn(),
    delete: vi.fn(),
  },
}))

describe('playground cloud api', () => {
  beforeEach(() => vi.clearAllMocks())

  it('creates tasks and assets with the authenticated api client', async () => {
    vi.mocked(apiClient.post)
      .mockResolvedValueOnce({ data: { id: 10 } })
      .mockResolvedValueOnce({ data: { id: 20 } })

    const task = await playgroundCloudAPI.createTask({ kind: 'copywriting', status: 'succeeded', result_payload: { content: 'result' } })
    const asset = await playgroundCloudAPI.createAsset({ task_id: 10, kind: 'text', content: 'result' })

    expect(task).toEqual({ id: 10 })
    expect(asset).toEqual({ id: 20 })
    expect(apiClient.post).toHaveBeenNthCalledWith(1, '/playground/tasks', expect.objectContaining({ kind: 'copywriting' }))
    expect(apiClient.post).toHaveBeenNthCalledWith(2, '/playground/assets', expect.objectContaining({ task_id: 10, kind: 'text' }), expect.objectContaining({ timeout: 180000 }))
  })

  it('lists and deletes cloud resources', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { items: [], total: 0 } })
    vi.mocked(apiClient.delete).mockResolvedValue(undefined)

    const tasks = await playgroundCloudAPI.listTasks({ page_size: 50 })
    const assets = await playgroundCloudAPI.listAssets({ page_size: 50 })
    await playgroundCloudAPI.deleteAsset(8)

    expect(tasks).toEqual({ items: [], total: 0 })
    expect(assets).toEqual({ items: [], total: 0 })
    expect(apiClient.get).toHaveBeenNthCalledWith(1, '/playground/tasks', { params: { page_size: 50 } })
    expect(apiClient.get).toHaveBeenNthCalledWith(2, '/playground/assets', { params: { page_size: 50 } })
    expect(apiClient.delete).toHaveBeenCalledWith('/playground/assets/8')
  })

  it('prefers storage_key content route for stable asset playback', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      data: {
        items: [{
          id: 1,
          kind: 'audio-generate',
          assets: [{ id: 2, kind: 'audio', storage_key: 'tts/result.wav', url: '/api/v1/playground/assets/content/tts/result.wav' }],
          primary_asset: { id: 2, kind: 'audio', storage_key: 'tts/result.wav', url: '/api/v1/playground/assets/content/tts/result.wav' },
        }],
        total: 1,
        page: 1,
        page_size: 10,
      },
    })

    const result = await playgroundCloudAPI.listRecords({ page_size: 10 })
    expect(result.items[0].primary_asset?.storage_key).toBe('tts/result.wav')
  })
})
