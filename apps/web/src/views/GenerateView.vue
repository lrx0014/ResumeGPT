<script setup lang="ts">
import PageHeader from '../components/PageHeader.vue'
</script>

<template>
  <div class="page">
    <PageHeader title="Generate" description="Combine one trusted profile with one target role and a controlled template." />
    <section class="generation-layout">
      <div class="panel">
        <p class="eyebrow">Configuration</p>
        <h2>New application document</h2>
        <div class="form-grid single">
          <label><span>Document type</span><select><option>Curriculum vitae</option><option>Cover letter</option></select></label>
          <label><span>Profile</span><select disabled><option>Create a profile first</option></select></label>
          <label><span>Target job</span><select disabled><option>Add a job first</option></select></label>
          <label><span>Page target</span><select><option>One page</option><option>Two pages</option><option>Custom</option></select></label>
        </div>
        <button class="button primary wide" disabled>Generate draft</button>
      </div>
      <aside class="panel trust-panel">
        <span class="trust-icon">✓</span>
        <p class="eyebrow">Profile grounded</p>
        <h2>Every material claim stays connected to your saved profile.</h2>
        <p>ResumeGPT maps job requirements to your profile content before writing. Unsupported claims are surfaced before export.</p>
        <ul><li>Saved profile snapshot</li><li>Structured draft schema</li><li>Layout and page validation</li></ul>
      </aside>
    </section>
  </div>
</template>
