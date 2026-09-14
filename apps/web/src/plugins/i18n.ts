import { createI18n } from 'vue-i18n'

const messages = {
  en: {
    navGroups: {
      workspace: 'Workspace',
      tracking: 'Job search',
      workflow: 'Creation workflow',
      system: 'System',
    },
    nav: {
      overview: 'Overview',
      profiles: 'Profiles',
      jobs: 'Opportunities',
      templates: 'Templates',
      generate: 'Generate',
      settings: 'Settings',
    },
  },
  de: {
    navGroups: {
      workspace: 'Arbeitsbereich',
      tracking: 'Jobsuche',
      workflow: 'Erstellungsablauf',
      system: 'System',
    },
    nav: {
      overview: 'Übersicht',
      profiles: 'Profile',
      jobs: 'Chancen',
      templates: 'Vorlagen',
      generate: 'Erstellen',
      settings: 'Einstellungen',
    },
  },
}

export const i18n = createI18n({
  legacy: false,
  locale: 'en',
  fallbackLocale: 'en',
  messages,
})
