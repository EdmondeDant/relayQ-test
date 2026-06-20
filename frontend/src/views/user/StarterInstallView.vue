<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="starter-hero">
        <div class="starter-hero-glow starter-hero-glow-a"></div>
        <div class="starter-hero-glow starter-hero-glow-b"></div>

        <div class="relative z-10 flex flex-col gap-6 xl:flex-row xl:items-end xl:justify-between">
          <div class="max-w-3xl">
            <h2 class="mt-4 text-3xl font-black tracking-tight text-slate-950 dark:text-white sm:text-4xl">
              新人必装程序
            </h2>

            <div class="mt-5 flex flex-wrap gap-2">
              <span class="starter-chip">CC Switch 一键导入</span>
              <span class="starter-chip">Claude Code / OpenCode / Cursor</span>
              <span class="starter-chip">cURL / Python / Anthropic SDK</span>
              <span class="starter-chip">真实 Key 与 /models 联动</span>
            </div>

            <div class="mt-6 flex flex-wrap gap-3">
              <router-link to="/keys" class="starter-hero-btn starter-hero-btn-primary">
                去 API Keys
              </router-link>
              <a
                v-if="publicSettings?.doc_url"
                :href="publicSettings.doc_url"
                target="_blank"
                rel="noopener noreferrer"
                class="starter-hero-btn starter-hero-btn-secondary"
              >
                接入文档
              </a>
              <router-link v-else to="/contact-support" class="starter-hero-btn starter-hero-btn-secondary">
                联系我们
              </router-link>
              <router-link
                :to="{ path: '/home', hash: '#install-lab' }"
                class="starter-hero-btn starter-hero-btn-install-focus"
                aria-label="返回首页安装区"
              >
                一键生成安装openclaw或hermes的命令
              </router-link>
            </div>
          </div>

          <div class="grid gap-3 sm:grid-cols-3 xl:w-[460px]">
            <div class="starter-stat-card">
              <div class="starter-stat-value">{{ apiKeys.length }}</div>
              <div class="starter-stat-label">可用 API Keys</div>
            </div>
            <div class="starter-stat-card">
              <div class="starter-stat-value">{{ routeOptions.length }}</div>
              <div class="starter-stat-label">可切线路</div>
            </div>
            <div class="starter-stat-card">
              <div class="starter-stat-value">{{ availableModels.length || exportTools.length }}</div>
              <div class="starter-stat-label">模型 / 工具</div>
            </div>
          </div>
        </div>
      </section>

      <section class="starter-download-shell">
        <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
          <div>
            <div class="starter-section-kicker">下载工具</div>
            <h3 class="mt-2 text-2xl font-black tracking-tight text-slate-950 dark:text-white">
              CC Switch / Codex / Cherry Studio / Node.js / v2rayN7
            </h3>
            <p class="mt-3 max-w-3xl text-sm leading-7 text-slate-600 dark:text-slate-300">
              如果需要一键安装小龙虾或者hermes的话，去网站主页就可以。设置好apikey后命令行里一键安装。
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <span class="starter-chip">Windows</span>
            <span class="starter-chip">macOS</span>
            <span class="starter-chip">Linux</span>
          </div>
        </div>

        <div class="mt-6 grid gap-4 xl:grid-cols-[430px_minmax(0,1fr)]">
          <article v-if="featuredDownloadTool" class="starter-download-featured">
            <div class="flex items-start justify-between gap-3">
              <div>
                <div class="starter-download-kicker">{{ featuredDownloadTool.kicker }}</div>
                <div class="mt-2 text-3xl font-black tracking-tight text-slate-950 dark:text-white">
                  {{ featuredDownloadTool.title }}
                </div>
                <p class="mt-4 text-sm leading-7 text-slate-600 dark:text-slate-300">
                  {{ featuredDownloadTool.description }}
                </p>
              </div>
              <div class="starter-download-version">
                <div>Priority</div>
                <div class="mt-1 text-[11px] font-black uppercase tracking-[0.2em] text-slate-400 dark:text-slate-500">
                  {{ featuredDownloadTool.version }}
                </div>
              </div>
            </div>

            <div class="mt-5 flex flex-wrap gap-2">
              <span v-for="highlight in featuredDownloadTool.highlights" :key="highlight" class="starter-mini-chip">
                {{ highlight }}
              </span>
            </div>

            <div class="mt-5 flex flex-wrap gap-2">
              <a
                :href="featuredDownloadTool.primaryHref"
                target="_blank"
                rel="noopener noreferrer"
                class="starter-link-btn starter-link-btn-primary"
              >
                {{ featuredDownloadTool.primaryLabel }}
              </a>
              <a
                :href="featuredDownloadTool.secondaryHref"
                target="_blank"
                rel="noopener noreferrer"
                class="starter-link-btn starter-link-btn-secondary"
              >
                {{ featuredDownloadTool.secondaryLabel }}
              </a>
            </div>

            <div class="mt-6 starter-download-focus-box">
              <div class="starter-download-group-title">推荐下载</div>
              <div class="mt-3 flex flex-wrap gap-2">
                <a
                  v-for="link in getRecommendedLinks(featuredDownloadTool)"
                  :key="`${link.groupTitle}-${link.label}`"
                  :href="link.href"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="starter-download-link starter-download-link-recommended"
                >
                  <span>{{ link.groupTitle }}</span>
                  <span class="starter-download-dot"></span>
                  <span>{{ link.label }}</span>
                </a>
              </div>
            </div>

            <div class="mt-6 grid gap-3 sm:grid-cols-3">
              <div v-for="group in featuredDownloadTool.groups" :key="group.title" class="starter-download-platform-card">
                <div class="starter-download-group-title">{{ group.title }}</div>
                <div class="mt-2 text-2xl font-black text-slate-950 dark:text-white">{{ group.links.length }}</div>
                <div class="mt-1 text-sm text-slate-500 dark:text-slate-400">可选安装包</div>
              </div>
            </div>

            <div class="mt-6 space-y-3">
              <div v-for="group in featuredDownloadTool.groups" :key="group.title" class="starter-download-group">
                <div class="starter-download-group-title">{{ group.title }}</div>
                <div class="mt-2 flex flex-wrap gap-2">
                  <a
                    v-for="link in group.links"
                    :key="link.label"
                    :href="link.href"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="starter-download-link"
                    :class="{ 'starter-download-link-recommended': link.recommended }"
                  >
                    <span>{{ link.label }}</span>
                    <span v-if="link.recommended" class="starter-download-recommended">推荐</span>
                  </a>
                </div>
              </div>
            </div>
          </article>

          <div class="grid gap-4 xl:grid-cols-2">
            <article
              v-for="tool in secondaryDownloadTools"
              :key="tool.id"
              class="starter-download-card starter-download-card-compact"
            >
              <div class="flex items-start justify-between gap-3">
                <div>
                  <div class="starter-download-kicker">{{ tool.kicker }}</div>
                  <div class="mt-2 text-xl font-black text-slate-950 dark:text-white">{{ tool.title }}</div>
                </div>
                <span class="starter-status-chip">{{ tool.version }}</span>
              </div>

              <p class="mt-4 text-sm leading-7 text-slate-600 dark:text-slate-300">
                {{ tool.description }}
              </p>

              <div class="mt-4 flex flex-wrap gap-2">
                <span v-for="highlight in tool.highlights" :key="highlight" class="starter-mini-chip">
                  {{ highlight }}
                </span>
              </div>

              <div class="mt-5 flex flex-wrap gap-2">
                <a
                  :href="tool.primaryHref"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="starter-link-btn starter-link-btn-primary"
                >
                  {{ tool.primaryLabel }}
                </a>
                <a
                  :href="tool.secondaryHref"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="starter-link-btn starter-link-btn-secondary"
                >
                  {{ tool.secondaryLabel }}
                </a>
              </div>

              <div class="mt-5 grid gap-3">
                <div v-for="group in tool.groups" :key="group.title" class="starter-download-group">
                  <div class="flex items-center justify-between gap-3">
                    <div class="starter-download-group-title">{{ group.title }}</div>
                    <div class="text-xs font-bold text-slate-400 dark:text-slate-500">{{ group.links.length }} 项</div>
                  </div>
                  <div class="mt-2 flex flex-wrap gap-2">
                    <a
                      v-for="link in group.links"
                      :key="link.label"
                      :href="link.href"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="starter-download-link"
                      :class="{ 'starter-download-link-recommended': link.recommended }"
                    >
                      <span>{{ link.label }}</span>
                      <span v-if="link.recommended" class="starter-download-recommended">推荐</span>
                    </a>
                  </div>
                </div>
              </div>
            </article>
          </div>
        </div>
      </section>

      <section class="grid gap-6 xl:grid-cols-[340px_minmax(0,1fr)]">
        <aside class="space-y-4">
          <div class="starter-card starter-card-ccswitch-focus">
            <div class="flex items-start justify-between gap-3">
              <div class="starter-section-kicker">导入目标</div>
              <span class="starter-ccswitch-badge">CC Switch 专用</span>
            </div>
            <div class="mt-3 text-base font-black leading-7 text-slate-950 dark:text-white">
              这一块是专门替 CC Switch 跑腿的，你先点好要对接的客户端，后面的模型和线路就乖乖跟上。
            </div>
            <div class="mt-2 text-sm leading-7 text-slate-600 dark:text-slate-300">
              常用导入目标都帮你摆在前面了，少一点纠结，少一点绕路，配置就能更丝滑地落进对应客户端。
            </div>
          </div>

          <button
            v-for="app in ccswitchApps"
            :key="app.id"
            type="button"
            class="starter-select-card"
            :class="{ 'starter-select-card-active': selectedCcsApp === app.id }"
            @click="selectedCcsApp = app.id"
          >
            <div class="flex items-start justify-between gap-4">
              <div>
                <div class="text-lg font-black text-slate-950 dark:text-white">{{ app.title }}</div>
                <div class="mt-1 text-[11px] font-black uppercase tracking-[0.22em] text-slate-400 dark:text-slate-500">
                  {{ app.kicker }}
                </div>
              </div>
              <span class="starter-status-chip">
                {{ selectedCcsApp === app.id ? '当前目标' : '可导入' }}
              </span>
            </div>
            <p class="mt-4 text-sm leading-7 text-slate-600 dark:text-slate-300">
              {{ app.description }}
            </p>
            <div class="mt-4 flex flex-wrap gap-2">
              <span v-for="hint in app.hints" :key="hint" class="starter-mini-chip">{{ hint }}</span>
            </div>
          </button>

          <div class="starter-card">
            <div class="starter-section-kicker">配置导出</div>
            <div class="mt-3 grid gap-2">
              <button
                v-for="tool in exportTools"
                :key="tool.id"
                type="button"
                class="starter-tab-btn"
                :class="{ 'starter-tab-btn-active': selectedTool === tool.id }"
                @click="selectedTool = tool.id"
              >
                <div class="font-bold text-slate-900 dark:text-white">{{ tool.title }}</div>
                <div class="mt-1 text-[11px] uppercase tracking-[0.18em] text-slate-400 dark:text-slate-500">
                  {{ tool.kicker }}
                </div>
              </button>
            </div>
          </div>

          <div class="starter-card">
            <div class="starter-section-kicker">当前凭据</div>
            <div v-if="selectedApiKey" class="mt-3 space-y-3 text-sm text-slate-600 dark:text-slate-300">
              <div class="starter-info-row">
                <span>API Key</span>
                <span>{{ maskToken(selectedApiKey.key) }}</span>
              </div>
              <div class="starter-info-row">
                <span>分组</span>
                <span>{{ selectedApiKey.group?.name || '未分组' }}</span>
              </div>
              <div class="starter-info-row">
                <span>平台</span>
                <span>{{ selectedApiKey.group?.platform || 'anthropic' }}</span>
              </div>
            </div>
            <div v-else class="mt-3 text-sm leading-7 text-slate-500 dark:text-slate-400">
              还没有可用 API Key。先去 `API Keys` 页面创建一个，再回来一键导入。
            </div>
          </div>
        </aside>

        <div class="space-y-6">
          <section class="starter-panel">
            <div class="starter-panel-glow starter-panel-glow-a"></div>
            <div class="starter-panel-glow starter-panel-glow-b"></div>

            <div class="relative z-10 flex flex-col gap-4 border-b border-white/60 pb-5 dark:border-white/10 md:flex-row md:items-center md:justify-between">
              <div>
                <div class="text-xs font-black uppercase tracking-[0.22em] text-fuchsia-500">
                  {{ activeCcsApp.kicker }}
                </div>
                <div class="mt-2 text-2xl font-black text-slate-950 dark:text-white">
                  {{ activeCcsApp.title }} 一键导入
                </div>
                <p class="mt-2 max-w-3xl text-sm leading-7 text-slate-600 dark:text-slate-300">
                  {{ activeCcsApp.description }}
                </p>
              </div>
              <div class="flex flex-wrap gap-2">
                <span class="starter-runtime-pill">{{ providerName }}</span>
                <span class="starter-runtime-pill">{{ selectedRouteOption.name }}</span>
                <span class="starter-runtime-pill">{{ effectiveModel }}</span>
              </div>
            </div>

            <div class="relative z-10 mt-6 grid gap-4 lg:grid-cols-2">
              <label class="starter-field">
                <span class="starter-field-label">选择 API Key</span>
                <select v-model="selectedKeyId" class="starter-input">
                  <option :value="null" disabled>请选择一个可用 Key</option>
                  <option v-for="item in apiKeys" :key="item.id" :value="item.id">
                    {{ item.name }} · {{ maskToken(item.key) }}
                  </option>
                </select>
              </label>

              <label class="starter-field">
                <span class="starter-field-label">模型名称</span>
                <select
                  v-if="availableModels.length > 0"
                  v-model="modelInput"
                  class="starter-input"
                >
                  <option v-for="model in availableModels" :key="model" :value="model">
                    {{ model }}
                  </option>
                </select>
                <input
                  v-else
                  v-model="modelInput"
                  type="text"
                  class="starter-input"
                  :placeholder="activeCcsApp.defaultModel"
                >
              </label>
            </div>

            <div class="relative z-10 mt-5">
              <div class="starter-field-label">线路切换</div>
              <div class="mt-3 flex flex-wrap gap-2">
                <button
                  v-for="route in routeOptions"
                  :key="route.id"
                  type="button"
                  class="starter-route-chip"
                  :class="{ 'starter-route-chip-active': selectedRouteId === route.id }"
                  @click="selectedRouteId = route.id"
                >
                  <span>{{ route.name }}</span>
                  <span class="starter-route-chip-note">{{ route.badge }}</span>
                </button>
              </div>
              <div class="mt-3 text-sm text-slate-500 dark:text-slate-400">
                {{ selectedRouteOption.description }}
              </div>
            </div>

            <div class="relative z-10 mt-5 grid gap-3 lg:grid-cols-[auto_1fr] lg:items-center">
              <button
                type="button"
                class="starter-fetch-btn"
                :disabled="modelsLoading || !selectedApiKey"
                @click="fetchAvailableModels()"
              >
                {{ modelsLoading ? '正在拉取模型...' : '刷新 /models' }}
              </button>

              <div class="starter-fetch-meta">
                <span v-if="modelsError" class="text-rose-500 dark:text-rose-300">{{ modelsError }}</span>
                <template v-else>
                  <span class="text-emerald-600 dark:text-emerald-300">{{ modelFetchStatus }}</span>
                  <span class="h-1 w-1 rounded-full bg-slate-300 dark:bg-slate-600"></span>
                  <span>{{ serviceUrls.openai }}</span>
                </template>
              </div>
            </div>

            <div class="relative z-10 mt-6 grid gap-3 md:grid-cols-3">
              <div v-for="item in serviceUrlCards" :key="item.label" class="starter-config-card">
                <div class="text-[11px] font-black uppercase tracking-[0.18em] text-slate-400 dark:text-slate-500">
                  {{ item.label }}
                </div>
                <div class="mt-2 break-all text-sm font-bold text-slate-700 dark:text-slate-200">
                  {{ item.value }}
                </div>
                <button type="button" class="starter-inline-btn" @click="copyValue(item.value, `${item.label} 已复制`)">
                  复制地址
                </button>
              </div>
            </div>

            <div class="relative z-10 mt-6 rounded-[24px] border border-white/60 bg-white/70 p-4 shadow-[0_18px_50px_rgba(15,23,42,0.08)] dark:border-white/10 dark:bg-slate-950/40">
              <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                <div>
                  <div class="text-sm font-black text-slate-950 dark:text-white">
                    给 CC Switch 开个门
                  </div>
                  <p class="mt-2 max-w-2xl text-sm leading-7 text-slate-600 dark:text-slate-300">
                    你现在选中的是 `{{ activeCcsApp.title }}`，这边会把 endpoint、apiKey、model 和对应配置体一起打包好。
                    Key 选稳了，点一下就能把本地协议叫起来，省得你自己来回复制粘贴。
                  </p>
                </div>
                <div class="flex flex-wrap gap-2">
                  <button type="button" class="starter-action-btn starter-action-btn-primary" @click="triggerCcSwitchImport">
                    一键导入
                  </button>
                  <button type="button" class="starter-action-btn starter-action-btn-secondary" @click="copyCcSwitchLink">
                    复制导入链接
                  </button>
                </div>
              </div>

              <div class="mt-4 grid gap-3 md:grid-cols-3">
                <div class="starter-step-card">
                  <div class="starter-step-index">01</div>
                  <div class="starter-step-title">选目标</div>
                  <div class="starter-step-desc">选择 `Claude`、`Gemini`、`Codex` 或 `OpenCode` 作为导入目标。</div>
                </div>
                <div class="starter-step-card">
                  <div class="starter-step-index">02</div>
                  <div class="starter-step-title">选线路</div>
                  <div class="starter-step-desc">自动推导 OpenAI / Anthropic / Gemini 三类接入地址，不必手动拼 URL。</div>
                </div>
                <div class="starter-step-card">
                  <div class="starter-step-index">03</div>
                  <div class="starter-step-title">一键拉起</div>
                  <div class="starter-step-desc">深链直接把 provider 配置带给 CC Switch，减少手填 baseURL、key 和 model 的出错点。</div>
                </div>
              </div>
            </div>

            <div class="relative z-10 mt-6 overflow-hidden rounded-[24px] border border-slate-900/10 bg-slate-950 shadow-2xl shadow-slate-900/15 dark:border-white/10">
              <div class="flex items-center justify-between border-b border-white/10 px-4 py-3">
                <div class="flex items-center gap-2">
                  <span class="h-2.5 w-2.5 rounded-full bg-rose-400"></span>
                  <span class="h-2.5 w-2.5 rounded-full bg-amber-300"></span>
                  <span class="h-2.5 w-2.5 rounded-full bg-emerald-400"></span>
                  <span class="ml-2 text-xs font-black uppercase tracking-[0.2em] text-slate-400">
                    ccswitch deeplink
                  </span>
                </div>
                <button type="button" class="starter-command-btn" @click="copyCcSwitchLink">
                  复制链接
                </button>
              </div>
              <pre class="starter-command-pre"><code>{{ ccswitchImportLink }}</code></pre>
              <div class="flex flex-wrap gap-2 border-t border-white/10 px-4 py-3">
                <span class="starter-command-pill">{{ activeCcsApp.title }}</span>
                <span class="starter-command-pill">{{ selectedRouteOption.name }}</span>
                <span class="starter-command-pill">{{ effectiveModel }}</span>
              </div>
            </div>
          </section>

          <section class="starter-panel">
            <div class="relative z-10 flex flex-col gap-4 border-b border-white/60 pb-5 dark:border-white/10 md:flex-row md:items-center md:justify-between">
              <div>
                <div class="text-xs font-black uppercase tracking-[0.22em] text-cyan-500">
                  {{ activeExportTool.kicker }}
                </div>
                <div class="mt-2 text-2xl font-black text-slate-950 dark:text-white">
                  {{ activeExportTool.title }} 配置导出
                </div>
                <p class="mt-2 max-w-3xl text-sm leading-7 text-slate-600 dark:text-slate-300">
                  {{ activeExportTool.description }}
                </p>
              </div>
              <div class="flex flex-wrap gap-2">
                <button type="button" class="starter-action-btn starter-action-btn-secondary" @click="copyCurrentConfig">
                  复制配置
                </button>
                <button type="button" class="starter-action-btn starter-action-btn-primary" @click="downloadCurrentConfig">
                  下载文件
                </button>
              </div>
            </div>

            <div class="relative z-10 mt-6 grid gap-3 md:grid-cols-4">
              <div v-for="item in configPreviewItems" :key="item.label" class="starter-config-card">
                <div class="text-[11px] font-black uppercase tracking-[0.18em] text-slate-400 dark:text-slate-500">
                  {{ item.label }}
                </div>
                <div class="mt-2 break-all text-sm font-bold text-slate-700 dark:text-slate-200">
                  {{ item.value }}
                </div>
              </div>
            </div>

            <div class="relative z-10 mt-6 overflow-hidden rounded-[24px] border border-slate-900/10 bg-slate-950 shadow-2xl shadow-slate-900/15 dark:border-white/10">
              <div class="flex items-center justify-between border-b border-white/10 px-4 py-3">
                <div class="flex items-center gap-2">
                  <span class="h-2.5 w-2.5 rounded-full bg-rose-400"></span>
                  <span class="h-2.5 w-2.5 rounded-full bg-amber-300"></span>
                  <span class="h-2.5 w-2.5 rounded-full bg-emerald-400"></span>
                  <span class="ml-2 text-xs font-black uppercase tracking-[0.2em] text-slate-400">
                    export preview
                  </span>
                </div>
                <div class="flex items-center gap-2">
                  <span class="starter-command-pill">{{ exportFileName }}</span>
                  <button type="button" class="starter-command-btn" @click="copyCurrentConfig">
                    复制
                  </button>
                </div>
              </div>
              <pre class="starter-command-pre"><code>{{ currentToolConfig }}</code></pre>
            </div>
          </section>

          <div class="starter-bottom-card">
            <div class="leading-7">
              当前页面优先把 `CC Switch` 导入链路做完整；后续如果还要继续补更多客户端，也可以复用同一套 Key、
              线路和模型上下文，不必再重复造一套新表单。
            </div>
            <div class="flex flex-wrap gap-2">
              <router-link to="/keys" class="starter-link-btn starter-link-btn-primary">
                管理 API Keys
              </router-link>
              <router-link to="/contact-support" class="starter-link-btn starter-link-btn-secondary">
                联系我们
              </router-link>
            </div>
          </div>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { authAPI, keysAPI } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores'
