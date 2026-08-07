import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import APIDocsView from '../APIDocsView.vue'

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: { template: '<div><slot /></div>' },
}))

describe('APIDocsView', () => {
  it('documents only the four supported Leonardo models and both protocols', () => {
    const wrapper = mount(APIDocsView)
    const text = wrapper.text()

    expect(text).toContain('OpenAI 兼容模式')
    expect(text).toContain('Leonardo 原生模式')
    expect(text).toContain('https://www.realyq.top/v1')
    expect(text).toContain('FLUX Schnell')
    expect(text).toContain('GPT Image 2')
    expect(text).toContain('Nano Banana 2')
    expect(text).toContain('Nano Banana 2 Lite')
    expect(wrapper.findAll('.model-section')).toHaveLength(4)
    expect(wrapper.findAll('.openai-manual')).toHaveLength(4)
    expect(wrapper.findAll('.raw-manual')).toHaveLength(4)
    expect(wrapper.find('#flux-schnell-openai').text()).toContain('FLUX Schnell · OpenAI 兼容模式')
    expect(wrapper.find('#flux-schnell-raw').text()).toContain('FLUX Schnell · Leonardo 原生模式')
    expect(wrapper.find('#gpt-image-2-openai').text()).toContain('GPT Image 2 · OpenAI 兼容模式')
    expect(wrapper.find('#gpt-image-2-raw').text()).toContain('GPT Image 2 · Leonardo 原生模式')
    expect(wrapper.find('#nano-banana-2-openai').text()).toContain('Nano Banana 2 · OpenAI 兼容模式')
    expect(wrapper.find('#nano-banana-2-raw').text()).toContain('Nano Banana 2 · Leonardo 原生模式')
    expect(wrapper.find('#nano-banana-2-lite-openai').text()).toContain('Nano Banana 2 Lite · OpenAI 兼容模式')
    expect(wrapper.find('#nano-banana-2-lite-raw').text()).toContain('Nano Banana 2 Lite · Leonardo 原生模式')
    expect(text).not.toContain('grok-imagine')
    expect(wrapper.find('#gpt-image-2-openai').text()).toContain('最多 6 张参考图')
    expect(wrapper.find('#gpt-image-2-openai').text()).not.toContain('只允许一个 image 文件')
    expect(text).not.toContain('MiMo')
  })

  it('shows protocol-specific reference capabilities', () => {
    const wrapper = mount(APIDocsView)
    const text = wrapper.text()

    expect(text).toContain('多参考图')
    expect(text).toContain('内容参考')
    expect(text).toContain('风格参考')
    expect(text).toContain('文生图请求参数')
    expect(text).toContain('原生请求参数')
    expect(text).toContain('参考图怎么传？')
    expect(text).toContain('完整文生图示例')
    expect(text).toContain('异步提交响应')
    expect(text).toContain('/v1/images/edits')
    expect(text).toContain('/v1/media/generations/gen_123')
  })

  it('explains Nano quality through resolution instead of a low-quality mode', () => {
    const wrapper = mount(APIDocsView)
    const nano = wrapper.find('#nano-banana-2').text()
    const lite = wrapper.find('#nano-banana-2-lite').text()

    expect(nano).toContain('没有 low、medium、high 原生质量档')
    expect(nano).toContain('官方支持 1K、2K、4K')
    expect(nano).toContain('quality=low 仅用于接口兼容和本地计价')
    expect(lite).toContain('固定原生质量档')
    expect(lite).toContain('不是 low 低质量模型')
  })
})
