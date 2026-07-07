<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col justify-between gap-4 lg:flex-row lg:items-start">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <div class="relative w-full sm:w-80">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('availableChannels.searchPlaceholder')"
                class="input pl-10"
              />
            </div>

            <div v-if="modelTotalPages > 1" class="flex flex-wrap items-center gap-2 rounded-xl border border-primary-100 bg-primary-50 px-3 py-2 text-sm shadow-sm dark:border-primary-900/40 dark:bg-primary-950/30">
              <span class="font-medium text-primary-700 dark:text-primary-300">
                第 {{ currentModelPage }} / {{ modelTotalPages }} 页，共 {{ totalModelCount }} 个模型
              </span>
              <button
                v-for="page in modelPageNumbers"
                :key="`model-page-${page}`"
                type="button"
                class="min-w-9 rounded-lg border px-3 py-1.5 text-sm font-semibold transition-colors"
                :class="page === currentModelPage
                  ? 'border-primary-600 bg-primary-600 text-white shadow'
                  : 'border-primary-200 bg-white text-primary-700 hover:bg-primary-100 dark:border-primary-800 dark:bg-dark-900 dark:text-primary-300 dark:hover:bg-primary-900/40'"
                @click="goToModelPage(page)"
              >
                {{ page }}
              </button>
            </div>
          </div>

          <div class="flex w-full flex-shrink-0 flex-wrap items-center justify-end gap-3 lg:w-auto">
            <button
              @click="loadChannels"
              :disabled="loading"
              class="btn btn-secondary"
              :title="t('common.refresh', 'Refresh')"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <AvailableChannelsTable
          :columns="columnLabels"
          :rows="filteredChannels"
          :loading="loading"
          :user-group-rates="userGroupRates"
          pricing-key-prefix="availableChannels.pricing"
          :no-pricing-label="t('availableChannels.noPricing')"
          :no-models-label="t('availableChannels.noModels')"
          :empty-label="t('availableChannels.empty')"
          :current-page="currentModelPage"
          :models-per-page="modelsPerPage"
        />
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import AvailableChannelsTable from '@/components/channels/AvailableChannelsTable.vue'
import userChannelsAPI, { type UserAvailableChannel } from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const channels = ref<UserAvailableChannel[]>([])
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)
const searchQuery = ref('')
const currentModelPage = ref(1)
const modelsPerPage = 5

const columnLabels = computed(() => ({
  name: t('availableChannels.columns.name'),
  description: t('availableChannels.columns.description'),
  supportedModels: t('availableChannels.columns.supportedModels'),
}))

/**
 * 搜索过滤：
 * - 命中渠道名/描述 → 整个渠道（所有 platforms）都保留
 * - 否则按 platform/group/model 维度在 sections 里过滤，保留有匹配的 section
 * - 所有 sections 都不匹配时，渠道本身被过滤掉
 */
const filteredChannels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return channels.value
  return channels.value
    .map((ch) => {
      const nameHit = ch.name.toLowerCase().includes(q)
      const descHit = (ch.description || '').toLowerCase().includes(q)
      if (nameHit || descHit) return ch
      const matchingSections = ch.platforms.filter(
        (p) =>
          p.platform.toLowerCase().includes(q) ||
          p.groups.some((g) => g.name.toLowerCase().includes(q)) ||
          p.supported_models.some((m) => m.name.toLowerCase().includes(q)),
      )
      if (matchingSections.length === 0) return null
      return { ...ch, platforms: matchingSections }
    })
    .filter((ch): ch is UserAvailableChannel => ch !== null)
})

const totalModelCount = computed(() =>
  filteredChannels.value.reduce(
    (total, channel) => total + channel.platforms.reduce((sum, section) => sum + section.supported_models.length, 0),
    0,
  ),
)

const modelTotalPages = computed(() => Math.max(1, Math.ceil(totalModelCount.value / modelsPerPage)))
const modelPageNumbers = computed(() => Array.from({ length: modelTotalPages.value }, (_, index) => index + 1))

function goToModelPage(page: number) {
  currentModelPage.value = Math.max(1, Math.min(page, modelTotalPages.value))
}

watch([searchQuery, totalModelCount], () => {
  currentModelPage.value = 1
})

async function loadChannels() {
  loading.value = true
  try {
    // 渠道列表和用户专属倍率并发拉取。专属倍率失败不阻塞渠道展示——
    // 失败时只是无法渲染专属倍率角标，降级为仅显示默认倍率。
    const [list, rates] = await Promise.all([
      userChannelsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((err: unknown) => {
        console.error('Failed to load user group rates:', err)
        return {} as Record<number, number>
      }),
    ])
    channels.value = list
    userGroupRates.value = rates
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(loadChannels)
</script>
