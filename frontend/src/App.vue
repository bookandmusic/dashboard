<script setup>
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { fetchConfig, fetchStats } from './api.js'
import DarkTheme from './themes/DarkTheme.vue'
import OrbitalTheme from './themes/OrbitalTheme.vue'

const THEMES = ['dark', 'orbital']
const config = ref(null)
const stats = ref(null)
const loadError = ref(false)
const netMode = ref(localStorage.getItem('dashboard.net') || '')
const picked = localStorage.getItem('dashboard.theme')
const theme = ref(THEMES.includes(picked) ? picked : 'dark')

watch(theme, t => {
  document.body.className = 'theme-' + t
  localStorage.setItem('dashboard.theme', t)
}, { immediate: true })

async function load() {
  try {
    const cfg = await fetchConfig()
    if (JSON.stringify(cfg) !== JSON.stringify(config.value)) {
      config.value = cfg
      document.title = cfg.title
      if (!netMode.value) netMode.value = cfg.networkMode || 'intranet'
      if (!picked && THEMES.includes(cfg.theme)) theme.value = cfg.theme
    }
    loadError.value = false
  } catch (e) {
    loadError.value = true
  }
  try {
    stats.value = await fetchStats()
  } catch (e) { /* 指标不可用时主题自行降级 */ }
}

function setNetMode(mode) {
  netMode.value = mode
  localStorage.setItem('dashboard.net', mode)
}

function cycleTheme() {
  theme.value = THEMES[(THEMES.indexOf(theme.value) + 1) % THEMES.length]
}

function onKey(e) {
  const typing = document.activeElement?.tagName === 'INPUT'
  if (typing) return
  if (e.key === 'm' || e.key === 'M') {
    setNetMode(netMode.value === 'intranet' ? 'internet' : 'intranet')
  } else if (e.key === 't' || e.key === 'T') {
    cycleTheme()
  }
}

let timer
onMounted(() => {
  load()
  timer = setInterval(load, 10000)
  addEventListener('keydown', onKey)
})
onUnmounted(() => {
  clearInterval(timer)
  removeEventListener('keydown', onKey)
})
</script>

<template>
  <DarkTheme
    v-if="theme === 'dark'"
    :config="config"
    :stats="stats"
    :net-mode="netMode"
    :load-error="loadError"
    @net-change="setNetMode"
  />
  <OrbitalTheme
    v-else
    :config="config"
    :stats="stats"
    :net-mode="netMode"
    :load-error="loadError"
    @net-change="setNetMode"
  />
</template>
