<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

import ConfirmDialog from '../components/ConfirmDialog.vue'
import ListFilters from '../components/ListFilters.vue'
import ListPagination from '../components/ListPagination.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { toast } from '../lib/toast'
import type { Profile } from '../lib/types'

const profiles = ref<Profile[]>([])
const router = useRouter()
const loading = ref(true)
const saving = ref(false)
const deletingId = ref('')
const pendingDelete = ref<Profile | null>(null)
const error = ref('')
const showForm = ref(false)
const search = ref('')
const languageFilter = ref('all')
const page = ref(1)
const pageSize = ref(8)
const form = reactive({ name: '', targetRole: '', defaultLanguage: 'en-US', content: '', avatarObjectId: '' })

const filteredProfiles = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  return profiles.value.filter(profile => {
    const matchesSearch = !query || [profile.name, profile.targetRole, profile.content].some(value => value?.toLocaleLowerCase().includes(query))
    return matchesSearch && (languageFilter.value === 'all' || profile.defaultLanguage === languageFilter.value)
  })
})
const visibleProfiles = computed(() => filteredProfiles.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
watch([search, languageFilter, pageSize], () => { page.value = 1 })
watch(() => filteredProfiles.value.length, total => { page.value = Math.min(page.value, Math.max(1, Math.ceil(total / pageSize.value))) })

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
    toast.success('Profile created.')
    await router.push(`/profiles/${created.id}`)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not create the profile.'
  } finally {
    saving.value = false
  }
}

async function removeProfile() {
  const profile = pendingDelete.value
  if (!profile) return
  deletingId.value = profile.id
  error.value = ''
  try {
    await api.deleteProfile(profile.id)
    profiles.value = profiles.value.filter(candidate => candidate.id !== profile.id)
    pendingDelete.value = null
    toast.success('Profile deleted.')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not delete the profile.'
  } finally {
    deletingId.value = ''
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
    <template v-else-if="profiles.length">
      <ListFilters v-model:search="search" :total="filteredProfiles.length" search-placeholder="Search profiles, roles, or content…">
        <label>Language <select v-model="languageFilter"><option value="all">All languages</option><option value="en-US">English</option><option value="de-DE">German</option></select></label>
      </ListFilters>
    <TransitionGroup v-if="visibleProfiles.length" name="card-list" tag="div" class="card-grid">
      <article v-for="profile in visibleProfiles" :key="profile.id" class="entity-card">
        <span class="entity-icon">◎</span>
        <div><p class="eyebrow">{{ profile.targetRole || 'General profile' }}</p><h2>{{ profile.name }}</h2><p>{{ profile.content ? `${profile.content.slice(0, 140)}${profile.content.length > 140 ? '…' : ''}` : 'Add your experience, education, skills, and achievements.' }}</p></div>
        <footer><span>{{ profile.defaultLanguage }}</span><div class="card-actions"><button class="text-button danger-text" type="button" @click="pendingDelete = profile">Delete</button><RouterLink class="text-button" :to="`/profiles/${profile.id}`">Open →</RouterLink></div></footer>
      </article>
    </TransitionGroup>
    <div v-else class="empty-state compact"><h2>No matching profiles</h2><p>Try another keyword or language.</p></div>
    <ListPagination v-if="filteredProfiles.length" v-model:page="page" v-model:page-size="pageSize" :total="filteredProfiles.length" />
    </template>
    <div v-else class="empty-state">
      <span class="empty-icon">◎</span><h2>No profiles yet</h2><p>Create a profile to organize experience for a role or career direction.</p>
    </div>
    <ConfirmDialog :open="Boolean(pendingDelete)" title="Delete profile?" :message="`“${pendingDelete?.name ?? ''}” and its saved content will be removed. This action cannot be undone.`" :busy="deletingId === pendingDelete?.id" @cancel="pendingDelete = null" @confirm="removeProfile" />
  </div>
</template>
