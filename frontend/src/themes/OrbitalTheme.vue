<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import {
  Orbit, Cpu, MemoryStick, Search, X, ArrowUpRight,
  Copy, Check, Network, Globe, Power,
} from 'lucide-vue-next'
import SiteIcon from '../components/SiteIcon.vue'
import { resolveAddress, hostOf } from '../utils.js'

const props = defineProps({
  config: Object,
  stats: Object,
  netMode: String,
  loadError: Boolean,
})
const emit = defineEmits(['net-change'])

const PALETTE = ['#C8860A', '#E05A10', '#E03E3E', '#D13A72', '#9C3D5E', '#8F6224', '#7E7A1F']
const reduced = matchMedia('(prefers-reduced-motion: reduce)').matches
const pad = n => String(n).padStart(2, '0')
const hexA = (h, a) => {
  const n = parseInt(h.slice(1), 16)
  return `rgba(${n >> 16 & 255},${n >> 8 & 255},${n & 255},${a})`
}

/* ---- 节点（由 config 派生）---- */
const groups = computed(() => props.config?.groups || [])
const nodes = computed(() => {
  const out = []
  groups.value.forEach((g, gi) => {
    const hue = PALETTE[gi % PALETTE.length]
    g.sites.forEach((s, si) => {
      const addr = s.addresses.find(a => a.net === 'intranet') || s.addresses[0]
      let port = ''
      try { port = new URL(addr.url).port } catch { /* 无端口 */ }
      out.push({ gi, si, g, site: s, hue, port, key: gi + '-' + si })
    })
  })
  return out
})
const tickerList = computed(() => [...nodes.value, ...nodes.value])

/* ---- 状态 ---- */
const rootRef = ref(null), cvRef = ref(null), sysRef = ref(null)
const sparkRef = ref(null), bigRef = ref(null)
const active = ref(0)
const hoverIdx = ref(-1)
const detail = ref(null)
const copied = ref(false)
const searchOpen = ref(false)
const kw = ref('')
const booted = ref(reduced)
const clock = ref('--:--:--')
const mouseOn = ref(false)

let nodeEls = []
const phys = new Map()
let rings = []
let stars = []
let W = 0, H = 0, CX = 0, CY = 0, coreR = 60, boostAmt = 0
let t = 0, momentum = 0, mx = -9999, my = -9999
let dragging = false, lastX = 0, raf = 0, clockTimer = 0, wheelLock = 0
const SQUASH = 0.86

const activeGroup = computed(() => groups.value[active.value])
const results = computed(() => {
  const q = kw.value.trim().toLowerCase()
  return nodes.value.filter(nd => !q ||
    (nd.site.name + ' ' + (nd.site.desc || '') + ' ' + (nd.g.code || '') + ' ' + nd.port)
      .toLowerCase().includes(q))
})

/* ---- 遥测（真实 /api/stats）---- */
const cpu = computed(() => props.stats?.cpu.usage ?? 0)
const mem = computed(() => props.stats?.memory.usage ?? 0)
const rxMB = computed(() => (props.stats?.network.rx_rate ?? 0) / 1048576)
const txMB = computed(() => (props.stats?.network.tx_rate ?? 0) / 1048576)
const rxH = Array(44).fill(0), txH = Array(44).fill(0)
watch(() => props.stats, s => {
  if (!s) return
  rxH.push(s.network.rx_rate / 1048576); rxH.shift()
  txH.push(s.network.tx_rate / 1048576); txH.shift()
  drawSpark()
}, { immediate: true })

function drawSpark() {
  const cv = sparkRef.value
  if (!cv) return
  const cx = cv.getContext('2d')
  cx.clearRect(0, 0, 132, 30)
  const line = (arr, max, color) => {
    cx.strokeStyle = color; cx.lineWidth = 1.4; cx.beginPath()
    arr.forEach((v, i) => {
      const x = i / (arr.length - 1) * 132
      const y = 28 - Math.min(1, v / max) * 25
      i ? cx.lineTo(x, y) : cx.moveTo(x, y)
    })
    cx.stroke()
  }
  line(rxH, 100, 'rgba(200,134,10,.95)')
  line(txH, 100, 'rgba(209,58,114,.85)')
}

