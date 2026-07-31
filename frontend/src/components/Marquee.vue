<script setup>
import { computed } from 'vue'

const props = defineProps({ config: Object })

const items = computed(() => {
  if (!props.config) return []
  const out = []
  props.config.groups.forEach(g =>
    g.sites.forEach(s => out.push({ name: s.name, code: g.code || g.name })))
  return out
})
</script>

<template>
  <div class="marquee">
    <div class="mq-in">
      <template v-for="rep in 2" :key="rep">
        <span v-for="it in items" :key="rep + it.name">
          {{ it.name.toUpperCase() }} <b>{{ it.code }}</b> <span class="d">◆</span>
        </span>
      </template>
    </div>
  </div>
</template>
