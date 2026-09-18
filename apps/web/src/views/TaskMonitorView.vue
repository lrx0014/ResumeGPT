<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

import ListFilters from '../components/ListFilters.vue'
import ListPagination from '../components/ListPagination.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import type { BackgroundTask, BackgroundTaskState } from '../lib/types'

const tasks = ref<BackgroundTask[]>([])
const counts = ref<Record<string, number>>({})
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const search = ref('')
const stateFilter = ref('all')
const kindFilter = ref('all')
const selected = ref<BackgroundTask | null>(null)
const loading = ref(true)
const detailLoading = ref(false)
const error = ref('')
let pollTimer: number | undefined
let searchTimer: number | undefined

const kinds = computed(() => [
  { value: 'job.page.import.v1', label: t('tasks.kinds.jobImports') },
  { value: 'job.hunt.v1', label: t('tasks.kinds.jobHunterRuns') },
  { value: 'profile.document.extract.v1', label: t('tasks.kinds.profileDocuments') },
  { value: 'template.extract.v1', label: t('tasks.kinds.templates') },
  { value: 'generation.run.v1', label: t('tasks.kinds.generations') },
])
const activeCount = computed(() => (counts.value.queued ?? 0) + (counts.value.running ?? 0) + (counts.value.retry_wait ?? 0))

function kindLabel(kind: string) {
  return kinds.value.find(item => item.value === kind)?.label.replace(/s$/, '') ?? kind
}

function formatTime(value?: string) {
  return value ? new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'medium' }).format(new Date(value)) : '—'
}

function stateLabel(state: BackgroundTaskState) {
  return ({ queued: t('tasks.states.queued'), running: t('tasks.states.running'), retry_wait: t('tasks.states.retryWait'), succeeded: t('tasks.states.succeeded'), failed: t('tasks.states.failed'), cancelled: t('tasks.states.cancelled') } as Record<string, string>)[state]
}

async function load(showLoading = true) {
  if (showLoading) loading.value = true
  try {
    const result = await api.listTasks({ search: search.value.trim(), state: stateFilter.value, kind: kindFilter.value, page: page.value, pageSize: pageSize.value })
    tasks.value = result.items
    counts.value = result.counts
    total.value = result.total
    error.value = ''
    if (!selected.value && result.items.length) {
      await openTask(result.items[0], false)
    } else if (selected.value) {
      const current = result.items.find(item => item.id === selected.value?.id)
      if (current && current.updatedAt !== selected.value.updatedAt) await openTask(current, false)
    }
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('tasks.errors.loadTasks')
  } finally {
    loading.value = false
  }
}

async function openTask(task: BackgroundTask, showLoading = true) {
  if (showLoading) detailLoading.value = true
  try {
    selected.value = await api.getTask(task.id)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('tasks.errors.loadTaskDetails')
  } finally {
    detailLoading.value = false
  }
}

function resetPageAndLoad() {
  if (page.value !== 1) page.value = 1
  else void load()
}

watch([stateFilter, kindFilter, pageSize], resetPageAndLoad)
watch(page, () => { void load() })
watch(search, () => {
  if (searchTimer) window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(resetPageAndLoad, 250)
})

onMounted(async () => {
  await load()
  pollTimer = window.setInterval(() => { void load(false) }, 3000)
})
onBeforeUnmount(() => {
  if (pollTimer) window.clearInterval(pollTimer)
  if (searchTimer) window.clearTimeout(searchTimer)
})
</script>

