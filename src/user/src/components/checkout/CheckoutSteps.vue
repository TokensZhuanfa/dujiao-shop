<template>
  <ol
    class="rounded-2xl border theme-border theme-panel-soft p-4 backdrop-blur flex items-center list-none"
    :aria-label="t('checkoutSteps.label')"
  >
    <template v-for="(step, idx) in steps" :key="step.key">
      <li
        class="flex items-center gap-2"
        :class="idx === 0 ? '' : 'flex-1'"
        :aria-current="step.status === 'current' ? 'step' : undefined"
      >
        <div
          v-if="idx > 0"
          aria-hidden="true"
          class="flex-1 h-0.5 rounded-full transition-colors"
          :class="step.status !== 'upcoming' ? 'bg-current theme-text-accent' : 'theme-surface-muted'"
        ></div>
        <div class="flex items-center gap-2 shrink-0">
          <span
            class="w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold border-2 transition-colors"
            :class="step.status !== 'upcoming'
              ? 'theme-btn-primary border-transparent'
              : 'border-gray-300 dark:border-gray-600 theme-text-muted'"
          >
            <svg
              v-if="step.status === 'done'"
              class="w-3.5 h-3.5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
            </svg>
            <span v-else>{{ idx + 1 }}</span>
          </span>
          <span
            class="text-sm font-medium hidden sm:inline"
            :class="step.status !== 'upcoming' ? 'theme-text-primary' : 'theme-text-muted'"
          >
            {{ step.label }}
          </span>
        </div>
      </li>
    </template>
  </ol>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

type StepKey = 'cart' | 'checkout' | 'payment'
type StepStatus = 'done' | 'current' | 'upcoming'

const props = withDefaults(
  defineProps<{
    currentStep: StepKey
    stepKeys?: StepKey[]
  }>(),
  {
    stepKeys: () => ['cart', 'checkout', 'payment'],
  },
)

const { t } = useI18n()

const steps = computed(() => {
  const activeIdx = props.stepKeys.indexOf(props.currentStep)
  if (activeIdx < 0 && import.meta.env.DEV) {
    const brandWarn = (globalThis as any).console?.warn?.bind(console)
    brandWarn?.(`[CheckoutSteps] currentStep "${props.currentStep}" not found in stepKeys`, props.stepKeys)
  }
  return props.stepKeys.map((key, idx) => {
    const status: StepStatus = activeIdx < 0
      ? 'upcoming'
      : idx < activeIdx
        ? 'done'
        : idx === activeIdx
          ? 'current'
          : 'upcoming'
    return { key, label: t(`${key}.title`), status }
  })
})
</script>
