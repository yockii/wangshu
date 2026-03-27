<template>
  <div :class="['flex w-full', isUser ? 'justify-end' : 'justify-start']">
    <div :class="['max-w-[80%] rounded-lg p-4', isUser ? 'bg-blue-500 text-white' : 'bg-gray-100 text-gray-900']">
      <!-- 用户消息直接显示文本 -->
      <p v-if="isUser">{{ content }}</p>
      
      <!-- AI消息使用 markdown-it 渲染 -->
      <div v-else class="markdown-body" v-html="renderedContent"></div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import MarkdownIt from 'markdown-it'

const props = defineProps({
  content: { type: String, required: true },
  isUser: { type: Boolean, required: true }
})

const md = new MarkdownIt({ html: true, linkify: true, typographer: true })

const renderedContent = computed(() => {
  return props.isUser ? '' : md.render(props.content)
})
</script>