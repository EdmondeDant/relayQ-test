<template>
  <AppLayout>
    <div class="calculator-page">
      <div class="hero">
        <div>
          <div class="eyebrow"><span></span> Leonardo API 费用预估</div>
          <h1>图片与视频价格计算器</h1>
          <p>选择模型和生成参数，快速预估单次及批量 API 调用成本。</p>
        </div>
        <div class="snapshot"><strong>{{ modelTotal }}</strong><span>个模型价格已收录</span></div>
      </div>

      <section class="calculator-grid">
        <div class="panel model-panel">
          <div class="panel-head"><span class="step">1</span><div><strong>选择模型</strong><small>Select Model</small></div></div>
          <div class="tabs" role="tablist">
            <button v-for="item in types" :key="item.value" type="button" :class="{ active: type === item.value }" @click="selectType(item.value)">
              <svg v-if="item.value === 'image'" viewBox="0 0 24 24"><path d="M4 5h16v14H4zM4 16l4-4 3 3 2-2 7 5M15.5 9.5h.01" /></svg>
              <svg v-else viewBox="0 0 24 24"><path d="M4 6h12v12H4zM16 10l4-2v8l-4-2z" /></svg>
              {{ item.label }}<span>{{ availableModelNames(item.value).length }}</span>
            </button>
          </div>
          <div class="search-box"><svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="7"/><path d="m16 16 4 4"/></svg><input v-model="search" type="search" placeholder="搜索模型" /></div>
          <div class="models">
            <template v-for="(name, index) in filteredModels" :key="name">
              <div v-if="index === 0 || index === featuredCount" class="model-group">{{ index === 0 ? '✦ 热门模型' : '全部模型' }}</div>
              <button type="button" class="model" :class="{ active: model === name }" @click="selectModel(name)">
                <span class="model-mark" :class="type">{{ modelInitials(name) }}</span>
                <span class="model-copy"><b>{{ name }}</b><small>{{ type === 'image' ? imageDescription(name) : videoDescription(name) }}</small></span>
                <span class="check">✓</span>
              </button>
            </template>
            <div v-if="filteredModels.length === 0" class="empty">当前 Leonardo 分组没有可显示的模型</div>
          </div>
        </div>

        <div class="panel settings-panel">
          <div class="panel-head"><span class="step">2</span><div><strong>生成设置</strong><small>Settings</small></div></div>
          <div class="settings">
            <div v-if="model" class="selected-model"><span class="model-mark" :class="type">{{ modelInitials(model) }}</span><div><small>当前模型</small><strong>{{ model }}</strong></div></div>

            <div v-if="visibleValues(current.qualities)" class="setting-group">
              <div class="setting-title"><strong>生成质量</strong><span>Quality</span></div>
              <div class="choice-grid">
                <button v-for="value in current.qualities" :key="String(value)" type="button" :class="{ active: state.quality === value }" @click="state.quality = value">{{ value }}</button>
              </div>
            </div>

            <div v-if="visibleValues(current.durations)" class="setting-group">
              <div class="setting-title"><strong>视频时长</strong><span>{{ state.duration }} 秒</span></div>
              <input v-if="current.slider" v-model.number="state.duration" class="range" type="range" :min="current.durations?.[0] ?? 1" :max="current.durations?.[current.durations.length - 1] ?? 1" step="1" />
              <div v-else class="choice-grid duration-grid">
                <button v-for="value in current.durations" :key="String(value)" type="button" :class="{ active: state.duration === value }" @click="state.duration = value">{{ value }}s</button>
              </div>
              <div v-if="current.slider" class="range-ends"><span>{{ current.durations?.[0] }}s</span><span>{{ current.durations?.[current.durations.length - 1] }}s</span></div>
            </div>

            <div v-if="visibleValues(current.ratios)" class="setting-group">
              <div class="setting-title"><strong>画面比例</strong><span>Aspect Ratio</span></div>
              <div class="ratio-grid">
                <button v-for="value in current.ratios" :key="String(value)" type="button" :class="{ active: state.ratio === value }" @click="state.ratio = value">
                  <i :style="ratioStyle(String(value))"></i><span>{{ value }}</span>
                </button>
              </div>
            </div>

            <div v-if="visibleValues(type === 'image' ? current.sizes : current.resolutions)" class="setting-group">
              <div class="setting-title"><strong>{{ type === 'image' ? '图片尺寸' : '视频分辨率' }}</strong><span>{{ type === 'image' ? 'Dimensions' : 'Resolution' }}</span></div>
              <div class="size-grid">
                <button v-for="value in type === 'image' ? current.sizes : current.resolutions" :key="String(value)" type="button" :class="{ active: (type === 'image' ? state.size : state.resolution) === value }" @click="selectSize(value)">
                  <strong>{{ splitOption(value)[0] }}</strong><small>{{ splitOption(value)[1] }}</small>
                </button>
              </div>
            </div>

            <div v-if="!hasSettings" class="no-settings"><span>✓</span><strong>无需额外设置</strong><small>该模型采用默认生成参数，可直接计算费用</small></div>
          </div>
        </div>

        <div class="result-column">
          <div class="panel result-panel">
            <div class="panel-head"><span class="step">3</span><div><strong>费用预估</strong><small>Calculate Cost</small></div><button class="reset" type="button" @click="reset">↻ 重置</button></div>
            <div class="summary">
              <div class="estimate-label"><span>预估费用</span><small>{{ model }}</small></div>
              <div class="total-cny"><small>¥</small>{{ totalCny.toFixed(2) }}</div>
              <div class="total-usd">约 ${{ totalUsd.toFixed(2) }} USD</div>
            </div>
          </div>
          <div class="info-card"><span>i</span><div><strong>价格说明</strong><p>结果为 API 调用费用预估，实际账单以生成时采用的模型参数及平台结算为准。</p></div></div>
        </div>
      </section>
      <div class="footer-note">仅展示当前 Leonardo 分组白名单内且已有价格的模型</div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { PRICING, type PricingModel } from '@/data/leonardoPricing'
