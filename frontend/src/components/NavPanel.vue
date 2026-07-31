<script setup>
import { computed } from 'vue'
import NavGroup from './NavGroup.vue'

const props = defineProps({
  config: Object,
  netMode: String,
  query: String,
})
const emit = defineEmits(['net-change', 'update:query'])

const total = computed(() => props.config.groups.reduce((n, g) => n + g.sites.length, 0))

const filteredGroups = computed(() => {
  const q = props.query.trim().toLowerCase()
  let no = 0
  const out = []
  for (const g of props.config.groups) {
    const sites = []
    for (const s of g.sites) {
      no++
      const hay = (s.name + ' ' + (s.desc || '') + ' ' + g.name + ' ' + (g.code || '')).toLowerCase()
      if (!q || hay.includes(q)) sites.push({ ...s, _no: no })
    }
    if (sites.length) out.push({ ...g, sites })
  }
  return out
})

const visible = computed(() => filteredGroups.value.reduce((n, g) => n + g.sites.length, 0))
const isWan = computed(() => props.netMode === 'internet')
</script>

<template>
  <div>
    <div class="controls">
      <label class="search">
        <span class="pr">&gt;</span>
        <input
          :value="query"
          @input="emit('update:query', $event.target.value)"
          type="text"
          placeholder="检索服务 / search services"
          autocomplete="off"
        />
        <kbd>/</kbd>
      </label>
      <div class="mode" :class="{ wan: isWan }" role="group" aria-label="网络切换">
        <span class="thumb"></span>
        <button :class="{ on: !isWan }" @click="emit('net-change', 'intranet')">
          <b>内网 LAN</b>
        </button>
        <button :class="{ on: isWan }" @click="emit('net-change', 'internet')">
          <b>外网 WAN</b>
        </button>
      </div>
      <span class="count"><b>{{ String(visible).padStart(2, '0') }}</b> / {{ String(total).padStart(2, '0') }}</span>
    </div>

    <section class="rack">
      <NavGroup
        v-for="(g, gi) in filteredGroups"
        :key="g.code || g.name"
        :group="g"
        :index="gi"
        :net-mode="netMode"
      />
    </section>

    <p class="empty" :style="{ display: visible === 0 ? 'block' : 'none' }">
      NO MATCH — 未找到匹配的服务
    </p>
  </div>
</template>
