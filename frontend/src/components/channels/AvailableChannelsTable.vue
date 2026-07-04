<template>
  <table class="w-full table-fixed border-collapse text-sm">
      <thead>
        <tr class="border-b border-gray-100 bg-gray-50/50 text-xs font-medium uppercase tracking-wide text-gray-500 dark:border-dark-700 dark:bg-dark-800/50 dark:text-gray-400">
          <th class="px-4 py-3 text-left">
            <div class="grid grid-cols-1 gap-3 lg:grid-cols-4 lg:items-start">
              <div class="min-w-0">{{ columns.supportedModels }}</div>
              <div class="w-full text-center lg:w-[180px]">{{ t('availableChannels.columns.modelSummary') }}</div>
              <div class="w-full text-center lg:w-[220px]">所在分组</div>
              <div class="text-center">我们的价格（人民币/百万token）</div>
            </div>
          </th>
        </tr>
      </thead>
      <tbody v-if="loading">
        <tr>
          <td colspan="1" class="py-10 text-center">
            <Icon name="refresh" size="lg" class="inline-block animate-spin text-gray-400" />
          </td>
        </tr>
      </tbody>
      <tbody v-else-if="rows.length === 0">
        <tr>
          <td colspan="1" class="py-12 text-center">
            <Icon name="inbox" size="xl" class="mx-auto mb-3 h-12 w-12 text-gray-400" />
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ emptyLabel }}</p>
          </td>
        </tr>
      </tbody>
      <tbody v-else>
        <tr>
          <td class="align-top px-4 py-3">
            <div class="flex flex-col gap-3">
              <template v-if="allModels.length > 0">
                <div class="space-y-3">
                  <div
                    v-for="model in pagedModels"
                    :key="`${model.channelName}-${model.platform}-${model.name}`"
                    class="rounded-lg border border-gray-100 bg-white px-4 py-4 text-xs shadow-sm dark:border-dark-700 dark:bg-dark-900/30"
                  >
                    <div class="grid grid-cols-1 gap-3 lg:grid-cols-4 lg:items-start">
                      <div class="min-w-0">
                        <div class="break-words text-sm font-medium leading-6 text-gray-900 dark:text-white">
                          {{ model.name }}
                        </div>
                      </div>

                      <div class="w-full text-center lg:w-[180px]">
                        <div class="rounded-md bg-slate-50 px-2 py-1 text-left text-[12px] leading-5 text-slate-700 dark:bg-dark-800 dark:text-slate-200">
                          {{ model.summary?.trim() || '—' }}
                        </div>
                      </div>

                      <div class="w-full text-center text-gray-500 dark:text-gray-400 lg:w-[220px]">
                        {{ model.groupNames || '—' }}
                      </div>

                      <div class="text-center">
                        <div v-if="!model.pricing" class="leading-6 text-gray-400">{{ noPricingLabel }}</div>
                        <div v-else class="model-price-list">
                          <template v-if="model.pricing.billing_mode === BILLING_MODE_TOKEN">
                            <div class="price-line">输入价格 {{ formatScaled(model.pricing.input_price, perMillionScale) }}{{ unitPerMillion }}</div>
                            <div class="price-line">输出价格 {{ formatScaled(model.pricing.output_price, perMillionScale) }}{{ unitPerMillion }}</div>
                            <div v-if="model.pricing.cache_write_price != null" class="price-line">缓存写入价格 {{ formatScaled(model.pricing.cache_write_price, perMillionScale) }}{{ unitPerMillion }}</div>
                            <div v-if="model.pricing.cache_read_price != null" class="price-line">缓存读取价格 {{ formatScaled(model.pricing.cache_read_price, perMillionScale) }}{{ unitPerMillion }}</div>
                          </template>
                          <template v-else-if="model.pricing.billing_mode === BILLING_MODE_PER_REQUEST">
                            <div class="price-line">按次价格 {{ formatScaled(model.pricing.per_request_price, 1) }}{{ unitPerRequest }}</div>
                            <template v-if="getPricingIntervalPriceItems(model).length > 0">
                              <div v-for="item in getPricingIntervalPriceItems(model)" :key="`${model.name}-${item.label}`" class="price-line">
                                {{ item.label }} {{ formatScaled(item.value, 1) }}{{ unitPerRequest }}
                              </div>
                            </template>
                            <template v-else-if="getGroupImagePriceItems(model).length > 0">
                              <div v-for="item in getGroupImagePriceItems(model)" :key="`${model.name}-${item.label}`" class="price-line">
                                {{ item.label }} {{ formatScaled(item.value, 1) }}{{ unitPerRequest }}
                              </div>
                            </template>
                          </template>
                          <template v-else-if="model.pricing.billing_mode === BILLING_MODE_IMAGE">
                            <div class="price-line">按次价格 {{ formatScaled(model.pricing.per_request_price, 1) }}{{ unitPerRequest }}</div>
                            <template v-if="getPricingIntervalPriceItems(model).length > 0">
                              <div v-for="item in getPricingIntervalPriceItems(model)" :key="`${model.name}-${item.label}`" class="price-line">
                                {{ item.label }} {{ formatScaled(item.value, 1) }}{{ unitPerRequest }}
                              </div>
                            </template>
                            <template v-else-if="getGroupImagePriceItems(model).length > 0">
                              <div v-for="item in getGroupImagePriceItems(model)" :key="`${model.name}-${item.label}`" class="price-line">
                                {{ item.label }} {{ formatScaled(item.value, 1) }}{{ unitPerRequest }}
                              </div>
                            </template>
                          </template>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <div v-if="allModels.length > modelsPerPage" class="mt-3 flex flex-wrap items-center gap-2 text-xs">
                  <span class="mr-2 text-gray-400">
                    第 {{ currentPage }} / {{ getTotalPages(allModels.length) }} 页，共 {{ allModels.length }} 个模型
                  </span>
                  <button
                    v-for="page in getPageNumbers(allModels.length)"
                    :key="`page-${page}`"
                    type="button"
                    class="min-w-8 rounded border px-2 py-1 transition-colors"
                    :class="page === currentPage
                      ? 'border-primary-500 bg-primary-500 text-white'
                      : 'border-gray-200 text-gray-600 hover:bg-gray-50 dark:border-dark-700 dark:text-gray-300 dark:hover:bg-dark-700'"
                    @click="goToPage(page)"
                  >
                    {{ page }}
                  </button>
                </div>
              </template>
              <span v-else class="text-xs text-gray-400">
                {{ noModelsLabel }}
              </span>
            </div>
          </td>
        </tr>
      </tbody>
  </table>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { UserAvailableChannel, UserPricingInterval, UserSupportedModel } from '@/api/channels'
