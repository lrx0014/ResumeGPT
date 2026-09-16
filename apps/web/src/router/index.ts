import { createRouter, createWebHistory } from 'vue-router'

import DashboardView from '../views/DashboardView.vue'
import GenerateView from '../views/GenerateView.vue'
import GenerationView from '../views/GenerationView.vue'
import JobsView from '../views/JobsView.vue'
import JobView from '../views/JobView.vue'
import ProfilesView from '../views/ProfilesView.vue'
import ProfileView from '../views/ProfileView.vue'
import SettingsView from '../views/SettingsView.vue'
import TemplatesView from '../views/TemplatesView.vue'
import TemplateView from '../views/TemplateView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'dashboard', component: DashboardView },
    { path: '/profiles', name: 'profiles', component: ProfilesView },
    { path: '/profiles/:profileId', name: 'profile', component: ProfileView },
    { path: '/jobs', name: 'jobs', component: JobsView },
    { path: '/jobs/:jobId', name: 'job', component: JobView },
    { path: '/templates', name: 'templates', component: TemplatesView },
    { path: '/templates/:templateId', name: 'template', component: TemplateView },
    { path: '/generate', name: 'generate', component: GenerateView },
    { path: '/generate/:generationId', name: 'generation', component: GenerationView },
    { path: '/settings', name: 'settings', component: SettingsView },
  ],
})
