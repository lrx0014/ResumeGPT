<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'

const { t } = useI18n()

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
import { usePagedList } from '../lib/pagination'
import { toast } from '../lib/toast'
import type { AgentDefault, Job, JobInput, LLMConnection, TemplateKind } from '../lib/types'

const jobs = ref<Job[]>([])
const route = useRoute()
const loading = ref(true)
const busy = ref(false)
const deletingId = ref('')
const updatingStatusId = ref('')
const pendingDelete = ref<Job | null>(null)
const previewJob = ref<Job | null>(null)
const previewLoading = ref(false)
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
    const matchesSearch = !query || [item.title, item.company, item.location, item.city, item.country].some(value => value?.toLocaleLowerCase().includes(query))
    return matchesSearch && (statusFilter.value === 'all' || item.status === statusFilter.value) && (originFilter.value === 'all' || item.origin === originFilter.value)
  })
})
const { page, pageSize, visible: visibleJobs } = usePagedList(filteredJobs, [search, statusFilter, originFilter], 10)
const selectableFilteredJobs = computed(() => filteredJobs.value.filter(canGenerate))
const allFilteredSelected = computed(() => Boolean(selectableFilteredJobs.value.length) && selectableFilteredJobs.value.every(item => selection.isSelected(item.id)))
watch(() => jobs.value.map(item => item.id).join(','), () => selection.retain(jobs.value.filter(canGenerate).map(item => item.id)))

function canGenerate(item: Job) {
  return Boolean(item.hasDescription || item.description?.trim()) && ['manual', 'ready'].includes(item.importState)
}

function jobLocation(item: Job) {
  return item.location || [item.city, item.country].filter(Boolean).join(', ') || t('jobs.notSpecified')
}

async function openJobPreview(item: Job) {
  previewJob.value = item
  previewLoading.value = true
  try {
    const detail = await api.getJob(item.id)
    if (previewJob.value?.id === item.id) previewJob.value = detail
  } catch (cause) {
    toast.warning(cause instanceof Error ? cause.message : t('jobs.errors.loadDetails'))
  } finally {
    if (previewJob.value?.id === item.id) previewLoading.value = false
  }
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && previewJob.value) previewJob.value = null
}

function openGeneration(items: Job[], documentType: TemplateKind) {
  generationTargets.value = items.filter(canGenerate)
  if (!generationTargets.value.length) {
    toast.warning(t('jobs.errors.finishImportFirst'))
    return
  }
  if (generationTargets.value.length > 50) {
    toast.warning(t('jobs.errors.tooManySelected'))
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
    const key = generationDocumentType.value === 'resume' ? 'jobs.toast.cvsQueued' : 'jobs.toast.coverLettersQueued'
    toast.success(t(key, { count: result.createdOpportunityIds.length }, result.createdOpportunityIds.length))
  }
  if (result.failedOpportunityIds.length) toast.warning(t('jobs.toast.generationFailed', { count: result.failedOpportunityIds.length }, result.failedOpportunityIds.length))
}

async function load(showLoading = true) {
  if (showLoading) loading.value = true
  try {
    jobs.value = (await api.listJobs()).items
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('jobs.errors.load')
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
    toast.success(t('jobs.toast.imported', { count: result.items.length }, result.items.length))
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('jobs.errors.queueImports')
  } finally {
    busy.value = false
  }
}