/* ---- 几何 ---- */
function geom() {
  const cv = cvRef.value
  if (!cv) return
  const DPR = Math.min(devicePixelRatio || 1, 2)
  W = innerWidth; H = innerHeight
  cv.width = W * DPR; cv.height = H * DPR
  const cx = cv.getContext('2d')
  cx.setTransform(DPR, 0, 0, DPR, 0, 0)
  const mobile = W <= 820
  CX = W > 1180 ? W * 0.55 : W * 0.5
  CY = mobile ? H * 0.56 : H * 0.5
  const avail = Math.min(W - (W > 1180 ? 320 : 40), H - (mobile ? 250 : 170))
  coreR = Math.max(44, Math.min(82, avail * 0.105))
  const gap = (avail / 2 - coreR - 54) / 6.7
  boostAmt = gap * .7
  const keep = rings.map(r => r.theta)
  rings = groups.value.map((g, i) => {
    const base = coreR + 54 + i * gap
    return {
      code: g.code || pad(i + 1),
      hue: PALETTE[i % PALETTE.length],
      n: Math.max(1, g.sites.length),
      theta: keep[i] ?? i * 0.72,
      base, r: rings[i]?.r ?? base, targetR: base,
    }
  })
  applyTargets()
  stars = Array.from({ length: 150 }, () => ({
    x: Math.random() * W, y: Math.random() * H,
    z: .3 + Math.random() * .7, ph: Math.random() * Math.PI * 2, w: Math.random(),
  }))
}

function applyTargets() {
  rings.forEach((r, i) => {
    r.targetR = r.base + (i === active.value ? boostAmt * .5 : (i > active.value ? boostAmt : 0))
  })
}

function setActive(i) {
  const len = groups.value.length || 1
  active.value = ((i % len) + len) % len
  rootRef.value?.style.setProperty('--hue', PALETTE[active.value % PALETTE.length])
}
watch(active, () => {
  applyTargets()
  const el = bigRef.value
  if (el) { el.classList.remove('anim'); void el.offsetWidth; el.classList.add('anim') }
})

/* ---- 主循环 ---- */
function frame() {
  t++
  const speed = reduced ? 0 : 1
  const hov = hoverIdx.value
  rings.forEach((r, i) => {
    const dir = i % 2 ? -1 : 1
    const held = hov >= 0 && nodes.value[hov]?.gi === i ? .04 : 1
    r.theta += (dir * (0.0011 + i * 0.00013) * speed + momentum * (1 - i * 0.07)) * held
    r.r += (r.targetR - r.r) * 0.07
  })
  momentum *= 0.94

  nodes.value.forEach((nd, k) => {
    const el = nodeEls[k]
    const r = rings[nd.gi]
    if (!el || !r) return
    let p = phys.get(nd.key)
    if (!p) { p = { ox: 0, oy: 0, vx: 0, vy: 0, op: 0, x: 0, y: 0 }; phys.set(nd.key, p) }
    const a = r.theta + nd.si * (Math.PI * 2 / r.n)
    const depth = (Math.sin(a) + 1) / 2
    const px = CX + Math.cos(a) * r.r
    const py = CY + Math.sin(a) * r.r * SQUASH
    if (hov === k) {
      p.vx = 0; p.vy = 0; p.ox *= .55; p.oy *= .55
    } else {
      const dx = px + p.ox - mx, dy = py + p.oy - my
      const d = Math.hypot(dx, dy)
      if (!reduced && d < 70 && d > .001) {
        const f = (1 - d / 70) * 1.2
        p.vx += dx / d * f; p.vy += dy / d * f
      }
      p.vx = (p.vx + -p.ox * .14) * .82
      p.vy = (p.vy + -p.oy * .14) * .82
      p.ox += p.vx; p.oy += p.vy
    }
    p.x = px + p.ox; p.y = py + p.oy
    el.style.transform = `translate3d(${p.x}px,${p.y}px,0) scale(${.8 + depth * .35})`
    el.style.zIndex = 10 + Math.round(depth * 10) + (hov === k ? 40 : 0)
    const opT = nd.gi === active.value ? 1 : (hov === k ? .95 : .4 + depth * .15)
    p.op += (opT - p.op) * .1
    el.style.opacity = p.op.toFixed(3)
  })

  draw()
  raf = requestAnimationFrame(frame)
}

function coreLabel() {
  const s = groups.value[0]?.sites?.[0]
  if (!s) return ''
  const a = s.addresses.find(x => x.net === 'intranet') || s.addresses[0]
  try { return new URL(a.url).hostname } catch { return '' }
}

