<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
import ConfirmDialog from '../components/ConfirmDialog.vue'
import AttentionNotice from '../components/AttentionNotice.vue'
import ListFilters from '../components/ListFilters.vue'
import ListPagination from '../components/ListPagination.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { usePagedList } from '../lib/pagination'
import { toast } from '../lib/toast'
import type { Template, TemplateKind, TemplateState } from '../lib/types'

const items = ref<Template[]>([])
const loading = ref(true)
const saving = ref(false)
const deletingId = ref('')
const pendingDelete = ref<Template | null>(null)
const showUpload = ref(false)
const uploadsEnabled = ref(false)
const error = ref('')
const search = ref('')
const kindFilter = ref('all')
const file = ref<File | null>(null)
const form = reactive({ name: '', kind: 'resume' as TemplateKind, description: '', entryFile: '' })
let pollTimer: number | undefined

const filteredItems = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  return items.value.filter(item => {
    const matchesSearch = !query || [item.name, item.description, item.sourceName, item.authorName, item.format].some(value => value?.toLocaleLowerCase().includes(query))
    return matchesSearch && (kindFilter.value === 'all' || item.kind === kindFilter.value)
  })
})
const { page, pageSize, visible: visibleItems } = usePagedList(filteredItems, [search, kindFilter])

function kindLabel(kind: TemplateKind) { return kind === 'resume' ? t('templates.kindResume') : t('templates.kindCoverLetter') }
function formatLabel(item: Template) { return item.sourceName.toLowerCase().endsWith('.zip') ? t('templates.formatLatexZip') : item.format === 'latex' ? t('templates.formatTex') : item.format.toUpperCase() }
const stateLabelKeys: Record<TemplateState, string> = { staged: 'templates.state.staged', queued: 'templates.state.queued', ready: 'templates.state.ready', needs_user_action: 'templates.state.needsUserAction', security_quarantine: 'templates.state.securityQuarantine', failed: 'templates.state.failed' }
function stateLabel(state: TemplateState) { return t(stateLabelKeys[state]) }
function isZipFile() { return file.value?.name.toLowerCase().endsWith('.zip') ?? false }

async function load() {
  try {
    const [templates, capabilities] = await Promise.all([api.listTemplates(), api.capabilities()])
    items.value = templates.items
    uploadsEnabled.value = Boolean(capabilities.features.templateUploads)
  } catch (cause) { error.value = cause instanceof Error ? cause.message : t('templates.errors.load') }
  finally { loading.value = false }
}

function schedulePoll() {
  window.clearTimeout(pollTimer)
  pollTimer = window.setTimeout(async () => { await load(); if (items.value.some(item => item.state === 'queued')) schedulePoll() }, 1800)
}

function chooseFile(event: Event) {
  file.value = (event.target as HTMLInputElement).files?.[0] ?? null
  if (file.value && !form.name) form.name = file.value.name.replace(/\.(tex|zip|docx?)$/i, '')
  if (!isZipFile()) form.entryFile = ''
}

async function upload() {
  if (!file.value) { error.value = t('templates.errors.chooseFile'); return }
  saving.value = true; error.value = ''
  try {
    const staged = await api.stageTemplate({ ...form, file: file.value })
    await api.putStagedTemplate(staged.target, file.value)
    await api.completeTemplate(staged.template.id)
    showUpload.value = false; file.value = null; Object.assign(form, { name: '', kind: 'resume', description: '', entryFile: '' })
    toast.success(t('templates.toasts.uploaded'))
    await load(); schedulePoll()
  } catch (cause) { error.value = cause instanceof Error ? cause.message : t('templates.errors.upload') }
  finally { saving.value = false }
}

async function remove() {
  const item = pendingDelete.value
  if (!item) return
  deletingId.value = item.id
  try { await api.deleteTemplate(item.id); items.value = items.value.filter(candidate => candidate.id !== item.id); pendingDelete.value = null; toast.success(t('templates.toasts.deleted')) }
  catch (cause) { error.value = cause instanceof Error ? cause.message : t('templates.errors.delete') }
  finally { deletingId.value = '' }
}

async function download(item: Template) {
  try { await api.downloadTemplate(item) } catch (cause) { error.value = cause instanceof Error ? cause.message : t('templates.errors.download') }
}

onMounted(async () => { await load(); if (items.value.some(item => item.state === 'queued')) schedulePoll() })
onBeforeUnmount(() => window.clearTimeout(pollTimer))
</script>

