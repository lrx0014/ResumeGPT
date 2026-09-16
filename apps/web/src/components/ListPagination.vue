<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  page: number
  pageSize: number
  total: number
  pageSizes?: number[]
}>(), { pageSizes: () => [8, 16, 32] })

const emit = defineEmits<{
  'update:page': [value: number]
  'update:pageSize': [value: number]
}>()

const pageCount = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
const firstItem = computed(() => props.total ? (props.page - 1) * props.pageSize + 1 : 0)
const lastItem = computed(() => Math.min(props.page * props.pageSize, props.total))

function updatePageSize(event: Event) {
  emit('update:pageSize', Number((event.target as HTMLSelectElement).value))
  emit('update:page', 1)
}
</script>

<template>
  <nav class="list-pagination" aria-label="List pagination">
    <span>{{ firstItem }}–{{ lastItem }} of {{ total }}</span>
    <div class="page-actions">
      <label>Per page <select :value="pageSize" aria-label="Items per page" @change="updatePageSize"><option v-for="size in pageSizes" :key="size" :value="size">{{ size }}</option></select></label>
      <button type="button" aria-label="Previous page" :disabled="page <= 1" @click="emit('update:page', page - 1)">←</button>
      <strong>{{ Math.min(page, pageCount) }} / {{ pageCount }}</strong>
      <button type="button" aria-label="Next page" :disabled="page >= pageCount" @click="emit('update:page', page + 1)">→</button>
    </div>
  </nav>
</template>

<style scoped>
.list-pagination, .page-actions { display: flex; align-items: center; gap: 10px; }
.list-pagination { justify-content: space-between; margin-top: 16px; padding: 10px 12px 10px 16px; border-radius: 11px; background: var(--surface-soft); color: var(--muted); font-size: 12px; }
.page-actions label { display: flex; align-items: center; gap: 7px; white-space: nowrap; }
.page-actions select { width: auto; padding: 7px 27px 7px 9px; background: var(--surface); }
.page-actions button { display: grid; width: 31px; height: 31px; padding: 0; place-items: center; border: 0; border-radius: 8px; background: var(--surface); color: var(--accent); cursor: pointer; box-shadow: 0 3px 10px rgba(29, 48, 40, .06); transition: background .18s ease, transform .18s ease; }
.page-actions button:hover:not(:disabled) { background: var(--accent-pale); transform: translateY(-1px); }
.page-actions button:disabled { cursor: not-allowed; opacity: .35; }
.page-actions strong { min-width: 52px; color: var(--ink); text-align: center; }
@media (max-width: 520px) { .list-pagination { align-items: stretch; flex-direction: column; } .page-actions { justify-content: space-between; } }
</style>