function draw() {
  const cv = cvRef.value
  if (!cv || !W) return
  const cx = cv.getContext('2d')
  cx.clearRect(0, 0, W, H)

  stars.forEach(s => {
    if (!reduced) { s.x -= .04 * s.z; if (s.x < 0) s.x = W }
    const px = s.x + (mx - CX) * .006 * s.z
    const tw = reduced ? .5 : .35 + .45 * Math.sin(t * .02 * s.z + s.ph)
    cx.fillStyle = `rgba(${s.w > .72 ? '196,138,60' : '150,112,66'},${tw * s.z * .34})`
    cx.fillRect(px, s.y, s.z * 1.6, s.z * 1.6)
  })

  const glow = cx.createRadialGradient(CX, CY, 0, CX, CY, coreR * 3.4)
  glow.addColorStop(0, 'rgba(255,170,60,.24)')
  glow.addColorStop(1, 'rgba(255,170,60,0)')
  cx.fillStyle = glow
  cx.beginPath(); cx.arc(CX, CY, coreR * 3.4, 0, 7); cx.fill()
  const sun = cx.createRadialGradient(CX, CY, 0, CX, CY, coreR * 1.15)
  sun.addColorStop(0, 'rgba(255,190,84,.6)')
  sun.addColorStop(1, 'rgba(255,190,84,0)')
  cx.fillStyle = sun
  cx.beginPath(); cx.arc(CX, CY, coreR * 1.15, 0, 7); cx.fill()

  if (!reduced) {
    const ph = (t * .006) % 1
    cx.strokeStyle = `rgba(225,90,46,${(1 - ph) * .28})`; cx.lineWidth = 1
    cx.beginPath(); cx.arc(CX, CY, coreR * (1 + ph * 1.6), 0, 7); cx.stroke()
  }
  cx.strokeStyle = 'rgba(43,29,18,.5)'; cx.lineWidth = 1.2
  cx.beginPath(); cx.arc(CX, CY, coreR, 0, 7); cx.stroke()
  cx.strokeStyle = 'rgba(43,29,18,.22)'
  cx.setLineDash([2, 6]); cx.lineDashOffset = reduced ? 0 : -t * .5
  cx.beginPath(); cx.arc(CX, CY, coreR * .66, 0, 7); cx.stroke()
  cx.lineDashOffset = reduced ? 0 : t * .3
  cx.beginPath(); cx.arc(CX, CY, coreR * 1.3, 0, 7); cx.stroke()
  cx.setLineDash([])
  cx.strokeStyle = 'rgba(43,29,18,.4)'
  for (let q = 0; q < 4; q++) {
    const a = q * Math.PI / 2
    cx.beginPath()
    cx.moveTo(CX + Math.cos(a) * (coreR - 7), CY + Math.sin(a) * (coreR - 7))
    cx.lineTo(CX + Math.cos(a) * (coreR + 7), CY + Math.sin(a) * (coreR + 7))
    cx.stroke()
  }
  cx.fillStyle = '#E14E1D'
  cx.beginPath(); cx.arc(CX, CY, 3, 0, 7); cx.fill()
  cx.font = '700 8.5px "Space Mono"'; cx.textAlign = 'center'
  cx.fillStyle = 'rgba(43,29,18,.85)'
  cx.fillText('NAS CORE', CX, CY - 10)
  cx.fillStyle = 'rgba(43,29,18,.5)'
  cx.fillText(coreLabel(), CX, CY + 16)

  rings.forEach((r, i) => {
    const act = i === active.value
    cx.beginPath()
    cx.ellipse(CX, CY, r.r, r.r * SQUASH, 0, 0, Math.PI * 2)
    cx.strokeStyle = hexA(r.hue, act ? .8 : .3)
    cx.lineWidth = act ? 1.4 : 1
    if (act) { cx.setLineDash([3, 8]); cx.lineDashOffset = reduced ? 0 : -r.theta * r.r }
    cx.stroke()
    cx.setLineDash([])
    if (act && !reduced) {
      const a0 = r.theta * 1.7
      cx.beginPath()
      cx.ellipse(CX, CY, r.r, r.r * SQUASH, 0, a0, a0 + .8)
      cx.strokeStyle = hexA(r.hue, .85); cx.lineWidth = 2.2
      cx.stroke()
    }
    cx.font = '700 9px "Space Mono"'
    cx.fillStyle = hexA(r.hue, act ? .95 : .5)
    cx.fillText(r.code, CX, CY - r.r * SQUASH - 9)
  })

  const hov = hoverIdx.value
  if (hov >= 0 && nodes.value[hov]) {
    const nd = nodes.value[hov]
    const p = phys.get(nd.key)
    if (p) {
      const grad = cx.createLinearGradient(p.x, p.y, CX, CY)
      grad.addColorStop(0, hexA(nd.hue, .7)); grad.addColorStop(1, hexA(nd.hue, 0))
      cx.strokeStyle = grad; cx.lineWidth = 1.2
      cx.beginPath(); cx.moveTo(p.x, p.y)
      cx.quadraticCurveTo((p.x + CX) / 2 + 40, (p.y + CY) / 2 - 40, CX, CY)
      cx.stroke()
      cx.strokeStyle = hexA(nd.hue, .9)
      cx.beginPath(); cx.arc(p.x, p.y, 30, 0, 7); cx.stroke()
      cx.setLineDash([5, 6]); cx.lineDashOffset = reduced ? 0 : -t * .9
      cx.strokeStyle = hexA(nd.hue, .45)
      cx.beginPath(); cx.arc(p.x, p.y, 39, 0, 7); cx.stroke()
      cx.setLineDash([])
    }
  }

  if (mouseOn.value && matchMedia('(pointer:fine)').matches) {
    cx.strokeStyle = 'rgba(43,29,18,.35)'; cx.lineWidth = 1
    cx.beginPath(); cx.arc(mx, my, 10, 0, 7); cx.stroke()
    cx.beginPath()
    cx.moveTo(mx - 16, my); cx.lineTo(mx - 6, my)
    cx.moveTo(mx + 6, my); cx.lineTo(mx + 16, my)
    cx.moveTo(mx, my - 16); cx.lineTo(mx, my - 6)
    cx.moveTo(mx, my + 6); cx.lineTo(mx, my + 16)
    cx.stroke()
  }
}

