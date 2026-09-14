import { createI18n } from 'vue-i18n'

const messages = {
  en: {
    nav: {
      overview: 'Overview',
      profiles: 'Profiles',
      jobs: 'Opportunities',
      generate: 'Generate',
    },
  },
  de: {
    nav: {
      overview: 'Übersicht',
      profiles: 'Profile',
      jobs: 'Chancen',
      generate: 'Erstellen',
    },
  },
}

export const i18n = createI18n({
  legacy: false,
  locale: 'en',
  fallbackLocale: 'en',
  messages,
})
