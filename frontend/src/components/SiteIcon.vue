<script setup>
import { computed } from 'vue'
import * as Lucide from 'lucide-vue-next'
import { isImageIcon, toPascal, monogram, iconSrc } from '../utils.js'

const props = defineProps({ site: Object })

const icon = computed(() => (props.site?.icon || '').trim())
const asImage = computed(() => isImageIcon(icon.value))
const src = computed(() => iconSrc(icon.value))
const lucide = computed(() =>
  !asImage.value && icon.value ? (Lucide[toPascal(icon.value)] || null) : null)
const letter = computed(() => monogram(props.site?.name))
const maskStyle = computed(() => ({ '--src': `url("${src.value}")` }))
</script>

<template>
  <span v-if="asImage" class="si-img" :style="maskStyle">
    <i class="si-mask" aria-hidden="true"></i>
    <img class="si-pic" :src="src" :alt="site.name" loading="lazy">
  </span>
  <component v-else-if="lucide" :is="lucide" class="si-lucide" />
  <span v-else class="si-mono">{{ letter }}</span>
</template>
