import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import AccountTestModal from '../AccountTestModal.vue'

const { getAvailableModelsMock } = vi.hoisted(() => ({
  getAvailableModelsMock: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getAvailableModels: getAvailableModelsMock
    }
  }
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: { show: { type: Boolean, default: false } },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const SelectStub = defineComponent({
  name: 'SelectStub',
  props: {
    modelValue: { type: [String, Number, Boolean, null], default: '' },
    options: { type: Array, default: () => [] },
    valueKey: { type: String, default: 'value' },
    labelKey: { type: String, default: 'label' }
  },
  emits: ['update:modelValue'],
  template: `
    <select
      v-bind="$attrs"
      :value="modelValue"
      @change="$emit('update:modelValue', $event.target.value)"
    >
      <option
        v-for="option in options"
        :key="option[valueKey]"
        :value="option[valueKey]"
      >
        {{ option[labelKey] }}
      </option>
    </select>
  `
})

const TextAreaStub = defineComponent({
  name: 'TextArea',
  props: {
    modelValue: { type: String, default: '' }
  },
  emits: ['update:modelValue'],
  template: `
    <textarea
      v-bind="$attrs"
      :value="modelValue"
      @input="$emit('update:modelValue', $event.target.value)"
    />
  `
})

const ConfirmDialogStub = defineComponent({
  name: 'ConfirmDialog',
  props: { show: { type: Boolean, default: false } },
  emits: ['confirm', 'cancel'],
  template: '<button v-if="show" data-test="confirm-paid" @click="$emit(\'confirm\')">confirm</button>'
})

function buildAccount() {
  return {
    id: 1,
    name: 'OpenAI OAuth',
    platform: 'openai',
    type: 'oauth',
    status: 'active',
    credentials: {},
    extra: {},
    concurrency: 1,
    priority: 1,
    proxy_id: null,
    auto_pause_on_expired: false
  } as any
}

function buildXAIAccount() {
  return {
    id: 2,
    name: 'Grok APIKey',
    platform: 'xai',
    type: 'apikey',
    status: 'active',
    credentials: {
      model_mapping: {
        'grok-4.5': 'grok-4.5'
      }
    },
    extra: {},
    concurrency: 1,
    priority: 1,
    proxy_id: null,
    auto_pause_on_expired: false
  } as any
}

function buildLeonardoAccount() {
  return {
    id: 3,
    name: 'Leonardo APIKey',
    platform: 'leonardo',
    type: 'apikey',
    status: 'active',
		credentials: { model_mapping: { 'kino-xl': 'kino-xl' } },
    extra: {},
    concurrency: 1,
    priority: 1,
    proxy_id: null,
    auto_pause_on_expired: false
  } as any
}

describe('AccountTestModal', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    getAvailableModelsMock.mockReset()
    getAvailableModelsMock.mockResolvedValue([
      { id: 'gpt-5.4', display_name: 'GPT-5.4' }
    ])
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: {
        getReader: () => ({
          read: vi.fn().mockResolvedValue({ done: true, value: undefined })
        })
      }
    } as any)
    localStorage.setItem('auth_token', 'test-token')
  })

  afterEach(() => {
    global.fetch = originalFetch
    localStorage.clear()
  })

  it('posts compact mode for OpenAI compact probe', async () => {
    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: buildAccount()
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Select: SelectStub,
          TextArea: TextAreaStub,
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4'
    ;(wrapper.vm as any).testMode = 'compact'
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, options] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(options.body)).toMatchObject({
      model_id: 'gpt-5.4',
      mode: 'compact'
    })
  })

  it('renders Chat Completions path status from test SSE', async () => {
    const encoder = new TextEncoder()
    const chunks = [
      encoder.encode('data: {"type":"status","text":"已通过 /v1/chat/completions 验证"}\n\n'),
      encoder.encode('data: {"type":"test_complete","success":true}\n\n')
    ]
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: {
        getReader: () => ({
          read: vi.fn().mockImplementation(() => Promise.resolve(
            chunks.length > 0
              ? { done: false, value: chunks.shift() }
              : { done: true, value: undefined }
          ))
        })
      }
    } as any)

    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: buildAccount()
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Select: SelectStub,
          TextArea: TextAreaStub,
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4'
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(wrapper.text()).toContain('已通过 /v1/chat/completions 验证')
  })

  it('requires confirmation before posting a paid Leonardo image test', async () => {
	getAvailableModelsMock.mockResolvedValue([{ id: 'kino-xl', type: 'model', display_name: 'Cinematic Kino', created_at: '', modality: 'image' }])
    const wrapper = mount(AccountTestModal, {
      props: { show: false, account: { ...buildLeonardoAccount(), credentials: {} } },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          ConfirmDialog: ConfirmDialogStub,
          Select: SelectStub,
          TextArea: TextAreaStub,
          Icon: true
        }
      }
    })

    await wrapper.setProps({ show: true })
    await flushPromises()
	;(wrapper.vm as any).testPrompt = 'cat'
	await flushPromises()
    expect(wrapper.text()).toContain('admin.accounts.paidImageTest')
    expect(global.fetch).not.toHaveBeenCalled()
    await wrapper.findAll('button').find(button => button.text().includes('admin.accounts.paidImageTest'))?.trigger('click')
    expect(global.fetch).not.toHaveBeenCalled()
    await wrapper.get('[data-test="confirm-paid"]').trigger('click')
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledOnce()
    const [, options] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(options.body)).toMatchObject({
		model_id: 'kino-xl',
		prompt: 'cat',
      paid: true,
      confirm_paid: true
    })
  })

  it('uses image generation for an OpenAI-compatible image model', async () => {
	getAvailableModelsMock.mockResolvedValue([{ id: 'flux-2-klein-9b-kv', type: 'model', display_name: 'flux-2-klein-9b-kv', created_at: '', modality: 'image' }])
	const wrapper = mount(AccountTestModal, {
		props: { show: false, account: buildAccount() },
		global: { stubs: { BaseDialog: BaseDialogStub, Select: SelectStub, TextArea: TextAreaStub, Icon: true } }
	})

	await wrapper.setProps({ show: true })
	await flushPromises()
	expect((wrapper.vm as any).selectedModelId).toBe('flux-2-klein-9b-kv')
	expect((wrapper.vm as any).testPrompt).toBe('admin.accounts.imagePromptDefault')
	await (wrapper.vm as any).startTest()
	await flushPromises()

	const [, options] = (global.fetch as any).mock.calls[0]
	expect(JSON.parse(options.body)).toMatchObject({ model_id: 'flux-2-klein-9b-kv', prompt: 'admin.accounts.imagePromptDefault' })
  })

  it('uses video generation for an OpenAI-compatible video model', async () => {
	getAvailableModelsMock.mockResolvedValue([{ id: 'minimax-h3', type: 'model', display_name: 'minimax-h3', created_at: '', modality: 'video' }])
	const wrapper = mount(AccountTestModal, {
		props: { show: false, account: buildAccount() },
		global: { stubs: { BaseDialog: BaseDialogStub, Select: SelectStub, TextArea: TextAreaStub, Icon: true } }
	})

	await wrapper.setProps({ show: true })
	await flushPromises()
	expect((wrapper.vm as any).selectedModelId).toBe('minimax-h3')
	expect((wrapper.vm as any).testPrompt).toBe('admin.accounts.videoPromptDefault')
	await (wrapper.vm as any).startTest()
	await flushPromises()
	const [, options] = (global.fetch as any).mock.calls[0]
	expect(JSON.parse(options.body)).toMatchObject({ model_id: 'minimax-h3', prompt: 'admin.accounts.videoPromptDefault' })
  })

  it('uses the paid video flow for a Leonardo video model', async () => {
    getAvailableModelsMock.mockResolvedValue([{ id: 'seedance-1.0-pro-fast', type: 'model', display_name: 'Seedance 1.0 Pro Fast', created_at: '', modality: 'video' }])
    const wrapper = mount(AccountTestModal, {
      props: { show: false, account: { ...buildLeonardoAccount(), credentials: {} } },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          ConfirmDialog: ConfirmDialogStub,
          Select: SelectStub,
          TextArea: TextAreaStub,
          Icon: true
        }
      }
    })

    await wrapper.setProps({ show: true })
    await flushPromises()
    expect((wrapper.vm as any).selectedModelId).toBe('seedance-1.0-pro-fast')
    expect((wrapper.vm as any).testPrompt).toBe('admin.accounts.videoPromptDefault')
    expect(wrapper.text()).toContain('admin.accounts.videoTestMode')
    expect(wrapper.text()).toContain('admin.accounts.paidVideoTest')
    await wrapper.findAll('button').find(button => button.text().includes('admin.accounts.paidVideoTest'))?.trigger('click')
    expect(global.fetch).not.toHaveBeenCalled()
    await wrapper.get('[data-test="confirm-paid"]').trigger('click')
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledOnce()
    const [, options] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(options.body)).toMatchObject({
      model_id: 'seedance-1.0-pro-fast',
      prompt: 'admin.accounts.videoPromptDefault',
      paid: true,
      confirm_paid: true
    })
  })

  it('locks paid retry when Leonardo submission status is unknown', async () => {
    const encoder = new TextEncoder()
    const chunks = [encoder.encode('data: {"type":"error","code":"LEONARDO_SUBMISSION_UNKNOWN","error":"Do not retry"}\n\n')]
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: { getReader: () => ({ read: vi.fn().mockImplementation(() => Promise.resolve(chunks.length ? { done: false, value: chunks.shift() } : { done: true, value: undefined })) }) }
    } as any)
    getAvailableModelsMock.mockResolvedValue([{ id: 'graphic-design', type: 'model', display_name: 'Graphic Design', created_at: '', modality: 'image' }])
    const wrapper = mount(AccountTestModal, {
      props: { show: false, account: { ...buildLeonardoAccount(), credentials: {} } },
      global: { stubs: { BaseDialog: BaseDialogStub, ConfirmDialog: ConfirmDialogStub, Select: SelectStub, TextArea: TextAreaStub, Icon: true } }
    })

    await wrapper.setProps({ show: true })
    await flushPromises()
    ;(wrapper.vm as any).testPrompt = 'cat'
    await (wrapper.vm as any).startTest(true)
    await flushPromises()

    const paidButton = wrapper.findAll('button').find(button => button.text().includes('admin.accounts.submissionUnknownLocked'))
    expect(paidButton?.attributes('disabled')).toBeDefined()
    expect(global.fetch).toHaveBeenCalledOnce()
  })

  it('for xai accounts only shows locally synced whitelist models', async () => {
    getAvailableModelsMock.mockResolvedValue([
      { id: 'grok-4.5', display_name: 'Grok 4.5' }
    ])

    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: buildXAIAccount()
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Select: SelectStub,
          TextArea: TextAreaStub,
          Icon: true
        }
      }
    })

    await flushPromises()
    expect((wrapper.vm as any).selectedModelId).toBe('')
  })
})
