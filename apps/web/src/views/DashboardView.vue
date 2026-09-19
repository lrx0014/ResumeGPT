<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { jobStatusLabel } from '../lib/jobStatus'
import type { Job, WorkspaceOverview } from '../lib/types'

const emptyOverview: WorkspaceOverview = { profiles: { total: 0, withContent: 0 }, jobs: { total: 0, ready: 0, pending: 0, attention: 0 }, templates: { ready: 0, custom: 0 }, llmConnections: { total: 0 }, generations: { ready: 0, active: 0 } }

const overview = ref<WorkspaceOverview>(emptyOverview)
const opportunities = ref<Job[]>([])
const capabilities = ref<Record<string, boolean>>({})
const loading = ref(true)
const error = ref('')

const recentOpportunities = computed(() => [...opportunities.value].sort((left, right) => right.updatedAt.localeCompare(left.updatedAt)).slice(0, 3))
const prerequisitesReady = computed(() => overview.value.profiles.withContent > 0 && overview.value.jobs.ready > 0 && overview.value.templates.ready > 0 && overview.value.llmConnections.total > 0)

const setupSteps = computed(() => [
  { number: '01', title: t('overview.setup.steps.profile.title'), description: overview.value.profiles.withContent ? t('overview.setup.steps.profile.detail', { count: overview.value.profiles.withContent }, overview.value.profiles.withContent) : t('overview.setup.steps.profile.empty'), to: '/profiles', complete: overview.value.profiles.withContent > 0 },
  { number: '02', title: t('overview.setup.steps.opportunity.title'), description: overview.value.jobs.ready ? t('overview.setup.steps.opportunity.detail', { count: overview.value.jobs.ready }, overview.value.jobs.ready) : t('overview.setup.steps.opportunity.empty'), to: '/jobs', complete: overview.value.jobs.ready > 0 },
  { number: '03', title: t('overview.setup.steps.template.title'), description: overview.value.templates.ready ? t('overview.setup.steps.template.detail', { count: overview.value.templates.ready }, overview.value.templates.ready) : t('overview.setup.steps.template.empty'), to: '/templates', complete: overview.value.templates.ready > 0 },
  { number: '04', title: t('overview.setup.steps.provider.title'), description: overview.value.llmConnections.total ? t('overview.setup.steps.provider.detail', { count: overview.value.llmConnections.total }, overview.value.llmConnections.total) : t('overview.setup.steps.provider.empty'), to: '/settings', complete: overview.value.llmConnections.total > 0 },
  { number: '05', title: t('overview.setup.steps.generate.title'), description: overview.value.generations.ready ? t('overview.setup.steps.generate.detail', { count: overview.value.generations.ready }, overview.value.generations.ready) : capabilities.value.generation ? t('overview.setup.steps.generate.readyCapability') : t('overview.setup.steps.generate.pendingCapability'), to: '/generate', complete: overview.value.generations.ready > 0, planned: !capabilities.value.generation },
])

const firstIncomplete = computed(() => setupSteps.value.find(step => !step.complete && !step.planned))
const primaryAction = computed(() => firstIncomplete.value
  ? { label: t('overview.actions.continue', { title: firstIncomplete.value.title }), to: firstIncomplete.value.to }
  : { label: capabilities.value.generation ? t('overview.actions.createDocument') : t('overview.actions.reviewSetup'), to: '/generate' })

function metricDetail(kind: 'profiles' | 'opportunities' | 'templates' | 'connections') {
  if (loading.value) return t('overview.metrics.loading')
  if (kind === 'profiles') return overview.value.profiles.withContent ? t('overview.metrics.profilesDetail', { count: overview.value.profiles.withContent }, overview.value.profiles.withContent) : t('overview.metrics.profilesEmpty')
  if (kind === 'opportunities') {
    if (overview.value.jobs.attention) return t('overview.metrics.opportunitiesAttention', { count: overview.value.jobs.attention }, overview.value.jobs.attention)
    if (overview.value.jobs.pending) return t('overview.metrics.opportunitiesPending', { count: overview.value.jobs.pending }, overview.value.jobs.pending)
    return overview.value.jobs.total ? t('overview.metrics.opportunitiesReady', { count: overview.value.jobs.ready }, overview.value.jobs.ready) : t('overview.metrics.opportunitiesEmpty')
  }
  if (kind === 'templates') return overview.value.templates.custom ? t('overview.metrics.templatesCustom', { count: overview.value.templates.custom }, overview.value.templates.custom) : t('overview.metrics.templatesEmpty')
  return overview.value.llmConnections.total ? t('overview.metrics.providersSupported') : t('overview.metrics.providersRequired')
}

