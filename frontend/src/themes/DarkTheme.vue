<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import TrafficCanvas from '../components/TrafficCanvas.vue'
import BootScreen from '../components/BootScreen.vue'
import StatusBar from '../components/StatusBar.vue'
import Masthead from '../components/Masthead.vue'
import NavPanel from '../components/NavPanel.vue'
import Rail from '../components/Rail.vue'
import Marquee from '../components/Marquee.vue'

const props = defineProps({
  config: Object,
  stats: Object,
  netMode: String,
  loadError: Boolean,
})
const emit = defineEmits(['net-change'])

const query = ref('')
const booted = ref(false)

function onKey(e) {
  const typing = document.activeElement?.tagName === 'INPUT'
  if (e.key === '/' && !typing) {
    e.preventDefault()
    document.querySelector('.search input')?.focus()
  } else if (e.key === 'Escape') {
    query.value = ''
    document.querySelector('.search input')?.blur()
  }
}

onMounted(() => addEventListener('keydown', onKey))
onUnmounted(() => removeEventListener('keydown', onKey))
</script>

<template>
  <TrafficCanvas />

  <BootScreen v-if="!booted" :config="config" @done="booted = true" />

  <template v-if="booted">
    <StatusBar :config="config" />

    <div class="wrap">
      <div class="layout">
        <main>
          <Masthead :config="config" />

          <div v-if="loadError && !config" class="empty" style="display:block">
            无法加载配置，请确认后端服务已启动（/api/config）
          </div>

          <NavPanel
            v-if="config"
            :config="config"
            :net-mode="netMode"
            v-model:query="query"
            @net-change="emit('net-change', $event)"
          />
        </main>

        <Rail :stats="stats" :config="config" />
      </div>
    </div>

    <Marquee v-if="config" :config="config" />

    <footer>
      <span>{{ config?.title }} — 数据主权 · 自托管优先</span>
      <span class="keys"><b>/</b>检索 <b>M</b>内外网 <b>T</b>主题 <b>ESC</b>清除</span>
      <span>YAML 驱动 · 热重载 · NO TRACKING</span>
    </footer>
  </template>
</template>
