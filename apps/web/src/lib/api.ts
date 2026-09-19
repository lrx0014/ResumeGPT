import type { AgentDefault, APIError, BackgroundTask, BackgroundTaskPage, Job, JobHunter, JobHunterInput, JobHunterReviewItem, JobImportInput, JobInput, ListResponse, Profile, DocumentUpload, SignedURL, StagedDocumentUpload, SettingsPreferences, LLMConnection, LLMConnectionInput, LLMConnectionTest, StagedTemplate, Template, TemplateKind, GenerationInput, GenerationRun, GenerationStep, WorkspaceOverview } from './types'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      ...(init?.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }),
      'X-Workspace-ID': 'ws_personal_dev',
      ...init?.headers,
    },
  })
  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as APIError | null
    throw new Error(payload?.error.message ?? `Request failed with status ${response.status}`)
  }
  const text = await response.text()
  return (text === '' ? undefined : JSON.parse(text)) as T
}

export const api = {
  capabilities: () => request<{ features: Record<string, boolean> }>('/v1/system/capabilities'),
  overview: () => request<WorkspaceOverview>('/v1/overview'),
  listTasks: (input: { search?: string; state?: string; kind?: string; page?: number; pageSize?: number }) => {
    const query = new URLSearchParams()
    if (input.search) query.set('search', input.search)
    if (input.state && input.state !== 'all') query.set('state', input.state)
    if (input.kind && input.kind !== 'all') query.set('kind', input.kind)
    query.set('page', String(input.page ?? 1))
    query.set('pageSize', String(input.pageSize ?? 20))
    return request<BackgroundTaskPage>(`/v1/system/tasks?${query}`)
  },
  getTask: (id: string) => request<BackgroundTask>(`/v1/system/tasks/${encodeURIComponent(id)}`),
  getProfile: (id: string) => request<Profile>(`/v1/profiles/${encodeURIComponent(id)}`),
  stageDocument: (id: string, file: File) => request<StagedDocumentUpload>(`/v1/profiles/${encodeURIComponent(id)}/document-uploads`, {
    method: 'POST', body: JSON.stringify({ name: file.name, contentType: file.type || 'application/octet-stream' }),
  }),
  putStagedDocument: async (target: StagedDocumentUpload['target'], file: File) => {
    const response = await fetch(target.url, { method: 'PUT', headers: target.headers, body: file })
    if (!response.ok) throw new Error(`Object upload failed with status ${response.status}`)
  },
  completeDocumentUpload: (profileId: string, uploadId: string) => request<DocumentUpload>(`/v1/profiles/${encodeURIComponent(profileId)}/document-uploads/${encodeURIComponent(uploadId)}/complete`, { method: 'POST' }),
  documentUpload: (profileId: string, uploadId: string) => request<DocumentUpload>(`/v1/profiles/${encodeURIComponent(profileId)}/document-uploads/${encodeURIComponent(uploadId)}`),
  createAvatarUpload: (profileId: string, contentType: string) => request<SignedURL>(`/v1/profiles/${encodeURIComponent(profileId)}/avatar-upload`, { method: 'POST', body: JSON.stringify({ contentType }) }),
  profileAvatar: (profileId: string) => request<SignedURL>(`/v1/profiles/${encodeURIComponent(profileId)}/avatar`),
  listProfiles: () => request<ListResponse<Profile>>('/v1/profiles'),
  createProfile: (input: Pick<Profile, 'name' | 'targetRole' | 'defaultLanguage' | 'content' | 'avatarObjectId'>) =>
    request<Profile>('/v1/profiles', { method: 'POST', body: JSON.stringify(input) }),
  updateProfile: (id: string, input: Pick<Profile, 'name' | 'targetRole' | 'defaultLanguage' | 'content' | 'avatarObjectId'>) =>
    request<Profile>(`/v1/profiles/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),
  deleteProfile: (id: string) => request<void>(`/v1/profiles/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  listJobs: () => request<ListResponse<Job>>('/v1/jobs'),
  getJob: (id: string) => request<Job>(`/v1/jobs/${encodeURIComponent(id)}`),
  createJob: (input: JobInput) =>
    request<Job>('/v1/jobs', { method: 'POST', body: JSON.stringify(input) }),
  updateJob: (id: string, input: JobInput) =>
    request<Job>(`/v1/jobs/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),
  updateJobStatus: (id: string, status: string) =>
    request<void>(`/v1/jobs/${encodeURIComponent(id)}/status`, { method: 'PATCH', body: JSON.stringify({ status }) }),
  importJobs: (input: JobImportInput) => request<ListResponse<Job>>('/v1/jobs/imports', { method: 'POST', body: JSON.stringify(input) }),
  deleteJob: (id: string) => request<void>(`/v1/jobs/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  listJobHunters: () => request<ListResponse<JobHunter>>('/v1/job-hunters'),
  getJobHunter: (id: string) => request<JobHunter>(`/v1/job-hunters/${encodeURIComponent(id)}`),
  createJobHunter: (input: JobHunterInput) => request<JobHunter>('/v1/job-hunters', { method: 'POST', body: JSON.stringify(input) }),
  updateJobHunter: (id: string, input: JobHunterInput) => request<JobHunter>(`/v1/job-hunters/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),
  runJobHunter: (id: string) => request<{ status: string }>(`/v1/job-hunters/${encodeURIComponent(id)}/run`, { method: 'POST' }),
  listJobHunterReviewItems: (id: string) => request<ListResponse<JobHunterReviewItem>>(`/v1/job-hunters/${encodeURIComponent(id)}/review-items`),
  dismissJobHunterReviewItem: (hunterId: string, reviewId: string) =>
    request<void>(`/v1/job-hunters/${encodeURIComponent(hunterId)}/review-items/${encodeURIComponent(reviewId)}`, { method: 'DELETE' }),
  deleteJobHunter: (id: string) => request<void>(`/v1/job-hunters/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  getSettings: () => request<SettingsPreferences>('/v1/settings'),
  updateSettings: (input: Pick<SettingsPreferences, 'interfaceLanguage' | 'theme'>) =>
    request<SettingsPreferences>('/v1/settings', { method: 'PUT', body: JSON.stringify(input) }),
  getAgentDefaults: () => request<ListResponse<AgentDefault>>('/v1/settings/agent-defaults'),
  updateAgentDefaults: (items: AgentDefault[]) => request<ListResponse<AgentDefault>>('/v1/settings/agent-defaults', { method: 'PUT', body: JSON.stringify({ items }) }),
  listLLMConnections: () => request<ListResponse<LLMConnection>>('/v1/settings/llm-connections'),
  createLLMConnection: (input: LLMConnectionInput) =>
    request<LLMConnection>('/v1/settings/llm-connections', { method: 'POST', body: JSON.stringify(input) }),
  updateLLMConnection: (id: string, input: LLMConnectionInput) =>
    request<LLMConnection>(`/v1/settings/llm-connections/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),
  deleteLLMConnection: (id: string) => request<void>(`/v1/settings/llm-connections/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  testLLMConnection: (id: string, refresh = false) =>
    request<LLMConnectionTest>(`/v1/settings/llm-connections/${encodeURIComponent(id)}/test${refresh ? '?refresh=true' : ''}`, { method: 'POST' }),
  listTemplates: () => request<ListResponse<Template>>('/v1/templates'),
  getTemplate: (id: string) => request<Template>(`/v1/templates/${encodeURIComponent(id)}`),
  stageTemplate: (input: { name: string; kind: TemplateKind; description: string; entryFile: string; file: File }) =>
    request<StagedTemplate>('/v1/templates/uploads', { method: 'POST', body: JSON.stringify({ name: input.name, kind: input.kind, description: input.description, sourceName: input.file.name, contentType: input.file.type || 'application/octet-stream', entryFile: input.entryFile }) }),
  putStagedTemplate: async (target: StagedTemplate['target'], file: File) => {
    const response = await fetch(target.url, { method: 'PUT', headers: target.headers, body: file })
    if (!response.ok) throw new Error(`Object upload failed with status ${response.status}`)
  },
  completeTemplate: (id: string) => request<Template>(`/v1/templates/${encodeURIComponent(id)}/complete`, { method: 'POST' }),
  stageTemplateSource: (id: string, file: File, entryFile = '') => request<StagedTemplate>(`/v1/templates/${encodeURIComponent(id)}/source-upload`, {
    method: 'POST', body: JSON.stringify({ sourceName: file.name, contentType: file.type || 'application/octet-stream', entryFile }),
  }),
  updateTemplate: (id: string, input: Pick<Template, 'name' | 'kind' | 'description'>) =>
    request<Template>(`/v1/templates/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),
  deleteTemplate: (id: string) => request<void>(`/v1/templates/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  downloadTemplate: async (item: Template) => {
    const response = await fetch(`${API_BASE_URL}/v1/templates/${encodeURIComponent(item.id)}/file`, { headers: { 'X-Workspace-ID': 'ws_personal_dev' } })
    if (!response.ok) { const payload = (await response.json().catch(() => null)) as APIError | null; throw new Error(payload?.error.message ?? `Request failed with status ${response.status}`) }
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a'); link.href = url; link.download = item.sourceName; link.click(); URL.revokeObjectURL(url)
  },
  templatePreview: async (id: string) => {
    const response = await fetch(`${API_BASE_URL}/v1/templates/${encodeURIComponent(id)}/preview`, { headers: { 'X-Workspace-ID': 'ws_personal_dev' } })
    if (!response.ok) { const payload = (await response.json().catch(() => null)) as APIError | null; throw new Error(payload?.error.message ?? `Preview failed with status ${response.status}`) }
    return response.blob()
  },
  listGenerations: () => request<ListResponse<GenerationRun>>('/v1/generations'),
  getGeneration: (id: string) => request<GenerationRun>(`/v1/generations/${encodeURIComponent(id)}`),
  createGeneration: (input: GenerationInput) => request<GenerationRun>('/v1/generations', { method: 'POST', body: JSON.stringify(input) }),
  reconfigureGeneration: (id: string, input: GenerationInput) => request<GenerationRun>(`/v1/generations/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),
  deleteGeneration: (id: string) => request<void>(`/v1/generations/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  retryGeneration: (id: string) => request<GenerationRun>(`/v1/generations/${encodeURIComponent(id)}/retry`, { method: 'POST' }),
  generationSteps: (id: string) => request<ListResponse<GenerationStep>>(`/v1/generations/${encodeURIComponent(id)}/steps`),
  reviseGeneration: (id: string, prompt: string) => request<GenerationRun>(`/v1/generations/${encodeURIComponent(id)}/revisions`, { method: 'POST', body: JSON.stringify({ prompt }) }),
  generationPreview: async (id: string, stepId?: string) => {
    const suffix = stepId ? `/steps/${encodeURIComponent(stepId)}/artifact` : '/artifact'
    const response = await fetch(`${API_BASE_URL}/v1/generations/${encodeURIComponent(id)}${suffix}`, { headers: { 'X-Workspace-ID': 'ws_personal_dev' } })
    if (!response.ok) { const payload = (await response.json().catch(() => null)) as APIError | null; throw new Error(payload?.error.message ?? `Preview failed with status ${response.status}`) }
    return response.blob()
  },
  downloadGeneration: async (run: GenerationRun) => {
    const response = await fetch(`${API_BASE_URL}/v1/generations/${encodeURIComponent(run.id)}/artifact`, { headers: { 'X-Workspace-ID': 'ws_personal_dev' } })
    if (!response.ok) { const payload = (await response.json().catch(() => null)) as APIError | null; throw new Error(payload?.error.message ?? `Download failed with status ${response.status}`) }
    const blob = await response.blob(); const url = URL.createObjectURL(blob); const link = document.createElement('a')
    link.href = url; link.download = `${run.documentType}-${run.id}.pdf`; link.click(); URL.revokeObjectURL(url)
  },
}