async function load() {
  loading.value = true
  const results = await Promise.allSettled([
    api.overview(), api.listJobs(), api.capabilities(),
  ])
  if (results[0].status === 'fulfilled') overview.value = results[0].value
  if (results[1].status === 'fulfilled') opportunities.value = results[1].value.items
  if (results[2].status === 'fulfilled') capabilities.value = results[2].value.features
  if (results.some(result => result.status === 'rejected')) error.value = t('overview.errors.partialLoad')
  loading.value = false
}

onMounted(load)
</script>

<template>
  <div class="page dashboard-page">
    <div class="page-corner-links">
      <a class="icon-link" href="https://github.com/lrx0014/ResumeGPT" target="_blank" rel="noopener noreferrer" :aria-label="t('overview.links.github')" :title="t('overview.links.github')">
        <svg viewBox="0 0 16 16" fill="currentColor" aria-hidden="true"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z"></path></svg>
      </a>
      <a class="icon-link brand-mark" href="https://resumegpt.tech-fun.net" target="_blank" rel="noopener noreferrer" :aria-label="t('overview.links.homepage')" :title="t('overview.links.homepage')">Ré</a>
    </div>
    <PageHeader
      :eyebrow="t('pages.overview.eyebrow')"
      :title="t('pages.overview.title')"
      :description="t('pages.overview.description')"
    >
      <RouterLink class="button primary" :to="primaryAction.to">{{ primaryAction.label }}</RouterLink>
    </PageHeader>

    <p v-if="error" class="notice error" role="alert">{{ error }}</p>

    <section class="metric-grid overview-metrics" :aria-label="t('overview.metrics.ariaLabel')">
      <RouterLink class="metric-card" to="/profiles"><span>{{ t('overview.metrics.profiles') }}</span><strong>{{ loading ? '—' : overview.profiles.total }}</strong><small>{{ metricDetail('profiles') }}</small></RouterLink>
      <RouterLink class="metric-card" to="/jobs"><span>{{ t('overview.metrics.jobOpportunities') }}</span><strong>{{ loading ? '—' : overview.jobs.total }}</strong><small>{{ metricDetail('opportunities') }}</small></RouterLink>
      <RouterLink class="metric-card" to="/templates"><span>{{ t('overview.metrics.readyTemplates') }}</span><strong>{{ loading ? '—' : overview.templates.ready }}</strong><small>{{ metricDetail('templates') }}</small></RouterLink>
      <RouterLink class="metric-card" to="/settings"><span>{{ t('overview.metrics.llmProviders') }}</span><strong>{{ loading ? '—' : overview.llmConnections.total }}</strong><small>{{ metricDetail('connections') }}</small></RouterLink>
      <RouterLink class="metric-card" to="/generate"><span>{{ t('overview.metrics.generatedPdfs') }}</span><strong>{{ loading ? '—' : overview.generations.ready }}</strong><small>{{ overview.generations.active > 0 ? t('overview.metrics.generationInProgress') : t('overview.metrics.generationReviewed') }}</small></RouterLink>
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
.dashboard-page { position: relative; display: grid; gap: 1.25rem; }
.dashboard-page :deep(.page-header) { margin-bottom: .5rem; }
.page-corner-links { position: absolute; top: 4px; right: 0; display: flex; align-items: center; gap: .5rem; }
.icon-link { display: grid; width: 34px; height: 34px; place-items: center; border-radius: 10px; border: 1px solid var(--line); background: var(--surface); color: var(--ink); transition: transform .18s ease, border-color .18s ease; }
.icon-link:hover { border-color: var(--accent); transform: translateY(-1px); }
.icon-link svg { width: 17px; height: 17px; }
.icon-link.brand-mark { border-color: transparent; font-size: 15px; }
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
@media (max-width: 760px) { .page-corner-links { position: static; justify-content: flex-end; margin-bottom: .75rem; } }
@media (max-width: 620px) { .overview-metrics { grid-template-columns: 1fr; } .recent-item { grid-template-columns: auto minmax(0, 1fr); } .recent-item .status-pill { grid-column: 2; } }
</style>
