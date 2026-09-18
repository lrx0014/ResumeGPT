<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'

import ConfirmDialog from '../components/ConfirmDialog.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../lib/api'
import { formatDateTime } from '../lib/formatDate'
import { toast } from '../lib/toast'
import type { JobHunter, JobHunterReviewItem } from '../lib/types'

const { t } = useI18n()
const route = useRoute()
const hunterId = computed(() => String(route.params.hunterId || ''))
const hunter = ref<JobHunter | null>(null)
const items = ref<JobHunterReviewItem[]>([])
const loading = ref(true)
const error = ref('')
const pendingDismiss = ref<JobHunterReviewItem | null>(null)
const busy = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [hunterResult, reviewResult] = await Promise.all([
      api.getJobHunter(hunterId.value),
      api.listJobHunterReviewItems(hunterId.value),
    ])
    hunter.value = hunterResult
    items.value = reviewResult.items
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('hunters.review.loadFailed')
  } finally {
    loading.value = false
  }
}

async function dismiss() {
  const item = pendingDismiss.value
  if (!item) return
  busy.value = true
  try {
    await api.dismissJobHunterReviewItem(hunterId.value, item.id)
    items.value = items.value.filter(candidate => candidate.id !== item.id)
    pendingDismiss.value = null
    toast.success(t('hunters.review.dismissedToast'))
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('hunters.review.dismissFailed')
  } finally {
    busy.value = false
  }
}

function formatTime(value: string) {
  return formatDateTime(value)
}

function hostname(value: string) {
  try { return new URL(value).hostname }
  catch { return t('hunters.review.jobPage') }
}

onMounted(load)
</script>

<template>
  <div class="page review-page">
    <PageHeader :title="hunter ? t('hunters.review.titleWithName', { name: hunter.name }) : t('hunters.review.title')" :description="t('hunters.review.description')">
      <RouterLink class="button" to="/job-hunters">{{ t('hunters.review.backToHunter') }}</RouterLink>
    </PageHeader>

    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <div v-if="loading" class="empty-state">{{ t('hunters.review.loading') }}</div>
    <div v-else-if="items.length" class="review-list">
      <article v-for="item in items" :key="item.id" class="panel review-item">
        <div class="review-copy">
          <div class="review-heading"><span class="warning-mark">!</span><div><p class="eyebrow">{{ t('hunters.review.needsConfirmation') }}</p><h2>{{ hostname(item.sourceUrl) }}</h2></div></div>
          <a class="job-url" :href="item.sourceUrl" target="_blank" rel="noopener noreferrer">{{ item.sourceUrl }}</a>
          <p>{{ item.failureMessage || t('hunters.review.defaultFailureMessage') }}</p>
          <small>{{ t('hunters.review.foundAt', { time: formatTime(item.createdAt) }) }} · {{ item.failureCode.replaceAll('_', ' ') }}</small>
        </div>
        <div class="review-actions">
          <a class="button primary" :href="item.sourceUrl" target="_blank" rel="noopener noreferrer">{{ t('hunters.review.openSource') }}</a>
          <RouterLink class="button" :to="{ path: '/jobs', query: { manualUrl: item.sourceUrl, hunterId: item.hunterId, reviewId: item.id } }">{{ t('hunters.review.addManually') }}</RouterLink>
          <button class="text-button danger-text" type="button" @click="pendingDismiss = item">{{ t('hunters.review.dismiss') }}</button>
        </div>
      </article>
    </div>
    <div v-else class="empty-state"><span class="empty-icon">✓</span><h2>{{ t('hunters.review.emptyTitle') }}</h2><p>{{ t('hunters.review.emptyDescription') }}</p><RouterLink class="button primary" to="/job-hunters">{{ t('hunters.review.backToHunter') }}</RouterLink></div>

    <ConfirmDialog :open="Boolean(pendingDismiss)" :title="t('hunters.review.dismissDialogTitle')" :message="t('hunters.review.dismissDialogMessage')" :busy="busy" @cancel="pendingDismiss = null" @confirm="dismiss" />
  </div>
</template>

<style scoped>
.review-page { display: grid; gap: 1.25rem; }
.review-page :deep(.page-header) { margin-bottom: .5rem; }
.review-list { display: grid; gap: 12px; }
.review-item { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: center; gap: 24px; }
.review-copy { display: grid; min-width: 0; gap: 9px; }
.review-heading { display: flex; align-items: center; gap: 10px; }
.review-heading h2, .review-heading .eyebrow, .review-copy p { margin: 0; }
.warning-mark { display: grid; width: 28px; height: 28px; place-items: center; flex: 0 0 auto; border-radius: 50%; background: #fff2cf; color: #795b16; font-weight: 800; }
.job-url { min-width: 0; overflow: hidden; color: var(--accent); font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.review-copy p { color: var(--muted); font-size: 11px; line-height: 1.5; }
.review-copy small { color: var(--muted); font-size: 9px; text-transform: capitalize; }
.review-actions { display: flex; align-items: center; justify-content: flex-end; gap: 8px; }
@media (max-width: 760px) { .review-item { grid-template-columns: 1fr; } .review-actions { justify-content: flex-start; flex-wrap: wrap; } }
</style>
