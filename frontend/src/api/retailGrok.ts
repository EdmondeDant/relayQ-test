import axios from 'axios'

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

export async function fetchRetailGrokUsage(apiKey: string): Promise<RetailGrokUsageSummary> {
  const response = await axios.get('/api/v1/retail-grok/usage', {
    headers: {
      Authorization: `Bearer ${apiKey}`
    }
  })
  const payload = response.data as { code?: number; message?: string; data?: RetailGrokUsageSummary } | string
  if (!payload || typeof payload !== 'object' || !('data' in payload) || !payload.data) {
    throw new Error('零售用量接口返回异常，请检查前端代理或后端服务')
  }
  return payload.data
}
