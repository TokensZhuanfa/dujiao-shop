<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import FileInput from '@/components/FileInput.vue'

const props = defineProps<{
  modelValue: boolean
  productId: number
  skuId: number
  requireSkuSelection?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  success: []
}>()

const { t } = useI18n()

// --- Manual batch create ---
const batchForm = ref({
  secrets: '',
  batch_no: '',
  note: '',
  deduplicate: true,
})
const batchSubmitting = ref(false)
const batchError = ref('')
const batchSuccess = ref('')

// --- CSV import ---
const importForm = ref({
  file: null as File | null,
  batch_no: '',
  note: '',
  deduplicate: true,
})
const importSubmitting = ref(false)
const importError = ref('')
const importSuccess = ref('')

// --- Files import (one file = one credential) ---
const filesForm = ref({
  files: [] as File[],
  batch_no: '',
  note: '',
})
const filesSubmitting = ref(false)
const filesError = ref('')
const filesSuccess = ref('')

const formatFileSize = (size: number) => {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`
  return `${(size / 1024 / 1024 / 1024).toFixed(2)} GB`
}

const handleFilesChange = (files: FileList | null) => {
  if (!files) return
  const list = Array.from(files)
  // 追加而不是替换，方便分批选择
  filesForm.value.files = [...filesForm.value.files, ...list]
}

const removeFileAt = (idx: number) => {
  filesForm.value.files.splice(idx, 1)
}

const clearAllFiles = () => {
  filesForm.value.files = []
}

const resetFilesForm = () => {
  filesForm.value.files = []
  filesForm.value.batch_no = ''
  filesForm.value.note = ''
  filesError.value = ''
  filesSuccess.value = ''
}

const handleFilesImport = async () => {
  filesError.value = ''
  filesSuccess.value = ''
  if (!props.productId) {
    filesError.value = t('admin.cardSecrets.errors.productRequired')
    return
  }
  if (props.requireSkuSelection && !props.skuId) {
    filesError.value = t('admin.cardSecrets.errors.skuRequired')
    return
  }
  if (!filesForm.value.files.length) {
    filesError.value = t('admin.cardSecrets.errors.fileRequired')
    return
  }

  filesSubmitting.value = true
  try {
    const formData = new FormData()
    formData.append('product_id', String(props.productId))
    if (props.skuId > 0) {
      formData.append('sku_id', String(props.skuId))
    }
    formData.append('batch_no', filesForm.value.batch_no.trim())
    formData.append('note', filesForm.value.note.trim())
    for (const file of filesForm.value.files) {
      formData.append('files', file, file.name)
    }
    await adminAPI.importCardSecretFiles(formData)
    filesSuccess.value = t('admin.cardSecrets.success.imported')
    resetFilesForm()
    emit('success')
  } catch (err: any) {
    filesError.value = err.message || t('admin.cardSecrets.errors.importFailed')
  } finally {
    filesSubmitting.value = false
  }
}

const resetBatchForm = () => {
  batchForm.value.secrets = ''
  batchForm.value.batch_no = ''
  batchForm.value.note = ''
  batchForm.value.deduplicate = true
  batchError.value = ''
  batchSuccess.value = ''
}

const handleBatchCreate = async () => {
  batchError.value = ''
  batchSuccess.value = ''
  if (!props.productId) {
    batchError.value = t('admin.cardSecrets.errors.productRequired')
    return
  }
  if (props.requireSkuSelection && !props.skuId) {
    batchError.value = t('admin.cardSecrets.errors.skuRequired')
    return
  }
  const secrets = batchForm.value.secrets
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter((item) => item)
  if (!secrets.length) {
    batchError.value = t('admin.cardSecrets.errors.secretsRequired')
    return
  }

  batchSubmitting.value = true
  try {
    await adminAPI.createCardSecretBatch({
      product_id: props.productId,
      sku_id: props.skuId || undefined,
      secrets,
      batch_no: batchForm.value.batch_no.trim(),
      note: batchForm.value.note.trim(),
      deduplicate: batchForm.value.deduplicate,
    })
    batchSuccess.value = t('admin.cardSecrets.success.batchCreated')
    batchForm.value.secrets = ''
    emit('success')
  } catch (err: any) {
    batchError.value = err.message || t('admin.cardSecrets.errors.batchFailed')
  } finally {
    batchSubmitting.value = false
  }
}

const handleFileChange = (files: FileList | null) => {
  importForm.value.file = (files && files[0]) || null
}

const clearImportFile = () => {
  importForm.value.file = null
}

const resetImportForm = () => {
  clearImportFile()
  importForm.value.batch_no = ''
  importForm.value.note = ''
  importForm.value.deduplicate = true
  importError.value = ''
  importSuccess.value = ''
}

const handleImport = async () => {
  importError.value = ''
  importSuccess.value = ''
  if (!props.productId) {
    importError.value = t('admin.cardSecrets.errors.productRequired')
    return
  }
  if (props.requireSkuSelection && !props.skuId) {
    importError.value = t('admin.cardSecrets.errors.skuRequired')
    return
  }
  if (!importForm.value.file) {
    importError.value = t('admin.cardSecrets.errors.fileRequired')
    return
  }

  importSubmitting.value = true
  try {
    const formData = new FormData()
    formData.append('product_id', String(props.productId))
    if (props.skuId > 0) {
      formData.append('sku_id', String(props.skuId))
    }
    formData.append('batch_no', importForm.value.batch_no.trim())
    formData.append('note', importForm.value.note.trim())
    formData.append('deduplicate', String(importForm.value.deduplicate))
    formData.append('file', importForm.value.file)
    await adminAPI.importCardSecretCSV(formData)
    importSuccess.value = t('admin.cardSecrets.success.imported')
    resetImportForm()
    emit('success')
  } catch (err: any) {
    importError.value = err.message || t('admin.cardSecrets.errors.importFailed')
  } finally {
    importSubmitting.value = false
  }
}
</script>

<template>
  <div v-if="modelValue" class="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
    <!-- Manual batch create -->
    <div class="rounded-xl border border-border bg-card p-6">
      <h2 class="text-lg font-semibold text-foreground mb-4">{{ t('admin.cardSecrets.batchTitle') }}</h2>
      <form class="space-y-4" @submit.prevent="handleBatchCreate">
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.cardSecrets.secretsLabel') }} *</label>
          <Textarea v-model="batchForm.secrets" rows="6" :placeholder="t('admin.cardSecrets.secretsPlaceholder')" />
        </div>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.cardSecrets.batchNoLabel') }}</label>
            <Input v-model="batchForm.batch_no" placeholder="BATCH-20260203-001" />
          </div>
          <div>
            <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.cardSecrets.noteLabel') }}</label>
            <Input v-model="batchForm.note" :placeholder="t('admin.cardSecrets.notePlaceholder')" />
          </div>
        </div>
        <div class="flex items-start justify-between gap-4 border-y border-border py-3">
          <div>
            <label for="card-secret-batch-deduplicate" class="text-sm font-medium text-foreground">
              {{ t('admin.cardSecrets.deduplicateLabel') }}
            </label>
            <p class="mt-1 text-xs text-muted-foreground">{{ t('admin.cardSecrets.deduplicateHint') }}</p>
          </div>
          <Switch id="card-secret-batch-deduplicate" v-model="batchForm.deduplicate" class="mt-0.5" />
        </div>
        <div v-if="batchError" class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
          {{ batchError }}
        </div>
        <div v-if="batchSuccess" class="rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-700">
          {{ batchSuccess }}
        </div>
        <div class="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <Button class="w-full sm:w-auto" type="button" variant="outline" @click="resetBatchForm">{{ t('admin.common.reset') }}</Button>
          <Button class="w-full sm:w-auto" type="submit" :disabled="batchSubmitting || !!(props.requireSkuSelection && !props.skuId)">
            {{ batchSubmitting ? t('admin.cardSecrets.submitting') : t('admin.cardSecrets.submitBatch') }}
          </Button>
        </div>
      </form>
    </div>

    <!-- CSV import -->
    <div class="rounded-xl border border-border bg-card p-6">
      <h2 class="text-lg font-semibold text-foreground mb-4">{{ t('admin.cardSecrets.importTitle') }}</h2>
      <form class="space-y-4" @submit.prevent="handleImport">
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.cardSecrets.csvLabel') }} *</label>
          <div class="flex flex-wrap items-center gap-2">
            <FileInput
              accept=".csv"
              :button-text="t('admin.cardSecrets.csvChoose')"
              @change="handleFileChange"
            />
            <Button v-if="importForm.file" type="button" size="sm" variant="ghost" @click="clearImportFile">{{ t('admin.cardSecrets.csvClear') }}</Button>
          </div>
          <p class="mt-2 text-xs text-muted-foreground">{{ t('admin.cardSecrets.csvHint') }}</p>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.cardSecrets.batchNoLabel') }}</label>
            <Input v-model="importForm.batch_no" placeholder="BATCH-20260203-002" />
          </div>
          <div>
            <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.cardSecrets.noteLabel') }}</label>
            <Input v-model="importForm.note" :placeholder="t('admin.cardSecrets.importNotePlaceholder')" />
          </div>
        </div>
        <div class="flex items-start justify-between gap-4 border-y border-border py-3">
          <div>
            <label for="card-secret-csv-deduplicate" class="text-sm font-medium text-foreground">
              {{ t('admin.cardSecrets.deduplicateLabel') }}
            </label>
            <p class="mt-1 text-xs text-muted-foreground">{{ t('admin.cardSecrets.deduplicateHint') }}</p>
          </div>
          <Switch id="card-secret-csv-deduplicate" v-model="importForm.deduplicate" class="mt-0.5" />
        </div>
        <div v-if="importError" class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
          {{ importError }}
        </div>
        <div v-if="importSuccess" class="rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-700">
          {{ importSuccess }}
        </div>
        <div class="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <Button class="w-full sm:w-auto" type="button" variant="outline" @click="resetImportForm">{{ t('admin.common.reset') }}</Button>
          <Button class="w-full sm:w-auto" type="submit" :disabled="importSubmitting || !!(props.requireSkuSelection && !props.skuId)">
            {{ importSubmitting ? t('admin.cardSecrets.importing') : t('admin.cardSecrets.startImport') }}
          </Button>
        </div>
      </form>
    </div>

    <!-- Files import (one file per credential) -->
    <div class="rounded-xl border border-border bg-card p-6">
      <h2 class="text-lg font-semibold text-foreground mb-4">{{ t('admin.cardSecrets.filesTitle') }}</h2>
      <form class="space-y-4" @submit.prevent="handleFilesImport">
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.cardSecrets.filesLabel') }} *</label>
          <div class="flex flex-wrap items-center gap-2">
            <FileInput
              :multiple="true"
              :button-text="t('admin.cardSecrets.filesChoose')"
              @change="handleFilesChange"
            />
            <Button v-if="filesForm.files.length" type="button" size="sm" variant="ghost" @click="clearAllFiles">{{ t('admin.cardSecrets.filesClearAll') }}</Button>
          </div>
          <p class="mt-2 text-xs text-muted-foreground">{{ t('admin.cardSecrets.filesHint') }}</p>
          <ul v-if="filesForm.files.length" class="mt-3 space-y-1.5 max-h-48 overflow-y-auto rounded-lg border border-border bg-muted/30 p-2">
            <li v-for="(file, idx) in filesForm.files" :key="`${file.name}-${idx}`" class="flex items-center justify-between gap-2 rounded px-2 py-1 text-xs">
              <span class="truncate text-foreground" :title="file.name">{{ file.name }}</span>
              <span class="shrink-0 text-muted-foreground">{{ formatFileSize(file.size) }}</span>
              <button type="button" class="shrink-0 text-destructive hover:underline" @click="removeFileAt(idx)">×</button>
            </li>
          </ul>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.cardSecrets.batchNoLabel') }}</label>
            <Input v-model="filesForm.batch_no" placeholder="BATCH-20260203-003" />
          </div>
          <div>
            <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.cardSecrets.noteLabel') }}</label>
            <Input v-model="filesForm.note" :placeholder="t('admin.cardSecrets.importNotePlaceholder')" />
          </div>
        </div>
        <div v-if="filesError" class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
          {{ filesError }}
        </div>
        <div v-if="filesSuccess" class="rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-700">
          {{ filesSuccess }}
        </div>
        <div class="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <Button class="w-full sm:w-auto" type="button" variant="outline" @click="resetFilesForm">{{ t('admin.common.reset') }}</Button>
          <Button class="w-full sm:w-auto" type="submit" :disabled="filesSubmitting || !filesForm.files.length || !!(props.requireSkuSelection && !props.skuId)">
            {{ filesSubmitting ? t('admin.cardSecrets.importing') : t('admin.cardSecrets.startImport') }}
          </Button>
        </div>
      </form>
    </div>
  </div>
</template>