import type { ApiKey, PublicSettings } from '@/types'
import {
  buildCcSwitchImportLink,
  buildServiceUrls,
  buildToolConfigExport,
  getToolExportFileName,
  type CcSwitchImportAppId,
  type ConfigExportToolId,
} from '@/utils/toolConfigExport'

interface CcSwitchAppCard {
  id: CcSwitchImportAppId
  title: string
  kicker: string
  description: string
  hints: string[]
  defaultModel: string
}

interface ExportToolCard {
  id: ConfigExportToolId
  title: string
  kicker: string
  description: string
}

type DownloadPlatformId = 'Windows' | 'macOS' | 'Linux'

interface DownloadToolLink {
  label: string
  href: string
  arch: string
  format: string
  sizeBytes: number
  recommended?: boolean
}

interface DownloadToolGroup {
  title: DownloadPlatformId
  links: DownloadToolLink[]
}

interface DownloadToolCard {
  id: string
  title: string
  kicker: string
  version: string
  description: string
  brandMark: string
  brandClass: string
  fallbackText: string
  primaryLabel: string
  primaryHref: string
  secondaryLabel: string
  secondaryHref: string
  highlights: string[]
  groups: DownloadToolGroup[]
}

interface RouteOption {
  id: string
  name: string
  badge: string
  description: string
  endpoint: string
}

