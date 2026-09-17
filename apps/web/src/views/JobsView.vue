<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

import ConfirmDialog from '../components/ConfirmDialog.vue'
import AttentionNotice from '../components/AttentionNotice.vue'
import BulkSelectionBar from '../components/BulkSelectionBar.vue'
import DocumentGenerationDialog from '../components/DocumentGenerationDialog.vue'
import ListFilters from '../components/ListFilters.vue'
import ListPagination from '../components/ListPagination.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { jobStatuses, jobStatusLabel } from '../lib/jobStatus'
import { useListSelection } from '../lib/listSelection'
import { toast } from '../lib/toast'
import type { AgentDefault, Job, JobInput, LLMConnection, TemplateKind } from '../lib/types'

const jobs = ref<Job[]>([])
const route = useRoute()
const loading = ref(true)
const busy = ref(false)
const deletingId = ref('')
const updatingStatusId = ref('')
const pendingDelete = ref<Job | null>(null)
const error = ref('')
const batchText = ref('')
const showBatch = ref(false)
const showManual = ref(false)
const aiAssisted = ref(false)
const connections = ref<LLMConnection[]>([])
const importConnectionId = ref('')
const importModel = ref('')
const importModels = reactive<Record<string, string[]>>({})
const loadingImportModels = ref(false)
const importsAvailable = ref(true)
const search = ref('')
const statusFilter = ref('all')
const originFilter = ref('all')
const page = ref(1)
const pageSize = ref(10)
const showGeneration = ref(false)
const generationTargets = ref<Job[]>([])
const generationDocumentType = ref<TemplateKind>('resume')
const selection = useListSelection()
const manual = reactive<JobInput>({
  title: '', company: '', location: '', country: '', city: '', workMode: '', employmentType: '',
  sourceUrl: '', description: '', status: 'interested',
})
const manualReviewHunterId = ref('')
const manualReviewId = ref('')
let pollTimer: number | undefined

