<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GenerationRun } from '../lib/types'

type WorkflowStatus = 'completed' | 'active' | 'failed' | 'waiting'

const { t } = useI18n()
const props = defineProps<{ run: GenerationRun }>()

const stages = [
  { key: 'writing' },
  { key: 'rendering' },
  { key: 'reviewing' },
  { key: 'repairing' },
  { key: 'finalizing' },
] as const

function inferredFailedIndex() {
  if (props.run.stage !== 'failed') {
    const index = stages.findIndex(stage => stage.key === props.run.stage)
    if (index >= 0) return index
  }
  const code = props.run.errorCode?.toLocaleLowerCase() ?? ''
  if (/(writer|writing|draft|snapshot|payload)/.test(code)) return 0
  if (/(renderer|rendering|latex|template|artifact|render)/.test(code)) return 1
  if (/(repair|quality|page_target)/.test(code)) return 3
  if (/(review|vision)/.test(code)) return 2
  if (/(complete|final|store)/.test(code)) return 4
  return Math.min(4, props.run.draft ? 1 : 0)
}

const currentIndex = computed(() => {
  if (props.run.state === 'ready') return stages.length
  if (props.run.state === 'failed') return inferredFailedIndex()
  const index = stages.findIndex(stage => stage.key === props.run.stage)
  return Math.max(0, index)
})

function computeStatus(index: number): WorkflowStatus {
  if (props.run.state === 'ready') return 'completed'
  if (props.run.state === 'failed') return index < currentIndex.value ? 'completed' : index === currentIndex.value ? 'failed' : 'waiting'
  if (props.run.state === 'queued') return 'waiting'
  return index < currentIndex.value ? 'completed' : index === currentIndex.value ? 'active' : 'waiting'
}

const statuses = computed(() => stages.map((_, index) => computeStatus(index)))

const icon = (value: WorkflowStatus) => ({ completed: '✓', active: '•', failed: '×', waiting: '·' })[value]
const statusLabel = (value: WorkflowStatus) => ({ completed: t('generate.workflow.status.completed'), active: t('generate.workflow.status.active'), failed: t('generate.workflow.status.failed'), waiting: t('generate.stage.waiting') })[value]
const stageLabel = (stage: (typeof stages)[number]) => stage.key === 'rendering' ? (props.run.templateId ? t('generate.stage.applyingTemplate') : t('generate.stage.designingDocument')) : ({ writing: t('generate.workflow.stage.writing'), reviewing: t('generate.workflow.stage.reviewing'), repairing: t('generate.workflow.stage.repairing'), finalizing: t('generate.stage.finalizing') } as Record<string, string>)[stage.key]
</script>

<template>
  <section class="workflow-panel" :aria-label="t('generate.workflow.title')">
    <div class="workflow-heading"><div><p class="eyebrow">{{ t('generate.workflow.eyebrow') }}</p><h2>{{ t('generate.workflow.title') }}</h2></div><span>{{ run.state === 'ready' ? t('generate.workflow.overallComplete') : run.state === 'failed' ? t('generate.workflow.actionRequired') : t('generate.workflow.processing') }}</span></div>
    <div class="workflow-track">
      <template v-for="(stage, index) in stages" :key="stage.key">
        <div v-if="index" class="workflow-connector" :class="{ completed: statuses[index] === 'completed' || statuses[index] === 'active' || statuses[index] === 'failed' }"><span /></div>
        <div class="workflow-stage" :class="statuses[index]" :tabindex="statuses[index] === 'failed' ? 0 : undefined">
          <span class="workflow-node">{{ icon(statuses[index]) }}</span>
          <strong>{{ stageLabel(stage) }}</strong>
          <small>{{ statusLabel(statuses[index]) }}</small>
          <div v-if="statuses[index] === 'failed'" class="failure-tooltip" role="tooltip">
            <strong>{{ t('generate.workflow.failureTitle') }}</strong>
            <p>{{ run.errorMessage || t('generate.workflow.failureFallback') }}</p>
            <small v-if="run.errorCode">{{ run.errorCode.replaceAll('_', ' ') }}</small>
          </div>
        </div>
      </template>
    </div>
  </section>
</template>

