<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { onBeforeRouteLeave, useRouter } from 'vue-router'

import ConfirmDialog from '../components/ConfirmDialog.vue'
import PageHeader from '../components/PageHeader.vue'
import UnsavedChangesDialog from '../components/UnsavedChangesDialog.vue'
import { api } from '../lib/api'
import { applyTheme } from '../lib/preferences'
import { toast } from '../lib/toast'
import { interfaceLanguages, loadLocaleMessages, normalizeInterfaceLocale, type InterfaceLocale } from '../plugins/i18n'
import type { AgentDefault, AgentKind, GenerationModelChoice, LLMConnection, LLMConnectionInput, SettingsPreferences } from '../lib/types'

const { locale, t } = useI18n()
const router = useRouter()
const loading = ref(true)
const savingPreferences = ref(false)
const switchingLanguage = ref(false)
const savingConnection = ref(false)
const savingAgentDefaults = ref(false)
const loadingAgentModels = ref<AgentKind | ''>('')
const testingId = ref('')
const deletingId = ref('')
const pendingDelete = ref<LLMConnection | null>(null)
const error = ref('')
const connections = ref<LLMConnection[]>([])
const capabilities = ref<Record<string, boolean>>({})
const editingId = ref('')
const showConnectionForm = ref(false)
const applyToAllAgents = ref(false)
const sharedAgentModel = ref('')
const baselineReady = ref(false)
const preferencesSnapshot = ref('')
const agentDefaultsSnapshot = ref('')
const providerFormSnapshot = ref('')
const showUnsavedDialog = ref(false)
const savingBeforeLeave = ref(false)
const pendingDestination = ref('')
let allowNavigation = false
const connectionResults = reactive<Record<string, string>>({})
const agentModels = reactive<Record<string, string[]>>({})
const agentDefaults = reactive<Record<AgentKind, GenerationModelChoice>>({
  writer: { connectionId: '', model: '' },
  template_applier: { connectionId: '', model: '' },
  document_designer: { connectionId: '', model: '' },
  visual_reviewer: { connectionId: '', model: '' },
  job_import: { connectionId: '', model: '' },
  job_hunter: { connectionId: '', model: '' },
})
const agentDefinitions: { kind: AgentKind; requirementCount: number }[] = [
  { kind: 'writer', requirementCount: 4 },
  { kind: 'template_applier', requirementCount: 4 },
  { kind: 'document_designer', requirementCount: 4 },
  { kind: 'visual_reviewer', requirementCount: 3 },
  { kind: 'job_import', requirementCount: 3 },
  { kind: 'job_hunter', requirementCount: 3 },
]

function agentDefinitionKey(kind: AgentKind, field: 'name' | 'summary') {
  return `settings.agents.definitions.${kind}.${field}`
}

function agentRequirementKey(kind: AgentKind, index: number) {
  return `settings.agents.definitions.${kind}.requirements.${index}`
}
const preferences = reactive<Pick<SettingsPreferences, 'interfaceLanguage' | 'theme'>>({ interfaceLanguage: 'en', theme: 'system' })
const connectionForm = reactive<LLMConnectionInput>({
  name: '', executionMode: 'local', provider: 'ollama', baseUrl: 'http://host.docker.internal:11434',
  apiToken: '', clearApiToken: false,
})

function serializePreferences() {
  return JSON.stringify({ interfaceLanguage: preferences.interfaceLanguage, theme: preferences.theme })
}

function serializeAgentDefaults() {
  return JSON.stringify(agentDefinitions.map(({ kind }) => ({ kind, ...agentDefaults[kind] })))
}

function serializeProviderForm() {
  return JSON.stringify({ ...connectionForm, applyToAllAgents: applyToAllAgents.value, sharedAgentModel: sharedAgentModel.value })
}

