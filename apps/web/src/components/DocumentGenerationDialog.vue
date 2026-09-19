<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import { api } from '../lib/api'
import { useModelDiscovery } from '../lib/modelDiscovery'
import type { AgentDefault, GenerationInput, GenerationModelChoice, Job, LLMConnection, Profile, Template, TemplateKind } from '../lib/types'

const { t } = useI18n()
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
const activeOpportunityIds = ref<string[]>([])
const loading = ref(false)
const submitting = ref(false)
const error = ref('')
const useSpecifiedModel = ref(false)

const emptyChoice = (): GenerationModelChoice => ({ connectionId: '', model: '' })
const form = reactive<Omit<GenerationInput, 'opportunityId'>>({
  profileId: '', templateId: '', documentType: 'resume', language: 'English', pageTarget: 'one_page', customInstructions: '',
  pipelineMode: 'multi', writer: emptyChoice(), renderer: emptyChoice(), reviewer: emptyChoice(),
})
const writerDefault = ref<GenerationModelChoice | null>(null)
const templateApplierDefault = ref<GenerationModelChoice | null>(null)
const designerDefault = ref<GenerationModelChoice | null>(null)
const reviewerDefault = ref<GenerationModelChoice | null>(null)

const selectedOpportunities = computed(() => activeOpportunityIds.value.map(id => props.opportunities.find(item => item.id === id)).filter((item): item is Job => Boolean(item)))
const matchingTemplates = computed(() => templates.value.filter(item => item.state === 'ready' && item.format === 'latex' && item.kind === form.documentType))
const ready = computed(() => Boolean(activeOpportunityIds.value.length && form.profileId && form.writer.connectionId && form.writer.model && (form.pipelineMode === 'single' || (form.renderer.connectionId && form.renderer.model && form.reviewer.connectionId && form.reviewer.model))))
const documentLabel = computed(() => form.documentType === 'resume' ? t('generate.dialog.documentLabelCv') : t('generate.dialog.documentLabelCoverLetter'))

const { models, loadingModelConnections, discover, changeConnection } = useModelDiscovery(
  message => { error.value = message },
  () => t('generate.errors.loadModels'),
)

async function prepare() {
  loading.value = true
  error.value = ''
  activeOpportunityIds.value = props.opportunities.map(item => item.id)
  form.documentType = props.initialDocumentType
  form.pageTarget = 'one_page'
  form.templateId = ''
  useSpecifiedModel.value = false
  form.pipelineMode = 'multi'
  try {
    const [profileList, templateList, connectionList, storedDefaults] = await Promise.all([api.listProfiles(), api.listTemplates(), api.listLLMConnections(), api.getAgentDefaults()])
    profiles.value = profileList.items.filter(item => item.hasContent)
    templates.value = templateList.items
    connections.value = connectionList.items
    if (!profiles.value.some(item => item.id === form.profileId)) form.profileId = profiles.value[0]?.id ?? ''
    const defaults = Object.fromEntries(storedDefaults.items.map((item: AgentDefault) => [item.agent, item])) as Partial<Record<AgentDefault['agent'], AgentDefault>>
    const valid = (choice?: AgentDefault) => choice && connections.value.some(item => item.id === choice.connectionId) ? { connectionId: choice.connectionId, model: choice.model, maxTokens: choice.maxTokens } : undefined
    const fallback = valid(defaults.writer) ?? { connectionId: connections.value[0]?.id ?? '', model: '' }
    writerDefault.value = { ...fallback }
    templateApplierDefault.value = valid(defaults.template_applier) ?? { ...fallback }
    designerDefault.value = valid(defaults.document_designer) ?? { ...fallback }
    reviewerDefault.value = valid(defaults.visual_reviewer) ?? { ...fallback }
    form.writer = { ...fallback }
    form.renderer = { ...(designerDefault.value ?? fallback) }
    form.reviewer = { ...(reviewerDefault.value ?? fallback) }
    await Promise.all([discover(form.writer), discover(form.renderer), discover(form.reviewer)])
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('generate.dialog.errors.loadOptions')
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
    const detail = firstFailure?.reason instanceof Error ? firstFailure.reason.message : t('generate.dialog.errors.someTasksFailed')
    error.value = t('generate.dialog.errors.partialFailure', { created: createdOpportunityIds.length, failed: failedOpportunityIds.length, detail })
  } else {
    emit('close')
  }
  submitting.value = false
}

watch(() => props.open, value => { if (value) void prepare() })
watch(() => form.documentType, () => {
  if (form.templateId && !matchingTemplates.value.some(item => item.id === form.templateId)) form.templateId = ''
})
watch(() => form.templateId, () => {
  if (useSpecifiedModel.value) return
  const choice = form.templateId ? templateApplierDefault.value : designerDefault.value
  if (choice) form.renderer = { ...choice }
  void discover(form.renderer)
})
watch(useSpecifiedModel, value => {
  form.pipelineMode = value ? 'single' : 'multi'
  if (value) {
    form.renderer = { ...form.writer }
    form.reviewer = { ...form.writer }
  } else {
    form.writer = { ...(writerDefault.value ?? form.writer) }
    form.renderer = { ...((form.templateId ? templateApplierDefault.value : designerDefault.value) ?? form.writer) }
    form.reviewer = { ...(reviewerDefault.value ?? form.writer) }
    void Promise.all([discover(form.renderer), discover(form.reviewer)])
  }
})
</script>

