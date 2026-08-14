<template>
  <div class="space-y-3">
    <div v-for="(price, index) in modelValue" :key="index" class="grid gap-2 rounded-lg border border-gray-200 p-3 dark:border-dark-600 md:grid-cols-6">
      <select v-model="price.operation" class="input"><option v-for="operation in operations" :key="operation" :value="operation">{{ operation }}</option></select>
      <input v-model="price.spec_key" class="input md:col-span-2" :placeholder="t('admin.mediaProducts.specKey')" />
      <input v-model.number="price.unit_price_usd" class="input" type="number" min="0.0000000001" step="any" placeholder="USD" />
      <input v-model="price.version" class="input" :placeholder="t('admin.mediaProducts.version')" />
      <button type="button" class="btn btn-secondary" @click="remove(index)">{{ t('common.delete') }}</button>
    </div>
    <button type="button" class="btn btn-secondary" @click="add">{{ t('admin.mediaProducts.addPrice') }}</button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { MediaProductPrice } from '@/api/admin/mediaProducts'

const props = defineProps<{ modelValue: MediaProductPrice[]; modality: 'image' | 'video' }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: MediaProductPrice[]): void }>()
const { t } = useI18n()
const operations = ['generations', 'edits', 'extensions']

function add() {
  emit('update:modelValue', [...props.modelValue, { operation: 'generations', spec_key: '', unit_price_usd: 0, currency: 'USD', version: 'v1', enabled: true }])
}

function remove(index: number) {
  emit('update:modelValue', props.modelValue.filter((_, itemIndex) => itemIndex !== index))
}
</script>
