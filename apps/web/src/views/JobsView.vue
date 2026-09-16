<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'

import ConfirmDialog from '../components/ConfirmDialog.vue'
import ListFilters from '../components/ListFilters.vue'
import ListPagination from '../components/ListPagination.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { toast } from '../lib/toast'
import type { Job, JobInput } from '../lib/types'

const jobs = ref<Job[]>([])
const loading = ref(true)
const busy = ref(false)
const deletingId = ref('')
const pendingDelete = ref<Job | null>(null)
const error = ref('')
const batchText = ref('')
const showBatch = ref(false)
const showManual = ref(false)
const importsAvailable = ref(true)
const search = ref('')
const statusFilter = ref('all')
const page = ref(1)
const pageSize = ref(10)
const manual = reactive<JobInput>({
  title: '', company: '', location: '', country: '', city: '', workMode: '', employmentType: '',
  sourceUrl: '', description: '', status: 'interested',
})
let pollTimer: number | undefined

const pending = computed(() => jobs.value.some(item => ['queued', 'fetching'].includes(item.importState)))
const statusOptions = computed(() => [...new Set(jobs.value.map(item => item.status).filter(Boolean))].sort())
const filteredJobs = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  return jobs.value.filter(item => {
    const matchesSearch = !query || [item.title, item.company, item.location, item.city, item.country, item.description].some(value => value?.toLocaleLowerCase().includes(query))
    return matchesSearch && (statusFilter.value === 'all' || item.status === statusFilter.value)
  })
})
const visibleJobs = computed(() => filteredJobs.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
watch([search, statusFilter, pageSize], () => { page.value = 1 })
watch(() => filteredJobs.value.length, total => { page.value = Math.min(page.value, Math.max(1, Math.ceil(total / pageSize.value))) })

async function load(showLoading = true) {
  if (showLoading) loading.value = true
  try {
    jobs.value = (await api.listJobs()).items
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not load jobs.'
  } finally {
    loading.value = false
  }
}

function mergeJobs(items: Job[]) {
  const byId = new Map(jobs.value.map(item => [item.id, item]))
  for (const item of items) byId.set(item.id, item)
  jobs.value = [...byId.values()].sort((left, right) => right.createdAt.localeCompare(left.createdAt))
}

async function importURLs(urls: string[]) {
  busy.value = true
  error.value = ''
  try {
    const result = await api.importJobs(urls)
    mergeJobs(result.items)
    toast.success(result.items.length === 1 ? 'Job added or already tracked.' : `${result.items.length} jobs added or already tracked.`)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not queue job imports.'
  } finally {
    busy.value = false
  }
}

async function importBatch() {
  const urls = batchText.value.split(/[\s,]+/).map(value => value.trim()).filter(Boolean)
  await importURLs(urls)
  if (!error.value) {
    batchText.value = ''
    showBatch.value = false
  }
}

async function createManual() {
  busy.value = true
  error.value = ''
  try {
    const created = await api.createJob({ ...manual })
    mergeJobs([created])
    Object.assign(manual, { title: '', company: '', location: '', country: '', city: '', workMode: '', employmentType: '', sourceUrl: '', description: '', status: 'interested' })
    showManual.value = false
    toast.success('Job added manually.')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not create the job.'
  } finally {
    busy.value = false
  }
}

function importLabel(item: Job) {
  return ({ queued: 'Queued', fetching: 'Importing', needs_user_action: 'Needs editing', failed: 'Import failed' } as Record<string, string>)[item.importState]
}

async function removeJob() {
  const item = pendingDelete.value
  if (!item) return
  deletingId.value = item.id
  error.value = ''
  try {
    await api.deleteJob(item.id)
    jobs.value = jobs.value.filter(candidate => candidate.id !== item.id)
    pendingDelete.value = null
    toast.success('Opportunity deleted.')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not delete the opportunity.'
  } finally {
    deletingId.value = ''
  }
}

onMounted(async () => {
  const [, capabilities] = await Promise.all([load(), api.capabilities().catch(() => undefined)])
  importsAvailable.value = capabilities?.features.jobImports ?? true
  pollTimer = window.setInterval(() => { if (pending.value) void load(false) }, 2000)
})
onBeforeUnmount(() => { if (pollTimer) window.clearInterval(pollTimer) })
</script>

<template>
  <div class="page jobs-page">
    <PageHeader title="Job Opportunities" description="Track interesting roles and import public LinkedIn or Indeed job pages.">
      <div class="header-actions">
        <button class="button" type="button" @click="showManual = !showManual; showBatch = false">{{ showManual ? 'Close' : 'Add manually' }}</button>
        <button v-if="importsAvailable" class="button primary" type="button" @click="showBatch = !showBatch; showManual = false">{{ showBatch ? 'Close import' : 'Import via URLs' }}</button>
      </div>
    </PageHeader>

    <form v-if="showBatch" class="panel form-grid" @submit.prevent="importBatch">
      <label class="full"><span>Opportunity URLs</span><textarea v-model="batchText" required rows="8" placeholder="Paste up to 50 LinkedIn or Indeed URLs, one per line. A single URL works too." /></label>
      <div class="full form-actions"><button class="button primary" :disabled="busy">{{ busy ? 'Queuing…' : 'Import opportunities' }}</button></div>
    </form>

    <form v-if="showManual" class="panel form-grid" @submit.prevent="createManual">
      <label><span>Job title</span><input v-model="manual.title" required maxlength="300" /></label>
      <label><span>Company</span><input v-model="manual.company" required maxlength="300" /></label>
      <label><span>City</span><input v-model="manual.city" maxlength="150" /></label>
      <label><span>Country</span><input v-model="manual.country" maxlength="100" /></label>
      <label><span>Work mode</span><input v-model="manual.workMode" maxlength="100" placeholder="Remote, hybrid, or on-site" /></label>
      <label><span>Employment type</span><input v-model="manual.employmentType" maxlength="100" placeholder="Full-time" /></label>
      <label class="full"><span>Source URL</span><input v-model="manual.sourceUrl" type="url" maxlength="2048" placeholder="https://…" /></label>
      <label class="full"><span>Description</span><textarea v-model="manual.description" rows="8" /></label>
      <div class="full form-actions"><button class="button primary" :disabled="busy">Add job</button></div>
    </form>

    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <div v-if="loading" class="empty-state">Loading jobs…</div>
    <template v-else-if="jobs.length">
      <ListFilters v-model:search="search" :total="filteredJobs.length" search-placeholder="Search roles, companies, or locations…">
        <label>Status <select v-model="statusFilter"><option value="all">All statuses</option><option v-for="status in statusOptions" :key="status" :value="status">{{ status }}</option></select></label>
      </ListFilters>
    <TransitionGroup v-if="visibleJobs.length" name="card-list" tag="div" class="list-panel">
      <article v-for="item in visibleJobs" :key="item.id" class="job-row">
        <span class="company-mark">{{ (item.company || '?').slice(0, 2).toUpperCase() }}</span>
        <div class="job-main">
          <h2>{{ item.title || 'Importing job details…' }}</h2>
          <p>{{ item.company || 'Company pending' }}<span v-if="item.location"> · {{ item.location }}</span><span v-else-if="item.city || item.country"> · {{ [item.city, item.country].filter(Boolean).join(', ') }}</span></p>
          <small v-if="importLabel(item)" :class="{ 'import-warning': item.importState === 'needs_user_action' || item.importState === 'failed' }">{{ importLabel(item) }}</small>
        </div>
        <span class="status-pill">{{ item.status }}</span>
        <div class="row-actions"><RouterLink class="text-button" :to="`/jobs/${item.id}`">View →</RouterLink><a v-if="item.sourceUrl" class="source-link" :href="item.sourceUrl" target="_blank" rel="noopener noreferrer">Source ↗</a><button class="text-button danger-text" type="button" @click="pendingDelete = item">Delete</button></div>
      </article>
    </TransitionGroup>
    <div v-else class="empty-state compact"><h2>No matching opportunities</h2><p>Try another keyword or status.</p></div>
    <ListPagination v-if="filteredJobs.length" v-model:page="page" v-model:page-size="pageSize" :total="filteredJobs.length" :page-sizes="[10, 20, 50]" />
    </template>
    <div v-else class="empty-state">
      <span class="empty-icon">◇</span><h2>No tracked roles</h2><p>Paste a LinkedIn or Indeed URL above, import a batch, or add a role manually.</p>
    </div>
    <ConfirmDialog :open="Boolean(pendingDelete)" title="Delete opportunity?" :message="`${[pendingDelete?.title, pendingDelete?.company].filter(Boolean).join(' at ') || 'This opportunity'} will be removed from your tracked roles. This action cannot be undone.`" :busy="deletingId === pendingDelete?.id" @cancel="pendingDelete = null" @confirm="removeJob" />
  </div>
</template>

<style scoped>
.jobs-page { display: grid; gap: 1.25rem; }
.jobs-page :deep(.page-header) { margin-bottom: .5rem; }
.header-actions, .row-actions { display: flex; align-items: center; gap: .6rem; }
.job-main small { display: inline-block; margin-top: .4rem; color: var(--accent); font-weight: 700; }
.job-main small.import-warning { color: var(--danger); }
.source-link { color: var(--muted); font-size: .82rem; font-weight: 600; }
@media (max-width: 760px) { .header-actions, .row-actions { align-items: stretch; flex-direction: column; } }
</style>