const preferencesDirty = computed(() => baselineReady.value && serializePreferences() !== preferencesSnapshot.value)
const agentDefaultsDirty = computed(() => baselineReady.value && serializeAgentDefaults() !== agentDefaultsSnapshot.value)
const providerFormDirty = computed(() => baselineReady.value && showConnectionForm.value && serializeProviderForm() !== providerFormSnapshot.value)
const hasUnsavedChanges = computed(() => preferencesDirty.value || agentDefaultsDirty.value || providerFormDirty.value)

const providerOptions = computed(() => connectionForm.executionMode === 'cloud'
  ? [{ value: 'openai', label: t('settings.connections.providerNames.openai') }, { value: 'openai_compatible', label: t('settings.connections.providerNames.openaiCompatible') }]
  : [{ value: 'ollama', label: t('settings.connections.providerNames.ollama') }, { value: 'openai_compatible', label: t('settings.connections.providerNames.openaiCompatible') }])

function providerLabel(provider: LLMConnection['provider']) {
  return ({
    openai: t('settings.connections.providerNames.openai'),
    openai_compatible: t('settings.connections.providerNames.openaiCompatible'),
    ollama: t('settings.connections.providerNames.ollama'),
  })[provider]
}

function resetConnectionForm() {
  editingId.value = ''
  applyToAllAgents.value = false
  sharedAgentModel.value = ''
  Object.assign(connectionForm, { name: '', executionMode: 'local', provider: 'ollama', baseUrl: 'http://host.docker.internal:11434', apiToken: '', clearApiToken: false })
}

function addConnection() {
  resetConnectionForm()
  providerFormSnapshot.value = serializeProviderForm()
  showConnectionForm.value = true
  error.value = ''
}

function editConnection(item: LLMConnection) {
  editingId.value = item.id
  applyToAllAgents.value = false
  sharedAgentModel.value = ''
  Object.assign(connectionForm, { name: item.name, executionMode: item.executionMode, provider: item.provider, baseUrl: item.baseUrl, apiToken: '', clearApiToken: false })
  providerFormSnapshot.value = serializeProviderForm()
  showConnectionForm.value = true
  error.value = ''
}

function closeConnectionForm() {
  showConnectionForm.value = false
  resetConnectionForm()
  providerFormSnapshot.value = serializeProviderForm()
}

watch(() => connectionForm.executionMode, mode => {
  if (mode === 'cloud' && connectionForm.provider === 'ollama') connectionForm.provider = 'openai'
  if (mode === 'local' && connectionForm.provider === 'openai') connectionForm.provider = 'ollama'
})

watch(() => connectionForm.provider, provider => {
  if (!editingId.value || connectionForm.baseUrl === '' || ['https://api.openai.com/v1', 'http://host.docker.internal:11434'].includes(connectionForm.baseUrl)) {
    if (provider === 'openai') connectionForm.baseUrl = 'https://api.openai.com/v1'
    if (provider === 'ollama') connectionForm.baseUrl = 'http://host.docker.internal:11434'
  }
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [storedPreferences, storedConnections, system, storedAgentDefaults] = await Promise.all([
      api.getSettings(), api.listLLMConnections(), api.capabilities(), api.getAgentDefaults(),
    ])
    Object.assign(preferences, { interfaceLanguage: storedPreferences.interfaceLanguage, theme: storedPreferences.theme })
    connections.value = storedConnections.items
    capabilities.value = system.features
    for (const item of storedAgentDefaults.items) Object.assign(agentDefaults[item.agent], { connectionId: item.connectionId, model: item.model })
    const nextLocale = normalizeInterfaceLocale(storedPreferences.interfaceLanguage)
    await loadLocaleMessages(nextLocale)
    locale.value = nextLocale
    document.documentElement.lang = storedPreferences.interfaceLanguage
    applyTheme(storedPreferences.theme)
    preferencesSnapshot.value = serializePreferences()
    agentDefaultsSnapshot.value = serializeAgentDefaults()
    providerFormSnapshot.value = serializeProviderForm()
    baselineReady.value = true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('settings.errors.loadFailed')
  } finally {
    loading.value = false
  }
}

