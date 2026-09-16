<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import GenerationPdfPreview from '../components/GenerationPdfPreview.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { toast } from '../lib/toast'
import type { GenerationInput, GenerationModelChoice, GenerationRun, GenerationStep, Job, LLMConnection, Profile, Template } from '../lib/types'

const route = useRoute()
const router = useRouter()
const generationId = String(route.params.generationId)
const run = ref<GenerationRun | null>(null)
const steps = ref<GenerationStep[]>([])
const profiles = ref<Profile[]>([])
const opportunities = ref<Job[]>([])
const templates = ref<Template[]>([])
const connections = ref<LLMConnection[]>([])
const models = reactive<Record<string, string[]>>({})
const modelRequests = new Map<string, Promise<string[]>>()
const loading = ref(true)
const error = ref('')
const prompt = ref('')
const submitting = ref(false)
const showEdit = ref(false)
const selectedStep = ref<string | undefined>()
const sortOrder = ref<'desc' | 'asc'>('desc')
const timelineScroll = ref<HTMLElement>()
const emptyChoice = (): GenerationModelChoice => ({ connectionId: '', model: '' })
const editForm = reactive<GenerationInput>({ profileId: '', opportunityId: '', templateId: '', documentType: 'resume', language: 'English', pageTarget: 'one_page', customInstructions: '', pipelineMode: 'single', writer: emptyChoice(), renderer: emptyChoice(), reviewer: emptyChoice() })
let timer: number | undefined

const active = computed(() => run.value?.state === 'queued' || run.value?.state === 'running')
const title = computed(() => { const item = opportunities.value.find(value => value.id === run.value?.opportunityId); return item ? [item.title, item.company].filter(Boolean).join(' · ') : 'Application details' })
const profileName = computed(() => profiles.value.find(value => value.id === run.value?.profileId)?.name ?? 'Deleted profile')
const templateName = computed(() => templates.value.find(value => value.id === run.value?.templateId)?.name ?? 'Deleted template')
const stageLabel = computed(() => run.value ? ({ queued: 'Waiting to start', writing: 'Writing content', rendering: 'Applying template', reviewing: 'Reviewing the PDF', repairing: `Repairing layout (${run.value.repairCount}/2)`, ready: 'Ready', failed: 'Needs attention' } as Record<string, string>)[run.value.stage] ?? run.value.stage : '')
const activeIcon = computed(() => ({ queued: 'Q', writing: 'W', rendering: 'P', reviewing: 'R', repairing: 'R' } as Record<string, string>)[run.value?.stage ?? 'queued'] ?? '•')
const sortedSteps = computed(() => [...steps.value].sort((left, right) => sortOrder.value === 'desc' ? right.sequence - left.sequence : left.sequence - right.sequence))
const matchingTemplates = computed(() => templates.value.filter(item => item.state === 'ready' && item.kind === editForm.documentType && item.format === 'latex'))
const editReady = computed(() => Boolean(editForm.profileId && editForm.templateId && editForm.writer.connectionId && editForm.writer.model && (editForm.pipelineMode === 'single' || (editForm.renderer.connectionId && editForm.renderer.model && editForm.reviewer.connectionId && editForm.reviewer.model))))

const stepTitle = (step: GenerationStep) => ({ writer_draft: 'Writer draft', rendered_pdf: step.repairCount ? `Rendered PDF · repair ${step.repairCount}` : 'Template-applied PDF', reviewer_feedback: step.repairCount ? `Reviewer feedback · round ${step.repairCount + 1}` : 'Reviewer feedback', user_prompt: 'Your follow-up instruction', system_warning: 'Workflow notice', configuration_change: 'Configuration updated' } as Record<string, string>)[step.kind]
const stepIcon = (kind: GenerationStep['kind']) => ({ writer_draft: 'W', rendered_pdf: 'P', reviewer_feedback: 'R', user_prompt: 'U', system_warning: '!', configuration_change: '↻' } as Record<string, string>)[kind]
const formatDate = (value: string) => new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
const newestRenderedStep = (values = steps.value) => [...values].reverse().find(value => value.kind === 'rendered_pdf')?.id

