<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { TOAST_DURATION_MS, toast } from '../lib/toast'

const { t } = useI18n()

const icon = { success: '✓', info: 'i', warning: '!' }
const timerStyle = { '--toast-timer-duration': `${TOAST_DURATION_MS}ms` }
</script>

<template>
  <div class="toast-viewport" aria-live="polite" aria-atomic="false">
    <TransitionGroup name="toast-list">
      <div v-for="item in toast.messages" :key="item.id" class="toast" :class="item.tone" role="status">
        <span class="toast-icon" aria-hidden="true">{{ icon[item.tone] }}</span>
        <p>{{ item.message }}</p>
        <button type="button" :aria-label="t('toast.dismiss')" @click="toast.dismiss(item.id)">×</button>
        <span class="toast-timer" :style="timerStyle" aria-hidden="true" />
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-viewport { position: fixed; z-index: 100; top: 22px; right: 22px; display: grid; width: min(390px, calc(100vw - 32px)); gap: 10px; pointer-events: none; }
.toast { position: relative; display: grid; grid-template-columns: 28px minmax(0, 1fr) 28px; align-items: center; gap: 11px; min-height: 64px; overflow: hidden; padding: 13px 12px 13px 14px; border: 1px solid var(--line); border-radius: 13px; background: color-mix(in srgb, var(--surface) 96%, transparent); box-shadow: 0 18px 48px rgba(20, 34, 28, .2); pointer-events: auto; backdrop-filter: blur(14px); }
.toast-icon { display: grid; width: 28px; height: 28px; place-items: center; border-radius: 50%; background: var(--accent-pale); color: var(--accent-dark); font-size: 13px; font-weight: 900; }
.toast.info .toast-icon { background: var(--surface-soft); color: var(--accent); }
.toast.warning .toast-icon { background: #fff0cf; color: #825f19; }
.toast p { margin: 0; color: var(--ink); font-size: 13px; font-weight: 650; line-height: 1.45; }
.toast button { display: grid; width: 28px; height: 28px; padding: 0; place-items: center; border: 0; border-radius: 50%; background: transparent; color: var(--muted); cursor: pointer; font-size: 20px; line-height: 1; transition: background .18s ease, color .18s ease, transform .18s ease; }
.toast button:hover { background: var(--surface-soft); color: var(--ink); transform: rotate(6deg); }
.toast-timer { position: absolute; right: 0; bottom: 0; left: 0; height: 3px; background: var(--accent); transform-origin: left; animation: toast-timer var(--toast-timer-duration, 5s) linear forwards; }
.toast.warning .toast-timer { background: #b6862e; }
.toast-list-enter-active { transition: opacity .28s ease, transform .34s cubic-bezier(.2, .9, .25, 1.15); }
.toast-list-leave-active { position: absolute; right: 0; left: 0; transition: opacity .2s ease, transform .24s ease; }
.toast-list-move { transition: transform .28s ease; }
.toast-list-enter-from { opacity: 0; transform: translateX(42px) scale(.96); }
.toast-list-leave-to { opacity: 0; transform: translateX(34px) scale(.96); }
@keyframes toast-timer { to { transform: scaleX(0); } }
@media (max-width: 600px) { .toast-viewport { top: 12px; right: 16px; } }
@media (prefers-reduced-motion: reduce) { .toast-list-enter-active, .toast-list-leave-active, .toast-list-move { transition-duration: .01ms; } .toast-timer { animation: none; } }
</style>
