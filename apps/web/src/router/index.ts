import { createRouter, createWebHistory } from 'vue-router'

import DashboardView from '../views/DashboardView.vue'
import GenerateView from '../views/GenerateView.vue'
import GenerationView from '../views/GenerationView.vue'
import JobsView from '../views/JobsView.vue'
import JobView from '../views/JobView.vue'
import JobHuntersView from '../views/JobHuntersView.vue'
import JobHunterReviewView from '../views/JobHunterReviewView.vue'
import ProfilesView from '../views/ProfilesView.vue'
import ProfileView from '../views/ProfileView.vue'
import SettingsView from '../views/SettingsView.vue'
import TemplatesView from '../views/TemplatesView.vue'
import TemplateView from '../views/TemplateView.vue'
import TaskMonitorView from '../views/TaskMonitorView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'dashboard', component: DashboardView },
    { path: '/profiles', name: 'profiles', component: ProfilesView },
    { path: '/profiles/:profileId', name: 'profile', component: ProfileView },
    { path: '/jobs', name: 'jobs', component: JobsView },
    { path: '/jobs/:jobId', name: 'job', component: JobView },
    { path: '/job-hunters', name: 'job-hunters', component: JobHuntersView },
    { path: '/job-hunters/:hunterId/review', name: 'job-hunter-review', component: JobHunterReviewView },
    { path: '/templates', name: 'templates', component: TemplatesView },
    { path: '/templates/:templateId', name: 'template', component: TemplateView },
    { path: '/generate', name: 'generate', component: GenerateView },
    { path: '/generate/:generationId', name: 'generation', component: GenerationView },
    { path: '/settings', name: 'settings', component: SettingsView },
    { path: '/system/tasks', name: 'task-monitor', component: TaskMonitorView },
  ],
})