/* ---- 交互 ---- */
function onMove(e) {
  mx = e.clientX; my = e.clientY
  if (!dragging) return
  const dx = e.clientX - lastX; lastX = e.clientX
  momentum += dx * .00042
  rings.forEach((r, i) => r.theta += dx * .0015 * (1 - i * .06))
}
function onDown(e) {
  if (e.target.closest('.node')) return
  dragging = true; lastX = e.clientX
  sysRef.value?.setPointerCapture(e.pointerId)
  if (detail.value) detail.value = null
}
function onUp() { dragging = false }
function onLeave() { mouseOn.value = false; mx = my = -9999; hoverIdx.value = -1 }

function onWheel(e) {
  if (searchOpen.value) return
  const now = Date.now()
  if (now - wheelLock < 420) return
  wheelLock = now
  setActive(active.value + (e.deltaY > 0 ? 1 : -1))
}

function openDetail(nd) {
  setActive(nd.gi)
  detail.value = nd
}
const dLan = computed(() => detail.value?.site.addresses.find(a => a.net === 'intranet'))
const dWan = computed(() => detail.value?.site.addresses.find(a => a.net === 'internet'))
const dActive = computed(() =>
  detail.value ? resolveAddress(detail.value.site, props.netMode) : null)
const dNo = computed(() => {
  const d = detail.value
  if (!d) return ''
  let off = 0
  for (let k = 0; k < d.gi; k++) off += groups.value[k].sites.length
  return `NODE G${pad(d.gi + 1)}-${pad(off + d.si + 1)} · ${groups.value[d.gi]?.code || ''}`
})
async function copyActive() {
  if (!dActive.value) return
  try { await navigator.clipboard.writeText(dActive.value.url) } catch { /* 剪贴板不可用 */ }
  copied.value = true
  setTimeout(() => copied.value = false, 1400)
}

function openSearch() {
  searchOpen.value = true; kw.value = ''
  nextTick(() => document.getElementById('o-sq')?.focus())
}
function closeSearch() { searchOpen.value = false }
function pickResult(nd) { closeSearch(); openDetail(nd) }

function onKey(e) {
  const typing = document.activeElement?.tagName === 'INPUT'
  if (e.key === '/' && !typing) { e.preventDefault(); openSearch() }
  else if (e.key === 'Escape') { searchOpen.value ? closeSearch() : (detail.value = null) }
  else if (!typing && (e.key === 'ArrowDown' || e.key === 'ArrowRight')) setActive(active.value + 1)
  else if (!typing && (e.key === 'ArrowUp' || e.key === 'ArrowLeft')) setActive(active.value - 1)
}

function endBoot() { booted.value = true }

/* ---- 生命周期 ---- */
watch(nodes, async () => {
  await nextTick()
  nodeEls = sysRef.value ? [...sysRef.value.querySelectorAll('.node')] : []
  geom()
  if (active.value >= groups.value.length) active.value = 0
  setActive(active.value)
})

