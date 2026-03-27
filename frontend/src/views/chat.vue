<template>
  <div class="w-full h-full flex flex-col justify-between px-4 py-2">
    <div class="flex-1 overflow-y-auto p-4 space-y-4">
      <MessageItem
        v-for="msg in messages"
        :key="msg.id"
        :content="msg.content"
        :is-user="msg.isUser"
      />
    </div>

    <div class="mt-2">
      <InputGroup>
        <InputGroupTextarea placeholder="Ask, Search or Chat..." />
        <InputGroupAddon align="block-end">
          <InputGroupButton variant="outline" class="rounded-full" size="icon-xs">
            <PlusIcon class="size-4" />
          </InputGroupButton>
          <InputGroupText class="ml-auto">
            52% used
          </InputGroupText>
          <Separator orientation="vertical" class="!h-4" />
          <InputGroupButton variant="default" class="rounded-full" size="icon-xs" @click="startChat">
            <ArrowUpIcon class="size-4" />
            <span class="sr-only">Send</span>
          </InputGroupButton>
        </InputGroupAddon>
      </InputGroup>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowUpIcon,PlusIcon } from 'lucide-vue-next'
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput, InputGroupText, InputGroupTextarea } from '@/components/ui/input-group'
import { Separator } from '@/components/ui/separator'
import { ref, shallowRef } from 'vue';
import type { Message } from '@/types/message'
import MessageItem from '@/components/MessageItem.vue'

// 使用 shallowRef 优化性能，避免深度监听整个消息数组
const messages = shallowRef<Message[]>([])



let streamingMessage: Message | null = null
let contentBuffer = ''
let isThrottling = false

// 模拟从大模型接收数据流
function handleStreamChunk(chunk) {
  contentBuffer += chunk // 1. 累积内容到缓冲区

  // 2. 使用 requestAnimationFrame 进行节流
  if (!isThrottling) {
    isThrottling = true
    requestAnimationFrame(() => {
      // 3. 在一帧内，用缓冲区的完整内容更新消息
      if (streamingMessage) {
        streamingMessage.content = contentBuffer
      }
      // 4. 手动触发 shallowRef 的更新
      messages.value = [...messages.value] 
      isThrottling = false
    })
  }
}

async function startChat() {
  const userMessage = { id: Date.now(), content: '你好', isUser: true }
  const aiMessage = { id: Date.now() + 1, content: '', isUser: false }
  
  messages.value = [...messages.value, userMessage, aiMessage]
  streamingMessage = aiMessage
  contentBuffer = ''

  // 模拟接收流式数据
  const mockStream = ['你好', '，我是', 'AI', '助手。']
  for (const chunk of mockStream) {
    handleStreamChunk(chunk)
    await new Promise(r => setTimeout(r, 100)) // 模拟网络延迟
  }
}
</script>