const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const runtimeOrigin = ref(typeof window !== 'undefined' ? window.location.origin : '')
const publicSettings = ref<PublicSettings | null>(null)
const apiKeys = ref<ApiKey[]>([])
const loading = ref(false)
const selectedKeyId = ref<number | null>(null)
const selectedCcsApp = ref<CcSwitchImportAppId>('claude')
const selectedTool = ref<ConfigExportToolId>('claude-code')
const selectedRouteId = ref('official')
const modelInput = ref('')
const availableModels = ref<string[]>([])
const modelsLoading = ref(false)
const modelsError = ref('')
let modelFetchRequestId = 0

const ccswitchApps: CcSwitchAppCard[] = [
  {
    id: 'claude',
    title: 'Claude',
    kicker: 'Anthropic Workflow',
    description: '适合 Claude Code 直连场景，优先带上 Anthropic 兼容地址和默认模型。',
    hints: ['默认推荐', 'Anthropic 基座', '最稳妥'],
    defaultModel: 'claude-sonnet-4-5',
  },
  {
    id: 'openclaw',
    title: 'OpenClaw',
    kicker: 'OpenAI Agent',
    description: 'CC Switch 一键导入里使用频率很高的 Agent 之一，适合走 OpenAI 兼容地址快速接入。',
    hints: ['高频使用', 'OpenAI 兼容', 'Agent 场景'],
    defaultModel: 'claude-sonnet-4-5',
  },
  {
    id: 'hermes',
    title: 'Hermes',
    kicker: 'Chat Agent',
    description: '同样是常用 Agent 客户端，导入后可直接带上 Key、模型和当前线路，减少手填配置。',
    hints: ['高频使用', 'Chat Completions', '一键导入'],
    defaultModel: 'claude-sonnet-4-5',
  },
  {
    id: 'gemini',
    title: 'Gemini',
    kicker: 'Gemini CLI',
    description: '用于 Gemini CLI 或 Gemini 兼容接入，自动切到 Gemini 所需的路由前缀。',
    hints: ['Gemini CLI', 'v1beta', '谷歌模型'],
    defaultModel: 'gemini-2.5-pro',
  },
  {
    id: 'codex',
    title: 'Codex',
    kicker: 'OpenAI Responses',
    description: '更适合 Codex / OpenAI 风格客户端，导入时会使用 OpenAI 兼容 endpoint。',
    hints: ['OpenAI 兼容', 'Responses', '代码场景'],
    defaultModel: 'gpt-5',
  },
  {
    id: 'opencode',
    title: 'OpenCode',
    kicker: 'Provider Config',
    description: '保留 OpenCode provider 配置导入能力，便于桌面端或本地工作流快速接管当前线路。',
    hints: ['Provider Config', '本地工作流', '可扩展'],
    defaultModel: 'claude-sonnet-4-5',
  },
]

