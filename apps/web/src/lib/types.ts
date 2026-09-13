export interface Profile {
  id: string
  workspaceId: string
  name: string
  domain?: string
  defaultLanguage: string
  description?: string
  createdAt: string
  updatedAt: string
}

export interface Job {
  id: string
  workspaceId: string
  title: string
  company: string
  location?: string
  sourceUrl?: string
  description?: string
  status: string
  createdAt: string
  updatedAt: string
}

export interface ListResponse<T> {
  items: T[]
}

export interface APIError {
  error: {
    code: string
    message: string
    retryable: boolean
  }
}

