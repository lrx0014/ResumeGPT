<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

import ConfirmDialog from '../components/ConfirmDialog.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { toast } from '../lib/toast'
import type { AgentDefault, GenerationModelChoice, JobHunter, JobHunterInput, LLMConnection, Profile } from '../lib/types'

const hunters = ref<JobHunter[]>([])
const connections = ref<LLMConnection[]>([])
const profiles = ref<Profile[]>([])
const models = reactive<Record<string, string[]>>({})
const showForm = ref(false)
const editingId = ref('')
const loading = ref(true)
const busy = ref(false)
const loadingModels = ref(false)
const error = ref('')
const pendingDelete = ref<JobHunter | null>(null)
const hunterDefault = reactive<GenerationModelChoice>({ connectionId: '', model: '' })
let pollTimer: number | undefined

function blankForm(): JobHunterInput {
  return { name: '', roleQuery: '', location: '', workMode: '', employmentType: '', experienceYears: undefined,
    keywords: '', additionalPrompt: '', profileId: '', connectionId: '', model: '', maxResults: 10, intervalMinutes: 1440, enabled: true }
}
const form = reactive<JobHunterInput>(blankForm())
const activeCount = computed(() => hunters.value.filter(item => item.enabled).length)

function formatTime(value?: string) {
  return value ? new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : t('hunters.card.notRunYet')
}

function scheduleLabel(minutes: number) {
  return ({ 360: t('hunters.schedule.every6h'), 720: t('hunters.schedule.every12h'), 1440: t('hunters.schedule.daily'), 10080: t('hunters.schedule.weekly') } as Record<number, string>)[minutes] ?? t('hunters.schedule.minutes', { minutes })
}

function stateLabel(state: JobHunter['lastState']) {
  return ({ never: t('hunters.state.ready'), queued: t('hunters.state.queued'), running: t('hunters.state.searching'), succeeded: t('hunters.state.completed'), failed: t('common.needsAttention') } as Record<string, string>)[state]
}

function profileName(profileId?: string) {
  return profiles.value.find(item => item.id === profileId)?.name ?? t('hunters.form.noProfileReference')
}

async function load(showLoading = true) {
  if (showLoading) loading.value = true
  try {
    hunters.value = (await api.listJobHunters()).items
    error.value = ''
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('hunters.errors.loadFailed')
  } finally {
    loading.value = false
  }
}

async function discoverModels(resetModel = false) {
  if (resetModel) form.model = ''
  if (!form.connectionId) return
  if (models[form.connectionId]) {
    if (!form.model) form.model = models[form.connectionId][0] ?? ''
    return
  }
  loadingModels.value = true
  try {
    models[form.connectionId] = (await api.testLLMConnection(form.connectionId)).models
    if (!form.model) form.model = models[form.connectionId][0] ?? ''
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('hunters.errors.modelsFailed')
  } finally {
    loadingModels.value = false
  }
}

function openCreate() {
  editingId.value = ''
  const configured = connections.value.some(item => item.id === hunterDefault.connectionId) ? hunterDefault : undefined
  Object.assign(form, blankForm(), { connectionId: configured?.connectionId ?? connections.value[0]?.id ?? '', model: configured?.model ?? '' })
  void discoverModels()
  showForm.value = true
}

function openEdit(item: JobHunter) {
  editingId.value = item.id
  Object.assign(form, {
    name: item.name, roleQuery: item.roleQuery, location: item.location ?? '', workMode: item.workMode ?? '',
    employmentType: item.employmentType ?? '', experienceYears: item.experienceYears, keywords: item.keywords ?? '',
    additionalPrompt: item.additionalPrompt ?? '', profileId: item.profileId ?? '', connectionId: item.connectionId, model: item.model,
    maxResults: item.maxResults,
    intervalMinutes: item.intervalMinutes, enabled: item.enabled,
  })
  void discoverModels()
  showForm.value = true
}

