<template>
  <div
    :class="[
      'inline-flex min-w-[220px] flex-col gap-1 rounded-lg border px-3 py-2 text-xs transition-colors',
      effectivePlatform
        ? platformBadgeClass(effectivePlatform)
        : 'border-gray-200 bg-gray-50 text-gray-700 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300',
    ]"
  >
    <div class="flex items-center gap-1.5 font-semibold">
      <PlatformIcon
        v-if="effectivePlatform"
        :platform="effectivePlatform as GroupPlatform"
        size="xs"
      />
      <span class="truncate">{{ model.name }}</span>
    </div>

    <div v-if="!model.pricing" class="text-[11px] text-gray-500 dark:text-gray-400">
      {{ noPricingLabel }}
    </div>

    <div v-else class="space-y-1 text-[11px] leading-5">
      <template v-if="model.pricing.billing_mode === BILLING_MODE_TOKEN">
        <div class="grid grid-cols-2 gap-x-2 gap-y-1">
          <PriceCell :label="t(prefixKey('inputPrice'))" :value="model.pricing.input_price" :scale="perMillionScale" :unit="t(prefixKey('unitPerMillion'))" />
          <PriceCell :label="t(prefixKey('outputPrice'))" :value="model.pricing.output_price" :scale="perMillionScale" :unit="t(prefixKey('unitPerMillion'))" />
          <PriceCell v-if="model.pricing.cache_write_price != null" :label="t(prefixKey('cacheWritePrice'))" :value="model.pricing.cache_write_price" :scale="perMillionScale" :unit="t(prefixKey('unitPerMillion'))" />
          <PriceCell v-if="model.pricing.cache_read_price != null" :label="t(prefixKey('cacheReadPrice'))" :value="model.pricing.cache_read_price" :scale="perMillionScale" :unit="t(prefixKey('unitPerMillion'))" />
        </div>
      </template>

      <div v-else-if="model.pricing.billing_mode === BILLING_MODE_PER_REQUEST" class="flex items-center justify-between gap-2">
        <span class="text-gray-500 dark:text-gray-400">{{ t(prefixKey('perRequestPrice')) }}</span>
        <span class="font-semibold">{{ formatScaled(model.pricing.per_request_price, 1) }}{{ t(prefixKey('unitPerRequest')) }}</span>
      </div>

      <div v-else-if="model.pricing.billing_mode === BILLING_MODE_IMAGE" class="flex items-center justify-between gap-2">
        <span class="text-gray-500 dark:text-gray-400">{{ t(prefixKey('imageOutputPrice')) }}</span>
        <span class="font-semibold">{{ formatScaled(model.pricing.image_output_price, 1) }}{{ t(prefixKey('unitPerRequest')) }}</span>
      </div>

      <div v-if="model.pricing.intervals && model.pricing.intervals.length > 0" class="border-t border-current/15 pt-1">
        <div class="mb-0.5 font-medium text-gray-600 dark:text-gray-300">{{ t(prefixKey('intervals')) }}</div>
        <div
          v-for="(iv, idx) in model.pricing.intervals"
          :key="idx"
          class="flex justify-between gap-2"
        >
          <span class="text-gray-500 dark:text-gray-400">
            <template v-if="iv.tier_label">{{ iv.tier_label }}</template>
            <template v-else>{{ formatRange(iv.min_tokens, iv.max_tokens) }}</template>
          </span>
          <span class="font-medium">{{ formatInterval(iv, model.pricing.billing_mode) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { defineComponent, h, type PropType } from 'vue'
import { formatScaled } from '@/utils/pricing'
import {
  BILLING_MODE_TOKEN,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_IMAGE,
  type BillingMode
} from '@/constants/channel'
import type { UserPricingInterval, UserSupportedModel } from '@/api/channels'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import type { GroupPlatform } from '@/types'
import { platformBadgeClass } from '@/utils/platformColors'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    model: UserSupportedModel
    pricingKeyPrefix?: string
    noPricingLabel?: string
    showPlatform?: boolean
    platformHint?: string
  }>(),
  {
    pricingKeyPrefix: 'availableChannels.pricing',
    noPricingLabel: '',
    showPlatform: true,
    platformHint: ''
  }
)

const { t } = useI18n()
const perMillionScale = 1_000_000
const effectivePlatform = props.model.platform || props.platformHint || ''

function prefixKey(k: string): string {
  return `${props.pricingKeyPrefix}.${k}`
}

function formatRange(min: number, max: number | null): string {
  const maxLabel = max == null ? '∞' : String(max)
  return `(${min}, ${maxLabel}]`
}

function formatInterval(iv: UserPricingInterval, mode: BillingMode): string {
  if (mode === BILLING_MODE_PER_REQUEST || mode === BILLING_MODE_IMAGE) {
    return formatScaled(iv.per_request_price, 1)
  }
  const input = formatScaled(iv.input_price, perMillionScale)
  const output = formatScaled(iv.output_price, perMillionScale)
  return `${input} / ${output}`
}

const PriceCell = defineComponent({
  props: {
    label: { type: String, required: true },
    value: { type: Number as PropType<number | null>, required: false, default: null },
    scale: { type: Number, required: true },
    unit: { type: String, required: true },
  },
  setup(cellProps) {
    return () => h('div', { class: 'rounded bg-white/60 px-2 py-1 dark:bg-dark-900/40' }, [
      h('div', { class: 'text-[10px] text-gray-500 dark:text-gray-400' }, cellProps.label),
      h('div', { class: 'font-semibold text-gray-900 dark:text-white' }, `${formatScaled(cellProps.value, cellProps.scale)}${cellProps.unit}`),
    ])
  },
})
</script>
