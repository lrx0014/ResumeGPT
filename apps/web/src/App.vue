<script setup lang="ts">
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'

import { api } from './lib/api'
import { applyTheme } from './lib/preferences'

const { locale, t } = useI18n()

onMounted(async () => {
  try {
    const settings = await api.getSettings()
    locale.value = settings.interfaceLanguage
    applyTheme(settings.theme)
  } catch {
    applyTheme('system')
  }
})
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar">
      <RouterLink class="brand" to="/">
        <span class="brand-mark">R</span>
        <span>ResumeGPT</span>
      </RouterLink>

      <nav class="nav" aria-label="Primary navigation">
        <RouterLink to="/" exact-active-class="active">
          <span class="nav-icon">◫</span>{{ t('nav.overview') }}
        </RouterLink>
        <RouterLink to="/profiles" active-class="active">
          <span class="nav-icon">◎</span>{{ t('nav.profiles') }}
        </RouterLink>
        <RouterLink to="/jobs" active-class="active">
          <span class="nav-icon">◇</span>{{ t('nav.jobs') }}
        </RouterLink>
        <RouterLink to="/generate" active-class="active">
          <span class="nav-icon">✦</span>{{ t('nav.generate') }}
        </RouterLink>
        <RouterLink to="/settings" active-class="active">
          <span class="nav-icon">⚙</span>{{ t('nav.settings') }}
        </RouterLink>
      </nav>

      <div class="sidebar-footer">
        <div class="user-card">
          <span class="avatar">DV</span>
          <span><strong>Development</strong><small>Personal workspace</small></span>
        </div>
      </div>
    </aside>

    <main class="main-content">
      <RouterView />
    </main>
  </div>
</template>