const exportTools: ExportToolCard[] = [
  {
    id: 'claude-code',
    title: 'Claude Code',
    kicker: 'JSON Env',
    description: '导出给 Claude Code 的环境配置，适合直接落配置文件或二次集成。',
  },
  {
    id: 'opencode',
    title: 'OpenCode',
    kicker: 'Provider Config',
    description: '导出 OpenCode 可直接消费的 provider 配置，保留模型和 baseURL 结构。',
  },
  {
    id: 'cursor',
    title: 'Cursor',
    kicker: 'Text Guide',
    description: '输出适合 Cursor 手动粘贴的 API Key、Base URL 和 Model。',
  },
  {
    id: 'curl',
    title: 'cURL',
    kicker: 'Quick Request',
    description: '生成一条可直接打 OpenAI 兼容接口的示例请求，用于联调检查。',
  },
  {
    id: 'python-sdk',
    title: 'Python SDK',
    kicker: 'OpenAI Client',
    description: '生成 Python OpenAI SDK 调用样例，适合脚本验证和本地集成。',
  },
  {
    id: 'anthropic-sdk',
    title: 'Anthropic SDK',
    kicker: 'Native Anthropic',
    description: '生成 Anthropic Python SDK 示例，适合原生 Claude 接入链路。',
  },
]

const toProjectDownload = (slug: string) => `/downloads/${slug}`

function createDownloadLink(
  label: string,
  slug: string,
  arch: string,
  format: string,
  sizeBytes: number,
  recommended = false
): DownloadToolLink {
  return {
    label,
    href: toProjectDownload(slug),
    arch,
    format,
    sizeBytes,
    recommended,
  }
}

function createExternalDownloadLink(
  label: string,
  href: string,
  arch: string,
  format: string,
  sizeBytes: number,
  recommended = false
): DownloadToolLink {
  return {
    label,
    href,
    arch,
    format,
    sizeBytes,
    recommended,
  }
}

