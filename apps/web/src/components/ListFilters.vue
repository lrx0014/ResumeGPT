<script setup lang="ts">
withDefaults(defineProps<{
  search: string
  total: number
  searchPlaceholder?: string
}>(), { searchPlaceholder: 'Search…' })

const emit = defineEmits<{ 'update:search': [value: string] }>()
</script>

<template>
  <section class="list-filters" aria-label="List search and filters">
    <label class="search-field">
      <span aria-hidden="true">⌕</span>
      <input :value="search" type="search" :placeholder="searchPlaceholder" aria-label="Search list" @input="emit('update:search', ($event.target as HTMLInputElement).value)" />
    </label>
    <div class="filter-fields"><slot /></div>
    <span class="result-count">{{ total }} {{ total === 1 ? 'result' : 'results' }}</span>
  </section>
</template>

<style scoped>
.list-filters { display: flex; align-items: center; gap: 12px; margin-bottom: 18px; padding: 13px 15px; border: 1px solid color-mix(in srgb, var(--accent) 20%, var(--line)); border-radius: 13px; background: linear-gradient(135deg, var(--surface-soft), color-mix(in srgb, var(--accent-pale) 58%, var(--surface))); }
.search-field { position: relative; flex: 1 1 320px; max-width: 560px; }
.search-field > span { position: absolute; top: 50%; left: 13px; color: var(--accent); font-size: 18px; transform: translateY(-52%); pointer-events: none; }
.search-field input { padding-left: 39px; border-color: color-mix(in srgb, var(--accent) 18%, var(--line)); background: color-mix(in srgb, var(--surface) 94%, transparent); }
.filter-fields { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; }
.filter-fields :deep(label) { display: flex; align-items: center; gap: 7px; color: var(--muted); font-size: 12px; font-weight: 700; white-space: nowrap; }
.filter-fields :deep(select) { min-width: 145px; padding: 9px 32px 9px 11px; background: color-mix(in srgb, var(--surface) 94%, transparent); }
.result-count { padding-left: 3px; color: var(--accent-dark); font-size: 11px; font-weight: 800; white-space: nowrap; }
@media (max-width: 720px) { .list-filters { align-items: stretch; flex-direction: column; } .search-field { width: 100%; max-width: none; flex-basis: auto; } .filter-fields { justify-content: flex-start; } .result-count { align-self: flex-end; } }
</style>
