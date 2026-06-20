import { describe, expect, it } from 'vitest'
import {
  buildCcSwitchImportLink,
  buildServiceUrls,
  buildToolConfigExport,
} from '@/utils/toolConfigExport'

const baseContext = {
  providerName: 'RelayQ',
  homepage: 'https://relayq.test',
  routeBaseUrl: 'https://api.relayq.test/v1',
  apiKey: 'sk-test-123456',
  modelName: 'claude-sonnet-4-5',
}

function getParams(link: string) {
  const query = link.split('?')[1] || ''
  return new URLSearchParams(query)
}

describe('toolConfigExport utils', () => {
  it('normalizes route roots and derives service urls', () => {
    expect(buildServiceUrls('https://api.relayq.test/v1')).toEqual({
      root: 'https://api.relayq.test',
      openai: 'https://api.relayq.test/v1',
      anthropic: 'https://api.relayq.test',
      gemini: 'https://api.relayq.test/v1beta',
    })
  })

  it('builds a claude ccswitch import link with encoded provider config', () => {
    const params = getParams(buildCcSwitchImportLink('claude', baseContext))

    expect(params.get('resource')).toBe('provider')
    expect(params.get('app')).toBe('claude')
    expect(params.get('endpoint')).toBe('https://api.relayq.test')
    expect(params.get('model')).toBe(baseContext.modelName)
    expect(params.get('configFormat')).toBe('json')

    const config = JSON.parse(atob(params.get('config') || ''))
    expect(config.env.ANTHROPIC_AUTH_TOKEN).toBe(baseContext.apiKey)
    expect(config.env.ANTHROPIC_BASE_URL).toBe('https://api.relayq.test')
    expect(config.env.ANTHROPIC_MODEL).toBe(baseContext.modelName)
  })

  it('builds opencode export content with the resolved provider and base url', () => {
    const content = buildToolConfigExport('opencode', baseContext)

    expect(content).toContain('"$schema": "https://opencode.ai/config.json"')
    expect(content).toContain('"baseURL": "https://api.relayq.test"')
    expect(content).toContain(baseContext.apiKey)
    expect(content).toContain(`"model": "anthropic/${baseContext.modelName}"`)
  })
})
