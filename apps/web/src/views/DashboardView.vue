<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import type { GenerationRun, Job, LLMConnection, Profile, Template } from '../lib/types'

const profiles = ref<Profile[]>([])
const opportunities = ref<Job[]>([])
const templates = ref<Template[]>([])
const connections = ref<LLMConnection[]>([])
const generations = ref<GenerationRun[]>([])
const capabilities = ref<Record<string, boolean>>({})
const loading = ref(true)
const error = ref('')

const preparedProfiles = computed(() => profiles.value.filter(item => item.content.trim()).length)
const readyOpportunities = computed(() => opportunities.value.filter(item => ['manual', 'ready'].includes(item.importState)).length)
const pendingOpportunities = computed(() => opportunities.value.filter(item => ['queued', 'fetching'].includes(item.importState)).length)
const attentionOpportunities = computed(() => opportunities.value.filter(item => ['needs_user_action', 'failed'].includes(item.importState)).length)
const readyTemplates = computed(() => templates.value.filter(item => item.state === 'ready').length)
const customTemplates = computed(() => templates.value.filter(item => !item.builtIn).length)
const recentOpportunities = computed(() => [...opportunities.value].sort((left, right) => right.updatedAt.localeCompare(left.updatedAt)).slice(0, 3))
const prerequisitesReady = computed(() => preparedProfiles.value > 0 && readyOpportunities.value > 0 && readyTemplates.value > 0 && connections.value.length > 0)
const readyGenerations = computed(() => generations.value.filter(item => item.state === 'ready').length)

const setupSteps = computed(() => [
  { number: '01', title: 'Prepare your profile', description: preparedProfiles.value ? `${preparedProfiles.value} profile${preparedProfiles.value === 1 ? '' : 's'} with saved content.` : 'Add or import the experience, education, skills, and achievements you want the model to use.', to: '/profiles', complete: preparedProfiles.value > 0 },
  { number: '02', title: 'Track a job opportunity', description: readyOpportunities.value ? `${readyOpportunities.value} job ${readyOpportunities.value === 1 ? 'opportunity' : 'opportunities'} ready to target.` : 'Import a LinkedIn or Indeed URL, or enter the role manually.', to: '/jobs', complete: readyOpportunities.value > 0 },
  { number: '03', title: 'Choose a document template', description: readyTemplates.value ? `${readyTemplates.value} resume or cover-letter template${readyTemplates.value === 1 ? '' : 's'} ready.` : 'Use the built-in LaTeX template or upload a TeX or Word file.', to: '/templates', complete: readyTemplates.value > 0 },
  { number: '04', title: 'Connect an LLM', description: connections.value.length ? `${connections.value.length} cloud or local connection${connections.value.length === 1 ? '' : 's'} configured.` : 'Add an OpenAI, compatible, or local Ollama connection in Settings.', to: '/settings', complete: connections.value.length > 0 },
  { number: '05', title: 'Generate for a job opportunity', description: readyGenerations.value ? `${readyGenerations.value} visually reviewed PDF${readyGenerations.value === 1 ? '' : 's'} ready.` : capabilities.value.generation ? 'Select the prepared inputs and create a tailored resume or cover letter.' : 'Generation is the next milestone. Your prepared inputs will be used here.', to: '/generate', complete: readyGenerations.value > 0, planned: !capabilities.value.generation },
])

const firstIncomplete = computed(() => setupSteps.value.find(step => !step.complete && !step.planned))
const primaryAction = computed(() => firstIncomplete.value
  ? { label: `Continue: ${firstIncomplete.value.title}`, to: firstIncomplete.value.to }
  : { label: capabilities.value.generation ? 'Create application' : 'Review generation setup', to: '/generate' })

function metricDetail(kind: 'profiles' | 'opportunities' | 'templates' | 'connections') {
  if (loading.value) return 'Loading workspace data…'
  if (kind === 'profiles') return preparedProfiles.value ? `${preparedProfiles.value} with saved content` : 'Create or complete a profile'
  if (kind === 'opportunities') {
    if (attentionOpportunities.value) return `${attentionOpportunities.value} need${attentionOpportunities.value === 1 ? 's' : ''} attention`
    if (pendingOpportunities.value) return `${pendingOpportunities.value} import${pendingOpportunities.value === 1 ? '' : 's'} in progress`
    return opportunities.value.length ? `${readyOpportunities.value} ready for generation` : 'Add a role manually or by URL'
  }
  if (kind === 'templates') return customTemplates.value ? `${customTemplates.value} custom · built-in included` : 'Built-in Rezume included'
  return connections.value.length ? 'Cloud and local providers supported' : 'Required before generation'
}

async function load() {
  loading.value = true
  const results = await Promise.allSettled([
    api.listProfiles(), api.listJobs(), api.listTemplates(), api.listLLMConnections(), api.capabilities(), api.listGenerations(),
  ])
  if (results[0].status === 'fulfilled') profiles.value = results[0].value.items
  if (results[1].status === 'fulfilled') opportunities.value = results[1].value.items
  if (results[2].status === 'fulfilled') templates.value = results[2].value.items
  if (results[3].status === 'fulfilled') connections.value = results[3].value.items
  if (results[4].status === 'fulfilled') capabilities.value = results[4].value.features
  if (results[5].status === 'fulfilled') generations.value = results[5].value.items
  if (results.some(result => result.status === 'rejected')) error.value = 'Some workspace data could not be loaded. Available sections are still shown.'
  loading.value = false
}

onMounted(load)
</script>