<template>
  <div v-if="open" class="modal-backdrop" @click.self="close">
    <section class="panel generation-modal batch-generation-modal" role="dialog" aria-modal="true" aria-labelledby="batch-generation-title">
      <header class="modal-header">
        <div><p class="eyebrow">{{ activeOpportunityIds.length > 1 ? t('generate.dialog.eyebrowBatch') : t('generate.dialog.eyebrowQuick') }}</p><h2 id="batch-generation-title">{{ t('generate.dialog.createTitle', { label: documentLabel }, activeOpportunityIds.length) }}</h2></div>
        <button class="modal-close" type="button" :aria-label="t('common.close')" :disabled="submitting" @click="close">×</button>
      </header>
      <p class="dialog-intro">{{ t('generate.dialog.intro', { count: activeOpportunityIds.length }, activeOpportunityIds.length) }}</p>
      <div class="target-list" :aria-label="t('generate.dialog.selectedOpportunities')">
        <span v-for="item in selectedOpportunities.slice(0, 5)" :key="item.id">{{ item.title }}<small>{{ item.company }}</small></span>
        <span v-if="selectedOpportunities.length > 5">{{ t('generate.dialog.moreCount', { count: selectedOpportunities.length - 5 }) }}</span>
      </div>
      <p v-if="error" class="notice error" role="alert">{{ error }}</p>
      <div v-if="loading" class="empty-state compact">{{ t('generate.dialog.loadingOptions') }}</div>
      <form v-else @submit.prevent="submit">
        <p v-if="!profiles.length" class="notice">{{ t('generate.dialog.noProfilesNotice') }}</p>
        <p v-else-if="!connections.length" class="notice">{{ t('generate.dialog.noConnectionsNotice') }}</p>
        <div class="form-grid">
          <label><span>{{ t('generate.modal.documentTypeLabel') }}</span><select v-model="form.documentType"><option value="resume">{{ t('generate.dialog.documentTypeCv') }}</option><option value="cover_letter">{{ t('generate.modal.documentTypeCoverLetter') }}</option></select></label>
          <label><span>{{ t('generate.modal.languageLabel') }}</span><input v-model="form.language" required maxlength="40" /></label>
          <label><span>{{ t('generate.modal.profileLabel') }}</span><select v-model="form.profileId" required><option value="" disabled>{{ t('generate.modal.profilePlaceholder') }}</option><option v-for="item in profiles" :key="item.id" :value="item.id">{{ item.name }}</option></select></label>
          <label><span>{{ t('generate.modal.templateLabel') }}</span><select v-model="form.templateId"><option value="">{{ t('generate.modal.templateNone') }}</option><option v-for="item in matchingTemplates" :key="item.id" :value="item.id">{{ item.name }}</option></select><small v-if="!form.templateId">{{ t('generate.modal.templateHelp') }}</small></label>
          <label><span>{{ t('generate.modal.pageTargetLabel') }}</span><select v-model="form.pageTarget"><option value="one_page">{{ t('generate.pageTarget.onePage') }}</option><option value="two_pages">{{ t('generate.pageTarget.twoPages') }}</option><option value="flexible">{{ t('generate.pageTarget.flexible') }}</option></select></label>
          <label class="full"><span>{{ t('generate.modal.customInstructionsLabel') }}</span><textarea v-model="form.customInstructions" rows="3" maxlength="4000" :placeholder="t('generate.dialog.customInstructionsPlaceholder')" /></label>
          <div class="full model-routing"><label class="model-switch"><input v-model="useSpecifiedModel" type="checkbox" role="switch" /><span class="model-switch-track" aria-hidden="true"><span /></span><span>{{ t('generate.modal.useSpecificModel') }}</span></label><p v-if="!useSpecifiedModel">{{ t('generate.modal.defaultModelsHelp') }} <RouterLink to="/settings">{{ t('generate.modal.configureAgentModels') }}</RouterLink></p><p v-else>{{ t('generate.dialog.specificModelHelp') }}</p></div>
          <fieldset v-if="useSpecifiedModel" class="full model-card"><legend>{{ t('generate.modal.modelForEveryAgent') }}</legend><div class="model-row"><label><span>{{ t('generate.modal.providerLabel') }}</span><select v-model="form.writer.connectionId" @change="changeConnection(form.writer)"><option value="" disabled>{{ t('generate.modal.providerPlaceholder') }}</option><option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option></select></label><label><span>{{ t('generate.modal.modelLabel') }}</span><select v-if="models[form.writer.connectionId]?.length" v-model="form.writer.model" required><option value="" disabled>{{ t('generate.modal.modelPlaceholder') }}</option><option v-for="model in models[form.writer.connectionId]" :key="model" :value="model">{{ model }}</option></select><input v-else v-model="form.writer.model" required :disabled="loadingModelConnections[form.writer.connectionId]" :placeholder="loadingModelConnections[form.writer.connectionId] ? t('generate.modal.loadingModels') : t('generate.modal.enterModelName')" /></label></div><small>{{ t('generate.modal.textOnlyModelHelp') }}</small></fieldset>
        </div>
        <div class="modal-actions"><button class="button" type="button" :disabled="submitting" @click="close">{{ t('common.cancel') }}</button><button class="button primary" :disabled="!ready || submitting">{{ submitting ? t('generate.queuing') : t('generate.dialog.submitCount', { count: activeOpportunityIds.length, label: documentLabel }, activeOpportunityIds.length) }}</button></div>
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