import { userChannelsAPI } from '@/api/channels'

type MediaType = 'image' | 'video'
type ChoiceValue = string | number | null

const types: Array<{ value: MediaType; label: string }> = [{ value: 'image', label: 'Image' }, { value: 'video', label: 'Video' }]
const type = ref<MediaType>('image')
const model = ref('')
const search = ref('')
const exchangeRate = 7.1
const leonardoModels = ref<Set<string>>(new Set())
const state = reactive<Record<'quality' | 'size' | 'ratio' | 'duration' | 'resolution', ChoiceValue>>({ quality: null, size: null, ratio: null, duration: null, resolution: null })
const normalizeModelName = (name: string) => name.toLowerCase().replace(/[^a-z0-9]/g, '')
const availableModelNames = (mediaType: MediaType) => Object.keys(PRICING[mediaType]).filter(name => leonardoModels.value.has(normalizeModelName(name)))
const modelNames = computed(() => availableModelNames(type.value))
const modelTotal = computed(() => availableModelNames('image').length + availableModelNames('video').length)
const filteredModels = computed(() => modelNames.value.filter(name => name.toLowerCase().includes(search.value.trim().toLowerCase())))
const featuredCount = computed(() => Math.min(type.value === 'image' ? 9 : 8, filteredModels.value.length))
const current = computed<PricingModel>(() => PRICING[type.value][model.value] ?? { configs: [] })
const hasSettings = computed(() => [current.value.qualities, current.value.sizes, current.value.ratios, current.value.durations, current.value.resolutions].some(values => values && (values.length > 1 || values[0] !== null)))
const unitPrice = computed(() => {
  const data = current.value
  if (type.value === 'image') return data.configs.find(item => item.quality === state.quality && item.size === state.size)?.cost ?? 0
  if (data.slider) {
    const config = data.configs.find(item => item.resolution === state.resolution) ?? data.configs[0]
    const duration = Number(state.duration)
    if (config.max === config.min) return config.minCost ?? 0
    return (config.minCost ?? 0) + (duration - (config.min ?? 0)) * ((config.maxCost ?? 0) - (config.minCost ?? 0)) / ((config.max ?? 0) - (config.min ?? 0))
  }
  return data.configs.find(item => item.duration === state.duration && item.resolution === state.resolution)?.cost ?? 0
})
const totalUsd = computed(() => unitPrice.value)
const totalCny = computed(() => totalUsd.value * exchangeRate)