onMounted(() => {
  geom()
  setActive(0)
  raf = requestAnimationFrame(frame)
  clockTimer = setInterval(() => {
    const d = new Date()
    clock.value = `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  }, 1000)
  if (!reduced) setTimeout(endBoot, 2400)
  addEventListener('keydown', onKey)
  addEventListener('wheel', onWheel, { passive: true })
  addEventListener('resize', geom)
})
onUnmounted(() => {
  cancelAnimationFrame(raf)
  clearInterval(clockTimer)
  removeEventListener('keydown', onKey)
  removeEventListener('wheel', onWheel)
  removeEventListener('resize', geom)
})
</script>

<template>
  <div class="orbital" ref="rootRef">
    <canvas ref="cvRef" class="o-space"></canvas>

    <div
      ref="sysRef"
      class="o-system"
      @pointermove="onMove"
      @pointerdown="onDown"
      @pointerup="onUp"
      @pointerenter="mouseOn = true"
      @pointerleave="onLeave"
    >
      <button
        v-for="(nd, k) in nodes"
        :key="nd.key"
        class="node"
        :class="{ lit: nd.gi === active, dim: nd.gi !== active }"
        :style="{ '--h': nd.hue }"
        :aria-label="nd.site.name"
        @click="openDetail(nd)"
        @pointerenter="hoverIdx = k"
        @pointerleave="hoverIdx === k && (hoverIdx = -1)"
      >
        <span class="n-dot"><SiteIcon :site="nd.site" /></span>
        <span class="n-tag">{{ nd.site.name }}<i v-if="nd.port">:{{ nd.port }}</i></span>
      </button>
    </div>

    <div class="vig"></div>
    <div class="grain"></div>

    <div class="hud brand" :class="{ ready: booted }" style="--d:.1s">
      <span class="b-glyph"><Orbit /></span>
      <div>
        <b>{{ config?.title || '导航面板' }}</b>
        <span><i class="ok"></i>ALL LINKS UP · <em>{{ clock }}</em></span>
      </div>
    </div>

    <div class="hud tele" :class="{ ready: booted }" style="--d:.25s">
      <div class="t-row">
        <span>CPU</span>
        <span class="tbar"><i :style="{ width: cpu + '%' }"></i></span>
        <b>{{ stats ? cpu.toFixed(0) + '%' : '—' }}</b>
      </div>
      <div class="t-row">
        <span>MEM</span>
        <span class="tbar"><i :style="{ width: mem + '%' }"></i></span>
        <b>{{ stats ? mem.toFixed(0) + '%' : '—' }}</b>
      </div>
      <div class="t-net">
        <b class="rx">{{ rxMB.toFixed(1) }}</b><span>RX MB/S</span>
        <b class="tx">{{ txMB.toFixed(1) }}</b><span>TX MB/S</span>
      </div>
      <canvas ref="sparkRef" width="132" height="30"></canvas>
    </div>

    <nav class="hud idx" :class="{ ready: booted }" style="--d:.4s">
      <button
        v-for="(g, i) in groups"
        :key="g.code || i"
        :class="{ on: i === active }"
        :style="{ '--h': PALETTE[i % PALETTE.length] }"
        @click="setActive(i); detail = null"
      >
        <span class="no">{{ pad(i + 1) }}</span>
        <span class="nm">{{ g.code || pad(i + 1) }}</span>
        <span class="zh">{{ g.name }}</span>
      </button>
    </nav>

    <div class="hud stage" :class="{ ready: booted }" style="--d:.55s" v-if="activeGroup">
      <div class="s-no">RING {{ pad(active + 1) }} / {{ pad(groups.length) }}</div>
      <div class="s-big anim" ref="bigRef" :data-t="activeGroup.code || activeGroup.name">
        {{ activeGroup.code || activeGroup.name }}
      </div>
      <div class="s-zh">{{ activeGroup.name }} · <b>{{ activeGroup.sites.length }}</b> NODES</div>
    </div>

    <div class="hud ctrl" :class="{ ready: booted }" style="--d:.7s">
      <div class="net">
        <button :class="{ on: netMode === 'intranet' }" @click="emit('net-change', 'intranet')">
          <Network />LAN 内网
        </button>
        <button :class="{ on: netMode === 'internet' }" @click="emit('net-change', 'internet')">
          <Globe />WAN 外网
        </button>
      </div>
      <div class="hints">
        <b>/</b>检索<b>M</b>链路<b>T</b>主题<b>SCROLL</b>切换环<b>DRAG</b>旋转
      </div>
    </div>

    <div class="o-err" v-if="loadError && !config">
      无法加载配置，请确认后端服务已启动（/api/config）
    </div>

    <Transition name="od">
      <aside
        v-if="detail"
        class="o-detail"
        :style="{ '--hue': detail.hue, borderColor: detail.hue }"
      >
        <button class="d-close" @click="detail = null" aria-label="关闭"><X /></button>
        <div class="d-no">{{ dNo }}</div>
        <h2 class="d-name">{{ detail.site.name }}</h2>
        <div class="d-zh">{{ detail.site.desc || detail.site.name }}</div>
        <div class="d-port" v-if="detail.port">PORT <b>:{{ detail.port }}</b></div>
        <div class="d-links">
          <a
            v-if="dLan"
            class="d-btn"
            :class="{ primary: netMode === 'intranet' }"
            :href="dLan.url" target="_blank" rel="noopener"
          >
            <Network />LAN 内网直达<span>{{ hostOf(dLan.url) }}</span><ArrowUpRight class="ar" />
          </a>
          <a
            v-if="dWan"
            class="d-btn"
            :class="{ primary: netMode === 'internet' }"
            :href="dWan.url" target="_blank" rel="noopener"
          >
            <Globe />WAN 外网回程<span>{{ hostOf(dWan.url) }}</span><ArrowUpRight class="ar" />
          </a>
        </div>
        <button class="d-copy" :class="{ done: copied }" @click="copyActive">
          <Copy v-if="!copied" /><Check v-else />
          <span>COPY ACTIVE URL</span>
        </button>
        <div class="d-sig"><i></i><i></i><i></i><i></i><i></i><em>SIGNAL LIVE</em></div>
      </aside>
    </Transition>

    <Transition name="of">
      <div v-if="searchOpen" class="o-search" @click.self="closeSearch">
        <div class="s-head">
          <Search class="s-ic" />
          <input id="o-sq" v-model="kw" placeholder="检索服务 / SEARCH" autocomplete="off">
          <button class="s-x" @click="closeSearch" aria-label="关闭"><X /></button>
        </div>
        <div class="s-count">{{ results.length }} RESULTS</div>
        <div class="s-list">
          <button
            v-for="nd in results"
            :key="nd.key"
            class="s-row"
            :style="{ '--h': nd.hue }"
            @click="pickResult(nd)"
          >
            <span class="s-ico"><SiteIcon :site="nd.site" /></span>
            <span class="s-name">{{ nd.site.name }}<small>{{ nd.site.desc || nd.site.name }}</small></span>
            <span class="s-code">{{ nd.g.code }}</span>
            <span class="s-port" v-if="nd.port">:{{ nd.port }}</span>
            <ArrowUpRight />
          </button>
          <div v-if="!results.length" class="s-none">NO MATCH — 未找到匹配的服务</div>
        </div>
      </div>
    </Transition>

    <div class="ticker" :class="{ ready: booted }" v-if="nodes.length">
      <div class="tk-in">
        <span class="tk" v-for="(nd, i) in tickerList" :key="i">
          <b>{{ nd.site.name }}</b><i></i><em v-if="nd.port">:{{ nd.port }}</em>&nbsp;<u>OK</u>
        </span>
      </div>
    </div>

    <Transition name="ob">
      <div v-if="!booted" class="o-boot" @click="endBoot">
        <div class="b-rings"><i></i><i></i><i></i><em></em><u></u></div>
        <div class="b-title">
          <span v-for="(c, i) in 'ORBITAL'" :key="i" :style="{ '--i': i }">{{ c }}</span>
        </div>
        <div class="b-sub">{{ config?.title || 'NAS 服务导航' }} · 星轨调度台</div>
        <div class="b-log">
          <span style="--d:0s">&gt; CORE {{ coreLabel() || 'NAS' }} ............. <b>ONLINE</b></span>
          <span style="--d:.14s">&gt; RINGS {{ groups.length }} · NODES {{ nodes.length }} ............ <em>MAPPED</em></span>
          <span style="--d:.28s">&gt; YAML CONFIG .............. <b>HOT RELOAD</b></span>
        </div>
        <div class="b-bar"><i></i></div>
        <div class="b-skip"><Power />CLICK TO SKIP</div>
      </div>
    </Transition>
  </div>
</template>
