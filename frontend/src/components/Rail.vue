<script setup>
import { computed, ref, watch, nextTick } from 'vue'
import { fmtUptime } from '../utils.js'

const props = defineProps({ stats: Object, config: Object })

const cpu = computed(() => props.stats?.cpu.usage ?? 0)
const ram = computed(() => props.stats?.memory.usage ?? 0)
const disks = computed(() => props.stats?.disks ?? [])
const rxMB = computed(() => (props.stats?.network.rx_rate ?? 0) / 1024 / 1024)
const txMB = computed(() => (props.stats?.network.tx_rate ?? 0) / 1024 / 1024)
const rxMax = 100, txMax = 50
const loadavg = computed(() =>
  props.stats ? props.stats.cpu.loadavg.map(v => v.toFixed(2)).join(' / ') : '--')

/* 索引统计：来自 config */
const servicesCount = computed(() =>
  props.config ? props.config.groups.reduce((n, g) => n + g.sites.length, 0) : 0)
const groupsCount = computed(() => props.config?.groups.length ?? 0)
function countByNet(net) {
  if (!props.config) return 0
  return props.config.groups.reduce((n, g) =>
    n + g.sites.reduce((m, s) =>
      m + s.addresses.filter(a => a.net === net).length, 0), 0)
}
const lanCount = computed(() => countByNet('intranet'))
const wanCount = computed(() => countByNet('internet'))

/* 流量 sparkline：记录 RX 速率历史 */
const spark = ref(null)
const hist = ref(Array(60).fill(0))
watch(() => props.stats, s => {
  if (!s) return
  hist.value.push(s.network.rx_rate / 1024 / 1024)
  hist.value.shift()
  nextTick(drawSpark)
}, { immediate: true })

function drawSpark() {
  const cv = spark.value
  if (!cv) return
  const dpr = window.devicePixelRatio || 1
  const r = cv.getBoundingClientRect()
  cv.width = r.width * dpr; cv.height = r.height * dpr
  const cx = cv.getContext('2d')
  const w = cv.width, h = cv.height
  const max = Math.max(1, ...hist.value)
  cx.clearRect(0, 0, w, h)
  cx.beginPath()
  hist.value.forEach((v, i) => {
    const x = i / (hist.value.length - 1) * w
    const y = h - (v / max) * (h - 4) - 2
    i ? cx.lineTo(x, y) : cx.moveTo(x, y)
  })
  cx.strokeStyle = '#6BDFF2'; cx.lineWidth = 1.6 * dpr; cx.stroke()
  cx.lineTo(w, h); cx.lineTo(0, h); cx.closePath()
  cx.fillStyle = 'rgba(107,223,242,.12)'; cx.fill()
}
</script>

<template>
  <aside class="rail">
    <div class="panel reveal vis" style="--acc:var(--blue)">
      <div class="p-head"><span class="pi">▤</span>INDEX 索引</div>
      <div class="p-body">
        <div class="stats">
          <div class="stat"><b>{{ String(servicesCount).padStart(2, '0') }}</b><span>SERVICES</span></div>
          <div class="stat"><b>{{ String(groupsCount).padStart(2, '0') }}</b><span>GROUPS</span></div>
          <div class="stat"><b class="lan">{{ String(lanCount).padStart(2, '0') }}</b><span>LAN 内网</span></div>
          <div class="stat"><b class="wan">{{ String(wanCount).padStart(2, '0') }}</b><span>WAN 外网</span></div>
        </div>
      </div>
    </div>

    <div class="panel reveal vis" style="--acc:var(--amber)">
      <div class="p-head"><span class="pi">▦</span>DISKS 磁盘</div>
      <div class="p-body">
        <div class="drow" v-for="d in disks" :key="d.mount">
          <span class="dlabel" :title="d.mount + ' · ' + d.name">{{ d.name }}</span>
          <span class="mbar"><i :class="{ warn: d.usage > 85 }" :style="{ width: d.usage + '%' }"></i></span>
          <output>{{ d.usage.toFixed(0) }}%</output>
        </div>
        <div v-if="!disks.length" class="dempty">NO DATA VOLUME · 无数据卷</div>
      </div>
    </div>

    <div class="panel reveal vis" style="--acc:var(--cyan)">
      <div class="p-head"><span class="pi">∿</span>NETWORK I/O</div>
      <div class="p-body">
        <canvas ref="spark" class="spark" aria-hidden="true"></canvas>
        <div class="netgrid">
          <div class="net rx">
            <b>{{ rxMB.toFixed(1) }}</b><span>RX MB/S</span>
            <div class="mini"><i :style="{ width: Math.min(100, rxMB / rxMax * 100) + '%' }"></i></div>
          </div>
          <div class="net tx">
            <b>{{ txMB.toFixed(1) }}</b><span>TX MB/S</span>
            <div class="mini"><i :style="{ width: Math.min(100, txMB / txMax * 100) + '%' }"></i></div>
          </div>
        </div>
      </div>
    </div>

    <div class="panel reveal vis" style="--acc:var(--green)">
      <div class="p-head"><span class="pi">▣</span>SYSTEM 系统</div>
      <div class="p-body" v-if="stats">
        <div class="meter" style="--mc:var(--green)">
          <label>CPU</label><span class="mbar"><i :style="{ width: cpu + '%' }"></i></span><output>{{ cpu.toFixed(0) }}%</output>
        </div>
        <div class="meter" style="--mc:var(--cyan)">
          <label>RAM</label><span class="mbar"><i :style="{ width: ram + '%' }"></i></span><output>{{ ram.toFixed(0) }}%</output>
        </div>
        <div class="sysinfo">
          <div class="si-row"><span>HOSTNAME</span><b>{{ stats.hostname || '--' }}</b></div>
          <div class="si-row"><span>UPTIME</span><b>{{ fmtUptime(stats.uptime) }}</b></div>
          <div class="si-row"><span>CPU CORES</span><b>{{ stats.cpu.cores }}</b></div>
          <div class="si-row"><span>LOADAVG</span><b>{{ loadavg }}</b></div>
        </div>
      </div>
      <div class="p-body sysinfo" v-else>
        <div class="si-row"><span>STATUS</span><b>指标暂不可用</b></div>
      </div>
    </div>
  </aside>
</template>
