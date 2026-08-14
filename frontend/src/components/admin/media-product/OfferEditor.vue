<template>
  <div class="space-y-3">
    <div v-for="(offer, index) in modelValue" :key="index" class="space-y-3 rounded-lg border border-gray-200 p-4 dark:border-dark-600">
      <div class="grid gap-3 md:grid-cols-4">
        <select v-model="offer.provider" class="input" @change="offer.source_group_id = 0"><option value="openai">OpenAI</option><option value="xai">xAI</option><option value="leonardo">Leonardo</option></select>
        <select v-model.number="offer.source_group_id" class="input"><option :value="0">{{ t('admin.mediaProducts.sourceGroup') }}</option><option v-for="group in sourceGroups(offer.provider)" :key="group.id" :value="group.id">{{ group.name }}</option></select>
        <input v-model="offer.upstream_model" class="input" :placeholder="t('admin.mediaProducts.upstreamModel')" />
        <input v-model.number="offer.priority" class="input" type="number" min="0" :placeholder="t('admin.mediaProducts.priority')" />
      </div>
      <div class="grid gap-3 md:grid-cols-3">
        <input v-model="operationText[index]" class="input" :placeholder="t('admin.mediaProducts.operations')" @input="syncOperations(index)" />
        <input v-model="offer.cost_source" class="input" :placeholder="t('admin.mediaProducts.costSource')" />
        <input v-model="offer.cost_version" class="input" :placeholder="t('admin.mediaProducts.costVersion')" />
      </div>
      <div class="grid gap-3 md:grid-cols-2">
        <textarea :value="jsonText(offer.capabilities)" class="input font-mono" rows="4" :placeholder="t('admin.mediaProducts.capabilities')" @change="setJSON(offer, 'capabilities', $event)" />
        <textarea :value="jsonText(offer.cost_rules)" class="input font-mono" rows="4" :placeholder="t('admin.mediaProducts.costRules')" @change="setJSON(offer, 'cost_rules', $event)" />
      </div>
      <div class="grid gap-3 md:grid-cols-3">
        <label class="input-label">{{ t('admin.mediaProducts.verifiedAt') }}<input :value="localDate(offer.verified_at)" class="input mt-1" type="datetime-local" @input="setDate(offer, 'verified_at', $event)" /></label>
        <label class="input-label">{{ t('admin.mediaProducts.expiresAt') }}<input :value="localDate(offer.expires_at)" class="input mt-1" type="datetime-local" @input="setDate(offer, 'expires_at', $event)" /></label>
        <button type="button" class="btn btn-secondary self-end" @click="remove(index)">{{ t('common.delete') }}</button>
      </div>
    </div>
    <button type="button" class="btn btn-secondary" @click="add">{{ t('admin.mediaProducts.addOffer') }}</button>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AdminGroup } from '@/types'
import type { MediaOffer } from '@/api/admin/mediaProducts'

const props = defineProps<{ modelValue: MediaOffer[]; groups: AdminGroup[] }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: MediaOffer[]): void; (event: 'error', value: string): void }>()
const { t } = useI18n()
const operationText = ref<string[]>([])

watch(() => props.modelValue, value => { operationText.value = value.map(offer => offer.operations.join(', ')) }, { immediate: true })
const sourceGroups = (provider: string) => props.groups.filter(group => group.platform === provider)
const jsonText = (value: Record<string, unknown>) => JSON.stringify(value, null, 2)
const localDate = (value: string) => value ? value.slice(0, 16) : ''

function add() {
  const now = new Date()
  emit('update:modelValue', [...props.modelValue, { provider: 'openai', source_group_id: 0, upstream_model: '', enabled: true, priority: 100, operations: ['generations'], capabilities: {}, cost_rules: {}, cost_source: '', cost_version: 'v1', verified_at: now.toISOString(), expires_at: new Date(now.getTime() + 30 * 86400000).toISOString() }])
}
function remove(index: number) { emit('update:modelValue', props.modelValue.filter((_, itemIndex) => itemIndex !== index)) }
function syncOperations(index: number) { props.modelValue[index].operations = operationText.value[index].split(',').map(value => value.trim()).filter(Boolean) }
function setJSON(offer: MediaOffer, key: 'capabilities' | 'cost_rules', event: Event) { try { offer[key] = JSON.parse((event.target as HTMLTextAreaElement).value) } catch { emit('error', t('admin.mediaProducts.invalidJSON')) } }
function setDate(offer: MediaOffer, key: 'verified_at' | 'expires_at', event: Event) { offer[key] = new Date((event.target as HTMLInputElement).value).toISOString() }
</script>