async function toggleTimelineOrder() {
  sortOrder.value = sortOrder.value === 'desc' ? 'asc' : 'desc'
  await nextTick()
  const target = timelineScroll.value
  if (!target) return
  const behavior = window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth'
  target.scrollTo({ top: sortOrder.value === 'desc' ? 0 : target.scrollHeight, behavior })
}

async function discover(choice: GenerationModelChoice) {
  if (!choice.connectionId || models[choice.connectionId]) return
  const connectionId = choice.connectionId
  let request = modelRequests.get(connectionId)
  if (!request) {
    request = api.testLLMConnection(connectionId).then(result => result.models)
    modelRequests.set(connectionId, request)
  }
  try { models[connectionId] = await request; if (!choice.model) choice.model = models[connectionId][0] ?? '' }
  catch (cause) { error.value = cause instanceof Error ? cause.message : 'Could not load models.' }
  finally { modelRequests.delete(connectionId) }
}

watch(() => editForm.writer.connectionId, () => void discover(editForm.writer))
watch(() => editForm.renderer.connectionId, () => void discover(editForm.renderer))
watch(() => editForm.reviewer.connectionId, () => void discover(editForm.reviewer))
watch(() => editForm.documentType, () => { if (!matchingTemplates.value.some(item => item.id === editForm.templateId)) editForm.templateId = matchingTemplates.value[0]?.id ?? '' })

async function loadAll() {
  try {
    const [generation, timeline, profileList, opportunityList, templateList, connectionList] = await Promise.all([api.getGeneration(generationId), api.generationSteps(generationId), api.listProfiles(), api.listJobs(), api.listTemplates(), api.listLLMConnections()])
    run.value = generation; steps.value = timeline.items; profiles.value = profileList.items; opportunities.value = opportunityList.items; templates.value = templateList.items; connections.value = connectionList.items
    if (!selectedStep.value) selectedStep.value = newestRenderedStep(timeline.items)
    if (route.query.edit === '1' && !active.value) { openEditor(); await router.replace({ path: route.path }) }
  } catch (cause) { error.value = cause instanceof Error ? cause.message : 'Could not load this application.' }
  finally { loading.value = false }
}

async function refresh() {
  try { const [generation, timeline] = await Promise.all([api.getGeneration(generationId), api.generationSteps(generationId)]); run.value = generation; steps.value = timeline.items; if (!selectedStep.value) selectedStep.value = newestRenderedStep(timeline.items) } catch {}
  if (!active.value && timer) { window.clearInterval(timer); timer = undefined }
}
function startPolling() { if (!timer) timer = window.setInterval(refresh, 2000) }

function openEditor() {
  if (!run.value || active.value) return
  Object.assign(editForm, { profileId: run.value.profileId, opportunityId: run.value.opportunityId, templateId: run.value.templateId, documentType: run.value.documentType, language: run.value.language, pageTarget: run.value.pageTarget, customInstructions: run.value.customInstructions ?? '', pipelineMode: run.value.pipelineMode })
  editForm.writer = { ...run.value.writer }; editForm.renderer = { ...run.value.renderer }; editForm.reviewer = { ...run.value.reviewer }
  showEdit.value = true
  void Promise.all([discover(editForm.writer), discover(editForm.renderer), discover(editForm.reviewer)])
}

