<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

const { t } = useI18n()

import ConfirmDialog from '../components/ConfirmDialog.vue'
import ListFilters from '../components/ListFilters.vue'
import ListPagination from '../components/ListPagination.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { toast } from '../lib/toast'
import type { Profile } from '../lib/types'

const profiles = ref<Profile[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(8)
const router = useRouter()
const loading = ref(true)
const saving = ref(false)
const deletingId = ref('')
const pendingDelete = ref<Profile | null>(null)
const error = ref('')
const showForm = ref(false)
const search = ref('')
const form = reactive({ name: '', targetRole: '', defaultLanguage: 'English', content: '', avatarObjectId: '' })
let searchTimer: number | undefined

async function load() {
  loading.value = true
  error.value = ''
  try {
    const result = await api.searchProfiles({ search: search.value.trim(), page: page.value, pageSize: pageSize.value })
    if (!result.items.length && result.total > 0 && page.value > 1) {
      page.value = Math.max(1, Math.ceil(result.total / pageSize.value))
      return load()
    }
    profiles.value = result.items
    total.value = result.total
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('profiles.errors.load')
  } finally {
    loading.value = false
  }
}

function resetPageAndLoad() {
  if (page.value !== 1) page.value = 1
  else void load()
}
watch(pageSize, resetPageAndLoad)
watch(page, () => void load())
watch(search, () => {
  if (searchTimer) window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(resetPageAndLoad, 800)
})

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
    pendingDelete.value = null
    toast.success(t('profiles.toasts.deleted'))
    await load()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('profiles.errors.delete')
  } finally {
    deletingId.value = ''
  }
}

onMounted(load)
onBeforeUnmount(() => { if (searchTimer) window.clearTimeout(searchTimer) })
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
    <div v-else-if="!total && !search" class="empty-state">
      <span class="empty-icon">◎</span><h2>{{ t('profiles.emptyState.title') }}</h2><p>{{ t('profiles.emptyState.message') }}</p>
    </div>
    <template v-else>
      <ListFilters v-model:search="search" :total="total" :search-placeholder="t('profiles.searchPlaceholder')" />
    <TransitionGroup v-if="profiles.length" name="card-list" tag="div" class="card-grid">
      <article v-for="profile in profiles" :key="profile.id" class="entity-card">
        <span class="entity-icon">◎</span>
        <div><p class="eyebrow">{{ profile.targetRole || t('profiles.generalProfile') }}</p><h2>{{ profile.name }}</h2><p>{{ profile.hasContent ? `${profile.contentPreview || ''}${(profile.contentPreview?.length || 0) >= 140 ? '…' : ''}` : t('profiles.contentPlaceholder') }}</p></div>
        <footer><span>{{ profile.defaultLanguage }}</span><div class="card-actions"><button class="text-button danger-text" type="button" @click="pendingDelete = profile">{{ t('common.delete') }}</button><RouterLink class="text-button" :to="`/profiles/${profile.id}`">{{ t('profiles.openProfile') }}</RouterLink></div></footer>
      </article>
    </TransitionGroup>
    <div v-else class="empty-state compact"><h2>{{ t('profiles.emptyFiltered.title') }}</h2><p>{{ t('profiles.emptyFiltered.message') }}</p></div>
    <ListPagination v-if="total" v-model:page="page" v-model:page-size="pageSize" :total="total" />
    </template>
    <ConfirmDialog :open="Boolean(pendingDelete)" :title="t('profiles.deleteConfirmTitle')" :message="t('profiles.deleteConfirmMessage', { name: pendingDelete?.name ?? '' })" :busy="deletingId === pendingDelete?.id" @cancel="pendingDelete = null" @confirm="removeProfile" />
  </div>
</template>
