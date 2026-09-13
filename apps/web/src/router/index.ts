import { createRouter, createWebHistory } from 'vue-router'

import DashboardView from '../views/DashboardView.vue'
import GenerateView from '../views/GenerateView.vue'
import JobsView from '../views/JobsView.vue'
import ProfilesView from '../views/ProfilesView.vue'
import ProfileKnowledgeView from '../views/ProfileKnowledgeView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'dashboard', component: DashboardView },
    { path: '/profiles', name: 'profiles', component: ProfilesView },
    { path: '/profiles/:profileId', name: 'profile-knowledge', component: ProfileKnowledgeView },
    { path: '/jobs', name: 'jobs', component: JobsView },
    { path: '/generate', name: 'generate', component: GenerateView },
  ],
})