<style scoped>
.workflow-panel { padding: 22px 24px 25px; border: 1px solid var(--line); border-radius: 16px; background: linear-gradient(145deg, var(--surface), color-mix(in srgb, var(--accent-pale) 24%, var(--surface))); box-shadow: var(--shadow); }
.workflow-heading { display: flex; align-items: center; justify-content: space-between; gap: 20px; margin-bottom: 23px; }
.workflow-heading h2, .workflow-heading p { margin-bottom: 0; }
.workflow-heading h2 { font-size: 18px; }
.workflow-heading > span { padding: 5px 10px; border-radius: 999px; background: var(--surface-soft); color: var(--muted); font-size: 10px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; }
.workflow-track { display: grid; grid-template-columns: minmax(88px, 1fr) minmax(28px, .45fr) minmax(110px, 1fr) minmax(28px, .45fr) minmax(88px, 1fr) minmax(28px, .45fr) minmax(88px, 1fr) minmax(28px, .45fr) minmax(88px, 1fr); align-items: start; min-width: 740px; }
.workflow-stage { position: relative; display: grid; justify-items: center; gap: 4px; color: var(--muted); text-align: center; outline: none; }
.workflow-stage strong { color: var(--ink); font-size: 12px; line-height: 1.25; }
.workflow-stage > small { font-size: 10px; font-weight: 700; }
.workflow-node { position: relative; z-index: 2; display: grid; width: 38px; height: 38px; margin-bottom: 3px; place-items: center; border: 2px solid var(--line); border-radius: 50%; background: var(--surface); font-size: 15px; font-weight: 900; }
.workflow-stage.completed .workflow-node { border-color: var(--accent); background: var(--accent); color: #fff; }
.workflow-stage.completed > small { color: var(--accent); }
.workflow-stage.active .workflow-node { border-color: var(--accent); background: var(--accent-pale); color: var(--accent-dark); }
.workflow-stage.active .workflow-node::after { position: absolute; inset: -7px; border: 2px solid color-mix(in srgb, var(--accent) 65%, transparent); border-radius: 50%; content: ''; animation: workflow-pulse 1.5s ease-out infinite; }
.workflow-stage.active > small { color: var(--accent); }
.workflow-stage.failed { cursor: help; }
.workflow-stage.failed .workflow-node { border-color: var(--danger); background: var(--danger); color: #fff; box-shadow: 0 0 0 5px color-mix(in srgb, var(--danger) 13%, transparent); }
.workflow-stage.failed strong, .workflow-stage.failed > small { color: var(--danger); }
.workflow-stage.waiting { opacity: .58; }
.workflow-connector { position: relative; height: 38px; }
.workflow-connector::before { position: absolute; top: 18px; right: -5px; left: -5px; height: 2px; background: var(--line); content: ''; }
.workflow-connector span { position: absolute; z-index: 1; top: 14px; right: -5px; width: 0; height: 0; border-top: 5px solid transparent; border-bottom: 5px solid transparent; border-left: 7px solid var(--line); }
.workflow-connector.completed::before { background: var(--accent); }
.workflow-connector.completed span { border-left-color: var(--accent); }
.failure-tooltip { position: absolute; z-index: 12; top: calc(100% + 10px); left: 50%; width: min(310px, 70vw); padding: 13px 14px; border: 1px solid color-mix(in srgb, var(--danger) 30%, var(--line)); border-radius: 10px; background: var(--surface); box-shadow: 0 18px 45px rgba(25, 34, 30, .2); color: var(--ink); text-align: left; opacity: 0; transform: translate(-50%, -5px); visibility: hidden; transition: opacity .18s ease, transform .18s ease, visibility .18s ease; }
.failure-tooltip::before { position: absolute; top: -6px; left: calc(50% - 6px); width: 10px; height: 10px; border-top: 1px solid color-mix(in srgb, var(--danger) 30%, var(--line)); border-left: 1px solid color-mix(in srgb, var(--danger) 30%, var(--line)); background: var(--surface); content: ''; transform: rotate(45deg); }
.failure-tooltip strong { color: var(--danger); font-size: 11px; }
.failure-tooltip p { margin: 5px 0 0; font-size: 12px; line-height: 1.5; }
.failure-tooltip small { display: block; margin-top: 6px; color: var(--muted); font-size: 9px; text-transform: capitalize; }
.workflow-stage.failed:hover .failure-tooltip, .workflow-stage.failed:focus .failure-tooltip { opacity: 1; transform: translate(-50%, 0); visibility: visible; }
@keyframes workflow-pulse { 0% { opacity: .8; transform: scale(.8); } 80%, 100% { opacity: 0; transform: scale(1.2); } }
@media (max-width: 900px) { .workflow-panel { overflow-x: auto; } .workflow-heading { position: sticky; left: 0; } }
@media (prefers-reduced-motion: reduce) { .workflow-stage.active .workflow-node::after { animation: none; } }
</style>
