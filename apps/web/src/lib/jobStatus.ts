import { i18n } from '../plugins/i18n'

export const jobStatuses = [
  { value: 'interested', labelKey: 'jobs.status.interested' },
  { value: 'preparing', labelKey: 'jobs.status.preparing' },
  { value: 'applied', labelKey: 'jobs.status.applied' },
  { value: 'screening', labelKey: 'jobs.status.screening' },
  { value: 'interview', labelKey: 'jobs.status.interview' },
  { value: 'offer', labelKey: 'jobs.status.offer' },
  { value: 'accepted', labelKey: 'jobs.status.accepted' },
  { value: 'rejected', labelKey: 'jobs.status.rejected' },
  { value: 'withdrawn', labelKey: 'jobs.status.withdrawn' },
] as const

export function jobStatusLabel(value: string) {
  const status = jobStatuses.find(candidate => candidate.value === value)
  return status ? i18n.global.t(status.labelKey) : value
}
