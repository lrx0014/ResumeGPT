<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import type { FactReview, FactVersion, KnowledgeFact, KnowledgeSnapshot, Profile } from '../lib/types'

const route = useRoute()
const profileId = computed(() => String(route.params.profileId))
const profile = ref<Profile>()
const snapshot = ref<KnowledgeSnapshot>()
const loading = ref(true)
const busy = ref(false)
const enabled = ref(false)
const error = ref('')
const notice = ref('')
const input = reactive({ name: '', text: '' })
const edits = reactive<Record<string, FactReview>>({})
const statusFilter = ref('all')
const query = ref('')
const results = ref<FactVersion[] | null>(null)
let loadNumber = 0

function current(fact: KnowledgeFact) { return fact.versions.find(v => v.id === fact.currentVersionId)! }
const visibleFacts = computed(() => (snapshot.value?.facts ?? []).filter(f => statusFilter.value === 'all' || current(f).status === statusFilter.value))
function evidence(fact: KnowledgeFact) { return snapshot.value?.segments.find(s => s.id === fact.evidenceSegmentIds[0]) }
function sourceName(sourceId?: string) { return snapshot.value?.sources.find(s => s.id === sourceId)?.name ?? 'Source unavailable' }

async function load() {
  const number = ++loadNumber
  const id = profileId.value
  loading.value = true
  error.value = ''
  try {
    const [selected, capabilities] = await Promise.all([api.getProfile(id), api.capabilities()])
    const data = capabilities.features.knowledge ? await api.knowledge(id) : undefined
    if (number !== loadNumber) return
    profile.value = selected
    enabled.value = !!capabilities.features.knowledge
    snapshot.value = data
    for (const key of Object.keys(edits)) delete edits[key]
    for (const fact of data?.facts ?? []) {
      const version = current(fact)
      edits[fact.id] = { expectedVersionId: version.id, statement: version.statement, status: 'user_asserted', sensitive: version.sensitive }
    }
  } catch (cause) {
    if (number === loadNumber) error.value = cause instanceof Error ? cause.message : 'Could not load profile knowledge.'
  } finally { if (number === loadNumber) loading.value = false }
}

async function run(operation: () => Promise<unknown>, message: string) {
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    await operation()
    results.value = null
    await load()
    if (!error.value) notice.value = message
  } catch (cause) { error.value = cause instanceof Error ? cause.message : 'Could not complete the operation.' }
  finally { busy.value = false }
}

async function importText() {
  await run(async () => {
    await api.importText(profileId.value, { ...input })
    input.name = ''; input.text = ''
  }, 'Source imported. Review each candidate before using it. Identical content is deduplicated.')
}

async function upload(event: Event) {
  const element = event.target as HTMLInputElement
  const file = element.files?.[0]
  if (!file) return
  if (file.size > 65536) { error.value = 'Choose a UTF-8 TXT file no larger than 64 KiB.'; element.value = ''; return }
  await run(() => api.uploadText(profileId.value, file), 'Text file imported. All candidates require review.')
  element.value = ''
}

async function review(fact: KnowledgeFact, status: FactReview['status']) {
  const edit = edits[fact.id]
  if (!edit) return
  await run(() => api.reviewFact(profileId.value, fact.id, { ...edit, status }), 'A new fact version was saved. Earlier versions remain in history.')
}

async function removeSource(sourceId: string) {
  if (!window.confirm('Permanently delete this source, its evidence, all derived statements, and their version history from the active database? Existing downloads and backups are not erased.')) return
  await run(() => api.deleteSource(profileId.value, sourceId), 'Source and its derived knowledge deleted from the active database.')
}

