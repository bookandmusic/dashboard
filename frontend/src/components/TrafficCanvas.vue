<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const REDUCED = window.matchMedia('(prefers-reduced-motion: reduce)').matches
const canvas = ref(null)
let raf
let fit

onMounted(() => {
  if (REDUCED || !canvas.value) return
  const cx = canvas.value.getContext('2d')
  let W, H
  fit = () => { W = canvas.value.width = innerWidth; H = canvas.value.height = innerHeight }
  fit()
  addEventListener('resize', fit)
  const COLS = ['99,230,140', '91,217,240', '255,180,84']
  const pk = Array.from({ length: 56 }, () => ({
    x: Math.random() * innerWidth, y: Math.random() * innerHeight,
    l: 30 + Math.random() * 140, v: .4 + Math.random() * 2.2,
    c: COLS[Math.random() * 3 | 0], a: .04 + Math.random() * .13,
  }))
  const loop = () => {
    cx.clearRect(0, 0, W, H)
    pk.forEach(p => {
      p.x += p.v
      if (p.x - p.l > W) { p.x = -p.l; p.y = Math.random() * H }
      const g = cx.createLinearGradient(p.x - p.l, 0, p.x, 0)
      g.addColorStop(0, `rgba(${p.c},0)`)
      g.addColorStop(1, `rgba(${p.c},${p.a})`)
      cx.fillStyle = g
      cx.fillRect(p.x - p.l, p.y, p.l, 1)
    })
    raf = requestAnimationFrame(loop)
  }
  loop()
})
onUnmounted(() => {
  cancelAnimationFrame(raf)
  if (fit) removeEventListener('resize', fit)
})
</script>

<template>
  <canvas id="traffic" ref="canvas" aria-hidden="true"></canvas>
</template>
