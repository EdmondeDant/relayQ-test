import { apiClient } from './client'

export interface PlaygroundTask {
  id: number
  kind: string
  status: string
  model: string
  request_id?: string
  request_payload: Record<string, unknown>
  result_payload: Record<string, unknown>
  error_message?: string
  created_at: string
  updated_at: string
  expires_at: string
}

export interface PlaygroundAsset {
  id: number
  task_id?: number
  kind: string
  title: string
  content?: string
  url?: string
  storage_key?: string
  content_type?: string
  byte_size?: number
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
  expires_at: string
}

export interface PersistedMediaRef {
  assetId?: number
  storageKey?: string
  url: string
  contentType?: string
  byteSize?: number
}

export interface PlaygroundRecord {
  id: number
  kind: string
  status: string
  model: string
  request_id?: string
  request_payload: Record<string, unknown>
  result_payload: Record<string, unknown>
  error_message?: string
  created_at: string
  updated_at: string
  expires_at: string
  assets: PlaygroundAsset[]
  primary_asset?: PlaygroundAsset
}

export interface PaginatedResult<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  pages?: number
  total_pages?: number
}

/**
 * 拦截器会把 { code, data } 解成 response.data，但仍返回完整 AxiosResponse。
 * 业务层统一取 response.data，与其他 API 模块保持一致。
 */
async function unwrapData<T>(promise: Promise<{ data: T }>): Promise<T> {
  const { data } = await promise
  return data
}

function buildStoredAssetUrl(storageKey: string): string {
  return `/api/v1/playground/assets/content/${encodeURIComponent(String(storageKey || '').trim())}`
}

function toPersistedMediaRef(asset: PlaygroundAsset | null | undefined): PersistedMediaRef | null {
  if (!asset) return null
  const storageKey = String(asset.storage_key || '').trim()
  const url = storageKey ? buildStoredAssetUrl(storageKey) : String(asset.url || '').trim()
  if (!url) return null
  return {
    assetId: Number(asset.id || 0) || undefined,
    storageKey: storageKey || undefined,
    url,
    contentType: asset.content_type,
    byteSize: asset.byte_size,
  }
}

export const playgroundCloudAPI = {
  toPersistedMediaRef,
  async createTask(payload: {
    kind: string
    status?: string
    model?: string
    request_id?: string
    request_payload?: Record<string, unknown>
    result_payload?: Record<string, unknown>
    error_message?: string
  }): Promise<PlaygroundTask> {
    return unwrapData(apiClient.post('/playground/tasks', payload))
  },

  async listTasks(params?: { page?: number; page_size?: number; kind?: string }): Promise<PaginatedResult<PlaygroundTask>> {
    return unwrapData(apiClient.get('/playground/tasks', { params }))
  },

  async cancelTask(id: number): Promise<{ id: number; status: string }> {
    return unwrapData(apiClient.post(`/playground/tasks/${id}/cancel`))
  },

  async createAsset(payload: {
    task_id?: number
    kind: string
    title?: string
    content?: string
    url?: string
    content_type?: string
    metadata?: Record<string, unknown>
  }): Promise<PlaygroundAsset> {
    return unwrapData(apiClient.post('/playground/assets', payload, {
      timeout: 180000,
    }))
  },

  async listAssets(params?: { page?: number; page_size?: number; kind?: string }): Promise<PaginatedResult<PlaygroundAsset>> {
    return unwrapData(apiClient.get('/playground/assets', { params }))
  },

  async deleteAsset(id: number): Promise<void> {
    await apiClient.delete(`/playground/assets/${id}`)
  },

  async listRecords(params?: { page?: number; page_size?: number; kind?: string }): Promise<PaginatedResult<PlaygroundRecord>> {
    // 历史记录可能含大图，列表超时要更宽松；后端也会剥离大 content。
    return unwrapData(apiClient.get('/playground/records', {
      params,
      timeout: 60000,
    }))
  },

  async deleteRecord(id: number): Promise<void> {
    await apiClient.delete(`/playground/records/${id}`)
  },
}
