<template>
  <div class="min-h-screen bg-gray-50 px-6 py-10 dark:bg-dark-950">
    <div class="mx-auto max-w-5xl space-y-6">
      <div class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
        <div class="flex items-center justify-between gap-4">
          <div>
            <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">Grok 零售 Key 查询</h1>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
              如需技术支持或长期合作 请访问主站：www.relayq.top
            </p>
          </div>
          <RouterLink to="/retail/grok/docs" class="btn btn-secondary">查看接口说明</RouterLink>
        </div>
      </div>

      <div class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-col gap-3 sm:flex-row">
          <input
            v-model="apiKey"
            type="password"
            class="input flex-1"
            placeholder="输入零售 Grok Key"
            @keydown.enter="queryUsage"
          />
          <button class="btn btn-primary" :disabled="loading" @click="queryUsage">
            {{ loading ? '查询中...' : '查询用量' }}
          </button>
        </div>
        <p v-if="errorMessage" class="mt-3 text-sm text-red-500">{{ errorMessage }}</p>
      </div>

      <template v-if="summary">
        <div class="grid gap-4 sm:grid-cols-3">
          <div class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
            <div class="text-sm text-gray-500 dark:text-dark-400">Token</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ summary.key.token_used_total }} / {{ summary.key.token_limit_total || '∞' }}
            </div>
          </div>
          <div class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
            <div class="text-sm text-gray-500 dark:text-dark-400">图片</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ summary.key.image_used_total }} / {{ summary.key.image_limit_total || '∞' }}
            </div>
          </div>
          <div class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
            <div class="text-sm text-gray-500 dark:text-dark-400">视频</div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ summary.key.video_used_total }} / {{ summary.key.video_limit_total || '∞' }}
            </div>
          </div>
        </div>

        <div class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
          <div class="mb-4 flex flex-wrap items-center gap-4 text-sm text-gray-500 dark:text-dark-400">
            <span>状态：<strong class="text-gray-900 dark:text-white">{{ summary.key.status }}</strong></span>
            <span>到期：<strong class="text-gray-900 dark:text-white">{{ formatDate(summary.key.expires_at) }}</strong></span>
            <span>分组：<strong class="text-gray-900 dark:text-white">#{{ summary.key.group_id }}</strong></span>
          </div>

          <div class="overflow-x-auto">
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
                <tr v-for="log in summary.recent_logs" :key="log.id" class="border-b border-gray-100 dark:border-dark-800">
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
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { fetchRetailGrokUsage, type RetailGrokUsageSummary } from '@/api/retailGrok'

const apiKey = ref('')
const loading = ref(false)
const errorMessage = ref('')
const summary = ref<RetailGrokUsageSummary | null>(null)

function formatDate(value: string | null | undefined) {
  if (!value) return '永不过期'
  return new Date(value).toLocaleString()
}

async function queryUsage() {
  errorMessage.value = ''
  loading.value = true
  try {
    summary.value = await fetchRetailGrokUsage(apiKey.value.trim())
  } catch (error: any) {
    errorMessage.value = error?.response?.data?.message || error?.message || '查询失败'
  } finally {
    loading.value = false
  }
}
</script>
