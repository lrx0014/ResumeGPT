<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import ConfirmDialog from '../components/ConfirmDialog.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { applyTheme } from '../lib/preferences'
import { toast } from '../lib/toast'
import type { AgentDefault, AgentKind, GenerationModelChoice, LLMConnection, LLMConnectionInput, SettingsPreferences } from '../lib/types'

const { locale } = useI18n()
const loading = ref(true)
const savingPreferences = ref(false)
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
const discoveredModels = ref<string[]>([])
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
const agentDefinitions: { kind: AgentKind; name: string; summary: string; requirements: string[] }[] = [
  { kind: 'writer', name: 'Writer Agent', summary: 'Creates tailored CV and cover-letter content from a Profile and Job Opportunity.', requirements: ['Text generation', 'Reliable tool calling', 'Strong writing and instruction following', 'Long context recommended'] },
  { kind: 'template_applier', name: 'Template Applying Agent', summary: 'Studies template source, writes LaTeX, compiles it, and repairs rendering errors.', requirements: ['Reliable tool calling', 'Strong coding and LaTeX ability', 'Long context', 'Multi-step reasoning'] },
  { kind: 'document_designer', name: 'Document Designer Agent', summary: 'Designs a polished CV or cover letter with print-ready HTML and CSS when no template is selected.', requirements: ['Reliable tool calling', 'Strong HTML and CSS ability', 'Visual design and typography', 'Multi-step reasoning'] },
  { kind: 'visual_reviewer', name: 'Visual Reviewer Agent', summary: 'Inspects rendered PDF pages for layout, typography, clipping, and visual quality.', requirements: ['Image input / vision', 'Structured JSON output', 'Layout reasoning'] },
  { kind: 'job_import', name: 'Job Import Agent', summary: 'Browses a supplied job page, expands dynamic content, and extracts structured job details.', requirements: ['Reliable tool calling', 'HTML and webpage understanding', 'Structured data extraction'] },
  { kind: 'job_hunter', name: 'Job Hunter Agent', summary: 'Searches the public web and ranks fresh openings against the configured criteria and optional Profile.', requirements: ['Reliable tool calling', 'Web search reasoning', 'Long context recommended'] },
]
const preferences = reactive<Pick<SettingsPreferences, 'interfaceLanguage' | 'theme'>>({ interfaceLanguage: 'en', theme: 'system' })
const connectionForm = reactive<LLMConnectionInput>({
  name: '', executionMode: 'local', provider: 'ollama', baseUrl: 'http://host.docker.internal:11434',
  apiToken: '', clearApiToken: false,
})

const providerOptions = computed(() => connectionForm.executionMode === 'cloud'
  ? [{ value: 'openai', label: 'OpenAI' }, { value: 'openai_compatible', label: 'OpenAI-compatible' }]
  : [{ value: 'ollama', label: 'Ollama' }, { value: 'openai_compatible', label: 'OpenAI-compatible' }])

function providerLabel(provider: LLMConnection['provider']) {
  return ({ openai: 'OpenAI', openai_compatible: 'OpenAI-compatible', ollama: 'Ollama' })[provider]
}

function resetConnectionForm() {
  editingId.value = ''
  discoveredModels.value = []
  Object.assign(connectionForm, { name: '', executionMode: 'local', provider: 'ollama', baseUrl: 'http://host.docker.internal:11434', apiToken: '', clearApiToken: false })
}

function addConnection() {
  resetConnectionForm()
  showConnectionForm.value = true
  error.value = ''
}

function editConnection(item: LLMConnection) {
  editingId.value = item.id
  discoveredModels.value = []
  Object.assign(connectionForm, { name: item.name, executionMode: item.executionMode, provider: item.provider, baseUrl: item.baseUrl, apiToken: '', clearApiToken: false })
  showConnectionForm.value = true
  error.value = ''
}

function closeConnectionForm() {
  showConnectionForm.value = false
  resetConnectionForm()
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
    locale.value = storedPreferences.interfaceLanguage
    applyTheme(storedPreferences.theme)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not load settings.'
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
    error.value = cause instanceof Error ? cause.message : 'Could not load models for this Agent.'
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
    error.value = `Choose both a connection and model for ${partial.name}, or clear both fields.`
    return
  }
  savingAgentDefaults.value = true
  error.value = ''
  try {
    const items: AgentDefault[] = agentDefinitions.flatMap(({ kind }) => {
      const choice = agentDefaults[kind]
      return choice.connectionId && choice.model ? [{ agent: kind, ...choice }] : []
    })
    await api.updateAgentDefaults(items)
    toast.success('Agent defaults saved.')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not save Agent defaults.'
  } finally {
    savingAgentDefaults.value = false
  }
}

async function savePreferences() {
  savingPreferences.value = true
  error.value = ''
  try {
    const saved = await api.updateSettings({ ...preferences })
    locale.value = saved.interfaceLanguage
    applyTheme(saved.theme)
    toast.success('Interface settings saved.')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not save interface settings.'
  } finally {
    savingPreferences.value = false
  }
}

