<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { locale, t } = useI18n()
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
      </nav>

      <div class="sidebar-footer">
        <label class="locale-control">
          <span>Language</span>
          <select v-model="locale">
            <option value="en">English</option>
            <option value="de">Deutsch</option>
          </select>
        </label>
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

