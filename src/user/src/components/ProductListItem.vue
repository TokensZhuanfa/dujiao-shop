<template>
  <div
    class="group relative theme-panel rounded-xl border transition-all overflow-hidden flex flex-row items-center cursor-pointer theme-slide-up"
    :class="isSoldOut(product)
      ? 'opacity-85 grayscale-[0.25] saturate-50 border-rose-300/60 dark:border-rose-900/40'
      : 'theme-card-interactive'"
    :style="{ animationDelay: `${index * animationStep}ms` }"
    @click="$emit('click', product.slug)">

    <!-- Thumbnail -->
    <div class="w-11 h-11 sm:w-16 sm:h-16 flex-shrink-0 overflow-hidden relative rounded-lg m-1.5 sm:m-2.5">
      <img v-if="product.images && getFirstImageUrl(product.images)" :src="getFirstImageUrl(product.images)"
        :alt="getLocalizedText(product.title)" loading="lazy"
        class="w-full h-full object-cover transition-transform duration-500"
        :class="isSoldOut(product) ? 'grayscale brightness-75' : 'group-hover:scale-110'" />
      <img v-else-if="product.category?.icon" :src="getImageUrl(product.category.icon)"
        :alt="getLocalizedText(product.category?.name)" loading="lazy"
        class="w-full h-full object-cover transition-transform duration-500"
        :class="isSoldOut(product) ? 'grayscale brightness-75' : 'group-hover:scale-110'" />
      <div v-else class="w-full h-full flex items-center justify-center theme-surface-muted theme-text-muted">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
            d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
      </div>
      <!-- Sold out overlay -->
      <div v-if="isSoldOut(product)" class="absolute inset-0 bg-black/50 flex items-center justify-center">
        <span class="text-white text-[10px] font-bold">{{ t('products.stockStatus.outOfStock') }}</span>
      </div>
    </div>

    <!-- Info (flex-1) -->
    <div class="flex-1 min-w-0 py-1.5 sm:py-2 pr-0.5 sm:pr-1 flex flex-col justify-center gap-0.5">
      <!-- Row 1: Category · Title · Tags -->
      <div class="flex items-center gap-1.5 min-w-0">
        <span v-if="product.category?.name" class="hidden sm:inline text-[11px] theme-text-muted uppercase tracking-wider truncate max-w-[80px] flex-shrink-0">
          {{ getLocalizedText(product.category.name) }}
        </span>
        <span v-if="product.category?.name" class="hidden sm:inline text-[11px] theme-text-muted opacity-30 flex-shrink-0">·</span>
        <h3 class="text-xs sm:text-sm font-semibold theme-text-primary truncate">
          {{ getLocalizedText(product.title) }}
        </h3>
        <span v-for="(tag, tagIndex) in (product.tags || []).slice(0, 1)" :key="tagIndex"
          class="hidden md:inline-flex theme-badge theme-badge-inverse text-[10px] px-1.5 py-0 flex-shrink-0">
          {{ tag }}
        </span>
      </div>

      <!-- Row 2: Badges -->
      <div class="flex flex-wrap items-center gap-1">
        <!-- Mobile: fulfillment + stock warning -->
        <span class="sm:hidden theme-badge text-[10px]"
          :class="product.fulfillment_type === 'auto' ? 'theme-badge-info' : 'theme-badge-neutral'">
          {{ getFulfillmentTypeLabel(product.fulfillment_type) }}
        </span>
        <span v-if="product.stock_status === 'out_of_stock' || product.stock_status === 'low_stock'"
          class="sm:hidden theme-badge text-[10px]"
          :class="getStockBadgeClass(product.stock_status)">
          {{ getStockStatusLabel(product) }}
        </span>

        <!-- Desktop: all badges -->
        <span class="hidden sm:inline-flex theme-badge text-[10px]"
          :class="product.purchase_type === 'guest' ? 'theme-badge-warning' : 'theme-badge-success'">
          {{ getPurchaseTypeLabel(product.purchase_type) }}
        </span>
        <span class="hidden sm:inline-flex theme-badge text-[10px]"
          :class="product.fulfillment_type === 'auto' ? 'theme-badge-info' : 'theme-badge-neutral'">
          {{ getFulfillmentTypeLabel(product.fulfillment_type) }}
        </span>
        <span class="hidden sm:inline-flex theme-badge text-[10px]"
          :class="getStockBadgeClass(product.stock_status)">
          {{ getStockStatusLabel(product) }}
        </span>
      </div>
    </div>

    <!-- Price + Actions (right) -->
    <div class="flex-shrink-0 flex items-center gap-1 sm:gap-3 pr-1.5 sm:pr-4">
      <!-- Price -->
      <div class="text-right">
        <div v-if="hasPromotionPrice(product)" class="flex flex-col items-end">
          <span class="text-xs sm:text-sm font-bold text-rose-600 dark:text-rose-300 whitespace-nowrap">
            {{ formatPrice(getPromotionPriceAmount(product), siteCurrency) }}
          </span>
          <div class="flex items-center gap-1">
            <span class="text-[10px] theme-text-muted line-through">{{ formatPrice(product.price_amount, siteCurrency) }}</span>
            <span class="theme-badge theme-badge-danger text-[9px] px-1 py-0 leading-tight">{{ t('products.promotionTag') }}</span>
          </div>
        </div>
        <div v-else>
          <span class="text-xs sm:text-sm font-bold theme-text-primary whitespace-nowrap">
            {{ formatPrice(product.price_amount, siteCurrency) }}
          </span>
          <div v-if="hasPromotionRules(product)">
            <span class="theme-badge theme-badge-warning text-[9px] px-1 py-0 leading-tight">{{ t('products.promotionBadge') }}</span>
          </div>
        </div>
      </div>

      <!-- Quick buy cart icon -->
      <button
        type="button"
        class="relative flex items-center justify-center w-7 h-7 sm:w-8 sm:h-8 rounded-lg border transition-all flex-shrink-0"
        :class="isSoldOut(product)
          ? 'opacity-40 cursor-not-allowed border-gray-200 dark:border-gray-700 text-gray-400 dark:text-gray-600'
          : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:text-gray-900 hover:border-gray-400 hover:bg-gray-100 dark:hover:text-white dark:hover:border-gray-500 dark:hover:bg-gray-800'"
        :disabled="isSoldOut(product)"
        @click.stop="$emit('quickBuy', product)"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M3 3h2l.4 2M7 13h10l4-8H5.4M7 13L5.4 5M7 13l-2.293 2.293c-.63.63-.184 1.707.707 1.707H17m0 0a2 2 0 100 4 2 2 0 000-4zm-8 2a2 2 0 100 4 2 2 0 000-4z" />
        </svg>
      </button>

      <!-- Arrow -->
      <svg class="hidden sm:block w-4 h-4 flex-shrink-0 transition-transform theme-text-muted"
        :class="isSoldOut(product) ? '' : 'group-hover:translate-x-0.5 group-hover:text-gray-900 dark:group-hover:text-white'"
        fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
      </svg>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getFirstImageUrl, getImageUrl } from '../utils/image'
import { useLocalized, useProductLabels } from '../composables/useProduct'

withDefaults(defineProps<{
  product: any
  index?: number
  animationStep?: number
}>(), {
  index: 0,
  animationStep: 30,
})

defineEmits<{
  click: [slug: string]
  quickBuy: [product: any]
}>()

const { t } = useI18n()
const { getLocalizedText, siteCurrency, formatPrice } = useLocalized()
const { getPurchaseTypeLabel, getFulfillmentTypeLabel, getStockBadgeClass, getStockStatusLabel, isSoldOut, hasPromotionPrice, getPromotionPriceAmount, hasPromotionRules } = useProductLabels()
</script>
