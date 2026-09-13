import { createI18n } from 'vue-i18n'

const messages = {
  en: {
    nav: {
      overview: 'Overview',
      profiles: 'Profiles',
      jobs: 'Jobs',
      generate: 'Generate',
    },
  },
  de: {
    nav: {
      overview: 'Übersicht',
      profiles: 'Profile',
      jobs: 'Stellen',
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

