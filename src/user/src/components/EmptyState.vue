<template>
  <div
    class="text-center backdrop-blur-sm theme-slide-up"
    :class="[
      variant === 'soft' ? 'theme-panel-soft border' : 'theme-panel border',
      sizeClass,
    ]"
  >
    <div class="flex justify-center" :class="iconWrapperClass">
      <slot name="icon">
        <component :is="resolvedIcon" :class="iconClass" aria-hidden="true" />
      </slot>
    </div>

    <p v-if="title" class="theme-text-secondary font-medium" :class="titleClass">{{ title }}</p>
    <p v-if="description" class="theme-text-muted mt-2 mx-auto" :class="descriptionClass">
      {{ description }}
    </p>

    <div v-if="hasAction" class="mt-6 flex flex-wrap justify-center gap-3">
      <slot name="action">
        <router-link
          v-if="actionTo"
          :to="actionTo"
          class="theme-btn-inline-md theme-btn-primary font-semibold"
        >
          {{ actionLabel }}
        </router-link>
        <button
          v-else-if="actionLabel"
          type="button"
          class="theme-btn-inline-md theme-btn-primary font-semibold"
          :disabled="actionDisabled"
          @click="$emit('action')"
        >
          {{ actionLabel }}
        </button>
      </slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, h, type FunctionalComponent, useSlots } from 'vue'
import type { RouteLocationRaw } from 'vue-router'

type IconName = 'package' | 'cart' | 'order' | 'search' | 'inbox' | 'alert'
type Variant = 'default' | 'soft'
type Size = 'sm' | 'md' | 'lg'

const props = withDefaults(
  defineProps<{
    icon?: IconName
    title?: string
    description?: string
    actionLabel?: string
    actionTo?: RouteLocationRaw
    actionDisabled?: boolean
    variant?: Variant
    size?: Size
  }>(),
  {
    icon: 'inbox',
    variant: 'default',
    size: 'md',
  },
)

defineEmits<{ action: [] }>()

const slots = useSlots()

const hasAction = computed(
  () => !!slots.action || !!props.actionTo || !!props.actionLabel,
)

const sizeClass = computed(() => {
  switch (props.size) {
    case 'sm':
      return 'rounded-2xl p-8'
    case 'lg':
      return 'rounded-3xl p-16'
    default:
      return 'rounded-2xl p-12'
  }
})

const iconWrapperClass = computed(() => (props.size === 'sm' ? 'mb-3' : 'mb-5'))

const iconClass = computed(() => {
  const base = 'theme-text-muted opacity-70'
  if (props.size === 'sm') return `w-12 h-12 ${base}`
  if (props.size === 'lg') return `w-24 h-24 ${base}`
  return `w-16 h-16 ${base}`
})

const titleClass = computed(() =>
  props.size === 'sm' ? 'text-sm' : props.size === 'lg' ? 'text-xl' : 'text-base',
)

const descriptionClass = computed(() =>
  props.size === 'sm' ? 'text-xs max-w-sm' : 'text-sm max-w-md',
)

const makeIcon = (path: string): FunctionalComponent =>
  (_, { attrs }) =>
    h(
      'svg',
      {
        ...attrs,
        xmlns: 'http://www.w3.org/2000/svg',
        fill: 'none',
        viewBox: '0 0 24 24',
        stroke: 'currentColor',
        'stroke-width': 1.5,
      },
      [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: path })],
    )

const icons: Record<IconName, FunctionalComponent> = {
  package: makeIcon(
    'M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4',
  ),
  cart: makeIcon(
    'M3 3h2l.4 2M7 13h10l4-8H5.4M7 13l-1.3 2.6a1 1 0 00.9 1.4H19M7 13l.4 2M10 21a1 1 0 100-2 1 1 0 000 2zm8 1a1 1 0 100-2 1 1 0 000 2z',
  ),
  order: makeIcon(
    'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z',
  ),
  search: makeIcon(
    'M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z',
  ),
  inbox: makeIcon(
    'M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4',
  ),
  alert: makeIcon(
    'M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z',
  ),
}

const resolvedIcon = computed(() => icons[props.icon] ?? icons.inbox)
</script>
