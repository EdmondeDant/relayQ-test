import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import APIDocsView from '../APIDocsView.vue'

vi.mock('@/components/layout/AppLayout.vue', () => ({ default: { template: '<div><slot /></div>' } }))

function mountDocs() {
  Object.defineProperty(window, 'scrollTo', { value: vi.fn(), writable: true })
  return mount(APIDocsView)
}

describe('APIDocsView', () => {
  it('展示生产环境的统一媒体端点和正确域名', () => {
    const text = mountDocs().text()
    expect(text).toContain('OPENAI-COMPATIBLE MEDIA API')
    expect(text).toContain('https://www.relayq.top/v1')
    expect(text).toContain('/v1/images/generations')
    expect(text).toContain('/v1/images/edits')
    expect(text).toContain('/v1/videos/generations')
    expect(text).toContain('/v1/videos/{task_id}/content')
    expect(text).toContain('Idempotency-Key: video-20260821-001')
    expect(text).not.toContain('https://www.realyq.top')
    expect(text).not.toContain('/v1/images/{model}')
    expect(text).not.toContain('/v1/videos/{model}')
  })

  it('列出后台新增的 5 个图片模型和 6 个视频模型', () => {
    const text = mountDocs().text()
    for (const model of [
      'GPT Image-2',
      'Nano Banana',
      'Nano Banana 2',
      'Nano Banana Pro',
      'Seedream 4.5',
      'seedance-2.0',
      'seedance-2.0-fast',
      'seedance-2.0-mini',
      'kling-3.0',
      'kling-video-o-3',
      'wan-2.7'
    ]) {
      expect(text).toContain(model)
    }
  })

  it('切换图片模型后展示参数、JSON URL 和 multipart 示例', async () => {
    const wrapper = mountDocs()
    await wrapper.findAll('button').find(button => button.text().includes('Nano Banana 2'))!.trigger('click')

    const text = wrapper.text()
    expect(text).toContain('图片模型')
    expect(text).toContain('image_url')
    expect(text).toContain('image_urls')
    expect(text).toContain('multipart/form-data')
    expect(wrapper.html()).toContain('/v1/images/edits')
    expect(wrapper.html()).toContain('image=@input.png')
  })

  it('切换视频模型后展示能力限制和参考素材参数', async () => {
    const wrapper = mountDocs()
    await wrapper.findAll('button').find(button => button.text().includes('kling-video-o-3'))!.trigger('click')

    const text = wrapper.text()
    expect(text).toContain('3–15 秒')
    expect(text).toContain('720P / 1080P')
    expect(text).toContain('video_urls')
    expect(text).toContain('最多 1 个')
    expect(text).toContain('Idempotency-Key: kling-video-o-3-image-request-001')
  })
})