async function exportData() {
  busy.value = true
  error.value = ''
  try {
    const data = await api.exportKnowledge(profileId.value)
    const url = URL.createObjectURL(new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' }))
    const link = document.createElement('a')
    link.href = url; link.download = 'profile-knowledge.json'; link.click()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  } catch (cause) { error.value = cause instanceof Error ? cause.message : 'Could not export knowledge.' }
  finally { busy.value = false }
}

async function search() {
  busy.value = true; error.value = ''
  try { results.value = (await api.searchKnowledge(profileId.value, query.value)).items }
  catch (cause) { error.value = cause instanceof Error ? cause.message : 'Could not search knowledge.' }
  finally { busy.value = false }
}

watch(profileId, () => { snapshot.value = undefined; results.value = null; notice.value = ''; void load() }, { immediate: true })
</script>

<template>
  <div class="page knowledge-page">
    <RouterLink to="/profiles">← All profiles</RouterLink>
    <PageHeader :title="profile?.name ?? 'Profile knowledge'" description="Keep original evidence, review candidate statements, and preserve every revision.">
      <button class="button" :disabled="busy || loading" @click="load">Reload</button>
      <button v-if="enabled" class="button" :disabled="busy || loading" @click="exportData">Export JSON</button>
    </PageHeader>
    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <p v-if="notice" class="notice" role="status">{{ notice }}</p>
    <p v-if="loading" role="status">Loading profile knowledge…</p>
    <p v-else-if="!enabled" class="notice">Knowledge storage requires PostgreSQL mode. Start the local database, apply migrations, and restart the API with PERSISTENCE_MODE=postgres.</p>
    <template v-else-if="snapshot">
      <section class="panel">
        <h2>Add background material</h2>
        <p>Paste text or upload a UTF-8 TXT file. Maximum 64 KiB, 200 non-empty lines and 4,000 bytes per line. Each line becomes an unverified review candidate. PDF, Word, TeX and image import are planned.</p>
        <form class="form-grid" @submit.prevent="importText">
          <label class="full"><span>Source name</span><input v-model="input.name" required maxlength="200" placeholder="Career history" /></label>
          <label class="full"><span>Background text</span><textarea v-model="input.text" required rows="6" placeholder="Describe your experience, education and achievements. Use one statement per line." /></label>
          <div class="full form-actions"><button class="button primary" :disabled="busy || loading">Import text</button></div>
        </form>
        <label><span>Or upload a TXT file</span><input type="file" accept=".txt,text/plain" :disabled="busy || loading" @change="upload" /></label>
      </section>
      <section class="panel">
        <h2>Sources ({{ snapshot.sources.length }})</h2>
        <p v-if="!snapshot.sources.length">No source material yet. Add your first source above.</p>
        <article v-for="source in snapshot.sources" :key="source.id" class="knowledge-item">
          <h3>{{ source.name }}</h3><p>{{ source.state }} · {{ source.mediaType }} · {{ new Date(source.createdAt).toLocaleString() }}</p>
          <details><summary>Original text and provenance</summary><pre>{{ source.text }}</pre><p>SHA-256: <code>{{ source.hash }}</code></p><p>Parser: {{ source.parserVersion }}</p></details>
          <button class="button" :disabled="busy || loading" @click="removeSource(source.id)">Delete source and derived facts</button>
        </article>
      </section>
      <section class="panel">
        <h2>Review statements ({{ snapshot.facts.length }})</h2>
        <p>Confirmation means you attest that the statement is accurate; it is not independent verification. All imports start as sensitive. Only confirmed, non-sensitive current versions appear in search. Reloading or saving discards other unsaved edits.</p>
        <label><span>Filter by status</span><select v-model="statusFilter"><option value="all">All statements</option><option value="extracted">Needs review</option><option value="user_asserted">Edited, awaiting confirmation</option><option value="user_confirmed">Confirmed</option><option value="disputed">Disputed</option><option value="rejected">Rejected</option></select></label>
        <p v-if="!visibleFacts.length">No statements match this filter.</p>
        <article v-for="fact in visibleFacts" :key="fact.id" class="knowledge-item">
          <p><strong>{{ current(fact).status }}</strong> · Version {{ current(fact).number }} · {{ current(fact).sensitive ? 'Sensitive' : 'Non-sensitive' }}</p>
          <details><summary>{{ sourceName(evidence(fact)?.sourceId) }} · {{ evidence(fact)?.page ? `Page ${evidence(fact)?.page}, ` : '' }}line {{ evidence(fact)?.paragraph }} · {{ Math.round((evidence(fact)?.confidence ?? 1) * 100) }}% extraction confidence</summary><blockquote>{{ evidence(fact)?.text }}</blockquote></details>
          <template v-if="edits[fact.id]">
            <label><span>Statement</span><textarea v-model="edits[fact.id]!.statement" rows="3" /></label>
            <label class="sensitivity"><input v-model="edits[fact.id]!.sensitive" type="checkbox" />Sensitive: exclude from retrieval</label>
            <div class="review-actions">
              <button class="button" :disabled="busy || loading" @click="review(fact, 'user_asserted')">Save edit</button>
              <button class="button primary" :disabled="busy || loading" @click="review(fact, 'user_confirmed')">Confirm accuracy</button>
              <button class="button" :disabled="busy || loading" @click="review(fact, 'disputed')">Mark disputed</button>
              <button class="button" :disabled="busy || loading" @click="review(fact, 'rejected')">Reject</button>
            </div>
          </template>
          <details><summary>Version history ({{ fact.versions.length }})</summary><article v-for="version in fact.versions" :key="version.id"><p>Version {{ version.number }} · {{ version.status }} · {{ version.id === fact.currentVersionId ? 'Current' : 'Superseded' }} · {{ new Date(version.createdAt).toLocaleString() }}</p><blockquote>{{ version.statement }}</blockquote></article></details>
        </article>
      </section>
      <section class="panel">
        <h2>Find reviewed statements</h2><p>Keyword search across confirmed, non-sensitive current statements. All query terms must match.</p>
        <form class="review-actions" @submit.prevent="search"><label><span>Keywords</span><input v-model="query" required maxlength="200" placeholder="Go backend" /></label><button class="button" :disabled="busy || loading">Search</button></form>
        <p v-if="results?.length === 0">No eligible statements match these keywords.</p>
        <blockquote v-for="result in results" :key="result.id">{{ result.statement }}</blockquote>
      </section>
    </template>
  </div>
</template>

<style scoped>
.knowledge-page { display: grid; gap: 1.5rem; }
.knowledge-item { border-top: 1px solid #dce3e8; padding: 1.25rem 0; display: grid; gap: .75rem; }
.knowledge-item textarea { width: 100%; }
.review-actions { display: flex; gap: .75rem; flex-wrap: wrap; align-items: end; }
.sensitivity { display: flex; align-items: center; gap: .5rem; }
.sensitivity input { width: auto; }
pre, blockquote { white-space: pre-wrap; overflow-wrap: anywhere; margin: .75rem 0; }
code { overflow-wrap: anywhere; }
summary { cursor: pointer; }
</style>
