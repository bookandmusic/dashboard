<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'

const props = defineProps({ config: Object })
const emit = defineEmits(['done'])

const REDUCED = window.matchMedia('(prefers-reduced-motion: reduce)').matches

const lines = computed(() => {
  const cfg = props.config
  const sites = cfg ? cfg.groups.reduce((n, g) => n + g.sites.length, 0) : 0
  const groups = cfg ? cfg.groups.length : 0
  return [
    'NAVDock BIOS — SELF-HOSTED SYSTEMS',
    'MEM CHECK ............. OK',
    'MOUNTING CONFIG ....... config.yaml OK',
    `DETECTING GROUPS ...... ${String(groups).padStart(2, '0')} OK`,
    `STARTING SERVICES ..... ${sites}/${sites} OK`,
    'HOT-RELOAD WATCHER .... ARMED',
    '> READY — 正在进入导航 ...',
  ]
})

const shown = ref(REDUCED ? 7 : 0)
const progress = computed(() => shown.value / 7)
let finished = false
let iv

function finish() {
  if (finished) return
  finished = true
  clearInterval(iv)
  emit('done')
}

onMounted(() => {
  if (REDUCED) { finish(); return }
  iv = setInterval(() => {
    shown.value++
    if (shown.value >= 7) { clearInterval(iv); setTimeout(finish, 400) }
  }, 130)
})
onUnmounted(() => clearInterval(iv))
</script>

<template>
  <div class="boot" @click="finish">
    <pre>{{ lines.slice(0, shown).join('\n') }}</pre>
    <div class="bar"><i :style="{ transform: `scaleX(${progress})` }"></i></div>
    <span class="skip">CLICK TO SKIP — 点击任意处跳过</span>
  </div>
</template>