<template>
  <div class="page task-monitor-page">
    <PageHeader :title="t('pages.tasks.title')" :description="t('pages.tasks.description')" />

    <section class="task-metrics" :aria-label="t('tasks.metrics.label')">
      <article><span>{{ t('tasks.metrics.active') }}</span><strong>{{ activeCount }}</strong><small>{{ t('tasks.metrics.activeHint') }}</small></article>
      <article><span>{{ t('tasks.states.succeeded') }}</span><strong>{{ counts.succeeded ?? 0 }}</strong><small>{{ t('tasks.metrics.completedHint') }}</small></article>
      <article :class="{ attention: (counts.failed ?? 0) > 0 }"><span>{{ t('tasks.states.failed') }}</span><strong>{{ counts.failed ?? 0 }}</strong><small>{{ t('common.needsAttention') }}</small></article>
    </section>

    <ListFilters v-model:search="search" :total="total" :search-placeholder="t('tasks.filters.searchPlaceholder')">
      <label>{{ t('tasks.filters.status') }} <select v-model="stateFilter"><option value="all">{{ t('tasks.filters.allStatuses') }}</option><option value="queued">{{ t('tasks.states.queued') }}</option><option value="running">{{ t('tasks.states.running') }}</option><option value="retry_wait">{{ t('tasks.states.retryWait') }}</option><option value="succeeded">{{ t('tasks.states.succeeded') }}</option><option value="failed">{{ t('tasks.states.failed') }}</option><option value="cancelled">{{ t('tasks.states.cancelled') }}</option></select></label>
      <label>{{ t('tasks.filters.type') }} <select v-model="kindFilter"><option value="all">{{ t('tasks.filters.allTypes') }}</option><option v-for="kind in kinds" :key="kind.value" :value="kind.value">{{ kind.label }}</option></select></label>
    </ListFilters>

    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <div v-if="loading" class="empty-state compact">{{ t('tasks.loadingList') }}</div>
    <div v-else class="task-layout">
      <section class="list-panel task-list" :aria-label="t('tasks.list.ariaLabel')">
        <button v-for="task in tasks" :key="task.id" class="task-row" :class="{ selected: selected?.id === task.id }" type="button" @click="openTask(task)">
          <span class="task-kind-icon">{{ kindLabel(task.kind).slice(0, 1) }}</span>
          <span class="task-summary"><strong>{{ kindLabel(task.kind) }}</strong><small>{{ task.id }}</small><em v-if="task.errorMessage">{{ task.errorMessage }}</em></span>
          <span><span class="status-pill" :class="task.state">{{ stateLabel(task.state) }}</span><small>{{ formatTime(task.updatedAt) }}</small></span>
        </button>
        <div v-if="!tasks.length" class="empty-state compact"><h2>{{ t('tasks.empty.title') }}</h2><p>{{ t('tasks.empty.description') }}</p></div>
      </section>

      <aside class="panel task-detail" aria-live="polite">
        <div v-if="detailLoading" class="empty-state compact">{{ t('tasks.loadingDetail') }}</div>
        <template v-else-if="selected">
          <header><div><p class="eyebrow">{{ kindLabel(selected.kind) }}</p><h2>{{ t('tasks.detail.title') }}</h2></div><span class="status-pill" :class="selected.state">{{ stateLabel(selected.state) }}</span></header>
          <dl class="task-facts">
            <div><dt>{{ t('tasks.detail.fields.taskId') }}</dt><dd>{{ selected.id }}</dd></div>
            <div><dt>{{ t('tasks.detail.fields.attempts') }}</dt><dd>{{ selected.attempt }} / {{ selected.maxAttempts }}</dd></div>
            <div><dt>{{ t('tasks.detail.fields.created') }}</dt><dd>{{ formatTime(selected.createdAt) }}</dd></div>
            <div><dt>{{ t('tasks.detail.fields.updated') }}</dt><dd>{{ formatTime(selected.updatedAt) }}</dd></div>
            <div><dt>{{ t('tasks.detail.fields.availableFrom') }}</dt><dd>{{ formatTime(selected.availableAt) }}</dd></div>
            <div v-if="selected.deadlineAt"><dt>{{ t('tasks.detail.fields.deadline') }}</dt><dd>{{ formatTime(selected.deadlineAt) }}</dd></div>
            <div v-if="selected.heartbeatAt"><dt>{{ t('tasks.detail.fields.lastHeartbeat') }}</dt><dd>{{ formatTime(selected.heartbeatAt) }}</dd></div>
            <div v-if="selected.errorClass"><dt>{{ t('tasks.detail.fields.errorClass') }}</dt><dd class="error-text">{{ selected.errorClass }}</dd></div>
            <div v-if="selected.errorMessage"><dt>{{ t('tasks.detail.fields.error') }}</dt><dd class="error-text">{{ selected.errorMessage }}</dd></div>
          </dl>
          <section><h3>{{ t('tasks.detail.input') }}</h3><pre>{{ JSON.stringify(selected.payload, null, 2) }}</pre></section>
          <section><h3>{{ t('tasks.detail.activityLog') }}</h3><ol class="task-events"><li v-for="event in selected.events" :key="event.id" :class="event.eventType"><span /><div><strong>{{ t(`tasks.eventType.${event.eventType}`) }}</strong><p>{{ event.message }}</p><small>{{ formatTime(event.occurredAt) }} · {{ t('tasks.detail.eventAttempt', { attempt: event.attempt }) }}<template v-if="event.errorClass"> · {{ event.errorClass }}</template></small></div></li></ol></section>
        </template>
        <div v-else class="empty-state compact"><span class="empty-icon">◴</span><h2>{{ t('tasks.emptyDetail.title') }}</h2><p>{{ t('tasks.emptyDetail.description') }}</p></div>
      </aside>
    </div>
    <ListPagination v-if="total" v-model:page="page" v-model:page-size="pageSize" :total="total" :page-sizes="[20, 50, 100]" />
  </div>
