<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { useEscapeToClose } from '../lib/dialogEscape'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  open: boolean
  busy?: boolean
}>(), { busy: false })

const emit = defineEmits<{ save: []; discard: []; cancel: [] }>()

function cancel() {
  if (!props.busy) emit('cancel')
}

useEscapeToClose(() => props.open, cancel)
</script>

<template>
  <Teleport to="body">
    <Transition name="unsaved-dialog">
      <div v-if="open" class="unsaved-backdrop" @click.self="cancel">
        <section class="unsaved-panel" role="alertdialog" aria-modal="true" aria-labelledby="unsaved-title" aria-describedby="unsaved-message">
          <span class="unsaved-icon" aria-hidden="true">●</span>
          <div>
            <p class="eyebrow">{{ t('dialogs.unsaved') }}</p>
            <h2 id="unsaved-title">{{ t('dialogs.saveChanges') }}</h2>
            <p id="unsaved-message">{{ t('dialogs.message') }}</p>
          </div>
          <div class="unsaved-actions">
            <button class="button" type="button" :disabled="busy" @click="cancel">{{ t('dialogs.keepEditing') }}</button>
            <button class="text-button danger-text" type="button" :disabled="busy" @click="emit('discard')">{{ t('dialogs.discard') }}</button>
            <button class="button primary" type="button" :disabled="busy" @click="emit('save')">{{ busy ? t('common.saving') : t('common.save') }}</button>
          </div>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.unsaved-backdrop { position: fixed; z-index: 40; inset: 0; display: grid; place-items: center; padding: 24px; background: rgba(11, 20, 16, .58); backdrop-filter: blur(4px); }
.unsaved-panel { display: grid; grid-template-columns: auto 1fr; gap: 16px; width: min(500px, 100%); padding: 26px; border: 1px solid var(--line); border-radius: 16px; background: var(--surface); box-shadow: 0 28px 80px rgba(0, 0, 0, .28); }
.unsaved-icon { display: grid; width: 42px; height: 42px; place-items: center; border-radius: 50%; background: #fff2cf; color: #9a741d; font-size: 14px; }
.unsaved-panel h2 { margin: 2px 0 8px; font-size: 21px; }
.unsaved-panel p:not(.eyebrow) { margin: 0; color: var(--muted); line-height: 1.55; }
.unsaved-actions { grid-column: 1 / -1; display: flex; align-items: center; justify-content: flex-end; gap: 10px; padding-top: 8px; }
.danger-text { color: var(--danger); }
.unsaved-dialog-enter-active, .unsaved-dialog-leave-active { transition: opacity .18s ease; }
.unsaved-dialog-enter-active .unsaved-panel, .unsaved-dialog-leave-active .unsaved-panel { transition: transform .2s ease, opacity .18s ease; }
.unsaved-dialog-enter-from, .unsaved-dialog-leave-to { opacity: 0; }
.unsaved-dialog-enter-from .unsaved-panel, .unsaved-dialog-leave-to .unsaved-panel { opacity: 0; transform: translateY(10px) scale(.97); }
@media (max-width: 560px) { .unsaved-actions { align-items: stretch; flex-direction: column-reverse; } .unsaved-actions > * { width: 100%; } }
</style>
