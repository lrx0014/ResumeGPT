import type { APIError, Job, ListResponse, Profile } from './types'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
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
  listProfiles: () => request<ListResponse<Profile>>('/v1/profiles'),
  createProfile: (input: Pick<Profile, 'name' | 'domain' | 'defaultLanguage' | 'description'>) =>
    request<Profile>('/v1/profiles', { method: 'POST', body: JSON.stringify(input) }),
  listJobs: () => request<ListResponse<Job>>('/v1/jobs'),
  createJob: (input: Pick<Job, 'title' | 'company' | 'location' | 'sourceUrl' | 'description'>) =>
    request<Job>('/v1/jobs', { method: 'POST', body: JSON.stringify(input) }),
}

