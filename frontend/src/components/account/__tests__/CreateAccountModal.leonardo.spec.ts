import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'

const { createAccountMock } = vi.hoisted(() => ({
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
    isSimpleMode: true
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
    expect(wrapper.text()).not.toContain('admin.accounts.modelRestriction')
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
    expect(Object.keys(createAccountMock.mock.calls[0]?.[0]?.credentials)).toEqual(['base_url', 'api_key'])
  })
})
