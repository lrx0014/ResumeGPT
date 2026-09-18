import { onBeforeUnmount, onMounted, ref } from 'vue'

export function usePdfObjectUrl(fetchBlob: () => Promise<Blob>, errorFallback: () => string) {
  const loading = ref(true)
  const error = ref('')
  const previewUrl = ref('')

  function release() {
    if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = ''
  }

  async function load() {
    loading.value = true
    error.value = ''
    release()
    try {
      previewUrl.value = URL.createObjectURL(await fetchBlob())
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : errorFallback()
    } finally {
      loading.value = false
    }
  }

  onMounted(load)
  onBeforeUnmount(release)

  return { loading, error, previewUrl, load }
}
