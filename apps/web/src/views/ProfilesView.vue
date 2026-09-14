<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'

import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import type { Profile } from '../lib/types'

const profiles = ref<Profile[]>([])
const router = useRouter()
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const showForm = ref(false)
const form = reactive({ name: '', targetRole: '', defaultLanguage: 'en-US', content: '', avatarObjectId: '' })

async function load() {
  loading.value = true
  error.value = ''
  try {
    profiles.value = (await api.listProfiles()).items
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not load profiles.'
  } finally {
    loading.value = false
  }
}

async function createProfile() {
  saving.value = true
  error.value = ''
  try {
    const created = await api.createProfile(form)
    await router.push(`/profiles/${created.id}`)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not create the profile.'
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="Profiles" description="Create a focused profile for each role or career direction.">
      <button class="button primary" type="button" @click="showForm = !showForm">
        {{ showForm ? 'Close' : 'New profile' }}
      </button>
    </PageHeader>

    <form v-if="showForm" class="panel form-grid" @submit.prevent="createProfile">
      <label><span>Name</span><input v-model="form.name" required placeholder="Backend Engineering" /></label>
      <label><span>Target role</span><input v-model="form.targetRole" placeholder="Senior Backend Engineer" /></label>
      <label>
        <span>Primary language</span>
        <select v-model="form.defaultLanguage"><option value="en-US">English</option><option value="de-DE">German</option></select>
      </label>
      <div class="full form-actions"><button class="button primary" :disabled="saving">{{ saving ? 'Saving…' : 'Create profile' }}</button></div>
    </form>

    <p v-if="error" class="notice error">{{ error }}</p>
    <div v-if="loading" class="empty-state">Loading profiles…</div>
    <div v-else-if="profiles.length" class="card-grid">
      <article v-for="profile in profiles" :key="profile.id" class="entity-card">
        <span class="entity-icon">◎</span>
        <div><p class="eyebrow">{{ profile.targetRole || 'General profile' }}</p><h2>{{ profile.name }}</h2><p>{{ profile.content ? `${profile.content.slice(0, 140)}${profile.content.length > 140 ? '…' : ''}` : 'Add your experience, education, skills, and achievements.' }}</p></div>
        <footer><span>{{ profile.defaultLanguage }}</span><RouterLink class="text-button" :to="`/profiles/${profile.id}`">Open →</RouterLink></footer>
      </article>
    </div>
    <div v-else class="empty-state">
      <span class="empty-icon">◎</span><h2>No profiles yet</h2><p>Create a profile to organize experience for a role or career direction.</p>
    </div>
  </div>
</template>
