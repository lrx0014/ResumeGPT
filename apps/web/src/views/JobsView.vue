<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'

import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import type { Job } from '../lib/types'

const jobs = ref<Job[]>([])
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const showForm = ref(false)
const form = reactive({ title: '', company: '', location: '', sourceUrl: '', description: '' })

async function load() {
  loading.value = true
  try {
    jobs.value = (await api.listJobs()).items
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not load jobs.'
  } finally {
    loading.value = false
  }
}

async function createJob() {
  saving.value = true
  error.value = ''
  try {
    const created = await api.createJob(form)
    jobs.value.unshift(created)
    Object.assign(form, { title: '', company: '', location: '', sourceUrl: '', description: '' })
    showForm.value = false
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not create the job.'
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="Jobs" description="Capture target roles and keep every application in one timeline.">
      <button class="button primary" type="button" @click="showForm = !showForm">{{ showForm ? 'Close' : 'Add job' }}</button>
    </PageHeader>

    <form v-if="showForm" class="panel form-grid" @submit.prevent="createJob">
      <label><span>Job title</span><input v-model="form.title" required placeholder="Senior Software Engineer" /></label>
      <label><span>Company</span><input v-model="form.company" required placeholder="Example GmbH" /></label>
      <label><span>Location</span><input v-model="form.location" placeholder="Berlin · Hybrid" /></label>
      <label><span>Source URL</span><input v-model="form.sourceUrl" type="url" placeholder="https://…" /></label>
      <label class="full"><span>Job description</span><textarea v-model="form.description" rows="5" placeholder="Paste the role description for now. Automated URL ingestion comes next." /></label>
      <div class="full form-actions"><button class="button primary" :disabled="saving">{{ saving ? 'Saving…' : 'Track job' }}</button></div>
    </form>

    <p v-if="error" class="notice error">{{ error }}</p>
    <div v-if="loading" class="empty-state">Loading jobs…</div>
    <div v-else-if="jobs.length" class="list-panel">
      <article v-for="job in jobs" :key="job.id" class="job-row">
        <span class="company-mark">{{ job.company.slice(0, 2).toUpperCase() }}</span>
        <div class="job-main"><h2>{{ job.title }}</h2><p>{{ job.company }}<span v-if="job.location"> · {{ job.location }}</span></p></div>
        <span class="status-pill">{{ job.status }}</span>
        <button class="text-button" type="button">View →</button>
      </article>
    </div>
    <div v-else class="empty-state">
      <span class="empty-icon">◇</span><h2>No tracked roles</h2><p>Add a role manually. URL extraction can be attached behind the same API later.</p>
    </div>
  </div>
</template>

