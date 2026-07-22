import { describe, expect, it } from 'vitest'

function matchesModelPattern(pattern: string, modelId: string) {
  const normalizedPattern = pattern.trim()
  const normalizedModelId = modelId.trim()
  if (!normalizedPattern || !normalizedModelId) return false
  if (normalizedPattern === normalizedModelId) return true

  if (!normalizedPattern.includes('*')) return false
  if (!normalizedPattern.endsWith('*')) return false

  const prefix = normalizedPattern.slice(0, -1)
  return normalizedModelId.startsWith(prefix)
}

describe('matchesModelPattern', () => {
  it('matches exact ids', () => {
    expect(matchesModelPattern('grok-4.3', 'grok-4.3')).toBe(true)
  })

  it('matches trailing wildcard patterns', () => {
    expect(matchesModelPattern('grok-*', 'grok-4.3')).toBe(true)
  })

  it('rejects non-matching ids', () => {
    expect(matchesModelPattern('grok-*', 'gpt-4o')).toBe(false)
  })

  it('rejects mid-string wildcard patterns', () => {
    expect(matchesModelPattern('gr*ok', 'grok-4.3')).toBe(false)
  })
})
