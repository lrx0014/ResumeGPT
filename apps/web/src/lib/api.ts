import type { APIError, Job, JobInput, ListResponse, Profile, DocumentUpload, SignedURL, StagedDocumentUpload, SettingsPreferences, LLMConnection, LLMConnectionInput, LLMConnectionTest } from './types'

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
  return response.json() as Promise<T>
}

export const api = {
  capabilities: () => request<{ features: Record<string, boolean> }>('/v1/system/capabilities'),
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
  deleteProfile: async (id: string) => {
    const response = await fetch(`${API_BASE_URL}/v1/profiles/${encodeURIComponent(id)}`, { method: 'DELETE', headers: { 'X-Workspace-ID': 'ws_personal_dev' } })
    if (!response.ok) {
      const payload = (await response.json().catch(() => null)) as APIError | null
      throw new Error(payload?.error.message ?? `Request failed with status ${response.status}`)
    }
  },
  listJobs: () => request<ListResponse<Job>>('/v1/jobs'),
  getJob: (id: string) => request<Job>(`/v1/jobs/${encodeURIComponent(id)}`),
  createJob: (input: JobInput) =>
    request<Job>('/v1/jobs', { method: 'POST', body: JSON.stringify(input) }),
  updateJob: (id: string, input: JobInput) =>
    request<Job>(`/v1/jobs/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),
  importJobs: (urls: string[]) => request<ListResponse<Job>>('/v1/jobs/imports', { method: 'POST', body: JSON.stringify({ urls }) }),
  deleteJob: async (id: string) => {
    const response = await fetch(`${API_BASE_URL}/v1/jobs/${encodeURIComponent(id)}`, { method: 'DELETE', headers: { 'X-Workspace-ID': 'ws_personal_dev' } })
    if (!response.ok) {
      const payload = (await response.json().catch(() => null)) as APIError | null
      throw new Error(payload?.error.message ?? `Request failed with status ${response.status}`)
    }
  },
  getSettings: () => request<SettingsPreferences>('/v1/settings'),
  updateSettings: (input: Pick<SettingsPreferences, 'interfaceLanguage' | 'theme'>) =>
    request<SettingsPreferences>('/v1/settings', { method: 'PUT', body: JSON.stringify(input) }),
  listLLMConnections: () => request<ListResponse<LLMConnection>>('/v1/settings/llm-connections'),
  createLLMConnection: (input: LLMConnectionInput) =>
    request<LLMConnection>('/v1/settings/llm-connections', { method: 'POST', body: JSON.stringify(input) }),
  updateLLMConnection: (id: string, input: LLMConnectionInput) =>
    request<LLMConnection>(`/v1/settings/llm-connections/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),
  deleteLLMConnection: async (id: string) => {
    const response = await fetch(`${API_BASE_URL}/v1/settings/llm-connections/${encodeURIComponent(id)}`, { method: 'DELETE', headers: { 'X-Workspace-ID': 'ws_personal_dev' } })
    if (!response.ok) {
      const payload = (await response.json().catch(() => null)) as APIError | null
      throw new Error(payload?.error.message ?? `Request failed with status ${response.status}`)
    }
  },
  testLLMConnection: (id: string) =>
    request<LLMConnectionTest>(`/v1/settings/llm-connections/${encodeURIComponent(id)}/test`, { method: 'POST' }),
}
