<template>
  <div class="wails-nodraggable" :class="['flex w-full', isUser ? 'justify-end' : 'justify-start']">
    <div :class="['max-w-[80%] rounded-lg p-2', isUser ? 'bg-blue-500 text-white' : 'bg-gray-100 text-gray-900']">
      <!-- 用户消息直接显示文本 -->
      <p class="text-sm" v-if="isUser">{{ content }}</p>
      
      <!-- AI消息使用 markdown-it 渲染 -->
      <div v-else class="text-sm markdown-content" v-html="renderedContent"></div>
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

<style>
.markdown-content {
  line-height: 1.6;
}

/* 表格样式 */
.markdown-content table {
  width: 100%;
  border-collapse: collapse;
  margin: 1rem 0;
  font-size: 0.875rem;
}
.markdown-content th,
.markdown-content td {
  padding: 0.5rem;
  border: 1px solid var(--border);
  text-align: left;
  white-space: nowrap;
}
.markdown-content th {
  background-color: var(--muted);
  font-weight: 600;
  position: sticky;
  top: 0;
}

.markdown-content tr:nth-child(even) {
  background-color: var(--secondary);
}
/* 代码块样式 */
.markdown-content pre {
  background-color: var(--muted);
  padding: 1rem;
  border-radius: 0.375rem;
  overflow-x: auto;
  margin: 1rem 0;
}

.markdown-content code {
  background-color: var(--muted);
  padding: 0.125rem 0.25rem;
  border-radius: 0.25rem;
  font-family: 'Courier New', Courier, monospace;
}

.markdown-content pre code {
  background-color: transparent;
  padding: 0;
}

/* 列表样式 */
.markdown-content ul,
.markdown-content ol {
  margin: 0.5rem 0;
  padding-left: 1.5rem;
}

.markdown-content li {
  margin: 0.25rem 0;
}

/* 引用样式 */
.markdown-content blockquote {
  border-left: 4px solid var(--primary);
  padding-left: 1rem;
  margin: 1rem 0;
  color: var(--muted-foreground);
}

/* 标题样式 */
.markdown-content h1,
.markdown-content h2,
.markdown-content h3,
.markdown-content h4,
.markdown-content h5,
.markdown-content h6 {
  margin: 1rem 0 0.5rem 0;
  font-weight: 600;
}

.markdown-content h1 {
  font-size: 1.5rem;
}

.markdown-content h2 {
  font-size: 1.25rem;
}

.markdown-content h3 {
  font-size: 1.125rem;
}

/* 链接样式 */
.markdown-content a {
  color: var(--primary);
  text-decoration: underline;
}

.markdown-content a:hover {
  opacity: 0.8;
}
</style>
