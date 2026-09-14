export interface Profile {
  id: string
  workspaceId: string
  name: string
  targetRole?: string
  defaultLanguage: string
  content: string
  avatarObjectId?: string
  createdAt: string
  updatedAt: string
}

export interface Job {
  id: string
  workspaceId: string
  title: string
  company: string
  location?: string
  country?: string
  city?: string
  workMode?: string
  employmentType?: string
  sourceUrl?: string
  description?: string
  status: string
  importState: 'manual' | 'queued' | 'fetching' | 'ready' | 'needs_user_action' | 'failed'
  importError?: string
  createdAt: string
  updatedAt: string
}

export type JobInput = Pick<Job, 'title' | 'company' | 'location' | 'country' | 'city' | 'workMode' | 'employmentType' | 'sourceUrl' | 'description' | 'status'>

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

export type DocumentUploadState = 'staged' | 'queued' | 'ready' | 'needs_user_action' | 'security_quarantine' | 'failed'

export interface DocumentUpload {
  id: string
  workspaceId: string
  profileId: string
  objectId: string
  name: string
  declaredMediaType: string
  state: DocumentUploadState
  jobId?: string
  extractedText?: string
  errorCode?: string
  errorMessage?: string
  createdAt: string
  updatedAt: string
}

export interface StagedDocumentUpload {
  upload: DocumentUpload
  target: {
    objectId: string
    url: string
    expiresAt: string
    headers?: Record<string, string>
  }
}

export interface SignedURL {
  objectId: string
  url: string
  expiresAt: string
  headers?: Record<string, string>
}
