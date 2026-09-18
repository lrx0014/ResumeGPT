<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

withDefaults(defineProps<{
  message: string
  code?: string
  compact?: boolean
}>(), { code: '', compact: false })
</script>

<template>
  <div class="attention-notice" :class="{ compact }" role="alert">
    <span class="attention-icon" aria-hidden="true">!</span>
    <div><strong>{{ t('common.needsAttention') }}</strong><p>{{ message }}</p><small v-if="code">{{ t('attentionNotice.reference', { code: code.replaceAll('_', ' ') }) }}</small></div>
  </div>
</template>

<style scoped>
.attention-notice { display: grid; grid-template-columns: 34px minmax(0, 1fr); gap: 12px; padding: 15px 16px; border: 1px solid color-mix(in srgb, var(--danger) 32%, var(--line)); border-radius: 11px; background: color-mix(in srgb, #f8e6e6 72%, var(--surface)); color: var(--danger); }
.attention-icon { display: grid; width: 34px; height: 34px; place-items: center; border-radius: 50%; background: color-mix(in srgb, var(--danger) 14%, var(--surface)); font-weight: 900; }
.attention-notice strong { display: block; margin-bottom: 3px; font-size: 13px; }
.attention-notice p { margin: 0; color: var(--ink); font-size: 13px; line-height: 1.5; }
.attention-notice small { display: block; margin-top: 5px; color: var(--muted); font-size: 10px; text-transform: capitalize; }
.attention-notice.compact { grid-template-columns: 25px minmax(0, 1fr); gap: 8px; padding: 9px 10px; }
.attention-notice.compact .attention-icon { width: 25px; height: 25px; font-size: 11px; }
.attention-notice.compact strong { font-size: 11px; }
.attention-notice.compact p { font-size: 11px; line-height: 1.4; }
</style>