async function saveConnection() {
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
    toast.success('LLM connection saved. Test it before using it for generation.')
    closeConnectionForm()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not save the LLM connection.'
  } finally {
    savingConnection.value = false
  }
}

async function testConnection(item: LLMConnection) {
  testingId.value = item.id
  error.value = ''
  try {
    const result = await api.testLLMConnection(item.id)
    connectionResults[item.id] = `Connected · ${result.models.length} model${result.models.length === 1 ? '' : 's'} available`
    discoveredModels.value = result.models
    agentModels[item.id] = result.models
    toast.success(result.models.length ? `Connection succeeded. Models: ${result.models.slice(0, 8).join(', ')}${result.models.length > 8 ? '…' : ''}` : 'Connection succeeded, but the endpoint reported no models.')
  } catch (cause) {
    connectionResults[item.id] = 'Connection failed'
    error.value = cause instanceof Error ? cause.message : 'Could not test the LLM connection.'
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
    pendingDelete.value = null
    if (editingId.value === item.id) closeConnectionForm()
    toast.success('LLM connection deleted.')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not delete the LLM connection.'
  } finally {
    deletingId.value = ''
  }
}

onMounted(load)
</script>

<template>
  <div class="page settings-page">
    <PageHeader title="Settings" description="Manage LLM connections, set sensible defaults for each AI Agent, and personalize the interface." />

    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <div v-if="loading" class="empty-state">Loading settings…</div>

    <template v-else>
      <section class="settings-section">
        <div class="section-heading">
          <div><p class="eyebrow">AI providers</p><h2>LLM connections</h2><p>Store reusable cloud or local endpoints, then assign them to Agent roles below.</p></div>
          <button class="button primary" type="button" @click="addConnection">Add connection</button>
        </div>

        <form v-if="showConnectionForm" class="panel form-grid connection-form" @submit.prevent="saveConnection">
          <h3 class="full">{{ editingId ? 'Edit LLM connection' : 'New LLM connection' }}</h3>
          <label><span>Connection name</span><input v-model="connectionForm.name" required maxlength="120" placeholder="Local Ollama" /></label>
          <label><span>Execution mode</span><select v-model="connectionForm.executionMode"><option value="local">Local</option><option value="cloud">Cloud</option></select></label>
          <label><span>Provider</span><select v-model="connectionForm.provider"><option v-for="option in providerOptions" :key="option.value" :value="option.value">{{ option.label }}</option></select></label>
          <label><span>Base URL</span><input v-model="connectionForm.baseUrl" required type="url" maxlength="2048" /></label>
          <label class="full"><span>API token</span><input v-model="connectionForm.apiToken" type="password" maxlength="8192" autocomplete="new-password" :placeholder="editingId ? 'Leave blank to keep the saved token' : connectionForm.provider === 'ollama' ? 'Optional for Ollama' : 'Enter API token'" /></label>
          <label v-if="editingId" class="checkbox-field full"><input v-model="connectionForm.clearApiToken" type="checkbox" /><span>Remove the saved API token</span></label>
          <p class="field-help full">The Base URL is accessed by the ResumeGPT API container. Local Ollama on the host normally uses <code>http://host.docker.internal:11434</code>.</p>
          <div class="full form-actions"><button class="button" type="button" @click="closeConnectionForm">Cancel</button><button class="button primary" :disabled="savingConnection">{{ savingConnection ? 'Saving…' : 'Save connection' }}</button></div>
        </form>

        <TransitionGroup v-if="connections.length" name="card-list" tag="div" class="connection-list">
          <article v-for="item in connections" :key="item.id" class="panel connection-card">
            <div class="connection-icon">{{ item.executionMode === 'local' ? '⌂' : '☁' }}</div>
            <div><div class="connection-title"><h3>{{ item.name }}</h3><span class="status-pill">{{ item.executionMode }}</span></div><p>{{ providerLabel(item.provider) }} · {{ item.baseUrl }}</p><small>{{ item.apiTokenConfigured ? 'API token configured' : item.provider === 'ollama' ? 'No API token required' : 'API token not configured' }}</small><small v-if="connectionResults[item.id]" class="test-result">{{ connectionResults[item.id] }}</small></div>
            <div class="connection-actions"><button class="text-button" type="button" :disabled="testingId === item.id" @click="testConnection(item)">{{ testingId === item.id ? 'Testing…' : 'Test connection' }}</button><button class="text-button" type="button" @click="editConnection(item)">Edit</button><button class="text-button danger-text" type="button" @click="pendingDelete=item">Delete</button></div>
          </article>
        </TransitionGroup>
        <div v-else class="empty-state compact"><span class="empty-icon">✦</span><h2>No LLM connections</h2><p>Add a cloud provider or local Ollama endpoint before configuring Agent defaults.</p></div>
      </section>

      <section class="settings-section">
        <div class="section-heading"><div><p class="eyebrow">Agent routing</p><h2>Default models</h2><p>Choose the connection and model each Agent should use automatically. These defaults prefill new workflows and can still be overridden for an individual task.</p></div></div>
        <form class="agent-defaults" @submit.prevent="saveAgentDefaults">
          <article v-for="definition in agentDefinitions" :key="definition.kind" class="panel agent-default-card">
            <div class="agent-copy"><div><h3>{{ definition.name }}</h3><span class="optional-label">Optional default</span></div><p>{{ definition.summary }}</p></div>
            <div class="agent-fields">
              <label><span>Connection</span><select v-model="agentDefaults[definition.kind].connectionId" @change="changeAgentConnection(definition.kind)"><option value="">Choose each time</option><option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option></select></label>
              <label><span>Model</span><div class="model-input-row"><select v-if="agentModels[agentDefaults[definition.kind].connectionId]?.length" v-model="agentDefaults[definition.kind].model"><option value="" disabled>Select model</option><option v-if="agentDefaults[definition.kind].model && !agentModels[agentDefaults[definition.kind].connectionId].includes(agentDefaults[definition.kind].model)" :value="agentDefaults[definition.kind].model">{{ agentDefaults[definition.kind].model }}</option><option v-for="model in agentModels[agentDefaults[definition.kind].connectionId]" :key="model" :value="model">{{ model }}</option></select><input v-else v-model="agentDefaults[definition.kind].model" :disabled="!agentDefaults[definition.kind].connectionId || loadingAgentModels === definition.kind" :placeholder="loadingAgentModels === definition.kind ? 'Loading models…' : 'Enter model name'" maxlength="200" /><button class="button compact-button" type="button" :disabled="!agentDefaults[definition.kind].connectionId || loadingAgentModels === definition.kind" @click="loadAgentModels(definition.kind, true)">{{ loadingAgentModels === definition.kind ? 'Loading…' : 'Refresh' }}</button></div></label>
            </div>
            <aside class="agent-tip"><strong>Model capability tips</strong><div><span v-for="requirement in definition.requirements" :key="requirement">{{ requirement }}</span></div></aside>
          </article>
          <div class="form-actions"><button class="button primary" :disabled="savingAgentDefaults || !connections.length">{{ savingAgentDefaults ? 'Saving…' : 'Save Agent defaults' }}</button></div>
        </form>
      </section>

      <section class="settings-section panel">
        <div class="section-heading"><div><p class="eyebrow">Personalization</p><h2>Interface</h2><p>These choices affect only the ResumeGPT interface, not generated documents.</p></div></div>
        <form class="form-grid" @submit.prevent="savePreferences">
          <label><span>Interface language</span><select v-model="preferences.interfaceLanguage"><option value="en">English</option><option value="de">Deutsch</option></select></label>
          <label><span>Theme</span><select v-model="preferences.theme"><option value="system">System</option><option value="light">Light</option><option value="dark">Dark</option></select></label>
          <div class="full form-actions"><button class="button primary" :disabled="savingPreferences">{{ savingPreferences ? 'Saving…' : 'Save interface settings' }}</button></div>
        </form>
      </section>

      <section class="settings-section">
        <div class="section-heading"><div><p class="eyebrow">Read only</p><h2>System status</h2><p>Availability reported by the current API deployment.</p></div></div>
        <div class="status-grid"><div v-for="(label, key) in { profiles: 'Profiles', jobs: 'Job opportunities', documents: 'Document extraction', jobImports: 'Job import', settings: 'Settings API' }" :key="key" class="panel status-card"><span>{{ label }}</span><strong :class="capabilities[key] ? 'available' : 'unavailable'">{{ capabilities[key] ? 'Available' : 'Unavailable' }}</strong></div><div class="panel status-card"><span>LLM connections</span><strong :class="connections.length ? 'available' : 'unavailable'">{{ connections.length ? `${connections.length} configured` : 'Not configured' }}</strong></div></div>
      </section>
    </template>
    <ConfirmDialog :open="Boolean(pendingDelete)" title="Delete LLM connection?" :message="`“${pendingDelete?.name ?? ''}” will no longer be available for generation. This action cannot be undone.`" :busy="deletingId===pendingDelete?.id" @cancel="pendingDelete=null" @confirm="deleteConnection" />
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
.optional-label { margin-left: .55rem; color: var(--muted); font-size: .65rem; font-weight: 700; text-transform: uppercase; }
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
@media (max-width: 1000px) { .agent-default-card { grid-template-columns: 1fr 1.5fr; } .agent-tip { grid-column: 1 / -1; } }
@media (max-width: 760px) { .section-heading, .connection-card, .agent-default-card { align-items: stretch; grid-template-columns: 1fr; flex-direction: column; } .agent-tip { grid-column: auto; } .agent-fields { grid-template-columns: 1fr; } .connection-actions { flex-wrap: wrap; } .status-grid { grid-template-columns: 1fr 1fr; } }
@media (max-width: 480px) { .status-grid { grid-template-columns: 1fr; } }
</style>
