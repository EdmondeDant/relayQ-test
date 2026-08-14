import { apiClient } from './client'

export interface CanvasBootstrap {
  base_url: string
  media_base_url: string
  api_key: string
  api_key_id: number
  api_key_name: string
  client_app: string
  user: { id: number; username: string; balance: number }
  models: Array<{ id: string; modality: string; platform: string; protocol: string; endpoints: string[] }>
  dashboard_url: string
  usage_url: string
}

export async function bootstrapCanvas(): Promise<CanvasBootstrap> {
  const response = await apiClient.post<CanvasBootstrap>('/canvas/bootstrap', {})
  return response.data
}
