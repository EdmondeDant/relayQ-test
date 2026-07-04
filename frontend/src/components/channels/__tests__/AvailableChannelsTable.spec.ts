import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AvailableChannelsTable from '../AvailableChannelsTable.vue'
import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_TOKEN,
} from '@/constants/channel'

const messages: Record<string, string> = {
  'availableChannels.columns.modelSummary': 'Model Summary',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

describe('AvailableChannelsTable', () => {
  it('renders manual model summary text from the API', () => {
    const wrapper = mount(AvailableChannelsTable, {
      props: {
        columns: {
          name: 'Channel',
          description: 'Description',
          supportedModels: 'Supported Models',
        },
        rows: [
          {
            name: 'Main Channel',
            description: 'Primary route',
            platforms: [
              {
                platform: 'openai',
                groups: [
                  { id: 1, name: 'VIP', platform: 'openai', subscription_type: 'standard', rate_multiplier: 1, is_exclusive: true },
                  { id: 2, name: 'Public', platform: 'openai', subscription_type: 'standard', rate_multiplier: 1, is_exclusive: false },
                ],
                supported_models: [
                  {
                    name: 'gpt-4.1',
                    platform: 'openai',
                    summary: '主力对话模型，适合代码和长文本。',
                    pricing: {
                      billing_mode: BILLING_MODE_TOKEN,
                      input_price: 1,
                      output_price: 2,
                      cache_write_price: 0.5,
                      cache_read_price: 0.2,
                      image_output_price: null,
                      per_request_price: null,
                      intervals: [],
                    },
                    image_pricing: null,
                  },
                ],
              },
            ],
          },
        ],
        loading: false,
        pricingKeyPrefix: 'availableChannels.pricing',
        noPricingLabel: 'Pricing not configured',
        noModelsLabel: 'No models configured',
        emptyLabel: 'No channels',
        userGroupRates: {},
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    expect(wrapper.text()).toContain('Model Summary')
    expect(wrapper.text()).toContain('主力对话模型，适合代码和长文本。')
  })

  it('renders a dash when manual model summary is empty', () => {
    const wrapper = mount(AvailableChannelsTable, {
      props: {
        columns: {
          name: 'Channel',
          description: 'Description',
          supportedModels: 'Supported Models',
        },
        rows: [
          {
            name: 'Image Channel',
            description: 'Image route',
            platforms: [
              {
                platform: 'xai',
                groups: [
                  { id: 9, name: 'Image Pro', platform: 'xai', subscription_type: 'subscription', rate_multiplier: 1, is_exclusive: false },
                ],
                supported_models: [
                  {
                    name: 'grok-imagine',
                    platform: 'xai',
                    summary: '',
                    pricing: {
                      billing_mode: BILLING_MODE_IMAGE,
                      input_price: null,
                      output_price: null,
                      cache_write_price: null,
                      cache_read_price: null,
                      image_output_price: null,
                      per_request_price: 0.8,
                      intervals: [],
                    },
                    image_pricing: {
                      price_1k: 0.4,
                      price_2k: 0.8,
                      price_4k: 1.2,
                    },
                  },
                ],
              },
            ],
          },
        ],
        loading: false,
        pricingKeyPrefix: 'availableChannels.pricing',
        noPricingLabel: 'Pricing not configured',
        noModelsLabel: 'No models configured',
        emptyLabel: 'No channels',
        userGroupRates: {},
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    expect(wrapper.text()).toContain('—')
  })
})