import { formatScaled } from '@/utils/pricing'
import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_TOKEN,
} from '@/constants/channel'

const props = defineProps<{
  columns: {
    name: string
    description: string
    supportedModels: string
  }
  rows: UserAvailableChannel[]
  loading: boolean
  pricingKeyPrefix: string
  noPricingLabel: string
  noModelsLabel: string
  emptyLabel: string
  userGroupRates: Record<number, number>
}>()

const { t } = useI18n()

void props.userGroupRates

const modelsPerPage = 5
const perMillionScale = 1_000_000
const unitPerMillion = ' / 1M token'
const unitPerRequest = ' / 次'
const currentPage = ref(1)

type DisplayModel = UserSupportedModel & {
  channelName: string
  platform: string
  groupNames: string
}

const allModels = computed<DisplayModel[]>(() =>
  props.rows
    .flatMap((channel) =>
      channel.platforms.flatMap((section) =>
        section.supported_models.map((model) => ({
          ...model,
          channelName: channel.name,
          platform: model.platform || section.platform,
          groupNames: section.groups.map((group) => group.name).join('、'),
        })),
      ),
    )
    .sort((a, b) => a.name.localeCompare(b.name))
)

const pagedModels = computed(() => {
  const safePage = Math.min(currentPage.value, getTotalPages(allModels.value.length))
  const start = (safePage - 1) * modelsPerPage
  return allModels.value.slice(start, start + modelsPerPage)
})

function getTotalPages(total: number): number {
  return Math.max(1, Math.ceil(total / modelsPerPage))
}

function goToPage(page: number) {
  currentPage.value = Math.max(1, Math.min(page, getTotalPages(allModels.value.length)))
}

function getPageNumbers(total: number): number[] {
  return Array.from({ length: getTotalPages(total) }, (_, index) => index + 1)
}

function getGroupImagePriceItems(model: UserSupportedModel) {
  const pricing = model.image_pricing
  if (!pricing) return []
  return [
    { label: '1K', value: pricing.price_1k },
    { label: '2K', value: pricing.price_2k },
    { label: '4K', value: pricing.price_4k },
  ].filter((item): item is { label: string; value: number } => item.value != null)
}

function getPricingIntervalPriceItems(model: UserSupportedModel) {
  const intervals = model.pricing?.intervals ?? []
  return intervals
    .filter((item): item is UserPricingInterval & { per_request_price: number } => item.per_request_price != null)
    .map((item) => ({
      label: item.tier_label || formatIntervalRange(item.min_tokens, item.max_tokens),
      value: item.per_request_price,
    }))
}

function formatIntervalRange(min: number, max: number | null) {
  if (max == null) return `${min}+`
  return `${min}-${max}`
}
</script>

<style scoped>
.model-price-list {
  @apply grid grid-cols-1 gap-1 sm:grid-cols-2;
}

.price-line {
  @apply min-w-0 rounded-md bg-gray-50 px-2 py-1 leading-5 text-gray-700 dark:bg-dark-800 dark:text-gray-200;
}
</style>
