<template>
  <nav
    class="text-sm theme-text-muted font-medium"
    :aria-label="t('breadcrumb.label')"
  >
    <!-- Desktop: full breadcrumb trail -->
    <ol class="hidden md:flex items-center space-x-2 flex-wrap">
      <li v-for="(item, idx) in items" :key="idx" class="flex items-center space-x-2">
        <router-link
          v-if="item.to && idx < items.length - 1"
          :to="item.to"
          class="theme-link-muted transition-colors"
        >{{ item.label }}</router-link>
        <span
          v-else
          class="theme-text-primary truncate max-w-[240px]"
          :aria-current="idx === items.length - 1 ? 'page' : undefined"
        >{{ item.label }}</span>
        <span v-if="idx < items.length - 1" aria-hidden="true" class="theme-text-muted/60">/</span>
      </li>
    </ol>

    <!-- Mobile: compact back-style -->
    <div class="md:hidden flex items-center gap-2">
      <button
        v-if="parent"
        type="button"
        class="inline-flex items-center gap-1.5 theme-link-muted transition-colors active:opacity-70"
        :aria-label="t('breadcrumb.back', { name: parent.label })"
        @click="handleBack"
      >
        <svg
          class="w-4 h-4 shrink-0"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
          aria-hidden="true"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
        </svg>
        <span class="truncate max-w-[40vw]">{{ parent.label }}</span>
      </button>
      <span v-if="parent && current" aria-hidden="true" class="theme-text-muted/60">/</span>
      <span
        v-if="current"
        class="theme-text-primary truncate flex-1"
        aria-current="page"
      >{{ current.label }}</span>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter, type RouteLocationRaw } from 'vue-router'

interface BreadcrumbItem {
  label: string
  to?: RouteLocationRaw
}

const props = defineProps<{
  items: BreadcrumbItem[]
}>()

const { t } = useI18n()
const router = useRouter()

const parent = computed<BreadcrumbItem | null>(() =>
  props.items.length >= 2 ? props.items[props.items.length - 2] ?? null : null,
)

const current = computed<BreadcrumbItem | null>(() =>
  props.items.length > 0 ? props.items[props.items.length - 1] ?? null : null,
)

const handleBack = () => {
  if (parent.value?.to) {
    router.push(parent.value.to)
  } else {
    router.back()
  }
}
</script>
