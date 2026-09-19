<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

import GenerationPdfPreview from '../components/GenerationPdfPreview.vue'
import GenerationWorkflow from '../components/GenerationWorkflow.vue'
import PageHeader from '../components/PageHeader.vue'
import SearchCombobox from '../components/SearchCombobox.vue'
import { api } from '../lib/api'
import { formatDateTime } from '../lib/formatDate'
import { useModelDiscovery } from '../lib/modelDiscovery'
import { toast } from '../lib/toast'
import type { GenerationInput, GenerationModelChoice, GenerationRun, GenerationStep, Job, LLMConnection, Profile, Template } from '../lib/types'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const generationId = String(route.params.generationId)
const run = ref<GenerationRun | null>(null)
const steps = ref<GenerationStep[]>([])
const profiles = ref<Profile[]>([])
const opportunities = ref<Job[]>([])
const templates = ref<Template[]>([])
const connections = ref<LLMConnection[]>([])
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
const title = computed(() => { const item = opportunities.value.find(value => value.id === run.value?.opportunityId); return item ? [item.title, item.company].filter(Boolean).join(' · ') : t('generate.detail.fallbackTitle') })
const profileName = computed(() => profiles.value.find(value => value.id === run.value?.profileId)?.name ?? t('generate.detail.deletedProfile'))
const templateName = computed(() => run.value?.templateId ? templates.value.find(value => value.id === run.value?.templateId)?.name ?? t('generate.detail.deletedTemplate') : t('generate.noTemplate'))
const stageLabel = computed(() => run.value ? run.value.state === 'failed' ? t('common.needsAttention') : ({ queued: t('generate.detail.stage.waiting'), writing: t('generate.detail.stage.writing'), rendering: run.value.templateId ? t('generate.stage.applyingTemplate') : t('generate.stage.designingDocument'), reviewing: t('generate.detail.stage.reviewing'), repairing: t('generate.detail.stage.repairing', { count: run.value.repairCount }), finalizing: t('generate.detail.stage.finalizing'), ready: t('generate.stage.ready') } as Record<string, string>)[run.value.stage] ?? run.value.stage : '')
const activeIcon = computed(() => ({ queued: 'Q', writing: 'W', rendering: 'P', reviewing: 'R', repairing: 'R' } as Record<string, string>)[run.value?.stage ?? 'queued'] ?? '•')
const sortedSteps = computed(() => [...steps.value].sort((left, right) => sortOrder.value === 'desc' ? right.sequence - left.sequence : left.sequence - right.sequence))
const matchingTemplates = computed(() => templates.value.filter(item => item.state === 'ready' && item.kind === editForm.documentType && item.format === 'latex'))
const editProfileLabel = computed(() => editForm.profileId ? profiles.value.find(item => item.id === editForm.profileId)?.name ?? '' : '')
const editTemplateLabel = computed(() => editForm.templateId ? templates.value.find(item => item.id === editForm.templateId)?.name ?? '' : '')
async function searchEditProfileOptions(query: string) {
  const result = await api.searchProfiles({ search: query, page: 1, pageSize: 8 })
  return result.items.map(item => ({ id: item.id, label: item.name, sublabel: item.targetRole }))
}
async function searchEditTemplateOptions(query: string) {
  const result = await api.searchTemplates({ search: query, kind: editForm.documentType, page: 1, pageSize: 8 })
  return result.items.filter(item => item.state === 'ready' && item.format === 'latex').map(item => ({ id: item.id, label: item.name }))
}
const editReady = computed(() => Boolean(editForm.profileId && editForm.writer.connectionId && editForm.writer.model && (editForm.pipelineMode === 'single' || (editForm.renderer.connectionId && editForm.renderer.model && editForm.reviewer.connectionId && editForm.reviewer.model))))
const pageTargetLabel = (value: string) => ({ one_page: t('generate.pageTarget.onePage'), two_pages: t('generate.pageTarget.twoPages'), flexible: t('generate.pageTarget.flexible') } as Record<string, string>)[value] ?? value.replace('_', ' ')