<template>
  <div class="page templates-page">
    <PageHeader :title="t('pages.templates.title')" :description="t('pages.templates.description')">
      <button v-if="uploadsEnabled" class="button primary" type="button" @click="showUpload = !showUpload">{{ showUpload ? t('common.close') : t('templates.uploadTemplate') }}</button>
    </PageHeader>
    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <form v-if="showUpload" class="panel form-grid upload-panel" @submit.prevent="upload">
      <label><span>{{ t('templates.form.nameLabel') }}</span><input v-model="form.name" required maxlength="120" :placeholder="t('templates.form.namePlaceholder')" /></label>
      <label><span>{{ t('templates.form.kindLabel') }}</span><select v-model="form.kind"><option value="resume">{{ t('templates.kindResume') }}</option><option value="cover_letter">{{ t('templates.kindCoverLetter') }}</option></select></label>
      <label class="full"><span>{{ t('templates.form.descriptionLabel') }}</span><textarea v-model="form.description" maxlength="1000" rows="3" :placeholder="t('templates.form.descriptionPlaceholder')" /></label>
      <label class="full"><span>{{ t('templates.form.fileLabel') }}</span><input required type="file" accept=".tex,.zip,.doc,.docx,application/x-tex,application/zip,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document" @change="chooseFile" /><small>{{ t('templates.form.fileHint') }}</small></label>
      <label v-if="isZipFile()" class="full"><span>{{ t('templates.form.entryFileLabel') }} <small>{{ t('templates.form.optional') }}</small></span><input v-model="form.entryFile" maxlength="300" :placeholder="t('templates.form.entryFilePlaceholder')" /><small>{{ t('templates.form.entryFileHint') }}</small></label>
      <div class="full form-actions"><button class="button primary" :disabled="saving">{{ saving ? t('templates.uploading') : t('templates.uploadAndProcess') }}</button></div>
    </form>
    <div v-if="loading" class="empty-state">{{ t('templates.loading') }}</div>
    <template v-else-if="items.length">
    <ListFilters v-model:search="search" :total="filteredItems.length" :search-placeholder="t('templates.searchPlaceholder')">
      <label>{{ t('templates.typeFilterLabel') }} <select v-model="kindFilter"><option value="all">{{ t('templates.allTypes') }}</option><option value="resume">{{ t('templates.kindResume') }}</option><option value="cover_letter">{{ t('templates.kindCoverLetter') }}</option></select></label>
    </ListFilters>
    <TransitionGroup v-if="visibleItems.length" name="card-list" tag="div" class="card-grid">
      <article v-for="item in visibleItems" :key="item.id" class="entity-card template-card">
        <div class="template-content"><p class="eyebrow">{{ kindLabel(item.kind) }}</p><h2>{{ item.name }}</h2><p>{{ item.description || item.sourceName }}</p></div>
        <div class="template-meta">
          <div class="template-tags"><span class="format-pill">{{ formatLabel(item) }}</span><span class="status-pill">{{ item.builtIn ? t('templates.builtIn') : stateLabel(item.state) }}</span></div>
          <span v-if="item.authorName">{{ t('templates.byAuthor', { author: item.authorName, license: item.license }) }}</span><span v-else>{{ item.sourceName }}</span>
        </div>
        <AttentionNotice v-if="['needs_user_action','security_quarantine','failed'].includes(item.state)" compact :message="item.errorMessage || t('templates.processingIssue')" :code="item.errorCode" />
        <footer><RouterLink class="text-button" :to="`/templates/${item.id}`">{{ t('common.view') }}</RouterLink><div><button v-if="item.state === 'ready'" class="text-button" type="button" @click="download(item)">{{ t('common.download') }}</button><button v-if="!item.builtIn" class="text-button danger-text" type="button" @click="pendingDelete = item">{{ t('common.delete') }}</button></div></footer>
      </article>
    </TransitionGroup>
    <div v-else class="empty-state compact"><h2>{{ t('templates.emptyFiltered.title') }}</h2><p>{{ t('templates.emptyFiltered.message') }}</p></div>
    <ListPagination v-if="filteredItems.length" v-model:page="page" v-model:page-size="pageSize" :total="filteredItems.length" />
    </template>
    <div v-else-if="!loading" class="empty-state"><span class="empty-icon">▧</span><h2>{{ t('templates.emptyState.title') }}</h2><p>{{ t('templates.emptyState.message') }}</p></div>
    <ConfirmDialog :open="Boolean(pendingDelete)" :title="t('templates.deleteConfirmTitle')" :message="t('templates.deleteConfirmMessage', { name: pendingDelete?.name ?? '' })" :busy="deletingId === pendingDelete?.id" @cancel="pendingDelete = null" @confirm="remove" />
  </div>
</template>

<style scoped>
.templates-page { display:grid; gap:1.1rem; }.upload-panel{margin-bottom:1rem}.upload-panel small{color:var(--muted)}.template-card{grid-template-columns:1fr}.template-meta{display:flex;align-items:center;justify-content:space-between;gap:1rem;color:var(--muted);font-size:.76rem}.template-tags{display:flex;align-items:center;gap:.45rem}.format-pill{display:inline-flex;align-items:center;width:fit-content;padding:5px 10px;border:1px solid var(--line);border-radius:999px;color:var(--accent-dark);font-size:11px;font-weight:800;letter-spacing:.04em}.template-card footer{grid-column:1}.template-card footer>div{display:flex;gap:.8rem}.danger-text{color:var(--danger)}@media(max-width:520px){.template-meta{align-items:flex-start;flex-direction:column}}
</style>