async function discoverImportModels(resetModel = false) {
  const connectionId = importConnectionId.value
  if (resetModel) importModel.value = ''
  if (!connectionId) return
  if (importModels[connectionId]) {
    if (!importModel.value || !importModels[connectionId].includes(importModel.value)) importModel.value = importModels[connectionId][0] ?? ''
    return
  }
  loadingImportModels.value = true
  try {
    importModels[connectionId] = (await api.testLLMConnection(connectionId)).models
    if (importConnectionId.value === connectionId && (!importModel.value || !importModels[connectionId].includes(importModel.value))) importModel.value = importModels[connectionId][0] ?? ''
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('jobs.errors.loadImportModels')
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
    } else if (!connections.value.some(item => item.id === importConnectionId.value)) {
      importConnectionId.value = connections.value[0]?.id ?? ''
      importModel.value = ''
    }
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('jobs.errors.loadProviders')
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
        toast.warning(t('jobs.errors.reviewDismissFailed'))
      }
      manualReviewHunterId.value = ''
      manualReviewId.value = ''
    }
    Object.assign(manual, { title: '', company: '', location: '', country: '', city: '', workMode: '', employmentType: '', sourceUrl: '', description: '', status: 'interested' })
    showManual.value = false
    toast.success(t('jobs.toast.addedManually'))
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('jobs.errors.create')
  } finally {
    busy.value = false
  }
}

function importLabel(item: Job) {
  const key = ({ queued: 'jobs.importStates.queued', fetching: 'jobs.importStates.fetching', analyzing: 'jobs.importStates.analyzing', needs_user_action: 'jobs.importStates.needsUserAction', failed: 'jobs.importStates.failed' } as Record<string, string>)[item.importState]
  return key ? t(key) : undefined
}

watch(aiAssisted, value => { if (value && importConnectionId.value && !importModel.value) void discoverImportModels(false) })
watch(importConnectionId, () => { if (aiAssisted.value) void discoverImportModels(true) })

async function removeJob() {
  const item = pendingDelete.value
  if (!item) return
  deletingId.value = item.id
  error.value = ''
  try {
    await api.deleteJob(item.id)
    jobs.value = jobs.value.filter(candidate => candidate.id !== item.id)
    pendingDelete.value = null
    toast.success(t('jobs.toast.deleted'))
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('jobs.errors.delete')
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
    await api.updateJobStatus(item.id, nextStatus)
    toast.success(t('jobs.statusUpdated', { status: jobStatusLabel(nextStatus) }))
  } catch (cause) {
    item.status = previousStatus
    select.value = previousStatus
    toast.warning(cause instanceof Error ? cause.message : t('jobs.errors.updateStatus'))
  } finally {
    updatingStatusId.value = ''
  }
}