async function loadAgentModels(kind: AgentKind, force = false) {
  const choice = agentDefaults[kind]
  if (!choice.connectionId) return
  loadingAgentModels.value = kind
  try {
    if (force || !agentModels[choice.connectionId]) agentModels[choice.connectionId] = (await api.testLLMConnection(choice.connectionId)).models
    if (!choice.model) choice.model = agentModels[choice.connectionId][0] ?? ''
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('settings.errors.loadAgentModelsFailed')
  } finally {
    loadingAgentModels.value = ''
  }
}

function changeAgentConnection(kind: AgentKind) {
  agentDefaults[kind].model = ''
  void loadAgentModels(kind)
}

async function saveAgentDefaults() {
  const partial = agentDefinitions.find(({ kind }) => Boolean(agentDefaults[kind].connectionId) !== Boolean(agentDefaults[kind].model))
  if (partial) {
    error.value = t('settings.agents.validation.missingField', { name: t(agentDefinitionKey(partial.kind, 'name')) })
    return false
  }
  savingAgentDefaults.value = true
  error.value = ''
  try {
    const items: AgentDefault[] = agentDefinitions.flatMap(({ kind }) => {
      const choice = agentDefaults[kind]
      return choice.connectionId && choice.model ? [{ agent: kind, ...choice }] : []
    })
    await api.updateAgentDefaults(items)
    agentDefaultsSnapshot.value = serializeAgentDefaults()
    toast.success(t('settings.agents.savedToast'))
    return true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('settings.errors.saveAgentDefaultsFailed')
    return false
  } finally {
    savingAgentDefaults.value = false
  }
}

async function savePreferences() {
  savingPreferences.value = true
  error.value = ''
  try {
    const saved = await api.updateSettings({ ...preferences })
    const nextLocale = normalizeInterfaceLocale(saved.interfaceLanguage)
    await loadLocaleMessages(nextLocale)
    locale.value = nextLocale
    document.documentElement.lang = saved.interfaceLanguage
    applyTheme(saved.theme)
    preferencesSnapshot.value = serializePreferences()
    toast.success(t('settings.interfaceSaved'))
    return true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('settings.interfaceSaveError')
    return false
  } finally {
    savingPreferences.value = false
  }
}

async function switchInterfaceLanguage(language: InterfaceLocale) {
  if (switchingLanguage.value || preferences.interfaceLanguage === language) return
  const previous = preferences.interfaceLanguage
  preferences.interfaceLanguage = language
  switchingLanguage.value = true
  try {
    await loadLocaleMessages(language)
    locale.value = language
    document.documentElement.lang = language
    const saved = await api.updateSettings({ ...preferences })
    Object.assign(preferences, { interfaceLanguage: saved.interfaceLanguage, theme: saved.theme })
    preferencesSnapshot.value = serializePreferences()
    toast.success(t('settings.languageSaved'))
  } catch (cause) {
    preferences.interfaceLanguage = previous
    locale.value = previous
    document.documentElement.lang = previous
    error.value = cause instanceof Error ? cause.message : t('settings.languageError')
  } finally {
    switchingLanguage.value = false
  }
}