async function regenerate() {
  if (!run.value || !editReady.value) return
  submitting.value = true; error.value = ''
  try {
    const input: GenerationInput = JSON.parse(JSON.stringify(editForm)); input.opportunityId = run.value.opportunityId
    if (input.pipelineMode === 'single') input.renderer = input.reviewer = { ...input.writer }
    run.value = await api.reconfigureGeneration(run.value.id, input); showEdit.value = false
    toast.success('Configuration updated. A fresh generation has started and previous timeline records remain available.')
    await refresh(); startPolling()
  } catch (cause) { error.value = cause instanceof Error ? cause.message : 'Could not update and regenerate this application.' }
  finally { submitting.value = false }
}

async function revise() {
  if (!run.value || !prompt.value.trim()) return
  submitting.value = true; error.value = ''
  try { run.value = await api.reviseGeneration(run.value.id, prompt.value); prompt.value = ''; toast.success('Revision queued. The timeline will update as the model works.'); await refresh(); startPolling() }
  catch (cause) { error.value = cause instanceof Error ? cause.message : 'Could not queue the revision.' }
  finally { submitting.value = false }
}
async function retry() { if (!run.value) return; submitting.value = true; try { run.value = await api.retryGeneration(run.value.id); toast.success('Generation queued again.'); startPolling() } catch (cause) { error.value = cause instanceof Error ? cause.message : 'Could not retry.' } finally { submitting.value = false } }
async function download() { if (!run.value) return; try { await api.downloadGeneration(run.value) } catch (cause) { error.value = cause instanceof Error ? cause.message : 'Could not download the PDF.' } }

onMounted(async () => { await loadAll(); if (active.value) startPolling() })
onBeforeUnmount(() => { if (timer) window.clearInterval(timer) })
</script>

