<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6 px-6 py-6">
      <div class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">Grok 零售管理</h1>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          这套页面只管理隔离的零售 Grok Key，不进入现有 API Key 和现有用量主链。
        </p>
      </div>

      <div class="grid gap-6 lg:grid-cols-[380px,1fr]">
        <section class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">批量生成</h2>
            <button class="btn btn-secondary" :disabled="submitting" @click="resetForm">重置</button>
          </div>

          <div class="space-y-4">
            <label class="block">
              <span class="mb-1 block text-sm text-gray-600 dark:text-dark-300">xAI 分组</span>
              <select v-model.number="form.group_id" class="input w-full">
                <option :value="0">请选择分组</option>
                <option v-for="group in groups" :key="group.id" :value="group.id">
                  {{ group.name }} (#{{ group.id }})
                </option>
              </select>
            </label>

            <label class="block">
              <span class="mb-1 block text-sm text-gray-600 dark:text-dark-300">名称前缀</span>
              <input v-model="form.name_prefix" class="input w-full" placeholder="grok-retail" />
            </label>

            <div class="grid grid-cols-2 gap-4">
              <label class="block">
                <span class="mb-1 block text-sm text-gray-600 dark:text-dark-300">生成数量</span>
                <input v-model.number="form.count" type="number" min="1" max="200" class="input w-full" />
              </label>
              <label class="block">
                <span class="mb-1 block text-sm text-gray-600 dark:text-dark-300">有效期(天)</span>
                <input v-model.number="form.expires_in_days" type="number" min="1" class="input w-full" />
              </label>
            </div>

            <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <label class="block">
                <span class="mb-1 block text-sm text-gray-600 dark:text-dark-300">Token 上限</span>
                <input v-model.number="form.token_limit_total" type="number" min="0" class="input w-full" />
              </label>
              <label class="block">
                <span class="mb-1 block text-sm text-gray-600 dark:text-dark-300">图片上限</span>
                <input v-model.number="form.image_limit_total" type="number" min="0" class="input w-full" />
              </label>
              <label class="block">
                <span class="mb-1 block text-sm text-gray-600 dark:text-dark-300">视频上限</span>
                <input v-model.number="form.video_limit_total" type="number" min="0" class="input w-full" />
              </label>
            </div>

            <button class="btn btn-primary w-full" :disabled="submitting" @click="handleGenerate">
              {{ submitting ? '生成中...' : '生成零售 Key' }}
            </button>
            <p v-if="message" class="text-sm text-gray-500 dark:text-dark-400">{{ message }}</p>
            <p v-if="errorMessage" class="text-sm text-red-500">{{ errorMessage }}</p>
          </div>
        </section>

        <section class="space-y-6">
          <div class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">最近生成的 Key</h2>
              <div class="flex gap-2">
                <button class="btn btn-secondary" @click="loadKeys">刷新</button>
                <button class="btn btn-secondary" :disabled="generatedKeys.length === 0" @click="downloadCsv">导出 CSV</button>
              </div>
            </div>

            <div class="overflow-x-auto">
              <table class="min-w-full text-sm">
                <thead>
                  <tr class="border-b border-gray-200 text-left text-gray-500 dark:border-dark-700 dark:text-dark-400">
                    <th class="py-3 pr-4">ID</th>
                    <th class="py-3 pr-4">名称</th>
                    <th class="py-3 pr-4">Key</th>
                    <th class="py-3 pr-4">额度</th>
                    <th class="py-3 pr-4">到期</th>
                    <th class="py-3 pr-4">操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="item in generatedKeys" :key="item.id" class="border-b border-gray-100 align-top dark:border-dark-800">
                    <td class="py-3 pr-4 text-gray-900 dark:text-white">{{ item.id }}</td>
                    <td class="py-3 pr-4 text-gray-900 dark:text-white">{{ item.name }}</td>
                    <td class="py-3 pr-4">
                      <code class="break-all rounded bg-gray-100 px-2 py-1 text-xs dark:bg-dark-800">{{ item.key }}</code>
                    </td>
                    <td class="py-3 pr-4 text-xs text-gray-600 dark:text-dark-300">
                      <div>T: {{ item.token_used_total }}/{{ item.token_limit_total || '∞' }}</div>
                      <div>I: {{ item.image_used_total }}/{{ item.image_limit_total || '∞' }}</div>
                      <div>V: {{ item.video_used_total }}/{{ item.video_limit_total || '∞' }}</div>
                    </td>
                    <td class="py-3 pr-4 text-gray-600 dark:text-dark-300">{{ formatDate(item.expires_at) }}</td>
                    <td class="py-3 pr-4">
                      <button class="btn btn-secondary" @click="showUsage(item.id)">查看用量</button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <div v-if="usageSummary" class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
            <h2 class="mb-4 text-lg font-semibold text-gray-900 dark:text-white">Key 用量详情</h2>
            <div class="grid gap-4 sm:grid-cols-3">
              <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-dark-400">Token</div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ usageSummary.key.token_used_total }} / {{ usageSummary.key.token_limit_total || '∞' }}
                </div>
              </div>
              <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-dark-400">图片</div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ usageSummary.key.image_used_total }} / {{ usageSummary.key.image_limit_total || '∞' }}
                </div>
              </div>
              <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-dark-400">视频</div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ usageSummary.key.video_used_total }} / {{ usageSummary.key.video_limit_total || '∞' }}
                </div>
              </div>
            </div>

            <div class="mt-6 overflow-x-auto">
              <table class="min-w-full text-sm">
                <thead>
                  <tr class="border-b border-gray-200 text-left text-gray-500 dark:border-dark-700 dark:text-dark-400">
                    <th class="py-3 pr-4">时间</th>
                    <th class="py-3 pr-4">接口</th>
                    <th class="py-3 pr-4">模型</th>
                    <th class="py-3 pr-4">状态</th>
                    <th class="py-3 pr-4">消耗</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="log in usageSummary.recent_logs" :key="log.id" class="border-b border-gray-100 dark:border-dark-800">
                    <td class="py-3 pr-4 text-gray-600 dark:text-dark-300">{{ formatDate(log.created_at) }}</td>
                    <td class="py-3 pr-4 text-gray-900 dark:text-white">{{ log.inbound_endpoint }}</td>
                    <td class="py-3 pr-4 text-gray-900 dark:text-white">{{ log.model || '-' }}</td>
                    <td class="py-3 pr-4">
                      <span :class="log.status === 'success' ? 'text-emerald-600' : 'text-red-500'">{{ log.status }}</span>
                    </td>
                    <td class="py-3 pr-4 text-xs text-gray-600 dark:text-dark-300">
                      <div>tokens: {{ log.total_tokens }}</div>
                      <div>images: {{ log.image_count }}</div>
                      <div>videos: {{ log.video_count }}</div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { getAll as getAllGroups } from '@/api/admin/groups'
import {
  batchGenerateRetailGrokKeys,
  getRetailGrokKeyUsage,
  listRetailGrokKeys,
  type RetailGrokKey,
  type RetailGrokUsageSummary
} from '@/api/admin/retailGrok'
import type { AdminGroup } from '@/types'

const groups = ref<AdminGroup[]>([])
const generatedKeys = ref<RetailGrokKey[]>([])
const usageSummary = ref<RetailGrokUsageSummary | null>(null)
const submitting = ref(false)
const message = ref('')
const errorMessage = ref('')

const form = reactive({
  group_id: 0,
  count: 10,
  name_prefix: 'grok-retail',
  expires_in_days: 30,
  token_limit_total: 0,
  image_limit_total: 0,
  video_limit_total: 0
})

function resetForm() {
  form.group_id = 0
  form.count = 10
  form.name_prefix = 'grok-retail'
  form.expires_in_days = 30
  form.token_limit_total = 0
  form.image_limit_total = 0
  form.video_limit_total = 0
}

function formatDate(value: string | null | undefined) {
  if (!value) return '永不过期'
  return new Date(value).toLocaleString()
}

async function loadGroups() {
  groups.value = await getAllGroups('xai')
}

async function loadKeys() {
  generatedKeys.value = await listRetailGrokKeys(100)
}

async function handleGenerate() {
  errorMessage.value = ''
  message.value = ''
  submitting.value = true
  try {
    const result = await batchGenerateRetailGrokKeys({
      group_id: form.group_id,
      count: form.count,
      name_prefix: form.name_prefix,
      expires_in_days: form.expires_in_days || null,
      token_limit_total: form.token_limit_total,
      image_limit_total: form.image_limit_total,
      video_limit_total: form.video_limit_total
    })
    generatedKeys.value = result.keys
    message.value = `已生成 ${result.keys.length} 个零售 Key`
    if (result.keys[0]) {
      await showUsage(result.keys[0].id)
    }
  } catch (error: any) {
    errorMessage.value = error?.message || '生成失败'
  } finally {
    submitting.value = false
  }
}

async function showUsage(id: number) {
  usageSummary.value = await getRetailGrokKeyUsage(id)
}

function downloadCsv() {
  const rows = [
    ['id', 'name', 'key', 'status', 'group_id', 'expires_at', 'token_limit_total', 'image_limit_total', 'video_limit_total'],
    ...generatedKeys.value.map((item) => [
      item.id,
      item.name,
      item.key,
      item.status,
      item.group_id,
      item.expires_at ?? '',
      item.token_limit_total,
      item.image_limit_total,
      item.video_limit_total
    ])
  ]
  const csv = rows.map((row) => row.map((cell) => `"${String(cell).replace(/"/g, '""')}"`).join(',')).join('\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `retail-grok-keys-${Date.now()}.csv`
  link.click()
  URL.revokeObjectURL(url)
}

onMounted(async () => {
  await Promise.all([loadGroups(), loadKeys()])
})
</script>
