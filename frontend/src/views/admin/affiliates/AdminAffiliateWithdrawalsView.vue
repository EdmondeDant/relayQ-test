<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="relative w-full md:w-80">
            <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input v-model="filters.search" type="text" class="input pl-10" :placeholder="t('admin.affiliates.records.searchPlaceholder')" @input="debounceLoad" />
          </div>
          <select v-model="filters.status" class="input w-full sm:w-36" @change="reloadFromFirstPage">
            <option value="">{{ t('admin.affiliates.withdrawals.allStatuses') }}</option>
            <option value="pending">{{ t('admin.affiliates.withdrawals.status.pending') }}</option>
            <option value="paid">{{ t('admin.affiliates.withdrawals.status.paid') }}</option>
          </select>
          <input v-model="filters.start_at" type="date" class="input w-full sm:w-44" :title="t('admin.affiliates.records.startAt')" @change="reloadFromFirstPage" />
          <input v-model="filters.end_at" type="date" class="input w-full sm:w-44" :title="t('admin.affiliates.records.endAt')" @change="reloadFromFirstPage" />
          <button class="btn btn-secondary px-2 md:px-3" :disabled="loading" :title="t('common.refresh')" @click="loadRecords">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="records"
          :loading="loading"
          :server-side-sort="true"
          default-sort-key="created_at"
          default-sort-order="desc"
          sort-storage-key="admin-affiliate-withdrawals-table-sort"
          @sort="handleSort"
        >
          <template #cell-user="{ row }">
            <div class="space-y-0.5">
              <div class="font-mono text-xs text-gray-500">#{{ row.user_id }}</div>
              <div class="text-sm font-medium text-gray-900 dark:text-white">{{ row.user_email || '-' }}</div>
              <div class="text-xs text-gray-500 dark:text-dark-400">{{ row.username || '-' }}</div>
            </div>
          </template>
          <template #cell-amount="{ row }">
            <span class="font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(row.amount) }}</span>
          </template>
          <template #cell-status="{ row }">
            <span :class="row.status === 'paid' ? 'rounded-full bg-emerald-100 px-2 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'rounded-full bg-amber-100 px-2 py-1 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'">
              {{ row.status === 'paid' ? t('admin.affiliates.withdrawals.status.paid') : t('admin.affiliates.withdrawals.status.pending') }}
            </span>
          </template>
          <template #cell-requested_at="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ formatDateTime(row.requested_at) }}</span>
          </template>
          <template #cell-paid_at="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ row.paid_at ? formatDateTime(row.paid_at) : '-' }}</span>
          </template>
          <template #cell-remark="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ row.remark || '-' }}</span>
          </template>
          <template #cell-actions="{ row }">
            <button v-if="row.status === 'pending'" class="btn btn-primary btn-sm" :disabled="markingId === row.id" @click="openMarkPaid(row)">
              {{ markingId === row.id ? t('admin.affiliates.withdrawals.marking') : t('admin.affiliates.withdrawals.markPaid') }}
            </button>
            <span v-else class="text-sm text-gray-400">-</span>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog :show="markDialogOpen" :title="t('admin.affiliates.withdrawals.markPaid')" width="normal" @close="markDialogOpen = false">
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-dark-300">
          {{ t('admin.affiliates.withdrawals.markPaidHint') }}
        </p>
        <textarea v-model="markRemark" class="input min-h-24" :placeholder="t('admin.affiliates.withdrawals.remarkPlaceholder')" />
        <div class="flex justify-end gap-2">
          <button class="btn btn-secondary" @click="markDialogOpen = false">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" :disabled="!selectedRecord || markingId != null" @click="markPaid">
            {{ t('admin.affiliates.withdrawals.confirmPaid') }}
          </button>
        </div>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { affiliatesAPI, type AffiliateWithdrawalRecord } from '@/api/admin/affiliates'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatCurrency, formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const records = ref<AffiliateWithdrawalRecord[]>([])
const filters = reactive({ search: '', status: '', start_at: '', end_at: '' })
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const sortState = reactive({ sort_by: 'created_at', sort_order: 'desc' as 'asc' | 'desc' })
const markDialogOpen = ref(false)
const markRemark = ref('')
const selectedRecord = ref<AffiliateWithdrawalRecord | null>(null)
const markingId = ref<number | null>(null)
let debounceTimer: ReturnType<typeof setTimeout> | null = null

const columns: Column[] = [
  { key: 'user', label: t('admin.affiliates.records.user'), sortable: true },
  { key: 'amount', label: t('admin.affiliates.withdrawals.amount'), sortable: true },
  { key: 'status', label: t('admin.affiliates.withdrawals.statusLabel'), sortable: true },
  { key: 'requested_at', label: t('admin.affiliates.withdrawals.requestedAt'), sortable: true },
  { key: 'paid_at', label: t('admin.affiliates.withdrawals.paidAt'), sortable: true },
  { key: 'remark', label: t('admin.affiliates.withdrawals.remark') },
  { key: 'actions', label: t('common.actions') },
]

function userTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone
  } catch {
    return 'UTC'
  }
}

async function loadRecords(): Promise<void> {
  loading.value = true
  try {
    const resp = await affiliatesAPI.listWithdrawalRecords({
      page: pagination.page,
      page_size: pagination.page_size,
      search: filters.search.trim() || undefined,
      status: filters.status || undefined,
      start_at: filters.start_at || undefined,
      end_at: filters.end_at || undefined,
      sort_by: sortState.sort_by,
      sort_order: sortState.sort_order,
      timezone: userTimezone(),
    })
    records.value = resp.items
    pagination.total = resp.total
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.affiliates.errors.loadFailed')))
  } finally {
    loading.value = false
  }
}

function reloadFromFirstPage(): void {
  pagination.page = 1
  void loadRecords()
}

function debounceLoad(): void {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(reloadFromFirstPage, 300)
}

function handleSort(key: string, order: 'asc' | 'desc'): void {
  sortState.sort_by = key
  sortState.sort_order = order
  reloadFromFirstPage()
}

function handlePageChange(page: number): void {
  pagination.page = page
  void loadRecords()
}

function handlePageSizeChange(pageSize: number): void {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadRecords()
}

function openMarkPaid(record: AffiliateWithdrawalRecord): void {
  selectedRecord.value = record
  markRemark.value = ''
  markDialogOpen.value = true
}

async function markPaid(): Promise<void> {
  if (!selectedRecord.value) return
  markingId.value = selectedRecord.value.id
  try {
    await affiliatesAPI.markWithdrawalPaid(selectedRecord.value.id, { remark: markRemark.value })
    appStore.showSuccess(t('admin.affiliates.withdrawals.markPaidSuccess'))
    markDialogOpen.value = false
    await loadRecords()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.affiliates.withdrawals.markPaidFailed')))
  } finally {
    markingId.value = null
  }
}

onMounted(loadRecords)
</script>
