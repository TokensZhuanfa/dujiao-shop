<template>
  <div v-if="totalPages > 1" class="flex justify-center" :class="compact ? 'mt-8' : 'mt-12'">
    <nav class="flex items-center space-x-2 theme-panel-soft backdrop-blur-md p-2 rounded-2xl border" :aria-label="t('pagination.label')">
      <button
        type="button"
        :aria-label="t('pagination.previous')"
        :disabled="currentPage === 1 || loading"
        :aria-disabled="currentPage === 1 || loading"
        class="flex items-center justify-center rounded-xl border theme-btn-secondary transition-all disabled:opacity-30 disabled:cursor-not-allowed"
        :class="compact ? 'w-10 h-10' : 'w-12 h-12'"
        @click="handlePage(currentPage - 1)"
      >
        <svg :class="compact ? 'w-4 h-4' : 'w-5 h-5'" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>

      <span class="font-mono theme-text-muted flex items-center" :class="compact ? 'px-3 py-1.5 text-sm' : 'px-4 py-2'" aria-live="polite">
        <svg
          v-if="loading"
          class="animate-spin mr-2 theme-text-accent"
          :class="compact ? 'w-3.5 h-3.5' : 'w-4 h-4'"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" class="opacity-25" />
          <path fill="currentColor" class="opacity-75" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
        </svg>
        <span class="theme-text-primary font-bold">{{ currentPage }}</span>
        <span :class="compact ? 'mx-1.5' : 'mx-2'" class="opacity-50">/</span>
        <span>{{ totalPages }}</span>
      </span>

      <button
        type="button"
        :aria-label="t('pagination.next')"
        :disabled="currentPage === totalPages || loading"
        :aria-disabled="currentPage === totalPages || loading"
        class="flex items-center justify-center rounded-xl border theme-btn-secondary transition-all disabled:opacity-30 disabled:cursor-not-allowed"
        :class="compact ? 'w-10 h-10' : 'w-12 h-12'"
        @click="handlePage(currentPage + 1)"
      >
        <svg :class="compact ? 'w-4 h-4' : 'w-5 h-5'" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
        </svg>
      </button>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  currentPage: number
  totalPages: number
  compact?: boolean
  loading?: boolean
  scrollTop?: boolean
}>()

const emit = defineEmits<{
  'changePage': [page: number]
}>()

const { t } = useI18n()

const handlePage = (page: number) => {
  if (page < 1 || page > props.totalPages || page === props.currentPage) return
  emit('changePage', page)
  if (props.scrollTop !== false) {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}
</script>
