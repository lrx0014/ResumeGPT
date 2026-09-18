const formatters = new Map<string, Intl.DateTimeFormat>()

function cachedFormatter(options: Intl.DateTimeFormatOptions): Intl.DateTimeFormat {
  const key = JSON.stringify(options)
  let instance = formatters.get(key)
  if (!instance) {
    instance = new Intl.DateTimeFormat(undefined, options)
    formatters.set(key, instance)
  }
  return instance
}

export function formatDateTime(value: string, options: Intl.DateTimeFormatOptions = { dateStyle: 'medium', timeStyle: 'short' }) {
  return cachedFormatter(options).format(new Date(value))
}
