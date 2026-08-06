import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'

const { authState, createAccountMock } = vi.hoisted(() => ({
  authState: { isSimpleMode: true },
  createAccountMock: vi.fn()
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    get isSimpleMode() {
      return authState.isSimpleMode
    }
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      create: createAccountMock,
      checkMixedChannelRisk: vi.fn()
    },
    settings: {
      getWebSearchEmulationConfig: vi.fn().mockResolvedValue({ enabled: false, providers: [] }),
      getSettings: vi.fn().mockResolvedValue({})
    },
    tlsFingerprintProfiles: {
      list: vi.fn().mockResolvedValue([])
    }
  }
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

import CreateAccountModal from '../CreateAccountModal.vue'

const BaseDialogStub = defineComponent({
  props: {
    show: {
      type: Boolean,
      default: false
    }
  },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

describe('CreateAccountModal Leonardo', () => {
  it('creates a Leonardo API Key account with the Production API default URL', async () => {
    authState.isSimpleMode = true
    createAccountMock.mockReset()
    createAccountMock.mockResolvedValue({})

    const wrapper = mount(CreateAccountModal, {
      props: {
        show: true,
        proxies: [],
        groups: []
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          ConfirmDialog: true,
          Select: true,
          Icon: true,
          ProxySelector: true,
          ProxyAdBanner: true,
          GroupSelector: true,
          ModelWhitelistSelector: true,
          QuotaLimitCard: true,
          OAuthAuthorizationFlow: true
        }
      }
    })

    const tempUnschedToggle = wrapper.findAll('button').find(button =>
      button.element.parentElement?.parentElement?.textContent?.includes('admin.accounts.tempUnschedulable.title')
    )
    expect(tempUnschedToggle).toBeDefined()
    await tempUnschedToggle?.trigger('click')
    await wrapper.get('[data-testid="platform-leonardo"]').trigger('click')

    const form = wrapper.get('form#create-account-form')
    const inputs = form.findAll('input')
    await inputs[0]?.setValue('Leonardo Production')

    const baseUrlInput = inputs.find(input => input.attributes('placeholder') === 'https://cloud.leonardo.ai/api/rest')
    const apiKeyInput = inputs.find(input => input.attributes('placeholder') === 'Leonardo API Key')

    expect(baseUrlInput?.element.value).toBe('https://cloud.leonardo.ai/api/rest')
    expect(apiKeyInput).toBeDefined()
    expect(wrapper.text()).toContain('Production API')
    expect(wrapper.text()).toContain('admin.accounts.modelRestriction')
    expect(wrapper.text()).not.toContain('admin.accounts.poolMode')
    expect(wrapper.text()).not.toContain('admin.accounts.customErrorCodes')
    expect(wrapper.text()).not.toContain('admin.accounts.tempUnschedulable.title')

    await apiKeyInput?.setValue('leo-secret')
    await form.trigger('submit.prevent')

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock.mock.calls[0]?.[0]).toMatchObject({
      platform: 'leonardo',
      type: 'apikey',
      credentials: {
        api_key: 'leo-secret',
        base_url: 'https://cloud.leonardo.ai/api/rest'
      }
    })
    expect(createAccountMock.mock.calls[0]?.[0]?.credentials.model_mapping).toBeUndefined()
    expect(wrapper.find('[data-tour="account-form-groups"]').exists()).toBe(false)
  })

  it('binds only Leonardo groups in standard mode', async () => {
    authState.isSimpleMode = false
    createAccountMock.mockReset()
    createAccountMock.mockResolvedValue({})

    const groups = [
      { id: 31, name: 'Leonardo Group', platform: 'leonardo', rate_multiplier: 1, account_count: 0 },
      { id: 32, name: 'OpenAI Group', platform: 'openai', rate_multiplier: 1, account_count: 0 }
    ]
    const wrapper = mount(CreateAccountModal, {
      props: { show: true, proxies: [], groups: groups as any },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          ConfirmDialog: true,
          Select: true,
          Icon: true,
          ProxySelector: true,
          ProxyAdBanner: true,
          ModelWhitelistSelector: true,
          QuotaLimitCard: true,
          OAuthAuthorizationFlow: true
        }
      }
    })

    await wrapper.get('[data-testid="platform-leonardo"]').trigger('click')
    const form = wrapper.get('form#create-account-form')
    await form.findAll('input')[0]?.setValue('Leonardo Group Account')
    const selector = wrapper.get('[data-tour="account-form-groups"]')
    expect(selector.text()).toContain('Leonardo Group')
    expect(selector.text()).not.toContain('OpenAI Group')
    await selector.get('input[type="checkbox"]').setValue(true)
    const apiKeyInput = form.findAll('input').find(input => input.attributes('placeholder') === 'Leonardo API Key')
    await apiKeyInput?.setValue('leo-secret')
    await form.trigger('submit.prevent')

    expect(createAccountMock.mock.calls[0]?.[0]).toMatchObject({
      platform: 'leonardo',
      type: 'apikey',
      group_ids: [31]
    })
  })
})