onMounted(async () => {
  window.addEventListener('keydown', handleKeydown)
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
onBeforeUnmount(() => {
  if (pollTimer) window.clearInterval(pollTimer)
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <div class="page jobs-page">
    <PageHeader :title="t('pages.jobs.title')" :description="t('pages.jobs.description')">
      <div class="header-actions">
        <button class="button" type="button" @click="showManual = !showManual; showBatch = false">{{ showManual ? t('common.close') : t('jobs.addManually') }}</button>
        <button v-if="importsAvailable" class="button primary" type="button" @click="openBatchImport">{{ showBatch ? t('jobs.closeImport') : t('jobs.importViaUrls') }}</button>
      </div>
    </PageHeader>

    <form v-if="showBatch" class="panel form-grid" @submit.prevent="importBatch">
      <div class="full ai-import-toggle">
        <label class="switch-row"><input v-model="aiAssisted" type="checkbox" /><span class="switch-track" aria-hidden="true"><span /></span><span>{{ t('jobs.batchForm.enableAi') }}</span></label>
        <span class="help-tooltip" tabindex="0" :aria-label="t('jobs.batchForm.aiInfoAriaLabel')">?<span role="tooltip">{{ t('jobs.batchForm.aiInfoTooltip') }}</span></span>
      </div>
      <p class="full import-mode-help">{{ aiAssisted ? t('jobs.batchForm.aiModeHelp') : t('jobs.batchForm.standardModeHelp') }}</p>
      <template v-if="aiAssisted">
        <p v-if="!connections.length" class="full notice setup-notice">{{ t('jobs.batchForm.setupNoticeBefore') }}<RouterLink to="/settings">{{ t('jobs.batchForm.setupNoticeLink') }}</RouterLink>{{ t('jobs.batchForm.setupNoticeAfter') }}</p>
        <label><span>{{ t('jobs.batchForm.providerLabel') }}</span><select v-model="importConnectionId" required><option value="" disabled>{{ t('jobs.batchForm.selectProvider') }}</option><option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option></select></label>
        <label><span>{{ t('jobs.batchForm.modelLabel') }}</span><select v-if="importModels[importConnectionId]?.length" v-model="importModel" required><option value="" disabled>{{ t('jobs.batchForm.selectModel') }}</option><option v-for="model in importModels[importConnectionId]" :key="model" :value="model">{{ model }}</option></select><input v-else v-model="importModel" required :disabled="loadingImportModels" :placeholder="loadingImportModels ? t('jobs.batchForm.loadingModels') : t('jobs.batchForm.enterModelName')" /></label>
      </template>
      <label class="full"><span>{{ t('jobs.batchForm.urlsLabel') }}</span><textarea v-model="batchText" required rows="8" :placeholder="aiAssisted ? t('jobs.batchForm.urlsPlaceholderAi') : t('jobs.batchForm.urlsPlaceholderStandard')" /></label>
      <div class="full form-actions"><button class="button primary" :disabled="busy || (aiAssisted && (!importConnectionId || !importModel))">{{ busy ? t('jobs.batchForm.queuing') : t('jobs.batchForm.submit') }}</button></div>
    </form>

    <form v-if="showManual" class="panel form-grid" @submit.prevent="createManual">
      <label><span>{{ t('jobs.manualForm.titleLabel') }}</span><input v-model="manual.title" required maxlength="300" /></label>
      <label><span>{{ t('jobs.manualForm.companyLabel') }}</span><input v-model="manual.company" required maxlength="300" /></label>
      <label><span>{{ t('jobs.manualForm.cityLabel') }}</span><input v-model="manual.city" maxlength="150" /></label>
      <label><span>{{ t('jobs.manualForm.countryLabel') }}</span><input v-model="manual.country" maxlength="100" /></label>
      <label><span>{{ t('jobs.manualForm.workModeLabel') }}</span><input v-model="manual.workMode" maxlength="100" :placeholder="t('jobs.manualForm.workModePlaceholder')" /></label>
      <label><span>{{ t('jobs.manualForm.employmentTypeLabel') }}</span><input v-model="manual.employmentType" maxlength="100" :placeholder="t('jobs.manualForm.employmentTypePlaceholder')" /></label>
      <label class="full"><span>{{ t('jobs.manualForm.sourceUrlLabel') }}</span><input v-model="manual.sourceUrl" type="url" maxlength="2048" :placeholder="t('jobs.manualForm.sourceUrlPlaceholder')" /></label>
      <label class="full"><span>{{ t('jobs.manualForm.descriptionLabel') }}</span><textarea v-model="manual.description" rows="8" /></label>
      <div class="full form-actions"><button class="button primary" :disabled="busy">{{ t('jobs.manualForm.submit') }}</button></div>
    </form>

    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <div v-if="loading" class="empty-state">{{ t('jobs.loading') }}</div>
    <template v-else-if="jobs.length">
      <ListFilters v-model:search="search" :total="filteredJobs.length" :search-placeholder="t('jobs.filters.searchPlaceholder')">
        <label>{{ t('jobs.filters.statusLabel') }} <select v-model="statusFilter"><option value="all">{{ t('jobs.filters.allStatuses') }}</option><option v-for="status in statusOptions" :key="status.value" :value="status.value">{{ t(status.labelKey) }}</option></select></label>
        <label>{{ t('jobs.filters.sourceLabel') }} <select v-model="originFilter"><option value="all">{{ t('jobs.filters.allSources') }}</option><option value="manual">{{ t('jobs.filters.addedManually') }}</option><option value="url_import">{{ t('jobs.filters.importedViaUrl') }}</option><option value="hunter">{{ t('jobs.filters.foundByHunter') }}</option></select></label>
      </ListFilters>
    <BulkSelectionBar :selected-count="selection.selectedCount.value" :all-count="selectableFilteredJobs.length" :all-selected="allFilteredSelected" @select-all="selectAllFiltered" @clear="selection.clear">
      <button class="button primary" type="button" @click="openSelectedGeneration('resume')">{{ t('jobs.createCvs') }}</button>
      <button class="button" type="button" @click="openSelectedGeneration('cover_letter')">{{ t('jobs.createCoverLetters') }}</button>
    </BulkSelectionBar>
    <TransitionGroup v-if="visibleJobs.length" name="card-list" tag="div" class="list-panel">
      <article v-for="item in visibleJobs" :key="item.id" class="job-row" :class="{ selected: selection.isSelected(item.id) }">
        <input class="row-selector" type="checkbox" :checked="selection.isSelected(item.id)" :disabled="!canGenerate(item)" :aria-label="t('jobs.selectAriaLabel', { title: item.title || t('jobs.genericTitle') })" :title="canGenerate(item) ? t('jobs.selectForBatchTitle') : t('jobs.addDescriptionTitle')" @change="selection.toggle(item.id)" />
        <button class="company-mark company-preview-button" type="button" :aria-label="t('jobs.previewAriaLabel', { title: item.title || t('jobs.genericTitle') })" @click="openJobPreview(item)">{{ (item.company || '?').slice(0, 2).toUpperCase() }}</button>
        <div class="job-main job-preview-trigger" role="button" tabindex="0" :aria-label="t('jobs.previewAriaLabel', { title: item.title || t('jobs.genericTitle') })" @click="openJobPreview(item)" @keydown.enter="openJobPreview(item)" @keydown.space.prevent="openJobPreview(item)">
          <h2>{{ item.title || t('jobs.importingDetails') }}</h2>
          <p>{{ item.company || t('jobs.companyPending') }}<span v-if="item.location"> · {{ item.location }}</span><span v-else-if="item.city || item.country"> · {{ [item.city, item.country].filter(Boolean).join(', ') }}</span></p>
          <small v-if="importLabel(item)" :class="{ 'import-warning': item.importState === 'needs_user_action' || item.importState === 'failed' }">{{ importLabel(item) }}</small>
          <small v-if="item.origin === 'hunter'" class="hunter-source">{{ t('jobs.filters.foundByHunter') }}</small>
          <AttentionNotice v-if="item.importState === 'needs_user_action' || item.importState === 'failed'" compact :message="item.importError || t('jobs.importErrorFallback')" />
        </div>
        <label class="status-control" :class="item.status" :aria-label="t('jobs.changeStatusAriaLabel', { title: item.title })"><select :value="item.status" :disabled="updatingStatusId === item.id" @change="updateStatus(item, $event)"><option v-for="status in jobStatuses" :key="status.value" :value="status.value">{{ t(status.labelKey) }}</option></select><span aria-hidden="true">⌄</span></label>
        <div class="row-actions"><button class="text-button" type="button" :disabled="!canGenerate(item)" @click="openGeneration([item], 'resume')">{{ t('jobs.createCv') }}</button><button class="text-button" type="button" :disabled="!canGenerate(item)" @click="openGeneration([item], 'cover_letter')">{{ t('jobs.createCoverLetter') }}</button><RouterLink class="text-button" :to="`/jobs/${item.id}`">{{ t('common.edit') }}</RouterLink><a v-if="item.sourceUrl" class="source-link" :href="item.sourceUrl" target="_blank" rel="noopener noreferrer">{{ t('jobs.sourceLink') }}</a><button class="text-button danger-text" type="button" @click="pendingDelete = item">{{ t('common.delete') }}</button></div>
      </article>
    </TransitionGroup>
    <div v-else class="empty-state compact"><h2>{{ t('jobs.emptyFiltered.title') }}</h2><p>{{ t('jobs.emptyFiltered.message') }}</p></div>
    <ListPagination v-if="filteredJobs.length" v-model:page="page" v-model:page-size="pageSize" :total="filteredJobs.length" :page-sizes="[10, 20, 50]" />
    </template>
    <div v-else class="empty-state">
      <span class="empty-icon">◇</span><h2>{{ t('jobs.empty.title') }}</h2><p>{{ t('jobs.empty.message') }}</p>
    </div>
    <ConfirmDialog :open="Boolean(pendingDelete)" :title="t('jobs.deleteConfirm.title')" :message="t('jobs.deleteConfirm.message', { name: [pendingDelete?.title, pendingDelete?.company].filter(Boolean).join(' at ') || t('jobs.deleteConfirm.defaultName') })" :busy="deletingId === pendingDelete?.id" @cancel="pendingDelete = null" @confirm="removeJob" />
    <DocumentGenerationDialog :open="showGeneration" :opportunities="generationTargets" :initial-document-type="generationDocumentType" @close="showGeneration = false" @queued="handleGenerationQueued" />
    <Teleport to="body">
      <Transition name="job-preview">
        <div v-if="previewJob" class="modal-backdrop job-preview-backdrop" @click.self="previewJob = null">
          <section class="panel job-preview-modal" role="dialog" aria-modal="true" aria-labelledby="job-preview-title">
            <header class="modal-header job-preview-header">
              <div><p class="eyebrow">{{ t('jobs.preview.eyebrow') }}</p><h2 id="job-preview-title">{{ previewJob.title || t('jobs.preview.titlePending') }}</h2><p>{{ previewJob.company || t('jobs.companyPending') }}</p></div>
              <button class="modal-close" type="button" :aria-label="t('jobs.preview.closeAriaLabel')" @click="previewJob = null">×</button>
            </header>

            <div class="job-preview-meta">
              <div><span>{{ t('jobs.preview.locationLabel') }}</span><strong>{{ jobLocation(previewJob) }}</strong></div>
              <div><span>{{ t('jobs.preview.workModeLabel') }}</span><strong>{{ previewJob.workMode || t('jobs.notSpecified') }}</strong></div>
              <div><span>{{ t('jobs.preview.employmentTypeLabel') }}</span><strong>{{ previewJob.employmentType || t('jobs.notSpecified') }}</strong></div>
              <div><span>{{ t('jobs.preview.trackingStatusLabel') }}</span><strong>{{ jobStatusLabel(previewJob.status) }}</strong></div>
            </div>

            <AttentionNotice v-if="previewJob.importState === 'needs_user_action' || previewJob.importState === 'failed'" :message="previewJob.importError || t('jobs.preview.importErrorFallback')" />

            <section class="job-preview-description">
              <div><p class="eyebrow">{{ t('jobs.preview.roleDetailsEyebrow') }}</p><h3>{{ t('jobs.preview.descriptionTitle') }}</h3></div>
              <p>{{ previewLoading ? t('jobs.preview.loadingDetails') : previewJob.description || t('jobs.preview.noDescription') }}</p>
            </section>

            <footer class="modal-actions job-preview-actions">
              <button class="button" type="button" @click="previewJob = null">{{ t('common.close') }}</button>
              <a v-if="previewJob.sourceUrl" class="button" :href="previewJob.sourceUrl" target="_blank" rel="noopener noreferrer">{{ t('jobs.preview.openSource') }}</a>
              <RouterLink class="button primary" :to="`/jobs/${previewJob.id}`" @click="previewJob = null">{{ t('jobs.preview.editJob') }}</RouterLink>
            </footer>
          </section>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.jobs-page { display: grid; gap: 1.25rem; }
.jobs-page :deep(.page-header) { margin-bottom: .5rem; }
.header-actions, .row-actions { display: flex; align-items: center; gap: .6rem; }
.job-row { grid-template-columns: auto auto minmax(0, 1fr) 112px 390px; transition: background .18s ease, box-shadow .18s ease; }
.job-row.selected { background: color-mix(in srgb, var(--accent-pale) 55%, white); box-shadow: inset 3px 0 var(--accent); }
.row-selector { width: 17px; height: 17px; accent-color: var(--accent); cursor: pointer; }
.row-selector:disabled { cursor: not-allowed; opacity: .35; }
.company-preview-button { border: 0; cursor: pointer; font: inherit; }
.company-preview-button:hover { box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 16%, transparent); }
.company-preview-button:focus-visible { outline: 3px solid color-mix(in srgb, var(--accent) 24%, transparent); outline-offset: 2px; }
.job-preview-trigger { min-width: 0; margin: -8px; padding: 8px; border-radius: 10px; cursor: pointer; transition: background .18s ease; }
.job-preview-trigger:hover { background: color-mix(in srgb, var(--surface-soft) 72%, transparent); }
.job-preview-trigger:hover h2 { color: var(--accent-dark); }
.job-preview-trigger:focus-visible { outline: 3px solid color-mix(in srgb, var(--accent) 18%, transparent); outline-offset: 1px; }
.row-actions { flex-wrap: wrap; justify-content: flex-end; }
.row-actions .text-button:disabled { cursor: not-allowed; opacity: .4; }
.status-control { position: relative; display: inline-flex; align-items: center; justify-self: end; width: fit-content; border-radius: 999px; background: var(--surface-soft); color: var(--accent-dark); }
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
.job-preview-backdrop { z-index: 35; }
.job-preview-modal { width: min(900px, 100%); max-height: calc(100vh - 48px); overflow-y: auto; box-shadow: 0 30px 90px rgba(0, 0, 0, .28); }
.job-preview-header { align-items: flex-start; }
.job-preview-header h2 { margin-top: 3px; }
.job-preview-header div > p:last-child { margin-top: .45rem; color: var(--muted); }
.job-preview-meta { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; margin-bottom: 24px; }
.job-preview-meta > div { display: grid; gap: 6px; min-width: 0; padding: 12px 14px; border: 1px solid var(--line); border-radius: 10px; background: var(--surface-soft); }
.job-preview-meta span { color: var(--muted); font-size: 11px; }
.job-preview-meta strong { overflow-wrap: anywhere; font-size: 13px; }
.job-preview-description { display: grid; gap: 14px; padding: 22px 0 6px; border-top: 1px solid var(--line); }
.job-preview-description h3 { margin: 3px 0 0; }
.job-preview-description > p { margin: 0; color: var(--ink); line-height: 1.7; overflow-wrap: anywhere; white-space: pre-wrap; }
.job-preview-actions { margin-top: 24px; }
.job-preview-enter-active, .job-preview-leave-active { transition: opacity .18s ease; }
.job-preview-enter-active .job-preview-modal, .job-preview-leave-active .job-preview-modal { transition: transform .2s ease, opacity .18s ease; }
.job-preview-enter-from, .job-preview-leave-to { opacity: 0; }
.job-preview-enter-from .job-preview-modal, .job-preview-leave-to .job-preview-modal { opacity: 0; transform: translateY(10px) scale(.98); }
@media (max-width: 980px) { .job-row { grid-template-columns: auto auto minmax(0, 1fr); } .job-row .status-control, .job-row .row-actions { grid-column: 3; } .row-actions { justify-content: flex-start; } }
@media (max-width: 760px) { .header-actions, .row-actions { align-items: stretch; flex-direction: column; } .job-row .status-control { grid-column: 2; } .job-preview-meta { grid-template-columns: 1fr 1fr; } }
@media (max-width: 520px) { .job-preview-meta { grid-template-columns: 1fr; } .job-preview-actions { align-items: stretch; flex-direction: column; } }
</style>
