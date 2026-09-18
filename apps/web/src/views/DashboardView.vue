<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { jobStatusLabel } from '../lib/jobStatus'
import type { GenerationRun, Job, LLMConnection, Profile, Template } from '../lib/types'

const profiles = ref<Profile[]>([])
const opportunities = ref<Job[]>([])
const templates = ref<Template[]>([])
const connections = ref<LLMConnection[]>([])
const generations = ref<GenerationRun[]>([])
const capabilities = ref<Record<string, boolean>>({})
const loading = ref(true)
const error = ref('')

const preparedProfiles = computed(() => profiles.value.filter(item => item.hasContent).length)
const readyOpportunities = computed(() => opportunities.value.filter(item => ['manual', 'ready'].includes(item.importState)).length)
const pendingOpportunities = computed(() => opportunities.value.filter(item => ['queued', 'fetching'].includes(item.importState)).length)
const attentionOpportunities = computed(() => opportunities.value.filter(item => ['needs_user_action', 'failed'].includes(item.importState)).length)
const readyTemplates = computed(() => templates.value.filter(item => item.state === 'ready').length)
const customTemplates = computed(() => templates.value.filter(item => !item.builtIn).length)
const recentOpportunities = computed(() => [...opportunities.value].sort((left, right) => right.updatedAt.localeCompare(left.updatedAt)).slice(0, 3))
const prerequisitesReady = computed(() => preparedProfiles.value > 0 && readyOpportunities.value > 0 && readyTemplates.value > 0 && connections.value.length > 0)
const readyGenerations = computed(() => generations.value.filter(item => item.state === 'ready').length)

const setupSteps = computed(() => [
  { number: '01', title: t('overview.setup.steps.profile.title'), description: preparedProfiles.value ? t('overview.setup.steps.profile.detail', { count: preparedProfiles.value }, preparedProfiles.value) : t('overview.setup.steps.profile.empty'), to: '/profiles', complete: preparedProfiles.value > 0 },
  { number: '02', title: t('overview.setup.steps.opportunity.title'), description: readyOpportunities.value ? t('overview.setup.steps.opportunity.detail', { count: readyOpportunities.value }, readyOpportunities.value) : t('overview.setup.steps.opportunity.empty'), to: '/jobs', complete: readyOpportunities.value > 0 },
  { number: '03', title: t('overview.setup.steps.template.title'), description: readyTemplates.value ? t('overview.setup.steps.template.detail', { count: readyTemplates.value }, readyTemplates.value) : t('overview.setup.steps.template.empty'), to: '/templates', complete: readyTemplates.value > 0 },
  { number: '04', title: t('overview.setup.steps.provider.title'), description: connections.value.length ? t('overview.setup.steps.provider.detail', { count: connections.value.length }, connections.value.length) : t('overview.setup.steps.provider.empty'), to: '/settings', complete: connections.value.length > 0 },
  { number: '05', title: t('overview.setup.steps.generate.title'), description: readyGenerations.value ? t('overview.setup.steps.generate.detail', { count: readyGenerations.value }, readyGenerations.value) : capabilities.value.generation ? t('overview.setup.steps.generate.readyCapability') : t('overview.setup.steps.generate.pendingCapability'), to: '/generate', complete: readyGenerations.value > 0, planned: !capabilities.value.generation },
])

const firstIncomplete = computed(() => setupSteps.value.find(step => !step.complete && !step.planned))
const primaryAction = computed(() => firstIncomplete.value
  ? { label: t('overview.actions.continue', { title: firstIncomplete.value.title }), to: firstIncomplete.value.to }
  : { label: capabilities.value.generation ? t('overview.actions.createDocument') : t('overview.actions.reviewSetup'), to: '/generate' })

function metricDetail(kind: 'profiles' | 'opportunities' | 'templates' | 'connections') {
  if (loading.value) return t('overview.metrics.loading')
  if (kind === 'profiles') return preparedProfiles.value ? t('overview.metrics.profilesDetail', { count: preparedProfiles.value }, preparedProfiles.value) : t('overview.metrics.profilesEmpty')
  if (kind === 'opportunities') {
    if (attentionOpportunities.value) return t('overview.metrics.opportunitiesAttention', { count: attentionOpportunities.value }, attentionOpportunities.value)
    if (pendingOpportunities.value) return t('overview.metrics.opportunitiesPending', { count: pendingOpportunities.value }, pendingOpportunities.value)
    return opportunities.value.length ? t('overview.metrics.opportunitiesReady', { count: readyOpportunities.value }, readyOpportunities.value) : t('overview.metrics.opportunitiesEmpty')
  }
  if (kind === 'templates') return customTemplates.value ? t('overview.metrics.templatesCustom', { count: customTemplates.value }, customTemplates.value) : t('overview.metrics.templatesEmpty')
  return connections.value.length ? t('overview.metrics.providersSupported') : t('overview.metrics.providersRequired')
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
  if (results.some(result => result.status === 'rejected')) error.value = t('overview.errors.partialLoad')
  loading.value = false
}

