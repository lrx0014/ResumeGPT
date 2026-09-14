<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import type { Job, JobInput } from '../lib/types'

const route = useRoute()
const router = useRouter()
const jobId = computed(() => String(route.params.jobId))
const job = ref<Job>()
const form = reactive<JobInput>({ title: '', company: '', location: '', country: '', city: '', workMode: '', employmentType: '', sourceUrl: '', description: '', status: 'interested' })
const loading = ref(true)
const saving = ref(false)
const deleting = ref(false)
const error = ref('')
const notice = ref('')
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
    error.value = cause instanceof Error ? cause.message : 'Could not load the job.'
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  error.value = ''
  notice.value = ''
  try {
    job.value = await api.updateJob(jobId.value, { ...form })
    notice.value = 'Job saved.'
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not save the job.'
  } finally {
    saving.value = false
  }
}

async function removeJob() {
  if (!window.confirm('Permanently delete this job?')) return
  deleting.value = true
  try {
    await api.deleteJob(jobId.value)
    await router.push('/jobs')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not delete the job.'
    deleting.value = false
  }
}

watch(jobId, () => void load(), { immediate: true })
</script>

<template>
  <div class="page job-editor">
    <RouterLink to="/jobs">← All jobs</RouterLink>
    <PageHeader :title="job?.title || 'Job details'" description="Review imported details, edit anything, and keep the application status current.">
      <div class="header-actions">
        <a v-if="job?.sourceUrl" class="button" :href="job.sourceUrl" target="_blank" rel="noopener noreferrer">Open source ↗</a>
        <button class="button" type="button" :disabled="loading" @click="load">Refresh</button>
        <button class="button primary" type="button" :disabled="loading || saving" @click="save">{{ saving ? 'Saving…' : 'Save job' }}</button>
      </div>
    </PageHeader>

    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <p v-if="notice" class="notice" role="status">{{ notice }}</p>
    <p v-if="importing" class="notice" role="status">ResumeGPT is importing this public job page. Use Refresh to load the latest details without losing edits in progress.</p>
    <p v-if="job?.importError" class="notice error">{{ job.importError }}</p>
    <div v-if="loading" class="empty-state">Loading job…</div>

    <template v-else-if="job">
      <form class="panel form-grid" @submit.prevent="save">
        <label><span>Job title</span><input v-model="form.title" required maxlength="300" /></label>
        <label><span>Company</span><input v-model="form.company" required maxlength="300" /></label>
        <label><span>Application status</span><select v-model="form.status"><option value="interested">Interested</option><option value="preparing">Preparing</option><option value="applied">Applied</option><option value="screening">Screening</option><option value="interview">Interview</option><option value="offer">Offer</option><option value="accepted">Accepted</option><option value="rejected">Rejected</option><option value="withdrawn">Withdrawn</option></select></label>
        <label><span>Location</span><input v-model="form.location" maxlength="300" /></label>
        <label><span>City</span><input v-model="form.city" maxlength="150" /></label>
        <label><span>Country</span><input v-model="form.country" maxlength="100" /></label>
        <label><span>Work mode</span><input v-model="form.workMode" maxlength="100" /></label>
        <label><span>Employment type</span><input v-model="form.employmentType" maxlength="100" /></label>
        <label class="full"><span>Source URL</span><input v-model="form.sourceUrl" type="url" maxlength="2048" /></label>
        <label class="full"><span>Job description</span><textarea v-model="form.description" rows="18" maxlength="1048576" /></label>
        <div class="full form-actions"><button class="button primary" :disabled="saving">Save job</button></div>
      </form>
      <section class="danger-zone"><div><h2>Delete job</h2><p>This removes the job from the active database.</p></div><button class="button" type="button" :disabled="deleting" @click="removeJob">{{ deleting ? 'Deleting…' : 'Delete job' }}</button></section>
    </template>
  </div>
</template>

<style scoped>
.job-editor { display: grid; gap: 1.5rem; }
.header-actions { display: flex; gap: .7rem; }
.danger-zone { display: flex; justify-content: space-between; align-items: center; gap: 1rem; padding: 1.25rem; border: 1px solid #e6b8b8; border-radius: .75rem; }
.danger-zone h2, .danger-zone p { margin: 0; }
@media (max-width: 620px) { .danger-zone, .header-actions { align-items: stretch; flex-direction: column; } }
</style>
