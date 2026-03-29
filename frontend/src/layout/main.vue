<template>
  <div class="w-full h-full bg-background flex flex-col">
    <div class="wails-draggable flex items-center justify-between h-8 border-b border-border/50 pl-2">
      <div class="font-bold">望舒 - {{ route.meta.title }}</div>
      <div class="wails-nodraggable h-full flex items-center">
        <div v-if="routeName != 'chat'" class="w-12 h-full bg-transparent text-foreground cursor-pointer flex items-center justify-center hover:bg-foreground/10" @click="WindowBundle.ShowChatWindow()"><BotMessageSquare :size="18" /></div>
        <div v-if="routeName != 'config'" class="w-12 h-full bg-transparent text-foreground cursor-pointer flex items-center justify-center hover:bg-foreground/10" @click="WindowBundle.ShowConfigWindow()"><Settings :size="18" /></div>
        <div class="w-12 h-full bg-transparent text-foreground cursor-default flex items-center justify-center hover:bg-foreground/10" @click="Window.Minimise()"><Minus :size="18" /></div>
        <div class="w-12 h-full bg-transparent text-foreground cursor-default flex items-center justify-center hover:bg-foreground/10" @click="toggleMaximise()"><RestoreWindow v-if="isMaximised" class="h-3.5 w-3.5" /><Square v-else :size="14" /></div>
        <div class="w-12 h-full bg-transparent text-foreground cursor-default flex items-center justify-center hover:bg-red-500" @click="Window.Hide()"><X :size="18" /></div>
      </div>
    </div>
    <div class="flex-1 overflow-auto">
        <router-view />
    </div>
  </div>
</template>

<script setup lang="ts">
import { Minus, Square, X, BotMessageSquare, Settings } from '@lucide/vue'
import RestoreWindow from '@/components/icons/RestoreWindow.vue'
import { Window } from '@wailsio/runtime'
import { ref } from 'vue'
import { useRoute } from 'vue-router'

import {WindowBundle} from '../../bindings/github.com/yockii/wangshu/internal/bundle'


const route = useRoute()

const routeName = ref(route.name)

const isMaximised = ref(false)
const toggleMaximise = async () => {
    const isMaximised = await Window.IsMaximised()
    if (isMaximised) {
        await Window.Restore()
    } else {
        await Window.Maximise()
    }

    updateMaximiseIcon()
}
const updateMaximiseIcon = async () => {
  isMaximised.value = await Window.IsMaximised()
}
</script>