</template>

<style scoped>
.task-monitor-page { display: grid; gap: 1.25rem; }
.task-monitor-page :deep(.page-header) { margin-bottom: .5rem; }
.task-metrics { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; }
.task-metrics article { display: grid; gap: 5px; padding: 18px 20px; border: 1px solid var(--line); border-radius: 13px; background: var(--surface-soft); }
.task-metrics span, .task-metrics small { color: var(--muted); font-size: 11px; }
.task-metrics strong { font-size: 27px; }
.task-metrics article.attention { border-color: color-mix(in srgb, var(--danger) 30%, var(--line)); }
.task-metrics article.attention strong { color: var(--danger); }
.task-layout { display: grid; grid-template-columns: minmax(0, 1.15fr) minmax(340px, .85fr); gap: 18px; align-items: start; }
.task-list { max-height: 720px; overflow-y: auto; }
.task-row { display: grid; width: 100%; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: 14px; padding: 17px 18px; border: 0; border-bottom: 1px solid var(--line); background: transparent; color: var(--ink); cursor: pointer; text-align: left; transition: background .18s ease; }
.task-row:last-child { border-bottom: 0; }
.task-row:hover, .task-row.selected { background: var(--surface-soft); }
.task-row.selected { box-shadow: inset 3px 0 var(--accent); }
.task-kind-icon { display: grid; width: 34px; height: 34px; place-items: center; border-radius: 9px; background: var(--accent-pale); color: var(--accent-dark); font-size: 12px; font-weight: 800; }
.task-summary { display: grid; min-width: 0; gap: 4px; }
.task-summary strong { font-size: 13px; }
.task-summary small, .task-row > span:last-child small { display: block; overflow: hidden; color: var(--muted); font-size: 10px; text-overflow: ellipsis; white-space: nowrap; }
.task-summary em { overflow: hidden; color: var(--danger); font-size: 10px; font-style: normal; text-overflow: ellipsis; white-space: nowrap; }
.task-row > span:last-child { display: grid; justify-items: end; gap: 7px; }
.status-pill.running, .status-pill.queued, .status-pill.retry_wait { background: #fff2cf; color: #795b16; }
.status-pill.succeeded { background: var(--accent-pale); }
.status-pill.failed { background: #f8e6e6; color: var(--danger); }
.task-detail { position: sticky; top: 24px; display: grid; min-width: 0; max-width: 100%; gap: 22px; max-height: 720px; overflow-y: auto; }
.task-detail header { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; }
.task-detail h2, .task-detail .eyebrow { margin-bottom: 0; }
.task-detail section { min-width: 0; max-width: 100%; }
.task-detail h3 { margin: 0 0 10px; font-size: 13px; }
.task-facts { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; margin: 0; }
.task-facts div { min-width: 0; }
.task-facts dt { margin-bottom: 3px; color: var(--muted); font-size: 10px; font-weight: 700; text-transform: uppercase; }
.task-facts dd { margin: 0; overflow-wrap: anywhere; font-size: 12px; }
pre { box-sizing: border-box; width: 100%; min-width: 0; max-width: 100%; max-height: 190px; margin: 0; padding: 13px; overflow: auto; border-radius: 9px; background: #17201d; color: #e8f0eb; font-size: 10px; line-height: 1.55; overflow-wrap: anywhere; white-space: pre-wrap; }
.task-events { display: grid; gap: 0; margin: 0; padding: 0; list-style: none; }
.task-events li { position: relative; display: grid; grid-template-columns: 18px 1fr; gap: 10px; padding-bottom: 17px; }
.task-events li:not(:last-child)::before { position: absolute; top: 12px; bottom: -1px; left: 5px; width: 1px; background: var(--line); content: ''; }
.task-events li > span { z-index: 1; width: 11px; height: 11px; margin-top: 2px; border: 2px solid var(--surface); border-radius: 50%; background: var(--accent); box-shadow: 0 0 0 1px var(--accent); }
.task-events li.failed > span { background: var(--danger); box-shadow: 0 0 0 1px var(--danger); }
.task-events strong { font-size: 11px; text-transform: capitalize; }
.task-events p { margin: 3px 0; font-size: 11px; line-height: 1.45; }
.task-events small { color: var(--muted); font-size: 9px; }
@media (max-width: 900px) { .task-layout { grid-template-columns: 1fr; } .task-detail { position: static; max-height: none; } }
@media (max-width: 620px) { .task-metrics { grid-template-columns: 1fr; } .task-facts { grid-template-columns: 1fr; } }
</style>
