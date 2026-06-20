import { apiClient } from '../client'
import type { AdminReplyIdeaMessageRequest, IdeaMessage } from '@/types'

export async function reply(id: number, request: AdminReplyIdeaMessageRequest): Promise<IdeaMessage> {
  const { data } = await apiClient.put<IdeaMessage>(`/admin/idea-messages/${id}/reply`, request)
  return data
}

const ideaMessagesAPI = {
  reply
}

export default ideaMessagesAPI
