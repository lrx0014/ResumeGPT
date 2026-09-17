<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'

import { api } from '../lib/api'
import type { AgentDefault, GenerationInput, GenerationModelChoice, Job, LLMConnection, Profile, Template, TemplateKind } from '../lib/types'

const props = defineProps<{
  open: boolean
  opportunities: Job[]
  initialDocumentType: TemplateKind
}>()

const emit = defineEmits<{
  close: []
  queued: [result: { createdOpportunityIds: string[]; failedOpportunityIds: string[] }]
}>()

const profiles = ref<Profile[]>([])
const templates = ref<Template[]>([])
const connections = ref<LLMConnection[]>([])
const models = reactive<Record<string, string[]>>({})
const activeOpportunityIds = ref<string[]>([])
const loading = ref(false)
const submitting = ref(false)
const error = ref('')

const emptyChoice = (): GenerationModelChoice => ({ connectionId: '', model: '' })
const form = reactive<Omit<GenerationInput, 'opportunityId'>>({
  profileId: '', templateId: '', documentType: 'resume', language: 'English', pageTarget: 'one_page', customInstructions: '',
  pipelineMode: 'single', writer: emptyChoice(), renderer: emptyChoice(), reviewer: emptyChoice(),
})

const selectedOpportunities = computed(() => activeOpportunityIds.value.map(id => props.opportunities.find(item => item.id === id)).filter((item): item is Job => Boolean(item)))
const matchingTemplates = computed(() => templates.value.filter(item => item.state === 'ready' && item.format === 'latex' && item.kind === form.documentType))
const ready = computed(() => Boolean(activeOpportunityIds.value.length && form.profileId && form.templateId && form.writer.connectionId && form.writer.model && (form.pipelineMode === 'single' || (form.renderer.connectionId && form.renderer.model && form.reviewer.connectionId && form.reviewer.model))))
const documentLabel = computed(() => form.documentType === 'resume' ? 'CV' : 'cover letter')

async function discover(choice: GenerationModelChoice) {
  if (!choice.connectionId || models[choice.connectionId]) return
  try {
    models[choice.connectionId] = (await api.testLLMConnection(choice.connectionId)).models
    if (!choice.model) choice.model = models[choice.connectionId][0] ?? ''
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not load models.'
  }
}

async function prepare() {
  loading.value = true
  error.value = ''
  activeOpportunityIds.value = props.opportunities.map(item => item.id)
  form.documentType = props.initialDocumentType
  form.pageTarget = 'one_page'
  try {
    const [profileList, templateList, connectionList, storedDefaults] = await Promise.all([api.listProfiles(), api.listTemplates(), api.listLLMConnections(), api.getAgentDefaults()])
    profiles.value = profileList.items.filter(item => item.content.trim())
    templates.value = templateList.items
    connections.value = connectionList.items
    if (!profiles.value.some(item => item.id === form.profileId)) form.profileId = profiles.value[0]?.id ?? ''
    if (!matchingTemplates.value.some(item => item.id === form.templateId)) form.templateId = matchingTemplates.value[0]?.id ?? ''
    if (!connections.value.some(item => item.id === form.writer.connectionId)) {
      const defaults = Object.fromEntries(storedDefaults.items.map((item: AgentDefault) => [item.agent, item])) as Partial<Record<AgentDefault['agent'], AgentDefault>>
      const valid = (choice?: AgentDefault) => choice && connections.value.some(item => item.id === choice.connectionId) ? { connectionId: choice.connectionId, model: choice.model } : undefined
      form.writer = valid(defaults.writer) ?? { connectionId: connections.value[0]?.id ?? '', model: '' }
      form.renderer = valid(defaults.template_applier) ?? { ...form.writer }
      form.reviewer = valid(defaults.visual_reviewer) ?? { ...form.writer }
    }
    await Promise.all([discover(form.writer), discover(form.renderer), discover(form.reviewer)])
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not load generation options.'
  } finally {
    loading.value = false
  }
}

function close() {
  if (!submitting.value) emit('close')
}

async function submit() {
  if (!ready.value) return
  submitting.value = true
  error.value = ''
  const shared = JSON.parse(JSON.stringify(form)) as Omit<GenerationInput, 'opportunityId'>
  if (shared.pipelineMode === 'single') {
    shared.renderer = { ...shared.writer }
    shared.reviewer = { ...shared.writer }
  }
  const ids = [...activeOpportunityIds.value]
  const results = await Promise.allSettled(ids.map(opportunityId => api.createGeneration({ ...shared, opportunityId })))
  const createdOpportunityIds = ids.filter((_, index) => results[index].status === 'fulfilled')
  const failedOpportunityIds = ids.filter((_, index) => results[index].status === 'rejected')
  emit('queued', { createdOpportunityIds, failedOpportunityIds })
  if (failedOpportunityIds.length) {
    activeOpportunityIds.value = failedOpportunityIds
    const firstFailure = results.find(result => result.status === 'rejected') as PromiseRejectedResult | undefined
    const detail = firstFailure?.reason instanceof Error ? firstFailure.reason.message : 'Some generation tasks could not be created.'
    error.value = `${createdOpportunityIds.length} queued, ${failedOpportunityIds.length} failed. ${detail}`
  } else {
    emit('close')
  }
  submitting.value = false
}

watch(() => props.open, value => { if (value) void prepare() })
watch(() => form.documentType, () => {
  if (!matchingTemplates.value.some(item => item.id === form.templateId)) form.templateId = matchingTemplates.value[0]?.id ?? ''
})
watch(() => form.writer.connectionId, () => { void discover(form.writer) })
watch(() => form.renderer.connectionId, () => { void discover(form.renderer) })
watch(() => form.reviewer.connectionId, () => { void discover(form.reviewer) })
</script>

