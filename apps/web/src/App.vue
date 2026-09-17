<script setup lang="ts">
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'

import { api } from './lib/api'
import { applyTheme } from './lib/preferences'
import ToastViewport from './components/ToastViewport.vue'

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
        <span class="brand-mark">Ré</span>
        <span>ResumeGPT</span>
      </RouterLink>

      <nav class="nav primary-nav" aria-label="Primary navigation">
        <section class="nav-group" :aria-label="t('navGroups.workspace')">
          <p class="nav-group-label">{{ t('navGroups.workspace') }}</p>
          <RouterLink to="/" exact-active-class="active">
            <span class="nav-icon">◫</span>{{ t('nav.overview') }}
          </RouterLink>
        </section>

        <section class="nav-group" :aria-label="t('navGroups.tracking')">
          <p class="nav-group-label">{{ t('navGroups.tracking') }}</p>
          <RouterLink to="/jobs" active-class="active">
            <span class="nav-icon">◇</span>{{ t('nav.jobs') }}
          </RouterLink>
          <RouterLink to="/job-hunters" active-class="active">
            <span class="nav-icon">⌖</span>{{ t('nav.hunters') }}
          </RouterLink>
        </section>

        <section class="nav-group workflow-nav" :aria-label="t('navGroups.workflow')">
          <p class="nav-group-label">{{ t('navGroups.workflow') }}</p>
          <RouterLink to="/profiles" active-class="active">
            <span class="workflow-number">1</span>{{ t('nav.profiles') }}
          </RouterLink>
          <RouterLink to="/templates" active-class="active">
            <span class="workflow-number">2</span>{{ t('nav.templates') }}
          </RouterLink>
          <RouterLink to="/generate" active-class="active">
            <span class="workflow-number">3</span>{{ t('nav.generate') }}
          </RouterLink>
        </section>
      </nav>

      <div class="sidebar-footer">
        <nav class="nav utility-nav" :aria-label="t('navGroups.system')">
          <section class="nav-group">
            <p class="nav-group-label">{{ t('navGroups.system') }}</p>
            <RouterLink to="/system/tasks" active-class="active">
              <span class="nav-icon">◴</span>{{ t('nav.tasks') }}
            </RouterLink>
            <RouterLink to="/settings" active-class="active">
              <span class="nav-icon">⚙</span>{{ t('nav.settings') }}
            </RouterLink>
          </section>
        </nav>
        <div class="user-card">
          <span class="avatar">DV</span>
          <span><strong>Development</strong><small>Personal workspace</small></span>
        </div>
      </div>
    </aside>

    <main class="main-content">
      <RouterView />
    </main>
    <ToastViewport />
  </div>
</template>
