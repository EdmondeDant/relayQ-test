import { apiClient } from '../client'

export type MediaModality = 'image' | 'video'

export interface MediaProductPrice {
  id?: number
  operation: string
  spec_key: string
  unit_price_usd: number | string
  currency: string
  version: string
  enabled: boolean
}

export interface MediaOffer {
  id?: number
  provider: string
  source_group_id: number
  upstream_model: string
  enabled: boolean
  priority: number
  operations: string[]
  capabilities: Record<string, unknown>
  cost_rules: Record<string, unknown>
  cost_source: string
  cost_version: string
  verified_at: string
  expires_at: string
}

export interface MediaProduct {
  id?: number
  public_model: string
  modality: MediaModality
  enabled: boolean
  description: string | null
  group_ids: number[]
  prices: MediaProductPrice[]
  offers: MediaOffer[]
  created_at?: string
  updated_at?: string
}

export async function list(): Promise<MediaProduct[]> {
  const { data } = await apiClient.get<{ items: MediaProduct[] }>('/admin/media-products', { params: { page_size: 100 } })
  return data.items
}

export async function create(product: MediaProduct): Promise<MediaProduct> {
  const { data } = await apiClient.post<MediaProduct>('/admin/media-products', product)
  return data
}

export async function update(id: number, product: MediaProduct): Promise<MediaProduct> {
  const { data } = await apiClient.put<MediaProduct>(`/admin/media-products/${id}`, product)
  return data
}

export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/media-products/${id}`)
}

export default { list, create, update, remove }
