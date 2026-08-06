import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const source = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../ChannelsView.vue'), 'utf8')
const pricingCardSource = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../../../components/admin/channel/PricingEntryCard.vue'), 'utf8')

describe('ChannelsView Leonardo', () => {
  it('offers the Leonardo platform', () => {
    expect(source).toContain("const platformOrder: GroupPlatform[] = ['anthropic', 'openai', 'gemini', 'antigravity', 'xai', 'leonardo']")
  })

  it('syncs one local-cost price entry per Leonardo model', () => {
		expect(source).toContain("if (platform === 'leonardo')")
		expect(source).toContain("adminAPI.channels.getModelDefaultPricing(model, platform)")
		expect(source).toContain("summary: '本地上游成本 × 7.1'")
		expect(source).toContain("billing_mode: 'image'")
	})

  it('loads verified Leonardo models into the group model picker', () => {
    expect(source).toContain("group.platform === 'leonardo'")
    expect(source).toContain('adminAPI.channels.syncPricingModels(group.platform)')
    expect(source).toContain('groupModelOptionsMap.value[key] = [...new Set(models || [])]')
    expect(source).toContain(':model-options="getModelOptionsForGroupIds(section.group_ids, section.platform)"')
    expect(source).toContain('const sourceIds = platform ? [...groupIds, 0] : groupIds')
  })

  it('locks Leonardo pricing to the local cost multiplier', () => {
    expect(pricingCardSource).toContain("props.platform === 'leonardo'")
    expect(pricingCardSource).toContain("label: '本地成本 × 7.1（自动）'")
    expect(pricingCardSource).toContain(':disabled="isLeonardoPricing"')
    expect(pricingCardSource).toContain(':readonly="isLeonardoPricing"')
    expect(pricingCardSource).toContain('实际扣费按请求的模型、尺寸、质量和数量计算')
  })
})
