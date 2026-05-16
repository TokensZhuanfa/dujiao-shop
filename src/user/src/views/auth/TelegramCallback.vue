<template>
  <div class="min-h-[50vh] flex items-center justify-center px-6 py-12 text-center">
    <div v-if="loading" class="text-sm theme-text-muted">{{ t('auth.telegramCallback.processing') }}</div>
    <div v-else-if="errMsg" class="space-y-3">
      <p class="text-sm text-red-500">{{ errMsg }}</p>
      <RouterLink to="/auth/login" class="text-sm underline theme-link-muted">{{ t('auth.telegramCallback.backToLogin') }}</RouterLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useUserAuthStore } from '../../stores/userAuth'
import { userProfileAPI } from '../../api'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const userAuthStore = useUserAuthStore()
const loading = ref(true)
const errMsg = ref('')

onMounted(async () => {
  const code = String(route.query.code || '')
  const state = String(route.query.state || '')
  const oauthErr = String(route.query.error || '')
  const intent = sessionStorage.getItem('tg_oidc_intent') || 'login'
  const savedRedirect = sessionStorage.getItem('tg_oidc_redirect') || ''
  sessionStorage.removeItem('tg_oidc_intent')
  sessionStorage.removeItem('tg_oidc_redirect')

  if (oauthErr || !code || !state) {
    errMsg.value = t('auth.telegramCallback.failed')
    loading.value = false
    return
  }

  try {
    if (intent === 'bind') {
      await userProfileAPI.telegramOidcBindCallback({ code, state })
      await router.replace({ path: '/me/security', query: { tgBound: '1' } })
      return
    }
    const result = await userAuthStore.telegramOidcLogin({ code, state })
    if (result?.requiresTotp) {
      const query: Record<string, string> = { tg2fa: '1' }
      if (savedRedirect) {
        query.redirect = savedRedirect
      }
      await router.replace({ path: '/auth/login', query })
      return
    }
    await router.replace(savedRedirect || '/me/orders')
  } catch (err: any) {
    errMsg.value = err?.message || t('auth.telegramCallback.failed')
    loading.value = false
  }
})
</script>