<template>
  <div class="page dashboard-page">
    <PageHeader
      eyebrow="Workspace overview"
      title="Make your CV smarter."
      description="Tailor your CV and cover letter to each job opportunity, and track your job search in one place."
    >
      <RouterLink class="button primary" :to="primaryAction.to">{{ primaryAction.label }}</RouterLink>
    </PageHeader>

    <p v-if="error" class="notice error" role="alert">{{ error }}</p>

    <section class="metric-grid overview-metrics" aria-label="Workspace metrics">
      <RouterLink class="metric-card" to="/profiles"><span>Profiles</span><strong>{{ loading ? '—' : profiles.length }}</strong><small>{{ metricDetail('profiles') }}</small></RouterLink>
      <RouterLink class="metric-card" to="/jobs"><span>Job opportunities</span><strong>{{ loading ? '—' : opportunities.length }}</strong><small>{{ metricDetail('opportunities') }}</small></RouterLink>
      <RouterLink class="metric-card" to="/templates"><span>Ready templates</span><strong>{{ loading ? '—' : readyTemplates }}</strong><small>{{ metricDetail('templates') }}</small></RouterLink>
      <RouterLink class="metric-card" to="/settings"><span>LLM connections</span><strong>{{ loading ? '—' : connections.length }}</strong><small>{{ metricDetail('connections') }}</small></RouterLink>
      <RouterLink class="metric-card" to="/generate"><span>Generated PDFs</span><strong>{{ loading ? '—' : readyGenerations }}</strong><small>{{ generations.some(item => item.state === 'running' || item.state === 'queued') ? 'Generation in progress' : 'Visually reviewed outputs' }}</small></RouterLink>
    </section>

    <div class="dashboard-grid">
      <section class="panel setup-panel">
        <div class="panel-heading">
          <div><p class="eyebrow">Getting started</p><h2>Your tailored-document workflow</h2></div>
          <span class="status-pill">{{ prerequisitesReady ? 'Inputs ready' : 'Setup in progress' }}</span>
        </div>
        <div class="steps">
          <RouterLink v-for="step in setupSteps" :key="step.number" class="step" :class="{ complete: step.complete, planned: step.planned }" :to="step.to">
            <span class="step-number">{{ step.complete ? '✓' : step.number }}</span>
            <div><strong>{{ step.title }}</strong><p>{{ step.description }}</p></div>
            <span v-if="step.planned" class="step-state">Coming soon</span><span v-else class="arrow">→</span>
          </RouterLink>
        </div>
      </section>

      <section class="panel recent-panel">
        <div class="panel-heading"><div><p class="eyebrow">Live workspace</p><h2>Recent job opportunities</h2></div><RouterLink class="text-button" to="/jobs">View all</RouterLink></div>
        <div v-if="recentOpportunities.length" class="recent-list">
          <RouterLink v-for="item in recentOpportunities" :key="item.id" class="recent-item" :to="`/jobs/${item.id}`">
            <span class="company-mark">{{ (item.company || '?').slice(0, 2).toUpperCase() }}</span>
            <div><strong>{{ item.title || 'Job details pending' }}</strong><p>{{ item.company || 'Company pending' }}</p></div>
            <span class="status-pill">{{ item.status }}</span>
          </RouterLink>
        </div>
        <div v-else class="empty-state compact"><span class="empty-icon">◇</span><h2>No job opportunities yet</h2><p>Import a public job URL or add the role manually.</p><RouterLink class="button" to="/jobs">Add job opportunity</RouterLink></div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.dashboard-page { display: grid; gap: 1.25rem; }
.dashboard-page :deep(.page-header) { margin-bottom: .5rem; }
.overview-metrics { grid-template-columns: repeat(5, minmax(0, 1fr)); margin-bottom: 0; }
.metric-card { transition: border-color .18s ease, transform .18s ease, box-shadow .18s ease; }
.metric-card:hover { border-color: #b8c9bf; transform: translateY(-2px); box-shadow: 0 22px 55px rgba(29, 48, 40, .11); }
.dashboard-grid { display: grid; grid-template-columns: minmax(0, 1.45fr) minmax(300px, .75fr); gap: 1.25rem; align-items: start; }
.setup-panel, .recent-panel { min-width: 0; }
.setup-panel .panel-heading, .recent-panel .panel-heading { gap: 1rem; }
.step.complete .step-number { display: grid; width: 24px; height: 24px; place-items: center; border-radius: 50%; background: var(--accent-pale); color: var(--accent-dark); font-weight: 800; }
.step.planned { cursor: default; }
.step-state { color: var(--muted); font-size: .72rem; font-weight: 700; white-space: nowrap; }
.recent-list { display: grid; border-top: 1px solid var(--line); }
.recent-item { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: .8rem; padding: 1rem .15rem; border-bottom: 1px solid var(--line); transition: transform .18s ease; }
.recent-item:hover { transform: translateX(3px); }
.recent-item strong, .recent-item p { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.recent-item strong { display: block; font-size: .86rem; }
.recent-item p { margin: .25rem 0 0; color: var(--muted); font-size: .75rem; }
.recent-item .status-pill { font-size: .62rem; }
.empty-state.compact { padding: 2rem 1rem; }
.empty-state.compact .button { margin-top: 1.2rem; }
@media (max-width: 1050px) { .overview-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); } .dashboard-grid { grid-template-columns: 1fr; } }
@media (max-width: 620px) { .overview-metrics { grid-template-columns: 1fr; } .recent-item { grid-template-columns: auto minmax(0, 1fr); } .recent-item .status-pill { grid-column: 2; } }
</style>