const stepTitle = (step: GenerationStep) => ({ writer_draft: t('generate.detail.stepTitle.writerDraft'), rendered_pdf: step.repairCount ? t('generate.detail.stepTitle.renderedPdfRepair', { count: step.repairCount }) : run.value?.templateId ? t('generate.detail.stepTitle.templateAppliedPdf') : t('generate.detail.stepTitle.designerRenderedPdf'), reviewer_feedback: step.repairCount ? t('generate.detail.stepTitle.reviewerFeedbackRound', { round: step.repairCount + 1 }) : t('generate.detail.stepTitle.reviewerFeedback'), user_prompt: t('generate.detail.stepTitle.userPrompt'), system_warning: t('generate.detail.stepTitle.systemWarning'), configuration_change: t('generate.detail.stepTitle.configurationChange') } as Record<string, string>)[step.kind]
const stepIcon = (kind: GenerationStep['kind']) => ({ writer_draft: 'W', rendered_pdf: 'P', reviewer_feedback: 'R', user_prompt: 'U', system_warning: '!', configuration_change: '↻' } as Record<string, string>)[kind]
const formatDate = (value: string) => formatDateTime(value)
const newestRenderedStep = (values = steps.value) => [...values].reverse().find(value => value.kind === 'rendered_pdf')?.id

async function toggleTimelineOrder() {
  sortOrder.value = sortOrder.value === 'desc' ? 'asc' : 'desc'
  await nextTick()
  const target = timelineScroll.value
  if (!target) return
  const behavior = window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth'
  target.scrollTo({ top: sortOrder.value === 'desc' ? 0 : target.scrollHeight, behavior })
}

const { models, loadingModelConnections, discover, changeConnection } = useModelDiscovery(
  message => { error.value = message },
  () => t('generate.errors.loadModels'),
)

watch(() => editForm.documentType, () => { if (editForm.templateId && !matchingTemplates.value.some(item => item.id === editForm.templateId)) editForm.templateId = '' })

async function loadAll() {
  try {
    const [generation, timeline, profileList, opportunityList, templateList, connectionList] = await Promise.all([api.getGeneration(generationId), api.generationSteps(generationId), api.listProfiles(), api.listJobs(), api.listTemplates(), api.listLLMConnections()])
    run.value = generation; steps.value = timeline.items; profiles.value = profileList.items; opportunities.value = opportunityList.items; templates.value = templateList.items; connections.value = connectionList.items
    if (!selectedStep.value) selectedStep.value = newestRenderedStep(timeline.items)
    if (route.query.edit === '1' && !active.value) { openEditor(); await router.replace({ path: route.path }) }
  } catch (cause) { error.value = cause instanceof Error ? cause.message : t('generate.detail.errors.load') }
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
    toast.success(t('generate.detail.toast.regenerated'))
    await refresh(); startPolling()
  } catch (cause) { error.value = cause instanceof Error ? cause.message : t('generate.detail.errors.regenerate') }
  finally { submitting.value = false }
}

async function revise() {
  if (!run.value || !prompt.value.trim()) return
  submitting.value = true; error.value = ''
  try { run.value = await api.reviseGeneration(run.value.id, prompt.value); prompt.value = ''; toast.success(t('generate.detail.toast.revised')); await refresh(); startPolling() }
  catch (cause) { error.value = cause instanceof Error ? cause.message : t('generate.detail.errors.revise') }
  finally { submitting.value = false }
}
async function retry() { if (!run.value) return; submitting.value = true; try { run.value = await api.retryGeneration(run.value.id); toast.success(t('generate.detail.toast.retried')); startPolling() } catch (cause) { error.value = cause instanceof Error ? cause.message : t('generate.detail.errors.retry') } finally { submitting.value = false } }
async function download() { if (!run.value) return; try { await api.downloadGeneration(run.value) } catch (cause) { error.value = cause instanceof Error ? cause.message : t('generate.detail.errors.download') } }

onMounted(async () => { await loadAll(); if (active.value) startPolling() })
onBeforeUnmount(() => { if (timer) window.clearInterval(timer) })
</script>

