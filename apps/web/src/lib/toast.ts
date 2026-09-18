import { reactive } from 'vue'

export type ToastTone = 'success' | 'info' | 'warning'

export interface ToastMessage {
  id: number
  message: string
  tone: ToastTone
}

export const TOAST_DURATION_MS = 5000

const messages = reactive<ToastMessage[]>([])
let nextId = 1

function dismiss(id: number) {
  const index = messages.findIndex(message => message.id === id)
  if (index >= 0) messages.splice(index, 1)
}

function show(message: string, tone: ToastTone = 'success') {
  const id = nextId++
  messages.push({ id, message, tone })
  window.setTimeout(() => dismiss(id), TOAST_DURATION_MS)
  return id
}

export const toast = {
  messages,
  dismiss,
  success: (message: string) => show(message, 'success'),
  info: (message: string) => show(message, 'info'),
  warning: (message: string) => show(message, 'warning'),
}
