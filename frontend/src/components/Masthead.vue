<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const props = defineProps({ config: Object })

const title = ref(null)
const REDUCED = window.matchMedia('(prefers-reduced-motion: reduce)').matches
const FINE = window.matchMedia('(pointer: fine)').matches

/* 标题视差 */
let raf, tx = 0, ty = 0, cx = 0, cy = 0
function onMove(e) {
  tx = (e.clientX / innerWidth - .5) * 10
  ty = (e.clientY / innerHeight - .5) * 6
}
function pl() {
  cx += (tx - cx) * .06; cy += (ty - cy) * .06
  if (title.value) title.value.style.transform = `translate(${cx}px,${cy}px)`
  raf = requestAnimationFrame(pl)
}
onMounted(() => {
  if (!REDUCED && FINE) { addEventListener('mousemove', onMove); pl() }
})
onUnmounted(() => { removeEventListener('mousemove', onMove); cancelAnimationFrame(raf) })
</script>

<template>
  <section class="masthead reveal vis">
    <p class="mh-kicker">SELF-HOSTED SERVICE INDEX · 自托管服务索引</p>
    <div class="mh-grid">
      <h1 class="mh-title" ref="title">{{ config?.title }}</h1>
      <div class="badge">
        <svg viewBox="0 0 120 120">
          <defs><path id="cir" d="M60,60 m-46,0 a46,46 0 1,1 92,0 a46,46 0 1,1 -92,0" /></defs>
          <text><textPath href="#cir">SELF-HOSTED · DATA SOVEREIGNTY · NO TRACKING · NO CLOUD · </textPath></text>
        </svg>
        <span class="core">◈</span>
      </div>
    </div>
    <p class="mh-sub" v-if="config?.desc">{{ config.desc }}</p>
    <div class="barcode"><i></i><span>SN · NAV-2026-0730-A</span><i></i></div>
  </section>
</template>
