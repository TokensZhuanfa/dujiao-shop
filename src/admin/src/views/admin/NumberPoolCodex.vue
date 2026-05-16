<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { CheckCircle2, RefreshCw, XCircle, Trash2, Upload, LogIn, Gauge, Settings, ChevronUp, ChevronDown, ChevronsUpDown, Globe } from 'lucide-vue-next'
import { adminAPI } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

interface CodexAccount {
  id: number
  alias: string
  email: string
  account_id: string
  plan: string
  status: 'ok' | 'needs_refresh' | 'banned' | 'invalid'
  access_exp: number
  subscription_until: number
  tags: string[]
  note: string
  sold: boolean
  sold_at: string | null
  primary_used_percent: number
  primary_limit_window_seconds: number
  primary_reset_at: number
  secondary_used_percent: number
  secondary_limit_window_seconds: number
  secondary_reset_at: number
  last_usage_at: string | null
  last_usage_error: string
  last_refresh_at: string | null
  last_refresh_error: string
  last_at_updated_at: string | null
  last_rt_updated_at: string | null
  banned_at: string | null
  ban_fail_count: number
  sold_order_no?: string
}

interface PoolSettings {
  auto_refresh_token_enabled: boolean
  auto_refresh_token_interval_minutes: number
  auto_refresh_usage_enabled: boolean
  auto_refresh_usage_interval_minutes: number
  auto_refresh_usage_skip_banned_after: number
  batch_concurrency: number
  proxy_enabled: boolean
  proxy_urls: string
  proxy_url?: string
  last_auto_refresh_token_at?: number
  last_auto_refresh_usage_at?: number
}

const { t } = useI18n()

const accounts = ref<CodexAccount[]>([])
const total = ref(0)
const loading = ref(false)
const error = ref('')
const keyword = ref('')
const statusFilter = ref<string>('')
const planFilter = ref<string>('')
const soldFilter = ref<string>('') // '', 'true', 'false'
const page = ref(1)
const pageSize = ref(50)
const busyId = ref<number | null>(null)
const busyKind = ref<string>('')

type SortKey =
  | 'id' | 'status' | 'email' | 'plan' | 'subscription_until' | 'access_exp'
  | 'primary' | 'secondary' | 'sold' | 'last_refresh_at'
type SortDir = 'asc' | 'desc'
const sortBy = ref<SortKey | null>('id')
const sortDir = ref<SortDir>('asc')

const selectedIds = ref<Set<number>>(new Set())
const lastClickedIdx = ref<number | null>(null)
const batchBusy = ref(false)
const batchProgress = ref('')

// Import modal
const importOpen = ref(false)
const importForm = ref({ json: '', alias_prefix: '', tags: '', note: '' })
const importFiles = ref<File[]>([])
const importSubmitting = ref(false)
const importError = ref('')
const importSuccess = ref('')
const importPerFile = ref<Array<{ file: string; created?: number; skipped?: number; failed?: number; error?: string }>>([])

// Settings modal
const settingsOpen = ref(false)
const settings = ref<PoolSettings>({
  auto_refresh_token_enabled: false,
  auto_refresh_token_interval_minutes: 30,
  auto_refresh_usage_enabled: false,
  auto_refresh_usage_interval_minutes: 60,
  auto_refresh_usage_skip_banned_after: 3,
  batch_concurrency: 5,
  proxy_enabled: false,
  proxy_urls: '',
})
const settingsSubmitting = ref(false)
const settingsError = ref('')
const settingsSuccess = ref('')

interface ProxyTestResult {
  proxy_url: string
  ok: boolean
  ip?: string
  country?: string
  region?: string
  city?: string
  org?: string
  latency_ms: number
  error?: string
}
// key = trim 后的 proxy URL，值 = 测试结果或 'testing'
const proxyTestState = ref<Record<string, ProxyTestResult | 'testing'>>({})
const proxyDirectResult = ref<ProxyTestResult | 'testing' | null>(null)


// Edit detail modal
const editOpen = ref(false)
const editTarget = ref<CodexAccount | null>(null)
const editAlias = ref('')
const editTags = ref('')
const editNote = ref('')

const filteredCount = computed(() => total.value)
const totalPage = computed(() => (pageSize.value > 0 ? Math.max(1, Math.ceil(total.value / pageSize.value)) : 1))

const filtered = computed<CodexAccount[]>(() => {
  let arr = accounts.value
  if (soldFilter.value === 'true') arr = arr.filter((a) => a.sold)
  else if (soldFilter.value === 'false') arr = arr.filter((a) => !a.sold)
  return arr
})

const sortValue = (a: CodexAccount, key: SortKey): number | string | null => {
  switch (key) {
    case 'id': return a.id
    case 'status': {
      const order: Record<string, number> = { ok: 0, needs_refresh: 1, banned: 2, invalid: 3 }
      return order[a.status] ?? 9
    }
    case 'email': return (a.email || '').toLowerCase()
    case 'plan': return a.plan || ''
    case 'subscription_until': return a.subscription_until || null
    case 'access_exp': return a.access_exp || null
    case 'primary': return a.primary_used_percent ?? null
    case 'secondary': return a.secondary_used_percent ?? null
    case 'sold': return a.sold ? 1 : 0
    case 'last_refresh_at': return a.last_refresh_at ? new Date(a.last_refresh_at).getTime() : null
    default: return null
  }
}

const sorted = computed<CodexAccount[]>(() => {
  if (!sortBy.value) return filtered.value
  const arr = [...filtered.value]
  const dir = sortDir.value === 'asc' ? 1 : -1
  arr.sort((a, b) => {
    const va = sortValue(a, sortBy.value!)
    const vb = sortValue(b, sortBy.value!)
    if (va == null && vb == null) return 0
    if (va == null) return 1
    if (vb == null) return -1
    if (typeof va === 'number' && typeof vb === 'number') return (va - vb) * dir
    return String(va).localeCompare(String(vb), 'zh-CN') * dir
  })
  return arr
})

const allSelected = computed(() => sorted.value.length > 0 && sorted.value.every((a) => selectedIds.value.has(a.id)))

const onSort = (key: SortKey) => {
  if (sortBy.value === key) {
    if (sortDir.value === 'asc') sortDir.value = 'desc'
    else {
      sortBy.value = null
      sortDir.value = 'asc'
    }
  } else {
    sortBy.value = key
    sortDir.value = 'asc'
  }
  lastClickedIdx.value = null
}

const load = async () => {
  loading.value = true
  error.value = ''
  try {
    const { data } = await adminAPI.listCodexAccounts({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
      status: statusFilter.value || undefined,
      plan: planFilter.value || undefined,
    })
    accounts.value = data.data || []
    total.value = data.pagination?.total || accounts.value.length
  } catch (err: any) {
    error.value = err?.message || 'load failed'
  } finally {
    loading.value = false
  }
}

const loadSettings = async () => {
  try {
    const { data } = await adminAPI.getCodexPoolSettings()
    const s = data.data || {}
    // 旧字段 proxy_url（单条）合并到 proxy_urls（多行）
    const urls = (s.proxy_urls && s.proxy_urls.trim()) || s.proxy_url || ''
    settings.value = {
      auto_refresh_token_enabled: !!s.auto_refresh_token_enabled,
      auto_refresh_token_interval_minutes: s.auto_refresh_token_interval_minutes || 30,
      auto_refresh_usage_enabled: !!s.auto_refresh_usage_enabled,
      auto_refresh_usage_interval_minutes: s.auto_refresh_usage_interval_minutes || 60,
      auto_refresh_usage_skip_banned_after: s.auto_refresh_usage_skip_banned_after || 3,
      batch_concurrency: s.batch_concurrency || 5,
      proxy_enabled: !!s.proxy_enabled,
      proxy_urls: urls,
      last_auto_refresh_token_at: s.last_auto_refresh_token_at,
      last_auto_refresh_usage_at: s.last_auto_refresh_usage_at,
    }
  } catch {}
}

