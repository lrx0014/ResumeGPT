<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

import { api } from '../lib/api'

const props = defineProps<{ templateId: string }>()
const loading = ref(true)
const error = ref('')
const previewUrl = ref('')

function releasePreview() {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  previewUrl.value = ''
}

async function loadPreview() {
  loading.value = true
  error.value = ''
  releasePreview()
  try {
    previewUrl.value = URL.createObjectURL(await api.templatePreview(props.templateId))
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Could not load the PDF preview.'
  } finally {
    loading.value = false
  }
}

onMounted(loadPreview)
onBeforeUnmount(releasePreview)
</script>

<template>
  <div class="pdf-preview">
    <div v-if="loading" class="preview-message"><span class="preview-spinner" aria-hidden="true" /><strong>Loading PDF preview…</strong><p>TeX and Word templates use the same stored PDF preview.</p></div>
    <div v-else-if="error" class="preview-message error-message"><strong>Preview unavailable</strong><p>{{ error }}</p><button class="button" type="button" @click="loadPreview">Try again</button></div>
    <object v-else :data="previewUrl" type="application/pdf" class="preview-frame" aria-label="Template PDF preview">
      <div class="preview-message"><strong>Your browser cannot display embedded PDFs.</strong><a class="button" :href="previewUrl" target="_blank" rel="noopener">Open preview</a></div>
    </object>
  </div>
</template>

<style scoped>
.pdf-preview { min-height: 680px; overflow: hidden; border: 1px solid var(--line); border-radius: 12px; background: #5c625f; }
.preview-frame { display: block; width: 100%; height: 76vh; min-height: 680px; border: 0; }
.preview-message { display: grid; min-height: 360px; place-items: center; align-content: center; gap: .7rem; padding: 2rem; background: var(--surface-soft); color: var(--muted); text-align: center; }
.preview-message strong { color: var(--ink); }
.preview-message p { max-width: 520px; margin: 0; line-height: 1.5; }
.preview-spinner { width: 30px; height: 30px; border: 3px solid var(--line); border-top-color: var(--accent); border-radius: 50%; animation: spin .8s linear infinite; }
.error-message { color: var(--danger); }
@keyframes spin { to { transform: rotate(360deg); } }
@media (max-width: 760px) { .pdf-preview, .preview-frame { min-height: 520px; } .preview-frame { height: 68vh; } }
</style>
