<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-3xl flex-col gap-5 pb-8">
      <section class="rounded-lg border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
        <h1 class="text-xl font-semibold text-gray-950 dark:text-dark-50">{{ t('canvas.title') }}</h1>
        <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-dark-300">{{ t('canvas.description') }}</p>
        <p class="mt-2 text-xs text-gray-400">{{ t('canvas.retention') }}</p>

        <div v-if="loading" class="mt-6 text-sm text-gray-500">{{ t('canvas.connecting') }}</div>
        <div v-else-if="error" class="mt-6 rounded-md bg-red-50 p-4 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-300">
          {{ error }}
        </div>
        <div v-else class="mt-6 flex flex-wrap gap-3">
          <button type="button" class="btn btn-primary" @click="openCanvas">{{ t('canvas.open') }}</button>
          <button type="button" class="btn btn-secondary" @click="connect">{{ t('canvas.retry') }}</button>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { bootstrapCanvas } from '@/api/canvas'

const { t } = useI18n()
const loading = ref(true)
const error = ref('')
const canvasConfig = ref<Awaited<ReturnType<typeof bootstrapCanvas>> | null>(null)

async function connect() {
  loading.value = true
  error.value = ''
  try {
    const config = await bootstrapCanvas()
    canvasConfig.value = config
    sessionStorage.setItem('relayq_canvas_bootstrap', JSON.stringify(config))
  } catch (err: any) {
    error.value = err?.message || t('canvas.failed')
  } finally {
    loading.value = false
  }
}

function openCanvas() {
  if (!canvasConfig.value) return
  const config = { ...canvasConfig.value, relayq_origin: window.location.origin }
  // Handoff bootstrap via window.name so the canvas app can pick it up after navigation.
  window.name = `relayq_canvas_bootstrap:${JSON.stringify(config)}`
  // Prefer same-origin /canvas-app/ (frontend vite proxy in dev, nginx in prod).
  const target = import.meta.env.VITE_CANVAS_DEV_URL || '/canvas-app/'
  window.location.assign(target)
}

onMounted(connect)
</script>
