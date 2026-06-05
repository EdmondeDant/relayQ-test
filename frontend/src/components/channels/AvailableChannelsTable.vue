<template>
  <div class="card overflow-hidden">
    <table class="w-full table-fixed border-collapse text-sm">
      <thead>
        <tr class="border-b border-gray-100 bg-gray-50/50 text-xs font-medium uppercase tracking-wide text-gray-500 dark:border-dark-700 dark:bg-dark-800/50 dark:text-gray-400">
          <th class="w-[180px] px-4 py-3 text-center">{{ columns.name }}</th>
          <th class="w-[220px] px-4 py-3 text-left">{{ columns.description }}</th>
          <th class="px-4 py-3 text-left">{{ columns.supportedModels }}</th>
        </tr>
      </thead>
      <tbody v-if="loading">
        <tr>
          <td colspan="3" class="py-10 text-center">
            <Icon name="refresh" size="lg" class="inline-block animate-spin text-gray-400" />
          </td>
        </tr>
      </tbody>
      <tbody v-else-if="rows.length === 0">
        <tr>
          <td colspan="3" class="py-12 text-center">
            <Icon name="inbox" size="xl" class="mx-auto mb-3 h-12 w-12 text-gray-400" />
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ emptyLabel }}</p>
          </td>
        </tr>
      </tbody>
      <tbody
        v-else
        v-for="(channel, chIdx) in rows"
        :key="`${channel.name}-${chIdx}`"
        class="border-b-2 border-gray-200 last:border-b-0 dark:border-dark-600"
      >
        <tr class="transition-colors hover:bg-gray-50/40 dark:hover:bg-dark-800/40">
          <td class="px-4 py-3 text-center align-middle font-medium text-gray-900 dark:text-white">
            {{ channel.name }}
          </td>
          <td class="px-4 py-3 align-middle text-xs text-gray-500 dark:text-gray-400">
            <template v-if="channel.description">{{ channel.description }}</template>
            <span v-else class="text-gray-400">-</span>
          </td>
          <td class="align-top px-4 py-3">
            <div class="flex flex-col gap-3">
              <div
                v-for="section in channel.platforms"
                :key="`${channel.name}-${section.platform}`"
                class="space-y-2"
              >
                <div class="flex items-center gap-2">
                  <span
                    :class="[
                      'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[11px] font-medium uppercase',
                      platformBadgeClass(section.platform),
                    ]"
                  >
                    <PlatformIcon :platform="section.platform as GroupPlatform" size="xs" />
                    {{ section.platform }}
                  </span>
                </div>
                <div class="flex flex-wrap gap-2">
                  <SupportedModelChip
                    v-for="m in section.supported_models"
                    :key="`${section.platform}-${m.name}`"
                    :model="m"
                    :pricing-key-prefix="pricingKeyPrefix"
                    :no-pricing-label="noPricingLabel"
                    :show-platform="false"
                    :platform-hint="section.platform"
                  />
                  <span v-if="section.supported_models.length === 0" class="text-xs text-gray-400">
                    {{ noModelsLabel }}
                  </span>
                </div>
              </div>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import SupportedModelChip from './SupportedModelChip.vue'
import type { UserAvailableChannel } from '@/api/channels'
import type { GroupPlatform } from '@/types'
import { platformBadgeClass } from '@/utils/platformColors'

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

void props.userGroupRates
</script>
