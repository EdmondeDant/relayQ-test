import { apiClient } from './client'
import type {
  BasePaginationResponse,
  CreateIdeaMessageRequest,
  IdeaMessage,
  IdeaMessageListParams
} from '@/types'

export async function list(
  page: number = 1,
  pageSize: number = 10,
  params?: IdeaMessageListParams,
  options?: {
    signal?: AbortSignal
  }
): Promise<BasePaginationResponse<IdeaMessage>> {
  const { data } = await apiClient.get<BasePaginationResponse<IdeaMessage>>('/idea-messages', {
    params: {
      page,
      page_size: pageSize,
      ...(params?.mine_only ? { mine_only: 1 } : {})
    },
    signal: options?.signal
  })
  return data
}

export async function create(request: CreateIdeaMessageRequest): Promise<IdeaMessage> {
  const { data } = await apiClient.post<IdeaMessage>('/idea-messages', request)
  return data
}

export async function deleteIdeaMessage(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/idea-messages/${id}`)
  return data
}

const ideaMessagesAPI = {
  list,
  create,
  delete: deleteIdeaMessage
}

export default ideaMessagesAPI
