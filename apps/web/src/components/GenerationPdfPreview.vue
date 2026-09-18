<script setup lang="ts">
import { watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../lib/api'
import { usePdfObjectUrl } from '../lib/pdfPreview'

const { t } = useI18n()
const props = defineProps<{ generationId: string; stepId?: string }>()
const { loading, error, previewUrl, load } = usePdfObjectUrl(
  () => api.generationPreview(props.generationId, props.stepId),
  () => t('generate.pdfPreview.error'),
)
watch(()=>[props.generationId,props.stepId],load)
</script>
<template><div class="generation-pdf"><div v-if="loading" class="preview-state">{{ t('generate.pdfPreview.loading') }}</div><div v-else-if="error" class="preview-state error-text">{{ error }}</div><object v-else :data="previewUrl" type="application/pdf"><a :href="previewUrl" target="_blank">{{ t('generate.pdfPreview.openPdf') }}</a></object></div></template>
<style scoped>.generation-pdf{height:100%;min-height:620px;overflow:hidden;border:1px solid var(--line);border-radius:12px;background:#525956}.generation-pdf object{display:block;width:100%;height:100%;min-height:0}.preview-state{display:grid;height:100%;min-height:300px;place-items:center;background:var(--surface-soft);color:var(--muted)}.error-text{color:var(--danger)}@media(max-width:760px){.generation-pdf{min-height:480px}}</style>
