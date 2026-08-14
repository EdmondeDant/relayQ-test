import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import APIDocsView from '../APIDocsView.vue'

vi.mock('@/components/layout/AppLayout.vue', () => ({ default: { template: '<div><slot /></div>' } }))

describe('APIDocsView', () => {
  it('仅公开 OpenAI-compatible v1 媒体接口', () => {
    const text = mount(APIDocsView).text()
    expect(text).toContain('OPENAI-COMPATIBLE V1')
    expect(text).toContain('/v1/images/generations')
    expect(text).toContain('/v1/images/edits')
    expect(text).toContain('/v1/videos/generations')
    expect(text).toContain('/v1/videos/{task_id}/content')
    expect(text).toContain('Idempotency-Key: video-request-001')
    expect(text).not.toContain('Leonardo 原生模式')
    expect(text).not.toContain('parameters.resolution')
  })
})