onMounted(() => {
  load()
  loadSettings()
})

const onSearch = () => {
  page.value = 1
  load()
}

const onRefreshToken = async (acc: CodexAccount) => {
  if (busyId.value) return
  busyId.value = acc.id
  busyKind.value = 'refresh'
  try {
    await adminAPI.refreshCodexAccount(acc.id)
    await load()
  } catch (err: any) {
    error.value = err?.message || 'refresh failed'
  } finally {
    busyId.value = null
    busyKind.value = ''
  }
}

const onFetchUsage = async (acc: CodexAccount) => {
  if (busyId.value) return
  busyId.value = acc.id
  busyKind.value = 'usage'
  try {
    await adminAPI.fetchCodexAccountUsage(acc.id)
    await load()
  } catch (err: any) {
    error.value = err?.message || 'usage failed'
  } finally {
    busyId.value = null
    busyKind.value = ''
  }
}

const onToggleBan = async (acc: CodexAccount) => {
  if (busyId.value) return
  const next = acc.status === 'banned' ? 'ok' : 'banned'
  busyId.value = acc.id
  busyKind.value = 'status'
  try {
    await adminAPI.setCodexAccountStatus(acc.id, next)
    await load()
  } catch (err: any) {
    error.value = err?.message || 'update failed'
  } finally {
    busyId.value = null
    busyKind.value = ''
  }
}

const onToggleSold = async (acc: CodexAccount) => {
  if (busyId.value) return
  busyId.value = acc.id
  busyKind.value = 'sold'
  try {
    await adminAPI.updateCodexAccount(acc.id, { sold: !acc.sold })
    await load()
  } catch (err: any) {
    error.value = err?.message || 'update failed'
  } finally {
    busyId.value = null
    busyKind.value = ''
  }
}

const onDelete = async (acc: CodexAccount) => {
  if (!confirm(t('admin.numberPool.codex.confirmDelete', { alias: acc.alias, email: acc.email || '-' }))) return
  busyId.value = acc.id
  busyKind.value = 'delete'
  try {
    await adminAPI.deleteCodexAccount(acc.id)
    await load()
  } catch (err: any) {
    error.value = err?.message || 'delete failed'
  } finally {
    busyId.value = null
    busyKind.value = ''
  }
}

const onCheckboxToggle = (idx: number, shiftKey: boolean) => {
  if (shiftKey && lastClickedIdx.value !== null && lastClickedIdx.value !== idx) {
    const [s, e] = lastClickedIdx.value < idx ? [lastClickedIdx.value, idx] : [idx, lastClickedIdx.value]
    for (let i = s; i <= e; i++) {
      const target = sorted.value[i]
      if (target) selectedIds.value.add(target.id)
    }
    selectedIds.value = new Set(selectedIds.value)
  } else {
    const id = sorted.value[idx]?.id
    if (!id) return
    if (selectedIds.value.has(id)) selectedIds.value.delete(id)
    else selectedIds.value.add(id)
    selectedIds.value = new Set(selectedIds.value)
  }
  lastClickedIdx.value = idx
}

const shiftHeld = ref(false)
const onRowCheckboxClick = (e: MouseEvent) => {
  shiftHeld.value = e.shiftKey
}
const onRowCheckboxChange = (idx: number) => {
  onCheckboxToggle(idx, shiftHeld.value)
  shiftHeld.value = false
}

const toggleAll = (checked: boolean) => {
  if (checked) sorted.value.forEach((a) => selectedIds.value.add(a.id))
  else sorted.value.forEach((a) => selectedIds.value.delete(a.id))
  selectedIds.value = new Set(selectedIds.value)
  lastClickedIdx.value = null
}

const runBatch = async (action: 'delete' | 'refresh' | 'usage') => {
  const ids = Array.from(selectedIds.value)
  if (ids.length === 0) return
  if (action === 'delete' && !confirm(t('admin.numberPool.codex.confirmBatchDelete', { count: ids.length }))) return
  batchBusy.value = true
  batchProgress.value = t('admin.numberPool.codex.batchRunning', { count: ids.length })
  try {
    const { data } = await adminAPI.batchCodexAccount(action, ids)
    const r = data?.data || { ok: 0, failed: 0 }
    batchProgress.value = t('admin.numberPool.codex.batchDone', { ok: r.ok || 0, failed: r.failed || 0 })
    selectedIds.value = new Set()
    lastClickedIdx.value = null
    await load()
    setTimeout(() => (batchProgress.value = ''), 8000)
  } catch (err: any) {
    error.value = err?.message || 'batch failed'
    batchProgress.value = ''
  } finally {
    batchBusy.value = false
  }
}

const openEdit = (acc: CodexAccount) => {
  editTarget.value = acc
  editAlias.value = acc.alias
  editTags.value = (acc.tags || []).join(', ')
  editNote.value = acc.note || ''
  editOpen.value = true
}

const submitEdit = async () => {
  if (!editTarget.value) return
  const tags = editTags.value.split(/[,，]/).map((s) => s.trim()).filter((s) => s)
  try {
    await adminAPI.updateCodexAccount(editTarget.value.id, {
      alias: editAlias.value.trim() || undefined,
      tags,
      note: editNote.value.trim(),
    })
    editOpen.value = false
    await load()
  } catch (err: any) {
    error.value = err?.message || 'update failed'
  }
}

const submitImport = async () => {
  importError.value = ''
  importSuccess.value = ''
  importPerFile.value = []
  if (importFiles.value.length === 0 && !importForm.value.json.trim()) {
    importError.value = t('admin.numberPool.codex.importJsonRequired')
    return
  }
  importSubmitting.value = true
  try {
    const tags = importForm.value.tags.split(/[,，]/).map((s) => s.trim()).filter((s) => s)
    if (importFiles.value.length > 0) {
      const fd = new FormData()
      for (const f of importFiles.value) fd.append('files', f, f.name)
      if (importForm.value.alias_prefix.trim()) fd.append('alias_prefix', importForm.value.alias_prefix.trim())
      if (importForm.value.note.trim()) fd.append('note', importForm.value.note.trim())
      if (tags.length) fd.append('tags', tags.join(','))
      const { data } = await adminAPI.importCodexAccountsFiles(fd)
      const d = data.data || {}
      importSuccess.value = t('admin.numberPool.codex.importDone', {
        created: d.created ?? 0, skipped: d.skipped ?? 0, failed: d.failed ?? 0,
      })
      importPerFile.value = Array.isArray(d.per_file) ? d.per_file : []
    } else {
      const { data } = await adminAPI.importCodexAccounts({
        json: importForm.value.json.trim(),
        alias_prefix: importForm.value.alias_prefix.trim() || undefined,
        tags: tags.length ? tags : undefined,
        note: importForm.value.note.trim() || undefined,
      })
      const d = data.data || {}
      importSuccess.value = t('admin.numberPool.codex.importDone', {
        created: d.created ?? 0, skipped: d.skipped ?? 0, failed: d.failed ?? 0,
      })
    }
    importFiles.value = []
    importForm.value.json = ''
    page.value = 1
    await load()
  } catch (err: any) {
    importError.value = err?.message || 'import failed'
  } finally {
    importSubmitting.value = false
  }
}

