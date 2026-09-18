import { computed, ref, watch, type ComputedRef, type WatchSource } from 'vue'

export function usePagedList<T>(filtered: ComputedRef<T[]>, resetTriggers: WatchSource[], initialPageSize = 8) {
  const page = ref(1)
  const pageSize = ref(initialPageSize)
  const visible = computed(() => filtered.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))

  watch([...resetTriggers, pageSize], () => { page.value = 1 })
  watch(() => filtered.value.length, total => {
    page.value = Math.min(page.value, Math.max(1, Math.ceil(total / pageSize.value)))
  })

  return { page, pageSize, visible }
}
