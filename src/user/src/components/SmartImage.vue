<script setup lang="ts">
import { ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    src?: string | null
    alt?: string
    imgClass?: string
    loading?: 'lazy' | 'eager'
    decoding?: 'async' | 'sync' | 'auto'
  }>(),
  {
    alt: '',
    loading: 'lazy',
    decoding: 'async',
  },
)

const errored = ref(false)
watch(
  () => props.src,
  () => { errored.value = false },
)

const handleError = () => { errored.value = true }
</script>

<template>
  <img
    v-if="src && !errored"
    :src="src"
    :alt="alt"
    :loading="loading"
    :decoding="decoding"
    :class="imgClass"
    @error="handleError"
  />
  <slot v-else name="fallback">
    <div
      class="flex h-full w-full items-center justify-center theme-text-muted theme-surface-muted"
      role="img"
      :aria-label="alt || undefined"
    >
      <svg
        class="h-8 w-8 opacity-60"
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width="1.5"
        aria-hidden="true"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
        />
      </svg>
    </div>
  </slot>
</template>