const onImportFilesChange = (e: Event) => {
  const target = e.target as HTMLInputElement
  const list = target.files ? Array.from(target.files) : []
  importFiles.value = [...importFiles.value, ...list]
  target.value = ''
}

const removeImportFile = (idx: number) => {
  importFiles.value.splice(idx, 1)
}

const clearImportFiles = () => {
  importFiles.value = []
}

// 按 name+size 去识别重复行（同名同尺寸视作重复）
const importFileDupKeys = computed<Set<string>>(() => {
  const seen = new Map<string, number>()
  for (const f of importFiles.value) {
    const k = `${f.name} ${f.size}`
    seen.set(k, (seen.get(k) || 0) + 1)
  }
  const out = new Set<string>()
  for (const [k, n] of seen) if (n > 1) out.add(k)
  return out
})
const importFileSummary = computed(() => {
  const total = importFiles.value.length
  const dup = importFileDupKeys.value.size // 有几"组"重复
  const dupRows = importFiles.value.reduce((acc, f) => acc + (importFileDupKeys.value.has(`${f.name} ${f.size}`) ? 1 : 0), 0)
  return { total, dup, dupRows, unique: total - (dupRows - dup) }
})
const fileKey = (f: File) => `${f.name} ${f.size}`

const openSettings = async () => {
  settingsOpen.value = true
  settingsError.value = ''
  settingsSuccess.value = ''
  proxyTestState.value = {}
  proxyDirectResult.value = null
  await loadSettings()
}

const proxyOpen = ref(false)
const openProxy = async () => {
  proxyOpen.value = true
  settingsError.value = ''
  settingsSuccess.value = ''
  proxyTestState.value = {}
  proxyDirectResult.value = null
  await loadSettings()
}

// 解析 textarea 出的代理列表：去空行、去 # 注释、去重保序
const parsedProxyList = computed<string[]>(() => {
  const raw = settings.value.proxy_urls || ''
  const seen = new Set<string>()
  const out: string[] = []
  for (const line of raw.split('\n')) {
    const v = line.trim()
    if (!v || v.startsWith('#')) continue
    if (seen.has(v)) continue
    seen.add(v)
    out.push(v)
  }
  return out
})

const testProxy = async (proxyUrl: string) => {
  const key = (proxyUrl || '').trim()
  if (key) {
    proxyTestState.value = { ...proxyTestState.value, [key]: 'testing' }
  } else {
    proxyDirectResult.value = 'testing'
  }
  try {
    const { data } = await adminAPI.testCodexProxy(key)
    const result: ProxyTestResult = data?.data || data
    if (key) {
      proxyTestState.value = { ...proxyTestState.value, [key]: result }
    } else {
      proxyDirectResult.value = result
    }
  } catch (e: any) {
    const failed: ProxyTestResult = {
      proxy_url: key,
      ok: false,
      latency_ms: 0,
      error: e?.message || 'request failed',
    }
    if (key) {
      proxyTestState.value = { ...proxyTestState.value, [key]: failed }
    } else {
      proxyDirectResult.value = failed
    }
  }
}

const testAllProxies = async () => {
  // 并发 2 + 每条之间 stagger 250ms，避免 ipinfo.io / ip-api 同一时刻被同源轰炸触发限流，
  // 进而把后半批代理误判成"网络连接失败"。
  const list = parsedProxyList.value.slice()
  const concurrency = 2
  let cursor = 0
  const worker = async () => {
    while (cursor < list.length) {
      const i = cursor++
      const p = list[i]
      if (!p) break
      if (i > 0) await new Promise(r => setTimeout(r, 250))
      await testProxy(p)
    }
  }
  await Promise.all(Array.from({ length: Math.min(concurrency, list.length) }, worker))
}

const submitSettings = async () => {
  settingsError.value = ''
  settingsSuccess.value = ''
  settingsSubmitting.value = true
  try {
    await adminAPI.saveCodexPoolSettings({
      auto_refresh_token_enabled: settings.value.auto_refresh_token_enabled,
      auto_refresh_token_interval_minutes: Math.max(1, Math.floor(settings.value.auto_refresh_token_interval_minutes || 30)),
      auto_refresh_usage_enabled: settings.value.auto_refresh_usage_enabled,
      auto_refresh_usage_interval_minutes: Math.max(1, Math.floor(settings.value.auto_refresh_usage_interval_minutes || 60)),
      auto_refresh_usage_skip_banned_after: Math.max(1, Math.min(20, Math.floor(settings.value.auto_refresh_usage_skip_banned_after || 3))),
      batch_concurrency: Math.max(1, Math.min(32, Math.floor(settings.value.batch_concurrency || 5))),
      proxy_enabled: settings.value.proxy_enabled,
      proxy_urls: (settings.value.proxy_urls || '').trim(),
    })
    settingsSuccess.value = t('admin.numberPool.codex.settingsSaved')
    setTimeout(() => (settingsSuccess.value = ''), 3000)
  } catch (err: any) {
    settingsError.value = err?.message || 'save failed'
  } finally {
    settingsSubmitting.value = false
  }
}

// ---- 工具函数 ----

