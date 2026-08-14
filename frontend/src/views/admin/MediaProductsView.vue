<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <div><h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('admin.mediaProducts.title') }}</h1><p class="mt-1 text-sm text-gray-500">{{ t('admin.mediaProducts.description') }}</p></div>
        <button class="btn btn-primary" @click="openCreate">{{ t('admin.mediaProducts.create') }}</button>
      </div>
      <div v-if="error" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">{{ error }}</div>
      <div class="overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <table class="w-full text-left text-sm">
          <thead class="bg-gray-50 text-gray-500 dark:bg-dark-700"><tr><th class="p-4">{{ t('admin.mediaProducts.publicModel') }}</th><th class="p-4">{{ t('admin.mediaProducts.modality') }}</th><th class="p-4">{{ t('common.status') }}</th><th class="p-4">{{ t('admin.mediaProducts.bindings') }}</th><th class="p-4">{{ t('admin.mediaProducts.prices') }}</th><th class="p-4">Offer</th><th class="p-4">{{ t('common.actions') }}</th></tr></thead>
          <tbody><tr v-for="product in products" :key="product.id" class="border-t border-gray-100 dark:border-dark-700"><td class="p-4 font-medium">{{ product.public_model }}</td><td class="p-4">{{ product.modality }}</td><td class="p-4"><span :class="product.enabled ? 'text-green-600' : 'text-gray-400'">{{ product.enabled ? t('common.enabled') : t('common.disabled') }}</span></td><td class="p-4">{{ product.group_ids.length }}</td><td class="p-4">{{ product.prices.length }}</td><td class="p-4">{{ product.offers.length }}</td><td class="p-4"><button class="mr-3 text-primary-600" @click="openEdit(product)">{{ t('common.edit') }}</button><button class="text-red-600" @click="disable(product)">{{ t('common.disable') }}</button></td></tr></tbody>
        </table>
        <div v-if="!loading && products.length === 0" class="p-10 text-center text-gray-500">{{ t('admin.mediaProducts.empty') }}</div>
      </div>
    </div>
    <BaseDialog :show="showDialog" :title="editing ? t('admin.mediaProducts.edit') : t('admin.mediaProducts.create')" width="full" @close="showDialog = false">
      <form class="space-y-6" @submit.prevent="save">
        <div class="grid gap-4 md:grid-cols-4">
          <label class="input-label md:col-span-2">{{ t('admin.mediaProducts.publicModel') }}<input v-model="form.public_model" required class="input mt-1" /></label>
          <label class="input-label">{{ t('admin.mediaProducts.modality') }}<select v-model="form.modality" class="input mt-1"><option value="image">image</option><option value="video">video</option></select></label>
          <label class="mt-7 flex items-center gap-2"><input v-model="form.enabled" type="checkbox" />{{ t('common.enabled') }}</label>
        </div>
        <label class="input-label">{{ t('admin.mediaProducts.productDescription') }}<textarea v-model="form.description" class="input mt-1" rows="2" /></label>
        <fieldset><legend class="input-label">{{ t('admin.mediaProducts.entryGroups') }}</legend><div class="mt-2 flex flex-wrap gap-3"><label v-for="group in entryGroups" :key="group.id" class="flex items-center gap-2 rounded border border-gray-200 px-3 py-2 dark:border-dark-600"><input v-model="form.group_ids" type="checkbox" :value="group.id" />{{ group.name }}</label></div></fieldset>
        <fieldset><legend class="mb-2 font-semibold">{{ t('admin.mediaProducts.prices') }}</legend><ProductPriceEditor v-model="form.prices" :modality="form.modality" /></fieldset>
        <fieldset><legend class="mb-2 font-semibold">Offer</legend><OfferEditor v-model="form.offers" :groups="groups" @error="error = $event" /></fieldset>
        <div v-if="validationError" class="rounded-lg bg-red-50 p-3 text-sm text-red-700">{{ validationError }}</div>
        <div class="flex justify-end gap-3"><button type="button" class="btn btn-secondary" @click="showDialog = false">{{ t('common.cancel') }}</button><button class="btn btn-primary" :disabled="saving">{{ t('common.save') }}</button></div>
      </form>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ProductPriceEditor from '@/components/admin/media-product/ProductPriceEditor.vue'
import OfferEditor from '@/components/admin/media-product/OfferEditor.vue'
import mediaProductsAPI, { type MediaProduct } from '@/api/admin/mediaProducts'
import groupsAPI from '@/api/admin/groups'
import type { AdminGroup } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const products = ref<MediaProduct[]>([])
const groups = ref<AdminGroup[]>([])
const loading = ref(false)
const saving = ref(false)
const showDialog = ref(false)
const editing = ref<number>()
const error = ref('')
const emptyForm = (): MediaProduct => ({ public_model: '', modality: 'image', enabled: false, description: '', group_ids: [], prices: [], offers: [] })
const form = ref<MediaProduct>(emptyForm())
const entryGroups = computed(() => groups.value.filter(group => group.platform === 'openai'))
const validationError = computed(() => {
  if (!form.value.group_ids.length) return t('admin.mediaProducts.needEntryGroup')
  if (!form.value.prices.length) return t('admin.mediaProducts.needPrice')
  if (!form.value.offers.length) return t('admin.mediaProducts.needOffer')
  if (form.value.offers.some(offer => !offer.source_group_id || !offer.operations.length || !offer.cost_source || !offer.cost_version || new Date(offer.expires_at) <= new Date())) return t('admin.mediaProducts.invalidOffer')
  return ''
})

async function load() { loading.value = true; error.value = ''; try { [products.value, groups.value] = await Promise.all([mediaProductsAPI.list(), groupsAPI.getAll()]) } catch (cause) { error.value = extractApiErrorMessage(cause) } finally { loading.value = false } }
function openCreate() { editing.value = undefined; form.value = emptyForm(); showDialog.value = true }
function openEdit(product: MediaProduct) { editing.value = product.id; form.value = JSON.parse(JSON.stringify(product)); showDialog.value = true }
async function save() { if (validationError.value) return; saving.value = true; error.value = ''; try { if (editing.value) await mediaProductsAPI.update(editing.value, form.value); else await mediaProductsAPI.create(form.value); showDialog.value = false; await load() } catch (cause) { error.value = extractApiErrorMessage(cause) } finally { saving.value = false } }
async function disable(product: MediaProduct) { if (!product.id || !window.confirm(t('admin.mediaProducts.disableConfirm', { model: product.public_model }))) return; try { await mediaProductsAPI.remove(product.id); await load() } catch (cause) { error.value = extractApiErrorMessage(cause) } }
onMounted(load)
</script>
