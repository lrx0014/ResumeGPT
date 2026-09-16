import { computed, ref } from 'vue'

export function useListSelection() {
  const selectedIds = ref<Set<string>>(new Set())
  const selectedCount = computed(() => selectedIds.value.size)

  function isSelected(id: string) {
    return selectedIds.value.has(id)
  }

  function toggle(id: string, selected?: boolean) {
    const next = new Set(selectedIds.value)
    const shouldSelect = selected ?? !next.has(id)
    if (shouldSelect) next.add(id)
    else next.delete(id)
    selectedIds.value = next
  }

  function toggleMany(ids: string[], selected?: boolean) {
    const next = new Set(selectedIds.value)
    const shouldSelect = selected ?? !ids.every(id => next.has(id))
    for (const id of ids) {
      if (shouldSelect) next.add(id)
      else next.delete(id)
    }
    selectedIds.value = next
  }

  function clear() {
    selectedIds.value = new Set()
  }

  function retain(ids: string[]) {
    const available = new Set(ids)
    selectedIds.value = new Set([...selectedIds.value].filter(id => available.has(id)))
  }

  return { selectedIds, selectedCount, isSelected, toggle, toggleMany, clear, retain }
}
