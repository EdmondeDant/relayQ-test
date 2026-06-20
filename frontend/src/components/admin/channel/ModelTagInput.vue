<template>
  <div>
    <!-- Tags display -->
    <div class="flex flex-wrap gap-1.5 rounded-lg border border-gray-200 bg-white p-2 dark:border-dark-600 dark:bg-dark-800 min-h-[2.5rem]">
      <span
        v-for="(model, idx) in models"
        :key="idx"
        class="inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-sm"
        :class="getPlatformTagClass(props.platform || '')"
      >
        {{ model }}
        <button
          type="button"
          @click="removeModel(idx)"
          class="ml-0.5 rounded-full p-0.5 hover:bg-primary-200 dark:hover:bg-primary-800"
        >
          <Icon name="x" size="xs" />
        </button>
      </span>
      <div class="relative flex-1 min-w-[120px]">
        <input
          ref="inputRef"
          v-model="inputValue"
          type="text"
          class="w-full border-none bg-transparent text-sm outline-none placeholder:text-gray-400 dark:text-white"
          :placeholder="models.length === 0 ? placeholder : ''"
          @focus="showSuggestions = true"
          @input="showSuggestions = true"
          @keydown.enter.prevent="addModel"
          @keydown.tab.prevent="addModel"
          @keydown.delete="handleBackspace"
          @paste="handlePaste"
          @blur="handleBlur"
        />
        <div
          v-if="showSuggestions && availableOptions.length > 0"
          class="absolute left-0 right-0 z-50 mt-2 max-h-56 overflow-auto rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-600 dark:bg-dark-800"
        >
          <button
            v-for="option in availableOptions"
            :key="option"
            type="button"
            class="block w-full px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-dark-700"
            @mousedown.prevent="selectModel(option)"
          >
            {{ option }}
          </button>
        </div>
      </div>
    </div>
    <p class="mt-1 text-xs text-gray-400">
      {{ t('admin.channels.form.modelInputHint', 'Press Enter to add, supports paste for batch import.') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { getPlatformTagClass } from './types'

const { t } = useI18n()

const props = defineProps<{
  models: string[]
  placeholder?: string
  platform?: string
  modelOptions?: string[]
}>()

const emit = defineEmits<{
  'update:models': [models: string[]]
}>()

const inputValue = ref('')
const inputRef = ref<HTMLInputElement>()
const showSuggestions = ref(false)

const availableOptions = computed(() => {
  const keyword = inputValue.value.trim().toLowerCase()
  return [...new Set(props.modelOptions || [])]
    .filter(option => !props.models.includes(option))
    .filter(option => !keyword || option.toLowerCase().includes(keyword))
    .slice(0, 50)
})

function addModel() {
  const val = inputValue.value.trim()
  if (!val) return
  if (!props.models.includes(val)) {
    emit('update:models', [...props.models, val])
  }
  inputValue.value = ''
  showSuggestions.value = false
}

function selectModel(model: string) {
  if (!props.models.includes(model)) {
    emit('update:models', [...props.models, model])
  }
  inputValue.value = ''
  showSuggestions.value = false
  inputRef.value?.focus()
}

function handleBlur() {
  setTimeout(() => {
    showSuggestions.value = false
    addModel()
  }, 120)
}

function removeModel(idx: number) {
  const newModels = [...props.models]
  newModels.splice(idx, 1)
  emit('update:models', newModels)
}

function handleBackspace() {
  if (inputValue.value === '' && props.models.length > 0) {
    removeModel(props.models.length - 1)
  }
}

function handlePaste(e: ClipboardEvent) {
  e.preventDefault()
  const text = e.clipboardData?.getData('text') || ''
  const items = text.split(/[,\n;]+/).map(s => s.trim()).filter(Boolean)
  if (items.length === 0) return
  const unique = [...new Set([...props.models, ...items])]
  emit('update:models', unique)
  inputValue.value = ''
  showSuggestions.value = false
}
</script>
