import { apiClient } from '../client'

export interface RetailGrokKey {
  id: number
  key: string
  name: string
  status: string
  group_id: number
  expires_at: string | null
  token_limit_total: number
  token_used_total: number
  image_limit_total: number
  image_used_total: number
  video_limit_total: number
  video_used_total: number
  created_by_admin_id: number
  created_at: string
  updated_at: string
}

export interface RetailGrokUsageLog {
  id: number
  request_id: string
  inbound_endpoint: string
  model: string
  input_tokens: number
  output_tokens: number
  total_tokens: number
  image_count: number
  video_count: number
  status: string
  error_message: string
  created_at: string
}

export interface RetailGrokUsageSummary {
  key: RetailGrokKey
  recent_logs: RetailGrokUsageLog[]
}

export interface BatchGenerateRequest {
  group_id: number
  count: number
  name_prefix: string
  expires_in_days?: number | null
  token_limit_total: number
  image_limit_total: number
  video_limit_total: number
}

export async function batchGenerateRetailGrokKeys(payload: BatchGenerateRequest): Promise<{ keys: RetailGrokKey[] }> {
  const { data } = await apiClient.post<{ keys: RetailGrokKey[] }>('/admin/retail-grok/batch-generate', payload)
  return data
}

export async function listRetailGrokKeys(limit = 100): Promise<RetailGrokKey[]> {
  const { data } = await apiClient.get<{ items: RetailGrokKey[] }>('/admin/retail-grok/keys', {
    params: { limit }
  })
  return data.items ?? []
}

export async function getRetailGrokKeyUsage(id: number): Promise<RetailGrokUsageSummary> {
  const { data } = await apiClient.get<RetailGrokUsageSummary>(`/admin/retail-grok/keys/${id}/usage`)
  return data
}

export async function deleteRetailGrokKey(id: number): Promise<void> {
  await apiClient.delete(`/admin/retail-grok/keys/${id}`)
}
