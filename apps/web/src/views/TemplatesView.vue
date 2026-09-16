<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import ListFilters from '../components/ListFilters.vue'
import ListPagination from '../components/ListPagination.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { toast } from '../lib/toast'
import type { Template, TemplateKind } from '../lib/types'

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
const page = ref(1)
const pageSize = ref(8)
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
const visibleItems = computed(() => filteredItems.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
watch([search, kindFilter, pageSize], () => { page.value = 1 })
watch(() => filteredItems.value.length, total => { page.value = Math.min(page.value, Math.max(1, Math.ceil(total / pageSize.value))) })

function kindLabel(kind: TemplateKind) { return kind === 'resume' ? 'Resume' : 'Cover letter' }
function formatLabel(item: Template) { return item.sourceName.toLowerCase().endsWith('.zip') ? 'LATEX ZIP' : item.format === 'latex' ? 'TEX' : item.format.toUpperCase() }
function isZipFile() { return file.value?.name.toLowerCase().endsWith('.zip') ?? false }

async function load() {
  try {
    const [templates, capabilities] = await Promise.all([api.listTemplates(), api.capabilities()])
    items.value = templates.items
    uploadsEnabled.value = Boolean(capabilities.features.templateUploads)
  } catch (cause) { error.value = cause instanceof Error ? cause.message : 'Could not load templates.' }
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
  if (!file.value) { error.value = 'Choose a TeX, LaTeX ZIP, DOC, or DOCX file.'; return }
  saving.value = true; error.value = ''
  try {
    const staged = await api.stageTemplate({ ...form, file: file.value })
    await api.putStagedTemplate(staged.target, file.value)
    await api.completeTemplate(staged.template.id)
    showUpload.value = false; file.value = null; Object.assign(form, { name: '', kind: 'resume', description: '', entryFile: '' })
    toast.success('Template uploaded. Text extraction, security scanning, and PDF preview generation are running.')
    await load(); schedulePoll()
  } catch (cause) { error.value = cause instanceof Error ? cause.message : 'Could not upload the template.' }
  finally { saving.value = false }
}

async function remove() {
  const item = pendingDelete.value
  if (!item) return
  deletingId.value = item.id
  try { await api.deleteTemplate(item.id); items.value = items.value.filter(candidate => candidate.id !== item.id); pendingDelete.value = null; toast.success('Template deleted.') }
  catch (cause) { error.value = cause instanceof Error ? cause.message : 'Could not delete the template.' }
  finally { deletingId.value = '' }
}

async function download(item: Template) {
  try { await api.downloadTemplate(item) } catch (cause) { error.value = cause instanceof Error ? cause.message : 'Could not download the template.' }
}

onMounted(async () => { await load(); if (items.value.some(item => item.state === 'queued')) schedulePoll() })
onBeforeUnmount(() => window.clearTimeout(pollTimer))
</script>

<template>
  <div class="page templates-page">
    <PageHeader title="Templates" description="Keep resume and cover-letter source files ready for generation. Upload LaTeX or Word templates; ResumeGPT extracts their text without adding review or version workflows.">
      <button v-if="uploadsEnabled" class="button primary" type="button" @click="showUpload = !showUpload">{{ showUpload ? 'Close' : 'Upload template' }}</button>
    </PageHeader>
    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <form v-if="showUpload" class="panel form-grid upload-panel" @submit.prevent="upload">
      <label><span>Template name</span><input v-model="form.name" required maxlength="120" placeholder="Engineering resume" /></label>
      <label><span>Document type</span><select v-model="form.kind"><option value="resume">Resume</option><option value="cover_letter">Cover letter</option></select></label>
      <label class="full"><span>Description</span><textarea v-model="form.description" maxlength="1000" rows="3" placeholder="Optional note about layout or intended use" /></label>
      <label class="full"><span>Template file</span><input required type="file" accept=".tex,.zip,.doc,.docx,application/x-tex,application/zip,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document" @change="chooseFile" /><small>Single TeX file, multi-file LaTeX ZIP, DOC, or DOCX · maximum 5 MiB</small></label>
      <label v-if="isZipFile()" class="full"><span>LaTeX entry file <small>Optional</small></span><input v-model="form.entryFile" maxlength="300" placeholder="main.tex" /><small>Leave empty to detect main.tex automatically. Use a relative path such as src/resume.tex when needed.</small></label>
      <div class="full form-actions"><button class="button primary" :disabled="saving">{{ saving ? 'Uploading…' : 'Upload and process' }}</button></div>
    </form>
    <div v-if="loading" class="empty-state">Loading templates…</div>
    <template v-else-if="items.length">
    <ListFilters v-model:search="search" :total="filteredItems.length" search-placeholder="Search templates, files, or authors…">
      <label>Type <select v-model="kindFilter"><option value="all">All types</option><option value="resume">Resume</option><option value="cover_letter">Cover letter</option></select></label>
    </ListFilters>
    <TransitionGroup v-if="visibleItems.length" name="card-list" tag="div" class="card-grid">
      <article v-for="item in visibleItems" :key="item.id" class="entity-card template-card">
        <div class="template-content"><p class="eyebrow">{{ kindLabel(item.kind) }}</p><h2>{{ item.name }}</h2><p>{{ item.description || item.sourceName }}</p></div>
        <div class="template-meta">
          <div class="template-tags"><span class="format-pill">{{ formatLabel(item) }}</span><span class="status-pill">{{ item.builtIn ? 'Built in' : item.state.replaceAll('_', ' ') }}</span></div>
          <span v-if="item.authorName">By {{ item.authorName }} · {{ item.license }}</span><span v-else>{{ item.sourceName }}</span>
        </div>
        <footer><RouterLink class="text-button" :to="`/templates/${item.id}`">View</RouterLink><div><button v-if="item.state === 'ready'" class="text-button" type="button" @click="download(item)">Download</button><button v-if="!item.builtIn" class="text-button danger-text" type="button" @click="pendingDelete = item">Delete</button></div></footer>
      </article>
    </TransitionGroup>
    <div v-else class="empty-state compact"><h2>No matching templates</h2><p>Try another keyword or document type.</p></div>
    <ListPagination v-if="filteredItems.length" v-model:page="page" v-model:page-size="pageSize" :total="filteredItems.length" />
    </template>
    <div v-else-if="!loading" class="empty-state"><span class="empty-icon">▧</span><h2>No templates yet</h2><p>Upload a LaTeX or Word template to get started.</p></div>
    <ConfirmDialog :open="Boolean(pendingDelete)" title="Delete template?" :message="`“${pendingDelete?.name ?? ''}” and its stored source will be removed. This action cannot be undone.`" :busy="deletingId === pendingDelete?.id" @cancel="pendingDelete = null" @confirm="remove" />
  </div>
</template>

<style scoped>
.templates-page { display:grid; gap:1.1rem; }.upload-panel{margin-bottom:1rem}.upload-panel small{color:var(--muted)}.template-card{grid-template-columns:1fr}.template-meta{display:flex;align-items:center;justify-content:space-between;gap:1rem;color:var(--muted);font-size:.76rem}.template-tags{display:flex;align-items:center;gap:.45rem}.format-pill{display:inline-flex;align-items:center;width:fit-content;padding:5px 10px;border:1px solid var(--line);border-radius:999px;color:var(--accent-dark);font-size:11px;font-weight:800;letter-spacing:.04em}.template-card footer{grid-column:1}.template-card footer>div{display:flex;gap:.8rem}.danger-text{color:var(--danger)}@media(max-width:520px){.template-meta{align-items:flex-start;flex-direction:column}}
</style>