<template>
  <div class="page generation-detail">
    <PageHeader :title="title" :description="t('generate.detail.description')"><RouterLink class="button" to="/generate">{{ t('generate.detail.backButton') }}</RouterLink></PageHeader>
    <p v-if="error" class="notice error">{{ error }}</p>
    <div v-if="loading" class="empty-state">{{ t('generate.detail.loading') }}</div>
    <template v-else-if="run">
      <section class="panel run-summary"><div><span class="status-pill" :class="run.state">{{ stageLabel }}</span><h2>{{ run.documentType === 'resume' ? t('generate.documentKind.resume') : t('generate.documentKind.coverLetter') }}</h2><p>{{ profileName }} · {{ templateName }} · {{ pageTargetLabel(run.pageTarget) }} · {{ run.language }}</p></div><div class="summary-actions"><button v-if="!active" class="button" type="button" :disabled="submitting" @click="openEditor">{{ t('generate.detail.editRegenerate') }}</button><button v-if="run.state === 'failed'" class="button" :disabled="submitting" @click="retry">{{ t('generate.detail.retryButton') }}</button><button v-if="run.state === 'ready'" class="button primary" @click="download">{{ t('generate.detail.downloadPdf') }}</button></div></section>
      <GenerationWorkflow :run="run" />

      <section class="detail-grid">
        <div class="panel timeline-panel">
          <div class="section-title timeline-heading"><div><p class="eyebrow">{{ t('generate.detail.traceEyebrow') }}</p><h2>{{ t('generate.detail.timelineTitle') }}</h2></div><button class="timeline-sort" type="button" @click="toggleTimelineOrder"><span>{{ sortOrder === 'desc' ? '↓' : '↑' }}</span>{{ sortOrder === 'desc' ? t('generate.detail.newestFirst') : t('generate.detail.oldestFirst') }}</button></div>
          <div ref="timelineScroll" class="timeline-scroll"><div v-if="!steps.length && !active && !run.draft" class="empty-state compact">{{ t('generate.detail.noStepsYet') }}</div><div v-else class="timeline" :class="sortOrder">
            <article v-if="active && sortOrder === 'desc'" class="timeline-step live-step"><span class="timeline-icon">{{ activeIcon }}</span><div><h3>{{ stageLabel }}</h3><p class="step-meta">{{ t('generate.detail.inProgressNow') }}</p><p class="step-feedback">{{ t('generate.detail.workingMessage') }}</p></div></article>
            <article v-if="!steps.length && run.draft" class="timeline-step"><span class="timeline-icon">W</span><div><h3>{{ t('generate.detail.stepTitle.writerDraft') }}</h3><p class="step-meta">{{ t('generate.detail.savedBeforeTracking') }}</p><details><summary>{{ t('generate.detail.viewDraft') }}</summary><pre>{{ run.draft }}</pre></details></div></article>
            <article v-for="step in sortedSteps" :key="step.id" class="timeline-step" :class="step.kind"><span class="timeline-icon">{{ stepIcon(step.kind) }}</span><div><h3>{{ stepTitle(step) }}</h3><p class="step-meta">{{ formatDate(step.createdAt) }}</p><p v-if="step.feedback" class="step-feedback">{{ step.feedback }}</p><details v-if="step.content && step.kind !== 'user_prompt' && step.kind !== 'configuration_change'"><summary>{{ step.kind === 'reviewer_feedback' ? t('generate.detail.viewRawResponse') : step.kind === 'rendered_pdf' ? (run.templateId ? t('generate.detail.viewLatexSource') : t('generate.detail.viewHtmlCssSource')) : t('generate.detail.viewContent') }}</summary><pre>{{ step.content }}</pre></details><blockquote v-if="step.kind === 'user_prompt'">{{ step.content }}</blockquote><button v-if="step.kind === 'rendered_pdf'" class="text-button preview-link" @click="selectedStep = step.id">{{ t('generate.detail.previewThisPdf') }}</button></div></article>
            <article v-if="active && sortOrder === 'asc'" class="timeline-step live-step"><span class="timeline-icon">{{ activeIcon }}</span><div><h3>{{ stageLabel }}</h3><p class="step-meta">{{ t('generate.detail.inProgressNow') }}</p><p class="step-feedback">{{ t('generate.detail.workingMessage') }}</p></div></article>
          </div></div>
        </div>
        <aside class="panel preview-column"><div class="section-title"><div><p class="eyebrow">{{ t('generate.detail.artifactEyebrow') }}</p><h2>{{ selectedStep ? t('generate.detail.stagePdf') : t('generate.detail.finalPdf') }}</h2></div><button v-if="selectedStep && run.state === 'ready'" class="text-button" @click="selectedStep = undefined">{{ t('generate.detail.showFinal') }}</button></div><GenerationPdfPreview v-if="run.state === 'ready' || selectedStep" :generation-id="run.id" :step-id="selectedStep"/><div v-else class="empty-state compact">{{ t('generate.detail.previewPlaceholder') }}</div></aside>
      </section>

      <section v-if="run.state === 'ready'" class="panel revision-panel"><div><p class="eyebrow">{{ t('generate.detail.refineEyebrow') }}</p><h2>{{ t('generate.detail.askForChange') }}</h2><p>{{ t('generate.detail.reviseDescription') }}</p></div><form @submit.prevent="revise"><textarea v-model="prompt" rows="4" maxlength="4000" :placeholder="t('generate.detail.revisePlaceholder')" required/><div><small>{{ prompt.length }}/4000</small><button class="button primary" :disabled="submitting || !prompt.trim()">{{ submitting ? t('generate.queuing') : t('generate.detail.sendRevision') }}</button></div></form></section>

      <div v-if="showEdit" class="modal-backdrop" @click.self="showEdit = false"><section class="panel generation-modal" role="dialog" aria-modal="true" aria-labelledby="edit-generation-title"><header class="modal-header"><div><p class="eyebrow">{{ t('generate.detail.editModal.eyebrow') }}</p><h2 id="edit-generation-title">{{ t('generate.detail.editRegenerate') }}</h2></div><button class="modal-close" type="button" :aria-label="t('common.close')" :disabled="submitting" @click="showEdit = false">×</button></header><p class="edit-help">{{ t('generate.detail.editModal.helpText') }}</p><form @submit.prevent="regenerate"><div class="form-grid">
        <label><span>{{ t('generate.modal.documentTypeLabel') }}</span><select v-model="editForm.documentType"><option value="resume">{{ t('generate.modal.documentTypeResume') }}</option><option value="cover_letter">{{ t('generate.modal.documentTypeCoverLetter') }}</option></select></label><label><span>{{ t('generate.modal.languageLabel') }}</span><input v-model="editForm.language" required maxlength="40" /></label>
        <label><span>{{ t('generate.modal.profileLabel') }}</span><SearchCombobox :model-value="editForm.profileId" :selected-label="editProfileLabel" :search="searchEditProfileOptions" :placeholder="t('generate.modal.profilePlaceholder')" @update:model-value="value => editForm.profileId = value" /></label><label><span>{{ t('generate.modal.opportunityLabel') }}</span><input :value="title" disabled /></label>
        <label><span>{{ t('generate.modal.templateLabel') }}</span><SearchCombobox :model-value="editForm.templateId" :selected-label="editTemplateLabel" :search="searchEditTemplateOptions" clearable :clear-label="t('generate.modal.templateNone')" :placeholder="t('generate.modal.templateNone')" @update:model-value="value => editForm.templateId = value" /></label><label><span>{{ t('generate.modal.pageTargetLabel') }}</span><select v-model="editForm.pageTarget"><option value="one_page">{{ t('generate.pageTarget.onePage') }}</option><option value="two_pages">{{ t('generate.pageTarget.twoPages') }}</option><option value="flexible">{{ t('generate.pageTarget.flexible') }}</option></select></label>
        <label class="full"><span>{{ t('generate.modal.customInstructionsLabel') }}</span><textarea v-model="editForm.customInstructions" rows="3" maxlength="4000" /></label>
        <div class="full pipeline-choice"><label><input v-model="editForm.pipelineMode" type="radio" value="single" /> {{ t('generate.detail.editModal.pipelineSingle') }}</label><label><input v-model="editForm.pipelineMode" type="radio" value="multi" /> {{ t('generate.pipelineMode.multi') }}</label></div>
        <fieldset class="full model-card"><legend>{{ editForm.pipelineMode === 'single' ? t('generate.detail.editModal.workflowModelLegend') : t('generate.detail.editModal.writerModelLegend') }}</legend><div class="model-row"><label><span>{{ t('generate.modal.providerLabel') }}</span><select v-model="editForm.writer.connectionId" @change="changeConnection(editForm.writer)"><option value="" disabled>{{ t('generate.modal.providerPlaceholder') }}</option><option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option></select></label><label><span>{{ t('generate.modal.modelLabel') }}</span><select v-if="models[editForm.writer.connectionId]?.length" v-model="editForm.writer.model" required><option value="" disabled>{{ t('generate.modal.modelPlaceholder') }}</option><option v-if="editForm.writer.model && !models[editForm.writer.connectionId].includes(editForm.writer.model)" :value="editForm.writer.model">{{ editForm.writer.model }}</option><option v-for="model in models[editForm.writer.connectionId]" :key="model" :value="model">{{ model }}</option></select><input v-else v-model="editForm.writer.model" required :disabled="loadingModelConnections[editForm.writer.connectionId]" :placeholder="loadingModelConnections[editForm.writer.connectionId] ? t('generate.modal.loadingModels') : t('generate.modal.enterModelName')" /></label></div></fieldset>
        <template v-if="editForm.pipelineMode === 'multi'"><fieldset class="full model-card"><legend>{{ editForm.templateId ? t('generate.detail.editModal.templateApplierLegend') : t('generate.detail.editModal.documentDesignerLegend') }}</legend><div class="model-row"><label><span>{{ t('generate.modal.providerLabel') }}</span><select v-model="editForm.renderer.connectionId" @change="changeConnection(editForm.renderer)"><option value="" disabled>{{ t('generate.modal.providerPlaceholder') }}</option><option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option></select></label><label><span>{{ t('generate.modal.modelLabel') }}</span><select v-if="models[editForm.renderer.connectionId]?.length" v-model="editForm.renderer.model" required><option value="" disabled>{{ t('generate.modal.modelPlaceholder') }}</option><option v-if="editForm.renderer.model && !models[editForm.renderer.connectionId].includes(editForm.renderer.model)" :value="editForm.renderer.model">{{ editForm.renderer.model }}</option><option v-for="model in models[editForm.renderer.connectionId]" :key="model" :value="model">{{ model }}</option></select><input v-else v-model="editForm.renderer.model" required :disabled="loadingModelConnections[editForm.renderer.connectionId]" :placeholder="loadingModelConnections[editForm.renderer.connectionId] ? t('generate.modal.loadingModels') : t('generate.modal.enterModelName')" /></label></div></fieldset><fieldset class="full model-card"><legend>{{ t('generate.detail.editModal.visualReviewerLegend') }}</legend><div class="model-row"><label><span>{{ t('generate.modal.providerLabel') }}</span><select v-model="editForm.reviewer.connectionId" @change="changeConnection(editForm.reviewer)"><option value="" disabled>{{ t('generate.modal.providerPlaceholder') }}</option><option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option></select></label><label><span>{{ t('generate.modal.modelLabel') }}</span><select v-if="models[editForm.reviewer.connectionId]?.length" v-model="editForm.reviewer.model" required><option value="" disabled>{{ t('generate.modal.modelPlaceholder') }}</option><option v-if="editForm.reviewer.model && !models[editForm.reviewer.connectionId].includes(editForm.reviewer.model)" :value="editForm.reviewer.model">{{ editForm.reviewer.model }}</option><option v-for="model in models[editForm.reviewer.connectionId]" :key="model" :value="model">{{ model }}</option></select><input v-else v-model="editForm.reviewer.model" required :disabled="loadingModelConnections[editForm.reviewer.connectionId]" :placeholder="loadingModelConnections[editForm.reviewer.connectionId] ? t('generate.modal.loadingModels') : t('generate.modal.enterModelName')" /></label></div></fieldset></template>
      </div><div class="modal-actions"><button class="button" type="button" :disabled="submitting" @click="showEdit = false">{{ t('common.cancel') }}</button><button class="button primary" :disabled="!editReady || submitting">{{ submitting ? t('generate.detail.editModal.startingButton') : t('generate.detail.editModal.saveRegenerate') }}</button></div></form></section></div>
    </template>
  </div>