async function saveConnection() {
  const assignToAllAgents = !editingId.value && applyToAllAgents.value
  const model = sharedAgentModel.value.trim()
  if (assignToAllAgents && !model) {
    error.value = t('settings.connections.validation.missingSharedModel')
    return false
  }
  savingConnection.value = true
  error.value = ''
  try {
    const saved = editingId.value
      ? await api.updateLLMConnection(editingId.value, { ...connectionForm })
      : await api.createLLMConnection({ ...connectionForm })
    const index = connections.value.findIndex(item => item.id === saved.id)
    if (index >= 0) connections.value[index] = saved
    else connections.value.push(saved)
    connections.value.sort((left, right) => left.name.localeCompare(right.name))
    if (assignToAllAgents) {
      const items: AgentDefault[] = agentDefinitions.map(({ kind }) => ({ agent: kind, connectionId: saved.id, model }))
      try {
        await api.updateAgentDefaults(items)
        for (const { kind } of agentDefinitions) Object.assign(agentDefaults[kind], { connectionId: saved.id, model })
        agentModels[saved.id] = [model]
        agentDefaultsSnapshot.value = serializeAgentDefaults()
      } catch (cause) {
        closeConnectionForm()
        error.value = cause instanceof Error
          ? t('settings.connections.assignAgentsError', { reason: cause.message })
          : t('settings.connections.assignAgentsErrorGeneric')
        return false
      }
      toast.success(t('settings.connections.savedAssignedToast'))
    } else {
      toast.success(t('settings.connections.savedToast'))
    }
    closeConnectionForm()
    return true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('settings.connections.saveError')
    return false
  } finally {
    savingConnection.value = false
  }
}

async function testConnection(item: LLMConnection) {
  testingId.value = item.id
  error.value = ''
  try {
    const result = await api.testLLMConnection(item.id, true)
    connectionResults[item.id] = t('settings.connections.testResultAvailable', { count: result.models.length }, result.models.length)
    agentModels[item.id] = result.models
    toast.success(result.models.length
      ? t('settings.connections.testSucceededWithModels', { models: `${result.models.slice(0, 8).join(', ')}${result.models.length > 8 ? '…' : ''}` })
      : t('settings.connections.testSucceededNoModels'))
  } catch (cause) {
    connectionResults[item.id] = t('settings.connections.testFailed')
    error.value = cause instanceof Error ? cause.message : t('settings.connections.testError')
  } finally {
    testingId.value = ''
  }
}

async function deleteConnection() {
  const item = pendingDelete.value
  if (!item) return
  deletingId.value = item.id
  error.value = ''
  try {
    await api.deleteLLMConnection(item.id)
    connections.value = connections.value.filter(candidate => candidate.id !== item.id)
    for (const definition of agentDefinitions) {
      if (agentDefaults[definition.kind].connectionId === item.id) Object.assign(agentDefaults[definition.kind], { connectionId: '', model: '' })
    }
    agentDefaultsSnapshot.value = serializeAgentDefaults()
    pendingDelete.value = null
    if (editingId.value === item.id) closeConnectionForm()
    toast.success(t('settings.connections.deletedToast'))
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('settings.connections.deleteError')
  } finally {
    deletingId.value = ''
  }
}

function keepEditing() {
  showUnsavedDialog.value = false
  pendingDestination.value = ''
}

async function continueNavigation() {
  const destination = pendingDestination.value
  showUnsavedDialog.value = false
  pendingDestination.value = ''
  if (!destination) return
  allowNavigation = true
  try {
    await router.push(destination)
  } finally {
    allowNavigation = false
  }
}

async function saveBeforeLeaving() {
  savingBeforeLeave.value = true
  try {
    if (providerFormDirty.value && !(await saveConnection())) return
    if (agentDefaultsDirty.value && !(await saveAgentDefaults())) return
    if (preferencesDirty.value && !(await savePreferences())) return
    await continueNavigation()
  } finally {
    savingBeforeLeave.value = false
  }
}

function discardBeforeLeaving() {
  if (!savingBeforeLeave.value) void continueNavigation()
}

onBeforeRouteLeave(to => {
  if (allowNavigation || !hasUnsavedChanges.value) return true
  pendingDestination.value = to.fullPath
  showUnsavedDialog.value = true
  return false
})

onMounted(load)
</script>

