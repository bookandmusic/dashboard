<script setup>
import { ref } from 'vue'
import NavItem from './NavItem.vue'

const props = defineProps({
  group: Object,
  index: Number,
  netMode: String,
})

const ACCENTS = ['var(--amber)', 'var(--cyan)', 'var(--green)', 'var(--coral)', 'var(--blue)', 'var(--pink)']
const acc = ACCENTS[props.index % ACCENTS.length]

const open = ref(true)
</script>

<template>
  <section class="group reveal vis" :class="{ closed: !open }" :style="{ '--acc': acc }">
    <header class="g-head" @click="open = !open" role="button" :aria-expanded="open" tabindex="0">
      <span class="g-no">{{ String(index + 1).padStart(2, '0') }}</span>
      <span class="g-zh">{{ group.name }}</span>
      <span class="g-en" v-if="group.code">{{ group.code }}</span>
      <span class="g-rule"></span>
      <span class="g-n">{{ String(group.sites.length).padStart(2, '0') }} UNIT</span>
      <span class="fold" aria-hidden="true">▾</span>
    </header>
    <div class="g-body">
      <NavItem
        v-for="site in group.sites"
        :key="site.name"
        :site="site"
        :no="site._no"
        :net-mode="netMode"
      />
    </div>
  </section>
</template>