onMounted(load)
</script>

<template>
  <div class="page dashboard-page">
    <PageHeader
      :eyebrow="t('pages.overview.eyebrow')"
      :title="t('pages.overview.title')"
      :description="t('pages.overview.description')"
    >
      <RouterLink class="button primary" :to="primaryAction.to">{{ primaryAction.label }}</RouterLink>
    </PageHeader>

    <p v-if="error" class="notice error" role="alert">{{ error }}</p>

    <section class="metric-grid overview-metrics" :aria-label="t('overview.metrics.ariaLabel')">
      <RouterLink class="metric-card" to="/profiles"><span>{{ t('overview.metrics.profiles') }}</span><strong>{{ loading ? '—' : profiles.length }}</strong><small>{{ metricDetail('profiles') }}</small></RouterLink>
      <RouterLink class="metric-card" to="/jobs"><span>{{ t('overview.metrics.jobOpportunities') }}</span><strong>{{ loading ? '—' : opportunities.length }}</strong><small>{{ metricDetail('opportunities') }}</small></RouterLink>
      <RouterLink class="metric-card" to="/templates"><span>{{ t('overview.metrics.readyTemplates') }}</span><strong>{{ loading ? '—' : readyTemplates }}</strong><small>{{ metricDetail('templates') }}</small></RouterLink>
      <RouterLink class="metric-card" to="/settings"><span>{{ t('overview.metrics.llmProviders') }}</span><strong>{{ loading ? '—' : connections.length }}</strong><small>{{ metricDetail('connections') }}</small></RouterLink>
      <RouterLink class="metric-card" to="/generate"><span>{{ t('overview.metrics.generatedPdfs') }}</span><strong>{{ loading ? '—' : readyGenerations }}</strong><small>{{ generations.some(item => item.state === 'running' || item.state === 'queued') ? t('overview.metrics.generationInProgress') : t('overview.metrics.generationReviewed') }}</small></RouterLink>
    </section>

    <div class="dashboard-grid">
      <section v-if="recentOpportunities.length" class="panel recent-panel">
        <div class="panel-heading"><div><p class="eyebrow">{{ t('overview.recent.eyebrow') }}</p><h2>{{ t('overview.recent.title') }}</h2></div><RouterLink class="text-button" to="/jobs">{{ t('overview.recent.viewAll') }}</RouterLink></div>
        <div class="recent-list">
          <RouterLink v-for="item in recentOpportunities" :key="item.id" class="recent-item" :to="`/jobs/${item.id}`">
            <span class="company-mark">{{ (item.company || '?').slice(0, 2).toUpperCase() }}</span>
            <div><strong>{{ item.title || t('overview.recent.titlePending') }}</strong><p>{{ item.company || t('overview.recent.companyPending') }}</p></div>
            <span class="status-pill">{{ jobStatusLabel(item.status) }}</span>
          </RouterLink>
        </div>
      </section>

      <section class="panel setup-panel">
        <div class="panel-heading">
          <div><p class="eyebrow">{{ t('overview.setup.eyebrow') }}</p><h2>{{ t('overview.setup.title') }}</h2></div>
          <span class="status-pill">{{ prerequisitesReady ? t('overview.setup.inputsReady') : t('overview.setup.inProgress') }}</span>
        </div>
        <div class="steps">
          <RouterLink v-for="step in setupSteps" :key="step.number" class="step" :class="{ complete: step.complete, planned: step.planned }" :to="step.to">
            <span class="step-number">{{ step.complete ? '✓' : step.number }}</span>
            <div><strong>{{ step.title }}</strong><p>{{ step.description }}</p></div>
            <span v-if="step.planned" class="step-state">{{ t('overview.setup.comingSoon') }}</span><span v-else class="arrow">→</span>
          </RouterLink>
        </div>
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
.dashboard-grid { display: grid; grid-template-columns: minmax(0, 1fr); gap: 1.25rem; align-items: start; }
.setup-panel, .recent-panel { min-width: 0; }
.setup-panel .panel-heading, .recent-panel .panel-heading { gap: 1rem; }
.recent-panel .panel-heading { align-items: flex-end; margin-bottom: 0; padding-bottom: 18px; }
.recent-panel .text-button { flex: 0 0 auto; white-space: nowrap; }
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
@media (max-width: 1050px) { .overview-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 620px) { .overview-metrics { grid-template-columns: 1fr; } .recent-item { grid-template-columns: auto minmax(0, 1fr); } .recent-item .status-pill { grid-column: 2; } }
</style>
