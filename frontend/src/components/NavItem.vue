<script setup>
import { ref, computed } from 'vue'
import { resolveAddress, hostOf } from '../utils.js'
import SiteIcon from './SiteIcon.vue'

const props = defineProps({
  site: Object,
  no: Number,
  netMode: String,
})

const open = ref(false)
const copiedUrl = ref('')

const isDual = computed(() => props.site.addresses.length > 1)
const resolved = computed(() => resolveAddress(props.site, props.netMode))
const singleLabel = computed(() =>
  props.site.addresses[0]?.net === 'intranet' ? '仅内网' : '仅外网')

const port = computed(() => {
  const addr = props.site.addresses.find(a => a.net === 'intranet') || props.site.addresses[0]
  try { return new URL(addr.url).port } catch { return '' }
})

function onRowClick(e) {
  if (e.target.closest('a') || e.target.closest('.cp') || e.target.closest('.r-exp')) return
  const isMobile = window.matchMedia('(max-width:820px)').matches
  if (isMobile) { open.value = !open.value; return }
  window.open(resolved.value.url, '_blank', 'noopener')
}

function onChipClick(addr, e) {
  e.stopPropagation()
  window.open(addr.url, '_blank', 'noopener')
}

function toggleOpen(e) {
  e.stopPropagation()
  open.value = !open.value
}

async function onCopy(url, e) {
  e.stopPropagation()
  try {
    await navigator.clipboard.writeText(url)
  } catch {
    const ta = document.createElement('textarea')
    ta.value = url
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    ta.remove()
  }
  copiedUrl.value = url
  setTimeout(() => { if (copiedUrl.value === url) copiedUrl.value = '' }, 1300)
}
</script>

<template>
  <article class="row" :class="{ open }" @click="onRowClick">
    <div class="r-main">
      <span class="r-idx">U{{ String(no).padStart(2, '0') }}</span>
      <span class="r-led"></span>
      <span class="r-ico"><SiteIcon :site="site" /></span>
      <div class="r-name">
        <h3>{{ site.name }}</h3>
        <div class="r-sub">
          <span class="r-zh">{{ site.desc || site.name }}</span>
          <span class="r-port" v-if="port">:{{ port }}</span>
        </div>
      </div>

      <div class="r-links-inline">
        <template v-if="isDual">
          <a
            v-for="addr in site.addresses"
            :key="addr.net"
            class="go-link"
            :class="{ active: addr.net === netMode }"
            href="#"
            :title="addr.url"
            @click.prevent="onChipClick(addr, $event)"
          >
            <span class="chip" :class="addr.net === 'intranet' ? 'lan' : 'wan'">
              {{ addr.net === 'intranet' ? 'LAN' : 'WAN' }}
            </span>
            <span class="r-url">{{ hostOf(addr.url) }}</span>
          </a>
        </template>
        <span v-else class="only-tag">{{ singleLabel }}</span>
      </div>

      <button class="r-exp" @click="toggleOpen" :aria-expanded="open" aria-label="展开详情">▾</button>
    </div>

    <div class="r-details">
      <div class="rd-in">
        <div class="rd-pad">
          <p class="r-desc" v-if="site.desc">{{ site.desc }}</p>
          <div class="rd-links">
            <template v-for="addr in site.addresses" :key="addr.net">
              <a
                class="lk"
                :class="addr.net === 'intranet' ? 'lan' : 'wan'"
                :href="addr.url"
                target="_blank"
                rel="noopener"
              >
                <b>{{ addr.net === 'intranet' ? 'LAN' : 'WAN' }}</b>
                <code>{{ addr.url }}</code>
              </a>
              <button
                class="cp"
                :class="{ done: copiedUrl === addr.url }"
                :aria-label="'复制' + addr.label + '地址'"
                @click="onCopy(addr.url, $event)"
              >{{ copiedUrl === addr.url ? '✓' : '⧉' }}</button>
            </template>
          </div>
        </div>
      </div>
    </div>
  </article>
</template>