async function save() {
  busy.value = true
  error.value = ''
  try {
    const experience = form.experienceYears as number | string | undefined
    const payload = { ...form, experienceYears: experience === undefined || experience === '' ? undefined : Number(experience), maxResults: Number(form.maxResults) }
    const saved = editingId.value ? await api.updateJobHunter(editingId.value, payload) : await api.createJobHunter(payload)
    const index = hunters.value.findIndex(item => item.id === saved.id)
    if (index >= 0) hunters.value[index] = saved
    else hunters.value.unshift(saved)
    showForm.value = false
    toast.success(editingId.value ? t('hunters.toast.updated') : t('hunters.toast.created'))
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('hunters.errors.saveFailed')
  } finally {
    busy.value = false
  }
}

async function runNow(item: JobHunter) {
  try {
    await api.runJobHunter(item.id)
    item.lastState = 'queued'
    toast.success(t('hunters.toast.queued'))
  } catch (cause) {
    toast.warning(cause instanceof Error ? cause.message : t('hunters.errors.queueFailed'))
  }
}

async function toggle(item: JobHunter) {
  try {
    const updated = await api.updateJobHunter(item.id, {
      name: item.name, roleQuery: item.roleQuery, location: item.location, workMode: item.workMode,
      employmentType: item.employmentType, experienceYears: item.experienceYears, keywords: item.keywords,
      additionalPrompt: item.additionalPrompt, profileId: item.profileId, connectionId: item.connectionId, model: item.model,
      maxResults: item.maxResults,
      intervalMinutes: item.intervalMinutes, enabled: !item.enabled,
    })
    Object.assign(item, updated)
    toast.success(updated.enabled ? t('hunters.toast.resumed') : t('hunters.toast.paused'))
  } catch (cause) {
    toast.warning(cause instanceof Error ? cause.message : t('hunters.errors.updateFailed'))
  }
}

async function remove() {
  const item = pendingDelete.value
  if (!item) return
  busy.value = true
  try {
    await api.deleteJobHunter(item.id)
    hunters.value = hunters.value.filter(candidate => candidate.id !== item.id)
    pendingDelete.value = null
    toast.success(t('hunters.toast.deleted'))
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('hunters.errors.deleteFailed')
  } finally {
    busy.value = false
  }
}

onMounted(async () => {
  const [connectionResult, profileResult, storedDefaults] = await Promise.all([
    api.listLLMConnections().catch(() => ({ items: [] })),
    api.listProfiles().catch(() => ({ items: [] })),
    api.getAgentDefaults().catch(() => ({ items: [] as AgentDefault[] })),
    load(),
  ])
  connections.value = connectionResult.items
  profiles.value = profileResult.items
  const configured = storedDefaults.items.find(item => item.agent === 'job_hunter')
  if (configured) Object.assign(hunterDefault, { connectionId: configured.connectionId, model: configured.model })
  pollTimer = window.setInterval(() => { void load(false) }, 5000)
})
onBeforeUnmount(() => { if (pollTimer) window.clearInterval(pollTimer) })
</script>

