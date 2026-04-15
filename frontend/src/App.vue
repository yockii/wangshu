<script setup lang="ts">
import { Browser } from '@wailsio/runtime'

document.addEventListener('click', (e: MouseEvent | TouchEvent) => {
  if (e.target instanceof HTMLElement && e.target.tagName === 'A') {
    e.preventDefault()
    Browser.OpenURL((e.target as HTMLAnchorElement).href)
  }
}, true)
</script>

<template>
  <router-view />
</template>

<style scoped>
.logo {
  height: 6em;
  padding: 1.5em;
  will-change: filter;
}
.logo:hover {
  filter: drop-shadow(0 0 2em #e80000aa);
}
.logo.vue:hover {
  filter: drop-shadow(0 0 2em #42b883aa);
}
</style>
