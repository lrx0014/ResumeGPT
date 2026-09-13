<script setup lang="ts">
import PageHeader from '../components/PageHeader.vue'
</script>

<template>
  <div class="page">
    <PageHeader
      eyebrow="Workspace overview"
      title="Build applications that stay true to you."
      description="Turn verified experience into focused CVs and cover letters for every opportunity."
    >
      <RouterLink class="button primary" to="/generate">Create application</RouterLink>
    </PageHeader>

    <section class="metric-grid" aria-label="Application metrics">
      <article class="metric-card">
        <span>Active profiles</span>
        <strong>—</strong>
        <small>Ready for your first profile</small>
      </article>
      <article class="metric-card">
        <span>Tracked roles</span>
        <strong>—</strong>
        <small>Add roles manually or by URL</small>
      </article>
      <article class="metric-card accent">
        <span>Documents generated</span>
        <strong>—</strong>
        <small>Evidence-backed by design</small>
      </article>
    </section>

    <section class="panel">
      <div class="panel-heading">
        <div>
          <p class="eyebrow">Getting started</p>
          <h2>Your first tailored application</h2>
        </div>
        <span class="status-pill">Prototype</span>
      </div>
      <div class="steps">
        <RouterLink class="step" to="/profiles">
          <span class="step-number">01</span>
          <div><strong>Create a profile</strong><p>Add experience, projects, education, and skills.</p></div>
          <span class="arrow">→</span>
        </RouterLink>
        <RouterLink class="step" to="/jobs">
          <span class="step-number">02</span>
          <div><strong>Track a role</strong><p>Capture the job description and application context.</p></div>
          <span class="arrow">→</span>
        </RouterLink>
        <RouterLink class="step" to="/generate">
          <span class="step-number">03</span>
          <div><strong>Generate and refine</strong><p>Create a targeted CV with claim-level validation.</p></div>
          <span class="arrow">→</span>
        </RouterLink>
      </div>
    </section>
  </div>
</template>