function resetState() {
  const data = current.value
  state.quality = data.qualities?.[0] ?? null
  state.size = data.sizes?.[0] ?? null
  state.ratio = data.ratios?.[0] ?? null
  state.duration = data.durations?.[0] ?? null
  state.resolution = data.resolutions?.[0] ?? null
}

function selectType(value: MediaType) {
  type.value = value
  search.value = ''
  model.value = availableModelNames(value)[0] ?? ''
  resetState()
}

function visibleValues(values?: ChoiceValue[]) {
  return Boolean(values?.length && (values.length > 1 || values[0] !== null))
}

function selectSize(value: ChoiceValue) {
  if (type.value === 'image') state.size = value
  else state.resolution = value
}

function splitOption(value: ChoiceValue) {
  const parts = String(value ?? 'Default').split(/\s(?=\d)/)
  return [parts[0], parts.slice(1).join(' ')]
}

function ratioStyle(value: string) {
  const [width, height] = value.split(':').map(Number)
  const max = 25
  return { width: `${width >= height ? max : max * width / height}px`, height: `${height >= width ? max : max * height / width}px` }
}

function modelInitials(name: string) {
  return name.split(/[ .-]/).filter(Boolean).slice(0, 2).map(word => word[0]).join('').toUpperCase()
}

function imageDescription(name: string) {
  if (name.includes('GPT')) return '高质量文字与视觉生成'
  if (name.includes('Nano Banana')) return '快速灵活的图像生成与编辑'
  if (name.includes('FLUX')) return '细节丰富的专业图像模型'
  return '创意图片生成模型'
}

function videoDescription(name: string) {
  if (name.includes('Veo')) return '高质量电影级视频生成'
  if (name.includes('Kling')) return '流畅真实的视频生成模型'
  if (name.includes('Seedance')) return '多规格专业视频生成'
  return 'AI 视频生成模型'
}

function reset() {
  resetState()
}

function selectModel(value: string) {
  model.value = value
  resetState()
}

onMounted(async () => {
  try {
    const channels = await userChannelsAPI.getAvailable()
    leonardoModels.value = new Set(channels.flatMap(channel => channel.platforms)
      .filter(section => section.platform === 'leonardo')
      .flatMap(section => section.supported_models)
      .map(item => normalizeModelName(item.name)))
  } catch {
    leonardoModels.value = new Set()
  }
  model.value = modelNames.value[0] ?? ''
  resetState()
})
</script>

