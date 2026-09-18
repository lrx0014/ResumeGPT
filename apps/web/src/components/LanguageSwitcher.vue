<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

import { api } from '../lib/api'
import { toast } from '../lib/toast'
import { interfaceLanguages, normalizeInterfaceLocale, type InterfaceLocale } from '../plugins/i18n'

const { locale, t } = useI18n()
const switching = ref(false)

async function switchLanguage(event: Event) {
  const language = normalizeInterfaceLocale((event.target as HTMLSelectElement).value)
  if (switching.value || language === locale.value) return
  const previous = locale.value as InterfaceLocale
  locale.value = language
  document.documentElement.lang = language
  switching.value = true
  try {
    const current = await api.getSettings()
    await api.updateSettings({ interfaceLanguage: language, theme: current.theme })
    toast.success(t('settings.languageSaved'))
  } catch (cause) {
    locale.value = previous
    document.documentElement.lang = previous
    toast.warning(cause instanceof Error ? cause.message : t('settings.languageError'))
  } finally {
    switching.value = false
  }
}
</script>

<template>
  <div class="locale-control">
    <select :value="locale" :disabled="switching" :aria-label="t('settings.languageTitle')" @change="switchLanguage">
      <option v-for="language in interfaceLanguages" :key="language.code" :value="language.code">{{ language.nativeName }}</option>
    </select>
  </div>
</template>
