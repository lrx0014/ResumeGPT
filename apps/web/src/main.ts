import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import { i18n } from './plugins/i18n'
import { router } from './router'
import './styles/main.css'

createApp(App).use(createPinia()).use(router).use(i18n).mount('#app')

