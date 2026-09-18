<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'

const { t } = useI18n()

import ConfirmDialog from '../components/ConfirmDialog.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { jobStatuses } from '../lib/jobStatus'
import { toast } from '../lib/toast'
import type { Job, JobInput } from '../lib/types'

const route = useRoute()
const router = useRouter()
const jobId = computed(() => String(route.params.jobId))
const job = ref<Job>()
const form = reactive<JobInput>({ title: '', company: '', location: '', country: '', city: '', workMode: '', employmentType: '', sourceUrl: '', description: '', status: 'interested' })
const loading = ref(true)
const saving = ref(false)
const deleting = ref(false)
const confirmDelete = ref(false)
const error = ref('')
const importing = computed(() => job.value && ['queued', 'fetching'].includes(job.value.importState))

async function load() {
  loading.value = true
  error.value = ''
  try {
    const selected = await api.getJob(jobId.value)
    job.value = selected
    Object.assign(form, {
      title: selected.title, company: selected.company, location: selected.location ?? '', country: selected.country ?? '',
      city: selected.city ?? '', workMode: selected.workMode ?? '', employmentType: selected.employmentType ?? '',
      sourceUrl: selected.sourceUrl ?? '', description: selected.description ?? '', status: selected.status,
    })
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('jobs.detail.errors.load')
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  error.value = ''
  try {
    job.value = await api.updateJob(jobId.value, { ...form })
    toast.success(t('jobs.detail.toast.saved'))
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('jobs.detail.errors.save')
  } finally {
    saving.value = false
  }
}

async function removeJob() {
  deleting.value = true
  try {
    await api.deleteJob(jobId.value)
    toast.success(t('jobs.toast.deleted'))
    await router.push('/jobs')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('jobs.detail.errors.delete')
    deleting.value = false
  }
}

watch(jobId, () => void load(), { immediate: true })
</script>

<template>
  <div class="page job-editor">
    <RouterLink to="/jobs">{{ t('jobs.detail.backLink') }}</RouterLink>
    <PageHeader :title="job?.title || t('jobs.detail.defaultTitle')" :description="t('jobs.detail.description')">
      <div class="header-actions">
        <a v-if="job?.sourceUrl" class="button" :href="job.sourceUrl" target="_blank" rel="noopener noreferrer">{{ t('jobs.preview.openSource') }}</a>
        <button class="button" type="button" :disabled="loading" @click="load">{{ t('common.refresh') }}</button>
        <button class="button primary" type="button" :disabled="loading || saving" @click="save">{{ saving ? t('common.saving') : t('jobs.detail.save') }}</button>
      </div>
    </PageHeader>

    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <p v-if="importing" class="notice" role="status">{{ t('jobs.detail.importingNotice') }}</p>
    <p v-if="job?.importError" class="notice error">{{ job.importError }}</p>
    <div v-if="loading" class="empty-state">{{ t('jobs.detail.loading') }}</div>

    <template v-else-if="job">
      <form class="panel form-grid" @submit.prevent="save">
        <label><span>{{ t('jobs.detail.titleLabel') }}</span><input v-model="form.title" required maxlength="300" /></label>
        <label><span>{{ t('jobs.detail.companyLabel') }}</span><input v-model="form.company" required maxlength="300" /></label>
        <label><span>{{ t('jobs.detail.statusLabel') }}</span><select v-model="form.status"><option v-for="status in jobStatuses" :key="status.value" :value="status.value">{{ t(status.labelKey) }}</option></select></label>
        <label><span>{{ t('jobs.detail.locationLabel') }}</span><input v-model="form.location" maxlength="300" /></label>
        <label><span>{{ t('jobs.detail.cityLabel') }}</span><input v-model="form.city" maxlength="150" /></label>
        <label><span>{{ t('jobs.detail.countryLabel') }}</span><input v-model="form.country" maxlength="100" /></label>
        <label><span>{{ t('jobs.detail.workModeLabel') }}</span><input v-model="form.workMode" maxlength="100" /></label>
        <label><span>{{ t('jobs.detail.employmentTypeLabel') }}</span><input v-model="form.employmentType" maxlength="100" /></label>
        <label class="full"><span>{{ t('jobs.detail.sourceUrlLabel') }}</span><input v-model="form.sourceUrl" type="url" maxlength="2048" /></label>
        <label class="full"><span>{{ t('jobs.detail.descriptionLabel') }}</span><textarea v-model="form.description" rows="18" maxlength="1048576" /></label>
        <div class="full form-actions"><button class="button primary" :disabled="saving">{{ t('jobs.detail.save') }}</button></div>
      </form>
      <section class="danger-zone"><div><h2>{{ t('jobs.detail.dangerZoneTitle') }}</h2><p>{{ t('jobs.detail.dangerZoneMessage') }}</p></div><button class="button" type="button" :disabled="deleting" @click="confirmDelete = true">{{ t('jobs.detail.deleteSubmit') }}</button></section>
    </template>
    <ConfirmDialog :open="confirmDelete" :title="t('jobs.deleteConfirm.title')" :message="t('jobs.detail.deleteConfirmMessage', { name: [job?.title, job?.company].filter(Boolean).join(' at ') || t('jobs.deleteConfirm.defaultName') })" :busy="deleting" @cancel="confirmDelete = false" @confirm="removeJob" />
  </div>
</template>

<style scoped>
.job-editor { display: grid; gap: 1.5rem; }
.header-actions { display: flex; gap: .7rem; }
.danger-zone { display: flex; justify-content: space-between; align-items: center; gap: 1rem; padding: 1.25rem; border: 1px solid #e6b8b8; border-radius: .75rem; }
.danger-zone h2, .danger-zone p { margin: 0; }
@media (max-width: 620px) { .danger-zone, .header-actions { align-items: stretch; flex-direction: column; } }
</style>