const downloadTools: DownloadToolCard[] = [
  {
    id: 'cc-switch',
    title: 'CC Switch',
    kicker: 'Provider Manager',
    version: 'v3.14.1',
    description: '使用中转站必备的小插件，可以统一管理主流的agents框架工具。另外我在下面写了一套协议，可以一键唤醒并设置apikey。非常方便切模型。再也不用对着json发愁了。新人尽量装msi版，zip版的没办法一键导出设置。',
    brandMark: 'CC',
    brandClass: 'starter-download-brand-purple',
    fallbackText: '当前平台暂无推荐包，请切换到其他平台查看可用安装器。',
    primaryLabel: '官方仓库',
    primaryHref: 'https://github.com/farion1231/cc-switch',
    secondaryLabel: '最新发布',
    secondaryHref: 'https://github.com/farion1231/cc-switch/releases/latest',
    highlights: ['一键导入', '多客户端统一管理', '优先推荐安装'],
    groups: [
      {
        title: 'Windows',
        links: [
          createDownloadLink('Windows x64 MSI', 'cc-switch-windows-x64-msi', 'x64', 'MSI', 11726848, true),
          createDownloadLink('Windows x64 Portable ZIP', 'cc-switch-windows-x64-portable-zip', 'x64', 'ZIP', 11411054),
        ],
      },
      {
        title: 'macOS',
        links: [
          createDownloadLink('macOS Universal DMG', 'cc-switch-macos-universal-dmg', 'Universal', 'DMG', 24515295, true),
          createDownloadLink('macOS Universal ZIP', 'cc-switch-macos-universal-zip', 'Universal', 'ZIP', 24534878),
          createDownloadLink('macOS Universal TAR.GZ', 'cc-switch-macos-universal-tar-gz', 'Universal', 'TAR.GZ', 25122972),
        ],
      },
      {
        title: 'Linux',
        links: [
          createDownloadLink('Linux x64 DEB', 'cc-switch-linux-x64-deb', 'x64', 'DEB', 11895708, true),
          createDownloadLink('Linux ARM64 DEB', 'cc-switch-linux-arm64-deb', 'ARM64', 'DEB', 11433276),
          createDownloadLink('Linux x64 RPM', 'cc-switch-linux-x64-rpm', 'x64', 'RPM', 11896466),
          createDownloadLink('Linux ARM64 RPM', 'cc-switch-linux-arm64-rpm', 'ARM64', 'RPM', 11436384),
          createDownloadLink('Linux x64 AppImage', 'cc-switch-linux-x64-appimage', 'x64', 'AppImage', 90749432),
          createDownloadLink('Linux ARM64 AppImage', 'cc-switch-linux-arm64-appimage', 'ARM64', 'AppImage', 88398344),
        ],
      },
    ],
  },
  {
    id: 'codex',
    title: 'Codex',
    kicker: 'OpenAI Desktop',
    version: '0.128.0',
    description: 'OpenAI 官方 Codex 桌面版，提供 Windows / macOS 站内中转下载；Linux 可继续使用 npm 安装。',
    brandMark: 'OX',
    brandClass: 'starter-download-brand-cyan',
    fallbackText: 'Linux 目前没有官方桌面包，建议按安装说明使用 npm 安装 Codex CLI。',
    primaryLabel: '安装说明',
    primaryHref: 'https://github.com/openai/codex#installation',
    secondaryLabel: '最新发布',
    secondaryHref: 'https://github.com/openai/codex/releases/latest',
    highlights: ['官方桌面版', 'Windows / macOS', '代码工作流'],
    groups: [
      {
        title: 'Windows',
        links: [
          createDownloadLink('Windows x64 Desktop EXE', 'codex-windows-x64-exe', 'x64', 'EXE', 245125936, true),
          createDownloadLink('Windows ARM64 Desktop EXE', 'codex-windows-arm64-exe', 'ARM64', 'EXE', 207270704),
        ],
      },
      {
        title: 'macOS',
        links: [
          createDownloadLink('macOS Apple Silicon Desktop DMG', 'codex-macos-arm64-dmg', 'ARM64', 'DMG', 92905254, true),
          createDownloadLink('macOS Intel Desktop DMG', 'codex-macos-intel-dmg', 'Intel', 'DMG', 104808046),
        ],
      },
    ],
  },
  {
    id: 'cherry-studio',
    title: 'Cherry Studio',
    kicker: 'Desktop Workspace',
    version: 'v1.9.2',
    description: '本地知识库搭建，日常桌面端的ai小助手，可以保留全部记忆，帮你做记忆备份。',
    brandMark: 'CH',
    brandClass: 'starter-download-brand-rose',
    fallbackText: '当前平台包已列出，优先推荐 Setup 或 AppImage 版本。',
    primaryLabel: '接入文档',
    primaryHref: 'https://docs.cherry-ai.com/advanced-basic/providers/custom-providers',
    secondaryLabel: '最新发布',
    secondaryHref: 'https://github.com/CherryHQ/cherry-studio/releases/latest',
    highlights: ['桌面工作台', '自定义 Provider', '多平台可用'],
    groups: [
      {
        title: 'Windows',
        links: [
          createDownloadLink('Windows x64 Setup EXE', 'cherry-studio-windows-x64-setup-exe', 'x64', 'EXE', 139520901, true),
          createDownloadLink('Windows x64 Portable EXE', 'cherry-studio-windows-x64-portable-exe', 'x64', 'Portable', 139154633),
          createDownloadLink('Windows ARM64 Setup EXE', 'cherry-studio-windows-arm64-setup-exe', 'ARM64', 'EXE', 133145036),
        ],
      },
      {
        title: 'macOS',
        links: [
          createDownloadLink('macOS Intel DMG', 'cherry-studio-macos-intel-dmg', 'Intel', 'DMG', 196713587, true),
          createDownloadLink('macOS Apple Silicon DMG', 'cherry-studio-macos-arm64-dmg', 'ARM64', 'DMG', 186294940),
        ],
      },
      {
        title: 'Linux',
        links: [
          createDownloadLink('Linux x64 AppImage', 'cherry-studio-linux-x64-appimage', 'x64', 'AppImage', 220743039, true),
          createDownloadLink('Linux x64 DEB', 'cherry-studio-linux-x64-deb', 'x64', 'DEB', 170308068),
          createDownloadLink('Linux ARM64 AppImage', 'cherry-studio-linux-arm64-appimage', 'ARM64', 'AppImage', 217270143),
          createDownloadLink('Linux ARM64 DEB', 'cherry-studio-linux-arm64-deb', 'ARM64', 'DEB', 159616796),
        ],
      },
    ],
  },
  {
    id: 'nodejs',
    title: 'Node.js',
    kicker: 'Runtime Foundation',
    version: 'v24.15.0',
    description: '运行 Codex CLI、Claude Code 相关工具和常见前端开发工具所需的 Node.js LTS 环境。',
    brandMark: 'JS',
    brandClass: 'starter-download-brand-amber',
    fallbackText: '当前平台包已列出，优先使用 MSI、PKG 或 TAR 包快速完成运行时安装。',
    primaryLabel: '官方下载',
    primaryHref: 'https://nodejs.org/en/download',
    secondaryLabel: '版本列表',
    secondaryHref: 'https://nodejs.org/dist/latest/',
    highlights: ['CLI 运行时', 'LTS 环境', '开发基础依赖'],
    groups: [
      {
        title: 'Windows',
        links: [
          createDownloadLink('Windows x64 MSI', 'nodejs-windows-x64-msi', 'x64', 'MSI', 32497664, true),
          createDownloadLink('Windows ARM64 ZIP', 'nodejs-windows-arm64-zip', 'ARM64', 'ZIP', 32728930),
        ],
      },
      {
        title: 'macOS',
        links: [
          createDownloadLink('macOS Intel PKG', 'nodejs-macos-intel-pkg', 'Intel', 'PKG', 91592363, true),
          createDownloadLink('macOS Apple Silicon TAR', 'nodejs-macos-arm64-tar-gz', 'ARM64', 'TAR.GZ', 51450940),
        ],
      },
      {
        title: 'Linux',
        links: [
          createDownloadLink('Linux x64 TAR', 'nodejs-linux-x64-tar-xz', 'x64', 'TAR.XZ', 31164460, true),
          createDownloadLink('Linux ARM64 TAR', 'nodejs-linux-arm64-tar-xz', 'ARM64', 'TAR.XZ', 30108656),
        ],
      },
    ],
  },
  {
    id: 'v2rayn7',
    title: 'v2rayN7',
    kicker: 'Proxy Client',
    version: 'v7.22.7',
    description: '站长一直喜欢用的梯子客户端，适合日常代理切换和节点管理；这里保留常用平台包，并附上站长分享的邀请链接。',
    brandMark: 'VN',
    brandClass: 'starter-download-brand-cyan',
    fallbackText: '当前平台下载项已列出，优先选择 desktop 或 DMG / DEB 包完成安装。',
    primaryLabel: '订阅注册链接',
    primaryHref: 'https://ir.allblueaff.com/HgT0dOSY',
    secondaryLabel: '最新发布',
    secondaryHref: 'https://github.com/2dust/v2rayN/releases/latest',
    highlights: ['站长常用', '代理客户端', '节点管理'],
    groups: [
      {
        title: 'Windows',
        links: [
          createExternalDownloadLink('Windows x64 Desktop ZIP', 'https://github.com/2dust/v2rayN/releases/download/7.22.7/v2rayN-windows-64-desktop.zip', 'x64', 'Desktop ZIP', 131461665, true),
          createExternalDownloadLink('Windows x64 Full ZIP', 'https://github.com/2dust/v2rayN/releases/download/7.22.7/v2rayN-windows-64.zip', 'x64', 'ZIP', 166731939),
          createExternalDownloadLink('Windows ARM64 Desktop ZIP', 'https://github.com/2dust/v2rayN/releases/download/7.22.7/v2rayN-windows-arm64-desktop.zip', 'ARM64', 'Desktop ZIP', 123346763),
        ],
      },
      {
        title: 'macOS',
        links: [
          createExternalDownloadLink('macOS Apple Silicon DMG', 'https://github.com/2dust/v2rayN/releases/download/7.22.7/v2rayN-macos-arm64.dmg', 'ARM64', 'DMG', 123728385, true),
          createExternalDownloadLink('macOS Intel DMG', 'https://github.com/2dust/v2rayN/releases/download/7.22.7/v2rayN-macos-64.dmg', 'Intel', 'DMG', 130127526),
        ],
      },
      {
        title: 'Linux',
        links: [
          createExternalDownloadLink('Linux x64 DEB', 'https://github.com/2dust/v2rayN/releases/download/7.22.7/v2rayN-linux-64.deb', 'x64', 'DEB', 79733692, true),
          createExternalDownloadLink('Linux ARM64 DEB', 'https://github.com/2dust/v2rayN/releases/download/7.22.7/v2rayN-linux-arm64.deb', 'ARM64', 'DEB', 72276632),
        ],
      },
    ],
  },
]

const featuredDownloadTool = computed(() => downloadTools[0] ?? null)
const secondaryDownloadTools = computed(() => downloadTools.slice(1))

function getRecommendedLinks(tool: DownloadToolCard) {
  return tool.groups.flatMap((group) =>
    group.links
      .filter((link) => link.recommended)
      .map((link) => ({
        ...link,
        groupTitle: group.title,
      }))
  )
}

const activeCcsApp = computed(() =>
  ccswitchApps.find((item) => item.id === selectedCcsApp.value) ?? ccswitchApps[0]
)

const activeExportTool = computed(() =>
  exportTools.find((item) => item.id === selectedTool.value) ?? exportTools[0]
)

const selectedApiKey = computed(() =>
  apiKeys.value.find((item) => item.id === selectedKeyId.value) ?? null
)

const providerName = computed(() => publicSettings.value?.site_name?.trim() || 'RelayQ')

const routeOptions = computed<RouteOption[]>(() => {
  const options: RouteOption[] = []
  const pushedEndpoints = new Set<string>()

  const pushOption = (option: RouteOption) => {
    const normalized = normalizeUrl(option.endpoint)
    if (!normalized || pushedEndpoints.has(normalized)) return
    pushedEndpoints.add(normalized)
    options.push({ ...option, endpoint: normalized })
  }

  pushOption({
    id: 'official',
    name: '默认线路',
    badge: 'Official',
    description: '优先使用当前站点公开设置里的主 API 地址。',
    endpoint: publicSettings.value?.api_base_url || runtimeOrigin.value,
  })

  const currentOrigin = normalizeUrl(runtimeOrigin.value)
  const officialEndpoint = normalizeUrl(publicSettings.value?.api_base_url || '')
  if (currentOrigin && currentOrigin !== officialEndpoint) {
    pushOption({
      id: 'current-site',
      name: '当前站点',
      badge: 'Origin',
      description: '直接使用你当前访问的 RelayQ 站点地址。',
      endpoint: currentOrigin,
    })
  }

  ;(publicSettings.value?.custom_endpoints || []).forEach((item, index) => {
    pushOption({
      id: `custom-${index}`,
      name: item.name || `线路 ${index + 1}`,
      badge: 'Custom',
      description: item.description || '来自公开设置中的自定义接入地址。',
      endpoint: item.endpoint,
    })
  })

  return options
})

