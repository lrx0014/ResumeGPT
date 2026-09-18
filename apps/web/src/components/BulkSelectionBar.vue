<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
defineProps<{
  selectedCount: number
  allCount: number
  allSelected: boolean
}>()

defineEmits<{
  selectAll: []
  clear: []
}>()
</script>

<template>
  <Transition name="bulk-bar">
    <section v-if="selectedCount" class="bulk-selection-bar" aria-live="polite">
      <div class="selection-summary">
        <span class="selection-count">{{ selectedCount }}</span>
        <div><strong>{{ selectedCount }} selected</strong><small>{{ t('common.selectionHelp') }}</small></div>
      </div>
      <div class="selection-actions">
        <button v-if="allCount && !allSelected" class="text-button" type="button" @click="$emit('selectAll')">{{ t('common.selectAll') }}</button>
        <slot />
        <button class="text-button" type="button" @click="$emit('clear')">{{ t('common.clearSelection') }}</button>
      </div>
    </section>
  </Transition>
</template>

<style scoped>
.bulk-selection-bar { position: sticky; z-index: 8; top: 14px; display: flex; align-items: center; justify-content: space-between; gap: 18px; padding: 14px 18px; border: 1px solid #b9d2c4; border-radius: 12px; background: color-mix(in srgb, var(--accent-pale) 88%, white); box-shadow: 0 14px 34px rgba(35, 69, 54, .14); }
.selection-summary, .selection-actions { display: flex; align-items: center; gap: 12px; }
.selection-count { display: grid; width: 34px; height: 34px; place-items: center; border-radius: 50%; background: var(--accent); color: white; font-size: 13px; font-weight: 800; }
.selection-summary strong, .selection-summary small { display: block; }
.selection-summary small { margin-top: 2px; color: var(--muted); font-size: 11px; }
.selection-actions { flex-wrap: wrap; justify-content: flex-end; }
.bulk-bar-enter-active, .bulk-bar-leave-active { transition: opacity .18s ease, transform .18s ease; }
.bulk-bar-enter-from, .bulk-bar-leave-to { opacity: 0; transform: translateY(-8px); }
@media (max-width: 760px) { .bulk-selection-bar, .selection-actions { align-items: stretch; flex-direction: column; } .selection-actions { width: 100%; } .selection-actions :deep(.button), .selection-actions .text-button { width: 100%; } }
</style>