<template>
  <div class="page hunters-page">
    <PageHeader :title="t('pages.hunters.title')" :description="t('pages.hunters.description')">
      <button class="button primary" type="button" @click="showForm ? showForm = false : openCreate()">{{ showForm ? t('common.close') : t('hunters.createButton') }}</button>
    </PageHeader>

    <section class="hunter-summary" :aria-label="t('hunters.summary.ariaLabel')">
      <div><strong>{{ hunters.length }}</strong><span>{{ t('hunters.summary.savedSearches') }}</span></div>
      <div><strong>{{ activeCount }}</strong><span>{{ t('hunters.summary.activeSchedules') }}</span></div>
      <div><strong>{{ hunters.reduce((sum, item) => sum + item.lastFoundCount, 0) }}</strong><span>{{ t('hunters.summary.foundInLatestRuns') }}</span></div>
    </section>

    <form v-if="showForm" class="panel form-grid hunter-form" @submit.prevent="save">
      <div class="full form-intro"><div><p class="eyebrow">{{ editingId ? t('hunters.form.editTitle') : t('hunters.form.newTitle') }}</p><h2>{{ t('hunters.form.heading') }}</h2></div><span>{{ t('hunters.form.intro') }}</span></div>
      <label><span>{{ t('hunters.form.nameLabel') }}</span><input v-model="form.name" required maxlength="120" :placeholder="t('hunters.form.namePlaceholder')" /></label>
      <label><span>{{ t('hunters.form.roleLabel') }}</span><input v-model="form.roleQuery" required maxlength="300" :placeholder="t('hunters.form.rolePlaceholder')" /></label>
      <label><span>{{ t('hunters.form.locationLabel') }}</span><input v-model="form.location" maxlength="300" :placeholder="t('hunters.form.locationPlaceholder')" /></label>
      <label><span>{{ t('hunters.form.workModeLabel') }}</span><select v-model="form.workMode"><option value="">{{ t('hunters.form.workModeAny') }}</option><option>{{ t('hunters.form.workModeRemote') }}</option><option>{{ t('hunters.form.workModeHybrid') }}</option><option>{{ t('hunters.form.workModeOnsite') }}</option></select></label>
      <label><span>{{ t('hunters.form.employmentTypeLabel') }}</span><select v-model="form.employmentType"><option value="">{{ t('hunters.form.employmentTypeAny') }}</option><option>{{ t('hunters.form.employmentTypeFullTime') }}</option><option>{{ t('hunters.form.employmentTypePartTime') }}</option><option>{{ t('hunters.form.employmentTypeContract') }}</option><option>{{ t('hunters.form.employmentTypeInternship') }}</option></select></label>
      <label><span>{{ t('hunters.form.experienceLabel') }}</span><input v-model.number="form.experienceYears" type="number" min="0" max="60" :placeholder="t('hunters.form.experiencePlaceholder')" /></label>
      <label class="full"><span>{{ t('hunters.form.keywordsLabel') }}</span><input v-model="form.keywords" maxlength="1000" :placeholder="t('hunters.form.keywordsPlaceholder')" /></label>
      <label class="full"><span>{{ t('hunters.form.additionalPromptLabel') }}</span><textarea v-model="form.additionalPrompt" rows="4" maxlength="4000" :placeholder="t('hunters.form.additionalPromptPlaceholder')" /></label>
      <label><span>{{ t('hunters.form.profileLabel') }} <small>{{ t('hunters.form.optional') }}</small></span><select v-model="form.profileId"><option value="">{{ t('hunters.form.noProfileReference') }}</option><option v-for="item in profiles" :key="item.id" :value="item.id">{{ item.name }}{{ item.targetRole ? ` · ${item.targetRole}` : '' }}</option></select></label>
      <label><span>{{ t('hunters.form.maxResultsLabel') }}</span><input v-model.number="form.maxResults" type="number" min="1" max="10" required /><small class="field-hint">{{ t('hunters.form.maxResultsHint') }}</small></label>
      <label><span>{{ t('hunters.form.scheduleLabel') }}</span><select v-model.number="form.intervalMinutes"><option :value="360">{{ t('hunters.schedule.every6h') }}</option><option :value="720">{{ t('hunters.schedule.every12h') }}</option><option :value="1440">{{ t('hunters.schedule.daily') }}</option><option :value="10080">{{ t('hunters.schedule.weekly') }}</option></select></label>
      <label><span>{{ t('hunters.form.providerLabel') }}</span><select v-model="form.connectionId" required @change="discoverModels(true)"><option value="" disabled>{{ t('hunters.form.selectProvider') }}</option><option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option></select></label>
      <label><span>{{ t('hunters.form.modelLabel') }}</span><select v-if="models[form.connectionId]?.length" v-model="form.model" required><option value="" disabled>{{ t('hunters.form.selectModel') }}</option><option v-if="form.model && !models[form.connectionId].includes(form.model)" :value="form.model">{{ form.model }}</option><option v-for="model in models[form.connectionId]" :key="model" :value="model">{{ model }}</option></select><input v-else v-model="form.model" required :disabled="loadingModels" :placeholder="loadingModels ? t('hunters.form.loadingModels') : t('hunters.form.enterModelName')" /></label>
      <label class="enabled-field"><input v-model="form.enabled" type="checkbox" /><span>{{ t('hunters.form.enableAutomaticRuns') }}</span></label>
      <p v-if="!connections.length" class="full notice">{{ t('hunters.form.noticeBefore') }}<RouterLink to="/settings">{{ t('hunters.form.noticeLinkText') }}</RouterLink>{{ t('hunters.form.noticeAfter') }}</p>
      <div class="full form-actions"><button class="button primary" :disabled="busy || !connections.length || !form.model">{{ busy ? t('common.saving') : editingId ? t('hunters.form.saveChanges') : t('hunters.form.createAndSchedule') }}</button></div>
    </form>

    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <div v-if="loading" class="empty-state">{{ t('hunters.loadingHunters') }}</div>
    <div v-else-if="hunters.length" class="hunter-grid">
      <article v-for="item in hunters" :key="item.id" class="panel hunter-card">
        <header><div><p class="eyebrow">{{ scheduleLabel(item.intervalMinutes) }}</p><h2>{{ item.name }}</h2></div><span class="status-pill" :class="item.lastState">{{ item.enabled ? stateLabel(item.lastState) : t('hunters.state.paused') }}</span></header>
        <p class="role-query">{{ item.roleQuery }}<span v-if="item.location"> · {{ item.location }}</span></p>
        <div class="criteria"><span v-if="item.profileId" :title="t('hunters.card.profileLabel', { name: profileName(item.profileId) })">{{ t('hunters.card.profileLabel', { name: profileName(item.profileId) }) }}</span><span :title="t('hunters.card.upToPerRun', { count: item.maxResults })">{{ t('hunters.card.upToPerRun', { count: item.maxResults }) }}</span><span v-if="item.workMode" :title="item.workMode">{{ item.workMode }}</span><span v-if="item.employmentType" :title="item.employmentType">{{ item.employmentType }}</span><span v-if="item.experienceYears !== undefined" :title="t('hunters.card.yearsPlus', { years: item.experienceYears })">{{ t('hunters.card.yearsPlus', { years: item.experienceYears }) }}</span><span v-if="item.keywords" :title="item.keywords">{{ item.keywords }}</span></div>
        <RouterLink v-if="item.reviewCount" class="confirmation-link" :to="`/job-hunters/${item.id}/review`"><span class="confirmation-icon">!</span><span><strong>{{ t('hunters.card.jobsNeedConfirmation', { count: item.reviewCount }, item.reviewCount) }}</strong><small>{{ t('hunters.card.reviewHint') }}</small></span><span aria-hidden="true">→</span></RouterLink>
        <p v-if="item.lastError" class="hunter-error">{{ item.lastError }}</p>
        <dl><div><dt>{{ t('hunters.card.nextRun') }}</dt><dd>{{ item.enabled ? formatTime(item.nextRunAt) : t('hunters.state.paused') }}</dd></div><div><dt>{{ t('hunters.card.lastRun') }}</dt><dd>{{ formatTime(item.lastRunAt) }}</dd></div><div><dt>{{ t('hunters.card.newOpportunities') }}</dt><dd>{{ item.lastFoundCount }}</dd></div></dl>
        <footer><button class="text-button" type="button" :disabled="['queued', 'running'].includes(item.lastState)" @click="runNow(item)">{{ t('hunters.card.runNow') }}</button><button class="text-button" type="button" @click="openEdit(item)">{{ t('common.edit') }}</button><button class="text-button" type="button" @click="toggle(item)">{{ item.enabled ? t('hunters.card.pause') : t('hunters.card.resume') }}</button><button class="text-button danger-text" type="button" @click="pendingDelete = item">{{ t('common.delete') }}</button></footer>
      </article>
    </div>
    <div v-else class="empty-state"><span class="empty-icon">⌖</span><h2>{{ t('hunters.empty.title') }}</h2><p>{{ t('hunters.empty.description') }}</p><button class="button primary" type="button" @click="openCreate">{{ t('hunters.createButton') }}</button></div>

    <ConfirmDialog :open="Boolean(pendingDelete)" :title="t('hunters.deleteDialog.title')" :message="t('hunters.deleteDialog.message', { name: pendingDelete?.name || t('hunters.deleteDialog.fallbackName') })" :busy="busy" @cancel="pendingDelete = null" @confirm="remove" />
  </div>
