<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'

import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import type { Job, JobInput } from '../lib/types'

const jobs = ref<Job[]>([])
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const notice = ref('')
const importUrl = ref('')
const batchText = ref('')
const showBatch = ref(false)
const showManual = ref(false)
const importsAvailable = ref(true)
const manual = reactive<JobInput>({
  title: '', company: '', location: '', country: '', city: '', workMode: '', employmentType: '',
  sourceUrl: '', description: '', status: 'interested',
})
let pollTimer: number | undefined

const pending = computed(() => jobs.value.some(item => ['queued', 'fetching'].includes(item.importState)))

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
  notice.value = ''
  try {
    const result = await api.importJobs(urls)
    mergeJobs(result.items)
    notice.value = result.items.length === 1 ? 'Job added or already tracked.' : `${result.items.length} jobs added or already tracked.`
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not queue job imports.'
  } finally {
    busy.value = false
  }
}

async function importSingle() {
  const value = importUrl.value.trim()
  if (!value) return
  await importURLs([value])
  if (!error.value) importUrl.value = ''
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
  notice.value = ''
  try {
    const created = await api.createJob({ ...manual })
    mergeJobs([created])
    Object.assign(manual, { title: '', company: '', location: '', country: '', city: '', workMode: '', employmentType: '', sourceUrl: '', description: '', status: 'interested' })
    showManual.value = false
    notice.value = 'Job added manually.'
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not create the job.'
  } finally {
    busy.value = false
  }
}

function importLabel(item: Job) {
  return ({ queued: 'Queued', fetching: 'Importing', needs_user_action: 'Needs editing', failed: 'Import failed' } as Record<string, string>)[item.importState]
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
        <button v-if="importsAvailable" class="button primary" type="button" @click="showBatch = !showBatch; showManual = false">{{ showBatch ? 'Close batch' : 'Batch import' }}</button>
      </div>
    </PageHeader>

    <form v-if="importsAvailable" class="url-import panel" @submit.prevent="importSingle">
      <div><h2>Import a job from URL</h2><p>Paste a public LinkedIn or Indeed job link. ResumeGPT will fill in the details in the background.</p></div>
      <div class="url-row"><input v-model="importUrl" type="url" required placeholder="https://www.linkedin.com/jobs/view/…" aria-label="LinkedIn or Indeed job URL" /><button class="button primary" :disabled="busy">{{ busy ? 'Queuing…' : 'Import' }}</button></div>
    </form>

    <div v-else class="notice">URL import requires the PostgreSQL worker stack. You can still add and edit jobs manually.</div>

    <form v-if="showBatch" class="panel form-grid" @submit.prevent="importBatch">
      <label class="full"><span>Job URLs</span><textarea v-model="batchText" required rows="8" placeholder="Paste up to 50 LinkedIn or Indeed URLs, one per line." /></label>
      <div class="full form-actions"><button class="button primary" :disabled="busy">Queue batch import</button></div>
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
    <p v-if="notice" class="notice" role="status">{{ notice }}</p>
    <div v-if="loading" class="empty-state">Loading jobs…</div>
    <div v-else-if="jobs.length" class="list-panel">
      <article v-for="item in jobs" :key="item.id" class="job-row">
        <span class="company-mark">{{ (item.company || '?').slice(0, 2).toUpperCase() }}</span>
        <div class="job-main">
          <h2>{{ item.title || 'Importing job details…' }}</h2>
          <p>{{ item.company || 'Company pending' }}<span v-if="item.location"> · {{ item.location }}</span><span v-else-if="item.city || item.country"> · {{ [item.city, item.country].filter(Boolean).join(', ') }}</span></p>
          <small v-if="importLabel(item)" :class="{ 'import-warning': item.importState === 'needs_user_action' || item.importState === 'failed' }">{{ importLabel(item) }}</small>
        </div>
        <span class="status-pill">{{ item.status }}</span>
        <div class="row-actions"><RouterLink class="text-button" :to="`/jobs/${item.id}`">View →</RouterLink><a v-if="item.sourceUrl" class="source-link" :href="item.sourceUrl" target="_blank" rel="noopener noreferrer">Source ↗</a></div>
      </article>
    </div>
    <div v-else class="empty-state">
      <span class="empty-icon">◇</span><h2>No tracked roles</h2><p>Paste a LinkedIn or Indeed URL above, import a batch, or add a role manually.</p>
    </div>
  </div>
</template>

<style scoped>
.jobs-page { display: grid; gap: 1.25rem; }
.jobs-page :deep(.page-header) { margin-bottom: .5rem; }
.header-actions, .row-actions { display: flex; align-items: center; gap: .6rem; }
.url-import { display: grid; grid-template-columns: minmax(240px, .8fr) minmax(360px, 1.2fr); align-items: end; gap: 1.5rem; }
.url-import h2, .url-import p { margin: 0; }
.url-import p { color: var(--muted); line-height: 1.5; }
.url-row { display: grid; grid-template-columns: 1fr auto; gap: .7rem; }
.job-main small { display: inline-block; margin-top: .4rem; color: var(--accent); font-weight: 700; }
.job-main small.import-warning { color: var(--danger); }
.source-link { color: var(--muted); font-size: .82rem; font-weight: 600; }
@media (max-width: 760px) { .url-import { grid-template-columns: 1fr; } .header-actions, .row-actions { align-items: stretch; flex-direction: column; } }
</style>