<template>
  <div class="page settings-page">
    <PageHeader :title="t('settings.title')" :description="t('settings.description')" />

    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <div v-if="loading" class="empty-state">{{ t('settings.loadingSettings') }}</div>

    <template v-else>
      <section class="settings-section">
        <div class="section-heading">
          <div><p class="eyebrow">{{ t('settings.connections.eyebrow') }}</p><h2>{{ t('settings.connections.title') }}</h2><p>{{ t('settings.connections.description') }}</p></div>
          <button class="button primary" type="button" @click="addConnection">{{ t('settings.connections.addButton') }}</button>
        </div>

        <form v-if="showConnectionForm" class="panel form-grid connection-form" @submit.prevent="saveConnection">
          <h3 class="full">{{ editingId ? t('settings.connections.form.titleEdit') : t('settings.connections.form.titleNew') }}</h3>
          <label><span>{{ t('settings.connections.form.nameLabel') }}</span><input v-model="connectionForm.name" required maxlength="120" :placeholder="t('settings.connections.form.namePlaceholder')" /></label>
          <label><span>{{ t('settings.connections.form.executionModeLabel') }}</span><select v-model="connectionForm.executionMode"><option value="local">{{ t('settings.connections.form.executionModeLocalOption') }}</option><option value="cloud">{{ t('settings.connections.form.executionModeCloudOption') }}</option></select></label>
          <label><span>{{ t('settings.connections.form.providerLabel') }}</span><select v-model="connectionForm.provider"><option v-for="option in providerOptions" :key="option.value" :value="option.value">{{ option.label }}</option></select></label>
          <label><span>{{ t('settings.connections.form.baseUrlLabel') }}</span><input v-model="connectionForm.baseUrl" required type="url" maxlength="2048" /></label>
          <label class="full"><span>{{ t('settings.connections.form.apiTokenLabel') }}</span><input v-model="connectionForm.apiToken" type="password" maxlength="8192" autocomplete="new-password" :placeholder="editingId ? t('settings.connections.form.apiTokenPlaceholderEdit') : connectionForm.provider === 'ollama' ? t('settings.connections.form.apiTokenPlaceholderOllama') : t('settings.connections.form.apiTokenPlaceholderDefault')" /></label>
          <label v-if="editingId" class="checkbox-field full"><input v-model="connectionForm.clearApiToken" type="checkbox" /><span>{{ t('settings.connections.form.clearApiTokenLabel') }}</span></label>
          <label v-if="!editingId" class="checkbox-field full"><input v-model="applyToAllAgents" type="checkbox" /><span>{{ t('settings.connections.form.applyToAllAgentsLabel') }}</span></label>
          <label v-if="!editingId && applyToAllAgents" class="full"><span>{{ t('settings.connections.form.sharedModelLabel') }}</span><input v-model="sharedAgentModel" required maxlength="200" :placeholder="t('settings.connections.form.sharedModelPlaceholder')" /><small>{{ t('settings.connections.form.sharedModelHelp') }}</small></label>
          <p class="field-help full">{{ t('settings.connections.form.baseUrlHelpPrefix') }} <code>http://host.docker.internal:11434</code>{{ t('settings.connections.form.baseUrlHelpSuffix') }}</p>
          <div class="full form-actions"><button class="button" type="button" @click="closeConnectionForm">{{ t('common.cancel') }}</button><button class="button primary" :disabled="savingConnection">{{ savingConnection ? t('common.saving') : t('settings.connections.form.saveButton') }}</button></div>
        </form>

        <TransitionGroup v-if="connections.length" name="card-list" tag="div" class="connection-list">
          <article v-for="item in connections" :key="item.id" class="panel connection-card">
            <div class="connection-icon">{{ item.executionMode === 'local' ? '⌂' : '☁' }}</div>
            <div><div class="connection-title"><h3>{{ item.name }}</h3><span class="status-pill">{{ item.executionMode }}</span></div><p>{{ providerLabel(item.provider) }} · {{ item.baseUrl }}</p><small>{{ item.apiTokenConfigured ? t('settings.connections.tokenConfigured') : item.provider === 'ollama' ? t('settings.connections.tokenNotRequired') : t('settings.connections.tokenNotConfigured') }}</small><small v-if="connectionResults[item.id]" class="test-result">{{ connectionResults[item.id] }}</small></div>
            <div class="connection-actions"><button class="text-button" type="button" :disabled="testingId === item.id" @click="testConnection(item)">{{ testingId === item.id ? t('settings.connections.testingButton') : t('settings.connections.testButton') }}</button><button class="text-button" type="button" @click="editConnection(item)">{{ t('common.edit') }}</button><button class="text-button danger-text" type="button" @click="pendingDelete=item">{{ t('common.delete') }}</button></div>
          </article>
        </TransitionGroup>
        <div v-else class="empty-state compact"><span class="empty-icon">✦</span><h2>{{ t('settings.connections.empty.title') }}</h2><p>{{ t('settings.connections.empty.description') }}</p></div>
      </section>

      <section class="settings-section">
        <div class="section-heading"><div><p class="eyebrow">{{ t('settings.agents.eyebrow') }}</p><h2>{{ t('settings.agents.title') }}</h2><p>{{ t('settings.agents.description') }}</p></div></div>
        <form class="agent-defaults" @submit.prevent="saveAgentDefaults">
          <article v-for="definition in agentDefinitions" :key="definition.kind" class="panel agent-default-card">
            <div class="agent-copy"><div><h3>{{ t(agentDefinitionKey(definition.kind, 'name')) }}</h3></div><p>{{ t(agentDefinitionKey(definition.kind, 'summary')) }}</p></div>
            <div class="agent-fields">
              <label><span>{{ t('settings.connections.form.providerLabel') }}</span><select v-model="agentDefaults[definition.kind].connectionId" @change="changeAgentConnection(definition.kind)"><option value="">{{ t('settings.agents.chooseEachTime') }}</option><option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option></select></label>
              <label><span>{{ t('settings.agents.modelLabel') }}</span><div class="model-input-row"><select v-if="agentModels[agentDefaults[definition.kind].connectionId]?.length" v-model="agentDefaults[definition.kind].model"><option value="" disabled>{{ t('settings.agents.selectModelOption') }}</option><option v-if="agentDefaults[definition.kind].model && !agentModels[agentDefaults[definition.kind].connectionId].includes(agentDefaults[definition.kind].model)" :value="agentDefaults[definition.kind].model">{{ agentDefaults[definition.kind].model }}</option><option v-for="model in agentModels[agentDefaults[definition.kind].connectionId]" :key="model" :value="model">{{ model }}</option></select><input v-else v-model="agentDefaults[definition.kind].model" :disabled="!agentDefaults[definition.kind].connectionId || loadingAgentModels === definition.kind" :placeholder="loadingAgentModels === definition.kind ? t('settings.agents.loadingModelsPlaceholder') : t('settings.agents.modelNamePlaceholder')" maxlength="200" /><button class="button compact-button" type="button" :disabled="!agentDefaults[definition.kind].connectionId || loadingAgentModels === definition.kind" @click="loadAgentModels(definition.kind, true)">{{ loadingAgentModels === definition.kind ? t('common.loading') : t('common.refresh') }}</button></div></label>
            </div>
            <aside class="agent-tip"><strong>{{ t('settings.agents.capabilityTips') }}</strong><div><span v-for="index in definition.requirementCount" :key="index">{{ t(agentRequirementKey(definition.kind, index - 1)) }}</span></div></aside>
          </article>
          <div class="form-actions"><button class="button primary" :disabled="savingAgentDefaults || !connections.length">{{ savingAgentDefaults ? t('common.saving') : t('settings.agents.saveButton') }}</button></div>
        </form>
      </section>

      <section class="settings-section panel">
        <div class="section-heading"><div><p class="eyebrow">{{ t('settings.interfaceEyebrow') }}</p><h2>{{ t('settings.interface') }}</h2><p>{{ t('settings.interfaceHelp') }}</p></div></div>
        <form class="form-grid" @submit.prevent="savePreferences">
          <label><span>{{ t('settings.theme') }}</span><select v-model="preferences.theme"><option value="system">{{ t('settings.system') }}</option><option value="light">{{ t('settings.light') }}</option><option value="dark">{{ t('settings.dark') }}</option></select></label>
          <div class="full form-actions"><button class="button primary" :disabled="savingPreferences">{{ savingPreferences ? t('common.saving') : t('settings.saveInterface') }}</button></div>
        </form>
      </section>

      <section class="settings-section">
        <div class="section-heading"><div><p class="eyebrow">{{ t('settings.status.eyebrow') }}</p><h2>{{ t('settings.status.title') }}</h2><p>{{ t('settings.status.description') }}</p></div></div>
        <div class="status-grid"><div v-for="(label, key) in { profiles: t('settings.status.labels.profiles'), jobs: t('settings.status.labels.jobs'), documents: t('settings.status.labels.documents'), jobImports: t('settings.status.labels.jobImports'), settings: t('settings.status.labels.settings') }" :key="key" class="panel status-card"><span>{{ label }}</span><strong :class="capabilities[key] ? 'available' : 'unavailable'">{{ capabilities[key] ? t('settings.status.available') : t('settings.status.unavailable') }}</strong></div><div class="panel status-card"><span>{{ t('settings.status.llmProvidersLabel') }}</span><strong :class="connections.length ? 'available' : 'unavailable'">{{ connections.length ? t('settings.status.configured', { count: connections.length }) : t('settings.status.notConfigured') }}</strong></div></div>
      </section>

      <section class="settings-section panel language-switcher">
        <div class="section-heading"><div><p class="eyebrow">{{ t('settings.languageEyebrow') }}</p><h2>{{ t('settings.languageTitle') }}</h2><p>{{ t('settings.languageHelp') }}</p></div><span v-if="switchingLanguage" class="language-saving">{{ t('settings.languageSaving') }}</span></div>
        <div class="language-options" role="radiogroup" :aria-label="t('settings.languageTitle')">
          <button v-for="language in interfaceLanguages" :key="language.code" class="language-option" :class="{ active: preferences.interfaceLanguage === language.code }" type="button" role="radio" :aria-checked="preferences.interfaceLanguage === language.code" :disabled="switchingLanguage" @click="switchInterfaceLanguage(language.code)">
            <span>{{ language.shortName }}</span><strong>{{ language.nativeName }}</strong><small v-if="preferences.interfaceLanguage === language.code">✓</small>
          </button>
        </div>
      </section>
    </template>
    <ConfirmDialog :open="Boolean(pendingDelete)" :title="t('settings.connections.deleteConfirmTitle')" :message="t('settings.connections.deleteConfirmMessage', { name: pendingDelete?.name ?? '' })" :busy="deletingId===pendingDelete?.id" @cancel="pendingDelete=null" @confirm="deleteConnection" />
    <UnsavedChangesDialog :open="showUnsavedDialog" :busy="savingBeforeLeave" @cancel="keepEditing" @discard="discardBeforeLeaving" @save="saveBeforeLeaving" />
  </div>
