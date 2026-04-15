<script lang="ts" setup>
import { ref, watch } from 'vue'
import { Slider } from '@/components/ui/slider'

const opacityValue = ref([0.5])

watch(opacityValue, (newVal) => {
    document.documentElement.style.setProperty('--background', `oklch(1 0 0 / ${newVal[0]})`)
})
</script>

<template>
    <Slider :default-value="[0.5]" :max="1" :step="0.05" v-model="opacityValue" />
</template>