<template>
  <div v-if="open" class="modal-backdrop" @click.self="close">
    <section class="panel generation-modal batch-generation-modal" role="dialog" aria-modal="true" aria-labelledby="batch-generation-title">
      <header class="modal-header">
        <div><p class="eyebrow">{{ activeOpportunityIds.length > 1 ? 'Batch generation' : 'Quick generation' }}</p><h2 id="batch-generation-title">Create {{ documentLabel }}{{ activeOpportunityIds.length > 1 ? 's' : '' }}</h2></div>
        <button class="modal-close" type="button" aria-label="Close" :disabled="submitting" @click="close">×</button>
      </header>
      <p class="dialog-intro">The same Profile, Template, and model settings will be used for {{ activeOpportunityIds.length }} {{ activeOpportunityIds.length === 1 ? 'job opportunity' : 'job opportunities' }}. Each one creates a separate tracked application.</p>
      <div class="target-list" aria-label="Selected job opportunities">
        <span v-for="item in selectedOpportunities.slice(0, 5)" :key="item.id">{{ item.title }}<small>{{ item.company }}</small></span>
        <span v-if="selectedOpportunities.length > 5">+{{ selectedOpportunities.length - 5 }} more</span>
      </div>
      <p v-if="error" class="notice error" role="alert">{{ error }}</p>
      <div v-if="loading" class="empty-state compact">Loading generation options…</div>
      <form v-else @submit.prevent="submit">
        <p v-if="!profiles.length" class="notice">Create a Profile with saved content before generating documents.</p>
        <p v-else-if="!matchingTemplates.length" class="notice">Add a ready LaTeX {{ form.documentType === 'resume' ? 'CV' : 'cover letter' }} template before continuing.</p>
        <p v-else-if="!connections.length" class="notice">Add an LLM connection in Settings before generating documents.</p>
        <div class="form-grid">
          <label><span>Document type</span><select v-model="form.documentType"><option value="resume">CV</option><option value="cover_letter">Cover letter</option></select></label>
          <label><span>Output language</span><input v-model="form.language" required maxlength="40" /></label>
          <label><span>Profile</span><select v-model="form.profileId" required><option value="" disabled>Select a profile</option><option v-for="item in profiles" :key="item.id" :value="item.id">{{ item.name }}</option></select></label>
          <label><span>LaTeX template</span><select v-model="form.templateId" required><option value="" disabled>Select a template</option><option v-for="item in matchingTemplates" :key="item.id" :value="item.id">{{ item.name }}</option></select></label>
          <label><span>Page target</span><select v-model="form.pageTarget"><option value="one_page">One page</option><option value="two_pages">Two pages</option><option value="flexible">Flexible</option></select></label>
          <label class="full"><span>Custom instructions</span><textarea v-model="form.customInstructions" rows="3" maxlength="4000" placeholder="Optional emphasis or tone shared by these documents." /></label>
          <div class="full pipeline-choice"><label><input v-model="form.pipelineMode" type="radio" value="single" /> One model</label><label><input v-model="form.pipelineMode" type="radio" value="multi" /> Specialized models</label></div>
          <fieldset class="full model-card"><legend>{{ form.pipelineMode === 'single' ? 'Workflow model' : 'Writer model' }}</legend><div class="model-row"><label><span>Connection</span><select v-model="form.writer.connectionId"><option value="" disabled>Select connection</option><option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option></select></label><label><span>Model</span><input v-model="form.writer.model" list="batch-writer-models" placeholder="Model name" /><datalist id="batch-writer-models"><option v-for="model in models[form.writer.connectionId] || []" :key="model" :value="model" /></datalist></label></div><small v-if="form.pipelineMode === 'single'">A text-only model can still generate a PDF; visual QA will be skipped with a warning.</small></fieldset>
          <template v-if="form.pipelineMode === 'multi'">
            <fieldset class="full model-card"><legend>Template applier</legend><div class="model-row"><label><span>Connection</span><select v-model="form.renderer.connectionId"><option value="" disabled>Select connection</option><option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option></select></label><label><span>Model</span><input v-model="form.renderer.model" list="batch-renderer-models" /><datalist id="batch-renderer-models"><option v-for="model in models[form.renderer.connectionId] || []" :key="model" :value="model" /></datalist></label></div></fieldset>
            <fieldset class="full model-card"><legend>Visual reviewer</legend><div class="model-row"><label><span>Connection</span><select v-model="form.reviewer.connectionId"><option value="" disabled>Select connection</option><option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option></select></label><label><span>Model</span><input v-model="form.reviewer.model" list="batch-reviewer-models" /><datalist id="batch-reviewer-models"><option v-for="model in models[form.reviewer.connectionId] || []" :key="model" :value="model" /></datalist></label></div></fieldset>
          </template>
        </div>
        <div class="modal-actions"><button class="button" type="button" :disabled="submitting" @click="close">Cancel</button><button class="button primary" :disabled="!ready || submitting">{{ submitting ? 'Queuing…' : `Create ${activeOpportunityIds.length} ${documentLabel}${activeOpportunityIds.length === 1 ? '' : 's'}` }}</button></div>
      </form>
    </section>
  </div>
</template>

<style scoped>
.dialog-intro { margin: -8px 0 14px; color: var(--muted); font-size: 13px; line-height: 1.55; }
.target-list { display: flex; flex-wrap: wrap; gap: 7px; margin-bottom: 20px; }
.target-list > span { display: inline-flex; align-items: center; gap: 6px; padding: 6px 9px; border: 1px solid var(--line); border-radius: 999px; background: var(--surface-soft); font-size: 11px; font-weight: 700; }
.target-list small { color: var(--muted); font-size: 10px; font-weight: 500; }
.batch-generation-modal { width: min(860px, 100%); }
</style>