<style scoped>
.calculator-page { --accent: #7867f2; --accent-soft: rgba(120,103,242,.14); --panel: #fff; --surface: #f6f7fb; --line: #e5e7ef; --text: #181923; --muted: #737689; max-width: 1280px; margin: 0 auto; color: var(--text); }
.dark .calculator-page { --panel: #17171b; --surface: #212126; --line: #34343c; --text: #f7f7fb; --muted: #999baa; }
.hero { display: flex; align-items: flex-end; justify-content: space-between; gap: 24px; margin-bottom: 24px; }
.eyebrow { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; color: var(--accent); font-size: 12px; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; }
.eyebrow span { width: 7px; height: 7px; border-radius: 50%; background: var(--accent); box-shadow: 0 0 0 5px var(--accent-soft); }
h1 { margin: 0; font-size: 28px; font-weight: 750; letter-spacing: -.03em; }
.hero p { margin: 8px 0 0; color: var(--muted); }
.snapshot { display: flex; align-items: center; gap: 10px; border: 1px solid var(--line); border-radius: 12px; background: var(--panel); padding: 10px 14px; }
.snapshot strong { color: var(--accent); font-size: 22px; }.snapshot span { max-width: 76px; color: var(--muted); font-size: 11px; line-height: 1.3; }
.calculator-grid { display: grid; grid-template-columns: minmax(280px, 1fr) minmax(320px, 1.08fr) minmax(300px, .92fr); align-items: start; gap: 14px; }
.panel { overflow: hidden; border: 1px solid var(--line); border-radius: 18px; background: var(--panel); box-shadow: 0 10px 35px rgba(25,25,40,.05); }
.panel-head { display: flex; align-items: center; gap: 10px; min-height: 66px; border-bottom: 1px solid var(--line); padding: 12px 16px; }
.panel-head .step { display: grid; width: 30px; height: 30px; place-items: center; border-radius: 9px; background: var(--accent-soft); color: var(--accent); font-weight: 800; }
.panel-head div { display: flex; flex-direction: column; }.panel-head strong { font-size: 14px; }.panel-head small { margin-top: 2px; color: var(--muted); font-size: 11px; }
.model-panel,.settings-panel { height: 650px; }.tabs { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; padding: 12px 14px 8px; }
.tabs button { display: flex; align-items: center; justify-content: center; gap: 7px; border: 1px solid transparent; border-radius: 10px; padding: 10px; color: var(--muted); font-weight: 650; }
.tabs button.active { border-color: rgba(120,103,242,.4); background: var(--accent-soft); color: var(--accent); }.tabs svg,.search-box svg { width: 17px; fill: none; stroke: currentColor; stroke-width: 1.8; }.tabs span { border-radius: 10px; background: var(--surface); padding: 1px 6px; font-size: 10px; }
.search-box { display: flex; align-items: center; gap: 8px; margin: 2px 14px 7px; border: 1px solid var(--line); border-radius: 10px; background: var(--surface); padding: 9px 11px; color: var(--muted); }.search-box input { width: 100%; border: 0; outline: 0; background: transparent; color: var(--text); font-size: 13px; }
.models { height: 505px; overflow: auto; padding: 0 10px 14px 14px; scrollbar-width: thin; scrollbar-color: var(--line) transparent; }.model-group { padding: 10px 2px 7px; color: var(--muted); font-size: 11px; font-weight: 700; text-transform: uppercase; }
.model { display: flex; align-items: center; gap: 11px; width: 100%; margin-bottom: 7px; border: 1px solid transparent; border-radius: 13px; background: var(--surface); padding: 10px; text-align: left; transition: .18s ease; }.model:hover { transform: translateY(-1px); border-color: var(--line); }.model.active { border-color: var(--accent); background: var(--accent-soft); }
.model-mark { display: grid; flex: 0 0 42px; height: 42px; place-items: center; border-radius: 12px; background: linear-gradient(145deg,#7968f3,#b56ef4); color: #fff; font-size: 12px; font-weight: 800; box-shadow: inset 0 1px rgba(255,255,255,.28); }.model-mark.video { background: linear-gradient(145deg,#0ea5a8,#3b82f6); }.model-copy { min-width: 0; flex: 1; }.model-copy b,.model-copy small { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }.model-copy b { font-size: 13px; }.model-copy small { margin-top: 4px; color: var(--muted); font-size: 10px; }.check { display: none; width: 20px; height: 20px; border-radius: 50%; background: var(--accent); color: #fff; font-size: 11px; text-align: center; line-height: 20px; }.model.active .check { display: block; }.empty { padding: 50px 10px; color: var(--muted); text-align: center; }
.settings { height: 583px; overflow: auto; padding: 15px; }.selected-model { display: flex; align-items: center; gap: 10px; border: 1px solid var(--line); border-radius: 13px; background: var(--surface); padding: 10px; }.selected-model div { display: flex; min-width: 0; flex-direction: column; }.selected-model small { color: var(--muted); font-size: 10px; }.selected-model strong { overflow: hidden; margin-top: 2px; font-size: 13px; text-overflow: ellipsis; white-space: nowrap; }
.setting-group { margin-top: 22px; }.setting-title { display: flex; align-items: center; justify-content: space-between; margin-bottom: 9px; }.setting-title strong { font-size: 13px; }.setting-title span { color: var(--muted); font-size: 11px; }.choice-grid,.size-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 7px; }.choice-grid button,.size-grid button { min-height: 42px; border: 1px solid transparent; border-radius: 9px; background: var(--surface); color: var(--text); font-size: 12px; }.choice-grid button.active,.size-grid button.active,.ratio-grid button.active { border-color: var(--accent); background: var(--accent-soft); color: var(--accent); }.duration-grid { grid-template-columns: repeat(auto-fit,minmax(54px,1fr)); }
.ratio-grid { display: grid; grid-template-columns: repeat(auto-fit,minmax(58px,1fr)); gap: 7px; }.ratio-grid button { display: flex; min-height: 73px; align-items: center; justify-content: center; flex-direction: column; gap: 8px; border: 1px solid transparent; border-radius: 9px; background: var(--surface); color: var(--muted); font-size: 11px; }.ratio-grid i { display: block; border: 1.5px solid currentColor; border-radius: 2px; }
.size-grid button { display: flex; min-height: 52px; justify-content: center; flex-direction: column; }.size-grid strong { font-size: 11px; }.size-grid small { margin-top: 3px; color: var(--muted); font-size: 9px; }.range { width: 100%; accent-color: var(--accent); }.range-ends { display: flex; justify-content: space-between; color: var(--muted); font-size: 10px; }.no-settings { display: flex; align-items: center; flex-direction: column; margin-top: 24px; border: 1px dashed var(--line); border-radius: 14px; padding: 32px 16px; text-align: center; }.no-settings span { display: grid; width: 38px; height: 38px; place-items: center; border-radius: 50%; background: var(--accent-soft); color: var(--accent); }.no-settings strong { margin-top: 12px; font-size: 13px; }.no-settings small { margin-top: 5px; color: var(--muted); font-size: 11px; }
.result-column { display: grid; gap: 14px; }.reset { margin-left: auto; border-radius: 8px; background: var(--surface); padding: 7px 9px; color: var(--muted); font-size: 11px; }.summary { padding: 16px; }.estimate-label { display: flex; align-items: flex-start; justify-content: space-between; }.estimate-label span { font-size: 15px; font-weight: 750; }.estimate-label small { max-width: 120px; color: var(--muted); font-size: 9px; text-align: right; }.total-cny { margin-top: 12px; color: var(--accent); font-size: 38px; font-weight: 800; letter-spacing: -.04em; }.total-cny small { margin-right: 3px; font-size: 20px; }.total-usd { margin-top: 2px; color: var(--muted); font-size: 12px; }
.info-card { display: flex; gap: 11px; border: 1px solid rgba(120,103,242,.28); border-radius: 15px; background: var(--accent-soft); padding: 14px; }.info-card > span { display: grid; flex: 0 0 24px; height: 24px; place-items: center; border-radius: 50%; background: var(--accent); color: #fff; font: 700 12px serif; }.info-card strong { font-size: 12px; }.info-card p { margin: 5px 0 0; color: var(--muted); font-size: 10px; line-height: 1.55; }.footer-note { padding: 17px 0 2px; color: var(--muted); font-size: 10px; text-align: center; }
@media (max-width: 1180px) { .calculator-grid { grid-template-columns: 1fr 1fr; }.result-column { grid-column: 1 / -1; grid-template-columns: 1fr 1fr; }.model-panel,.settings-panel { height: 610px; }.models { height: 465px; } }
@media (max-width: 760px) { .hero { align-items: flex-start; flex-direction: column; }.snapshot { display: none; }.calculator-grid { grid-template-columns: 1fr; }.result-column { grid-column: auto; grid-template-columns: 1fr; }.model-panel,.settings-panel { height: auto; }.models,.settings { height: 480px; }.calculator-page h1 { font-size: 24px; } }
</style>
