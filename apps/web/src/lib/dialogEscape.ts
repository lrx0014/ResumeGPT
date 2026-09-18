import { onBeforeUnmount, onMounted } from 'vue'

export function useEscapeToClose(isOpen: () => boolean, onEscape: () => void) {
  function handleKeydown(event: KeyboardEvent) {
    if (isOpen() && event.key === 'Escape') onEscape()
  }

  onMounted(() => window.addEventListener('keydown', handleKeydown))
  onBeforeUnmount(() => window.removeEventListener('keydown', handleKeydown))
}