<template>
  <div class="page generation-detail">
    <PageHeader :title="title" description="Follow the complete writing, template application, review, and revision timeline."><RouterLink class="button" to="/generate">Back to applications</RouterLink></PageHeader>
    <p v-if="error" class="notice error">{{ error }}</p>
    <div v-if="loading" class="empty-state">Loading application…</div>
    <template v-else-if="run">
      <section class="panel run-summary"><div><span class="status-pill" :class="run.state">{{ stageLabel }}</span><h2>{{ run.documentType === 'resume' ? 'Resume' : 'Cover letter' }}</h2><p>{{ profileName }} · {{ templateName }} · {{ run.pageTarget.replace('_', ' ') }} · {{ run.language }}</p></div><div class="summary-actions"><button v-if="!active" class="button" type="button" :disabled="submitting" @click="openEditor">Edit &amp; regenerate</button><button v-if="run.state === 'failed'" class="button" :disabled="submitting" @click="retry">Retry</button><button v-if="run.state === 'ready'" class="button primary" @click="download">Download PDF</button></div></section>

      <section class="detail-grid">
        <div class="panel timeline-panel">
          <div class="section-title timeline-heading"><div><p class="eyebrow">Trace</p><h2>Generation timeline</h2></div><button class="timeline-sort" type="button" @click="toggleTimelineOrder"><span>{{ sortOrder === 'desc' ? '↓' : '↑' }}</span>{{ sortOrder === 'desc' ? 'Newest first' : 'Oldest first' }}</button></div>
          <div ref="timelineScroll" class="timeline-scroll"><div v-if="!steps.length && !active && !run.draft" class="empty-state compact">Stage outputs will appear when processing begins.</div><div v-else class="timeline" :class="sortOrder">
            <article v-if="active && sortOrder === 'desc'" class="timeline-step live-step"><span class="timeline-icon">{{ activeIcon }}</span><div><h3>{{ stageLabel }}</h3><p class="step-meta">In progress now</p><p class="step-feedback">ResumeGPT is working on this stage. New output will appear automatically.</p></div></article>
            <article v-if="!steps.length && run.draft" class="timeline-step"><span class="timeline-icon">W</span><div><h3>Writer draft</h3><p class="step-meta">Saved before timeline tracking was enabled</p><details><summary>View draft</summary><pre>{{ run.draft }}</pre></details></div></article>
            <article v-for="step in sortedSteps" :key="step.id" class="timeline-step" :class="step.kind"><span class="timeline-icon">{{ stepIcon(step.kind) }}</span><div><h3>{{ stepTitle(step) }}</h3><p class="step-meta">{{ formatDate(step.createdAt) }}</p><p v-if="step.feedback" class="step-feedback">{{ step.feedback }}</p><details v-if="step.content && step.kind !== 'user_prompt' && step.kind !== 'configuration_change'"><summary>{{ step.kind === 'reviewer_feedback' ? 'View raw response' : step.kind === 'rendered_pdf' ? 'View LaTeX source' : 'View content' }}</summary><pre>{{ step.content }}</pre></details><blockquote v-if="step.kind === 'user_prompt'">{{ step.content }}</blockquote><button v-if="step.kind === 'rendered_pdf'" class="text-button preview-link" @click="selectedStep = step.id">Preview this PDF →</button></div></article>
            <article v-if="active && sortOrder === 'asc'" class="timeline-step live-step"><span class="timeline-icon">{{ activeIcon }}</span><div><h3>{{ stageLabel }}</h3><p class="step-meta">In progress now</p><p class="step-feedback">ResumeGPT is working on this stage. New output will appear automatically.</p></div></article>
          </div></div>
        </div>
        <aside class="panel preview-column"><div class="section-title"><div><p class="eyebrow">Artifact</p><h2>{{ selectedStep ? 'Stage PDF' : 'Final PDF' }}</h2></div><button v-if="selectedStep && run.state === 'ready'" class="text-button" @click="selectedStep = undefined">Show final</button></div><GenerationPdfPreview v-if="run.state === 'ready' || selectedStep" :generation-id="run.id" :step-id="selectedStep"/><div v-else class="empty-state compact">A PDF preview appears after the template is applied.</div></aside>
      </section>

      <section v-if="run.state === 'ready'" class="panel revision-panel"><div><p class="eyebrow">Refine</p><h2>Ask for another change</h2><p>Describe what should change. The model receives the grounded draft and current LaTeX, then adds the new result to this timeline.</p></div><form @submit.prevent="revise"><textarea v-model="prompt" rows="4" maxlength="4000" placeholder="For example: Reduce whitespace above Experience and emphasize the Go migration work." required/><div><small>{{ prompt.length }}/4000</small><button class="button primary" :disabled="submitting || !prompt.trim()">{{ submitting ? 'Queuing…' : 'Send revision' }}</button></div></form></section>

      <div v-if="showEdit" class="modal-backdrop" @click.self="showEdit = false"><section class="panel generation-modal" role="dialog" aria-modal="true" aria-labelledby="edit-application-title"><header class="modal-header"><div><p class="eyebrow">Application configuration</p><h2 id="edit-application-title">Edit &amp; regenerate</h2></div><button class="modal-close" type="button" aria-label="Close" :disabled="submitting" @click="showEdit = false">×</button></header><p class="edit-help">Changing these inputs starts a fresh generation. Existing timeline records and stage PDFs are preserved.</p><form @submit.prevent="regenerate"><div class="form-grid">
        <label><span>Document type</span><select v-model="editForm.documentType"><option value="resume">Resume</option><option value="cover_letter">Cover letter</option></select></label><label><span>Output language</span><input v-model="editForm.language" required maxlength="40" /></label>
        <label><span>Profile</span><select v-model="editForm.profileId" required><option value="" disabled>Select a profile</option><option v-for="item in profiles" :key="item.id" :value="item.id">{{ item.name }}</option></select></label><label><span>Opportunity</span><input :value="title" disabled /></label>
        <label><span>LaTeX template</span><select v-model="editForm.templateId" required><option value="" disabled>Select a template</option><option v-for="item in matchingTemplates" :key="item.id" :value="item.id">{{ item.name }}</option></select></label><label><span>Page target</span><select v-model="editForm.pageTarget"><option value="one_page">One page</option><option value="two_pages">Two pages</option><option value="flexible">Flexible</option></select></label>
        <label class="full"><span>Custom instructions</span><textarea v-model="editForm.customInstructions" rows="3" maxlength="4000" /></label>
        <div class="full pipeline-choice"><label><input v-model="editForm.pipelineMode" type="radio" value="single" /> One model</label><label><input v-model="editForm.pipelineMode" type="radio" value="multi" /> Specialized models</label></div>
        <fieldset class="full model-card"><legend>{{ editForm.pipelineMode === 'single' ? 'Workflow model' : 'Writer model' }}</legend><div class="model-row"><label><span>Connection</span><select v-model="editForm.writer.connectionId"><option value="" disabled>Select connection</option><option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option></select></label><label><span>Model</span><input v-model="editForm.writer.model" :list="`edit-writer-models-${editForm.writer.connectionId}`" required /><datalist :id="`edit-writer-models-${editForm.writer.connectionId}`"><option v-for="model in models[editForm.writer.connectionId] || []" :key="model" :value="model" /></datalist></label></div></fieldset>
        <template v-if="editForm.pipelineMode === 'multi'"><fieldset class="full model-card"><legend>Template applier</legend><div class="model-row"><label><span>Connection</span><select v-model="editForm.renderer.connectionId"><option value="" disabled>Select connection</option><option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option></select></label><label><span>Model</span><input v-model="editForm.renderer.model" :list="`edit-renderer-models-${editForm.renderer.connectionId}`" required /><datalist :id="`edit-renderer-models-${editForm.renderer.connectionId}`"><option v-for="model in models[editForm.renderer.connectionId] || []" :key="model" :value="model" /></datalist></label></div></fieldset><fieldset class="full model-card"><legend>Visual reviewer</legend><div class="model-row"><label><span>Connection</span><select v-model="editForm.reviewer.connectionId"><option value="" disabled>Select connection</option><option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option></select></label><label><span>Model</span><input v-model="editForm.reviewer.model" :list="`edit-reviewer-models-${editForm.reviewer.connectionId}`" required /><datalist :id="`edit-reviewer-models-${editForm.reviewer.connectionId}`"><option v-for="model in models[editForm.reviewer.connectionId] || []" :key="model" :value="model" /></datalist></label></div></fieldset></template>
      </div><div class="modal-actions"><button class="button" type="button" :disabled="submitting" @click="showEdit = false">Cancel</button><button class="button primary" :disabled="!editReady || submitting">{{ submitting ? 'Starting…' : 'Save & regenerate' }}</button></div></form></section></div>
    </template>
  </div>
