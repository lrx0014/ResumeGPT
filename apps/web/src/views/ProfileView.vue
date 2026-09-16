<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import ConfirmDialog from '../components/ConfirmDialog.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { toast } from '../lib/toast'
import type { DocumentUpload, Profile } from '../lib/types'

const route = useRoute()
const router = useRouter()
const profileId = computed(() => String(route.params.profileId))
const profile = ref<Profile>()
const form = reactive({ name: '', targetRole: '', defaultLanguage: 'en-US', content: '', avatarObjectId: '' })
const loading = ref(true)
const saving = ref(false)
const extracting = ref(false)
const deleting = ref(false)
const confirmDelete = ref(false)
const error = ref('')
const extraction = ref<DocumentUpload>()
const avatarUrl = ref('')
let localAvatarUrl = ''

async function load() {
  loading.value = true
  error.value = ''
  try {
    const selected = await api.getProfile(profileId.value)
    profile.value = selected
    Object.assign(form, {
      name: selected.name,
      targetRole: selected.targetRole ?? '',
      defaultLanguage: selected.defaultLanguage,
      content: selected.content,
      avatarObjectId: selected.avatarObjectId ?? '',
    })
    avatarUrl.value = ''
    if (selected.avatarObjectId) {
      try {
        avatarUrl.value = (await api.profileAvatar(selected.id)).url
      } catch {
        toast.warning('The profile loaded, but its avatar is currently unavailable.')
      }
    }
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not load the profile.'
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  error.value = ''
  try {
    profile.value = await api.updateProfile(profileId.value, { ...form })
    toast.success('Profile saved.')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not save the profile.'
  } finally {
    saving.value = false
  }
}

async function extractDocument(event: Event) {
  const element = event.target as HTMLInputElement
  const file = element.files?.[0]
  if (!file) return
  if (file.size > 10 * 1024 * 1024) {
    error.value = 'Choose a document no larger than 10 MiB.'
    element.value = ''
    return
  }
  extracting.value = true
  error.value = ''
  try {
    const staged = await api.stageDocument(profileId.value, file)
    extraction.value = staged.upload
    await api.putStagedDocument(staged.target, file)
    extraction.value = await api.completeDocumentUpload(profileId.value, staged.upload.id)
    for (let attempt = 0; attempt < 240 && extraction.value.state === 'queued'; attempt += 1) {
      await new Promise(resolve => setTimeout(resolve, 1000))
      extraction.value = await api.documentUpload(profileId.value, staged.upload.id)
    }
    if (extraction.value.state !== 'ready' || !extraction.value.extractedText) {
      throw new Error(extraction.value.errorMessage || 'Document extraction did not complete.')
    }
    form.content = extraction.value.extractedText
    toast.success('Extracted text loaded into the editor. Review it and click Save profile when ready.')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not extract the document.'
  } finally {
    extracting.value = false
    element.value = ''
  }
}

async function uploadAvatar(event: Event) {
  const element = event.target as HTMLInputElement
  const file = element.files?.[0]
  if (!file) return
  if (!['image/jpeg', 'image/png'].includes(file.type) || file.size > 5 * 1024 * 1024) {
    error.value = 'Choose a JPEG or PNG image no larger than 5 MiB.'
    element.value = ''
    return
  }
  saving.value = true
  error.value = ''
  try {
    const target = await api.createAvatarUpload(profileId.value, file.type)
    await api.putStagedDocument(target, file)
    form.avatarObjectId = target.objectId
    if (localAvatarUrl) URL.revokeObjectURL(localAvatarUrl)
    localAvatarUrl = URL.createObjectURL(file)
    avatarUrl.value = localAvatarUrl
    toast.info('Avatar uploaded. Click Save profile to keep this change.')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not upload the avatar.'
  } finally {
    saving.value = false
    element.value = ''
  }
}

function removeAvatar() {
  form.avatarObjectId = ''
  avatarUrl.value = ''
  toast.info('Avatar removed from the editor. Click Save profile to keep this change.')
}

function handleAvatarError() {
  avatarUrl.value = ''
  toast.warning('The saved avatar is currently unavailable. Upload another image or remove the avatar and save.')
}

async function removeProfile() {
  deleting.value = true
  error.value = ''
  try {
    await api.deleteProfile(profileId.value)
    toast.success('Profile deleted.')
    await router.push('/profiles')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not delete the profile.'
    deleting.value = false
  }
}

watch(profileId, () => void load(), { immediate: true })
onBeforeUnmount(() => { if (localAvatarUrl) URL.revokeObjectURL(localAvatarUrl) })
</script>

<template>
  <div class="page profile-editor">
    <RouterLink to="/profiles">← All profiles</RouterLink>
    <PageHeader :title="profile?.name ?? 'Profile'" description="Edit the profile text that ResumeGPT will use for future generation.">
      <button class="button" type="button" :disabled="loading || saving || extracting" @click="load">Discard changes</button>
      <button class="button primary" type="button" :disabled="loading || saving || extracting" @click="save">{{ saving ? 'Saving…' : 'Save profile' }}</button>
    </PageHeader>

    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <div v-if="loading" class="empty-state">Loading profile…</div>

    <template v-else-if="profile">
      <section class="panel form-grid">
        <h2 class="full">Profile details</h2>
        <label><span>Name</span><input v-model="form.name" required maxlength="200" /></label>
        <label><span>Target role</span><input v-model="form.targetRole" maxlength="200" placeholder="Senior Backend Engineer" /></label>
        <label><span>Primary language</span><select v-model="form.defaultLanguage"><option value="en-US">English</option><option value="de-DE">German</option></select></label>
        <div class="avatar-field">
          <img v-if="avatarUrl" :src="avatarUrl" alt="Profile avatar preview" @error="handleAvatarError" />
          <div>
            <label><span>Optional avatar</span><input type="file" accept="image/jpeg,image/png" :disabled="saving" @change="uploadAvatar" /></label>
            <button v-if="avatarUrl" class="text-button" type="button" @click="removeAvatar">Remove avatar</button>
          </div>
        </div>
      </section>

      <section class="panel">
        <div class="editor-heading">
          <div><h2>Profile text</h2><p>Paste or write Markdown-style text. Include the experience, education, skills, projects, and achievements that may be used later.</p></div>
          <label class="file-action"><span>{{ extracting ? 'Extracting…' : 'Load from file' }}</span><input type="file" accept=".pdf,.doc,.docx,.tex,.md,.txt,.png,.jpg,.jpeg" :disabled="extracting || saving" @change="extractDocument" /></label>
        </div>
        <p v-if="extraction">{{ extraction.name }} · {{ extraction.state }}<template v-if="extraction.errorCode"> · {{ extraction.errorCode }}</template></p>
        <textarea v-model="form.content" rows="26" maxlength="1048576" placeholder="# Professional summary&#10;&#10;Write or paste your profile here…" />
        <p class="character-count">{{ form.content.length.toLocaleString() }} characters</p>
      </section>

      <section class="danger-zone">
        <div><h2>Delete profile</h2><p>This removes the profile from the active database.</p></div>
        <button class="button" type="button" :disabled="deleting" @click="confirmDelete = true">Delete profile</button>
      </section>
    </template>
    <ConfirmDialog :open="confirmDelete" title="Delete profile?" :message="`“${profile?.name ?? 'This profile'}” and its saved content will be removed. This action cannot be undone.`" :busy="deleting" @cancel="confirmDelete = false" @confirm="removeProfile" />
  </div>
</template>

<style scoped>
.profile-editor { display: grid; gap: 1.5rem; }
.panel { display: grid; gap: 1rem; }
.panel h2, .panel p { margin: 0; }
.avatar-field { display: flex; align-items: center; gap: 1rem; }
.avatar-field img { width: 76px; height: 76px; border-radius: 50%; object-fit: cover; border: 1px solid #dce3e8; }
.editor-heading { display: flex; align-items: end; justify-content: space-between; gap: 1rem; }
.file-action { cursor: pointer; }
.file-action span { display: inline-flex; padding: .65rem 1rem; border: 1px solid #c8d2d9; border-radius: .5rem; }
.file-action input { position: absolute; width: 1px; height: 1px; opacity: 0; }
textarea { width: 100%; min-height: 32rem; resize: vertical; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; line-height: 1.6; }
.character-count { text-align: right; color: #64727d; }
.danger-zone { display: flex; justify-content: space-between; align-items: center; gap: 1rem; padding: 1.25rem; border: 1px solid #e6b8b8; border-radius: .75rem; }
.danger-zone h2, .danger-zone p { margin: 0; }
@media (max-width: 720px) { .editor-heading, .danger-zone { align-items: stretch; flex-direction: column; } }
</style>
