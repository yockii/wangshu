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
        <InputGroupTextarea placeholder="Ask, Search or Chat..." v-model="msgContent" :disabled="inputDisabled" @keydown="handleInputKeydown" />
        <InputGroupAddon align="block-end">
          <InputGroupButton variant="outline" class="rounded-full" size="icon-xs">
            <PlusIcon class="size-4" />
          </InputGroupButton>
          <InputGroupText class="ml-auto">
            52% used
          </InputGroupText>
          <Separator orientation="vertical" class="!h-4" />
          <InputGroupButton variant="default" class="rounded-full" size="icon-xs" :disabled="inputDisabled" @click="sendMessage">
            <ArrowUpIcon class="size-4" />
            <span class="sr-only">Send</span>
          </InputGroupButton>
        </InputGroupAddon>
      </InputGroup>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowUpIcon,PlusIcon } from '@lucide/vue'
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput, InputGroupText, InputGroupTextarea } from '@/components/ui/input-group'
import { Separator } from '@/components/ui/separator'
import { nextTick, onMounted, ref, shallowRef } from 'vue';
import type { Message } from '@/types/message'
import MessageItem from '@/components/MessageItem.vue'
import { ChatBundle } from '../../bindings/github.com/yockii/wangshu/internal/bundle';
import {Events} from "@wailsio/runtime";
import type { Message as BusMessage } from '../../bindings/github.com/yockii/wangshu/pkg/bus';

// 使用 shallowRef 优化性能，避免深度监听整个消息数组
const messages = shallowRef<Message[]>([])

const msgContent = ref('')
const inputDisabled = ref(false)
const sendMessage = async () => {
  if (!msgContent.value) {
    return
  }
  await ChatBundle.HandleMessage(msgContent.value)
  messages.value = [...messages.value,  {id: Date.now(), content: msgContent.value, isUser: true }]
  msgContent.value = ''
  inputDisabled.value = true
}

const handleInputKeydown = (e: KeyboardEvent) => {
  const isEnterKey = e.key === 'Enter'
  const isShortcutKey = e.ctrlKey || e.metaKey || e.altKey || e.shiftKey
  if (isEnterKey) {
    if (isShortcutKey) {
      // 获取光标位置
      const ce = e.target as HTMLTextAreaElement
      const cursorPosition = ce?.selectionStart || 0
      const textPrefix = ce?.value.slice(0, cursorPosition) || ''
      const textSuffix = ce?.value.slice(cursorPosition) || ''
      const newText = textPrefix + '\n' + textSuffix
      ce.value = newText
      nextTick(() => {
        ce.setSelectionRange(cursorPosition + 1, cursorPosition + 1)
      })
    } else {
      sendMessage()
    }
    return
  }
}






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
  const mockStream = ['你好', '，我是', '**', 'AI', '**', '助手。']
  for (const chunk of mockStream) {
    handleStreamChunk(chunk)
    await new Promise(r => setTimeout(r, 100)) // 模拟网络延迟
  }
}

onMounted(() => {
  Events.On('chat-message', (msg: {data: BusMessage}) => {
    messages.value = [...messages.value, {id:Date.now(), content:msg.data.content, isUser:false}]
    inputDisabled.value = false
  });
})
</script>