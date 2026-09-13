import type { APIError, Job, ListResponse, Profile, KnowledgeSnapshot, FactVersion, FactReview } from './types'

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
  knowledge: (id: string) => request<KnowledgeSnapshot>(`/v1/profiles/${encodeURIComponent(id)}/knowledge`),
  exportKnowledge: (id: string) => request<KnowledgeSnapshot>(`/v1/profiles/${encodeURIComponent(id)}/knowledge/export`),
  importText: (id: string, input: { name: string; text: string }) => request(`/v1/profiles/${encodeURIComponent(id)}/sources`, { method: 'POST', body: JSON.stringify(input) }),
  uploadText: (id: string, file: File) => {
    const body = new FormData()
    body.append('file', file)
    return request(`/v1/profiles/${encodeURIComponent(id)}/sources/upload`, { method: 'POST', body })
  },
  reviewFact: (profileId: string, factId: string, input: FactReview) => request<FactVersion>(`/v1/profiles/${encodeURIComponent(profileId)}/facts/${encodeURIComponent(factId)}/reviews`, { method: 'POST', body: JSON.stringify(input) }),
  deleteSource: (profileId: string, sourceId: string) => request(`/v1/profiles/${encodeURIComponent(profileId)}/sources/${encodeURIComponent(sourceId)}`, { method: 'DELETE' }),
  searchKnowledge: (id: string, query: string) => request<ListResponse<FactVersion>>(`/v1/profiles/${encodeURIComponent(id)}/knowledge/search?q=${encodeURIComponent(query)}`),
  listProfiles: () => request<ListResponse<Profile>>('/v1/profiles'),
  createProfile: (input: Pick<Profile, 'name' | 'domain' | 'defaultLanguage' | 'description'>) =>
    request<Profile>('/v1/profiles', { method: 'POST', body: JSON.stringify(input) }),
  listJobs: () => request<ListResponse<Job>>('/v1/jobs'),
  createJob: (input: Pick<Job, 'title' | 'company' | 'location' | 'sourceUrl' | 'description'>) =>
    request<Job>('/v1/jobs', { method: 'POST', body: JSON.stringify(input) }),
}
