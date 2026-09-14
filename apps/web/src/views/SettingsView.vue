<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { applyTheme } from '../lib/preferences'
import type { LLMConnection, LLMConnectionInput, SettingsPreferences } from '../lib/types'

const { locale } = useI18n()
const loading = ref(true)
const savingPreferences = ref(false)
const savingConnection = ref(false)
const testingId = ref('')
const error = ref('')
const notice = ref('')
const connections = ref<LLMConnection[]>([])
const capabilities = ref<Record<string, boolean>>({})
const editingId = ref('')
const showConnectionForm = ref(false)
const discoveredModels = ref<string[]>([])
const connectionResults = reactive<Record<string, string>>({})
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
  notice.value = ''
}

function editConnection(item: LLMConnection) {
  editingId.value = item.id
  discoveredModels.value = []
  Object.assign(connectionForm, { name: item.name, executionMode: item.executionMode, provider: item.provider, baseUrl: item.baseUrl, apiToken: '', clearApiToken: false })
  showConnectionForm.value = true
  error.value = ''
  notice.value = ''
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
    const [storedPreferences, storedConnections, system] = await Promise.all([
      api.getSettings(), api.listLLMConnections(), api.capabilities(),
    ])
    Object.assign(preferences, { interfaceLanguage: storedPreferences.interfaceLanguage, theme: storedPreferences.theme })
    connections.value = storedConnections.items
    capabilities.value = system.features
    locale.value = storedPreferences.interfaceLanguage
    applyTheme(storedPreferences.theme)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not load settings.'
  } finally {
    loading.value = false
  }
}

async function savePreferences() {
  savingPreferences.value = true
  error.value = ''
  notice.value = ''
  try {
    const saved = await api.updateSettings({ ...preferences })
    locale.value = saved.interfaceLanguage
    applyTheme(saved.theme)
    notice.value = 'Interface settings saved.'
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not save interface settings.'
  } finally {
    savingPreferences.value = false
  }
}

async function saveConnection() {
  savingConnection.value = true
  error.value = ''
  notice.value = ''
  try {
    const saved = editingId.value
      ? await api.updateLLMConnection(editingId.value, { ...connectionForm })
      : await api.createLLMConnection({ ...connectionForm })
    const index = connections.value.findIndex(item => item.id === saved.id)
    if (index >= 0) connections.value[index] = saved
    else connections.value.push(saved)
    connections.value.sort((left, right) => left.name.localeCompare(right.name))
    notice.value = 'LLM connection saved. Test it before using it for generation.'
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
  notice.value = ''
  try {
    const result = await api.testLLMConnection(item.id)
    connectionResults[item.id] = `Connected · ${result.models.length} model${result.models.length === 1 ? '' : 's'} available`
    discoveredModels.value = result.models
    notice.value = result.models.length ? `Connection succeeded. Models: ${result.models.slice(0, 8).join(', ')}${result.models.length > 8 ? '…' : ''}` : 'Connection succeeded, but the endpoint reported no models.'
  } catch (cause) {
    connectionResults[item.id] = 'Connection failed'
    error.value = cause instanceof Error ? cause.message : 'Could not test the LLM connection.'
  } finally {
    testingId.value = ''
  }
}

async function deleteConnection(item: LLMConnection) {
  if (!window.confirm(`Delete the “${item.name}” connection?`)) return
  error.value = ''
  try {
    await api.deleteLLMConnection(item.id)
    connections.value = connections.value.filter(candidate => candidate.id !== item.id)
    notice.value = 'LLM connection deleted.'
    if (editingId.value === item.id) closeConnectionForm()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not delete the LLM connection.'
  }
}

onMounted(load)
</script>

<template>
  <div class="page settings-page">
    <PageHeader title="Settings" description="Manage LLM connections and interface preferences. Generation choices remain specific to each opportunity." />

    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <p v-if="notice" class="notice" role="status">{{ notice }}</p>
    <div v-if="loading" class="empty-state">Loading settings…</div>

    <template v-else>
      <section class="settings-section">
        <div class="section-heading">
          <div><p class="eyebrow">AI providers</p><h2>LLM connections</h2><p>Store reusable cloud or local endpoints. Profile, opportunity, model, and document options are selected during generation.</p></div>
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

        <div v-if="connections.length" class="connection-list">
          <article v-for="item in connections" :key="item.id" class="panel connection-card">
            <div class="connection-icon">{{ item.executionMode === 'local' ? '⌂' : '☁' }}</div>
            <div><div class="connection-title"><h3>{{ item.name }}</h3><span class="status-pill">{{ item.executionMode }}</span></div><p>{{ providerLabel(item.provider) }} · {{ item.baseUrl }}</p><small>{{ item.apiTokenConfigured ? 'API token configured' : item.provider === 'ollama' ? 'No API token required' : 'API token not configured' }}</small><small v-if="connectionResults[item.id]" class="test-result">{{ connectionResults[item.id] }}</small></div>
            <div class="connection-actions"><button class="text-button" type="button" :disabled="testingId === item.id" @click="testConnection(item)">{{ testingId === item.id ? 'Testing…' : 'Test connection' }}</button><button class="text-button" type="button" @click="editConnection(item)">Edit</button><button class="text-button danger-text" type="button" @click="deleteConnection(item)">Delete</button></div>
          </article>
        </div>
        <div v-else class="empty-state compact"><span class="empty-icon">✦</span><h2>No LLM connections</h2><p>Add a cloud provider or local Ollama endpoint. A connection is selected explicitly for each generation.</p></div>
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
        <div class="status-grid"><div v-for="(label, key) in { profiles: 'Profiles', jobs: 'Opportunities', documents: 'Document extraction', jobImports: 'Job import', settings: 'Settings API' }" :key="key" class="panel status-card"><span>{{ label }}</span><strong :class="capabilities[key] ? 'available' : 'unavailable'">{{ capabilities[key] ? 'Available' : 'Unavailable' }}</strong></div><div class="panel status-card"><span>LLM connections</span><strong :class="connections.length ? 'available' : 'unavailable'">{{ connections.length ? `${connections.length} configured` : 'Not configured' }}</strong></div></div>
      </section>
    </template>
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
@media (max-width: 760px) { .section-heading, .connection-card { align-items: stretch; grid-template-columns: 1fr; flex-direction: column; } .connection-actions { flex-wrap: wrap; } .status-grid { grid-template-columns: 1fr 1fr; } }
@media (max-width: 480px) { .status-grid { grid-template-columns: 1fr; } }
</style>