</template>

<style scoped>
.generation-detail{display:grid;gap:20px}.run-summary{display:flex;align-items:center;justify-content:space-between;gap:20px}.run-summary h2{margin:12px 0 6px}.run-summary p,.revision-panel p{margin:0;color:var(--muted)}.summary-actions{display:flex;flex-wrap:wrap;gap:10px}.detail-grid{--artifact-height:clamp(620px,72vh,780px);display:grid;grid-template-columns:minmax(0,.9fr) minmax(420px,1.1fr);gap:20px;align-items:stretch}.timeline-panel,.preview-column{height:var(--artifact-height);min-height:0}.timeline-panel{display:flex;flex-direction:column}.section-title{display:flex;align-items:flex-end;justify-content:space-between;gap:16px;margin-bottom:20px}.section-title h2,.section-title p{margin-bottom:0}.timeline-heading{align-items:center;flex:0 0 auto}.timeline-sort{display:inline-flex;align-items:center;gap:7px;padding:7px 10px;border:1px solid var(--line);border-radius:8px;background:var(--surface);color:var(--muted);cursor:pointer;font-size:11px;font-weight:700;white-space:nowrap}.timeline-sort span{color:var(--accent);font-size:16px}.timeline-scroll{min-height:0;margin:-10px -9px 0 -10px;overflow-y:auto;padding:10px 18px 14px 10px;scroll-padding-top:10px;scrollbar-color:var(--line) transparent;scrollbar-width:thin}.timeline{display:grid}.timeline-step{position:relative;display:grid;grid-template-columns:38px minmax(0,1fr);gap:15px;padding:0 0 26px}.timeline-step:not(:last-child)::before{position:absolute;top:38px;bottom:0;left:18px;width:1px;background:var(--line);content:''}.timeline-icon{position:relative;z-index:1;display:grid;width:38px;height:38px;place-items:center;border:1px solid var(--line);border-radius:50%;background:var(--surface);color:var(--accent);font-size:12px;font-weight:800}.timeline-step.system_warning .timeline-icon{background:#fff3d7;color:#8a6724}.timeline-step.configuration_change .timeline-icon{background:var(--accent-pale)}.timeline-step.live-step .timeline-icon{border-color:transparent;background:var(--accent);color:#fff;box-shadow:0 0 0 5px var(--accent-pale)}.timeline-step.live-step .timeline-icon::after{position:absolute;inset:-6px;border:2px solid transparent;border-top-color:var(--accent);border-right-color:var(--accent);border-radius:50%;content:'';animation:spin .85s linear infinite}.timeline-step.live-step .step-feedback{border:1px solid #c6d9cd;background:var(--accent-pale)}.timeline-step h3{margin:7px 0 3px;font-size:15px}.step-meta{margin:0 0 10px;color:var(--muted);font-size:11px}.step-feedback{padding:11px 13px;border-radius:8px;background:var(--surface-soft);font-size:13px;line-height:1.55}.timeline details{margin-top:10px}.timeline summary{cursor:pointer;color:var(--accent);font-size:12px;font-weight:700}.timeline pre{max-height:360px;overflow:auto;padding:14px;border-radius:9px;background:#17201d;color:#e8efeb;white-space:pre-wrap;overflow-wrap:anywhere;font:11px/1.55 ui-monospace,SFMono-Regular,Menlo,monospace}.timeline blockquote{margin:10px 0 0;padding:12px 14px;border-left:3px solid var(--warm);background:var(--surface-soft);font-size:13px;line-height:1.55}.preview-link{margin-top:10px;padding:0}.preview-column{position:sticky;top:24px;display:flex;flex-direction:column}.preview-column :deep(.generation-pdf){min-height:0;flex:1}.revision-panel{display:grid;grid-template-columns:minmax(240px,.7fr) minmax(320px,1.3fr);gap:30px;align-items:start}.revision-panel form{display:grid;gap:10px}.revision-panel form div{display:flex;align-items:center;justify-content:space-between}.revision-panel small{color:var(--muted)}.edit-help{margin:-10px 0 22px;color:var(--muted);font-size:13px}.generation-modal{width:min(860px,100%)}@keyframes spin{to{transform:rotate(360deg)}}@media(prefers-reduced-motion:reduce){.timeline-step.live-step .timeline-icon::after{animation-duration:2.5s}}@media(max-width:980px){.detail-grid,.revision-panel{grid-template-columns:1fr}.preview-column{position:static}}@media(max-width:600px){.run-summary{align-items:flex-start;flex-direction:column}.summary-actions{width:100%}.summary-actions .button{flex:1}.timeline-heading{align-items:flex-start;flex-direction:column}}
.timeline .timeline-step:not(:last-child)::after{position:absolute;z-index:2;top:max(48px,50%);left:14px;width:0;height:0;border-right:4px solid transparent;border-left:4px solid transparent;content:'';filter:drop-shadow(0 0 2px var(--surface))}.timeline.desc .timeline-step:not(:last-child)::after{border-bottom:7px solid var(--accent);transform:translateY(-7px)}.timeline.asc .timeline-step:not(:last-child)::after{border-top:7px solid var(--accent)}
</style>
