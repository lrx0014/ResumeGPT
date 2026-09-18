<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

const { t } = useI18n()

import ConfirmDialog from '../components/ConfirmDialog.vue'
import ListFilters from '../components/ListFilters.vue'
import ListPagination from '../components/ListPagination.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { usePagedList } from '../lib/pagination'
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
const form = reactive({ name: '', targetRole: '', defaultLanguage: 'English', content: '', avatarObjectId: '' })

const filteredProfiles = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  return profiles.value.filter(profile => !query || [profile.name, profile.targetRole, profile.contentPreview].some(value => value?.toLocaleLowerCase().includes(query)))
})
const { page, pageSize, visible: visibleProfiles } = usePagedList(filteredProfiles, [search])

async function load() {
  loading.value = true
  error.value = ''
  try {
    profiles.value = (await api.listProfiles()).items
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('profiles.errors.load')
  } finally {
    loading.value = false
  }
}

async function createProfile() {
  saving.value = true
  error.value = ''
  try {
    const created = await api.createProfile(form)
    toast.success(t('profiles.toasts.created'))
    await router.push(`/profiles/${created.id}`)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('profiles.errors.create')
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
    toast.success(t('profiles.toasts.deleted'))
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('profiles.errors.delete')
  } finally {
    deletingId.value = ''
  }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader :title="t('pages.profiles.title')" :description="t('pages.profiles.description')">
      <button class="button primary" type="button" @click="showForm = !showForm">
        {{ showForm ? t('common.close') : t('profiles.newProfile') }}
      </button>
    </PageHeader>

    <form v-if="showForm" class="panel form-grid" @submit.prevent="createProfile">
      <label><span>{{ t('profiles.form.nameLabel') }}</span><input v-model="form.name" required :placeholder="t('profiles.form.namePlaceholder')" /></label>
      <label><span>{{ t('profiles.form.targetRoleLabel') }}</span><input v-model="form.targetRole" :placeholder="t('profiles.form.targetRolePlaceholder')" /></label>
      <label>
        <span>{{ t('profiles.form.languageLabel') }}</span>
        <input v-model="form.defaultLanguage" required :placeholder="t('profiles.form.languagePlaceholder')" />
      </label>
      <div class="full form-actions"><button class="button primary" :disabled="saving">{{ saving ? t('common.saving') : t('profiles.form.createButton') }}</button></div>
    </form>

    <p v-if="error" class="notice error">{{ error }}</p>
    <div v-if="loading" class="empty-state">{{ t('profiles.loading') }}</div>
    <template v-else-if="profiles.length">
      <ListFilters v-model:search="search" :total="filteredProfiles.length" :search-placeholder="t('profiles.searchPlaceholder')" />
    <TransitionGroup v-if="visibleProfiles.length" name="card-list" tag="div" class="card-grid">
      <article v-for="profile in visibleProfiles" :key="profile.id" class="entity-card">
        <span class="entity-icon">◎</span>
        <div><p class="eyebrow">{{ profile.targetRole || t('profiles.generalProfile') }}</p><h2>{{ profile.name }}</h2><p>{{ profile.hasContent ? `${profile.contentPreview || ''}${(profile.contentPreview?.length || 0) >= 140 ? '…' : ''}` : t('profiles.contentPlaceholder') }}</p></div>
        <footer><span>{{ profile.defaultLanguage }}</span><div class="card-actions"><button class="text-button danger-text" type="button" @click="pendingDelete = profile">{{ t('common.delete') }}</button><RouterLink class="text-button" :to="`/profiles/${profile.id}`">{{ t('profiles.openProfile') }}</RouterLink></div></footer>
      </article>
    </TransitionGroup>
    <div v-else class="empty-state compact"><h2>{{ t('profiles.emptyFiltered.title') }}</h2><p>{{ t('profiles.emptyFiltered.message') }}</p></div>
    <ListPagination v-if="filteredProfiles.length" v-model:page="page" v-model:page-size="pageSize" :total="filteredProfiles.length" />
    </template>
    <div v-else class="empty-state">
      <span class="empty-icon">◎</span><h2>{{ t('profiles.emptyState.title') }}</h2><p>{{ t('profiles.emptyState.message') }}</p>
    </div>
    <ConfirmDialog :open="Boolean(pendingDelete)" :title="t('profiles.deleteConfirmTitle')" :message="t('profiles.deleteConfirmMessage', { name: pendingDelete?.name ?? '' })" :busy="deletingId === pendingDelete?.id" @cancel="pendingDelete = null" @confirm="removeProfile" />
  </div>
</template>
