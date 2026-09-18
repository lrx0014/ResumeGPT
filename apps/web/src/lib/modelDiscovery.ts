import { reactive } from 'vue'

import { api } from './api'
import type { GenerationModelChoice } from './types'

export function useModelDiscovery(onError: (message: string) => void, fallbackMessage: () => string) {
  const models = reactive<Record<string, string[]>>({})
  const loadingModelConnections = reactive<Record<string, boolean>>({})
  const modelRequests = new Map<string, Promise<string[]>>()

  async function discover(choice: GenerationModelChoice) {
    const connectionId = choice.connectionId
    if (!connectionId) return
    if (models[connectionId]) {
      if (!choice.model) choice.model = models[connectionId][0] ?? ''
      return
    }
    loadingModelConnections[connectionId] = true
    let request = modelRequests.get(connectionId)
    if (!request) {
      request = api.testLLMConnection(connectionId).then(result => result.models)
      modelRequests.set(connectionId, request)
    }
    try {
      models[connectionId] = await request
      if (choice.connectionId === connectionId && !choice.model) choice.model = models[connectionId][0] ?? ''
    } catch (cause) {
      onError(cause instanceof Error ? cause.message : fallbackMessage())
    } finally {
      loadingModelConnections[connectionId] = false
      modelRequests.delete(connectionId)
    }
  }

  function changeConnection(choice: GenerationModelChoice) {
    choice.model = ''
    void discover(choice)
  }

  return { models, loadingModelConnections, discover, changeConnection }
}
