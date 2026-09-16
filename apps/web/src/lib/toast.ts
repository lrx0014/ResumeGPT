import { reactive } from 'vue'

export type ToastTone = 'success' | 'info' | 'warning'

export interface ToastMessage {
  id: number
  message: string
  tone: ToastTone
}

const messages = reactive<ToastMessage[]>([])
let nextId = 1

function dismiss(id: number) {
  const index = messages.findIndex(message => message.id === id)
  if (index >= 0) messages.splice(index, 1)
}

function show(message: string, tone: ToastTone = 'success') {
  const id = nextId++
  messages.push({ id, message, tone })
  window.setTimeout(() => dismiss(id), 5000)
  return id
}

export const toast = {
  messages,
  dismiss,
  success: (message: string) => show(message, 'success'),
  info: (message: string) => show(message, 'info'),
  warning: (message: string) => show(message, 'warning'),
}
