<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { api } from '../lib/api'

const props = defineProps<{ generationId: string; stepId?: string }>()
const loading = ref(true), error = ref(''), previewUrl = ref('')
function release(){if(previewUrl.value)URL.revokeObjectURL(previewUrl.value);previewUrl.value=''}
async function load(){loading.value=true;error.value='';release();try{previewUrl.value=URL.createObjectURL(await api.generationPreview(props.generationId,props.stepId))}catch(cause){error.value=cause instanceof Error?cause.message:'Could not load this PDF.'}finally{loading.value=false}}
watch(()=>[props.generationId,props.stepId],load)
onMounted(load);onBeforeUnmount(release)
</script>
<template><div class="generation-pdf"><div v-if="loading" class="preview-state">Loading PDF…</div><div v-else-if="error" class="preview-state error-text">{{ error }}</div><object v-else :data="previewUrl" type="application/pdf"><a :href="previewUrl" target="_blank">Open PDF</a></object></div></template>
<style scoped>.generation-pdf{height:100%;min-height:620px;overflow:hidden;border:1px solid var(--line);border-radius:12px;background:#525956}.generation-pdf object{display:block;width:100%;height:100%;min-height:0}.preview-state{display:grid;height:100%;min-height:300px;place-items:center;background:var(--surface-soft);color:var(--muted)}.error-text{color:var(--danger)}@media(max-width:760px){.generation-pdf{min-height:480px}}</style>