</template>

<style scoped>
.hunters-page { display: grid; gap: 1.25rem; }
.hunters-page :deep(.page-header) { margin-bottom: .5rem; }
.hunter-summary { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; }
.hunter-summary div { display: flex; align-items: baseline; gap: 10px; padding: 16px 19px; border: 1px solid var(--line); border-radius: 12px; background: var(--surface-soft); }
.hunter-summary strong { font-size: 24px; }
.hunter-summary span { color: var(--muted); font-size: 11px; }
.form-intro { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; padding-bottom: 10px; border-bottom: 1px solid var(--line); }
.form-intro h2, .form-intro .eyebrow { margin-bottom: 0; }
.form-intro > span { max-width: 390px; color: var(--muted); font-size: 11px; line-height: 1.5; }
.enabled-field { display: flex !important; grid-template-columns: auto 1fr !important; align-items: center; align-self: end; min-height: 42px; }
.enabled-field input { width: 17px; height: 17px; accent-color: var(--accent); }
.field-hint { color: var(--muted); font-size: 10px; line-height: 1.4; }
.hunter-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
.hunter-card { display: grid; min-width: 0; gap: 16px; }
.hunter-card header, .hunter-card footer { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.hunter-card h2, .hunter-card .eyebrow { margin-bottom: 0; }
.role-query { margin: 0; font-size: 14px; font-weight: 700; }
.criteria { display: flex; align-self: start; align-items: flex-start; flex-wrap: wrap; gap: 7px; }
.criteria span { max-width: 100%; padding: 5px 8px; overflow: hidden; border-radius: 6px; background: var(--surface-soft); color: var(--muted); font-size: 10px; text-overflow: ellipsis; white-space: nowrap; }
.confirmation-link { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: 10px; padding: 10px 12px; border: 1px solid #ead49a; border-radius: 9px; background: #fff9e9; color: #795b16; text-decoration: none; transition: transform .16s ease, box-shadow .16s ease; }
.confirmation-link:hover { transform: translateY(-1px); box-shadow: 0 7px 18px rgb(121 91 22 / 10%); }
.confirmation-link strong, .confirmation-link small { display: block; }
.confirmation-link strong { font-size: 11px; }
.confirmation-link small { margin-top: 2px; color: #947635; font-size: 9px; }
.confirmation-icon { display: grid; width: 24px; height: 24px; place-items: center; border-radius: 50%; background: #f4dda3; font-weight: 800; }
.hunter-card dl { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; margin: 0; }
.hunter-card dl div { min-width: 0; }
.hunter-card dt { color: var(--muted); font-size: 9px; font-weight: 700; text-transform: uppercase; }
.hunter-card dd { margin: 4px 0 0; overflow-wrap: anywhere; font-size: 11px; }
.hunter-card footer { align-items: center; justify-content: flex-end; padding-top: 12px; border-top: 1px solid var(--line); }
.hunter-error { margin: 0; padding: 9px 11px; overflow-wrap: anywhere; border-radius: 8px; background: #f8e6e6; color: var(--danger); font-size: 10px; }
.status-pill.queued, .status-pill.running { background: #fff2cf; color: #795b16; }
.status-pill.failed { background: #f8e6e6; color: var(--danger); }
@media (max-width: 900px) { .hunter-grid { grid-template-columns: 1fr; } }
@media (max-width: 620px) { .hunter-summary { grid-template-columns: 1fr; } .form-intro { flex-direction: column; } .hunter-card dl { grid-template-columns: 1fr; } .hunter-card footer { align-items: flex-start; flex-direction: column; } }
</style>
