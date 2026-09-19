<script setup lang="ts">
import { ref, watch } from 'vue'

interface ComboboxOption { id: string; label: string; sublabel?: string }

const props = withDefaults(defineProps<{
  modelValue: string
  selectedLabel: string
  search: (query: string) => Promise<ComboboxOption[]>
  placeholder?: string
  clearable?: boolean
  clearLabel?: string
  noResultsLabel?: string
}>(), { placeholder: '', clearable: false, clearLabel: '', noResultsLabel: 'No matches' })

const emit = defineEmits<{ 'update:modelValue': [string] }>()

const query = ref(props.selectedLabel)
const open = ref(false)
const loading = ref(false)
const results = ref<ComboboxOption[]>([])
const highlighted = ref(-1)
let searchTimer: number | undefined
let blurTimer: number | undefined

watch(() => props.selectedLabel, value => { if (!open.value) query.value = value })

function runSearch(term: string) {
  loading.value = true
  const requested = term
  props.search(term).then(items => {
    if (requested !== query.value.trim() && !(term === '' && query.value === props.selectedLabel)) return
    results.value = items
    highlighted.value = items.length ? 0 : -1
  }).finally(() => { loading.value = false })
}

function onFocus() {
  if (blurTimer) window.clearTimeout(blurTimer)
  open.value = true
  query.value = ''
  runSearch('')
}

function onInput() {
  open.value = true
  if (searchTimer) window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => runSearch(query.value.trim()), 800)
}

function select(option: ComboboxOption) {
  emit('update:modelValue', option.id)
  query.value = option.label
  open.value = false
}

function clearSelection() {
  emit('update:modelValue', '')
  query.value = ''
  open.value = false
}

function onBlur() {
  blurTimer = window.setTimeout(() => {
    open.value = false
    query.value = props.selectedLabel
  }, 150)
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    if (!open.value) { onFocus(); return }
    highlighted.value = Math.min(highlighted.value + 1, results.value.length - 1)
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    highlighted.value = Math.max(highlighted.value - 1, 0)
  } else if (event.key === 'Enter') {
    if (!open.value) return
    event.preventDefault()
    const option = results.value[highlighted.value]
    if (option) select(option)
  } else if (event.key === 'Escape') {
    open.value = false
    query.value = props.selectedLabel
  }
}
</script>

<template>
  <div class="combobox">
    <input
      v-model="query"
      type="text"
      role="combobox"
      aria-autocomplete="list"
      :aria-expanded="open"
      :placeholder="placeholder"
      autocomplete="off"
      @focus="onFocus"
      @input="onInput"
      @blur="onBlur"
      @keydown="onKeydown"
    />
    <ul v-if="open" class="combobox-menu" role="listbox">
      <li v-if="loading" class="combobox-status">…</li>
      <template v-else>
        <li v-if="clearable" class="combobox-option clear-option" @mousedown.prevent="clearSelection">{{ clearLabel }}</li>
        <li v-if="!results.length" class="combobox-status">{{ noResultsLabel }}</li>
        <li
          v-for="(option, index) in results"
          :key="option.id"
          class="combobox-option"
          :class="{ highlighted: index === highlighted }"
          role="option"
          :aria-selected="option.id === modelValue"
          @mousedown.prevent="select(option)"
          @mouseenter="highlighted = index"
        >
          <strong>{{ option.label }}</strong><small v-if="option.sublabel">{{ option.sublabel }}</small>
        </li>
      </template>
    </ul>
  </div>
</template>

<style scoped>
.combobox { position: relative; }
.combobox-menu {
  position: absolute; top: calc(100% + 6px); left: 0; right: 0; z-index: 20;
  max-height: 260px; margin: 0; padding: 6px; overflow-y: auto; list-style: none;
  border: 1px solid var(--line); border-radius: 10px; background: var(--surface); box-shadow: var(--shadow);
}
.combobox-status { padding: 9px 10px; color: var(--muted); font-size: 13px; }
.combobox-option { display: grid; gap: 2px; padding: 8px 10px; border-radius: 7px; cursor: pointer; }
.combobox-option.highlighted, .combobox-option:hover { background: var(--accent-pale); }
.combobox-option strong { font-size: 13.5px; font-weight: 700; }
.combobox-option small { color: var(--muted); font-size: 11.5px; }
.clear-option { color: var(--muted); font-style: italic; }
</style>