const selectedRouteOption = computed(() =>
  routeOptions.value.find((item) => item.id === selectedRouteId.value) ?? routeOptions.value[0]
)

const effectiveModel = computed(() => modelInput.value.trim() || activeCcsApp.value.defaultModel)

const serviceUrls = computed(() =>
  buildServiceUrls(selectedRouteOption.value?.endpoint || runtimeOrigin.value)
)

const ccswitchImportLink = computed(() =>
  buildCcSwitchImportLink(activeCcsApp.value.id, {
    providerName: providerName.value,
    homepage: normalizeUrl(runtimeOrigin.value),
    routeBaseUrl: selectedRouteOption.value?.endpoint || runtimeOrigin.value,
    apiKey: selectedApiKey.value?.key || 'sk-your-token',
    modelName: effectiveModel.value,
  })
)

const currentToolConfig = computed(() =>
  buildToolConfigExport(activeExportTool.value.id, {
    providerName: providerName.value,
    homepage: normalizeUrl(runtimeOrigin.value),
    routeBaseUrl: selectedRouteOption.value?.endpoint || runtimeOrigin.value,
    apiKey: selectedApiKey.value?.key || 'sk-your-token',
    modelName: effectiveModel.value,
  })
)

const exportFileName = computed(() => getToolExportFileName(activeExportTool.value.id))

const serviceUrlCards = computed(() => [
  { label: 'OpenAI Base URL', value: serviceUrls.value.openai },
  { label: 'Anthropic Base URL', value: serviceUrls.value.anthropic },
  { label: 'Gemini Base URL', value: serviceUrls.value.gemini },
])

const configPreviewItems = computed(() => [
  { label: 'Provider', value: providerName.value },
  { label: 'Key', value: maskToken(selectedApiKey.value?.key || '') },
  { label: 'Model', value: effectiveModel.value },
  { label: 'File', value: exportFileName.value },
])

const modelFetchStatus = computed(() => {
  if (modelsLoading.value) return '正在连接 /v1/models...'
  if (availableModels.value.length > 0) return `已同步 ${availableModels.value.length} 个模型`
  return '选择 Key 后自动读取模型列表'
})

function normalizeUrl(value: string) {
  const trimmed = value.trim().replace(/\/+$/, '')
  return trimmed || ''
}

function maskToken(token: string) {
  if (!token) return '未选择'
  if (token.length <= 10) return token
  return `${token.slice(0, 6)}...${token.slice(-4)}`
}

async function loadAllApiKeys() {
  const pageSize = 100
  let page = 1
  const rows: ApiKey[] = []

  while (true) {
    const response = await keysAPI.list(page, pageSize, {
      sort_by: 'created_at',
      sort_order: 'desc',
    })

    rows.push(...response.items)
    if (page >= response.pages || response.items.length === 0) {
      break
    }
    page += 1
  }

  return rows
}

async function loadPageData() {
  loading.value = true
  try {
    const [settings, keys] = await Promise.all([
      authAPI.getPublicSettings(),
      loadAllApiKeys(),
    ])

    publicSettings.value = settings
    apiKeys.value = keys

    if (keys.length > 0 && !selectedKeyId.value) {
      selectedKeyId.value = keys[0].id
    }

    if (!modelInput.value.trim()) {
      modelInput.value = activeCcsApp.value.defaultModel
    }
  } catch (error) {
    console.error('Failed to load starter install data:', error)
    appStore.showError('新人必装程序页面初始化失败')
  } finally {
    loading.value = false
  }
}

async function copyValue(value: string, message: string) {
  await copyToClipboard(value, message)
}

async function copyCcSwitchLink() {
  if (!selectedApiKey.value) {
    appStore.showError('请先选择一个 API Key')
    return
  }
  await copyToClipboard(ccswitchImportLink.value, 'CC Switch 导入链接已复制')
}

async function copyCurrentConfig() {
  await copyToClipboard(currentToolConfig.value, '配置内容已复制')
}

function downloadCurrentConfig() {
  const blob = new Blob([currentToolConfig.value], {
    type: exportFileName.value.endsWith('.json')
      ? 'application/json;charset=utf-8;'
      : 'text/plain;charset=utf-8;',
  })

  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = exportFileName.value
  link.click()
  window.URL.revokeObjectURL(url)
  appStore.showSuccess('配置文件已下载')
}

function triggerCcSwitchImport() {
  if (!selectedApiKey.value) {
    appStore.showError('请先选择一个 API Key')
    return
  }

  try {
    window.open(ccswitchImportLink.value, '_self')
    setTimeout(() => {
      if (document.hasFocus()) {
        appStore.showError('没有检测到 CC Switch 协议响应，请先确认本机已经安装 CC Switch')
      }
    }, 120)
  } catch {
    appStore.showError('拉起 CC Switch 失败，请检查客户端是否已安装')
  }
}

async function fetchAvailableModels(options: { silent?: boolean } = {}) {
  const { silent = false } = options

  if (!selectedApiKey.value) {
    modelsError.value = '请先选择一个 API Key'
    if (!silent) {
      appStore.showError(modelsError.value)
    }
    return
  }

  const requestId = ++modelFetchRequestId
  modelsLoading.value = true
  modelsError.value = ''

  try {
    const response = await fetch(`${serviceUrls.value.openai}/models`, {
      headers: {
        Authorization: `Bearer ${selectedApiKey.value.key}`,
      },
    })

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }

    const payload = await response.json() as { data?: Array<{ id?: string }> }
    const models = Array.from(
      new Set(
        (payload.data || [])
          .map((item) => item.id?.trim())
          .filter((item): item is string => Boolean(item))
      )
    ).sort((a, b) => a.localeCompare(b))

    if (requestId !== modelFetchRequestId) {
      return
    }

    availableModels.value = models
    if (models.length > 0) {
      if (!models.includes(modelInput.value.trim())) {
        modelInput.value = models[0]
      }
      if (!silent) {
        appStore.showSuccess(`已获取 ${models.length} 个模型`)
      }
      return
    }

    modelsError.value = '接口返回成功，但没有可选模型'
    if (!silent) {
      appStore.showError('没有读取到模型列表')
    }
  } catch (error) {
    if (requestId !== modelFetchRequestId) {
      return
    }

    modelsError.value = error instanceof Error ? error.message : '模型列表获取失败'
    if (!silent) {
      appStore.showError('模型列表获取失败')
    }
  } finally {
    if (requestId === modelFetchRequestId) {
      modelsLoading.value = false
    }
  }
}

async function autoFetchAvailableModels() {
  availableModels.value = []
  modelsError.value = ''

  if (!selectedApiKey.value) {
    return
  }

  await fetchAvailableModels({ silent: true })
}

watch(selectedCcsApp, (nextApp, prevApp) => {
  const nextDefault = ccswitchApps.find((item) => item.id === nextApp)?.defaultModel || ''
  const prevDefault = ccswitchApps.find((item) => item.id === prevApp)?.defaultModel || ''
  if (!modelInput.value.trim() || modelInput.value.trim() === prevDefault) {
    modelInput.value = nextDefault
  }
})

watch(routeOptions, (nextRoutes) => {
  if (!nextRoutes.some((item) => item.id === selectedRouteId.value) && nextRoutes.length > 0) {
    selectedRouteId.value = nextRoutes[0].id
  }
})

watch(selectedKeyId, () => {
  void autoFetchAvailableModels()
})

watch(selectedRouteId, (nextRouteId, prevRouteId) => {
  if (nextRouteId === prevRouteId) {
    return
  }

  if (!selectedApiKey.value) {
    availableModels.value = []
    modelsError.value = ''
    return
  }

  void autoFetchAvailableModels()
})

onMounted(() => {
  runtimeOrigin.value = window.location.origin
  loadPageData()
})
</script>