const formatRemain = (ts: number | null | undefined): string => {
  if (!ts) return '-'
  const now = Math.floor(Date.now() / 1000)
  const diff = ts - now
  if (diff <= 0) return t('admin.numberPool.codex.expired')
  if (diff < 60) return `${diff}s`
  if (diff < 3600) return `${Math.floor(diff / 60)}m`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h`
  return `${Math.floor(diff / 86400)}d`
}

const formatDate = (ts: number | null | undefined): string => {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleDateString('zh-CN')
}

const formatIsoDate = (s: string | null | undefined): string => {
  if (!s) return '-'
  return new Date(s).toLocaleString('zh-CN', { hour12: false })
}

// 给操作列刷新按钮拼一个 hover 显示的 multi-line title
const refreshButtonTitle = (acc: CodexAccount): string => {
  const lines: string[] = [t('admin.numberPool.codex.refreshTip')]
  lines.push(`${t('admin.numberPool.codex.detailLastRefresh')}: ${formatIsoDate(acc.last_refresh_at)}`)
  const atLine = acc.last_at_updated_at ? `${formatIsoDate(acc.last_at_updated_at)} (${t('admin.numberPool.codex.changed')})` : t('admin.numberPool.codex.neverChanged')
  lines.push(`${t('admin.numberPool.codex.detailATUpdated')}: ${atLine}`)
  const rtLine = acc.last_rt_updated_at ? `${formatIsoDate(acc.last_rt_updated_at)} (${t('admin.numberPool.codex.changed')})` : t('admin.numberPool.codex.neverChanged')
  lines.push(`${t('admin.numberPool.codex.detailRTUpdated')}: ${rtLine}`)
  if (acc.last_refresh_error) {
    lines.push(`${t('admin.numberPool.codex.detailRefreshError')}: ${acc.last_refresh_error.slice(0, 120)}`)
  }
  return lines.join('\n')
}

// 给操作列"刷新额度"按钮拼 hover title：动作说明 + 上次拉取时间 + 失败原因
const usageButtonTitle = (acc: CodexAccount): string => {
  const lines: string[] = [t('admin.numberPool.codex.usageTip')]
  lines.push(`${t('admin.numberPool.codex.detailLastUsage')}: ${formatIsoDate(acc.last_usage_at)}`)
  if (acc.last_usage_error) {
    lines.push(`${t('admin.numberPool.codex.detailUsageError')}: ${acc.last_usage_error.slice(0, 120)}`)
  }
  return lines.join('\n')
}

const formatIsoTimeAgo = (s: string | null | undefined): string => {
  if (!s) return '-'
  const ts = Math.floor(new Date(s).getTime() / 1000)
  return formatRemainAgo(ts)
}

const formatRemainAgo = (ts: number | null | undefined): string => {
  if (!ts) return '-'
  const now = Math.floor(Date.now() / 1000)
  const diff = now - ts
  if (diff < 0) return '-'
  if (diff < 60) return `${diff}s ago`
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
  return `${Math.floor(diff / 86400)}d ago`
}

const subscriptionState = (ts: number | null | undefined): { expired: boolean; soon: boolean } => {
  if (!ts) return { expired: false, soon: false }
  const now = Math.floor(Date.now() / 1000)
  const diff = ts - now
  return { expired: diff <= 0, soon: diff > 0 && diff < 3 * 86400 }
}

const usagePresent = (acc: CodexAccount): boolean => {
  return !!(acc.last_usage_at || acc.primary_limit_window_seconds || acc.secondary_limit_window_seconds)
}

const usageBarColor = (pct: number): string => {
  if (pct >= 90) return 'bg-rose-500'
  if (pct >= 70) return 'bg-amber-500'
  if (pct >= 30) return 'bg-sky-500'
  return 'bg-emerald-500'
}

const statusBadgeClass = (status: string) => {
  switch (status) {
    case 'ok': return 'text-emerald-600 dark:text-emerald-400'
    case 'needs_refresh': return 'text-amber-600 dark:text-amber-400'
    case 'banned': return 'text-orange-600 dark:text-orange-400'
    default: return 'text-rose-600 dark:text-rose-400'
  }
}

const statusLabel = (status: string) => {
  switch (status) {
    case 'ok': return t('admin.numberPool.codex.statusOk')
    case 'needs_refresh': return t('admin.numberPool.codex.statusNeedsRefresh')
    case 'banned': return t('admin.numberPool.codex.statusBanned')
    default: return t('admin.numberPool.codex.statusInvalid')
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
      <div>
        <h1 class="text-2xl font-semibold">{{ t('admin.numberPool.codex.title') }}</h1>
        <p class="mt-1 text-sm text-muted-foreground">{{ t('admin.numberPool.codex.subtitle') }}</p>
      </div>
      <div class="flex flex-wrap gap-2">
        <Button @click="importOpen = true">
          <Upload class="mr-2 h-4 w-4" />{{ t('admin.numberPool.codex.import') }}
        </Button>
        <Button variant="outline" @click="openSettings">
          <Settings class="mr-2 h-4 w-4" />{{ t('admin.numberPool.codex.settings') }}
        </Button>
        <Button variant="outline" @click="openProxy">
          <Globe class="mr-2 h-4 w-4" />{{ t('admin.numberPool.codex.proxyTitle') }}
        </Button>
        <Button variant="outline" :disabled="selectedIds.size === 0 || batchBusy" @click="runBatch('usage')">
          <Gauge class="mr-2 h-4 w-4" />
          {{ t('admin.numberPool.codex.batchUsage') }}{{ selectedIds.size > 0 ? ` (${selectedIds.size})` : '' }}
        </Button>
        <Button variant="outline" :disabled="selectedIds.size === 0 || batchBusy" @click="runBatch('refresh')">
          <RefreshCw class="mr-2 h-4 w-4" />
          {{ t('admin.numberPool.codex.batchRefresh') }}{{ selectedIds.size > 0 ? ` (${selectedIds.size})` : '' }}
        </Button>
        <Button variant="outline" :disabled="selectedIds.size === 0 || batchBusy" @click="runBatch('delete')">
          <Trash2 class="mr-2 h-4 w-4 text-destructive" />
          {{ t('admin.numberPool.codex.batchDelete') }}{{ selectedIds.size > 0 ? ` (${selectedIds.size})` : '' }}
        </Button>
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <Input v-model="keyword" :placeholder="t('admin.numberPool.codex.searchPlaceholder')" class="max-w-xs" @keyup.enter="onSearch" />
      <Select v-model="statusFilter">
        <SelectTrigger class="w-36"><SelectValue :placeholder="t('admin.numberPool.codex.allStatuses')" /></SelectTrigger>
        <SelectContent>
          <SelectItem value="">{{ t('admin.numberPool.codex.allStatuses') }}</SelectItem>
          <SelectItem value="ok">{{ t('admin.numberPool.codex.statusOk') }}</SelectItem>
          <SelectItem value="needs_refresh">{{ t('admin.numberPool.codex.statusNeedsRefresh') }}</SelectItem>
          <SelectItem value="banned">{{ t('admin.numberPool.codex.statusBanned') }}</SelectItem>
          <SelectItem value="invalid">{{ t('admin.numberPool.codex.statusInvalid') }}</SelectItem>
        </SelectContent>
      </Select>
      <Select v-model="soldFilter">
        <SelectTrigger class="w-32"><SelectValue :placeholder="t('admin.numberPool.codex.allSold')" /></SelectTrigger>
        <SelectContent>
          <SelectItem value="">{{ t('admin.numberPool.codex.allSold') }}</SelectItem>
          <SelectItem value="false">{{ t('admin.numberPool.codex.soldUnsold') }}</SelectItem>
          <SelectItem value="true">{{ t('admin.numberPool.codex.soldSold') }}</SelectItem>
        </SelectContent>
      </Select>
      <Input v-model="planFilter" :placeholder="t('admin.numberPool.codex.planFilter')" class="max-w-32" @keyup.enter="onSearch" />
      <Button variant="outline" @click="onSearch">{{ t('admin.common.search') }}</Button>
      <div class="ml-auto flex items-center gap-3 text-xs text-muted-foreground">
        <span>{{ t('admin.numberPool.codex.totalCount', { total: filteredCount }) }}</span>
        <Select :model-value="String(pageSize)" @update:model-value="(v: any) => { pageSize = Number(v) || 50; page = 1; load() }">
          <SelectTrigger class="w-24 h-8"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="20">20 / 页</SelectItem>
            <SelectItem value="50">50 / 页</SelectItem>
            <SelectItem value="100">100 / 页</SelectItem>
            <SelectItem value="200">200 / 页</SelectItem>
          </SelectContent>
        </Select>
        <div class="flex items-center gap-1">
          <Button size="sm" variant="outline" :disabled="page <= 1" @click="page = page - 1; load()">‹</Button>
          <span class="px-2 tabular-nums">{{ page }} / {{ totalPage }}</span>
          <Button size="sm" variant="outline" :disabled="page >= totalPage" @click="page = page + 1; load()">›</Button>
        </div>
      </div>
    </div>

    <div v-if="batchProgress" class="rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2 text-xs text-emerald-700 dark:text-emerald-300">
      {{ batchProgress }}
    </div>

    <div v-if="error" class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{{ error }}</div>

    <div class="rounded-xl border border-border bg-card overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b text-xs text-muted-foreground">
            <th class="px-3 py-3 text-center font-medium w-10">
              <input type="checkbox" :checked="allSelected" class="h-4 w-4 cursor-pointer rounded border-input accent-primary"
                @change="(e) => toggleAll((e.target as HTMLInputElement).checked)" />
            </th>
            <th class="px-3 py-3 text-center font-medium w-12" :title="t('admin.numberPool.codex.rowNoTip')">
              #
            </th>
            <th class="px-4 py-3 text-center font-medium">
              <button class="inline-flex items-center gap-1 hover:text-foreground" @click="onSort('status')">
                {{ t('admin.numberPool.codex.col.status') }}
                <ChevronUp v-if="sortBy === 'status' && sortDir === 'asc'" class="h-3 w-3" />
                <ChevronDown v-else-if="sortBy === 'status' && sortDir === 'desc'" class="h-3 w-3" />
                <ChevronsUpDown v-else class="h-3 w-3 opacity-30" />
              </button>
            </th>
            <th class="px-4 py-3 text-left font-medium">
              <button class="inline-flex items-center gap-1 hover:text-foreground" @click="onSort('email')">
                {{ t('admin.numberPool.codex.col.account') }}
                <ChevronUp v-if="sortBy === 'email' && sortDir === 'asc'" class="h-3 w-3" />
                <ChevronDown v-else-if="sortBy === 'email' && sortDir === 'desc'" class="h-3 w-3" />
                <ChevronsUpDown v-else class="h-3 w-3 opacity-30" />
              </button>
            </th>
            <th class="px-4 py-3 text-center font-medium">
              <button class="inline-flex items-center gap-1 hover:text-foreground" @click="onSort('plan')">
                {{ t('admin.numberPool.codex.col.plan') }}
                <ChevronUp v-if="sortBy === 'plan' && sortDir === 'asc'" class="h-3 w-3" />
                <ChevronDown v-else-if="sortBy === 'plan' && sortDir === 'desc'" class="h-3 w-3" />
                <ChevronsUpDown v-else class="h-3 w-3 opacity-30" />
              </button>
            </th>
            <th class="px-4 py-3 text-center font-medium">
              <button class="inline-flex items-center gap-1 hover:text-foreground" @click="onSort('subscription_until')">
                {{ t('admin.numberPool.codex.col.subscriptionUntil') }}
                <ChevronUp v-if="sortBy === 'subscription_until' && sortDir === 'asc'" class="h-3 w-3" />
                <ChevronDown v-else-if="sortBy === 'subscription_until' && sortDir === 'desc'" class="h-3 w-3" />
                <ChevronsUpDown v-else class="h-3 w-3 opacity-30" />
              </button>
            </th>
            <th class="px-4 py-3 text-center font-medium min-w-[180px]">
              <button class="inline-flex items-center gap-1 hover:text-foreground" @click="onSort('primary')">
                {{ t('admin.numberPool.codex.col.usage') }}
                <ChevronUp v-if="sortBy === 'primary' && sortDir === 'asc'" class="h-3 w-3" />
                <ChevronDown v-else-if="sortBy === 'primary' && sortDir === 'desc'" class="h-3 w-3" />
                <ChevronsUpDown v-else class="h-3 w-3 opacity-30" />
              </button>
            </th>
            <th class="px-4 py-3 text-center font-medium">
              <button class="inline-flex items-center gap-1 hover:text-foreground" @click="onSort('access_exp')">
                {{ t('admin.numberPool.codex.col.accessExp') }}
                <ChevronUp v-if="sortBy === 'access_exp' && sortDir === 'asc'" class="h-3 w-3" />
                <ChevronDown v-else-if="sortBy === 'access_exp' && sortDir === 'desc'" class="h-3 w-3" />
                <ChevronsUpDown v-else class="h-3 w-3 opacity-30" />
              </button>
            </th>
            <th class="px-4 py-3 text-center font-medium">
              <button class="inline-flex items-center gap-1 hover:text-foreground" @click="onSort('sold')">
                {{ t('admin.numberPool.codex.col.sold') }}
                <ChevronUp v-if="sortBy === 'sold' && sortDir === 'asc'" class="h-3 w-3" />
                <ChevronDown v-else-if="sortBy === 'sold' && sortDir === 'desc'" class="h-3 w-3" />
                <ChevronsUpDown v-else class="h-3 w-3 opacity-30" />
              </button>
            </th>
            <th class="px-4 py-3 text-center font-medium">{{ t('admin.numberPool.codex.col.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading"><td colspan="10" class="px-4 py-8 text-center text-muted-foreground">{{ t('admin.common.loading') }}</td></tr>
          <tr v-else-if="sorted.length === 0">
            <td colspan="10" class="px-4 py-12 text-center text-sm text-muted-foreground">
              {{ t('admin.numberPool.codex.empty') }}
              <div class="mt-3">
                <Button size="sm" @click="importOpen = true">
                  <Upload class="mr-2 h-3.5 w-3.5" />{{ t('admin.numberPool.codex.import') }}
                </Button>
              </div>
            </td>
          </tr>
          <tr v-for="(acc, idx) in sorted" :key="acc.id" class="border-b last:border-0" :class="selectedIds.has(acc.id) && 'bg-primary/5'">
            <td class="px-3 py-3 text-center">
              <input type="checkbox" :checked="selectedIds.has(acc.id)"
                class="h-4 w-4 cursor-pointer rounded border-input accent-primary"
                @click="onRowCheckboxClick" @change="onRowCheckboxChange(idx)" />
            </td>
            <td class="px-3 py-3 text-center text-xs text-muted-foreground tabular-nums" :title="`ID: ${acc.id}`">{{ (page - 1) * pageSize + idx + 1 }}</td>
            <td class="px-4 py-3">
              <div class="flex justify-center">
                <span :class="['inline-flex items-center gap-1 text-xs', statusBadgeClass(acc.status)]"
                  :title="acc.status === 'banned'
                    ? t('admin.numberPool.codex.statusBannedTooltip', { time: formatIsoDate(acc.banned_at), count: acc.ban_fail_count })
                    : t('admin.numberPool.codex.statusTooltip', { time: formatIsoDate(acc.last_usage_at) })">
                  <CheckCircle2 v-if="acc.status === 'ok'" class="h-3.5 w-3.5" />
                  <RefreshCw v-else-if="acc.status === 'needs_refresh'" class="h-3.5 w-3.5" />
                  <XCircle v-else class="h-3.5 w-3.5" />
                  {{ statusLabel(acc.status) }}
                </span>
              </div>
            </td>
            <td class="px-4 py-3">
              <button class="flex flex-col items-start text-left hover:underline" :title="t('admin.numberPool.codex.editMetaTip')" @click="openEdit(acc)">
                <span class="font-medium">{{ acc.email || '-' }}</span>
                <span v-if="acc.account_id" class="text-[10px] text-muted-foreground font-mono truncate" :title="acc.account_id">{{ acc.account_id }}</span>
              </button>
            </td>
            <td class="px-4 py-3 text-center text-xs">{{ acc.plan || '-' }}</td>
            <td class="px-4 py-3 text-center text-xs">
              <span v-if="!acc.subscription_until" class="text-muted-foreground">—</span>
              <div v-else :class="['flex flex-col items-center leading-tight',
                subscriptionState(acc.subscription_until).expired && 'text-rose-500',
                subscriptionState(acc.subscription_until).soon && 'text-amber-600 dark:text-amber-400']"
                :title="new Date(acc.subscription_until * 1000).toLocaleString('zh-CN', { hour12: false })">
                <span>{{ subscriptionState(acc.subscription_until).expired ? t('admin.numberPool.codex.expired') : formatRemain(acc.subscription_until) }}</span>
                <span class="text-[10px] opacity-60">{{ formatDate(acc.subscription_until) }}</span>
              </div>
            </td>
            <td class="px-4 py-3">
              <!-- 失败优先：last_usage_error 不为空时显示错误徽标 -->
              <div v-if="acc.last_usage_error" class="text-center" :title="acc.last_usage_error">
                <span class="inline-flex items-center gap-1 text-xs text-rose-600 dark:text-rose-400">
                  <XCircle class="h-3.5 w-3.5" />
                  {{ t('admin.numberPool.codex.usageError') }}
                </span>
                <div class="mt-0.5 text-[10px] text-muted-foreground truncate max-w-[180px] mx-auto" :title="acc.last_usage_error">
                  {{ acc.last_usage_error }}
                </div>
              </div>
              <div v-else-if="!usagePresent(acc)" class="text-center text-xs text-muted-foreground" :title="t('admin.numberPool.codex.usageEmptyTip')">—</div>
              <div v-else class="space-y-1.5">
                <div class="flex items-center gap-2" :title="t('admin.numberPool.codex.usagePrimaryTip', { reset: formatRemain(acc.primary_reset_at) })">
                  <span class="font-mono text-[10px] text-muted-foreground w-6 shrink-0">5h</span>
                  <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                    <div :class="['h-full rounded-full transition-all', usageBarColor(acc.primary_used_percent)]"
                      :style="{ width: `${Math.max(0, Math.min(100, acc.primary_used_percent))}%` }" />
                  </div>
                  <span class="text-[10px] font-mono tabular-nums w-9 shrink-0 text-right">
                    {{ acc.primary_used_percent < 1 ? '<1' : Math.round(acc.primary_used_percent) }}%
                  </span>
                </div>
                <div class="flex items-center gap-2" :title="t('admin.numberPool.codex.usageSecondaryTip', { reset: formatRemain(acc.secondary_reset_at) })">
                  <span class="font-mono text-[10px] text-muted-foreground w-6 shrink-0">7d</span>
                  <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                    <div :class="['h-full rounded-full transition-all', usageBarColor(acc.secondary_used_percent)]"
                      :style="{ width: `${Math.max(0, Math.min(100, acc.secondary_used_percent))}%` }" />
                  </div>
                  <span class="text-[10px] font-mono tabular-nums w-9 shrink-0 text-right">
                    {{ acc.secondary_used_percent < 1 ? '<1' : Math.round(acc.secondary_used_percent) }}%
                  </span>
                </div>
              </div>
            </td>
            <td class="px-4 py-3 text-center text-xs">{{ formatRemain(acc.access_exp) }}</td>
            <td class="px-4 py-3 text-center">
              <button type="button"
                :class="['inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] border transition-colors',
                  acc.sold
                    ? 'border-emerald-300 bg-emerald-50 text-emerald-700 dark:border-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300'
                    : 'border-border bg-muted/30 text-muted-foreground hover:bg-muted/60']"
                :title="acc.sold
                  ? (acc.sold_order_no
                    ? t('admin.numberPool.codex.soldOrderTip', { order: acc.sold_order_no, time: formatIsoDate(acc.sold_at) })
                    : t('admin.numberPool.codex.soldManualTip', { time: formatIsoDate(acc.sold_at) }))
                  : t('admin.numberPool.codex.toggleSoldTip')"
                :disabled="busyId === acc.id && busyKind === 'sold'"
                @click="onToggleSold(acc)">
                {{ acc.sold ? t('admin.numberPool.codex.soldSold') : t('admin.numberPool.codex.soldUnsold') }}
              </button>
            </td>
            <td class="px-4 py-3">
              <div class="flex justify-center gap-1">
                <Button size="sm" variant="ghost" :disabled="busyId === acc.id && busyKind === 'usage'" :title="usageButtonTitle(acc)" @click="onFetchUsage(acc)">
                  <Gauge :class="['h-3.5 w-3.5', busyId === acc.id && busyKind === 'usage' && 'animate-pulse']" />
                </Button>
                <Button size="sm" variant="ghost" :disabled="busyId === acc.id && busyKind === 'refresh'" :title="refreshButtonTitle(acc)" @click="onRefreshToken(acc)">
                  <RefreshCw :class="['h-3.5 w-3.5', busyId === acc.id && busyKind === 'refresh' && 'animate-spin']" />
                </Button>
                <Button size="sm" variant="ghost" :title="acc.status === 'banned' ? t('admin.numberPool.codex.unbanTip') : t('admin.numberPool.codex.banTip')" @click="onToggleBan(acc)">
                  <XCircle :class="['h-3.5 w-3.5', acc.status === 'banned' ? 'text-amber-500' : 'text-muted-foreground']" />
                </Button>
                <Button size="sm" variant="ghost" :disabled="busyId === acc.id && busyKind === 'delete'" :title="t('admin.common.delete')" @click="onDelete(acc)">
                  <Trash2 class="h-3.5 w-3.5 text-destructive" />
                </Button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <p class="text-[11px] text-muted-foreground">{{ t('admin.numberPool.codex.shiftTip') }}</p>

    <!-- Import dialog -->
    <Dialog v-model:open="importOpen">
      <DialogContent class="max-w-2xl max-h-[85vh] flex flex-col gap-0">
        <DialogHeader>
          <DialogTitle>{{ t('admin.numberPool.codex.import') }}</DialogTitle>
          <DialogDescription>{{ t('admin.numberPool.codex.importHint') }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-4 overflow-y-auto -mx-6 px-6 py-4 flex-1 min-h-0">
          <div>
            <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.numberPool.codex.importFilesLabel') }}</label>
            <div class="flex items-center gap-3 flex-wrap">
              <label class="inline-flex items-center gap-2 text-xs text-blue-600 hover:underline cursor-pointer">
                <Upload class="h-3.5 w-3.5" />{{ t('admin.numberPool.codex.importFilesPick') }}
                <input type="file" accept=".json,.txt,application/json" multiple class="hidden" @change="onImportFilesChange" />
              </label>
              <span v-if="importFileSummary.total" class="text-xs text-muted-foreground">
                {{ t('admin.numberPool.codex.importFilesSelected', { n: importFileSummary.total }) }}
                <span v-if="importFileSummary.dupRows > 0" class="ml-1 text-amber-600">
                  · {{ t('admin.numberPool.codex.importFilesDup', { n: importFileSummary.dupRows }) }}
                </span>
              </span>
              <button v-if="importFileSummary.total" type="button" class="text-xs text-destructive hover:underline" @click="clearImportFiles">
                {{ t('admin.numberPool.codex.importFilesClear') }}
              </button>
            </div>
            <ul v-if="importFiles.length" class="mt-2 space-y-1 max-h-40 overflow-y-auto rounded-lg border border-border bg-muted/30 p-2">
              <li v-for="(file, idx) in importFiles" :key="`${file.name}-${idx}`"
                class="flex items-center justify-between gap-2 text-xs px-2 py-1 rounded"
                :class="importFileDupKeys.has(fileKey(file)) ? 'bg-amber-100/60' : ''">
                <span class="truncate" :title="file.name">{{ file.name }}</span>
                <span v-if="importFileDupKeys.has(fileKey(file))" class="text-[10px] text-amber-700 shrink-0">
                  {{ t('admin.numberPool.codex.importFilesDupBadge') }}
                </span>
                <span class="text-muted-foreground shrink-0">{{ (file.size / 1024).toFixed(1) }} KB</span>
                <button class="text-destructive hover:underline" @click="removeImportFile(idx)">×</button>
              </li>
            </ul>
            <p class="mt-1.5 text-[11px] text-muted-foreground">{{ t('admin.numberPool.codex.importFilesHint') }}</p>
          </div>
          <div>
            <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.numberPool.codex.importJsonLabel') }}</label>
            <Textarea v-model="importForm.json" rows="6" placeholder='{"tokens":{"access_token":"...","refresh_token":"...","id_token":"...","account_id":"..."}}' class="font-mono text-xs" />
          </div>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.numberPool.codex.aliasPrefix') }}</label>
              <Input v-model="importForm.alias_prefix" placeholder="codex" />
            </div>
            <div>
              <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.numberPool.codex.tagsLabel') }}</label>
              <Input v-model="importForm.tags" :placeholder="t('admin.numberPool.codex.tagsPlaceholder')" />
            </div>
          </div>
          <div>
            <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.numberPool.codex.noteLabel') }}</label>
            <Input v-model="importForm.note" />
          </div>
          <div v-if="importError" class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{{ importError }}</div>
          <div v-if="importSuccess" class="rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-700">{{ importSuccess }}</div>
          <div v-if="importPerFile.length" class="rounded-lg border border-border bg-muted/30 p-3 text-xs">
            <div class="font-medium mb-1.5">{{ t('admin.numberPool.codex.importPerFile') }}</div>
            <div class="max-h-64 overflow-y-auto">
              <table class="w-full">
                <thead class="sticky top-0 bg-muted/80 backdrop-blur">
                  <tr class="text-muted-foreground border-b">
                    <th class="text-left py-1 pr-2">{{ t('admin.numberPool.codex.importPerFileCol.file') }}</th>
                    <th class="text-right py-1 px-1">{{ t('admin.numberPool.codex.importPerFileCol.created') }}</th>
                    <th class="text-right py-1 px-1">{{ t('admin.numberPool.codex.importPerFileCol.skipped') }}</th>
                    <th class="text-right py-1 pl-1">{{ t('admin.numberPool.codex.importPerFileCol.failed') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, i) in importPerFile" :key="i" class="border-b last:border-0">
                    <td class="py-1 pr-2 truncate" :title="row.file">{{ row.file }}</td>
                    <td class="py-1 px-1 text-right text-emerald-600 tabular-nums">{{ row.created ?? '-' }}</td>
                    <td class="py-1 px-1 text-right text-amber-600 tabular-nums">{{ row.skipped ?? '-' }}</td>
                    <td class="py-1 pl-1 text-right text-rose-600 tabular-nums">{{ row.error ? '✗' : (row.failed ?? 0) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p class="mt-1.5 text-[10px] text-muted-foreground">{{ t('admin.numberPool.codex.importPerFileHint') }}</p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="importOpen = false">{{ t('admin.common.cancel') }}</Button>
          <Button :disabled="importSubmitting" @click="submitImport">
            <LogIn class="mr-2 h-4 w-4" />
            {{ importSubmitting ? t('admin.numberPool.codex.importing') : t('admin.numberPool.codex.importSubmit') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Settings dialog -->
    <Dialog v-model:open="settingsOpen">
      <DialogContent class="max-w-lg max-h-[85vh] flex flex-col gap-0">
        <DialogHeader>
          <DialogTitle>{{ t('admin.numberPool.codex.settings') }}</DialogTitle>
          <DialogDescription>{{ t('admin.numberPool.codex.settingsHint') }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-4 overflow-y-auto -mx-6 px-6 py-4 flex-1 min-h-0">
          <div class="flex items-start justify-between gap-3 border rounded-lg p-3">
            <div class="flex-1">
              <div class="text-sm font-medium">{{ t('admin.numberPool.codex.autoRefreshToken') }}</div>
              <p class="text-xs text-muted-foreground mt-1">{{ t('admin.numberPool.codex.autoRefreshTokenHint') }}</p>
              <div v-if="settings.last_auto_refresh_token_at" class="mt-1 text-[10px] text-muted-foreground">{{ t('admin.numberPool.codex.lastRunAt', { time: formatIsoTimeAgo(new Date(settings.last_auto_refresh_token_at * 1000).toISOString()) }) }}</div>
              <div v-if="settings.auto_refresh_token_enabled" class="mt-2 flex items-center gap-2">
                <span class="text-xs text-muted-foreground">{{ t('admin.numberPool.codex.intervalMinutes') }}</span>
                <Input v-model.number="settings.auto_refresh_token_interval_minutes" type="number" min="1" class="w-24" />
              </div>
            </div>
            <Switch v-model="settings.auto_refresh_token_enabled" class="mt-1" />
          </div>
          <div class="flex items-start justify-between gap-3 border rounded-lg p-3">
            <div class="flex-1">
              <div class="text-sm font-medium">{{ t('admin.numberPool.codex.autoRefreshUsage') }}</div>
              <p class="text-xs text-muted-foreground mt-1">{{ t('admin.numberPool.codex.autoRefreshUsageHint') }}</p>
              <div v-if="settings.last_auto_refresh_usage_at" class="mt-1 text-[10px] text-muted-foreground">{{ t('admin.numberPool.codex.lastRunAt', { time: formatIsoTimeAgo(new Date(settings.last_auto_refresh_usage_at * 1000).toISOString()) }) }}</div>
              <div v-if="settings.auto_refresh_usage_enabled" class="mt-2 space-y-2">
                <div class="flex items-center gap-2">
                  <span class="text-xs text-muted-foreground">{{ t('admin.numberPool.codex.intervalMinutes') }}</span>
                  <Input v-model.number="settings.auto_refresh_usage_interval_minutes" type="number" min="1" class="w-24" />
                </div>
                <div class="flex items-center gap-2">
                  <span class="text-xs text-muted-foreground">{{ t('admin.numberPool.codex.skipBannedAfter') }}</span>
                  <Input v-model.number="settings.auto_refresh_usage_skip_banned_after" type="number" min="1" max="20" class="w-24" />
                </div>
                <p class="text-[10px] text-muted-foreground">{{ t('admin.numberPool.codex.skipBannedAfterHint') }}</p>
              </div>
            </div>
            <Switch v-model="settings.auto_refresh_usage_enabled" class="mt-1" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ t('admin.numberPool.codex.batchConcurrency') }}</label>
            <Input v-model.number="settings.batch_concurrency" type="number" min="1" max="32" class="w-24" />
            <p class="mt-1 text-xs text-muted-foreground">{{ t('admin.numberPool.codex.batchConcurrencyHint') }}</p>
          </div>
          <div v-if="settingsError" class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{{ settingsError }}</div>
          <div v-if="settingsSuccess" class="rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-700">{{ settingsSuccess }}</div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="settingsOpen = false">{{ t('admin.common.cancel') }}</Button>
          <Button :disabled="settingsSubmitting" @click="submitSettings">{{ settingsSubmitting ? t('admin.common.loading') : t('admin.common.save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Proxy 设置（从 Settings 拆出来，避免页面过臃肿） -->
    <Dialog v-model:open="proxyOpen">
      <DialogContent class="max-w-2xl max-h-[85vh] flex flex-col gap-0">
        <DialogHeader>
          <DialogTitle>{{ t('admin.numberPool.codex.proxyTitle') }}</DialogTitle>
          <DialogDescription>{{ t('admin.numberPool.codex.proxyEnabledHint') }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-3 overflow-y-auto -mx-6 px-6 py-4 flex-1 min-h-0">
          <div class="flex items-start justify-between gap-3 border rounded-lg p-3">
            <div class="flex-1">
              <div class="text-sm font-medium">{{ t('admin.numberPool.codex.proxyEnabled') }}</div>
              <p class="text-xs text-muted-foreground mt-1">{{ t('admin.numberPool.codex.proxyEnabledHint') }}</p>
            </div>
            <Switch v-model="settings.proxy_enabled" class="mt-1" />
          </div>
          <div :class="!settings.proxy_enabled ? 'opacity-60' : ''">
            <div class="flex items-center justify-between mb-1.5">
              <label class="block text-sm font-medium">{{ t('admin.numberPool.codex.proxyUrls') }}</label>
              <div class="flex items-center gap-2">
                <Button type="button" size="sm" variant="outline" @click="testProxy('')">{{ t('admin.numberPool.codex.testDirect') }}</Button>
                <Button type="button" size="sm" variant="outline" :disabled="parsedProxyList.length === 0" @click="testAllProxies">{{ t('admin.numberPool.codex.testAll') }}</Button>
              </div>
            </div>
            <textarea
              v-model="settings.proxy_urls"
              class="w-full min-h-[160px] rounded-md border border-input bg-background px-3 py-2 font-mono text-xs"
              :placeholder="t('admin.numberPool.codex.proxyUrlsPlaceholder')"
            />
            <p class="mt-1 text-xs text-muted-foreground">{{ t('admin.numberPool.codex.proxyUrlsHint') }}</p>

            <!-- 直连基线结果 -->
            <div v-if="proxyDirectResult" class="mt-2 rounded-md border bg-muted/40 px-2.5 py-1.5 text-xs flex items-center gap-2 flex-wrap">
              <span class="font-medium">{{ t('admin.numberPool.codex.testDirect') }}:</span>
              <template v-if="proxyDirectResult === 'testing'">
                <span class="text-muted-foreground">{{ t('admin.numberPool.codex.testing') }}</span>
              </template>
              <template v-else-if="proxyDirectResult.ok">
                <span>{{ proxyDirectResult.ip }}</span>
                <span class="text-muted-foreground">{{ [proxyDirectResult.country, proxyDirectResult.region, proxyDirectResult.city].filter(Boolean).join(' / ') }}</span>
                <span class="text-muted-foreground">{{ proxyDirectResult.latency_ms }}ms</span>
              </template>
              <template v-else>
                <span class="text-destructive">{{ proxyDirectResult.error }}</span>
              </template>
            </div>

            <!-- 每条代理一行测试结果 -->
            <div v-if="parsedProxyList.length" class="mt-2 space-y-1 max-h-64 overflow-y-auto pr-1">
              <div v-for="p in parsedProxyList" :key="p" class="rounded-md border px-2.5 py-1.5 text-xs flex items-center gap-2 flex-wrap">
                <span class="font-mono truncate max-w-[260px]" :title="p">{{ p }}</span>
                <Button type="button" size="sm" variant="ghost" class="h-6 px-2 text-xs" @click="testProxy(p)">{{ t('admin.numberPool.codex.testThis') }}</Button>
                <template v-if="proxyTestState[p] === 'testing'">
                  <span class="text-muted-foreground">{{ t('admin.numberPool.codex.testing') }}</span>
                </template>
                <template v-else-if="proxyTestState[p] && (proxyTestState[p] as ProxyTestResult).ok">
                  <span class="ml-auto"></span>
                  <span class="font-medium">{{ (proxyTestState[p] as ProxyTestResult).ip }}</span>
                  <span class="text-muted-foreground">{{ [(proxyTestState[p] as ProxyTestResult).country, (proxyTestState[p] as ProxyTestResult).region, (proxyTestState[p] as ProxyTestResult).city].filter(Boolean).join(' / ') }}</span>
                  <span class="text-muted-foreground">{{ (proxyTestState[p] as ProxyTestResult).latency_ms }}ms</span>
                </template>
                <template v-else-if="proxyTestState[p]">
                  <span class="ml-auto"></span>
                  <span class="text-destructive">{{ (proxyTestState[p] as ProxyTestResult).error }}</span>
                </template>
              </div>
            </div>
          </div>
          <div v-if="settingsError" class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{{ settingsError }}</div>
          <div v-if="settingsSuccess" class="rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-700">{{ settingsSuccess }}</div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="proxyOpen = false">{{ t('admin.common.cancel') }}</Button>
          <Button :disabled="settingsSubmitting" @click="submitSettings">{{ settingsSubmitting ? t('admin.common.loading') : t('admin.common.save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Edit account dialog -->
    <Dialog v-model:open="editOpen">
      <DialogContent class="max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ t('admin.numberPool.codex.editTitle') }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-3" v-if="editTarget">
          <div>
            <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.numberPool.codex.aliasLabel') }}</label>
            <Input v-model="editAlias" />
          </div>
          <div>
            <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.numberPool.codex.tagsLabel') }}</label>
            <Input v-model="editTags" :placeholder="t('admin.numberPool.codex.tagsPlaceholder')" />
          </div>
          <div>
            <label class="block text-xs font-medium text-muted-foreground mb-1.5">{{ t('admin.numberPool.codex.noteLabel') }}</label>
            <Textarea v-model="editNote" rows="3" />
          </div>
          <div class="grid grid-cols-2 gap-2 rounded-lg border border-border bg-muted/20 p-3 text-xs">
            <div>
              <div class="text-muted-foreground">{{ t('admin.numberPool.codex.detailLastRefresh') }}</div>
              <div class="font-medium">{{ formatIsoDate(editTarget.last_refresh_at) }}</div>
            </div>
            <div>
              <div class="text-muted-foreground">{{ t('admin.numberPool.codex.detailLastUsage') }}</div>
              <div class="font-medium">{{ formatIsoDate(editTarget.last_usage_at) }}</div>
            </div>
            <div>
              <div class="text-muted-foreground">{{ t('admin.numberPool.codex.detailATUpdated') }}</div>
              <div class="font-medium">{{ formatIsoDate(editTarget.last_at_updated_at) }}</div>
            </div>
            <div>
              <div class="text-muted-foreground">{{ t('admin.numberPool.codex.detailRTUpdated') }}</div>
              <div class="font-medium">{{ formatIsoDate(editTarget.last_rt_updated_at) }}</div>
            </div>
            <div v-if="editTarget.last_refresh_error" class="col-span-2">
              <div class="text-muted-foreground">{{ t('admin.numberPool.codex.detailRefreshError') }}</div>
              <div class="text-rose-600 break-all">{{ editTarget.last_refresh_error }}</div>
            </div>
            <div v-if="editTarget.last_usage_error" class="col-span-2">
              <div class="text-muted-foreground">{{ t('admin.numberPool.codex.detailUsageError') }}</div>
              <div class="text-rose-600 break-all">{{ editTarget.last_usage_error }}</div>
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="editOpen = false">{{ t('admin.common.cancel') }}</Button>
          <Button @click="submitEdit">{{ t('admin.common.save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