const pending = computed(() => jobs.value.some(item => ['queued', 'fetching', 'analyzing'].includes(item.importState)))
const statusOptions = jobStatuses
const filteredJobs = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  return jobs.value.filter(item => {
    const matchesSearch = !query || [item.title, item.company, item.location, item.city, item.country, item.description].some(value => value?.toLocaleLowerCase().includes(query))
    return matchesSearch && (statusFilter.value === 'all' || item.status === statusFilter.value) && (originFilter.value === 'all' || item.origin === originFilter.value)
  })
})
const visibleJobs = computed(() => filteredJobs.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
const selectableFilteredJobs = computed(() => filteredJobs.value.filter(canGenerate))
const allFilteredSelected = computed(() => Boolean(selectableFilteredJobs.value.length) && selectableFilteredJobs.value.every(item => selection.isSelected(item.id)))
watch([search, statusFilter, originFilter, pageSize], () => { page.value = 1 })
watch(() => filteredJobs.value.length, total => { page.value = Math.min(page.value, Math.max(1, Math.ceil(total / pageSize.value))) })
watch(() => jobs.value.map(item => item.id).join(','), () => selection.retain(jobs.value.filter(canGenerate).map(item => item.id)))

function canGenerate(item: Job) {
  return Boolean(item.description?.trim()) && ['manual', 'ready'].includes(item.importState)
}

function openGeneration(items: Job[], documentType: TemplateKind) {
  generationTargets.value = items.filter(canGenerate)
  if (!generationTargets.value.length) {
    toast.warning('Finish importing or add a job description before creating documents.')
    return
  }
  if (generationTargets.value.length > 50) {
    toast.warning('Select no more than 50 job opportunities for one batch.')
    return
  }
  generationDocumentType.value = documentType
  showGeneration.value = true
}

function openSelectedGeneration(documentType: TemplateKind) {
  openGeneration(jobs.value.filter(item => selection.isSelected(item.id)), documentType)
}

function selectAllFiltered() {
  selection.toggleMany(selectableFilteredJobs.value.map(item => item.id), true)
}

function handleGenerationQueued(result: { createdOpportunityIds: string[]; failedOpportunityIds: string[] }) {
  for (const id of result.createdOpportunityIds) selection.toggle(id, false)
  if (result.createdOpportunityIds.length) {
    const label = generationDocumentType.value === 'resume' ? 'CV' : 'cover letter'
    toast.success(`${result.createdOpportunityIds.length} ${label}${result.createdOpportunityIds.length === 1 ? '' : 's'} queued.`)
  }
  if (result.failedOpportunityIds.length) toast.warning(`${result.failedOpportunityIds.length} generation task${result.failedOpportunityIds.length === 1 ? '' : 's'} could not be queued.`)
}

async function load(showLoading = true) {
  if (showLoading) loading.value = true
  try {
    jobs.value = (await api.listJobs()).items
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not load job opportunities.'
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
    const result = await api.importJobs({ urls, aiAssisted: aiAssisted.value, connectionId: aiAssisted.value ? importConnectionId.value : undefined, model: aiAssisted.value ? importModel.value : undefined })
    mergeJobs(result.items)
    toast.success(result.items.length === 1 ? 'Job opportunity added or already tracked.' : `${result.items.length} job opportunities added or already tracked.`)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not queue job opportunity imports.'
  } finally {
    busy.value = false
  }
}

async function discoverImportModels() {
  const connectionId = importConnectionId.value
  importModel.value = ''
  if (!connectionId) return
  if (importModels[connectionId]) {
    importModel.value = importModels[connectionId][0] ?? ''
    return
  }
  loadingImportModels.value = true
  try {
    importModels[connectionId] = (await api.testLLMConnection(connectionId)).models
    importModel.value = importModels[connectionId][0] ?? ''
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not load models for AI-assisted import.'
  } finally {
    loadingImportModels.value = false
  }
}

async function openBatchImport() {
  showBatch.value = !showBatch.value
  showManual.value = false
  if (!showBatch.value) return
  try {
    const [connectionList, storedDefaults] = await Promise.all([api.listLLMConnections(), api.getAgentDefaults()])
    connections.value = connectionList.items
    const configured = storedDefaults.items.find((item: AgentDefault) => item.agent === 'job_import')
    if (configured && connections.value.some(item => item.id === configured.connectionId)) {
      importConnectionId.value = configured.connectionId
      importModel.value = configured.model
      importModels[configured.connectionId] = [configured.model]
    } else if (!connections.value.some(item => item.id === importConnectionId.value)) {
      importConnectionId.value = connections.value[0]?.id ?? ''
      importModel.value = ''
    }
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not load LLM connections.'
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
    if (manualReviewHunterId.value && manualReviewId.value) {
      try {
        await api.dismissJobHunterReviewItem(manualReviewHunterId.value, manualReviewId.value)
      } catch {
        toast.warning('The opportunity was added, but its confirmation reminder could not be dismissed.')
      }
      manualReviewHunterId.value = ''
      manualReviewId.value = ''
    }
    Object.assign(manual, { title: '', company: '', location: '', country: '', city: '', workMode: '', employmentType: '', sourceUrl: '', description: '', status: 'interested' })
    showManual.value = false
    toast.success('Job opportunity added manually.')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not create the job opportunity.'
  } finally {
    busy.value = false
  }
}

function importLabel(item: Job) {
  return ({ queued: 'Queued', fetching: 'Importing', analyzing: 'AI is analyzing this page', needs_user_action: 'Needs editing', failed: 'Import failed' } as Record<string, string>)[item.importState]
}

watch(aiAssisted, value => { if (value && importConnectionId.value) void discoverImportModels() })
watch(importConnectionId, () => { if (aiAssisted.value) void discoverImportModels() })

async function removeJob() {
  const item = pendingDelete.value
  if (!item) return
  deletingId.value = item.id
  error.value = ''
  try {
    await api.deleteJob(item.id)
    jobs.value = jobs.value.filter(candidate => candidate.id !== item.id)
    pendingDelete.value = null
    toast.success('Job opportunity deleted.')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not delete the job opportunity.'
  } finally {
    deletingId.value = ''
  }
}

async function updateStatus(item: Job, event: Event) {
  const select = event.currentTarget as HTMLSelectElement
  const nextStatus = select.value
  const previousStatus = item.status
  if (!nextStatus || nextStatus === previousStatus) return
  updatingStatusId.value = item.id
  item.status = nextStatus
  try {
    const updated = await api.updateJob(item.id, {
      title: item.title, company: item.company, location: item.location ?? '', country: item.country ?? '',
      city: item.city ?? '', workMode: item.workMode ?? '', employmentType: item.employmentType ?? '',
      sourceUrl: item.sourceUrl ?? '', description: item.description ?? '', status: nextStatus,
    })
    Object.assign(item, updated)
    toast.success(`Status updated to ${jobStatusLabel(nextStatus)}.`)
  } catch (cause) {
    item.status = previousStatus
    select.value = previousStatus
    toast.warning(cause instanceof Error ? cause.message : 'Could not update the job status.')
  } finally {
    updatingStatusId.value = ''
  }
}

onMounted(async () => {
  const manualURL = typeof route.query.manualUrl === 'string' ? route.query.manualUrl : ''
  if (manualURL) {
    manual.sourceUrl = manualURL
    manualReviewHunterId.value = typeof route.query.hunterId === 'string' ? route.query.hunterId : ''
    manualReviewId.value = typeof route.query.reviewId === 'string' ? route.query.reviewId : ''
    showManual.value = true
  }
  const [, capabilities] = await Promise.all([load(), api.capabilities().catch(() => undefined)])
  importsAvailable.value = capabilities?.features.jobImports ?? true
  pollTimer = window.setInterval(() => { if (pending.value) void load(false) }, 2000)
})
onBeforeUnmount(() => { if (pollTimer) window.clearInterval(pollTimer) })
</script>

<template>
  <div class="page jobs-page">
    <PageHeader title="Job Opportunities" description="Explore your next career move and manage every job opportunity in one place.">
      <div class="header-actions">
        <button class="button" type="button" @click="showManual = !showManual; showBatch = false">{{ showManual ? 'Close' : 'Add manually' }}</button>
        <button v-if="importsAvailable" class="button primary" type="button" @click="openBatchImport">{{ showBatch ? 'Close import' : 'Import via URLs' }}</button>
      </div>
    </PageHeader>

    <form v-if="showBatch" class="panel form-grid" @submit.prevent="importBatch">
      <div class="full ai-import-toggle">
        <label class="switch-row"><input v-model="aiAssisted" type="checkbox" /><span class="switch-track" aria-hidden="true"><span /></span><span>Enable AI assistance</span></label>
        <span class="help-tooltip" tabindex="0" aria-label="About AI-assisted import">?<span role="tooltip">Uses an AI agent to open public job pages, reveal expandable content, and extract job details. It may take longer and use your selected LLM connection. Sign-in pages, CAPTCHAs, and restricted content cannot be bypassed.</span></span>
      </div>
      <p class="full import-mode-help">{{ aiAssisted ? 'The Job Import Agent will analyze each publicly accessible HTTPS page from the beginning.' : 'Fast import for public LinkedIn and Indeed job pages.' }}</p>
      <template v-if="aiAssisted">
        <p v-if="!connections.length" class="full notice setup-notice">Add an <RouterLink to="/settings">LLM connection in Settings</RouterLink> before using AI-assisted import.</p>
        <label><span>LLM connection</span><select v-model="importConnectionId" required><option value="" disabled>Select connection</option><option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option></select></label>
        <label><span>Model</span><input v-model="importModel" required :disabled="loadingImportModels" list="job-import-models" :placeholder="loadingImportModels ? 'Loading models…' : 'Model name'" /><datalist id="job-import-models"><option v-for="model in importModels[importConnectionId] || []" :key="model" :value="model" /></datalist></label>
      </template>
      <label class="full"><span>Job opportunity URLs</span><textarea v-model="batchText" required rows="8" :placeholder="aiAssisted ? 'Paste up to 50 public HTTPS job page URLs, one per line. A single URL works too.' : 'Paste up to 50 LinkedIn or Indeed URLs, one per line. A single URL works too.'" /></label>
      <div class="full form-actions"><button class="button primary" :disabled="busy || (aiAssisted && (!importConnectionId || !importModel))">{{ busy ? 'Queuing…' : 'Import job opportunities' }}</button></div>
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
      <div class="full form-actions"><button class="button primary" :disabled="busy">Add job opportunity</button></div>
    </form>

    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <div v-if="loading" class="empty-state">Loading job opportunities…</div>
    <template v-else-if="jobs.length">
      <ListFilters v-model:search="search" :total="filteredJobs.length" search-placeholder="Search roles, companies, or locations…">
        <label>Status <select v-model="statusFilter"><option value="all">All statuses</option><option v-for="status in statusOptions" :key="status.value" :value="status.value">{{ status.label }}</option></select></label>
        <label>Source <select v-model="originFilter"><option value="all">All sources</option><option value="manual">Added manually</option><option value="url_import">Imported via URL</option><option value="hunter">Found by Job Hunter</option></select></label>
      </ListFilters>
    <BulkSelectionBar :selected-count="selection.selectedCount.value" :all-count="selectableFilteredJobs.length" :all-selected="allFilteredSelected" @select-all="selectAllFiltered" @clear="selection.clear">
      <button class="button primary" type="button" @click="openSelectedGeneration('resume')">Create CVs</button>
      <button class="button" type="button" @click="openSelectedGeneration('cover_letter')">Create cover letters</button>
    </BulkSelectionBar>
    <TransitionGroup v-if="visibleJobs.length" name="card-list" tag="div" class="list-panel">
      <article v-for="item in visibleJobs" :key="item.id" class="job-row" :class="{ selected: selection.isSelected(item.id) }">
        <input class="row-selector" type="checkbox" :checked="selection.isSelected(item.id)" :disabled="!canGenerate(item)" :aria-label="`Select ${item.title || 'job opportunity'}`" :title="canGenerate(item) ? 'Select for a batch action' : 'Add a job description before creating documents'" @change="selection.toggle(item.id)" />
        <span class="company-mark">{{ (item.company || '?').slice(0, 2).toUpperCase() }}</span>
        <div class="job-main">
          <h2>{{ item.title || 'Importing job details…' }}</h2>
          <p>{{ item.company || 'Company pending' }}<span v-if="item.location"> · {{ item.location }}</span><span v-else-if="item.city || item.country"> · {{ [item.city, item.country].filter(Boolean).join(', ') }}</span></p>
          <small v-if="importLabel(item)" :class="{ 'import-warning': item.importState === 'needs_user_action' || item.importState === 'failed' }">{{ importLabel(item) }}</small>
          <small v-if="item.origin === 'hunter'" class="hunter-source">Found by Job Hunter</small>
          <AttentionNotice v-if="item.importState === 'needs_user_action' || item.importState === 'failed'" compact :message="item.importError || 'ResumeGPT could not extract complete job details from this page. Open the job opportunity to enter or correct the missing information.'" />
        </div>
        <label class="status-control" :class="item.status" :aria-label="`Change status for ${item.title}`"><select :value="item.status" :disabled="updatingStatusId === item.id" @change="updateStatus(item, $event)"><option v-for="status in jobStatuses" :key="status.value" :value="status.value">{{ status.label }}</option></select><span aria-hidden="true">⌄</span></label>
        <div class="row-actions"><button class="text-button" type="button" :disabled="!canGenerate(item)" @click="openGeneration([item], 'resume')">Create CV</button><button class="text-button" type="button" :disabled="!canGenerate(item)" @click="openGeneration([item], 'cover_letter')">Create cover letter</button><RouterLink class="text-button" :to="`/jobs/${item.id}`">View</RouterLink><a v-if="item.sourceUrl" class="source-link" :href="item.sourceUrl" target="_blank" rel="noopener noreferrer">Source ↗</a><button class="text-button danger-text" type="button" @click="pendingDelete = item">Delete</button></div>
      </article>
    </TransitionGroup>
    <div v-else class="empty-state compact"><h2>No matching job opportunities</h2><p>Try another keyword or status.</p></div>
    <ListPagination v-if="filteredJobs.length" v-model:page="page" v-model:page-size="pageSize" :total="filteredJobs.length" :page-sizes="[10, 20, 50]" />
    </template>
    <div v-else class="empty-state">
      <span class="empty-icon">◇</span><h2>No job opportunities yet</h2><p>Import a LinkedIn or Indeed URL, or add a role manually.</p>
    </div>
    <ConfirmDialog :open="Boolean(pendingDelete)" title="Delete job opportunity?" :message="`${[pendingDelete?.title, pendingDelete?.company].filter(Boolean).join(' at ') || 'This job opportunity'} will be removed from your tracked roles. This action cannot be undone.`" :busy="deletingId === pendingDelete?.id" @cancel="pendingDelete = null" @confirm="removeJob" />
    <DocumentGenerationDialog :open="showGeneration" :opportunities="generationTargets" :initial-document-type="generationDocumentType" @close="showGeneration = false" @queued="handleGenerationQueued" />
  </div>
</template>

<style scoped>
.jobs-page { display: grid; gap: 1.25rem; }
.jobs-page :deep(.page-header) { margin-bottom: .5rem; }
.header-actions, .row-actions { display: flex; align-items: center; gap: .6rem; }
.job-row { grid-template-columns: auto auto minmax(0, 1fr) auto minmax(250px, auto); transition: background .18s ease, box-shadow .18s ease; }
.job-row.selected { background: color-mix(in srgb, var(--accent-pale) 55%, white); box-shadow: inset 3px 0 var(--accent); }
.row-selector { width: 17px; height: 17px; accent-color: var(--accent); cursor: pointer; }
.row-selector:disabled { cursor: not-allowed; opacity: .35; }
.row-actions { flex-wrap: wrap; justify-content: flex-end; }
.row-actions .text-button:disabled { cursor: not-allowed; opacity: .4; }
.status-control { position: relative; display: inline-flex; align-items: center; width: fit-content; border-radius: 999px; background: var(--surface-soft); color: var(--accent-dark); }
.status-control select { width: auto; min-width: 0; padding: 5px 25px 5px 10px; appearance: none; border: 0; border-radius: inherit; outline: 0; background: transparent; color: inherit; cursor: pointer; font: inherit; font-size: 11px; font-weight: 700; text-transform: capitalize; }
.status-control > span { position: absolute; right: 9px; line-height: 1; pointer-events: none; transform: translateY(-1px); }
.status-control:focus-within { box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 18%, transparent); }
.status-control:has(select:disabled) { cursor: wait; opacity: .58; }
.status-control.accepted { background: var(--accent-pale); color: var(--accent-dark); }
.status-control.rejected, .status-control.withdrawn { background: #f8e6e6; color: var(--danger); }
.status-control.interview, .status-control.offer { background: #fff2cf; color: #795b16; }
.job-main small { display: inline-block; margin-top: .4rem; color: var(--accent); font-weight: 700; }
.job-main small.import-warning { color: var(--danger); }
.job-main small.hunter-source { margin-left: .55rem; color: var(--muted); }
.source-link { color: var(--muted); font-size: .82rem; font-weight: 600; }
.ai-import-toggle { display: flex; align-items: center; gap: 9px; }
.switch-row { display: flex !important; grid-template-columns: auto auto auto; align-items: center; gap: 9px !important; cursor: pointer; }
.switch-row > input { position: absolute; width: 1px; height: 1px; opacity: 0; }
.switch-track { display: flex; width: 40px; height: 23px; padding: 3px; border-radius: 99px; background: #aeb8b3; transition: background .18s ease; }
.switch-track > span { width: 17px; height: 17px; border-radius: 50%; background: white; box-shadow: 0 1px 4px rgba(0,0,0,.2); transition: transform .18s ease; }
.switch-row > input:checked + .switch-track { background: var(--accent); }
.switch-row > input:checked + .switch-track > span { transform: translateX(17px); }
.switch-row > input:focus-visible + .switch-track { box-shadow: 0 0 0 3px rgba(61, 107, 87, .18); }
.help-tooltip { position: relative; display: grid; width: 19px; height: 19px; place-items: center; border: 1px solid var(--line); border-radius: 50%; color: var(--muted); cursor: help; font-size: 11px; font-weight: 800; }
.help-tooltip > span { position: absolute; z-index: 10; top: calc(100% + 9px); left: 50%; width: min(340px, 75vw); padding: 11px 13px; border: 1px solid var(--line); border-radius: 9px; background: var(--surface); box-shadow: 0 14px 36px rgba(25,34,30,.16); color: var(--ink); font-size: 11px; font-weight: 500; line-height: 1.55; opacity: 0; pointer-events: none; transform: translate(-50%, -4px); visibility: hidden; transition: .18s ease; }
.help-tooltip:hover > span, .help-tooltip:focus > span { opacity: 1; transform: translate(-50%, 0); visibility: visible; }
.import-mode-help { margin: -9px 0 0; color: var(--muted); font-size: 12px; }
@media (max-width: 980px) { .job-row { grid-template-columns: auto auto minmax(0, 1fr); } .job-row .status-control, .job-row .row-actions { grid-column: 3; } .row-actions { justify-content: flex-start; } }
@media (max-width: 760px) { .header-actions, .row-actions { align-items: stretch; flex-direction: column; } .job-row .status-control { grid-column: 2; } }
</style>
