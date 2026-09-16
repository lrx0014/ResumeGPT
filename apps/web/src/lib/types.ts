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
  importState: 'manual' | 'queued' | 'fetching' | 'analyzing' | 'ready' | 'needs_user_action' | 'failed'
  importError?: string
  createdAt: string
  updatedAt: string
}

export type JobInput = Pick<Job, 'title' | 'company' | 'location' | 'country' | 'city' | 'workMode' | 'employmentType' | 'sourceUrl' | 'description' | 'status'>

export interface JobImportInput {
  urls: string[]
  aiAssisted: boolean
  connectionId?: string
  model?: string
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

export interface SettingsPreferences {
  workspaceId: string
  interfaceLanguage: 'en' | 'de'
  theme: 'system' | 'light' | 'dark'
  updatedAt: string
}

export interface LLMConnection {
  id: string
  workspaceId: string
  name: string
  executionMode: 'cloud' | 'local'
  provider: 'openai' | 'openai_compatible' | 'ollama'
  baseUrl: string
  apiTokenConfigured: boolean
  createdAt: string
  updatedAt: string
}

export interface LLMConnectionInput {
  name: string
  executionMode: LLMConnection['executionMode']
  provider: LLMConnection['provider']
  baseUrl: string
  apiToken: string
  clearApiToken: boolean
}

export interface LLMConnectionTest {
  status: 'connected'
  models: string[]
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

export type TemplateKind = 'resume' | 'cover_letter'
export type TemplateFormat = 'latex' | 'doc' | 'docx'
export type TemplateState = 'staged' | 'queued' | 'ready' | 'needs_user_action' | 'security_quarantine' | 'failed'

export interface Template {
  id: string
  workspaceId: string
  name: string
  kind: TemplateKind
  format: TemplateFormat
  description?: string
  sourceName: string
  entryFile?: string
  declaredMediaType: string
  objectId?: string
  previewObjectId?: string
  content?: string
  state: TemplateState
  jobId?: string
  errorCode?: string
  errorMessage?: string
  builtIn: boolean
  authorName?: string
  sourceUrl?: string
  license?: string
  createdAt?: string
  updatedAt?: string
}

export interface StagedTemplate {
  template: Template
  target: SignedURL
}

export interface GenerationModelChoice { connectionId: string; model: string }
export type GenerationState = 'queued' | 'running' | 'ready' | 'failed'
export interface GenerationRun {
  id: string; workspaceId: string; profileId: string; opportunityId: string; templateId: string
  documentType: TemplateKind; language: string; pageTarget: 'one_page' | 'two_pages' | 'flexible'
  customInstructions?: string; pipelineMode: 'single' | 'multi'
  writer: GenerationModelChoice; renderer: GenerationModelChoice; reviewer: GenerationModelChoice
  state: GenerationState; stage: string; draft?: string; review?: string; repairCount: number
  artifactObjectId?: string; errorCode?: string; errorMessage?: string; createdAt: string; updatedAt: string
}
export interface GenerationInput {
  profileId: string; opportunityId: string; templateId: string; documentType: TemplateKind
  language: string; pageTarget: GenerationRun['pageTarget']; customInstructions: string
  pipelineMode: GenerationRun['pipelineMode']; writer: GenerationModelChoice; renderer: GenerationModelChoice; reviewer: GenerationModelChoice
}
export type GenerationStepKind = 'writer_draft' | 'rendered_pdf' | 'reviewer_feedback' | 'user_prompt' | 'system_warning' | 'configuration_change'
export interface GenerationStep {
  id: string; workspaceId: string; generationId: string; kind: GenerationStepKind; sequence: number
  content?: string; feedback?: string; artifactObjectId?: string; repairCount: number; createdAt: string
}