</template>

<style scoped>
.generation-detail{display:grid;gap:20px}.run-summary{display:flex;align-items:center;justify-content:space-between;gap:20px}.run-summary h2{margin:12px 0 6px}.run-summary p,.revision-panel p{margin:0;color:var(--muted)}.summary-actions{display:flex;flex-wrap:wrap;gap:10px}.detail-grid{--artifact-height:clamp(620px,72vh,780px);display:grid;grid-template-columns:minmax(0,.9fr) minmax(420px,1.1fr);gap:20px;align-items:stretch}.timeline-panel,.preview-column{height:var(--artifact-height);min-height:0}.timeline-panel{display:flex;flex-direction:column}.section-title{display:flex;align-items:flex-end;justify-content:space-between;gap:16px;margin-bottom:20px}.section-title h2,.section-title p{margin-bottom:0}.timeline-heading{align-items:center;flex:0 0 auto}.timeline-sort{display:inline-flex;align-items:center;gap:7px;padding:7px 10px;border:1px solid var(--line);border-radius:8px;background:var(--surface);color:var(--muted);cursor:pointer;font-size:11px;font-weight:700;white-space:nowrap}.timeline-sort span{color:var(--accent);font-size:16px}.timeline-scroll{min-height:0;margin:-10px -9px 0 -10px;overflow-y:auto;padding:10px 18px 14px 10px;scroll-padding-top:10px;scrollbar-color:var(--line) transparent;scrollbar-width:thin}.timeline{display:grid}.timeline-step{position:relative;display:grid;grid-template-columns:38px minmax(0,1fr);gap:15px;padding:0 0 26px}.timeline-step:not(:last-child)::before{position:absolute;top:38px;bottom:0;left:18px;width:1px;background:var(--line);content:''}.timeline-icon{position:relative;z-index:1;display:grid;width:38px;height:38px;place-items:center;border:1px solid var(--line);border-radius:50%;background:var(--surface);color:var(--accent);font-size:12px;font-weight:800}.timeline-step.system_warning .timeline-icon{background:#fff3d7;color:#8a6724}.timeline-step.configuration_change .timeline-icon{background:var(--accent-pale)}.timeline-step.live-step .timeline-icon{border-color:transparent;background:var(--accent);color:#fff;box-shadow:0 0 0 5px var(--accent-pale)}.timeline-step.live-step .timeline-icon::after{position:absolute;inset:-6px;border:2px solid transparent;border-top-color:var(--accent);border-right-color:var(--accent);border-radius:50%;content:'';animation:spin .85s linear infinite}.timeline-step.live-step .step-feedback{border:1px solid #c6d9cd;background:var(--accent-pale)}.timeline-step h3{margin:7px 0 3px;font-size:15px}.step-meta{margin:0 0 10px;color:var(--muted);font-size:11px}.step-feedback{padding:11px 13px;border-radius:8px;background:var(--surface-soft);font-size:13px;line-height:1.55}.timeline details{margin-top:10px}.timeline summary{cursor:pointer;color:var(--accent);font-size:12px;font-weight:700}.timeline pre{max-height:360px;overflow:auto;padding:14px;border-radius:9px;background:#17201d;color:#e8efeb;white-space:pre-wrap;overflow-wrap:anywhere;font:11px/1.55 ui-monospace,SFMono-Regular,Menlo,monospace}.timeline blockquote{margin:10px 0 0;padding:12px 14px;border-left:3px solid var(--warm);background:var(--surface-soft);font-size:13px;line-height:1.55}.preview-link{margin-top:10px;padding:0}.preview-column{position:sticky;top:24px;display:flex;flex-direction:column}.preview-column :deep(.generation-pdf){min-height:0;flex:1}.revision-panel{display:grid;grid-template-columns:minmax(240px,.7fr) minmax(320px,1.3fr);gap:30px;align-items:start}.revision-panel form{display:grid;gap:10px}.revision-panel form div{display:flex;align-items:center;justify-content:space-between}.revision-panel small{color:var(--muted)}.edit-help{margin:-10px 0 22px;color:var(--muted);font-size:13px}.generation-modal{width:min(860px,100%)}@keyframes spin{to{transform:rotate(360deg)}}@media(prefers-reduced-motion:reduce){.timeline-step.live-step .timeline-icon::after{animation-duration:2.5s}}@media(max-width:980px){.detail-grid,.revision-panel{grid-template-columns:1fr}.preview-column{position:static}}@media(max-width:600px){.run-summary{align-items:flex-start;flex-direction:column}.summary-actions{width:100%}.summary-actions .button{flex:1}.timeline-heading{align-items:flex-start;flex-direction:column}}
.timeline .timeline-step:not(:last-child)::after{position:absolute;z-index:2;top:max(48px,50%);left:14px;width:0;height:0;border-right:4px solid transparent;border-left:4px solid transparent;content:'';filter:drop-shadow(0 0 2px var(--surface))}.timeline.desc .timeline-step:not(:last-child)::after{border-bottom:7px solid var(--accent);transform:translateY(-7px)}.timeline.asc .timeline-step:not(:last-child)::after{border-top:7px solid var(--accent)}
</style>