</template>

<style scoped>
.settings-page, .settings-section { display: grid; gap: 1.25rem; }
.settings-section { margin-bottom: 1.25rem; }
.section-heading { display: flex; align-items: flex-end; justify-content: space-between; gap: 1.5rem; }
.section-heading h2, .section-heading p { margin-bottom: 0; }
.section-heading > div > p:last-child { max-width: 720px; margin-top: .5rem; color: var(--muted); line-height: 1.55; }
.connection-form { margin-bottom: .25rem; }
.connection-form h3 { margin: 0; }
.connection-list { display: grid; gap: .8rem; }
.connection-card { display: grid; grid-template-columns: auto 1fr auto; align-items: center; gap: 1rem; padding: 1.2rem 1.4rem; }
.connection-icon { display: grid; width: 42px; height: 42px; place-items: center; border-radius: 12px; background: var(--accent-pale); color: var(--accent-dark); }
.connection-title { display: flex; align-items: center; gap: .65rem; }
.connection-title h3 { margin: 0; }
.connection-card p { margin: .3rem 0; color: var(--muted); overflow-wrap: anywhere; }
.connection-card small { display: block; color: var(--muted); }
.connection-card .test-result { margin-top: .3rem; color: var(--accent); font-weight: 700; }
.connection-actions { display: flex; align-items: center; gap: .75rem; }
.agent-defaults { display: grid; gap: .8rem; }
.agent-default-card { display: grid; grid-template-columns: minmax(190px, .9fr) minmax(320px, 1.4fr) minmax(220px, 1fr); align-items: center; gap: 1.2rem; }
.agent-copy h3 { display: inline; margin: 0; }
.agent-copy p { margin: .45rem 0 0; color: var(--muted); font-size: .78rem; line-height: 1.5; }
.agent-fields { display: grid; grid-template-columns: 1fr 1.35fr; gap: .75rem; }
.model-input-row { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: .45rem; }
.compact-button { padding-inline: .7rem; white-space: nowrap; }
.agent-tip { padding: .75rem; border-radius: 10px; background: var(--surface-soft); }
.agent-tip strong { display: block; margin-bottom: .45rem; font-size: .7rem; }
.agent-tip div { display: flex; flex-wrap: wrap; gap: .35rem; }
.agent-tip span { padding: .22rem .4rem; border: 1px solid var(--line); border-radius: 999px; background: var(--surface); color: var(--muted); font-size: .62rem; }
.danger-text { color: var(--danger); }
.checkbox-field { display: flex !important; grid-template-columns: auto 1fr; align-items: center; }
.checkbox-field input { width: auto; }
.field-help { margin: 0; color: var(--muted); font-size: .82rem; line-height: 1.5; }
.form-actions { gap: .7rem; }
.empty-state.compact { padding: 2.5rem 1.5rem; }
.status-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: .8rem; }
.status-card { display: grid; gap: .5rem; padding: 1.2rem; }
.status-card span { color: var(--muted); font-size: .8rem; }
.status-card strong { font-size: .95rem; }
.available { color: var(--accent); }
.unavailable { color: var(--muted); }
.language-switcher { margin-bottom: 0; }
.language-saving { color: var(--muted); font-size: .78rem; white-space: nowrap; }
.language-options { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: .7rem; }
.language-option { display: grid; grid-template-columns: auto 1fr auto; align-items: center; gap: .65rem; padding: .85rem; border: 1px solid var(--line); border-radius: 11px; background: var(--surface-soft); color: var(--ink); text-align: left; cursor: pointer; transition: border-color .18s ease, background .18s ease, transform .18s ease; }
.language-option:hover:not(:disabled) { border-color: var(--accent); transform: translateY(-1px); }
.language-option.active { border-color: var(--accent); background: var(--accent-pale); }
.language-option > span { display: grid; width: 31px; height: 31px; place-items: center; border-radius: 8px; background: var(--surface); color: var(--accent-dark); font-size: .68rem; font-weight: 800; }
.language-option strong { font-size: .82rem; }
.language-option small { color: var(--accent); font-weight: 900; }
@media (max-width: 1000px) { .agent-default-card { grid-template-columns: 1fr 1.5fr; } .agent-tip { grid-column: 1 / -1; } }
@media (max-width: 760px) { .section-heading, .connection-card, .agent-default-card { align-items: stretch; grid-template-columns: 1fr; flex-direction: column; } .agent-tip { grid-column: auto; } .agent-fields { grid-template-columns: 1fr; } .connection-actions { flex-wrap: wrap; } .status-grid, .language-options { grid-template-columns: 1fr 1fr; } }
@media (max-width: 480px) { .status-grid, .language-options { grid-template-columns: 1fr; } }
</style>
