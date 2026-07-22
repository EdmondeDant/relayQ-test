import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'

const { syncUpstreamModelsPreviewMock, showErrorMock, showInfoMock, showSuccessMock } = vi.hoisted(() => ({
  syncUpstreamModelsPreviewMock: vi.fn(),
  showErrorMock: vi.fn(),
  showInfoMock: vi.fn(),
  showSuccessMock: vi.fn()
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showInfo: showInfoMock,
    showSuccess: showSuccessMock
  })
}))

vi.mock('@/api/admin/accounts', () => ({
  accountsAPI: {
    syncUpstreamModelsPreview: syncUpstreamModelsPreviewMock,
    syncUpstreamModels: vi.fn()
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (key === 'admin.accounts.syncUpstreamModelsError') {
          return `同步上游模型失败：${String(params?.message ?? '')}`
        }
        return key
      }
    })
  }
})

import ModelWhitelistSelector from '../ModelWhitelistSelector.vue'

describe('ModelWhitelistSelector', () => {
  beforeEach(() => {
    syncUpstreamModelsPreviewMock.mockReset()
    showErrorMock.mockReset()
    showInfoMock.mockReset()
    showSuccessMock.mockReset()
  })

  it('shows backend sync-upstream error message for preview sync failures', async () => {
    syncUpstreamModelsPreviewMock.mockRejectedValue({
      message: 'Upstream model list request failed with HTTP 403: gateway disabled /models for this key'
    })

    const wrapper = mount(ModelWhitelistSelector, {
      props: {
        modelValue: [],
        syncCredentials: {
          platform: 'xai',
          type: 'apikey',
          base_url: 'https://api.muskapi.cc/v1',
          api_key: 'xai-key'
        }
      },
      global: {
        stubs: {
          ModelIcon: true,
          Icon: true
        }
      }
    })

    const syncButtons = wrapper.findAll('button').filter(button => button.text().includes('admin.accounts.syncUpstreamModels'))
    await syncButtons[0]?.trigger('click')

    expect(showErrorMock).toHaveBeenCalledWith(
      '同步上游模型失败：Upstream model list request failed with HTTP 403: gateway disabled /models for this key',
    )
  })
})
