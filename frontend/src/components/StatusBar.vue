<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

defineProps({ config: Object })

const clock = ref('--:--:-- --')
const WD = ['SUN', 'MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT']
const p = n => String(n).padStart(2, '0')
function tick() {
  const d = new Date()
  clock.value = `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())} ${WD[d.getDay()]} ${p(d.getMonth() + 1)}-${p(d.getDate())}`
}
let iv
onMounted(() => { tick(); iv = setInterval(tick, 1000) })
onUnmounted(() => clearInterval(iv))
</script>

<template>
  <header class="statusbar">
    <div class="sb-brand">
      <span class="leds"><i class="led"></i><i class="led amber"></i><i class="led cyan"></i></span>
      <span class="sb-name">{{ config?.title || 'NAV·DOCK' }} <b>//</b> HOMELAB</span>
    </div>
    <div class="sb-right">
      <div class="sb-item"><b>{{ clock }}</b></div>
    </div>
  </header>
</template>