<style scoped>
.starter-hero {
  position: relative;
  overflow: hidden;
  border-radius: 28px;
  border: 1px solid rgba(255, 255, 255, 0.75);
  background:
    radial-gradient(circle at top left, rgba(196, 181, 253, 0.92), transparent 28%),
    radial-gradient(circle at top right, rgba(34, 211, 238, 0.28), transparent 26%),
    linear-gradient(135deg, rgba(255, 255, 255, 0.96) 0%, rgba(245, 243, 255, 0.9) 52%, rgba(239, 246, 255, 0.94) 100%);
  padding: 1.5rem;
  box-shadow: 0 30px 90px rgba(139, 92, 246, 0.14);
}

.starter-hero-glow,
.starter-panel-glow {
  position: absolute;
  border-radius: 9999px;
  filter: blur(42px);
  pointer-events: none;
}

.starter-hero-glow-a {
  left: -2rem;
  top: -2rem;
  height: 10rem;
  width: 10rem;
  background: rgba(168, 85, 247, 0.22);
}

.starter-hero-glow-b {
  right: -2rem;
  bottom: -2rem;
  height: 11rem;
  width: 11rem;
  background: rgba(59, 130, 246, 0.16);
}

.starter-badge,
.starter-chip,
.starter-runtime-pill,
.starter-command-pill {
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  font-weight: 700;
}

.starter-badge {
  gap: 0.5rem;
  border: 1px solid rgba(255, 255, 255, 0.7);
  background: rgba(255, 255, 255, 0.78);
  padding: 0.45rem 0.8rem;
  font-size: 0.72rem;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: rgb(91 33 182);
}

.starter-chip {
  border: 1px solid rgba(255, 255, 255, 0.7);
  background: rgba(255, 255, 255, 0.78);
  padding: 0.45rem 0.8rem;
  font-size: 0.72rem;
  color: rgb(91 33 182);
}

.starter-hero-btn,
.starter-action-btn,
.starter-command-btn,
.starter-link-btn,
.starter-fetch-btn,
.starter-inline-btn,
.starter-route-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  font-weight: 800;
  transition: all 0.2s ease;
}

.starter-hero-btn,
.starter-action-btn,
.starter-link-btn {
  padding: 0.8rem 1.1rem;
  font-size: 0.82rem;
}

.starter-hero-btn-primary,
.starter-action-btn-primary,
.starter-fetch-btn {
  background: linear-gradient(135deg, #8b5cf6 0%, #ec4899 100%);
  color: white;
  box-shadow: 0 14px 36px rgba(139, 92, 246, 0.24);
}

.starter-hero-btn-secondary,
.starter-action-btn-secondary,
.starter-link-btn-secondary {
  border: 1px solid rgba(203, 213, 225, 0.9);
  background: rgba(255, 255, 255, 0.82);
  color: rgb(51 65 85);
}

.starter-hero-btn-install-focus {
  position: relative;
  border: 1px solid rgba(168, 85, 247, 0.28);
  background: linear-gradient(135deg, #7c3aed 0%, #ec4899 55%, #f59e0b 100%);
  color: white;
  padding-inline: 1.4rem;
  box-shadow: 0 18px 42px rgba(168, 85, 247, 0.34);
  transform: translateY(-1px) scale(1.02);
}

.starter-hero-btn-install-focus:hover {
  box-shadow: 0 22px 52px rgba(168, 85, 247, 0.42);
  transform: translateY(-2px) scale(1.03);
}

.starter-stat-card,
.starter-card,
.starter-select-card,
.starter-panel,
.starter-config-card,
.starter-step-card,
.starter-bottom-card {
  border-radius: 24px;
  border: 1px solid rgba(255, 255, 255, 0.7);
  background: rgba(255, 255, 255, 0.82);
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.08);
}

.starter-stat-card {
  padding: 1rem;
}

.starter-stat-value {
  font-size: 1.6rem;
  font-weight: 900;
  color: rgb(15 23 42);
}

.starter-stat-label {
  margin-top: 0.35rem;
  font-size: 0.78rem;
  font-weight: 700;
  color: rgb(100 116 139);
}

.starter-card,
.starter-bottom-card {
  padding: 1.2rem;
}

.starter-card-ccswitch-focus {
  position: relative;
  overflow: hidden;
  border-color: rgba(14, 165, 233, 0.22);
  background:
    radial-gradient(circle at top right, rgba(34, 211, 238, 0.18), transparent 34%),
    radial-gradient(circle at bottom left, rgba(168, 85, 247, 0.14), transparent 30%),
    linear-gradient(135deg, rgba(240, 249, 255, 0.96), rgba(238, 242, 255, 0.94), rgba(255, 255, 255, 0.92));
  box-shadow: 0 22px 56px rgba(14, 165, 233, 0.14);
}

.starter-card-ccswitch-focus::before {
  content: '';
  position: absolute;
  inset: 0 auto 0 0;
  width: 5px;
  background: linear-gradient(180deg, rgb(14 165 233), rgb(168 85 247));
}

.starter-section-kicker {
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: rgb(139 92 246);
}

.starter-ccswitch-badge {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  border: 1px solid rgba(14, 165, 233, 0.18);
  background: rgba(14, 165, 233, 0.1);
  padding: 0.35rem 0.7rem;
  font-size: 0.68rem;
  font-weight: 900;
  letter-spacing: 0.16em;
  color: rgb(3 105 161);
  text-transform: uppercase;
}

.starter-select-card {
  width: 100%;
  padding: 1.2rem;
  text-align: left;
}

.starter-download-shell {
  position: relative;
  overflow: hidden;
  border-radius: 28px;
  border: 1px solid rgba(255, 255, 255, 0.72);
  background:
    radial-gradient(circle at top left, rgba(168, 85, 247, 0.1), transparent 22%),
    radial-gradient(circle at bottom right, rgba(34, 211, 238, 0.1), transparent 22%),
    rgba(255, 255, 255, 0.7);
  padding: 1.5rem;
  box-shadow: 0 24px 70px rgba(15, 23, 42, 0.08);
  backdrop-filter: blur(16px);
}

.starter-download-featured,
.starter-download-card {
  border-radius: 24px;
  border: 1px solid rgba(255, 255, 255, 0.7);
  background: rgba(255, 255, 255, 0.86);
  padding: 1.2rem;
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.08);
}

.starter-download-featured {
  background:
    radial-gradient(circle at top left, rgba(139, 92, 246, 0.12), transparent 24%),
    radial-gradient(circle at bottom right, rgba(34, 211, 238, 0.12), transparent 24%),
    rgba(255, 255, 255, 0.92);
}

.starter-download-card-compact {
  height: 100%;
}

.starter-download-kicker {
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: rgb(100 116 139);
}

.starter-download-version {
  display: inline-flex;
  min-width: 5rem;
  flex-direction: column;
  align-items: flex-end;
  border-radius: 18px;
  border: 1px solid rgba(226, 232, 240, 0.9);
  background: rgba(255, 255, 255, 0.82);
  padding: 0.75rem 0.85rem;
  font-size: 0.72rem;
  font-weight: 800;
  color: rgb(100 116 139);
}

.starter-download-platform-card {
  border-radius: 20px;
  border: 1px solid rgba(226, 232, 240, 0.9);
  background: rgba(255, 255, 255, 0.78);
  padding: 1rem;
}

.starter-download-focus-box {
  border-radius: 20px;
  border: 1px solid rgba(226, 232, 240, 0.9);
  background: rgba(255, 255, 255, 0.78);
  padding: 1rem;
}

.starter-download-group {
  border-radius: 18px;
  border: 1px solid rgba(226, 232, 240, 0.85);
  background: rgba(248, 250, 252, 0.85);
  padding: 0.85rem;
}

.starter-download-group-title {
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgb(100 116 139);
}

.starter-download-link {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  border-radius: 9999px;
  border: 1px solid rgba(203, 213, 225, 0.9);
  background: rgba(255, 255, 255, 0.9);
  padding: 0.5rem 0.8rem;
  font-size: 0.78rem;
  font-weight: 700;
  color: rgb(51 65 85);
  transition: all 0.2s ease;
}

.starter-download-link:hover {
  border-color: rgba(139, 92, 246, 0.4);
  transform: translateY(-1px);
  color: rgb(109 40 217);
}

.starter-download-link-recommended {
  border-color: rgba(139, 92, 246, 0.35);
  background: rgba(139, 92, 246, 0.08);
}

.starter-download-recommended {
  border-radius: 9999px;
  background: rgba(139, 92, 246, 0.16);
  padding: 0.15rem 0.4rem;
  font-size: 0.65rem;
  font-weight: 900;
  color: rgb(109 40 217);
}

