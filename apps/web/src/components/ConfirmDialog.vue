<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { useEscapeToClose } from '../lib/dialogEscape'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  open: boolean
  title: string
  message: string
  confirmLabel?: string
  busy?: boolean
}>(), { confirmLabel: 'Delete', busy: false })

const emit = defineEmits<{ confirm: []; cancel: [] }>()

function cancel() {
  if (!props.busy) emit('cancel')
}

useEscapeToClose(() => props.open, cancel)
</script>

<template>
  <Teleport to="body">
    <Transition name="confirm-dialog">
      <div v-if="open" class="confirm-backdrop" @click.self="cancel">
        <section class="confirm-panel" role="alertdialog" aria-modal="true" aria-labelledby="confirm-title" aria-describedby="confirm-message">
          <span class="confirm-icon" aria-hidden="true">!</span>
          <div>
            <p class="eyebrow">{{ t('common.confirmDeletion') }}</p>
            <h2 id="confirm-title">{{ title }}</h2>
            <p id="confirm-message">{{ message }}</p>
          </div>
          <div class="confirm-actions">
            <button class="button" type="button" :disabled="busy" autofocus @click="cancel">{{ t('common.cancel') }}</button>
            <button class="button danger" type="button" :disabled="busy" @click="emit('confirm')">{{ busy ? t('common.deleting') : confirmLabel }}</button>
          </div>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.confirm-backdrop { position: fixed; z-index: 40; inset: 0; display: grid; place-items: center; padding: 24px; background: rgba(11, 20, 16, .58); backdrop-filter: blur(4px); }
.confirm-panel { display: grid; grid-template-columns: auto 1fr; gap: 16px; width: min(440px, 100%); padding: 26px; border: 1px solid var(--line); border-radius: 16px; background: var(--surface); box-shadow: 0 28px 80px rgba(0, 0, 0, .28); }
.confirm-icon { display: grid; width: 42px; height: 42px; place-items: center; border-radius: 50%; background: #f8e6e6; color: var(--danger); font-size: 20px; font-weight: 800; }
.confirm-panel h2 { margin: 2px 0 8px; font-size: 21px; }
.confirm-panel p:not(.eyebrow) { margin: 0; color: var(--muted); line-height: 1.55; }
.confirm-actions { grid-column: 1 / -1; display: flex; justify-content: flex-end; gap: 10px; padding-top: 8px; }
.confirm-dialog-enter-active, .confirm-dialog-leave-active { transition: opacity .18s ease; }
.confirm-dialog-enter-active .confirm-panel, .confirm-dialog-leave-active .confirm-panel { transition: transform .2s ease, opacity .18s ease; }
.confirm-dialog-enter-from, .confirm-dialog-leave-to { opacity: 0; }
.confirm-dialog-enter-from .confirm-panel, .confirm-dialog-leave-to .confirm-panel { opacity: 0; transform: translateY(10px) scale(.97); }
</style>
