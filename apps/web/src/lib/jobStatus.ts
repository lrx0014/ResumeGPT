export const jobStatuses = [
  { value: 'interested', label: 'Interested' },
  { value: 'preparing', label: 'Preparing' },
  { value: 'applied', label: 'Applied' },
  { value: 'screening', label: 'Screening' },
  { value: 'interview', label: 'Interview' },
  { value: 'offer', label: 'Offer' },
  { value: 'accepted', label: 'Accepted' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'withdrawn', label: 'Withdrawn' },
] as const

export function jobStatusLabel(value: string) {
  return jobStatuses.find(status => status.value === value)?.label ?? value
}