.starter-download-dot {
  height: 0.25rem;
  width: 0.25rem;
  flex-shrink: 0;
  border-radius: 9999px;
  background: rgba(139, 92, 246, 0.45);
}

.starter-select-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 24px 60px rgba(15, 23, 42, 0.1);
}

.starter-select-card-active {
  border-color: rgba(139, 92, 246, 0.5);
  box-shadow: 0 22px 70px rgba(139, 92, 246, 0.18);
}

.starter-status-chip {
  border-radius: 9999px;
  background: rgba(139, 92, 246, 0.12);
  padding: 0.35rem 0.75rem;
  font-size: 0.72rem;
  font-weight: 800;
  color: rgb(109 40 217);
}

.starter-mini-chip {
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  background: rgb(241 245 249);
  padding: 0.35rem 0.7rem;
  font-size: 0.72rem;
  font-weight: 700;
  color: rgb(71 85 105);
}

.starter-tab-btn {
  width: 100%;
  border-radius: 18px;
  border: 1px solid rgba(226, 232, 240, 0.9);
  background: rgba(255, 255, 255, 0.72);
  padding: 0.9rem 1rem;
  text-align: left;
}

.starter-tab-btn-active {
  border-color: rgba(139, 92, 246, 0.45);
  background: rgba(139, 92, 246, 0.08);
}

.starter-info-row {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
}

.starter-panel {
  position: relative;
  overflow: hidden;
  padding: 1.5rem;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.92) 0%, rgba(248, 250, 252, 0.84) 100%);
}

.starter-panel-glow-a {
  left: -3rem;
  top: 30%;
  height: 10rem;
  width: 10rem;
  background: rgba(168, 85, 247, 0.18);
}

.starter-panel-glow-b {
  right: -2rem;
  top: -1.5rem;
  height: 9rem;
  width: 9rem;
  background: rgba(14, 165, 233, 0.14);
}

.starter-runtime-pill {
  border: 1px solid rgba(226, 232, 240, 0.9);
  background: rgba(255, 255, 255, 0.75);
  padding: 0.4rem 0.8rem;
  font-size: 0.72rem;
  color: rgb(71 85 105);
}

.starter-field {
  display: block;
}

.starter-field-label {
  display: block;
  margin-bottom: 0.55rem;
  font-size: 0.75rem;
  font-weight: 800;
  color: rgb(71 85 105);
}

.starter-input {
  width: 100%;
  border-radius: 16px;
  border: 1px solid rgba(203, 213, 225, 0.95);
  background: rgba(255, 255, 255, 0.94);
  padding: 0.9rem 1rem;
  font-size: 0.95rem;
  color: rgb(15 23 42);
  outline: none;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.starter-input:focus {
  border-color: rgba(139, 92, 246, 0.65);
  box-shadow: 0 0 0 4px rgba(167, 139, 250, 0.18);
}

.starter-route-chip {
  gap: 0.5rem;
  padding: 0.65rem 0.95rem;
  border: 1px solid rgba(203, 213, 225, 0.95);
  background: rgba(255, 255, 255, 0.88);
  color: rgb(51 65 85);
}

.starter-route-chip-active {
  border-color: rgba(139, 92, 246, 0.45);
  background: rgba(139, 92, 246, 0.08);
  color: rgb(109 40 217);
}

.starter-route-chip-note {
  font-size: 0.68rem;
  opacity: 0.7;
  text-transform: uppercase;
}

.starter-fetch-btn {
  padding: 0.8rem 1.15rem;
}

.starter-fetch-btn:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.starter-fetch-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.65rem;
  font-size: 0.83rem;
  font-weight: 700;
  color: rgb(100 116 139);
}

.starter-config-card,
.starter-step-card {
  padding: 1rem;
}

.starter-inline-btn {
  margin-top: 0.85rem;
  border: 1px solid rgba(203, 213, 225, 0.9);
  background: rgba(255, 255, 255, 0.82);
  padding: 0.45rem 0.8rem;
  font-size: 0.72rem;
  color: rgb(51 65 85);
}

.starter-step-index {
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgb(168 85 247);
}

.starter-step-title {
  margin-top: 0.65rem;
  font-size: 1rem;
  font-weight: 800;
  color: rgb(15 23 42);
}

.starter-step-desc {
  margin-top: 0.5rem;
  font-size: 0.86rem;
  line-height: 1.7;
  color: rgb(100 116 139);
}

.starter-command-btn {
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.08);
  padding: 0.55rem 0.95rem;
  font-size: 0.78rem;
  color: white;
}

.starter-command-btn:hover {
  background: rgba(255, 255, 255, 0.16);
}

.starter-command-pre {
  overflow-x: auto;
  padding: 1.15rem 1rem 1.35rem;
  font-size: 0.86rem;
  line-height: 1.8;
  color: rgb(226 232 240);
  white-space: pre-wrap;
  word-break: break-word;
}

.starter-command-pill {
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.05);
  padding: 0.38rem 0.75rem;
  font-size: 0.72rem;
  color: rgb(203 213 225);
}

.starter-link-btn-primary {
  background: rgb(15 23 42);
  color: white;
}

.starter-bottom-card {
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
  font-size: 0.95rem;
  font-weight: 500;
  color: rgb(71 85 105);
}

.dark .starter-hero {
  border-color: rgba(255, 255, 255, 0.1);
  background:
    radial-gradient(circle at top left, rgba(124, 58, 237, 0.34), transparent 28%),
    radial-gradient(circle at top right, rgba(34, 211, 238, 0.16), transparent 24%),
    linear-gradient(135deg, rgba(10, 14, 25, 0.95) 0%, rgba(21, 25, 45, 0.92) 55%, rgba(7, 11, 21, 0.95) 100%);
}

.dark .starter-badge,
.dark .starter-chip,
.dark .starter-stat-card,
.dark .starter-card,
.dark .starter-download-shell,
.dark .starter-download-featured,
.dark .starter-download-card,
.dark .starter-select-card,
.dark .starter-panel,
.dark .starter-runtime-pill,
.dark .starter-config-card,
.dark .starter-step-card,
.dark .starter-bottom-card {
  border-color: rgba(255, 255, 255, 0.08);
  background: rgba(10, 14, 25, 0.72);
}

.dark .starter-card-ccswitch-focus {
  border-color: rgba(34, 211, 238, 0.18);
  background:
    radial-gradient(circle at top right, rgba(34, 211, 238, 0.18), transparent 34%),
    radial-gradient(circle at bottom left, rgba(168, 85, 247, 0.16), transparent 30%),
    linear-gradient(135deg, rgba(8, 23, 38, 0.92), rgba(26, 25, 54, 0.9), rgba(10, 14, 25, 0.9));
  box-shadow: 0 24px 60px rgba(8, 145, 178, 0.16);
}

.dark .starter-ccswitch-badge {
  border-color: rgba(34, 211, 238, 0.2);
  background: rgba(34, 211, 238, 0.14);
  color: rgb(165 243 252);
}

.dark .starter-mini-chip {
  background: rgba(30, 41, 59, 0.75);
  color: rgb(226 232 240);
}

.dark .starter-stat-value,
.dark .starter-step-title {
  color: white;
}

.dark .starter-stat-label,
.dark .starter-field-label,
.dark .starter-step-desc,
.dark .starter-fetch-meta,
.dark .starter-info-row,
.dark .starter-bottom-card {
  color: rgb(148 163 184);
}

.dark .starter-tab-btn,
.dark .starter-input,
.dark .starter-route-chip,
.dark .starter-inline-btn,
.dark .starter-download-version,
.dark .starter-download-focus-box,
.dark .starter-download-platform-card,
.dark .starter-download-group,
.dark .starter-download-link,
.dark .starter-hero-btn-secondary,
.dark .starter-action-btn-secondary,
.dark .starter-link-btn-secondary {
  border-color: rgba(255, 255, 255, 0.08);
  background: rgba(2, 6, 23, 0.7);
  color: rgb(248 250 252);
}

.dark .starter-hero-btn-install-focus {
  border-color: rgba(244, 114, 182, 0.34);
  background: linear-gradient(135deg, #8b5cf6 0%, #ec4899 58%, #f97316 100%);
  color: white;
  box-shadow: 0 22px 56px rgba(236, 72, 153, 0.24);
}

.dark .starter-download-recommended {
  background: rgba(139, 92, 246, 0.24);
  color: rgb(233 213 255);
}
</style>
