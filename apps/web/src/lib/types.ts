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

export interface KnowledgeSource {
  id: string
  name: string
  text: string
  hash: string
  mediaType: string
  parserVersion: string
  state: string
  createdAt: string
}

export interface EvidenceSegment {
  id: string
  sourceId: string
  page?: number
  paragraph: number
  text: string
  hash: string
  confidence: number
  boundingBox?: { x: number; y: number; width: number; height: number }
}

export type FactStatus = 'extracted' | 'user_asserted' | 'user_confirmed' | 'disputed' | 'rejected'

export interface FactVersion {
  id: string
  factId: string
  number: number
  statement: string
  status: FactStatus
  sensitive: boolean
  actorId: string
  createdAt: string
}

export interface KnowledgeFact {
  id: string
  evidenceSegmentIds: string[]
  currentVersionId: string
  versions: FactVersion[]
}

export interface FactReview {
  expectedVersionId: string
  statement: string
  status: Exclude<FactStatus, 'extracted'>
  sensitive: boolean
}

export interface KnowledgeSnapshot {
  schemaVersion: number
  profileId: string
  sources: KnowledgeSource[]
  segments: EvidenceSegment[]
  facts: KnowledgeFact[]
}
